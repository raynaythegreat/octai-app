package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/agent"
)

// LoopTaskResponse is the API representation of a scheduled loop task.
type LoopTaskResponse struct {
	ID           string   `json:"id"`
	Prompt       string   `json:"prompt"`
	ScheduleType string   `json:"schedule_type"`
	Interval     string   `json:"interval,omitempty"`
	CronExpr     string   `json:"cron_expr,omitempty"`
	RunAt        *string  `json:"run_at,omitempty"`
	Timezone     string   `json:"timezone,omitempty"`
	MaxRuns      int      `json:"max_runs,omitempty"`
	RunCount     int      `json:"run_count"`
	Status       string   `json:"status"`
	NextRunAt    *string  `json:"next_run_at,omitempty"`
	LastRunAt    *string  `json:"last_run_at,omitempty"`
	CreatedAt    string   `json:"created_at"`
	Tags         []string `json:"tags,omitempty"`
}

// LoopCreateRequest is the body for creating a new loop task.
type LoopCreateRequest struct {
	Prompt       string   `json:"prompt"`
	ScheduleType string   `json:"schedule_type"`
	Interval     string   `json:"interval,omitempty"`
	CronExpr     string   `json:"cron_expr,omitempty"`
	RunAt        string   `json:"run_at,omitempty"`
	Timezone     string   `json:"timezone,omitempty"`
	MaxRuns      int      `json:"max_runs,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

func (h *Handler) registerLoopRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/loops", h.handleListLoops)
	mux.HandleFunc("POST /api/v1/loops", h.handleCreateLoop)
	mux.HandleFunc("GET /api/v1/loops/{id}", h.handleGetLoop)
	mux.HandleFunc("DELETE /api/v1/loops/{id}", h.handleDeleteLoop)
	mux.HandleFunc("POST /api/v1/loops/{id}/pause", h.handlePauseLoop)
	mux.HandleFunc("POST /api/v1/loops/{id}/resume", h.handleResumeLoop)
}

func (h *Handler) getLoopScheduler() *agent.LoopScheduler {
	h.loopSchedOnce.Do(func() {
		h.loopSched = agent.NewLoopScheduler(nil)
	})
	return h.loopSched
}

func toLoopResponse(task agent.LoopTask) LoopTaskResponse {
	r := LoopTaskResponse{
		ID:           task.ID,
		Prompt:       task.Prompt,
		ScheduleType: string(task.ScheduleType),
		CronExpr:     task.CronExpr,
		Timezone:     task.Timezone,
		MaxRuns:      task.MaxRuns,
		RunCount:     task.RunCount,
		Status:       task.Status,
		CreatedAt:    task.CreatedAt.Format(time.RFC3339),
		Tags:         task.Tags,
	}
	if task.Interval > 0 {
		r.Interval = task.Interval.String()
	}
	if task.RunAt != nil {
		s := task.RunAt.Format(time.RFC3339)
		r.RunAt = &s
	}
	if !task.NextRunAt.IsZero() {
		s := task.NextRunAt.Format(time.RFC3339)
		r.NextRunAt = &s
	}
	if !task.LastRunAt.IsZero() {
		s := task.LastRunAt.Format(time.RFC3339)
		r.LastRunAt = &s
	}
	return r
}

func (h *Handler) handleListLoops(w http.ResponseWriter, r *http.Request) {
	sched := h.getLoopScheduler()
	tasks := sched.List()
	responses := make([]LoopTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		responses = append(responses, toLoopResponse(t))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"loops": responses,
		"total": len(responses),
	})
}

func (h *Handler) handleCreateLoop(w http.ResponseWriter, r *http.Request) {
	var req LoopCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if req.ScheduleType == "" {
		req.ScheduleType = "interval"
	}

	task := agent.LoopTask{
		Prompt:       req.Prompt,
		ScheduleType: agent.ScheduleType(req.ScheduleType),
		CronExpr:     req.CronExpr,
		Timezone:     req.Timezone,
		MaxRuns:      req.MaxRuns,
		Tags:         req.Tags,
	}

	if req.Interval != "" {
		d, err := time.ParseDuration(req.Interval)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid interval: %v", err), http.StatusBadRequest)
			return
		}
		task.Interval = d
	}

	if req.RunAt != "" {
		t, err := time.Parse(time.RFC3339, req.RunAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid run_at: %v", err), http.StatusBadRequest)
			return
		}
		task.RunAt = &t
	}

	sched := h.getLoopScheduler()
	if err := sched.Add(task); err != nil {
		http.Error(w, fmt.Sprintf("failed to create loop: %v", err), http.StatusInternalServerError)
		return
	}

	// Retrieve the created task to get generated ID.
	created, ok := sched.Get(task.ID)
	if !ok {
		// Fallback to what we have.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(toLoopResponse(task))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toLoopResponse(*created))
}

func (h *Handler) handleGetLoop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "loop id is required", http.StatusBadRequest)
		return
	}
	sched := h.getLoopScheduler()
	task, ok := sched.Get(id)
	if !ok {
		http.Error(w, "loop not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLoopResponse(*task))
}

func (h *Handler) handleDeleteLoop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "loop id is required", http.StatusBadRequest)
		return
	}
	sched := h.getLoopScheduler()
	if err := sched.Remove(id); err != nil {
		http.Error(w, fmt.Sprintf("loop not found: %v", err), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handlePauseLoop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "loop id is required", http.StatusBadRequest)
		return
	}
	sched := h.getLoopScheduler()
	if err := sched.Pause(id); err != nil {
		http.Error(w, fmt.Sprintf("failed to pause loop: %v", err), http.StatusNotFound)
		return
	}
	task, _ := sched.Get(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLoopResponse(*task))
}

func (h *Handler) handleResumeLoop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "loop id is required", http.StatusBadRequest)
		return
	}
	sched := h.getLoopScheduler()
	if err := sched.Resume(id); err != nil {
		http.Error(w, fmt.Sprintf("failed to resume loop: %v", err), http.StatusNotFound)
		return
	}
	task, _ := sched.Get(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toLoopResponse(*task))
}
