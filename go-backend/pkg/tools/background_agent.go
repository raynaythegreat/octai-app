package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// BackgroundSubmitter is the interface the gateway implements to accept
// background tasks. The gateway creates a task record, enqueues execution,
// and returns a stable task ID that the caller can use with check_background.
type BackgroundSubmitter interface {
	Submit(ctx context.Context, prompt, description string) (taskID string, err error)
}

// BackgroundTaskStore stores and retrieves results of background tasks.
type BackgroundTaskStore interface {
	Get(ctx context.Context, taskID string) (*BackgroundTask, error)
}

// BackgroundTask holds the full state of a background task at a point in time.
type BackgroundTask struct {
	ID          string     `json:"id"`
	Prompt      string     `json:"prompt"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"` // "pending", "running", "completed", "failed"
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// --- BackgroundAgentTool ---

// BackgroundAgentTool spawns a sub-task that runs independently.
// It returns immediately with a task ID; the agent can check status later
// using check_background.
type BackgroundAgentTool struct {
	submitter BackgroundSubmitter
}

// NewBackgroundAgentTool creates a BackgroundAgentTool. submitter may be nil
// (e.g. during tests or when the gateway has not been wired up yet); in that
// case Execute returns a descriptive error message rather than panicking.
func NewBackgroundAgentTool(submitter BackgroundSubmitter) *BackgroundAgentTool {
	return &BackgroundAgentTool{submitter: submitter}
}

func (b *BackgroundAgentTool) Name() string { return "background_agent" }

func (b *BackgroundAgentTool) Description() string {
	return "Spawn a background sub-task that runs independently. Returns a task_id immediately — use check_background to get the result later. Use for long-running operations like research, data processing, or tasks that shouldn't block the current response."
}

func (b *BackgroundAgentTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "The task for the background agent to complete",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Brief description of what this task does (shown in status)",
			},
		},
		"required": []string{"prompt"},
	}
}

func (b *BackgroundAgentTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if b.submitter == nil {
		return ErrorResult("Background agents require the gateway to be running.")
	}

	prompt, ok := args["prompt"].(string)
	if !ok || strings.TrimSpace(prompt) == "" {
		return ErrorResult("prompt is required and must be a non-empty string")
	}

	description, _ := args["description"].(string)

	taskID, err := b.submitter.Submit(ctx, prompt, description)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to submit background task: %v", err))
	}

	return NewToolResult(fmt.Sprintf(
		"Background task started: %s. Check result with check_background tool.", taskID,
	))
}

// --- BackgroundResultTool ---

// BackgroundResultTool checks the status/result of a background task.
type BackgroundResultTool struct {
	store BackgroundTaskStore
}

// NewBackgroundResultTool creates a BackgroundResultTool. store may be nil;
// Execute returns a descriptive message rather than panicking.
func NewBackgroundResultTool(store BackgroundTaskStore) *BackgroundResultTool {
	return &BackgroundResultTool{store: store}
}

func (b *BackgroundResultTool) Name() string { return "check_background" }

func (b *BackgroundResultTool) Description() string {
	return "Check the status and result of a background task spawned with background_agent."
}

func (b *BackgroundResultTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "The task ID returned by background_agent",
			},
		},
		"required": []string{"task_id"},
	}
}

func (b *BackgroundResultTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	if b.store == nil {
		return ErrorResult("Background task store not available.")
	}

	taskID, ok := args["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return ErrorResult("task_id is required and must be a non-empty string")
	}

	task, err := b.store.Get(ctx, strings.TrimSpace(taskID))
	if err != nil {
		return ErrorResult(fmt.Sprintf("Error retrieving task %s: %v", taskID, err))
	}
	if task == nil {
		return ErrorResult(fmt.Sprintf("No background task found with ID: %s", taskID))
	}

	return NewToolResult(formatBackgroundTask(task))
}

// formatBackgroundTask renders a BackgroundTask as a human-readable string.
func formatBackgroundTask(t *BackgroundTask) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] status=%s", t.ID, t.Status))

	if t.Description != "" {
		sb.WriteString(fmt.Sprintf("  description=%q", t.Description))
	}

	sb.WriteString(fmt.Sprintf("\n  created:   %s", t.CreatedAt.UTC().Format("2006-01-02 15:04:05 UTC")))

	if t.CompletedAt != nil {
		sb.WriteString(fmt.Sprintf("\n  completed: %s", t.CompletedAt.UTC().Format("2006-01-02 15:04:05 UTC")))
	}

	if t.Prompt != "" {
		prompt := t.Prompt
		const maxPromptLen = 200
		runes := []rune(prompt)
		if len(runes) > maxPromptLen {
			prompt = string(runes[:maxPromptLen]) + "…"
		}
		sb.WriteString(fmt.Sprintf("\n  prompt:    %s", prompt))
	}

	switch t.Status {
	case "completed":
		result := t.Result
		const maxResultLen = 500
		runes := []rune(result)
		if len(runes) > maxResultLen {
			result = string(runes[:maxResultLen]) + "…"
		}
		sb.WriteString(fmt.Sprintf("\n  result:    %s", result))
	case "failed":
		sb.WriteString(fmt.Sprintf("\n  error:     %s", t.Error))
	case "pending", "running":
		sb.WriteString("\n  (task is still in progress)")
	}

	return sb.String()
}

// --- InMemoryBackgroundStore ---

// InMemoryBackgroundStore is a simple in-memory implementation of
// BackgroundTaskStore. It is suitable for single-process deployments and
// testing. Concurrent access is safe.
type InMemoryBackgroundStore struct {
	mu    sync.RWMutex
	tasks map[string]*BackgroundTask
}

// NewInMemoryBackgroundStore creates an empty InMemoryBackgroundStore.
func NewInMemoryBackgroundStore() *InMemoryBackgroundStore {
	return &InMemoryBackgroundStore{
		tasks: make(map[string]*BackgroundTask),
	}
}

// Get retrieves a snapshot of the task with the given ID.
// Returns (nil, nil) when no task exists with that ID.
func (s *InMemoryBackgroundStore) Get(_ context.Context, id string) (*BackgroundTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	if !ok {
		return nil, nil
	}

	// Return a shallow copy to prevent callers from mutating stored state.
	cpy := *task
	if task.CompletedAt != nil {
		t := *task.CompletedAt
		cpy.CompletedAt = &t
	}
	return &cpy, nil
}

// Set stores or replaces the task with task.ID. The store takes its own copy
// so the caller may safely modify the original after returning.
func (s *InMemoryBackgroundStore) Set(task *BackgroundTask) {
	if task == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cpy := *task
	if task.CompletedAt != nil {
		t := *task.CompletedAt
		cpy.CompletedAt = &t
	}
	s.tasks[task.ID] = &cpy
}
