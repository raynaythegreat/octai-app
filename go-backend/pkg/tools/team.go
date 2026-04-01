// OctAi - Team Tool
// Allows an orchestrator agent to delegate tasks to specialist team members.
// Builds on the existing SubTurnSpawner infrastructure — no parallel systems.
package tools

import (
	"context"
	"fmt"
	"strings"
)

// TeamResolver maps roles and agent IDs to concrete agent IDs for the TeamTool.
// Implemented by agent.TeamRegistry — kept as an interface to avoid circular imports.
type TeamResolver interface {
	// FindByRole returns an agent ID for the given team+role combination.
	FindByRole(teamID string, role string) (string, error)
	// TeamOf returns the team ID the calling agent belongs to.
	TeamOf(agentID string) (string, bool)
}

// TeamTool lets an orchestrator agent delegate tasks to specialist team members.
// It uses SubTurnSpawner (the same infrastructure as subagent/spawn tools) so all
// depth limits, concurrency caps, and token budgets are automatically respected.
type TeamTool struct {
	spawner     SubTurnSpawner
	resolver    TeamResolver
	agentID     string // the orchestrator's own agent ID
	defaultModel string
}

// NewTeamTool creates a TeamTool for the given orchestrator agent.
func NewTeamTool(agentID, defaultModel string, resolver TeamResolver) *TeamTool {
	return &TeamTool{
		agentID:      agentID,
		defaultModel: defaultModel,
		resolver:     resolver,
	}
}

// SetSpawner injects the SubTurnSpawner (called after AgentLoop construction to avoid cycles).
func (t *TeamTool) SetSpawner(spawner SubTurnSpawner) {
	t.spawner = spawner
}

func (t *TeamTool) Name() string { return "team" }

func (t *TeamTool) Description() string {
	return "Delegate a task to a specialist team member by role or agent ID. " +
		"Use this when you need research, sales outreach, content creation, analytics, or support from a specialist. " +
		"Specify 'role' (e.g. 'research', 'sales', 'content') to pick by specialization, or 'agent_id' for a specific agent. " +
		"Set 'async' to true for fire-and-forget delegation."
}

func (t *TeamTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task": map[string]any{
				"type":        "string",
				"description": "Clear description of the work to be done, including any relevant context and success criteria.",
			},
			"role": map[string]any{
				"type":        "string",
				"description": "Target member by specialization. One of: orchestrator, sales, support, research, content, analytics, admin, custom.",
				"enum": []string{
					"orchestrator", "sales", "support", "research",
					"content", "analytics", "admin", "custom",
				},
			},
			"agent_id": map[string]any{
				"type":        "string",
				"description": "Target a specific agent by ID (alternative to role).",
			},
			"label": map[string]any{
				"type":        "string",
				"description": "Short identifier for this task, shown in status messages (e.g. 'competitor-research').",
			},
			"context": map[string]any{
				"type":        "string",
				"description": "Additional shared context to prepend to the member agent's system prompt (e.g. prior results, user background).",
			},
			"async": map[string]any{
				"type":        "boolean",
				"description": "If true, delegate in the background and return immediately. Default: false (wait for result).",
			},
		},
		"required": []string{"task"},
	}
}

func (t *TeamTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	task, _ := args["task"].(string)
	if task == "" {
		return ErrorResult("task parameter is required").WithError(fmt.Errorf("task is required"))
	}

	role, _ := args["role"].(string)
	targetAgentID, _ := args["agent_id"].(string)
	label, _ := args["label"].(string)
	extraContext, _ := args["context"].(string)
	async, _ := args["async"].(bool)

	// Resolve the target agent ID.
	resolvedID, err := t.resolveTarget(role, targetAgentID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Cannot resolve team member: %v", err)).WithError(err)
	}

	if t.spawner == nil {
		return ErrorResult("Team tool not fully initialized (spawner not set)").
			WithError(fmt.Errorf("spawner is nil"))
	}

	// Build a role-aware system prompt for the member agent.
	systemPrompt := buildMemberSystemPrompt(role, resolvedID, extraContext, task)

	labelStr := label
	if labelStr == "" {
		if role != "" {
			labelStr = role + "-task"
		} else {
			labelStr = resolvedID + "-task"
		}
	}

	cfg := SubTurnConfig{
		Model:        t.defaultModel,
		SystemPrompt: systemPrompt,
		Async:        async,
	}

	result, err := t.spawner.SpawnSubTurn(ctx, cfg)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Team delegation failed: %v", err)).WithError(err)
	}

	// Format the result for the orchestrator.
	memberDesc := resolvedID
	if role != "" {
		memberDesc = role + " agent (" + resolvedID + ")"
	}

	userSummary := result.ForUser
	if userSummary == "" {
		userSummary = result.ForLLM
	}
	const maxUserLen = 600
	if len(userSummary) > maxUserLen {
		userSummary = userSummary[:maxUserLen] + "..."
	}

	llmContent := fmt.Sprintf(
		"Team delegation result:\nMember: %s\nTask: %s\nLabel: %s\n\nResult:\n%s",
		memberDesc, task, labelStr, result.ForLLM,
	)

	return &ToolResult{
		ForLLM:  llmContent,
		ForUser: fmt.Sprintf("[%s] %s", labelStr, userSummary),
		Silent:  false,
		IsError: result.IsError,
		Async:   async,
	}
}

// resolveTarget determines the concrete agent ID from a role or explicit agent ID.
func (t *TeamTool) resolveTarget(role, agentID string) (string, error) {
	if agentID != "" {
		return agentID, nil
	}
	if role == "" {
		return "", fmt.Errorf("either 'role' or 'agent_id' must be specified")
	}
	if t.resolver == nil {
		return "", fmt.Errorf("team resolver not configured — is this agent part of a team?")
	}
	teamID, ok := t.resolver.TeamOf(t.agentID)
	if !ok {
		return "", fmt.Errorf("agent %q is not a member of any team", t.agentID)
	}
	return t.resolver.FindByRole(teamID, role)
}

// buildMemberSystemPrompt constructs a system prompt for the member agent.
func buildMemberSystemPrompt(role, agentID, extraContext, task string) string {
	var sb strings.Builder

	// Role orientation hint.
	hint := RoleContextHint(role)
	sb.WriteString(hint)
	sb.WriteString("\n\n")

	// Injected shared context from orchestrator.
	if extraContext != "" {
		sb.WriteString("## Context from orchestrator\n")
		sb.WriteString(extraContext)
		sb.WriteString("\n\n")
	}

	// The actual task.
	sb.WriteString("## Your task\n")
	sb.WriteString(task)

	return sb.String()
}
