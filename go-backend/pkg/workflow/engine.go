// OctAi - Workflow Engine
// Executes workflow DAGs by dispatching agent tasks through the SubTurnSpawner.
package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/pkg/tools"
)

// AgentDispatcher is the interface the workflow engine uses to execute agent tasks.
// Implemented by agent.AgentLoopSpawner so the workflow package doesn't import agent.
type AgentDispatcher interface {
	// Dispatch sends a task to an agent (by role or ID) and returns the result.
	Dispatch(ctx context.Context, agentID, task string) (string, error)
}

// Engine executes workflow DAGs.
type Engine struct {
	spawner  tools.SubTurnSpawner
	resolver AgentRoleResolver
	store    Store
}

// AgentRoleResolver resolves a role to an agent ID within a team.
type AgentRoleResolver interface {
	FindByRole(teamID, role string) (string, error)
}

// NewEngine creates a workflow engine.
func NewEngine(spawner tools.SubTurnSpawner, resolver AgentRoleResolver, store Store) *Engine {
	return &Engine{
		spawner:  spawner,
		resolver: resolver,
		store:    store,
	}
}

// Run starts a new workflow run and executes it to completion (or failure).
func (e *Engine) Run(ctx context.Context, def WorkflowDefinition) (*WorkflowRun, error) {
	run := &WorkflowRun{
		ID:         newRunID(),
		WorkflowID: def.ID,
		Status:     RunStatusRunning,
		NodeStates: make(map[string]NodeRunState),
		StartedAt:  time.Now(),
	}

	logger.InfoCF("workflow", "Starting workflow run",
		map[string]any{
			"workflow_id": def.ID,
			"run_id":      run.ID,
			"node_count":  len(def.Nodes),
		})

	if e.store != nil {
		_ = e.store.SaveRun(ctx, run)
	}

	err := e.executeDAG(ctx, def, run)
	run.EndedAt = time.Now()
	if err != nil {
		run.Status = RunStatusFailed
		run.Error = err.Error()
	} else {
		run.Status = RunStatusCompleted
	}

	if e.store != nil {
		_ = e.store.SaveRun(ctx, run)
	}

	logger.InfoCF("workflow", "Workflow run finished",
		map[string]any{
			"workflow_id": def.ID,
			"run_id":      run.ID,
			"status":      string(run.Status),
			"duration_ms": run.EndedAt.Sub(run.StartedAt).Milliseconds(),
		})

	return run, err
}

// executeDAG runs a topological sort and executes nodes in dependency order.
func (e *Engine) executeDAG(ctx context.Context, def WorkflowDefinition, run *WorkflowRun) error {
	// Build adjacency / in-degree map.
	nodeMap := make(map[string]NodeDefinition, len(def.Nodes))
	inDegree := make(map[string]int)
	dependents := make(map[string][]string) // node → nodes that depend on it

	for _, n := range def.Nodes {
		nodeMap[n.ID] = n
		if _, ok := inDegree[n.ID]; !ok {
			inDegree[n.ID] = 0
		}
		for _, dep := range n.DependsOn {
			inDegree[n.ID]++
			dependents[dep] = append(dependents[dep], n.ID)
		}
	}

	// Collect root nodes (no dependencies).
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var mu sync.Mutex
	results := make(map[string]string) // node ID → text result

	for len(queue) > 0 {
		// Pop all ready nodes and run them concurrently.
		batch := queue
		queue = nil

		var wg sync.WaitGroup
		errs := make([]error, len(batch))

		for i, nodeID := range batch {
			wg.Add(1)
			go func(idx int, id string) {
				defer wg.Done()
				node := nodeMap[id]

				// Build context from upstream results.
				mu.Lock()
				upstreamCtx := buildUpstreamContext(node, results)
				mu.Unlock()

				result, err := e.executeNode(ctx, node, def.TeamID, upstreamCtx)
				ns := NodeRunState{
					NodeID:    id,
					StartedAt: time.Now(),
					EndedAt:   time.Now(),
				}
				if err != nil {
					ns.Status = RunStatusFailed
					ns.Error = err.Error()
					errs[idx] = fmt.Errorf("node %q failed: %w", id, err)
				} else {
					ns.Status = RunStatusCompleted
					ns.Result = result
				}

				mu.Lock()
				run.NodeStates[id] = ns
				if err == nil {
					results[id] = result
				}
				mu.Unlock()
			}(i, nodeID)
		}
		wg.Wait()

		// Collect errors.
		for _, err := range errs {
			if err != nil {
				return err
			}
		}

		// Reduce in-degrees of dependents and enqueue newly ready nodes.
		mu.Lock()
		for _, nodeID := range batch {
			for _, dep := range dependents[nodeID] {
				inDegree[dep]--
				if inDegree[dep] == 0 {
					queue = append(queue, dep)
				}
			}
		}
		mu.Unlock()
	}

	return nil
}

// executeNode dispatches a single workflow node to the appropriate executor.
func (e *Engine) executeNode(ctx context.Context, node NodeDefinition, teamID, upstreamCtx string) (string, error) {
	switch node.Type {
	case "agent_task":
		return e.runAgentTask(ctx, node, teamID, upstreamCtx)
	case "wait":
		dur := time.Duration(node.Config.DurationSec) * time.Second
		if dur > 0 {
			select {
			case <-time.After(dur):
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		return "wait completed", nil
	case "condition":
		// Simple condition: check if upstream result contains a truthy string.
		if strings.Contains(strings.ToLower(upstreamCtx), "yes") ||
			strings.Contains(strings.ToLower(upstreamCtx), "true") ||
			strings.Contains(strings.ToLower(upstreamCtx), "success") {
			return node.Config.TrueNode, nil
		}
		return node.Config.FalseNode, nil
	default:
		return "", fmt.Errorf("unsupported node type: %q", node.Type)
	}
}

// runAgentTask delegates a task to an agent via SubTurnSpawner.
func (e *Engine) runAgentTask(ctx context.Context, node NodeDefinition, teamID, upstreamCtx string) (string, error) {
	if e.spawner == nil {
		return "", fmt.Errorf("workflow engine: spawner not configured")
	}

	task := node.Config.Task
	if node.Config.TaskTemplate != "" && upstreamCtx != "" {
		task = strings.ReplaceAll(node.Config.TaskTemplate, "{{.PrevResult}}", upstreamCtx)
	}
	if task == "" {
		return "", fmt.Errorf("node %q: task is empty", node.ID)
	}

	systemPrompt := task
	if upstreamCtx != "" {
		systemPrompt = "## Context from previous steps\n" + upstreamCtx + "\n\n## Your task\n" + task
	}

	result, err := e.spawner.SpawnSubTurn(ctx, tools.SubTurnConfig{
		SystemPrompt: systemPrompt,
		Async:        false,
	})
	if err != nil {
		return "", err
	}
	return result.ForLLM, nil
}

// buildUpstreamContext collects results from all dependencies and concatenates them.
func buildUpstreamContext(node NodeDefinition, results map[string]string) string {
	if len(node.DependsOn) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, dep := range node.DependsOn {
		if r, ok := results[dep]; ok && r != "" {
			sb.WriteString(fmt.Sprintf("[%s result]: %s\n\n", dep, r))
		}
	}
	return sb.String()
}

func newRunID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "run-" + hex.EncodeToString(b)
}
