package analytics

import (
	"time"
)

type EventType string

const (
	EventTypeLLMRequest    EventType = "llm_request"
	EventTypeLLMResponse   EventType = "llm_response"
	EventTypeToolExecStart EventType = "tool_exec_start"
	EventTypeToolExecEnd   EventType = "tool_exec_end"
	EventTypeTurnStart     EventType = "turn_start"
	EventTypeTurnEnd       EventType = "turn_end"
	EventTypeMessage       EventType = "message"
	EventTypeSessionStart  EventType = "session_start"
	EventTypeSessionEnd    EventType = "session_end"
	EventTypeError         EventType = "error"
)

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

type Event struct {
	ID         string         `json:"id"`
	Type       EventType      `json:"type"`
	TenantID   string         `json:"tenant_id"`
	Timestamp  time.Time      `json:"timestamp"`
	AgentID    string         `json:"agent_id,omitempty"`
	SessionID  string         `json:"session_id,omitempty"`
	Channel    string         `json:"channel,omitempty"`
	ChatID     string         `json:"chat_id,omitempty"`
	Model      string         `json:"model,omitempty"`
	Provider   string         `json:"provider,omitempty"`
	TokensIn   int            `json:"tokens_in,omitempty"`
	TokensOut  int            `json:"tokens_out,omitempty"`
	DurationMs int            `json:"duration_ms,omitempty"`
	Cost       float64        `json:"cost,omitempty"`
	Success    bool           `json:"success"`
	ErrorCode  string         `json:"error_code,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

func (e *Event) TotalTokens() int {
	return e.TokensIn + e.TokensOut
}

type UsageEvent struct {
	ID             string    `json:"id"`
	TenantID       string    `json:"tenant_id"`
	Timestamp      time.Time `json:"timestamp"`
	Messages       int64     `json:"messages"`
	MessagesUser   int64     `json:"messages_user"`
	MessagesAI     int64     `json:"messages_assistant"`
	TokensInput    int64     `json:"tokens_input"`
	TokensOutput   int64     `json:"tokens_output"`
	TokensTotal    int64     `json:"tokens_total"`
	APICalls       int64     `json:"api_calls"`
	APICallsBatch  int64     `json:"api_calls_batch"`
	APICallsStream int64     `json:"api_calls_streaming"`
	AgentID        string    `json:"agent_id,omitempty"`
	Channel        string    `json:"channel,omitempty"`
	Model          string    `json:"model,omitempty"`
	SessionID      string    `json:"session_id,omitempty"`
}

type PerformanceEvent struct {
	ID               string    `json:"id"`
	TenantID         string    `json:"tenant_id"`
	Timestamp        time.Time `json:"timestamp"`
	ResponseTime     int64     `json:"response_time_ms"`
	ResponseTimeP50  int64     `json:"response_time_p50_ms,omitempty"`
	ResponseTimeP95  int64     `json:"response_time_p95_ms,omitempty"`
	ResponseTimeP99  int64     `json:"response_time_p99_ms,omitempty"`
	TimeToFirstToken int64     `json:"time_to_first_token_ms,omitempty"`
	TokensPerSecond  float64   `json:"tokens_per_second,omitempty"`
	Success          bool      `json:"success"`
	Error            string    `json:"error,omitempty"`
	ErrorCode        string    `json:"error_code,omitempty"`
	ToolName         string    `json:"tool_name,omitempty"`
	ToolDurationMs   int64     `json:"tool_duration_ms,omitempty"`
	Provider         string    `json:"provider,omitempty"`
	Model            string    `json:"model,omitempty"`
	AgentID          string    `json:"agent_id,omitempty"`
	Channel          string    `json:"channel,omitempty"`
}

type CostEvent struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Timestamp  time.Time `json:"timestamp"`
	Provider   string    `json:"provider"`
	Model      string    `json:"model"`
	TokensIn   int64     `json:"tokens_in"`
	TokensOut  int64     `json:"tokens_out"`
	Cost       float64   `json:"cost"`
	Currency   string    `json:"currency"`
	InputRate  float64   `json:"input_rate_per_million"`
	OutputRate float64   `json:"output_rate_per_million"`
	Channel    string    `json:"channel,omitempty"`
	AgentID    string    `json:"agent_id,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	CacheHit   bool      `json:"cache_hit"`
	Fallback   bool      `json:"fallback"`
	Savings    float64   `json:"savings,omitempty"`
}

type UsageSummary struct {
	MessagesTotal     int64   `json:"messages_total"`
	MessagesUser      int64   `json:"messages_user"`
	MessagesAssistant int64   `json:"messages_assistant"`
	TokensInput       int64   `json:"tokens_input"`
	TokensOutput      int64   `json:"tokens_output"`
	TokensTotal       int64   `json:"tokens_total"`
	APICalls          int64   `json:"api_calls"`
	UniqueSessions    int64   `json:"unique_sessions"`
	AvgSessionLength  float64 `json:"avg_session_length"`
}

type PerformanceSummary struct {
	TotalRequests      int64            `json:"total_requests"`
	SuccessfulRequests int64            `json:"successful_requests"`
	FailedRequests     int64            `json:"failed_requests"`
	SuccessRate        float64          `json:"success_rate"`
	AvgLatencyMs       int64            `json:"avg_latency_ms"`
	LatencyPercentiles map[string]int64 `json:"latency_percentiles"`
	AvgTTFTMs          int64            `json:"avg_ttft_ms"`
	AvgTPS             float64          `json:"avg_tokens_per_second"`
	RetryCount         int64            `json:"retry_count"`
	CooldownCount      int64            `json:"cooldown_count"`
}

type CostSummary struct {
	TotalCost         float64 `json:"total_cost"`
	TotalTokensIn     int64   `json:"total_tokens_in"`
	TotalTokensOut    int64   `json:"total_tokens_out"`
	AvgCostPerMsg     float64 `json:"avg_cost_per_message"`
	AvgCostPerSession float64 `json:"avg_cost_per_session"`
	ProjectedMonthly  float64 `json:"projected_monthly"`
	TotalSavings      float64 `json:"total_savings"`
	SavingsPct        float64 `json:"savings_pct"`
}

type EngagementSummary struct {
	DAU                int64   `json:"dau"`
	MAU                int64   `json:"mau"`
	DAUMAURatio        float64 `json:"dau_mau_ratio"`
	TotalSessions      int64   `json:"total_sessions"`
	AvgSessionLength   float64 `json:"avg_session_length"`
	AvgMessagesPerUser float64 `json:"avg_messages_per_user"`
	PeakConcurrent     int64   `json:"peak_concurrent"`
}

type QueryParams struct {
	Start       time.Time
	End         time.Time
	Resolution  string
	Channel     string
	AgentID     string
	Model       string
	Provider    string
	Percentiles []int
	TenantID    string
}

type TimelinePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
}

type MetricBreakdown struct {
	Key     string  `json:"key"`
	Value   int64   `json:"value"`
	Percent float64 `json:"pct"`
	Cost    float64 `json:"cost,omitempty"`
}

type AggregatedMetrics struct {
	Period struct {
		Start time.Time `json:"start"`
		End   time.Time `json:"end"`
	} `json:"period"`
	Usage       *UsageSummary       `json:"usage,omitempty"`
	Performance *PerformanceSummary `json:"performance,omitempty"`
	Cost        *CostSummary        `json:"cost,omitempty"`
	Engagement  *EngagementSummary  `json:"engagement,omitempty"`
	Timeline    []TimelinePoint     `json:"timeline,omitempty"`
	ByChannel   []MetricBreakdown   `json:"by_channel,omitempty"`
	ByModel     []MetricBreakdown   `json:"by_model,omitempty"`
	ByProvider  []MetricBreakdown   `json:"by_provider,omitempty"`
	ByAgent     []MetricBreakdown   `json:"by_agent,omitempty"`
	Errors      []ErrorBreakdown    `json:"errors,omitempty"`
}

type ErrorBreakdown struct {
	Code  string  `json:"code"`
	Count int64   `json:"count"`
	Pct   float64 `json:"pct"`
}

type HourlyMetric struct {
	Timestamp    time.Time `json:"timestamp"`
	TenantID     string    `json:"tenant_id"`
	AgentID      string    `json:"agent_id,omitempty"`
	Channel      string    `json:"channel,omitempty"`
	Model        string    `json:"model,omitempty"`
	Messages     int64     `json:"messages"`
	TokensIn     int64     `json:"tokens_in"`
	TokensOut    int64     `json:"tokens_out"`
	APICalls     int64     `json:"api_calls"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	P95LatencyMs float64   `json:"p95_latency_ms"`
	ErrorCount   int64     `json:"error_count"`
	CostEstimate float64   `json:"cost_estimate"`
}

type DailyMetric struct {
	Date         time.Time `json:"date"`
	TenantID     string    `json:"tenant_id"`
	AgentID      string    `json:"agent_id,omitempty"`
	Channel      string    `json:"channel,omitempty"`
	Model        string    `json:"model,omitempty"`
	Messages     int64     `json:"messages"`
	TokensIn     int64     `json:"tokens_in"`
	TokensOut    int64     `json:"tokens_out"`
	APICalls     int64     `json:"api_calls"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	ErrorCount   int64     `json:"error_count"`
	CostEstimate float64   `json:"cost_estimate"`
	DAU          int64     `json:"dau"`
	Sessions     int64     `json:"sessions"`
}

type ModelPricing struct {
	Model       string    `json:"model"`
	Provider    string    `json:"provider"`
	InputPrice  float64   `json:"input_price_per_million"`
	OutputPrice float64   `json:"output_price_per_million"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (p *ModelPricing) CalculateCost(tokensIn, tokensOut int64) float64 {
	inputCost := (float64(tokensIn) / 1_000_000) * p.InputPrice
	outputCost := (float64(tokensOut) / 1_000_000) * p.OutputPrice
	return inputCost + outputCost
}

type CollectorConfig struct {
	BatchSize     int
	FlushInterval time.Duration
	BufferSize    int
	Enabled       bool
	SampleRate    float64
}

func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		BufferSize:    10000,
		Enabled:       true,
		SampleRate:    1.0,
	}
}

type EventBuffer struct {
	Events []*Event
	Size   int
}

func NewEventBuffer(size int) *EventBuffer {
	return &EventBuffer{
		Events: make([]*Event, 0, size),
		Size:   size,
	}
}

func (b *EventBuffer) Add(event *Event) bool {
	if len(b.Events) >= b.Size {
		return false
	}
	b.Events = append(b.Events, event)
	return true
}

func (b *EventBuffer) IsFull() bool {
	return len(b.Events) >= b.Size
}

func (b *EventBuffer) Clear() {
	b.Events = b.Events[:0]
}

func (b *EventBuffer) Len() int {
	return len(b.Events)
}

func (b *EventBuffer) GetEvents() []*Event {
	return b.Events
}
