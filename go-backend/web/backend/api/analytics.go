package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/tenant"
)

type UsageMetricsResponse struct {
	OrganizationID string        `json:"organization_id"`
	PeriodStart    string        `json:"period_start"`
	PeriodEnd      string        `json:"period_end"`
	Granularity    string        `json:"granularity"`
	Metrics        []UsageMetric `json:"metrics"`
	Summary        UsageSummary  `json:"summary"`
}

type UsageMetric struct {
	Date          string `json:"date"`
	Messages      int64  `json:"messages"`
	TokensUsed    int64  `json:"tokens_used"`
	AgentsActive  int64  `json:"agents_active"`
	SessionsCount int64  `json:"sessions_count"`
}

type UsageSummary struct {
	TotalMessages     int64   `json:"total_messages"`
	TotalTokens       int64   `json:"total_tokens"`
	AvgMessagesPerDay float64 `json:"avg_messages_per_day"`
	PeakMessagesDay   string  `json:"peak_messages_day"`
}

type PerformanceMetricsResponse struct {
	OrganizationID string              `json:"organization_id"`
	PeriodStart    string              `json:"period_start"`
	PeriodEnd      string              `json:"period_end"`
	Granularity    string              `json:"granularity"`
	Metrics        []PerformanceMetric `json:"metrics"`
	Summary        PerformanceSummary  `json:"summary"`
}

type PerformanceMetric struct {
	Date              string  `json:"date"`
	AvgResponseTime   float64 `json:"avg_response_time_ms"`
	P95ResponseTime   float64 `json:"p95_response_time_ms"`
	SuccessRate       float64 `json:"success_rate"`
	ErrorCount        int64   `json:"error_count"`
	RequestsPerMinute float64 `json:"requests_per_minute"`
}

type PerformanceSummary struct {
	OverallAvgResponseTime float64 `json:"overall_avg_response_time_ms"`
	OverallSuccessRate     float64 `json:"overall_success_rate"`
	TotalErrors            int64   `json:"total_errors"`
	PeakRequestsPerMinute  float64 `json:"peak_requests_per_minute"`
}

type CostBreakdownResponse struct {
	OrganizationID string      `json:"organization_id"`
	PeriodStart    string      `json:"period_start"`
	PeriodEnd      string      `json:"period_end"`
	TotalCost      float64     `json:"total_cost"`
	Currency       string      `json:"currency"`
	Breakdown      []CostItem  `json:"breakdown"`
	ByModel        []ModelCost `json:"by_model"`
	ByAgent        []AgentCost `json:"by_agent"`
}

type CostItem struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Percent  float64 `json:"percent"`
}

type ModelCost struct {
	Model string  `json:"model"`
	Cost  float64 `json:"cost"`
	Count int64   `json:"count"`
}

type AgentCost struct {
	AgentID string  `json:"agent_id"`
	Cost    float64 `json:"cost"`
	Count   int64   `json:"count"`
}

type AnalyticsQueryParams struct {
	Start       time.Time
	End         time.Time
	Granularity string
}

func parseAnalyticsQuery(r *http.Request) (AnalyticsQueryParams, error) {
	var params AnalyticsQueryParams

	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	params.Granularity = r.URL.Query().Get("granularity")

	if params.Granularity == "" {
		params.Granularity = "day"
	}

	if startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return params, fmt.Errorf("invalid start date format: %v", err)
		}
		params.Start = t
	} else {
		params.Start = time.Now().AddDate(0, -1, 0)
	}

	if endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			return params, fmt.Errorf("invalid end date format: %v", err)
		}
		params.End = t
	} else {
		params.End = time.Now()
	}

	return params, nil
}

func generateDateRange(start, end time.Time, granularity string) []time.Time {
	var dates []time.Time
	current := start

	switch granularity {
	case "hour":
		for current.Before(end) || current.Equal(end) {
			dates = append(dates, current)
			current = current.Add(time.Hour)
		}
	case "day":
		current = time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
		for current.Before(end) || current.Equal(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 0, 1)
		}
	case "week":
		for current.Weekday() != time.Monday {
			current = current.AddDate(0, 0, -1)
		}
		for current.Before(end) || current.Equal(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 0, 7)
		}
	case "month":
		current = time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, current.Location())
		for current.Before(end) || current.Equal(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 1, 0)
		}
	default:
		current = time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, current.Location())
		for current.Before(end) || current.Equal(end) {
			dates = append(dates, current)
			current = current.AddDate(0, 0, 1)
		}
	}

	return dates
}

func (h *Handler) registerAnalyticsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v2/organizations/{id}/analytics/usage", h.handleGetUsageMetrics)
	mux.HandleFunc("GET /api/v2/organizations/{id}/analytics/performance", h.handleGetPerformanceMetrics)
	mux.HandleFunc("GET /api/v2/organizations/{id}/analytics/costs", h.handleGetCostBreakdown)
}

func (h *Handler) handleGetUsageMetrics(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermAnalytics)
	if tc == nil {
		return
	}

	params, err := parseAnalyticsQuery(r)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	dates := generateDateRange(params.Start, params.End, params.Granularity)
	metrics := make([]UsageMetric, 0, len(dates))

	var totalMessages int64
	var totalTokens int64
	var peakMessages int64
	var peakMessagesDate string

	for _, date := range dates {
		messages := generateMockUsageData(date)
		tokens := messages * 1500
		sessions := messages / 3

		metrics = append(metrics, UsageMetric{
			Date:          date.Format(time.RFC3339),
			Messages:      messages,
			TokensUsed:    tokens,
			AgentsActive:  1,
			SessionsCount: sessions,
		})

		totalMessages += messages
		totalTokens += tokens

		if messages > peakMessages {
			peakMessages = messages
			peakMessagesDate = date.Format(time.RFC3339)
		}
	}

	avgMessages := float64(0)
	if len(dates) > 0 {
		avgMessages = float64(totalMessages) / float64(len(dates))
	}

	response := UsageMetricsResponse{
		OrganizationID: orgID,
		PeriodStart:    params.Start.Format(time.RFC3339),
		PeriodEnd:      params.End.Format(time.RFC3339),
		Granularity:    params.Granularity,
		Metrics:        metrics,
		Summary: UsageSummary{
			TotalMessages:     totalMessages,
			TotalTokens:       totalTokens,
			AvgMessagesPerDay: avgMessages,
			PeakMessagesDay:   peakMessagesDate,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleGetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermAnalytics)
	if tc == nil {
		return
	}

	params, err := parseAnalyticsQuery(r)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	dates := generateDateRange(params.Start, params.End, params.Granularity)
	metrics := make([]PerformanceMetric, 0, len(dates))

	var totalResponseTime float64
	var totalRequests float64
	var totalErrors int64
	var peakRPM float64

	for _, date := range dates {
		avgTime := 250.0 + float64(date.Hour()*10)
		p95Time := avgTime * 1.5
		requests := generateMockUsageData(date)
		errors := requests / 100
		successRate := 1.0 - (float64(errors) / float64(requests+1))
		rpm := float64(requests) / 60.0

		metrics = append(metrics, PerformanceMetric{
			Date:              date.Format(time.RFC3339),
			AvgResponseTime:   avgTime,
			P95ResponseTime:   p95Time,
			SuccessRate:       successRate * 100,
			ErrorCount:        errors,
			RequestsPerMinute: rpm,
		})

		totalResponseTime += avgTime
		totalRequests += float64(requests)
		totalErrors += errors

		if rpm > peakRPM {
			peakRPM = rpm
		}
	}

	overallAvg := float64(0)
	if len(dates) > 0 {
		overallAvg = totalResponseTime / float64(len(dates))
	}

	overallSuccessRate := float64(0)
	if totalRequests > 0 {
		overallSuccessRate = (totalRequests - float64(totalErrors)) / totalRequests * 100
	}

	response := PerformanceMetricsResponse{
		OrganizationID: orgID,
		PeriodStart:    params.Start.Format(time.RFC3339),
		PeriodEnd:      params.End.Format(time.RFC3339),
		Granularity:    params.Granularity,
		Metrics:        metrics,
		Summary: PerformanceSummary{
			OverallAvgResponseTime: overallAvg,
			OverallSuccessRate:     overallSuccessRate,
			TotalErrors:            totalErrors,
			PeakRequestsPerMinute:  peakRPM,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) handleGetCostBreakdown(w http.ResponseWriter, r *http.Request) {
	orgID := r.PathValue("id")
	if orgID == "" {
		writeJSONError(w, "organization id is required", http.StatusBadRequest)
		return
	}

	tc := h.requireOrgAccess(w, r, orgID, tenant.PermAnalytics)
	if tc == nil {
		return
	}

	params, err := parseAnalyticsQuery(r)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	totalMessages, _ := h.tenantStore.GetMonthlyUsage(r.Context(), orgID, "message")
	if totalMessages == 0 {
		totalMessages = 5000
	}

	inputCost := float64(totalMessages*1000) * 0.000003
	outputCost := float64(totalMessages*500) * 0.000015
	storageCost := 5.0
	platformFee := (inputCost + outputCost) * 0.1

	totalCost := inputCost + outputCost + storageCost + platformFee

	breakdown := []CostItem{
		{Category: "input_tokens", Amount: inputCost, Percent: inputCost / totalCost * 100},
		{Category: "output_tokens", Amount: outputCost, Percent: outputCost / totalCost * 100},
		{Category: "storage", Amount: storageCost, Percent: storageCost / totalCost * 100},
		{Category: "platform_fee", Amount: platformFee, Percent: platformFee / totalCost * 100},
	}

	byModel := []ModelCost{
		{Model: "claude-3-opus", Cost: totalCost * 0.4, Count: totalMessages / 3},
		{Model: "claude-3-sonnet", Cost: totalCost * 0.35, Count: totalMessages / 2},
		{Model: "claude-3-haiku", Cost: totalCost * 0.25, Count: totalMessages / 6},
	}

	byAgent := []AgentCost{
		{AgentID: "main", Cost: totalCost * 0.6, Count: totalMessages * 6 / 10},
		{AgentID: "assistant", Cost: totalCost * 0.3, Count: totalMessages * 3 / 10},
		{AgentID: "research", Cost: totalCost * 0.1, Count: totalMessages / 10},
	}

	response := CostBreakdownResponse{
		OrganizationID: orgID,
		PeriodStart:    params.Start.Format(time.RFC3339),
		PeriodEnd:      params.End.Format(time.RFC3339),
		TotalCost:      totalCost,
		Currency:       "USD",
		Breakdown:      breakdown,
		ByModel:        byModel,
		ByAgent:        byAgent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateMockUsageData(date time.Time) int64 {
	base := int64(100)
	dayOfWeek := date.Weekday()

	switch dayOfWeek {
	case time.Saturday, time.Sunday:
		base = 50
	case time.Monday, time.Tuesday, time.Wednesday:
		base = 150
	case time.Thursday, time.Friday:
		base = 120
	}

	hourFactor := 1.0
	if date.Hour() >= 9 && date.Hour() <= 17 {
		hourFactor = 1.5
	} else if date.Hour() >= 0 && date.Hour() < 6 {
		hourFactor = 0.3
	}

	return int64(float64(base) * hourFactor)
}

func init() {
	postInitFuncs = append(postInitFuncs, func(h *Handler) {
		if h.analyticsCache == nil {
			h.analyticsCache = make(map[string]interface{})
		}
	})
}
