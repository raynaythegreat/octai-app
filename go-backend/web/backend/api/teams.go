package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/marketplace"
)

// TeamMemberResponse is the API representation of a single team agent.
type TeamMemberResponse struct {
	AgentID string `json:"agent_id"`
	Name    string `json:"name,omitempty"`
	Role    string `json:"role,omitempty"`
}

// TeamResponse is the API representation of a configured team.
type TeamResponse struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	OrchestratorID string               `json:"orchestrator_id"`
	Members        []TeamMemberResponse `json:"members"`
	SharedKBPath   string               `json:"shared_kb_path,omitempty"`
	TokenBudget    int                  `json:"token_budget,omitempty"`
	MaxConcurrent  int                  `json:"max_concurrent,omitempty"`
}

// TeamTemplateResponse is the API representation of a marketplace team template.
type TeamTemplateResponse struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Category    string                     `json:"category"`
	Agents      []marketplace.AgentSpec    `json:"agents"`
	Workflows   []marketplace.WorkflowSpec `json:"workflows,omitempty"`
	Author      string                     `json:"author"`
	Price       float64                    `json:"price"`
	Rating      float64                    `json:"rating"`
	Downloads   int64                      `json:"downloads"`
	Tags        []string                   `json:"tags,omitempty"`
	CreatedAt   string                     `json:"created_at"`
	UpdatedAt   string                     `json:"updated_at"`
}

func (h *Handler) registerTeamRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/teams", h.handleListTeams)
	mux.HandleFunc("POST /api/v1/teams", h.handleCreateTeam)
	mux.HandleFunc("GET /api/v1/teams/templates", h.handleListTeamTemplates)
	mux.HandleFunc("GET /api/v1/teams/{id}", h.handleGetTeam)
	mux.HandleFunc("PUT /api/v1/teams/{id}", h.handleUpdateTeam)
	mux.HandleFunc("DELETE /api/v1/teams/{id}", h.handleDeleteTeam)
}

func (h *Handler) handleListTeams(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	// Build a name map for agents so we can enrich responses.
	agentNames := make(map[string]string, len(cfg.Agents.List))
	agentRoles := make(map[string]string, len(cfg.Agents.List))
	for _, ac := range cfg.Agents.List {
		if ac.Name != "" {
			agentNames[ac.ID] = ac.Name
		}
		if ac.Role != "" {
			agentRoles[ac.ID] = ac.Role
		}
	}

	teams := make([]TeamResponse, 0, len(cfg.Agents.Teams))
	for _, tc := range cfg.Agents.Teams {
		members := make([]TeamMemberResponse, 0, len(tc.MemberIDs))
		for _, mid := range tc.MemberIDs {
			members = append(members, TeamMemberResponse{
				AgentID: mid,
				Name:    agentNames[mid],
				Role:    agentRoles[mid],
			})
		}
		name := tc.Name
		if name == "" {
			name = tc.ID
		}
		teams = append(teams, TeamResponse{
			ID:             tc.ID,
			Name:           name,
			OrchestratorID: tc.OrchestratorID,
			Members:        members,
			SharedKBPath:   tc.SharedKBPath,
			TokenBudget:    tc.TokenBudget,
			MaxConcurrent:  tc.MaxConcurrent,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"teams": teams,
		"total": len(teams),
	})
}

func (h *Handler) handleGetTeam(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		http.Error(w, "team id is required", http.StatusBadRequest)
		return
	}

	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}

	agentNames := make(map[string]string, len(cfg.Agents.List))
	agentRoles := make(map[string]string, len(cfg.Agents.List))
	for _, ac := range cfg.Agents.List {
		agentNames[ac.ID] = ac.Name
		agentRoles[ac.ID] = ac.Role
	}

	for _, tc := range cfg.Agents.Teams {
		if tc.ID != teamID {
			continue
		}
		members := make([]TeamMemberResponse, 0, len(tc.MemberIDs))
		for _, mid := range tc.MemberIDs {
			members = append(members, TeamMemberResponse{
				AgentID: mid,
				Name:    agentNames[mid],
				Role:    agentRoles[mid],
			})
		}
		name := tc.Name
		if name == "" {
			name = tc.ID
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TeamResponse{
			ID:             tc.ID,
			Name:           name,
			OrchestratorID: tc.OrchestratorID,
			Members:        members,
			SharedKBPath:   tc.SharedKBPath,
			TokenBudget:    tc.TokenBudget,
			MaxConcurrent:  tc.MaxConcurrent,
		})
		return
	}

	http.Error(w, "team not found", http.StatusNotFound)
}

// TeamRequest is the request body for creating/updating a team.
type TeamRequest struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	OrchestratorID string   `json:"orchestrator_id"`
	MemberIDs      []string `json:"member_ids"`
	SharedKBPath   string   `json:"shared_kb_path,omitempty"`
	TokenBudget    int      `json:"token_budget,omitempty"`
	MaxConcurrent  int      `json:"max_concurrent,omitempty"`
}

func (h *Handler) handleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var req TeamRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
}

func (h *Handler) handleUpdateTeam(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		http.Error(w, "team id is required", http.StatusBadRequest)
		return
	}
	var req TeamRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
}

func (h *Handler) handleDeleteTeam(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("id")
	if teamID == "" {
		http.Error(w, "team id is required", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to load config: %v", err), http.StatusInternalServerError)
		return
	}
	newTeams := make([]config.TeamConfig, 0, len(cfg.Agents.Teams))
	found := false
	for _, t := range cfg.Agents.Teams {
		if t.ID == teamID {
			found = true
			continue
		}
		newTeams = append(newTeams, t)
	}
	if !found {
		http.Error(w, "team not found", http.StatusNotFound)
		return
	}
	cfg.Agents.Teams = newTeams
	if err := config.SaveConfig(h.configPath, cfg); err != nil {
		http.Error(w, fmt.Sprintf("failed to save config: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListTeamTemplates(w http.ResponseWriter, r *http.Request) {
	raw := marketplace.BuiltinTeamTemplates()
	templates := make([]TeamTemplateResponse, 0, len(raw))
	for _, t := range raw {
		cr := t.CreatedAt
		if cr.IsZero() {
			cr = time.Now()
		}
		ur := t.UpdatedAt
		if ur.IsZero() {
			ur = time.Now()
		}
		templates = append(templates, TeamTemplateResponse{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Category:    string(t.Category),
			Agents:      t.Agents,
			Workflows:   t.Workflows,
			Author:      t.Author,
			Price:       t.Price,
			Rating:      t.Rating,
			Downloads:   t.Downloads,
			Tags:        t.Tags,
			CreatedAt:   cr.Format(time.RFC3339),
			UpdatedAt:   ur.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"templates": templates,
		"total":     len(templates),
	})
}
