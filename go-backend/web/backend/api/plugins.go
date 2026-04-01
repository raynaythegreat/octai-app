package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/raynaythegreat/octai-app/pkg/config"
)

// Plugin is a unified view of any externally-integrated capability:
// MCP servers, scanner-discovered tools/connections, and non-builtin skills.
type Plugin struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // "mcp_server", "skill", "tool", "plugin", "connection", "other"
	Description string `json:"description"` // human-readable summary
	Enabled     bool   `json:"enabled"`     // whether the plugin is active
	Source      string `json:"source"`      // scanner URL or "manual"
}

func (h *Handler) registerPluginRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/plugins", h.handleListPlugins)
	mux.HandleFunc("DELETE /api/plugins/{name}", h.handleDeletePlugin)
	mux.HandleFunc("PATCH /api/plugins/{name}", h.handleTogglePlugin)
}

// handleListPlugins returns all installed plugins from MCP servers, skills (workspace),
// and the discovered_integrations.json list.
//
//	GET /api/plugins
func (h *Handler) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	var plugins []Plugin

	// 1. MCP servers from config
	for name, srv := range cfg.Tools.MCP.Servers {
		desc := ""
		if len(srv.Args) > 0 {
			desc = strings.Join(srv.Args, " ")
		}
		plugins = append(plugins, Plugin{
			Name:        name,
			Type:        "mcp_server",
			Description: desc,
			Enabled:     srv.Enabled,
			Source:      "manual",
		})
	}

	// 2. Skills in workspace/skills/
	workspace := cfg.WorkspacePath()
	skillsDir := filepath.Join(workspace, "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillName := e.Name()
			desc := readSkillDescription(filepath.Join(skillsDir, skillName, "SKILL.md"))
			plugins = append(plugins, Plugin{
				Name:        skillName,
				Type:        "skill",
				Description: desc,
				Enabled:     true,
				Source:      "workspace",
			})
		}
	}

	// 3. Discovered integrations (tools, connections, etc.)
	discovered := loadDiscoveredPlugins(cfg)
	plugins = append(plugins, discovered...)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"plugins": plugins})
}

// handleDeletePlugin removes a plugin by name and type (passed via query param ?type=).
//
//	DELETE /api/plugins/{name}
func (h *Handler) handleDeletePlugin(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	pluginType := r.URL.Query().Get("type")

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	switch pluginType {
	case "mcp_server":
		delete(cfg.Tools.MCP.Servers, name)
		if err := config.SaveConfig(h.configPath, cfg); err != nil {
			http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
			return
		}
	case "skill":
		workspace := cfg.WorkspacePath()
		skillDir := filepath.Join(workspace, "skills", sanitizeName(name))
		if err := os.RemoveAll(skillDir); err != nil && !os.IsNotExist(err) {
			http.Error(w, fmt.Sprintf("failed to remove skill: %v", err), http.StatusInternalServerError)
			return
		}
	default:
		// Remove from discovered_integrations.json
		if err := removeDiscoveredPlugin(cfg, name); err != nil {
			http.Error(w, fmt.Sprintf("failed to remove plugin: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleTogglePlugin enables or disables a plugin.
//
//	PATCH /api/plugins/{name}
func (h *Handler) handleTogglePlugin(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	srv, ok := cfg.Tools.MCP.Servers[name]
	if !ok {
		http.Error(w, "plugin not found or not toggleable", http.StatusNotFound)
		return
	}
	srv.Enabled = body.Enabled
	cfg.Tools.MCP.Servers[name] = srv

	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// readSkillDescription extracts the description from a SKILL.md file's frontmatter or first paragraph.
func readSkillDescription(skillMDPath string) string {
	data, err := os.ReadFile(skillMDPath)
	if err != nil {
		return ""
	}
	content := string(data)
	// Look for "description:" in frontmatter
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "description:") {
			return strings.TrimSpace(line[len("description:"):])
		}
	}
	// Fall back to first non-empty, non-heading line
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			if len(line) > 120 {
				return line[:120] + "..."
			}
			return line
		}
	}
	return ""
}

// loadDiscoveredPlugins reads discovered_integrations.json and returns Plugin entries.
func loadDiscoveredPlugins(cfg *config.Config) []Plugin {
	filePath := discoveredIntegrationsPath(cfg)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	type entry struct {
		Type        string         `json:"type"`
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Config      map[string]any `json:"config,omitempty"`
	}

	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}

	plugins := make([]Plugin, 0, len(entries))
	for _, e := range entries {
		plugins = append(plugins, Plugin{
			Name:        e.Name,
			Type:        e.Type,
			Description: e.Description,
			Enabled:     true,
			Source:      "scanner",
		})
	}
	return plugins
}

// removeDiscoveredPlugin deletes an entry from discovered_integrations.json.
func removeDiscoveredPlugin(cfg *config.Config, name string) error {
	filePath := discoveredIntegrationsPath(cfg)
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	type entry struct {
		Type        string         `json:"type"`
		Name        string         `json:"name"`
		Description string         `json:"description,omitempty"`
		Config      map[string]any `json:"config,omitempty"`
	}

	var entries []entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	filtered := entries[:0]
	for _, e := range entries {
		if e.Name != name {
			filtered = append(filtered, e)
		}
	}

	out, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, out, 0o644)
}

func discoveredIntegrationsPath(cfg *config.Config) string {
	home := os.Getenv("OCTAI_HOME")
	if home == "" {
		userHome, _ := os.UserHomeDir()
		home = filepath.Join(userHome, ".octai")
	}
	return filepath.Join(home, "workspace", "discovered_integrations.json")
}
