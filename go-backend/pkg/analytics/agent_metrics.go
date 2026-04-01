// OctAi - Per-Agent Analytics Metrics
// Extends the analytics package with agent-level and team-level performance tracking.
package analytics

import "time"

// AgentMetricEvent is a structured analytics event capturing a completed agent turn.
type AgentMetricEvent struct {
	// AgentID is the agent that processed the turn.
	AgentID string `json:"agent_id"`
	// AgentRole is the agent's assigned role (e.g. "research", "sales").
	AgentRole string `json:"agent_role,omitempty"`
	// TeamID links this event to a team (empty for standalone agents).
	TeamID string `json:"team_id,omitempty"`
	// TurnID is the unique identifier for this turn.
	TurnID string `json:"turn_id"`
	// ParentTurnID is set for sub-turns spawned by an orchestrator.
	ParentTurnID string `json:"parent_turn_id,omitempty"`
	// Model is the LLM model used for this turn.
	Model string `json:"model"`
	// Success indicates whether the turn completed without error.
	Success bool `json:"success"`
	// DurationMs is the wall-clock time for the turn in milliseconds.
	DurationMs int64 `json:"duration_ms"`
	// Iterations is the number of tool-use cycles in the turn.
	Iterations int `json:"iterations"`
	// InputTokens is the number of tokens sent to the LLM.
	InputTokens int64 `json:"input_tokens,omitempty"`
	// OutputTokens is the number of tokens received from the LLM.
	OutputTokens int64 `json:"output_tokens,omitempty"`
	// ToolsUsed lists the tool names called during this turn.
	ToolsUsed []string `json:"tools_used,omitempty"`
	// Channel is the originating channel (e.g. "telegram", "slack").
	Channel string `json:"channel,omitempty"`
	// OrgID is the tenant organization for SaaS deployments.
	OrgID string `json:"org_id,omitempty"`
	// Timestamp is when this event was recorded.
	Timestamp time.Time `json:"timestamp"`
}

// TeamMetricSummary aggregates metrics for an entire team over a time window.
type TeamMetricSummary struct {
	TeamID           string    `json:"team_id"`
	WindowStart      time.Time `json:"window_start"`
	WindowEnd        time.Time `json:"window_end"`
	TotalTurns       int64     `json:"total_turns"`
	SuccessfulTurns  int64     `json:"successful_turns"`
	FailedTurns      int64     `json:"failed_turns"`
	SuccessRate      float64   `json:"success_rate"`
	AvgDurationMs    int64     `json:"avg_duration_ms"`
	TotalTokens      int64     `json:"total_tokens"`
	TotalCostUSD     float64   `json:"total_cost_usd"`
	DelegationCount  int64     `json:"delegation_count"` // number of team-tool calls
}

// ROIMetric estimates the return-on-investment for agent automation.
type ROIMetric struct {
	TeamID            string    `json:"team_id"`
	Period            string    `json:"period"` // e.g. "2026-03"
	TasksAutomated    int64     `json:"tasks_automated"`
	TasksPartial      int64     `json:"tasks_partial"`   // required human intervention
	AutomationRate    float64   `json:"automation_rate"` // fully automated / total
	EstTimeSavedHours float64   `json:"est_time_saved_hours"`
	CostLLMUSD        float64   `json:"cost_llm_usd"`
	// EfficiencyRatio = time_saved / cost. Higher is better.
	EfficiencyRatio   float64   `json:"efficiency_ratio"`
	ComputedAt        time.Time `json:"computed_at"`
}

// EstimateROI computes a basic ROI metric from a team summary.
// avgHumanMinutesPerTask is the estimated manual time for each task.
func EstimateROI(summary TeamMetricSummary, avgHumanMinutesPerTask float64, costUSD float64) ROIMetric {
	automated := summary.SuccessfulTurns
	total := summary.TotalTurns

	var automationRate float64
	if total > 0 {
		automationRate = float64(automated) / float64(total)
	}

	timeSaved := (float64(automated) * avgHumanMinutesPerTask) / 60 // hours

	var efficiency float64
	if costUSD > 0 {
		efficiency = timeSaved / costUSD
	}

	return ROIMetric{
		TeamID:            summary.TeamID,
		TasksAutomated:    automated,
		TasksPartial:      summary.FailedTurns,
		AutomationRate:    automationRate,
		EstTimeSavedHours: timeSaved,
		CostLLMUSD:        costUSD,
		EfficiencyRatio:   efficiency,
		ComputedAt:        time.Now(),
	}
}
