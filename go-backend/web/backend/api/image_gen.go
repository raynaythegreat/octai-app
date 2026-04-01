package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
)

func (h *Handler) registerImageGenRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/image/generate", h.handleImageGenerate)
}

type imageGenerateRequest struct {
	Prompt     string `json:"prompt"`
	ModelIndex int    `json:"model_index"`
	Size       string `json:"size"`    // "1024x1024", "1792x1024", "1024x1792"
	Quality    string `json:"quality"` // "standard", "hd"
	N          int    `json:"n"`
}

type imageGenerateResponse struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Error         string `json:"error,omitempty"`
}

// handleImageGenerate calls an OpenAI-compatible image generation API.
func (h *Handler) handleImageGenerate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req imageGenerateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if req.Size == "" {
		req.Size = "1024x1024"
	}
	if req.Quality == "" {
		req.Quality = "standard"
	}
	if req.N <= 0 {
		req.N = 1
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	if len(cfg.ImageModelList) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(imageGenerateResponse{Error: "No image models configured"})
		return
	}

	if req.ModelIndex < 0 || req.ModelIndex >= len(cfg.ImageModelList) {
		req.ModelIndex = 0
	}

	modelCfg := cfg.ImageModelList[req.ModelIndex]
	apiKey := modelCfg.APIKey()
	apiBase := modelCfg.APIBase
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}

	// Strip protocol prefix (e.g. "openai/dall-e-3" → "dall-e-3")
	modelID := modelCfg.Model
	if idx := strings.Index(modelID, "/"); idx >= 0 {
		modelID = modelID[idx+1:]
	}

	result, err := callOpenAIImageGen(r.Context(), apiKey, apiBase, modelID, req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(imageGenerateResponse{Error: err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func callOpenAIImageGen(ctx context.Context, apiKey, apiBase, model string, req imageGenerateRequest) (*imageGenerateResponse, error) {
	payload := map[string]any{
		"model":   model,
		"prompt":  req.Prompt,
		"size":    req.Size,
		"quality": req.Quality,
		"n":       req.N,
	}

	payloadBytes, _ := json.Marshal(payload)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiBase+"/images/generations", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("image generation request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image generation failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			URL           string `json:"url"`
			B64JSON       string `json:"b64_json"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no images returned")
	}

	return &imageGenerateResponse{
		URL:           result.Data[0].URL,
		B64JSON:       result.Data[0].B64JSON,
		RevisedPrompt: result.Data[0].RevisedPrompt,
	}, nil
}
