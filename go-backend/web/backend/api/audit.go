package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/audit"
	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/pkg/tenant"
)

type AuditLogResponse struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	UserID         string                 `json:"user_id"`
	Action         string                 `json:"action"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     string                 `json:"resource_id"`
	Changes        map[string]interface{} `json:"changes,omitempty"`
	IPAddress      string                 `json:"ip_address,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`
	Location       *audit.Location        `json:"location,omitempty"`
	Status         string                 `json:"status"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	Timestamp      string                 `json:"timestamp"`
}

type AuditListResponse struct {
	Logs    []AuditLogResponse `json:"logs"`
	Total   int64              `json:"total"`
	Limit   int                `json:"limit"`
	Offset  int                `json:"offset"`
	HasMore bool               `json:"has_more"`
}

type AuditRetentionResponse struct {
	OrganizationID string `json:"organization_id"`
	RetentionDays  int    `json:"retention_days"`
}

type AuditRetentionRequest struct {
	RetentionDays int `json:"retention_days"`
}

type AuditStatsResponse struct {
	OrganizationID string                 `json:"organization_id"`
	TotalLogs      int64                  `json:"total_logs"`
	ByAction       map[string]int64       `json:"by_action"`
	ByResource     map[string]int64       `json:"by_resource"`
	ByUser         map[string]int64       `json:"by_user"`
	RecentActivity []AuditActivitySummary `json:"recent_activity"`
}

type AuditActivitySummary struct {
	Date    string `json:"date"`
	Count   int64  `json:"count"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
}

func (h *Handler) getAuditStore() audit.AuditStore {
	if h.auditCache == nil {
		h.auditCache = make(map[string]interface{})
	}
	if store, ok := h.auditCache["audit_store"]; ok {
		return store.(audit.AuditStore)
	}
	store := audit.NewMemoryAuditStore(100000)
	h.auditCache["audit_store"] = store
	return store
}

func (h *Handler) registerAuditRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/organizations/{id}/audit", h.handleListAuditLogs)
	mux.HandleFunc("GET /api/v2/organizations/{id}/audit/export", h.handleExportAuditLogs)
	mux.HandleFunc("GET /api/v2/organizations/{id}/audit/stats", h.handleGetAuditStats)
	mux.HandleFunc("GET /api/v2/organizations/{id}/audit/retention", h.handleGetAuditRetention)
	mux.HandleFunc("PUT /api/v2/organizations/{id}/audit/retention", h.handleUpdateAuditRetention)
}

func (h *Handler) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	if !tc.HasFeature("audit_logs") {
		writeJSONError(w, "audit logs feature not available on current plan", http.StatusForbidden)
		return
	}

	filters := parseAuditFilters(r)

	store := h.getAuditStore()
	result, err := store.Query(r.Context(), orgID, filters)
	if err != nil {
		logger.ErrorC("api", "failed to query audit logs: "+err.Error())
		writeJSONError(w, "failed to query audit logs", http.StatusInternalServerError)
		return
	}

	response := AuditListResponse{
		Logs:    make([]AuditLogResponse, 0, len(result.Logs)),
		Total:   result.Total,
		Limit:   result.Limit,
		Offset:  result.Offset,
		HasMore: result.HasMore,
	}

	for _, log := range result.Logs {
		response.Logs = append(response.Logs, AuditLogResponse{
			ID:             log.ID,
			OrganizationID: log.OrganizationID,
			UserID:         log.UserID,
			Action:         string(log.Action),
			ResourceType:   string(log.ResourceType),
			ResourceID:     log.ResourceID,
			Changes:        log.Changes,
			IPAddress:      log.IPAddress,
			UserAgent:      log.UserAgent,
			Location:       log.Location,
			Status:         log.Status,
			ErrorMessage:   log.ErrorMessage,
			Timestamp:      log.Timestamp.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func parseAuditFilters(r *http.Request) audit.QueryFilters {
	filters := audit.QueryFilters{
		UserID:       r.URL.Query().Get("user_id"),
		Action:       audit.Action(r.URL.Query().Get("action")),
		ResourceType: audit.ResourceType(r.URL.Query().Get("resource_type")),
		ResourceID:   r.URL.Query().Get("resource_id"),
		Status:       r.URL.Query().Get("status"),
		IPAddress:    r.URL.Query().Get("ip_address"),
	}

	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			filters.StartDate = t
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			filters.EndDate = t
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 500 {
			filters.Limit = l
		}
	}
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filters.Offset = o
		}
	}

	filters.SortOrder = r.URL.Query().Get("sort")
	if filters.SortOrder == "" {
		filters.SortOrder = "desc"
	}

	return filters
}

func (h *Handler) handleExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	if !tc.HasFeature("audit_logs") {
		writeJSONError(w, "audit logs feature not available on current plan", http.StatusForbidden)
		return
	}

	filters := parseAuditFilters(r)
	filters.Limit = 10000
	filters.Offset = 0

	store := h.getAuditStore()
	result, err := store.Query(r.Context(), orgID, filters)
	if err != nil {
		logger.ErrorC("api", "failed to query audit logs for export: "+err.Error())
		writeJSONError(w, "failed to export audit logs", http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	switch format {
	case "csv":
		h.exportAuditCSV(w, result.Logs, orgID)
	case "json":
		h.exportAuditJSON(w, result.Logs, orgID)
	default:
		writeJSONError(w, "unsupported export format", http.StatusBadRequest)
	}
}

func (h *Handler) exportAuditCSV(w http.ResponseWriter, logs []*audit.AuditLog, orgID string) {
	filename := fmt.Sprintf("audit_logs_%s_%s.csv", orgID, time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"ID", "Timestamp", "User ID", "Action", "Resource Type", "Resource ID", "IP Address", "Status", "Error Message"}
	writer.Write(header)

	for _, log := range logs {
		record := []string{
			log.ID,
			log.Timestamp.Format(time.RFC3339),
			log.UserID,
			string(log.Action),
			string(log.ResourceType),
			log.ResourceID,
			log.IPAddress,
			log.Status,
			log.ErrorMessage,
		}
		writer.Write(record)
	}
}

func (h *Handler) exportAuditJSON(w http.ResponseWriter, logs []*audit.AuditLog, orgID string) {
	filename := fmt.Sprintf("audit_logs_%s_%s.json", orgID, time.Now().Format("20060102-150405"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	response := make([]AuditLogResponse, 0, len(logs))
	for _, log := range logs {
		response = append(response, AuditLogResponse{
			ID:             log.ID,
			OrganizationID: log.OrganizationID,
			UserID:         log.UserID,
			Action:         string(log.Action),
			ResourceType:   string(log.ResourceType),
			ResourceID:     log.ResourceID,
			Changes:        log.Changes,
			IPAddress:      log.IPAddress,
			UserAgent:      log.UserAgent,
			Location:       log.Location,
			Status:         log.Status,
			ErrorMessage:   log.ErrorMessage,
			Timestamp:      log.Timestamp.Format(time.RFC3339),
		})
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleGetAuditStats(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	if !tc.HasFeature("audit_logs") {
		writeJSONError(w, "audit logs feature not available on current plan", http.StatusForbidden)
		return
	}

	filters := audit.QueryFilters{}
	if startDate := r.URL.Query().Get("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			filters.StartDate = t
		}
	}
	if endDate := r.URL.Query().Get("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			filters.EndDate = t
		}
	}
	if filters.StartDate.IsZero() {
		filters.StartDate = time.Now().AddDate(0, 0, -30)
	}
	if filters.EndDate.IsZero() {
		filters.EndDate = time.Now()
	}

	store := h.getAuditStore()
	result, err := store.Query(r.Context(), orgID, filters)
	if err != nil {
		logger.ErrorC("api", "failed to get audit stats: "+err.Error())
		writeJSONError(w, "failed to get audit stats", http.StatusInternalServerError)
		return
	}

	stats := AuditStatsResponse{
		OrganizationID: orgID,
		TotalLogs:      result.Total,
		ByAction:       make(map[string]int64),
		ByResource:     make(map[string]int64),
		ByUser:         make(map[string]int64),
		RecentActivity: make([]AuditActivitySummary, 0),
	}

	for _, log := range result.Logs {
		stats.ByAction[string(log.Action)]++
		stats.ByResource[string(log.ResourceType)]++
		stats.ByUser[log.UserID]++
	}

	stats.RecentActivity = generateActivitySummary(result.Logs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func generateActivitySummary(logs []*audit.AuditLog) []AuditActivitySummary {
	byDate := make(map[string]*AuditActivitySummary)

	for _, log := range logs {
		dateKey := log.Timestamp.Format("2006-01-02")
		if _, exists := byDate[dateKey]; !exists {
			byDate[dateKey] = &AuditActivitySummary{Date: dateKey}
		}
		byDate[dateKey].Count++
		if log.Status == audit.StatusSuccess {
			byDate[dateKey].Success++
		} else {
			byDate[dateKey].Failed++
		}
	}

	result := make([]AuditActivitySummary, 0, len(byDate))
	for _, summary := range byDate {
		result = append(result, *summary)
	}

	return result
}

func (h *Handler) handleGetAuditRetention(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgRead)
	if tc == nil {
		return
	}

	store := h.getAuditStore()
	retentionDays, err := store.GetRetentionSettings(r.Context(), orgID)
	if err != nil {
		retentionDays = audit.DefaultRetentionDays
	}

	response := AuditRetentionResponse{
		OrganizationID: orgID,
		RetentionDays:  retentionDays,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleUpdateAuditRetention(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermOrgWrite)
	if tc == nil {
		return
	}

	if tc.SubscriptionTier != tenant.TierEnterprise {
		writeJSONError(w, "custom retention only available for enterprise plans", http.StatusForbidden)
		return
	}

	var req AuditRetentionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSONError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.RetentionDays < 30 || req.RetentionDays > 2555 {
		writeJSONError(w, "retention days must be between 30 and 2555 (7 years)", http.StatusBadRequest)
		return
	}

	store := h.getAuditStore()
	if err := store.SetRetentionSettings(r.Context(), orgID, req.RetentionDays); err != nil {
		logger.ErrorC("api", "failed to update retention settings: "+err.Error())
		writeJSONError(w, "failed to update retention settings", http.StatusInternalServerError)
		return
	}

	logger.Infof("audit retention updated for org %s: %d days", orgID, req.RetentionDays)

	response := AuditRetentionResponse{
		OrganizationID: orgID,
		RetentionDays:  req.RetentionDays,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
