// OctAi - Observability Alerts
// Defines alert rules that fire when thresholds are breached.
package observability

import (
	"fmt"
	"time"
)

// AlertSeverity indicates how serious an alert is.
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert is a fired alert instance.
type Alert struct {
	RuleName  string        `json:"rule_name"`
	Severity  AlertSeverity `json:"severity"`
	AgentID   string        `json:"agent_id,omitempty"`
	Message   string        `json:"message"`
	Value     float64       `json:"value"`
	Threshold float64       `json:"threshold"`
	FiredAt   time.Time     `json:"fired_at"`
}

// AlertRule evaluates a condition and produces alerts when breached.
type AlertRule interface {
	// Name returns the rule's unique identifier.
	Name() string
	// Evaluate checks current state and returns any fired alerts.
	Evaluate(health *HealthTracker, costs *CostTracker) []Alert
}

// CostThresholdRule fires when total cost exceeds a limit.
type CostThresholdRule struct {
	RuleName  string
	Threshold float64 // USD
	Severity  AlertSeverity
}

func (r *CostThresholdRule) Name() string { return r.RuleName }

func (r *CostThresholdRule) Evaluate(_ *HealthTracker, costs *CostTracker) []Alert {
	total := costs.TotalCostUSD()
	if total >= r.Threshold {
		return []Alert{{
			RuleName:  r.RuleName,
			Severity:  r.Severity,
			Message:   fmt.Sprintf("Total LLM cost $%.4f exceeds threshold $%.2f", total, r.Threshold),
			Value:     total,
			Threshold: r.Threshold,
			FiredAt:   time.Now(),
		}}
	}
	return nil
}

// ErrorRateRule fires when an agent's success rate drops below a threshold.
type ErrorRateRule struct {
	RuleName        string
	MinSuccessRate  float64 // 0.0–1.0
	MinTurns        int64   // don't fire until this many turns have been observed
	Severity        AlertSeverity
}

func (r *ErrorRateRule) Name() string { return r.RuleName }

func (r *ErrorRateRule) Evaluate(health *HealthTracker, _ *CostTracker) []Alert {
	var alerts []Alert
	for _, snap := range health.All() {
		if snap.TurnsTotal < r.MinTurns {
			continue
		}
		if snap.SuccessRate < r.MinSuccessRate {
			alerts = append(alerts, Alert{
				RuleName:  r.RuleName,
				Severity:  r.Severity,
				AgentID:   snap.AgentID,
				Message:   fmt.Sprintf("Agent %s success rate %.0f%% below threshold %.0f%%", snap.AgentID, snap.SuccessRate*100, r.MinSuccessRate*100),
				Value:     snap.SuccessRate,
				Threshold: r.MinSuccessRate,
				FiredAt:   time.Now(),
			})
		}
	}
	return alerts
}

// AlertManager runs a set of alert rules and collects fired alerts.
type AlertManager struct {
	rules   []AlertRule
	health  *HealthTracker
	costs   *CostTracker
	handler func(Alert)
}

// NewAlertManager creates an AlertManager with default rules.
func NewAlertManager(health *HealthTracker, costs *CostTracker, handler func(Alert)) *AlertManager {
	return &AlertManager{
		rules: []AlertRule{
			&CostThresholdRule{
				RuleName:  "high_cost",
				Threshold: 10.0, // $10 default warning
				Severity:  AlertSeverityWarning,
			},
			&CostThresholdRule{
				RuleName:  "critical_cost",
				Threshold: 50.0,
				Severity:  AlertSeverityCritical,
			},
			&ErrorRateRule{
				RuleName:       "low_success_rate",
				MinSuccessRate: 0.7,
				MinTurns:       10,
				Severity:       AlertSeverityWarning,
			},
		},
		health:  health,
		costs:   costs,
		handler: handler,
	}
}

// AddRule adds a custom alert rule.
func (am *AlertManager) AddRule(rule AlertRule) {
	am.rules = append(am.rules, rule)
}

// Evaluate runs all rules and calls the handler for each fired alert.
func (am *AlertManager) Evaluate() {
	for _, rule := range am.rules {
		for _, alert := range rule.Evaluate(am.health, am.costs) {
			if am.handler != nil {
				am.handler(alert)
			}
		}
	}
}
