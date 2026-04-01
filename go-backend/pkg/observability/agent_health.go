// OctAi - Agent Health Observability
// Tracks per-agent health metrics by subscribing to the EventBus.
package observability

import (
	"sync"
	"sync/atomic"
	"time"
)

// AgentState describes the lifecycle state of an agent.
type AgentState string

const (
	AgentStateInitializing AgentState = "initializing"
	AgentStateReady        AgentState = "ready"
	AgentStateBusy         AgentState = "busy"
	AgentStateDegraded     AgentState = "degraded"
	AgentStateRetired      AgentState = "retired"
)

// AgentHealthSnapshot is a point-in-time health snapshot for an agent.
type AgentHealthSnapshot struct {
	AgentID      string     `json:"agent_id"`
	State        AgentState `json:"state"`
	TurnsTotal   int64      `json:"turns_total"`
	TurnsSuccess int64      `json:"turns_success"`
	TurnsFailed  int64      `json:"turns_failed"`
	SuccessRate  float64    `json:"success_rate"`
	AvgLatencyMs int64      `json:"avg_latency_ms"`
	ActiveTurns  int64      `json:"active_turns"`
	TokensUsed   int64      `json:"tokens_used"`
	LastSeenAt   time.Time  `json:"last_seen_at"`
}

// agentHealth holds mutable health counters for a single agent.
type agentHealth struct {
	state         atomic.Value // AgentState
	turnsTotal    atomic.Int64
	turnsSuccess  atomic.Int64
	turnsFailed   atomic.Int64
	activeTurns   atomic.Int64
	tokensUsed    atomic.Int64
	totalLatencyMs atomic.Int64
	lastSeenAt    atomic.Value // time.Time
}

func newAgentHealth() *agentHealth {
	h := &agentHealth{}
	h.state.Store(AgentStateReady)
	h.lastSeenAt.Store(time.Now())
	return h
}

func (h *agentHealth) snapshot(agentID string) AgentHealthSnapshot {
	total := h.turnsTotal.Load()
	success := h.turnsSuccess.Load()
	failed := h.turnsFailed.Load()
	active := h.activeTurns.Load()

	var successRate float64
	if total > 0 {
		successRate = float64(success) / float64(total)
	}
	var avgLatency int64
	if total > 0 {
		avgLatency = h.totalLatencyMs.Load() / total
	}

	var lastSeen time.Time
	if t, ok := h.lastSeenAt.Load().(time.Time); ok {
		lastSeen = t
	}

	state := AgentStateReady
	if s, ok := h.state.Load().(AgentState); ok {
		state = s
	}
	if active > 0 {
		state = AgentStateBusy
	}
	if total > 10 && successRate < 0.5 {
		state = AgentStateDegraded
	}

	return AgentHealthSnapshot{
		AgentID:      agentID,
		State:        state,
		TurnsTotal:   total,
		TurnsSuccess: success,
		TurnsFailed:  failed,
		SuccessRate:  successRate,
		AvgLatencyMs: avgLatency,
		ActiveTurns:  active,
		TokensUsed:   h.tokensUsed.Load(),
		LastSeenAt:   lastSeen,
	}
}

// HealthTracker maintains health snapshots for all agents.
// It is updated by the EventBus subscriber in the AgentLoop.
type HealthTracker struct {
	agents map[string]*agentHealth
	mu     sync.RWMutex
}

// NewHealthTracker creates an empty HealthTracker.
func NewHealthTracker() *HealthTracker {
	return &HealthTracker{agents: make(map[string]*agentHealth)}
}

func (t *HealthTracker) getOrCreate(agentID string) *agentHealth {
	t.mu.RLock()
	h, ok := t.agents[agentID]
	t.mu.RUnlock()
	if ok {
		return h
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if h, ok = t.agents[agentID]; ok {
		return h
	}
	h = newAgentHealth()
	t.agents[agentID] = h
	return h
}

// RecordTurnStart marks an agent as having started a turn.
func (t *HealthTracker) RecordTurnStart(agentID string) {
	h := t.getOrCreate(agentID)
	h.activeTurns.Add(1)
	h.lastSeenAt.Store(time.Now())
}

// RecordTurnEnd marks a turn as completed or failed and records its latency.
func (t *HealthTracker) RecordTurnEnd(agentID string, success bool, latencyMs int64) {
	h := t.getOrCreate(agentID)
	h.activeTurns.Add(-1)
	h.turnsTotal.Add(1)
	h.totalLatencyMs.Add(latencyMs)
	if success {
		h.turnsSuccess.Add(1)
	} else {
		h.turnsFailed.Add(1)
	}
	h.lastSeenAt.Store(time.Now())
}

// RecordTokens adds to an agent's cumulative token count.
func (t *HealthTracker) RecordTokens(agentID string, tokens int64) {
	h := t.getOrCreate(agentID)
	h.tokensUsed.Add(tokens)
}

// Snapshot returns a health snapshot for a specific agent.
func (t *HealthTracker) Snapshot(agentID string) (AgentHealthSnapshot, bool) {
	t.mu.RLock()
	h, ok := t.agents[agentID]
	t.mu.RUnlock()
	if !ok {
		return AgentHealthSnapshot{}, false
	}
	return h.snapshot(agentID), true
}

// All returns health snapshots for all tracked agents.
func (t *HealthTracker) All() []AgentHealthSnapshot {
	t.mu.RLock()
	ids := make([]string, 0, len(t.agents))
	for id := range t.agents {
		ids = append(ids, id)
	}
	t.mu.RUnlock()

	snapshots := make([]AgentHealthSnapshot, 0, len(ids))
	for _, id := range ids {
		t.mu.RLock()
		h := t.agents[id]
		t.mu.RUnlock()
		snapshots = append(snapshots, h.snapshot(id))
	}
	return snapshots
}
