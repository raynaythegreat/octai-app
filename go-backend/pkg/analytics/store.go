package analytics

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrEventNotFound    = errors.New("event not found")
	ErrInvalidTimeRange = errors.New("invalid time range")
	ErrTenantRequired   = errors.New("tenant ID required")
)

type AnalyticsStore interface {
	StoreEvents(ctx context.Context, events []*Event) error
	GetEvents(ctx context.Context, params QueryParams) ([]*Event, error)

	GetUsageMetrics(ctx context.Context, tenantID string, start, end time.Time) (*UsageSummary, error)
	GetPerformanceMetrics(ctx context.Context, tenantID string, start, end time.Time) (*PerformanceSummary, error)
	GetCostMetrics(ctx context.Context, tenantID string, start, end time.Time) (*CostSummary, error)
	GetEngagementMetrics(ctx context.Context, tenantID string, start, end time.Time) (*EngagementSummary, error)

	GetHourlyMetrics(ctx context.Context, tenantID string, start, end time.Time) ([]*HourlyMetric, error)
	GetDailyMetrics(ctx context.Context, tenantID string, start, end time.Time) ([]*DailyMetric, error)

	GetModelPricing(ctx context.Context, model string) (*ModelPricing, error)
	UpdateModelPricing(ctx context.Context, pricing *ModelPricing) error

	DeleteOldEvents(ctx context.Context, before time.Time) (int64, error)
}

type MemoryAnalyticsStore struct {
	mu            sync.RWMutex
	events        []*Event
	hourlyMetrics map[string]*HourlyMetric
	dailyMetrics  map[string]*DailyMetric
	modelPricing  map[string]*ModelPricing
	maxEvents     int
}

func NewMemoryAnalyticsStore(maxEvents int) *MemoryAnalyticsStore {
	if maxEvents <= 0 {
		maxEvents = 100000
	}
	return &MemoryAnalyticsStore{
		events:        make([]*Event, 0, 1000),
		hourlyMetrics: make(map[string]*HourlyMetric),
		dailyMetrics:  make(map[string]*DailyMetric),
		modelPricing:  make(map[string]*ModelPricing),
		maxEvents:     maxEvents,
	}
}

func (s *MemoryAnalyticsStore) StoreEvents(ctx context.Context, events []*Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	available := s.maxEvents - len(s.events)
	if available < len(events) {
		overflow := len(events) - available
		if overflow < len(s.events) {
			s.events = s.events[overflow:]
		} else {
			s.events = s.events[:0]
		}
	}

	s.events = append(s.events, events...)

	for _, event := range events {
		s.updateHourlyMetric(event)
		s.updateDailyMetric(event)
	}

	return nil
}

func (s *MemoryAnalyticsStore) updateHourlyMetric(event *Event) {
	hourKey := event.Timestamp.Truncate(time.Hour).Format(time.RFC3339) + ":" + event.TenantID + ":" + event.AgentID + ":" + event.Channel

	metric, exists := s.hourlyMetrics[hourKey]
	if !exists {
		metric = &HourlyMetric{
			Timestamp: event.Timestamp.Truncate(time.Hour),
			TenantID:  event.TenantID,
			AgentID:   event.AgentID,
			Channel:   event.Channel,
			Model:     event.Model,
		}
		s.hourlyMetrics[hourKey] = metric
	}

	switch event.Type {
	case EventTypeMessage, EventTypeLLMResponse:
		metric.Messages++
	case EventTypeLLMRequest:
		metric.APICalls++
	}

	metric.TokensIn += int64(event.TokensIn)
	metric.TokensOut += int64(event.TokensOut)
	metric.CostEstimate += event.Cost

	if event.DurationMs > 0 {
		if metric.AvgLatencyMs == 0 {
			metric.AvgLatencyMs = float64(event.DurationMs)
		} else {
			metric.AvgLatencyMs = (metric.AvgLatencyMs*float64(metric.APICalls-1) + float64(event.DurationMs)) / float64(metric.APICalls)
		}
	}

	if !event.Success {
		metric.ErrorCount++
	}
}

func (s *MemoryAnalyticsStore) updateDailyMetric(event *Event) {
	dayKey := event.Timestamp.Truncate(24*time.Hour).Format("2006-01-02") + ":" + event.TenantID + ":" + event.AgentID + ":" + event.Channel

	metric, exists := s.dailyMetrics[dayKey]
	if !exists {
		date := event.Timestamp.Truncate(24 * time.Hour)
		metric = &DailyMetric{
			Date:     date,
			TenantID: event.TenantID,
			AgentID:  event.AgentID,
			Channel:  event.Channel,
			Model:    event.Model,
		}
		s.dailyMetrics[dayKey] = metric
	}

	switch event.Type {
	case EventTypeMessage, EventTypeLLMResponse:
		metric.Messages++
	case EventTypeLLMRequest:
		metric.APICalls++
	case EventTypeSessionStart:
		metric.Sessions++
	}

	metric.TokensIn += int64(event.TokensIn)
	metric.TokensOut += int64(event.TokensOut)
	metric.CostEstimate += event.Cost

	if event.DurationMs > 0 {
		if metric.AvgLatencyMs == 0 {
			metric.AvgLatencyMs = float64(event.DurationMs)
		} else {
			metric.AvgLatencyMs = (metric.AvgLatencyMs*float64(metric.APICalls-1) + float64(event.DurationMs)) / float64(metric.APICalls)
		}
	}

	if !event.Success {
		metric.ErrorCount++
	}
}

func (s *MemoryAnalyticsStore) GetEvents(ctx context.Context, params QueryParams) ([]*Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Event

	for _, event := range s.events {
		if !params.Start.IsZero() && event.Timestamp.Before(params.Start) {
			continue
		}
		if !params.End.IsZero() && event.Timestamp.After(params.End) {
			continue
		}
		if params.TenantID != "" && event.TenantID != params.TenantID {
			continue
		}
		if params.Channel != "" && event.Channel != params.Channel {
			continue
		}
		if params.AgentID != "" && event.AgentID != params.AgentID {
			continue
		}
		if params.Model != "" && event.Model != params.Model {
			continue
		}
		if params.Provider != "" && event.Provider != params.Provider {
			continue
		}

		result = append(result, event)
	}

	return result, nil
}

func (s *MemoryAnalyticsStore) GetUsageMetrics(ctx context.Context, tenantID string, start, end time.Time) (*UsageSummary, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return nil, ErrInvalidTimeRange
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &UsageSummary{}
	sessions := make(map[string]struct{})
	sessionMessages := make(map[string]int64)

	for _, event := range s.events {
		if event.TenantID != tenantID {
			continue
		}
		if !start.IsZero() && event.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && event.Timestamp.After(end) {
			continue
		}

		switch event.Type {
		case EventTypeMessage:
			summary.MessagesTotal++
		case EventTypeLLMResponse:
			summary.MessagesAssistant++
			summary.MessagesTotal++
		case EventTypeLLMRequest:
			summary.APICalls++
		}

		summary.TokensInput += int64(event.TokensIn)
		summary.TokensOutput += int64(event.TokensOut)
		summary.TokensTotal += int64(event.TokensIn + event.TokensOut)

		if event.SessionID != "" {
			sessions[event.SessionID] = struct{}{}
			sessionMessages[event.SessionID]++
		}
	}

	summary.UniqueSessions = int64(len(sessions))

	if len(sessionMessages) > 0 {
		var total float64
		for _, count := range sessionMessages {
			total += float64(count)
		}
		summary.AvgSessionLength = total / float64(len(sessionMessages))
	}

	return summary, nil
}

func (s *MemoryAnalyticsStore) GetPerformanceMetrics(ctx context.Context, tenantID string, start, end time.Time) (*PerformanceSummary, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &PerformanceSummary{
		LatencyPercentiles: make(map[string]int64),
	}

	var latencies []int64

	for _, event := range s.events {
		if event.TenantID != tenantID {
			continue
		}
		if !start.IsZero() && event.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && event.Timestamp.After(end) {
			continue
		}

		summary.TotalRequests++

		if event.Success {
			summary.SuccessfulRequests++
		} else {
			summary.FailedRequests++
		}

		if event.DurationMs > 0 {
			latencies = append(latencies, int64(event.DurationMs))
		}
	}

	if summary.TotalRequests > 0 {
		summary.SuccessRate = float64(summary.SuccessfulRequests) / float64(summary.TotalRequests)
	}

	if len(latencies) > 0 {
		summary.AvgLatencyMs = calculateAverage(latencies)
		summary.LatencyPercentiles["p50"] = calculatePercentile(latencies, 50)
		summary.LatencyPercentiles["p95"] = calculatePercentile(latencies, 95)
		summary.LatencyPercentiles["p99"] = calculatePercentile(latencies, 99)
	}

	return summary, nil
}

func (s *MemoryAnalyticsStore) GetCostMetrics(ctx context.Context, tenantID string, start, end time.Time) (*CostSummary, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &CostSummary{}
	sessions := make(map[string]struct{})

	for _, event := range s.events {
		if event.TenantID != tenantID {
			continue
		}
		if !start.IsZero() && event.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && event.Timestamp.After(end) {
			continue
		}

		summary.TotalCost += event.Cost
		summary.TotalTokensIn += int64(event.TokensIn)
		summary.TotalTokensOut += int64(event.TokensOut)

		if meta, ok := event.Metadata["savings"].(float64); ok {
			summary.TotalSavings += meta
		}

		if event.SessionID != "" {
			sessions[event.SessionID] = struct{}{}
		}
	}

	if len(sessions) > 0 {
		summary.AvgCostPerSession = summary.TotalCost / float64(len(sessions))
	}

	if summary.TotalCost > 0 {
		summary.SavingsPct = (summary.TotalSavings / (summary.TotalCost + summary.TotalSavings)) * 100
	}

	return summary, nil
}

func (s *MemoryAnalyticsStore) GetEngagementMetrics(ctx context.Context, tenantID string, start, end time.Time) (*EngagementSummary, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if tenantID == "" {
		return nil, ErrTenantRequired
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := &EngagementSummary{}

	usersByDay := make(map[string]map[string]struct{})
	usersByMonth := make(map[string]map[string]struct{})
	sessions := make(map[string]int64)
	sessionMessages := make(map[string]int64)
	concurrentByHour := make(map[string]int64)

	for _, event := range s.events {
		if event.TenantID != tenantID {
			continue
		}
		if !start.IsZero() && event.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && event.Timestamp.After(end) {
			continue
		}

		dayKey := event.Timestamp.Format("2006-01-02")
		monthKey := event.Timestamp.Format("2006-01")

		if event.SessionID != "" {
			if usersByDay[dayKey] == nil {
				usersByDay[dayKey] = make(map[string]struct{})
			}
			usersByDay[dayKey][event.SessionID] = struct{}{}

			if usersByMonth[monthKey] == nil {
				usersByMonth[monthKey] = make(map[string]struct{})
			}
			usersByMonth[monthKey][event.SessionID] = struct{}{}

			sessions[event.SessionID]++
			if event.Type == EventTypeMessage || event.Type == EventTypeLLMResponse {
				sessionMessages[event.SessionID]++
			}

			hourKey := event.Timestamp.Format("2006-01-02T15")
			concurrentByHour[hourKey]++
		}
	}

	for _, users := range usersByDay {
		if int64(len(users)) > summary.DAU {
			summary.DAU = int64(len(users))
		}
	}

	for _, users := range usersByMonth {
		if int64(len(users)) > summary.MAU {
			summary.MAU = int64(len(users))
		}
	}

	if summary.MAU > 0 {
		summary.DAUMAURatio = float64(summary.DAU) / float64(summary.MAU)
	}

	summary.TotalSessions = int64(len(sessions))

	if len(sessionMessages) > 0 {
		var total float64
		for _, count := range sessionMessages {
			total += float64(count)
		}
		summary.AvgSessionLength = total / float64(len(sessionMessages))
	}

	if summary.DAU > 0 {
		var totalMessages int64
		for _, count := range sessionMessages {
			totalMessages += count
		}
		summary.AvgMessagesPerUser = float64(totalMessages) / float64(summary.DAU)
	}

	var peakConcurrent int64
	for _, count := range concurrentByHour {
		if count > peakConcurrent {
			peakConcurrent = count
		}
	}
	summary.PeakConcurrent = peakConcurrent

	return summary, nil
}

func (s *MemoryAnalyticsStore) GetHourlyMetrics(ctx context.Context, tenantID string, start, end time.Time) ([]*HourlyMetric, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*HourlyMetric

	for _, metric := range s.hourlyMetrics {
		if metric.TenantID != tenantID {
			continue
		}
		if !start.IsZero() && metric.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && metric.Timestamp.After(end) {
			continue
		}

		result = append(result, metric)
	}

	return result, nil
}

func (s *MemoryAnalyticsStore) GetDailyMetrics(ctx context.Context, tenantID string, start, end time.Time) ([]*DailyMetric, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*DailyMetric

	for _, metric := range s.dailyMetrics {
		if metric.TenantID != tenantID {
			continue
		}
		if !start.IsZero() && metric.Date.Before(start) {
			continue
		}
		if !end.IsZero() && metric.Date.After(end) {
			continue
		}

		result = append(result, metric)
	}

	return result, nil
}

func (s *MemoryAnalyticsStore) GetModelPricing(ctx context.Context, model string) (*ModelPricing, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	pricing, exists := s.modelPricing[model]
	if !exists {
		return nil, ErrEventNotFound
	}

	return pricing, nil
}

func (s *MemoryAnalyticsStore) UpdateModelPricing(ctx context.Context, pricing *ModelPricing) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pricing.UpdatedAt = time.Now()
	s.modelPricing[pricing.Model] = pricing

	return nil
}

func (s *MemoryAnalyticsStore) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var newEvents []*Event
	var deleted int64

	for _, event := range s.events {
		if event.Timestamp.Before(before) {
			deleted++
		} else {
			newEvents = append(newEvents, event)
		}
	}

	s.events = newEvents

	return deleted, nil
}

func (s *MemoryAnalyticsStore) GetEventCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}

type NoopAnalyticsStore struct{}

func NewNoopAnalyticsStore() *NoopAnalyticsStore {
	return &NoopAnalyticsStore{}
}

func (s *NoopAnalyticsStore) StoreEvents(ctx context.Context, events []*Event) error {
	return nil
}

func (s *NoopAnalyticsStore) GetEvents(ctx context.Context, params QueryParams) ([]*Event, error) {
	return nil, nil
}

func (s *NoopAnalyticsStore) GetUsageMetrics(ctx context.Context, tenantID string, start, end time.Time) (*UsageSummary, error) {
	return &UsageSummary{}, nil
}

func (s *NoopAnalyticsStore) GetPerformanceMetrics(ctx context.Context, tenantID string, start, end time.Time) (*PerformanceSummary, error) {
	return &PerformanceSummary{LatencyPercentiles: make(map[string]int64)}, nil
}

func (s *NoopAnalyticsStore) GetCostMetrics(ctx context.Context, tenantID string, start, end time.Time) (*CostSummary, error) {
	return &CostSummary{}, nil
}

func (s *NoopAnalyticsStore) GetEngagementMetrics(ctx context.Context, tenantID string, start, end time.Time) (*EngagementSummary, error) {
	return &EngagementSummary{}, nil
}

func (s *NoopAnalyticsStore) GetHourlyMetrics(ctx context.Context, tenantID string, start, end time.Time) ([]*HourlyMetric, error) {
	return nil, nil
}

func (s *NoopAnalyticsStore) GetDailyMetrics(ctx context.Context, tenantID string, start, end time.Time) ([]*DailyMetric, error) {
	return nil, nil
}

func (s *NoopAnalyticsStore) GetModelPricing(ctx context.Context, model string) (*ModelPricing, error) {
	return nil, ErrEventNotFound
}

func (s *NoopAnalyticsStore) UpdateModelPricing(ctx context.Context, pricing *ModelPricing) error {
	return nil
}

func (s *NoopAnalyticsStore) DeleteOldEvents(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}
