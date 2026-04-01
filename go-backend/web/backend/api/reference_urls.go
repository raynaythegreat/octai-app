package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/providers"
)

// registerReferenceURLRoutes binds the reference URL endpoints to the ServeMux.
func (h *Handler) registerReferenceURLRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/reference-urls", h.handleListReferenceURLs)
	mux.HandleFunc("POST /api/reference-urls", h.handleAddReferenceURL)
	mux.HandleFunc("DELETE /api/reference-urls/{id}", h.handleDeleteReferenceURL)
	mux.HandleFunc("PATCH /api/reference-urls/{id}", h.handleUpdateReferenceURL)
}

// ReferenceURL represents a saved reference URL for agents.
type ReferenceURL struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes,omitempty"`
	AddedAt     string   `json:"added_at"`
}

func referenceURLsPath(cfg *config.Config) string {
	home := os.Getenv("OCTAI_HOME")
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".octai")
	}
	return filepath.Join(home, "workspace", "reference_urls.json")
}

func loadReferenceURLs(path string) ([]ReferenceURL, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []ReferenceURL{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []ReferenceURL
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func saveReferenceURLs(path string, items []ReferenceURL) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// handleListReferenceURLs returns all saved reference URLs.
//
//	GET /api/reference-urls
func (h *Handler) handleListReferenceURLs(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}
	items, err := loadReferenceURLs(referenceURLsPath(cfg))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load references: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"references": items})
}

// handleAddReferenceURL saves a new reference URL, fetching and AI-categorizing it.
//
//	POST /api/reference-urls
func (h *Handler) handleAddReferenceURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL   string `json:"url"`
		Notes string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil || req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	// Fetch page content
	content, fetchErr := fetchWebContent(req.URL)
	if fetchErr != nil {
		content = ""
	}

	// AI categorization
	ref := ReferenceURL{
		ID:      generateID(),
		URL:     req.URL,
		Notes:   req.Notes,
		AddedAt: time.Now().UTC().Format(time.RFC3339),
		Tags:    []string{},
	}

	if aiResult, err := categorizeURLWithLLM(cfg, req.URL, content); err == nil {
		ref.Title = aiResult.Title
		ref.Description = aiResult.Description
		ref.Category = aiResult.Category
		ref.Tags = aiResult.Tags
	} else {
		// Fallback: use URL as title
		ref.Title = req.URL
		ref.Category = "other"
	}

	// Save
	path := referenceURLsPath(cfg)
	items, err := loadReferenceURLs(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load existing references: %v", err), http.StatusInternalServerError)
		return
	}
	items = append(items, ref)
	if err := saveReferenceURLs(path, items); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save reference: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ref)
}

// handleDeleteReferenceURL removes a reference URL by ID.
//
//	DELETE /api/reference-urls/{id}
func (h *Handler) handleDeleteReferenceURL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	path := referenceURLsPath(cfg)
	items, err := loadReferenceURLs(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load references: %v", err), http.StatusInternalServerError)
		return
	}

	filtered := items[:0]
	found := false
	for _, item := range items {
		if item.ID == id {
			found = true
		} else {
			filtered = append(filtered, item)
		}
	}
	if !found {
		http.Error(w, "Reference not found", http.StatusNotFound)
		return
	}

	if err := saveReferenceURLs(path, filtered); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save references: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleUpdateReferenceURL updates notes and/or category for a reference.
//
//	PATCH /api/reference-urls/{id}
func (h *Handler) handleUpdateReferenceURL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	var req struct {
		Notes    *string `json:"notes"`
		Category *string `json:"category"`
	}
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	path := referenceURLsPath(cfg)
	items, err := loadReferenceURLs(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load references: %v", err), http.StatusInternalServerError)
		return
	}

	var updated *ReferenceURL
	for i := range items {
		if items[i].ID == id {
			if req.Notes != nil {
				items[i].Notes = *req.Notes
			}
			if req.Category != nil {
				items[i].Category = *req.Category
			}
			updated = &items[i]
			break
		}
	}
	if updated == nil {
		http.Error(w, "Reference not found", http.StatusNotFound)
		return
	}

	if err := saveReferenceURLs(path, items); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save references: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

// categorizationResult holds the AI's analysis of a URL.
type categorizationResult struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}

// categorizeURLWithLLM uses the default model to categorize and describe a URL.
func categorizeURLWithLLM(cfg *config.Config, pageURL, content string) (*categorizationResult, error) {
	if len(cfg.ModelList) == 0 {
		return nil, fmt.Errorf("no models configured")
	}

	mc := pickScannerModel(cfg)
	provider, modelID, err := providers.CreateProviderFromConfig(mc)
	if err != nil {
		return nil, fmt.Errorf("creating provider: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Analyze this URL and its content. Return a JSON object with exactly these fields:\n")
	sb.WriteString(`- "title": concise page/resource title (string)` + "\n")
	sb.WriteString(`- "description": one-sentence summary of what this resource is (string)` + "\n")
	sb.WriteString(`- "category": one of [documentation, api-reference, tutorial, tool, library, blog, example, specification, dataset, video, other] (string)` + "\n")
	sb.WriteString(`- "tags": 3-5 relevant keywords (array of strings)` + "\n\n")
	sb.WriteString("Output ONLY a JSON object wrapped in ```json ... ``` fences.\n\n")
	sb.WriteString("URL: ")
	sb.WriteString(pageURL)
	if content != "" {
		sb.WriteString("\n\n=== CONTENT (truncated) ===\n")
		if len(content) > 8000 {
			content = content[:8000]
		}
		sb.WriteString(content)
	}

	messages := []providers.Message{
		{Role: "user", Content: sb.String()},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, messages, nil, modelID, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	jsonStr := extractJSONBlock(resp.Content)
	if jsonStr == "" {
		// Try finding raw object
		start := strings.Index(resp.Content, "{")
		end := strings.LastIndex(resp.Content, "}")
		if start >= 0 && end > start {
			jsonStr = resp.Content[start : end+1]
		}
	}
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON in LLM response")
	}

	var result categorizationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("parsing LLM response: %w", err)
	}
	if result.Tags == nil {
		result.Tags = []string{}
	}
	return &result, nil
}
