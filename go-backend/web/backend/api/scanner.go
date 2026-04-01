package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/providers"
)

// registerScannerRoutes binds the AI URL scanner endpoints to the ServeMux.
func (h *Handler) registerScannerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/scanner/analyze", h.handleScannerAnalyze)
	mux.HandleFunc("POST /api/scanner/integrate", h.handleScannerIntegrate)
}

// scannerItem represents a discovered MCP server, skill, tool, or plugin.
type scannerItem struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config,omitempty"`
}

// analyzeRequest is the body for POST /api/scanner/analyze.
type analyzeRequest struct {
	URL        string `json:"url"`
	CrawlDepth int    `json:"crawlDepth,omitempty"`
	MaxPages   int    `json:"maxPages,omitempty"`
	SameDomain bool   `json:"sameDomain,omitempty"`
}

// analyzeResponse is the response for POST /api/scanner/analyze.
type analyzeResponse struct {
	Items   []scannerItem `json:"items"`
	URL     string        `json:"url"`
	URLType string        `json:"url_type"` // "github" or "website"
}

// integrateRequestItem is one item in the integrate request.
type integrateRequestItem struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
}

// integrateRequest is the full body for POST /api/scanner/integrate.
// Items is the list of items to integrate. Resolve maps item names to "replace" or "skip"
// for items that were previously flagged as conflicts.
type integrateRequest struct {
	Items   []integrateRequestItem `json:"items"`
	Resolve map[string]string      `json:"resolve,omitempty"` // name → "replace" | "skip"
}

// integrateResultItem is one result entry in the integrate response.
type integrateResultItem struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// conflictItem describes an item that already exists.
type conflictItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

var (
	httpClient   = &http.Client{Timeout: 30 * time.Second}
	htmlTagRe    = regexp.MustCompile(`(?s)<[^>]*>`)
	multiSpaceRe = regexp.MustCompile(`\s+`)
)

// handleScannerAnalyze fetches and analyzes a URL for MCP servers, skills, tools, and plugins.
//
//	POST /api/scanner/analyze
func (h *Handler) handleScannerAnalyze(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req analyzeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	// Determine if this is a GitHub URL
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid URL: %v", err), http.StatusBadRequest)
		return
	}

	var content string
	var urlType string

	if strings.Contains(parsedURL.Host, "github.com") {
		urlType = "github"
		content, err = fetchGitHubContent(req.URL)
	} else if req.CrawlDepth > 0 || req.MaxPages > 1 {
		urlType = "website"
		content, err = fetchWebContentCrawl(req.URL, req.CrawlDepth, req.MaxPages, req.SameDomain)
	} else {
		urlType = "website"
		content, err = fetchWebContent(req.URL)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch URL: %v", err), http.StatusBadGateway)
		return
	}

	// Load config and build provider
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	items, err := analyzeContentWithLLM(cfg, content, req.URL, urlType)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to analyze content: %v", err), http.StatusInternalServerError)
		return
	}

	// For GitHub repos, also directly scan for SKILL.md files in the repo tree.
	// These are more accurate than LLM-detected skills.
	if urlType == "github" {
		pathParts := strings.SplitN(strings.TrimPrefix(parsedURL.Path, "/"), "/", 3)
		if len(pathParts) >= 2 {
			directSkills := fetchGitHubSkillFiles(pathParts[0], pathParts[1])
			if len(directSkills) > 0 {
				// Build a map of LLM-detected skill names for dedup
				llmSkillNames := make(map[string]bool)
				for _, item := range items {
					if item.Type == "skill" {
						llmSkillNames[item.Name] = true
					}
				}
				// Remove LLM-detected skills that have a real file version
				merged := make([]scannerItem, 0, len(items)+len(directSkills))
				for _, item := range items {
					if item.Type != "skill" {
						merged = append(merged, item)
					}
				}
				// Add direct skill items (real content wins)
				for _, ds := range directSkills {
					merged = append(merged, ds)
				}
				items = merged
			}
		}
	}

	resp := analyzeResponse{
		Items:   items,
		URL:     req.URL,
		URLType: urlType,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleScannerIntegrate integrates discovered items into the system configuration.
// Supports duplicate detection: if conflicts are found and no resolve map is provided,
// returns HTTP 409 with the conflict list. Re-submit with resolve map to proceed.
//
//	POST /api/scanner/integrate
func (h *Handler) handleScannerIntegrate(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Support both legacy array format and new object format with resolve map
	var req integrateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// Try legacy: plain array
		var items []integrateRequestItem
		if err2 := json.Unmarshal(body, &items); err2 != nil {
			http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
			return
		}
		req.Items = items
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Detect conflicts for items that weren't given an explicit resolution
	var conflicts []conflictItem
	if req.Resolve == nil {
		for _, item := range req.Items {
			if itemExists(cfg, item) {
				conflicts = append(conflicts, conflictItem{Name: item.Name, Type: item.Type})
			}
		}
	}

	// If there are unresolved conflicts, return 409 so the client can ask the user
	if len(conflicts) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]any{
			"conflicts": conflicts,
			"message":   "Some items already exist. Provide a resolve map with 'replace' or 'skip' for each conflict.",
		})
		return
	}

	results := make([]integrateResultItem, 0, len(req.Items))

	for _, item := range req.Items {
		result := integrateResultItem{
			Name: item.Name,
			Type: item.Type,
		}

		// Check resolve decision for this item
		if decision, ok := req.Resolve[item.Name]; ok && decision == "skip" {
			result.Success = true
			results = append(results, result)
			continue
		}

		var intErr error
		switch item.Type {
		case "mcp_server":
			intErr = integrateMCPServer(cfg, item)
		case "skill":
			intErr = integrateSkill(cfg, item)
		case "reference_url":
			intErr = integrateReferenceURL(cfg, item)
		case "tool", "plugin", "connection", "other":
			if hasExecutableConfig(item) {
				intErr = integrateMCPServer(cfg, item)
			} else {
				intErr = saveDiscoveredItem(cfg, item)
			}
		default:
			intErr = saveDiscoveredItem(cfg, item)
		}

		if intErr != nil {
			result.Success = false
			result.Error = intErr.Error()
		} else {
			result.Success = true
		}

		results = append(results, result)
	}

	// Save the updated config once after all integrations
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// itemExists checks whether an item with the same name already exists in config.
func itemExists(cfg *config.Config, item integrateRequestItem) bool {
	switch item.Type {
	case "mcp_server":
		_, exists := cfg.Tools.MCP.Servers[item.Name]
		return exists
	case "skill":
		home := os.Getenv("OCTAI_HOME")
		if home == "" {
			userHome, _ := os.UserHomeDir()
			home = filepath.Join(userHome, ".octai")
		}
		skillDir := filepath.Join(home, "workspace", "skills", sanitizeName(item.Name))
		_, err := os.Stat(skillDir)
		return err == nil
	default:
		// For discovered items, duplicate check is already in saveDiscoveredItem
		return false
	}
}

// integrateMCPServer adds an MCP server to the config's tools.mcp.servers map.
func integrateMCPServer(cfg *config.Config, item integrateRequestItem) error {
	if item.Name == "" {
		return fmt.Errorf("mcp_server name is required")
	}

	if cfg.Tools.MCP.Servers == nil {
		cfg.Tools.MCP.Servers = make(map[string]config.MCPServerConfig)
	}

	serverCfg := config.MCPServerConfig{
		Enabled: true,
	}

	// Map common config fields
	if item.Config != nil {
		if cmd, ok := item.Config["command"].(string); ok {
			serverCfg.Command = cmd
		}
		if rawArgs, ok := item.Config["args"]; ok {
			switch v := rawArgs.(type) {
			case []string:
				serverCfg.Args = v
			case []any:
				for _, a := range v {
					if s, ok := a.(string); ok {
						serverCfg.Args = append(serverCfg.Args, s)
					}
				}
			}
		}
		if t, ok := item.Config["type"].(string); ok {
			serverCfg.Type = t
		}
		if u, ok := item.Config["url"].(string); ok {
			serverCfg.URL = u
		}
		if envRaw, ok := item.Config["env"]; ok {
			if envMap, ok := envRaw.(map[string]any); ok {
				serverCfg.Env = make(map[string]string)
				for k, v := range envMap {
					if s, ok := v.(string); ok {
						serverCfg.Env[k] = s
					}
				}
			}
		}
	}

	cfg.Tools.MCP.Servers[item.Name] = serverCfg
	return nil
}

// hasExecutableConfig returns true if the item has enough config to be treated as an MCP server.
func hasExecutableConfig(item integrateRequestItem) bool {
	if item.Config == nil {
		return false
	}
	_, hasCmd := item.Config["command"]
	_, hasURL := item.Config["url"]
	return hasCmd || hasURL
}

// integrateSkill saves a discovered skill as a minimal skill directory in the workspace.
func integrateSkill(cfg *config.Config, item integrateRequestItem) error {
	if item.Name == "" {
		return fmt.Errorf("skill name is required")
	}

	home := os.Getenv("OCTAI_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		home = filepath.Join(userHome, ".octai")
	}

	skillDir := filepath.Join(home, "workspace", "skills", sanitizeName(item.Name))
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	skillFile := filepath.Join(skillDir, "SKILL.md")

	// If the scanner fetched the actual SKILL.md content from a GitHub repo, use it directly
	// but prepend proper frontmatter with name, description, and source_url.
	if item.Config != nil {
		if rawContent, ok := item.Config["content"].(string); ok && strings.TrimSpace(rawContent) != "" {
			sourceURL, _ := item.Config["source_url"].(string)

			description := item.Description
			if description == "" {
				description = extractFirstParagraph(rawContent)
			}
			if description == "" {
				description = item.Name
			}

			var sb strings.Builder
			sb.WriteString("---\n")
			sb.WriteString("name: ")
			sb.WriteString(item.Name)
			sb.WriteString("\n")
			sb.WriteString("description: ")
			sb.WriteString(strings.ReplaceAll(description, "\n", " "))
			sb.WriteString("\n")
			if sourceURL != "" {
				sb.WriteString("source_url: ")
				sb.WriteString(sourceURL)
				sb.WriteString("\n")
			}
			sb.WriteString("---\n\n")
			// Strip any existing frontmatter from the fetched content
			body := rawContent
			if strings.HasPrefix(strings.TrimSpace(body), "---") {
				if idx := strings.Index(body[3:], "---"); idx >= 0 {
					body = strings.TrimLeft(body[3+idx+3:], "\n\r")
				}
			}
			sb.WriteString(body)
			if !strings.HasSuffix(sb.String(), "\n") {
				sb.WriteString("\n")
			}

			if err := os.WriteFile(skillFile, []byte(sb.String()), 0o644); err != nil {
				return fmt.Errorf("writing SKILL.md: %w", err)
			}
			return nil
		}
	}

	// Fallback: build a minimal SKILL.md from name/description only
	description := item.Description
	if description == "" {
		description = item.Name
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("name: ")
	sb.WriteString(item.Name)
	sb.WriteString("\n")
	sb.WriteString("description: ")
	sb.WriteString(strings.ReplaceAll(description, "\n", " "))
	sb.WriteString("\n")
	sb.WriteString("---\n\n")
	sb.WriteString("# ")
	sb.WriteString(item.Name)
	sb.WriteString("\n\n")
	sb.WriteString(description)
	sb.WriteString("\n")

	if err := os.WriteFile(skillFile, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}
	return nil
}

// integrateReferenceURL saves a discovered reference URL to workspace/references.json.
func integrateReferenceURL(cfg *config.Config, item integrateRequestItem) error {
	if item.Name == "" {
		return fmt.Errorf("reference_url name is required")
	}

	home := os.Getenv("OCTAI_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		home = filepath.Join(userHome, ".octai")
	}

	filePath := filepath.Join(home, "workspace", "references.json")

	type refEntry struct {
		Name        string `json:"name"`
		URL         string `json:"url,omitempty"`
		Description string `json:"description,omitempty"`
	}

	var entries []refEntry
	if data, err := os.ReadFile(filePath); err == nil {
		_ = json.Unmarshal(data, &entries)
	}

	refURL := ""
	if item.Config != nil {
		if u, ok := item.Config["url"].(string); ok {
			refURL = u
		}
	}

	// Avoid duplicates by name+url
	for _, e := range entries {
		if e.Name == item.Name && e.URL == refURL {
			return nil
		}
	}

	entries = append(entries, refEntry{
		Name:        item.Name,
		URL:         refURL,
		Description: item.Description,
	})

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling references: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("writing references: %w", err)
	}
	return nil
}

// saveDiscoveredItem appends an item to a discovered_integrations.json file in the workspace.
func saveDiscoveredItem(cfg *config.Config, item integrateRequestItem) error {
	home := os.Getenv("OCTAI_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		home = filepath.Join(userHome, ".octai")
	}

	filePath := filepath.Join(home, "workspace", "discovered_integrations.json")

	// Read existing entries
	type entry struct {
		Type              string         `json:"type"`
		Name              string         `json:"name"`
		Description       string         `json:"description,omitempty"`
		Config            map[string]any `json:"config,omitempty"`
		SourceURL         string         `json:"source_url,omitempty"`
		LastChecked       string         `json:"last_checked,omitempty"`
		VersionIdentifier string         `json:"version_identifier,omitempty"`
	}

	var entries []entry
	if data, err := os.ReadFile(filePath); err == nil {
		_ = json.Unmarshal(data, &entries)
	}

	// Avoid duplicates by name+type
	for _, e := range entries {
		if e.Name == item.Name && e.Type == item.Type {
			return nil // already recorded
		}
	}

	entries = append(entries, entry{
		Type:        item.Type,
		Name:        item.Name,
		Description: item.Description,
		Config:      item.Config,
	})

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling entries: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0o600); err != nil {
		return fmt.Errorf("writing discovered integrations: %w", err)
	}
	return nil
}

// sanitizeName converts a name to a safe directory/file name.
func sanitizeName(name string) string {
	name = strings.ToLower(name)
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sb.WriteRune(r)
		} else if r == ' ' {
			sb.WriteRune('-')
		}
	}
	result := sb.String()
	if result == "" {
		return "item"
	}
	return result
}

// fetchGitHubContent fetches content from a GitHub repository URL.
// It reads the README, checks for common MCP config files, and returns
// a combined text blob suitable for LLM analysis.
func fetchGitHubContent(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	// Extract owner/repo from path: /owner/repo[/...]
	pathParts := strings.SplitN(strings.TrimPrefix(parsed.Path, "/"), "/", 3)
	if len(pathParts) < 2 {
		return "", fmt.Errorf("could not parse GitHub owner/repo from URL: %s", rawURL)
	}
	owner := pathParts[0]
	repo := pathParts[1]

	var sb strings.Builder

	// Fetch repo metadata
	repoMeta, err := githubAPIGet(fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo))
	if err == nil {
		sb.WriteString("=== Repository Metadata ===\n")
		sb.WriteString(repoMeta)
		sb.WriteString("\n\n")
	}

	// Fetch README
	readmeContent, err := githubGetReadme(owner, repo)
	if err == nil && readmeContent != "" {
		sb.WriteString("=== README ===\n")
		if len(readmeContent) > 20000 {
			readmeContent = readmeContent[:20000]
		}
		sb.WriteString(readmeContent)
		sb.WriteString("\n\n")
	}

	// Fetch known config files
	knownFiles := []string{".mcp.json", "mcp.json", "skills.json", "package.json"}
	for _, fname := range knownFiles {
		fileContent, err := githubGetFileContent(owner, repo, fname)
		if err != nil || fileContent == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("=== %s ===\n", fname))
		if len(fileContent) > 5000 {
			fileContent = fileContent[:5000]
		}
		sb.WriteString(fileContent)
		sb.WriteString("\n\n")
	}

	return sb.String(), nil
}

// fetchGitHubSkillFiles scans a GitHub repo tree for SKILL.md files and returns
// scannerItems with the actual file content in config["content"] and config["source_url"].
func fetchGitHubSkillFiles(owner, repo string) []scannerItem {
	// Fetch tree (recursive)
	treeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/HEAD?recursive=1", owner, repo)
	raw, err := githubAPIGet(treeURL)
	if err != nil {
		return nil
	}

	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}
	if err := json.Unmarshal([]byte(raw), &tree); err != nil {
		return nil
	}

	sourceRepoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)
	var items []scannerItem

	for _, node := range tree.Tree {
		if node.Type != "blob" || !strings.HasSuffix(node.Path, "/SKILL.md") {
			continue
		}
		// Derive skill name from directory containing SKILL.md
		parts := strings.Split(node.Path, "/")
		if len(parts) < 2 {
			continue
		}
		skillDirName := parts[len(parts)-2]

		content, err := githubGetFileContent(owner, repo, node.Path)
		if err != nil || content == "" {
			continue
		}

		// Parse description from first paragraph of the SKILL.md
		description := extractFirstParagraph(content)

		items = append(items, scannerItem{
			Type:        "skill",
			Name:        skillDirName,
			Description: description,
			Config: map[string]any{
				"content":    content,
				"source_url": sourceRepoURL,
			},
		})
	}
	return items
}

// extractFirstParagraph returns the first non-heading, non-empty paragraph from markdown.
func extractFirstParagraph(md string) string {
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "---") {
			continue
		}
		if len(line) > 200 {
			line = line[:200]
		}
		return line
	}
	return ""
}

// githubAPIGet makes an unauthenticated GET request to the GitHub API.
func githubAPIGet(apiURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "octai-scanner/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// githubGetReadme fetches and decodes the README for a repository.
func githubGetReadme(owner, repo string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", owner, repo)
	raw, err := githubAPIGet(apiURL)
	if err != nil {
		return "", err
	}

	var readmeResp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal([]byte(raw), &readmeResp); err != nil {
		return "", err
	}

	if readmeResp.Encoding == "base64" {
		// GitHub returns base64 with newlines
		cleaned := strings.ReplaceAll(readmeResp.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	return readmeResp.Content, nil
}

// githubGetFileContent fetches a file's decoded content from a repository.
func githubGetFileContent(owner, repo, filename string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, filename)
	raw, err := githubAPIGet(apiURL)
	if err != nil {
		return "", err
	}

	var fileResp struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.Unmarshal([]byte(raw), &fileResp); err != nil {
		return "", err
	}

	if fileResp.Encoding == "base64" {
		cleaned := strings.ReplaceAll(fileResp.Content, "\n", "")
		decoded, err := base64.StdEncoding.DecodeString(cleaned)
		if err != nil {
			return "", err
		}
		return string(decoded), nil
	}

	return fileResp.Content, nil
}

// fetchWebContent fetches a web page and extracts its text content.
func fetchWebContent(pageURL string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "octai-scanner/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP GET returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}

	// Strip HTML tags for cleaner LLM input
	text := stripHTMLTags(string(data))
	if len(text) > 30000 {
		text = text[:30000]
	}
	return text, nil
}

var linkRe = regexp.MustCompile(`href=["']?([^"'>\s]+)["']?`)

func extractLinks(content string, baseURL string) []string {
	matches := linkRe.FindAllStringSubmatch(content, -1)
	var links []string
	base, _ := url.Parse(baseURL)
	for _, match := range matches {
		if len(match) > 1 {
			u, err := url.Parse(match[1])
			if err != nil {
				continue
			}
			resolved := base.ResolveReference(u)
			links = append(links, resolved.String())
		}
	}
	return links
}

func isAllowedURL(link string, startURL string, sameDomain bool) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	if !strings.HasPrefix(u.Scheme, "http") {
		return false
	}
	if sameDomain {
		start, _ := url.Parse(startURL)
		return u.Host == start.Host
	}
	return true
}

func fetchWebContentCrawl(startURL string, depth int, maxPages int, sameDomain bool) (string, error) {
	visited := make(map[string]bool)
	toVisit := []string{startURL}
	var allContent strings.Builder

	// Default values if not specified
	if depth <= 0 {
		depth = 0
	}
	if maxPages <= 0 {
		maxPages = 1
	}

	for len(toVisit) > 0 && len(visited) < maxPages {
		current := toVisit[0]
		toVisit = toVisit[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		content, err := fetchWebContent(current)
		if err != nil {
			continue // Skip failed pages
		}
		allContent.WriteString("=== Page: " + current + " ===\n")
		allContent.WriteString(content)
		allContent.WriteString("\n\n---\n\n")

		// Extract links if we haven't reached max depth
		if depth > 0 {
			links := extractLinks(content, current)
			for _, link := range links {
				if isAllowedURL(link, startURL, sameDomain) && !visited[link] {
					toVisit = append(toVisit, link)
				}
			}
		}

		// Simple rate limiting
		time.Sleep(200 * time.Millisecond)
	}

	return allContent.String(), nil
}

// stripHTMLTags removes HTML tags and collapses whitespace.
func stripHTMLTags(html string) string {
	text := htmlTagRe.ReplaceAllString(html, " ")
	text = multiSpaceRe.ReplaceAllString(text, "\n")
	return strings.TrimSpace(text)
}

// pickScannerModel returns the best ModelConfig to use for the scanner LLM call.
// Priority: (1) configured default model, (2) first model with a non-empty API key,
// (3) first model overall.
func pickScannerModel(cfg *config.Config) *config.ModelConfig {
	if len(cfg.ModelList) == 0 {
		return nil
	}
	// Prefer the agent default model
	if defaultName := cfg.Agents.Defaults.ModelName; defaultName != "" {
		if mc, err := cfg.GetModelConfig(defaultName); err == nil {
			return mc
		}
	}
	// Fall back to first model with an API key
	for i := range cfg.ModelList {
		if cfg.ModelList[i].APIKey() != "" {
			return cfg.ModelList[i]
		}
	}
	return cfg.ModelList[0]
}

// analyzeContentWithLLM sends the fetched content to the configured LLM and parses the response.
func analyzeContentWithLLM(cfg *config.Config, content, sourceURL, urlType string) ([]scannerItem, error) {
	if len(cfg.ModelList) == 0 {
		return nil, fmt.Errorf("no models configured")
	}

	mc := pickScannerModel(cfg)
	provider, modelID, err := providers.CreateProviderFromConfig(mc)
	if err != nil {
		return nil, fmt.Errorf("creating provider: %w", err)
	}

	prompt := buildScannerPrompt(content, sourceURL, urlType)

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := provider.Chat(ctx, messages, nil, modelID, nil)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return parseScannerLLMResponse(resp.Content), nil
}

// buildScannerPrompt constructs the LLM prompt for URL analysis.
func buildScannerPrompt(content, sourceURL, urlType string) string {
	var sb strings.Builder
	sb.WriteString("You are analyzing a ")
	if urlType == "github" {
		sb.WriteString("GitHub repository")
	} else {
		sb.WriteString("web page")
	}
	sb.WriteString(" at: ")
	sb.WriteString(sourceURL)
	sb.WriteString("\n\n")

	isAwesomeList := strings.Contains(strings.ToLower(sourceURL), "awesome")

	if isAwesomeList {
		sb.WriteString("This appears to be a curated 'awesome list' containing links to useful resources, tools, and services.\n")
		sb.WriteString("Your primary task is to extract the most valuable curated links as 'reference_url' items.\n\n")
	}

	sb.WriteString("Identify any MCP (Model Context Protocol) servers, AI skills, tools, plugins, or curated reference links described in the following content.\n\n")
	sb.WriteString("For each item found, output a JSON array where each element has:\n")
	sb.WriteString(`- "type": one of "mcp_server", "skill", "tool", "plugin", "reference_url"` + "\n")
	sb.WriteString(`- "name": the identifier/name of the item` + "\n")
	sb.WriteString(`- "description": a brief description` + "\n")
	sb.WriteString(`- "config": an object with relevant configuration fields (e.g. command, args, url, type for MCP; "url" for reference_url)` + "\n\n")

	if isAwesomeList {
		sb.WriteString("For 'reference_url' items: extract up to 25 of the most useful/notable links. Set config.url to the full URL, name to the resource title, description to the annotation from the list.\n\n")
	}

	sb.WriteString("Output ONLY a JSON array wrapped in ```json ... ``` fences. If nothing is found, output an empty array.\n\n")
	sb.WriteString("=== CONTENT ===\n")
	sb.WriteString(content)
	return sb.String()
}

// parseScannerLLMResponse parses the LLM output and extracts scannerItems.
// It looks for a ```json ... ``` block first, then falls back to raw JSON array parsing.
func parseScannerLLMResponse(response string) []scannerItem {
	// Try to find ```json ... ``` fence
	jsonContent := extractJSONBlock(response)
	if jsonContent == "" {
		// Fallback: look for a raw JSON array
		start := strings.Index(response, "[")
		end := strings.LastIndex(response, "]")
		if start >= 0 && end > start {
			jsonContent = response[start : end+1]
		}
	}

	if jsonContent == "" {
		return []scannerItem{}
	}

	var items []scannerItem
	if err := json.Unmarshal([]byte(jsonContent), &items); err != nil {
		// Try as array of map[string]any and convert
		var rawItems []map[string]any
		if err2 := json.Unmarshal([]byte(jsonContent), &rawItems); err2 != nil {
			return []scannerItem{}
		}
		for _, raw := range rawItems {
			item := scannerItem{}
			if t, ok := raw["type"].(string); ok {
				item.Type = t
			}
			if n, ok := raw["name"].(string); ok {
				item.Name = n
			}
			if d, ok := raw["description"].(string); ok {
				item.Description = d
			}
			if c, ok := raw["config"].(map[string]any); ok {
				item.Config = c
			}
			if item.Name != "" {
				items = append(items, item)
			}
		}
	}

	return items
}

// extractJSONBlock extracts content from a ```json ... ``` fenced block.
func extractJSONBlock(s string) string {
	const fence = "```json"
	const closeFence = "```"

	start := strings.Index(s, fence)
	if start < 0 {
		return ""
	}
	start += len(fence)

	end := strings.Index(s[start:], closeFence)
	if end < 0 {
		return ""
	}

	return strings.TrimSpace(s[start : start+end])
}

// buildOpenAIRequest constructs an OpenAI-compatible chat completion request body.
// This is used as a fallback when no provider is configured.
func buildOpenAIRequest(model, prompt string) []byte {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
	}
	req := request{
		Model: model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}
	data, _ := json.Marshal(req)
	return data
}

// callOpenAICompat calls an OpenAI-compatible API directly via net/http.
// Used as a fallback if the providers package cannot be used.
func callOpenAICompat(apiBase, apiKey, model, prompt string) (string, error) {
	reqBody := buildOpenAIRequest(model, prompt)

	endpointURL := strings.TrimRight(apiBase, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, endpointURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in API response")
	}
	return result.Choices[0].Message.Content, nil
}
