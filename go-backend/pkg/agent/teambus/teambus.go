// OctAi - Team Event Bus
// Provides team-scoped event broadcasting built on the agent EventBus.
package teambus

import (
	"sync"
)

// TeamEvent represents a message broadcast within a team.
type TeamEvent struct {
	// TeamID identifies which team this event belongs to.
	TeamID string
	// Kind describes the type of team event.
	Kind TeamEventKind
	// SenderAgentID is the agent that emitted this event.
	SenderAgentID string
	// TargetAgentID is empty for broadcast, or set for directed messages.
	TargetAgentID string
	// TargetRole is empty for broadcast, or filters by role.
	TargetRole string
	// Payload holds event-specific data.
	Payload any
}

// TeamEventKind identifies the type of a TeamEvent.
type TeamEventKind string

const (
	// TeamEventTaskAssigned signals that the orchestrator has assigned a task to a member.
	TeamEventTaskAssigned TeamEventKind = "team_task_assigned"
	// TeamEventTaskCompleted signals that a member agent completed its assigned task.
	TeamEventTaskCompleted TeamEventKind = "team_task_completed"
	// TeamEventBroadcast is a general message sent to all (or a role subset of) team members.
	TeamEventBroadcast TeamEventKind = "team_broadcast"
	// TeamEventContextShared signals that the orchestrator has shared context with the team.
	TeamEventContextShared TeamEventKind = "team_context_shared"
)

// Subscriber receives TeamEvents.
type Subscriber func(evt TeamEvent)

// TeamBus is a lightweight pub/sub bus scoped to a single team.
// It is safe for concurrent use.
type TeamBus struct {
	teamID      string
	subscribers []Subscriber
	mu          sync.RWMutex
}

// NewTeamBus creates a TeamBus for the given team.
func NewTeamBus(teamID string) *TeamBus {
	return &TeamBus{teamID: teamID}
}

// Subscribe registers a subscriber and returns an unsubscribe function.
func (tb *TeamBus) Subscribe(fn Subscriber) func() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	idx := len(tb.subscribers)
	tb.subscribers = append(tb.subscribers, fn)

	return func() {
		tb.mu.Lock()
		defer tb.mu.Unlock()
		if idx < len(tb.subscribers) {
			tb.subscribers[idx] = nil
		}
	}
}

// Publish broadcasts a TeamEvent to all active subscribers.
// Nil subscriber slots (from unsubscription) are skipped.
func (tb *TeamBus) Publish(evt TeamEvent) {
	evt.TeamID = tb.teamID

	tb.mu.RLock()
	subs := make([]Subscriber, len(tb.subscribers))
	copy(subs, tb.subscribers)
	tb.mu.RUnlock()

	for _, sub := range subs {
		if sub != nil {
			sub(evt)
		}
	}
}

// Broadcast sends a broadcast event to all team members.
func (tb *TeamBus) Broadcast(senderID string, payload any) {
	tb.Publish(TeamEvent{
		Kind:          TeamEventBroadcast,
		SenderAgentID: senderID,
		Payload:       payload,
	})
}

// ShareContext notifies team members that new shared context is available.
func (tb *TeamBus) ShareContext(senderID string, context string) {
	tb.Publish(TeamEvent{
		Kind:          TeamEventContextShared,
		SenderAgentID: senderID,
		Payload:       context,
	})
}

// Registry manages TeamBus instances across all active teams.
type Registry struct {
	buses map[string]*TeamBus
	mu    sync.RWMutex
}

// NewRegistry creates an empty TeamBus registry.
func NewRegistry() *Registry {
	return &Registry{buses: make(map[string]*TeamBus)}
}

// GetOrCreate returns the TeamBus for teamID, creating it if needed.
func (r *Registry) GetOrCreate(teamID string) *TeamBus {
	r.mu.RLock()
	if tb, ok := r.buses[teamID]; ok {
		r.mu.RUnlock()
		return tb
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()
	if tb, ok := r.buses[teamID]; ok {
		return tb
	}
	tb := NewTeamBus(teamID)
	r.buses[teamID] = tb
	return tb
}

// Get returns the TeamBus for teamID, or nil if it does not exist.
func (r *Registry) Get(teamID string) *TeamBus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.buses[teamID]
}
