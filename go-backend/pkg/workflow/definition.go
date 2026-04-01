// OctAi - Workflow Engine: Definitions
// Defines the serializable data structures for workflow DAGs.
package workflow

import "time"

// WorkflowDefinition is the top-level serializable workflow structure.
// It describes a directed acyclic graph of nodes connected by edges.
type WorkflowDefinition struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	TeamID      string            `json:"team_id,omitempty"`
	OrgID       string            `json:"org_id,omitempty"`
	Nodes       []NodeDefinition  `json:"nodes"`
	Triggers    []TriggerDef      `json:"triggers,omitempty"`
	CreatedAt   time.Time         `json:"created_at,omitempty"`
	UpdatedAt   time.Time         `json:"updated_at,omitempty"`
}

// NodeDefinition describes a single node in the workflow DAG.
type NodeDefinition struct {
	// ID uniquely identifies this node within the workflow.
	ID string `json:"id"`
	// Type determines how the node executes.
	// One of: agent_task, condition, parallel, loop, wait, webhook
	Type string `json:"type"`
	// Label is a human-readable name shown in the UI.
	Label string `json:"label,omitempty"`
	// DependsOn lists node IDs that must complete before this node can run.
	DependsOn []string `json:"depends_on,omitempty"`
	// Config holds node-type-specific configuration.
	Config NodeConfig `json:"config"`
}

// NodeConfig holds per-node-type configuration.
type NodeConfig struct {
	// AgentTask fields
	AgentRole    string `json:"agent_role,omitempty"`    // target role for agent_task nodes
	AgentID      string `json:"agent_id,omitempty"`      // explicit agent ID (alternative to role)
	Task         string `json:"task,omitempty"`          // task prompt for the agent
	// Template fields — {{.PrevResult}} is substituted from the upstream node's result
	TaskTemplate string `json:"task_template,omitempty"`

	// Condition fields
	Condition    string `json:"condition,omitempty"` // expression evaluated against node results
	TrueNode     string `json:"true_node,omitempty"` // node ID to execute if condition is true
	FalseNode    string `json:"false_node,omitempty"`

	// Loop fields
	LoopCount    int    `json:"loop_count,omitempty"`
	LoopNodeID   string `json:"loop_node_id,omitempty"`

	// Parallel fields
	NodeIDs      []string `json:"node_ids,omitempty"` // nodes to run in parallel

	// Wait fields
	DurationSec  int `json:"duration_sec,omitempty"`

	// Webhook fields
	WebhookURL   string            `json:"webhook_url,omitempty"`
	WebhookMethod string           `json:"webhook_method,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
}

// TriggerDef describes what starts a workflow run.
type TriggerDef struct {
	// Type is one of: schedule, webhook, event, manual
	Type string `json:"type"`
	// Schedule is a cron expression (for schedule triggers).
	Schedule string `json:"schedule,omitempty"`
	// EventKind is the agent event kind to listen for (for event triggers).
	EventKind string `json:"event_kind,omitempty"`
	// WebhookPath is the HTTP path for webhook triggers.
	WebhookPath string `json:"webhook_path,omitempty"`
}

// RunStatus describes the state of a workflow run.
type RunStatus string

const (
	RunStatusPending   RunStatus = "pending"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCanceled  RunStatus = "canceled"
)

// WorkflowRun tracks the execution state of a single workflow invocation.
type WorkflowRun struct {
	ID         string            `json:"id"`
	WorkflowID string            `json:"workflow_id"`
	Status     RunStatus         `json:"status"`
	NodeStates map[string]NodeRunState `json:"node_states,omitempty"`
	StartedAt  time.Time         `json:"started_at,omitempty"`
	EndedAt    time.Time         `json:"ended_at,omitempty"`
	Error      string            `json:"error,omitempty"`
}

// NodeRunState captures the execution result of a single node.
type NodeRunState struct {
	NodeID    string    `json:"node_id"`
	Status    RunStatus `json:"status"`
	Result    string    `json:"result,omitempty"`
	Error     string    `json:"error,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}
