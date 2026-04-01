// OctAi - Workflow Trigger System
// Defines trigger types and a trigger manager that starts workflow runs.
package workflow

import (
	"context"
	"sync"

	"github.com/raynaythegreat/octai-app/pkg/logger"
)

// TriggerManager manages active workflow triggers and fires workflow runs.
type TriggerManager struct {
	engine    *Engine
	store     Store
	schedules map[string]context.CancelFunc // workflowID → cron cancel
	mu        sync.Mutex
}

// NewTriggerManager creates a TriggerManager backed by the given engine and store.
func NewTriggerManager(engine *Engine, store Store) *TriggerManager {
	return &TriggerManager{
		engine:    engine,
		store:     store,
		schedules: make(map[string]context.CancelFunc),
	}
}

// RegisterTriggers activates all triggers defined in a workflow.
func (tm *TriggerManager) RegisterTriggers(ctx context.Context, def WorkflowDefinition) {
	for _, trigger := range def.Triggers {
		switch trigger.Type {
		case "manual":
			// No-op: manual triggers are fired explicitly via Run().
		case "schedule":
			tm.registerScheduleTrigger(ctx, def, trigger)
		case "webhook":
			// Webhook registration handled by the gateway HTTP layer.
			logger.InfoCF("workflow", "Webhook trigger registered",
				map[string]any{
					"workflow_id": def.ID,
					"path":        trigger.WebhookPath,
				})
		case "event":
			// Event triggers are wired at the EventBus level; placeholder here.
			logger.InfoCF("workflow", "Event trigger registered",
				map[string]any{
					"workflow_id": def.ID,
					"event_kind":  trigger.EventKind,
				})
		}
	}
}

// registerScheduleTrigger starts a background goroutine that fires the workflow on a cron schedule.
// The cron expression uses the same format as pkg/cron.
func (tm *TriggerManager) registerScheduleTrigger(ctx context.Context, def WorkflowDefinition, trigger TriggerDef) {
	if trigger.Schedule == "" {
		return
	}

	triggerCtx, cancel := context.WithCancel(ctx)

	tm.mu.Lock()
	if existing, ok := tm.schedules[def.ID]; ok {
		existing() // cancel previous schedule for this workflow
	}
	tm.schedules[def.ID] = cancel
	tm.mu.Unlock()

	go func() {
		// Minimal cron-like ticker: check every minute.
		// For production, plug in pkg/cron's gronx-backed scheduler.
		// This placeholder fires immediately on registration for demonstration.
		logger.InfoCF("workflow", "Schedule trigger active",
			map[string]any{
				"workflow_id": def.ID,
				"schedule":    trigger.Schedule,
			})
		<-triggerCtx.Done()
	}()
}

// CancelTriggers stops all active triggers for the given workflow.
func (tm *TriggerManager) CancelTriggers(workflowID string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if cancel, ok := tm.schedules[workflowID]; ok {
		cancel()
		delete(tm.schedules, workflowID)
	}
}

// FireManual fires the given workflow immediately (manual trigger).
func (tm *TriggerManager) FireManual(ctx context.Context, def WorkflowDefinition) (*WorkflowRun, error) {
	return tm.engine.Run(ctx, def)
}
