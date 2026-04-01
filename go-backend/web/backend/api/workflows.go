package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/workflow"
)

// WorkflowRunResponse is the API representation of a workflow run record.
type WorkflowRunResponse struct {
	ID          string  `json:"id"`
	WorkflowID  string  `json:"workflow_id"`
	Status      string  `json:"status"`
	StartedAt   *string `json:"started_at,omitempty"`
	CompletedAt *string `json:"completed_at,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// WorkflowSummaryResponse is the API representation of a stored workflow.
type WorkflowSummaryResponse struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	TeamID      string                `json:"team_id,omitempty"`
	TriggerKind string                `json:"trigger_kind"`
	NodeCount   int                   `json:"node_count"`
	RecentRuns  []WorkflowRunResponse `json:"recent_runs,omitempty"`
	CreatedAt   string                `json:"created_at"`
	UpdatedAt   string                `json:"updated_at"`
}

func openWorkflowStore(configPath string) (*workflow.SQLiteWorkflowStore, error) {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	workspace := cfg.WorkspacePath()
	if workspace == "" {
		workspace = "workspace"
	}
	dbPath := filepath.Join(workspace, "workflows.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create workflow db dir: %w", err)
	}
	return workflow.NewSQLiteWorkflowStore(dbPath)
}

func toRunResponse(run workflow.WorkflowRun) WorkflowRunResponse {
	rr := WorkflowRunResponse{
		ID:         run.ID,
		WorkflowID: run.WorkflowID,
		Status:     string(run.Status),
		Error:      run.Error,
	}
	if !run.StartedAt.IsZero() {
		s := run.StartedAt.Format(time.RFC3339)
		rr.StartedAt = &s
	}
	if !run.EndedAt.IsZero() {
		s := run.EndedAt.Format(time.RFC3339)
		rr.CompletedAt = &s
	}
	return rr
}

func toWorkflowSummary(def workflow.WorkflowDefinition, runs []workflow.WorkflowRun) WorkflowSummaryResponse {
	triggerKind := ""
	if len(def.Triggers) > 0 {
		triggerKind = def.Triggers[0].Type
	}
	runResponses := make([]WorkflowRunResponse, 0, len(runs))
	for _, r := range runs {
		runResponses = append(runResponses, toRunResponse(r))
	}
	cr := def.CreatedAt
	if cr.IsZero() {
		cr = time.Now()
	}
	ur := def.UpdatedAt
	if ur.IsZero() {
		ur = time.Now()
	}
	return WorkflowSummaryResponse{
		ID:          def.ID,
		Name:        def.Name,
		Description: def.Description,
		TeamID:      def.TeamID,
		TriggerKind: triggerKind,
		NodeCount:   len(def.Nodes),
		RecentRuns:  runResponses,
		CreatedAt:   cr.Format(time.RFC3339),
		UpdatedAt:   ur.Format(time.RFC3339),
	}
}

func (h *Handler) registerWorkflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/workflows", h.handleListWorkflows)
	mux.HandleFunc("POST /api/v1/workflows", h.handleCreateWorkflow)
	mux.HandleFunc("GET /api/v1/workflows/{id}", h.handleGetWorkflow)
	mux.HandleFunc("DELETE /api/v1/workflows/{id}", h.handleDeleteWorkflow)
	mux.HandleFunc("POST /api/v1/workflows/{id}/run", h.handleTriggerWorkflow)
}

// WorkflowCreateRequest is the body for creating a new workflow.
type WorkflowCreateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	TeamID      string          `json:"team_id,omitempty"`
	TriggerKind string          `json:"trigger_kind,omitempty"`
	Definition  json.RawMessage `json:"definition,omitempty"`
}

func (h *Handler) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req WorkflowCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	store, err := openWorkflowStore(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open workflow store: %v", err), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	def := workflow.WorkflowDefinition{
		ID:          fmt.Sprintf("wf-%d", now.UnixMilli()),
		Name:        req.Name,
		Description: req.Description,
		TeamID:      req.TeamID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.TriggerKind != "" {
		def.Triggers = []workflow.TriggerDef{{Type: req.TriggerKind}}
	}
	// If the caller supplied a raw definition, try to merge nodes/triggers from it.
	if len(req.Definition) > 0 {
		var partial workflow.WorkflowDefinition
		if err := json.Unmarshal(req.Definition, &partial); err == nil {
			if len(partial.Nodes) > 0 {
				def.Nodes = partial.Nodes
			}
			if len(partial.Triggers) > 0 {
				def.Triggers = partial.Triggers
			}
		}
	}

	if err := store.SaveWorkflow(r.Context(), def); err != nil {
		http.Error(w, fmt.Sprintf("failed to save workflow: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toWorkflowSummary(def, nil))
}

func (h *Handler) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "workflow id is required", http.StatusBadRequest)
		return
	}
	store, err := openWorkflowStore(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open workflow store: %v", err), http.StatusInternalServerError)
		return
	}
	if err := store.DeleteWorkflow(r.Context(), id); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete workflow: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	store, err := openWorkflowStore(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open workflow store: %v", err), http.StatusInternalServerError)
		return
	}

	// List workflows for all teams by passing empty string.
	// The store returns workflows where team_id matches; passing "" returns global ones.
	// For a full listing we also try to load per-team workflows from config.
	teamID := r.URL.Query().Get("team_id")
	defs, err := store.ListWorkflows(r.Context(), teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list workflows: %v", err), http.StatusInternalServerError)
		return
	}

	summaries := make([]WorkflowSummaryResponse, 0, len(defs))
	for _, def := range defs {
		summaries = append(summaries, toWorkflowSummary(def, nil))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"workflows": summaries,
		"total":     len(summaries),
	})
}

func (h *Handler) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "workflow id is required", http.StatusBadRequest)
		return
	}

	store, err := openWorkflowStore(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open workflow store: %v", err), http.StatusInternalServerError)
		return
	}

	def, err := store.GetWorkflow(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get workflow: %v", err), http.StatusInternalServerError)
		return
	}
	if def == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	runs, _ := store.ListRuns(r.Context(), id, 10)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toWorkflowSummary(*def, runs))
}

func (h *Handler) handleTriggerWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "workflow id is required", http.StatusBadRequest)
		return
	}

	store, err := openWorkflowStore(h.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open workflow store: %v", err), http.StatusInternalServerError)
		return
	}

	def, err := store.GetWorkflow(r.Context(), id)
	if err != nil || def == nil {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	// Create a pending run — actual execution is handled by the agent loop
	// when it picks up the pending run via the workflow engine.
	run := &workflow.WorkflowRun{
		ID:         fmt.Sprintf("run-%d", time.Now().UnixNano()),
		WorkflowID: id,
		Status:     workflow.RunStatusPending,
		StartedAt:  time.Now(),
	}
	if err := store.SaveRun(r.Context(), run); err != nil {
		http.Error(w, fmt.Sprintf("failed to create run: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"run_id":  run.ID,
		"status":  string(run.Status),
		"message": "workflow run queued",
	})
}
