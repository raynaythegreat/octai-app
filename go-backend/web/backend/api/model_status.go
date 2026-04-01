package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
)

const modelProbeTimeout = 800 * time.Millisecond

var (
	probeTCPServiceFunc            = probeTCPService
	probeOllamaModelFunc           = probeOllamaModel
	probeOpenAICompatibleModelFunc = probeOpenAICompatibleModel
)

func hasModelConfiguration(m *config.ModelConfig) bool {
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	apiKey := strings.TrimSpace(m.APIKey())

	if authMethod == "oauth" || authMethod == "token" {
		if provider, ok := oauthProviderForModel(m.Model); ok {
			cred, err := oauthGetCredential(provider)
			if err != nil || cred == nil {
				return false
			}
			return strings.TrimSpace(cred.AccessToken) != "" || strings.TrimSpace(cred.RefreshToken) != ""
		}
		return true
	}

	if apiKey != "" {
		return true
	}

	if hasOAuthCredentialForModel(m) {
		return true
	}

	if requiresRuntimeProbe(m) {
		return true
	}

	return false
}

// isModelConfigured reports whether a model is currently available to use.
// Local models must be reachable; remote/API-key models only need saved config.
func isModelConfigured(m *config.ModelConfig) bool {
	if !hasModelConfiguration(m) {
		return false
	}
	if requiresRuntimeProbe(m) {
		return probeLocalModelAvailability(m)
	}
	return true
}

func requiresRuntimeProbe(m *config.ModelConfig) bool {
	authMethod := strings.ToLower(strings.TrimSpace(m.AuthMethod))
	if authMethod == "local" {
		return true
	}

	switch modelProtocol(m.Model) {
	case "claude-cli", "claudecli", "codex-cli", "codexcli", "github-copilot", "copilot":
		return true
	case "ollama", "vllm":
		apiBase := strings.TrimSpace(m.APIBase)
		return apiBase == "" || hasLocalAPIBase(apiBase)
	}

	if hasLocalAPIBase(m.APIBase) {
		return true
	}

	return false
}

func probeLocalModelAvailability(m *config.ModelConfig) bool {
	apiBase := modelProbeAPIBase(m)
	protocol, modelID := splitModel(m.Model)
	switch protocol {
	case "ollama":
		return probeOllamaModelFunc(apiBase, modelID)
	case "vllm":
		return probeOpenAICompatibleModelFunc(apiBase, modelID, m.APIKey())
	case "github-copilot", "copilot":
		return probeTCPServiceFunc(apiBase)
	case "claude-cli", "claudecli", "codex-cli", "codexcli":
		return true
	default:
		if hasLocalAPIBase(apiBase) {
			return probeOpenAICompatibleModelFunc(apiBase, modelID, m.APIKey())
		}
		return false
	}
}

func modelProbeAPIBase(m *config.ModelConfig) string {
	if apiBase := strings.TrimSpace(m.APIBase); apiBase != "" {
		return normalizeModelProbeAPIBase(apiBase)
	}

	switch modelProtocol(m.Model) {
	case "ollama":
		return "http://localhost:11434/v1"
	case "vllm":
		return "http://localhost:8000/v1"
	case "github-copilot", "copilot":
		return "localhost:4321"
	default:
		return ""
	}
}

func normalizeModelProbeAPIBase(raw string) string {
	u, err := parseAPIBase(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}

	switch strings.ToLower(u.Hostname()) {
	case "0.0.0.0":
		u.Host = net.JoinHostPort("127.0.0.1", u.Port())
	case "::":
		u.Host = net.JoinHostPort("::1", u.Port())
	default:
		return strings.TrimSpace(raw)
	}

	if u.Port() == "" {
		u.Host = u.Hostname()
	}

	return u.String()
}

func oauthProviderForModel(model string) (string, bool) {
	switch modelProtocol(model) {
	case "openai":
		return oauthProviderOpenAI, true
	case "anthropic":
		return oauthProviderAnthropic, true
	case "antigravity", "google-antigravity":
		return oauthProviderGoogleAntigravity, true
	default:
		return "", false
	}
}

func hasOAuthCredentialForModel(m *config.ModelConfig) bool {
	provider, ok := oauthProviderForModel(m.Model)
	if !ok {
		return false
	}

	cred, err := oauthGetCredential(provider)
	if err != nil || cred == nil {
		return false
	}

	return strings.TrimSpace(cred.AccessToken) != "" || strings.TrimSpace(cred.RefreshToken) != ""
}

func modelProtocol(model string) string {
	protocol, _ := splitModel(model)
	return protocol
}

func splitModel(model string) (protocol, modelID string) {
	model = strings.ToLower(strings.TrimSpace(model))
	protocol, _, found := strings.Cut(model, "/")
	if !found {
		return "openai", model
	}
	return protocol, strings.TrimSpace(model[strings.Index(model, "/")+1:])
}

func hasLocalAPIBase(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		u, err = url.Parse("//" + raw)
		if err != nil {
			return false
		}
	}

	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1", "0.0.0.0":
		return true
	default:
		return false
	}
}

func probeTCPService(raw string) bool {
	hostPort, err := hostPortFromAPIBase(raw)
	if err != nil {
		return false
	}

	conn, err := net.DialTimeout("tcp", hostPort, modelProbeTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func probeOllamaModel(apiBase, modelID string) bool {
	root, err := apiRootFromAPIBase(apiBase)
	if err != nil {
		return false
	}

	var resp struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := getJSON(root+"/api/tags", &resp, ""); err != nil {
		return false
	}

	for _, model := range resp.Models {
		if ollamaModelMatches(model.Name, modelID) || ollamaModelMatches(model.Model, modelID) {
			return true
		}
	}
	return false
}

func probeOpenAICompatibleModel(apiBase, modelID, apiKey string) bool {
	if strings.TrimSpace(apiBase) == "" {
		return false
	}

	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := getJSON(strings.TrimRight(strings.TrimSpace(apiBase), "/")+"/models", &resp, apiKey); err != nil {
		return false
	}

	for _, model := range resp.Data {
		if strings.EqualFold(strings.TrimSpace(model.ID), modelID) {
			return true
		}
	}
	return false
}

func getJSON(rawURL string, out any, apiKey string) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: modelProbeTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func apiRootFromAPIBase(raw string) (string, error) {
	u, err := parseAPIBase(raw)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: u.Scheme, Host: u.Host}).String(), nil
}

func hostPortFromAPIBase(raw string) (string, error) {
	u, err := parseAPIBase(raw)
	if err != nil {
		return "", err
	}

	if port := u.Port(); port != "" {
		return u.Host, nil
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return net.JoinHostPort(u.Hostname(), "443"), nil
	default:
		return net.JoinHostPort(u.Hostname(), "80"), nil
	}
}

func parseAPIBase(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty api base")
	}

	u, err := url.Parse(raw)
	if err == nil && u.Hostname() != "" {
		return u, nil
	}

	u, err = url.Parse("//" + raw)
	if err != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid api base %q", raw)
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u, nil
}

func ollamaModelMatches(candidate, want string) bool {
	candidate = strings.TrimSpace(candidate)
	want = strings.TrimSpace(want)
	if candidate == "" || want == "" {
		return false
	}
	if strings.EqualFold(candidate, want) {
		return true
	}

	base, _, _ := strings.Cut(candidate, ":")
	return strings.EqualFold(base, want)
}

const testKeyTimeout = 10 * time.Second

// resolveProviderAPIBase returns the best API base URL for a model:
// uses the model's configured api_base if set, otherwise falls back to
// the well-known default for the provider's protocol prefix.
func resolveProviderAPIBase(m *config.ModelConfig) string {
	if base := strings.TrimSpace(m.APIBase); base != "" {
		return strings.TrimRight(base, "/")
	}

	switch modelProtocol(m.Model) {
	case "openai":
		return "https://api.openai.com/v1"
	case "anthropic":
		return "https://api.anthropic.com/v1"
	case "gemini":
		return "https://generativelanguage.googleapis.com/v1beta/openai"
	case "groq":
		return "https://api.groq.com/openai/v1"
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "mistral":
		return "https://api.mistral.ai/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "grok", "xai":
		return "https://api.x.ai/v1"
	case "cerebras":
		return "https://api.cerebras.ai/v1"
	case "perplexity":
		return "https://api.perplexity.ai"
	case "together":
		return "https://api.together.xyz/v1"
	case "moonshot":
		return "https://api.moonshot.cn/v1"
	case "nvidia":
		return "https://integrate.api.nvidia.com/v1"
	case "zhipu":
		return "https://open.bigmodel.cn/api/paas/v4"
	case "qwen":
		return "https://dashscope.aliyuncs.com/compatible-mode/v1"
	case "volcengine":
		return "https://ark.cn-beijing.volces.com/api/v3"
	case "avian":
		return "https://api.avian.io/v1"
	case "minimax":
		return "https://api.minimaxi.com/v1"
	case "longcat":
		return "https://api.longcat.chat/openai"
	case "modelscope":
		return "https://api-inference.modelscope.cn/v1"
	default:
		return ""
	}
}

// listProviderModels calls GET {apiBase}/models with the given API key and returns
// a list of model IDs. Uses testKeyTimeout (10s) instead of the short probe timeout.
// Handles both OpenAI-style { data: [{ id }] } and Anthropic-style { models: [{ id }] } responses.
func listProviderModels(apiBase, apiKey string) ([]string, error) {
	modelsURL := strings.TrimRight(apiBase, "/") + "/models"

	req, err := http.NewRequest(http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, err
	}
	if k := strings.TrimSpace(apiKey); k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}

	client := &http.Client{Timeout: testKeyTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	// Try OpenAI-compatible shape first: { "data": [{ "id": "..." }] }
	var openAIResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	// Also try Anthropic shape: { "models": [{ "id": "..." }] }
	var anthropicResp struct {
		Models []struct {
			ID string `json:"id"`
		} `json:"models"`
	}

	// Read body once and try both shapes
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var ids []string

	if err := json.Unmarshal(bodyBytes, &openAIResp); err == nil && len(openAIResp.Data) > 0 {
		for _, d := range openAIResp.Data {
			if d.ID != "" {
				ids = append(ids, d.ID)
			}
		}
	}

	if len(ids) == 0 {
		if err := json.Unmarshal(bodyBytes, &anthropicResp); err == nil && len(anthropicResp.Models) > 0 {
			for _, m := range anthropicResp.Models {
				if m.ID != "" {
					ids = append(ids, m.ID)
				}
			}
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no models returned from provider; check API key validity")
	}

	return ids, nil
}
