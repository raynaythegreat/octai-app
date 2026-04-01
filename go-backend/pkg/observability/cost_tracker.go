// OctAi - Cost Tracker
// Tracks real-time LLM token costs per agent and team.
package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

// ModelPricing holds the cost per million tokens for a model.
type ModelPricing struct {
	InputPerMillion  float64 `json:"input_per_million"`
	OutputPerMillion float64 `json:"output_per_million"`
}

// DefaultPricing provides a lookup table for common model pricing (USD per million tokens).
// Update these periodically as provider pricing changes.
var DefaultPricing = map[string]ModelPricing{
	"claude-opus-4-6":            {InputPerMillion: 15.0, OutputPerMillion: 75.0},
	"claude-sonnet-4-6":          {InputPerMillion: 3.0, OutputPerMillion: 15.0},
	"claude-haiku-4-5-20251001":  {InputPerMillion: 0.8, OutputPerMillion: 4.0},
	"gpt-4o":                     {InputPerMillion: 2.5, OutputPerMillion: 10.0},
	"gpt-4o-mini":                {InputPerMillion: 0.15, OutputPerMillion: 0.6},
	"gemini-1.5-pro":             {InputPerMillion: 3.5, OutputPerMillion: 10.5},
}

// CostEntry records token usage and cost for a single LLM request.
type CostEntry struct {
	AgentID     string    `json:"agent_id"`
	Model       string    `json:"model"`
	InputTokens int64     `json:"input_tokens"`
	OutputTokens int64    `json:"output_tokens"`
	CostUSD     float64   `json:"cost_usd"`
	Timestamp   time.Time `json:"timestamp"`
}

// CacheStats holds aggregate cache token metrics for an agent.
type CacheStats struct {
	TotalCacheReads     int     `json:"total_cache_reads"`
	TotalCacheWrites    int     `json:"total_cache_writes"`
	EstimatedSavingsUSD float64 `json:"estimated_savings_usd"` // based on 90% savings on cache reads
}

// agentCostState tracks cumulative cost for a single agent.
type agentCostState struct {
	totalInputTokens      atomic.Int64
	totalOutputTokens     atomic.Int64
	totalCostMicros       atomic.Int64 // USD × 1_000_000 to avoid float races
	totalCacheReadTokens  atomic.Int64
	totalCacheWriteTokens atomic.Int64
	cacheSavingsMicros    atomic.Int64 // USD × 1_000_000
}

// CostTracker accumulates real-time token usage and cost estimates.
type CostTracker struct {
	agents  map[string]*agentCostState
	history []CostEntry
	pricing map[string]ModelPricing
	mu      sync.RWMutex
	maxHistory int
}

// NewCostTracker creates a CostTracker with the given model pricing table.
// Pass nil to use DefaultPricing.
func NewCostTracker(pricing map[string]ModelPricing) *CostTracker {
	if pricing == nil {
		pricing = DefaultPricing
	}
	return &CostTracker{
		agents:     make(map[string]*agentCostState),
		pricing:    pricing,
		maxHistory: 1000,
	}
}

// RecordUsage records token usage for an agent request and computes cost.
// cacheReadTokens and cacheWriteTokens are the cache token counts from UsageInfo
// (pass 0 if the provider does not report them).
//
// Anthropic cache pricing:
//   - Cache reads  : 10% of input price  → 90% savings vs normal input
//   - Cache writes : 110% of input price → slight premium to write the cache
func (ct *CostTracker) RecordUsage(agentID, model string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int64) float64 {
	pricing := ct.pricing[model]

	// Standard token cost (input already excludes cached tokens on Anthropic)
	costUSD := (float64(inputTokens)/1_000_000)*pricing.InputPerMillion +
		(float64(outputTokens)/1_000_000)*pricing.OutputPerMillion

	// Cache write cost: 110% of input price
	if cacheWriteTokens > 0 {
		costUSD += (float64(cacheWriteTokens) / 1_000_000) * pricing.InputPerMillion * 1.1
	}

	// Cache read cost: 10% of input price
	if cacheReadTokens > 0 {
		costUSD += (float64(cacheReadTokens) / 1_000_000) * pricing.InputPerMillion * 0.1
	}

	// Savings = what we would have paid at full input price minus what we paid
	savingsUSD := (float64(cacheReadTokens) / 1_000_000) * pricing.InputPerMillion * 0.9

	ct.mu.Lock()
	state, ok := ct.agents[agentID]
	if !ok {
		state = &agentCostState{}
		ct.agents[agentID] = state
	}
	state.totalInputTokens.Add(inputTokens)
	state.totalOutputTokens.Add(outputTokens)
	state.totalCostMicros.Add(int64(costUSD * 1_000_000))
	state.totalCacheReadTokens.Add(cacheReadTokens)
	state.totalCacheWriteTokens.Add(cacheWriteTokens)
	state.cacheSavingsMicros.Add(int64(savingsUSD * 1_000_000))

	entry := CostEntry{
		AgentID:      agentID,
		Model:        model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CostUSD:      costUSD,
		Timestamp:    time.Now(),
	}
	ct.history = append(ct.history, entry)
	if len(ct.history) > ct.maxHistory {
		ct.history = ct.history[len(ct.history)-ct.maxHistory:]
	}
	ct.mu.Unlock()

	return costUSD
}

// AgentCostSummary returns the total cost for a specific agent.
type AgentCostSummary struct {
	AgentID           string     `json:"agent_id"`
	TotalInputTokens  int64      `json:"total_input_tokens"`
	TotalOutputTokens int64      `json:"total_output_tokens"`
	TotalCostUSD      float64    `json:"total_cost_usd"`
	Cache             CacheStats `json:"cache"`
}

// AgentSummary returns cost summary for a single agent.
func (ct *CostTracker) AgentSummary(agentID string) (AgentCostSummary, bool) {
	ct.mu.RLock()
	state, ok := ct.agents[agentID]
	ct.mu.RUnlock()
	if !ok {
		return AgentCostSummary{}, false
	}
	return AgentCostSummary{
		AgentID:           agentID,
		TotalInputTokens:  state.totalInputTokens.Load(),
		TotalOutputTokens: state.totalOutputTokens.Load(),
		TotalCostUSD:      float64(state.totalCostMicros.Load()) / 1_000_000,
		Cache: CacheStats{
			TotalCacheReads:     int(state.totalCacheReadTokens.Load()),
			TotalCacheWrites:    int(state.totalCacheWriteTokens.Load()),
			EstimatedSavingsUSD: float64(state.cacheSavingsMicros.Load()) / 1_000_000,
		},
	}, true
}

// AgentCacheStats returns cache statistics for a single agent.
func (ct *CostTracker) AgentCacheStats(agentID string) (CacheStats, bool) {
	ct.mu.RLock()
	state, ok := ct.agents[agentID]
	ct.mu.RUnlock()
	if !ok {
		return CacheStats{}, false
	}
	return CacheStats{
		TotalCacheReads:     int(state.totalCacheReadTokens.Load()),
		TotalCacheWrites:    int(state.totalCacheWriteTokens.Load()),
		EstimatedSavingsUSD: float64(state.cacheSavingsMicros.Load()) / 1_000_000,
	}, true
}

// AllSummaries returns cost summaries for all tracked agents.
func (ct *CostTracker) AllSummaries() []AgentCostSummary {
	ct.mu.RLock()
	ids := make([]string, 0, len(ct.agents))
	for id := range ct.agents {
		ids = append(ids, id)
	}
	ct.mu.RUnlock()

	summaries := make([]AgentCostSummary, 0, len(ids))
	for _, id := range ids {
		if s, ok := ct.AgentSummary(id); ok {
			summaries = append(summaries, s)
		}
	}
	return summaries
}

// RecentHistory returns the most recent N cost entries across all agents.
func (ct *CostTracker) RecentHistory(limit int) []CostEntry {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	if limit <= 0 || limit >= len(ct.history) {
		out := make([]CostEntry, len(ct.history))
		copy(out, ct.history)
		return out
	}
	out := make([]CostEntry, limit)
	copy(out, ct.history[len(ct.history)-limit:])
	return out
}

// TotalCostUSD returns the total cost across all agents.
func (ct *CostTracker) TotalCostUSD() float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var total int64
	for _, state := range ct.agents {
		total += state.totalCostMicros.Load()
	}
	return float64(total) / 1_000_000
}
