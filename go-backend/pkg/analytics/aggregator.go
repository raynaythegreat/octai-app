package analytics

import (
	"math"
	"sort"
	"sync"
	"time"
)

type Aggregator struct {
	events    []*Event
	eventsMu  sync.RWMutex
	maxEvents int
}

func NewAggregator(maxEvents int) *Aggregator {
	if maxEvents <= 0 {
		maxEvents = 100000
	}
	return &Aggregator{
		events:    make([]*Event, 0, 1000),
		maxEvents: maxEvents,
	}
}

func (a *Aggregator) RecordEvent(event *Event) {
	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()

	if len(a.events) >= a.maxEvents {
		a.events = a.events[1:]
	}
	a.events = append(a.events, event)
}

func (a *Aggregator) RecordEvents(events []*Event) {
	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()

	available := a.maxEvents - len(a.events)
	if available <= 0 {
		overflow := len(events) - available
		if overflow > 0 {
			a.events = a.events[overflow:]
		}
	}
	a.events = append(a.events, events...)
}

func (a *Aggregator) GetEvents() []*Event {
	a.eventsMu.RLock()
	defer a.eventsMu.RUnlock()

	result := make([]*Event, len(a.events))
	copy(result, a.events)
	return result
}

func (a *Aggregator) Clear() {
	a.eventsMu.Lock()
	defer a.eventsMu.Unlock()
	a.events = a.events[:0]
}

func (a *Aggregator) AggregateByHour(events []*Event) map[string]*HourlyMetric {
	result := make(map[string]*HourlyMetric)

	for _, event := range events {
		hourKey := event.Timestamp.Truncate(time.Hour).Format(time.RFC3339)
		metricKey := hourKey + ":" + event.TenantID + ":" + event.AgentID + ":" + event.Channel

		metric, exists := result[metricKey]
		if !exists {
			metric = &HourlyMetric{
				Timestamp: event.Timestamp.Truncate(time.Hour),
				TenantID:  event.TenantID,
				AgentID:   event.AgentID,
				Channel:   event.Channel,
				Model:     event.Model,
			}
			result[metricKey] = metric
		}

		a.updateHourlyMetric(metric, event)
	}

	return result
}

func (a *Aggregator) updateHourlyMetric(metric *HourlyMetric, event *Event) {
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
			metric.AvgLatencyMs = (metric.AvgLatencyMs + float64(event.DurationMs)) / 2
		}
	}

	if !event.Success {
		metric.ErrorCount++
	}
}

func (a *Aggregator) AggregateByDay(events []*Event) map[string]*DailyMetric {
	result := make(map[string]*DailyMetric)
	sessionsByDay := make(map[string]map[string]struct{})

	for _, event := range events {
		dayKey := event.Timestamp.Truncate(24 * time.Hour).Format("2006-01-02")
		metricKey := dayKey + ":" + event.TenantID + ":" + event.AgentID + ":" + event.Channel

		metric, exists := result[metricKey]
		if !exists {
			date, _ := time.Parse("2006-01-02", dayKey)
			metric = &DailyMetric{
				Date:     date,
				TenantID: event.TenantID,
				AgentID:  event.AgentID,
				Channel:  event.Channel,
				Model:    event.Model,
			}
			result[metricKey] = metric
			sessionsByDay[metricKey] = make(map[string]struct{})
		}

		a.updateDailyMetric(metric, event, sessionsByDay[metricKey])
	}

	for _, metric := range result {
		metric.DAU = int64(len(sessionsByDay[metric.Date.Format("2006-01-02")+":"+metric.TenantID]))
	}

	return result
}

func (a *Aggregator) updateDailyMetric(metric *DailyMetric, event *Event, sessions map[string]struct{}) {
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
			metric.AvgLatencyMs = (metric.AvgLatencyMs + float64(event.DurationMs)) / 2
		}
	}

	if !event.Success {
		metric.ErrorCount++
	}

	if event.SessionID != "" {
		sessions[event.SessionID] = struct{}{}
	}
}

func (a *Aggregator) AggregateByTenant(events []*Event) map[string]*AggregatedMetrics {
	result := make(map[string]*AggregatedMetrics)

	byTenant := a.groupEventsBy(events, func(e *Event) string {
		return e.TenantID
	})

	for tenantID, tenantEvents := range byTenant {
		metrics := &AggregatedMetrics{}
		metrics.Usage = a.CalculateUsageMetrics(tenantEvents)
		metrics.Performance = a.CalculatePerformanceMetrics(tenantEvents)
		metrics.Cost = a.CalculateCostMetrics(tenantEvents)
		result[tenantID] = metrics
	}

	return result
}

func (a *Aggregator) groupEventsBy(events []*Event, keyFn func(*Event) string) map[string][]*Event {
	result := make(map[string][]*Event)
	for _, event := range events {
		key := keyFn(event)
		result[key] = append(result[key], event)
	}
	return result
}

func (a *Aggregator) CalculateUsageMetrics(events []*Event) *UsageSummary {
	summary := &UsageSummary{}
	sessions := make(map[string]struct{})
	sessionMessages := make(map[string]int64)

	for _, event := range events {
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

	return summary
}

func (a *Aggregator) CalculatePerformanceMetrics(events []*Event) *PerformanceSummary {
	summary := &PerformanceSummary{
		LatencyPercentiles: make(map[string]int64),
	}

	var latencies []int64
	var ttfts []int64
	var tpsValues []float64

	for _, event := range events {
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

	if len(ttfts) > 0 {
		summary.AvgTTFTMs = calculateAverage(ttfts)
	}

	if len(tpsValues) > 0 {
		var sum float64
		for _, v := range tpsValues {
			sum += v
		}
		summary.AvgTPS = sum / float64(len(tpsValues))
	}

	return summary
}

func (a *Aggregator) CalculateCostMetrics(events []*Event) *CostSummary {
	summary := &CostSummary{}
	sessions := make(map[string]struct{})

	for _, event := range events {
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

	if summary.TotalCost > 0 && len(events) > 0 {
		var msgCount int64
		for _, e := range events {
			if e.Type == EventTypeMessage || e.Type == EventTypeLLMResponse {
				msgCount++
			}
		}
		if msgCount > 0 {
			summary.AvgCostPerMsg = summary.TotalCost / float64(msgCount)
		}
	}

	if len(sessions) > 0 {
		summary.AvgCostPerSession = summary.TotalCost / float64(len(sessions))
	}

	if summary.TotalCost > 0 {
		summary.SavingsPct = (summary.TotalSavings / (summary.TotalCost + summary.TotalSavings)) * 100
	}

	return summary
}

func (a *Aggregator) CalculateEngagementMetrics(events []*Event) *EngagementSummary {
	summary := &EngagementSummary{}

	usersByDay := make(map[string]map[string]struct{})
	usersByMonth := make(map[string]map[string]struct{})
	sessions := make(map[string]int64)
	sessionMessages := make(map[string]int64)
	var peakConcurrent int64
	concurrentByHour := make(map[string]int64)

	for _, event := range events {
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

	for _, count := range concurrentByHour {
		if count > peakConcurrent {
			peakConcurrent = count
		}
	}
	summary.PeakConcurrent = peakConcurrent

	return summary
}

func (a *Aggregator) CalculateMetrics(events []*Event) *AggregatedMetrics {
	return &AggregatedMetrics{
		Usage:       a.CalculateUsageMetrics(events),
		Performance: a.CalculatePerformanceMetrics(events),
		Cost:        a.CalculateCostMetrics(events),
		Engagement:  a.CalculateEngagementMetrics(events),
	}
}

func (a *Aggregator) GetBreakdownByChannel(events []*Event) []MetricBreakdown {
	return a.getBreakdown(events, func(e *Event) string { return e.Channel })
}

func (a *Aggregator) GetBreakdownByModel(events []*Event) []MetricBreakdown {
	return a.getBreakdown(events, func(e *Event) string { return e.Model })
}

func (a *Aggregator) GetBreakdownByProvider(events []*Event) []MetricBreakdown {
	return a.getBreakdown(events, func(e *Event) string { return e.Provider })
}

func (a *Aggregator) GetBreakdownByAgent(events []*Event) []MetricBreakdown {
	return a.getBreakdown(events, func(e *Event) string { return e.AgentID })
}

func (a *Aggregator) getBreakdown(events []*Event, keyFn func(*Event) string) []MetricBreakdown {
	counts := make(map[string]int64)
	costs := make(map[string]float64)
	var total int64

	for _, event := range events {
		key := keyFn(event)
		if key == "" {
			key = "unknown"
		}
		counts[key]++
		costs[key] += event.Cost
		total++
	}

	var result []MetricBreakdown
	for key, count := range counts {
		var pct float64
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		result = append(result, MetricBreakdown{
			Key:     key,
			Value:   count,
			Percent: pct,
			Cost:    costs[key],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Value > result[j].Value
	})

	return result
}

func (a *Aggregator) GetErrorBreakdown(events []*Event) []ErrorBreakdown {
	errorCounts := make(map[string]int64)
	var total int64

	for _, event := range events {
		if !event.Success && event.ErrorCode != "" {
			errorCounts[event.ErrorCode]++
			total++
		}
	}

	var result []ErrorBreakdown
	for code, count := range errorCounts {
		var pct float64
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		result = append(result, ErrorBreakdown{
			Code:  code,
			Count: count,
			Pct:   pct,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

func (a *Aggregator) GetTimeline(events []*Event, resolution time.Duration) []TimelinePoint {
	if len(events) == 0 {
		return nil
	}

	points := make(map[time.Time]int64)

	for _, event := range events {
		key := event.Timestamp.Truncate(resolution)
		points[key]++
	}

	var result []TimelinePoint
	for ts, count := range points {
		result = append(result, TimelinePoint{
			Timestamp: ts,
			Value:     count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Timestamp.Before(result[j].Timestamp)
	})

	return result
}

func calculateAverage(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, v := range values {
		sum += v
	}
	return sum / int64(len(values))
}

func calculatePercentile(values []int64, percentile int) int64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]int64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(math.Ceil(float64(percentile)/100*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

func (a *Aggregator) FilterEvents(events []*Event, params QueryParams) []*Event {
	var result []*Event

	for _, event := range events {
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

	return result
}
