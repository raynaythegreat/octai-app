package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/knowledge"
)

// KnowledgeDocumentResponse is the API representation of a knowledge document.
type KnowledgeDocumentResponse struct {
	ID        string `json:"id"`
	TeamID    string `json:"team_id"`
	Title     string `json:"title"`
	SourceURL string `json:"source_url,omitempty"`
	ChunkSize int    `json:"chunk_count,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// KnowledgeSearchResult is the API representation of a search result.
type KnowledgeSearchResult struct {
	ChunkID    string  `json:"chunk_id"`
	DocumentID string  `json:"document_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
}

func (h *Handler) registerKnowledgeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/teams/{team_id}/knowledge", h.handleListKnowledge)
	mux.HandleFunc("POST /api/v1/teams/{team_id}/knowledge", h.handleAddKnowledge)
	mux.HandleFunc("DELETE /api/v1/teams/{team_id}/knowledge/{doc_id}", h.handleDeleteKnowledge)
	mux.HandleFunc("POST /api/v1/teams/{team_id}/knowledge/search", h.handleSearchKnowledge)
}

// openTeamKBStore opens the SQLiteStore for a team's knowledge base.
// The path is derived the same way as pkg/agent/loop.go getOrOpenKnowledgeStore.
func (h *Handler) openTeamKBStore(teamID string) (knowledge.KnowledgeStore, error) {
	cfg, err := config.LoadConfig(h.configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	dbPath := ""
	for _, tc := range cfg.Agents.Teams {
		if tc.ID == teamID && tc.SharedKBPath != "" {
			dbPath = tc.SharedKBPath
			break
		}
	}
	if dbPath == "" {
		workspace := cfg.WorkspacePath()
		if workspace == "" {
			workspace = "workspace"
		}
		dbPath = filepath.Join(workspace, "knowledge", teamID+".db")
	}

	return knowledge.NewSQLiteStore(dbPath)
}

func (h *Handler) handleListKnowledge(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("team_id")
	if teamID == "" {
		http.Error(w, "team_id is required", http.StatusBadRequest)
		return
	}

	store, err := h.openTeamKBStore(teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open knowledge store: %v", err), http.StatusInternalServerError)
		return
	}
	defer store.Close()

	docs, err := store.ListDocuments(r.Context(), teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list documents: %v", err), http.StatusInternalServerError)
		return
	}

	resp := make([]KnowledgeDocumentResponse, 0, len(docs))
	for _, d := range docs {
		resp = append(resp, KnowledgeDocumentResponse{
			ID:        d.ID,
			TeamID:    d.TeamID,
			Title:     d.Title,
			SourceURL: d.SourceURL,
			CreatedAt: d.CreatedAt.Format(time.RFC3339),
			UpdatedAt: d.UpdatedAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"documents": resp,
		"total":     len(resp),
	})
}

func (h *Handler) handleAddKnowledge(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("team_id")
	if teamID == "" {
		http.Error(w, "team_id is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20)) // 2 MB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Title     string `json:"title"`
		Content   string `json:"content"`
		SourceURL string `json:"source_url,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.Title == "" || req.Content == "" {
		http.Error(w, "title and content are required", http.StatusBadRequest)
		return
	}

	store, err := h.openTeamKBStore(teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open knowledge store: %v", err), http.StatusInternalServerError)
		return
	}
	defer store.Close()

	docID, err := knowledge.IngestMarkdown(r.Context(), store, teamID, req.Title, req.Content, req.SourceURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to add document: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":      docID,
		"team_id": teamID,
		"status":  "added",
	})
}

func (h *Handler) handleDeleteKnowledge(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("team_id")
	docID := r.PathValue("doc_id")
	if teamID == "" || docID == "" {
		http.Error(w, "team_id and doc_id are required", http.StatusBadRequest)
		return
	}

	store, err := h.openTeamKBStore(teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open knowledge store: %v", err), http.StatusInternalServerError)
		return
	}
	defer store.Close()

	if err := store.DeleteDocument(r.Context(), docID); err != nil {
		http.Error(w, fmt.Sprintf("failed to delete document: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleSearchKnowledge(w http.ResponseWriter, r *http.Request) {
	teamID := r.PathValue("team_id")
	if teamID == "" {
		http.Error(w, "team_id is required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 32<<10)) // 32 KB limit
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req struct {
		Query string `json:"query"`
		Limit string `json:"limit,omitempty"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		http.Error(w, "query is required", http.StatusBadRequest)
		return
	}

	limit := 5
	if req.Limit != "" {
		if n, err := strconv.Atoi(req.Limit); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 20 {
		limit = 20
	}

	store, err := h.openTeamKBStore(teamID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open knowledge store: %v", err), http.StatusInternalServerError)
		return
	}
	defer store.Close()

	results, err := store.Search(r.Context(), teamID, req.Query, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("search failed: %v", err), http.StatusInternalServerError)
		return
	}

	resp := make([]KnowledgeSearchResult, 0, len(results))
	for _, sr := range results {
		resp = append(resp, KnowledgeSearchResult{
			ChunkID:    sr.ChunkID,
			DocumentID: sr.DocumentID,
			Title:      sr.Title,
			Content:    sr.Content,
			Score:      sr.Score,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": resp,
		"total":   len(resp),
	})
}
