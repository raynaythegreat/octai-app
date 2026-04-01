// OctAi - Team Task Protocol
// Defines structured task schemas for each agent role.
package tools

// TeamTaskRequest is the structured input for delegating a task to a team member.
type TeamTaskRequest struct {
	// Task is the natural-language description of the work to be done.
	Task string
	// Role is the target agent role (e.g. "research", "sales"). Mutually exclusive with AgentID.
	Role string
	// AgentID targets a specific agent by ID. Mutually exclusive with Role.
	AgentID string
	// Label is a short human-readable identifier for the task (used in status messages).
	Label string
	// Context is additional shared context injected into the member agent's system prompt.
	Context string
	// Async, when true, spawns the member agent in the background and returns immediately.
	Async bool
}

// TeamTaskResult is the outcome returned by a team member to the orchestrator.
type TeamTaskResult struct {
	// AgentID is the member agent that produced this result.
	AgentID string
	// Role is the member agent's role.
	Role string
	// Label mirrors the request label for correlation.
	Label string
	// Output is the member agent's response.
	Output string
	// IsError indicates whether the member agent failed.
	IsError bool
	// Error holds the error message when IsError is true.
	Error string
}

// roleContextHints maps role names to a brief context hint injected before the task.
// This gives member agents immediate orientation without needing to read a full config.
var roleContextHints = map[string]string{
	"orchestrator": "You are the team orchestrator. Coordinate work and synthesize results.",
	"sales":        "You are the Sales agent. Focus on CRM data, lead qualification, and outreach.",
	"support":      "You are the Support agent. Focus on resolving customer issues and retrieving knowledge base content.",
	"research":     "You are the Research agent. Conduct thorough web research and synthesize findings with citations.",
	"content":      "You are the Content agent. Produce complete, polished written content ready for publication.",
	"analytics":    "You are the Analytics agent. Analyze data and produce structured reports with actionable insights.",
	"admin":        "You are the Admin agent. Handle configuration, user management, and compliance tasks precisely.",
	"custom":       "You are a specialized agent. Complete the assigned task accurately and efficiently.",
}

// RoleContextHint returns a short context hint for the given role.
// Returns a generic hint for unknown roles.
func RoleContextHint(role string) string {
	if hint, ok := roleContextHints[role]; ok {
		return hint
	}
	return "You are a specialized agent. Complete the assigned task accurately and efficiently."
}
