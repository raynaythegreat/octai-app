// OctAi - Agent Memory Tools
// Provides save_memory and recall_memory tools backed by SQLiteAgentMemoryStore.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/raynaythegreat/octai-app/pkg/memory"
)

// SaveMemoryTool allows an agent to persist information for future sessions.
type SaveMemoryTool struct {
	store   *memory.SQLiteAgentMemoryStore
	agentID string
}

// NewSaveMemoryTool creates a SaveMemoryTool bound to the given store and agent.
func NewSaveMemoryTool(store *memory.SQLiteAgentMemoryStore, agentID string) *SaveMemoryTool {
	return &SaveMemoryTool{store: store, agentID: agentID}
}

func (t *SaveMemoryTool) Name() string { return "save_memory" }

func (t *SaveMemoryTool) Description() string {
	return "Persist important information, user preferences, or decisions that should be remembered across conversations."
}

func (t *SaveMemoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type": map[string]any{
				"type":        "string",
				"description": "Category of memory.",
				"enum":        []string{"user", "project", "feedback", "reference", "fact"},
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The full text of the information to remember.",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "A brief one-line summary used for retrieval (optional).",
			},
			"tags": map[string]any{
				"type":        "array",
				"description": "Optional labels for categorisation and search.",
				"items":       map[string]any{"type": "string"},
			},
		},
		"required": []string{"type", "content"},
	}
}

func (t *SaveMemoryTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	memType, ok := args["type"].(string)
	if !ok || strings.TrimSpace(memType) == "" {
		return ErrorResult("save_memory: 'type' is required and must be a non-empty string.")
	}
	content, ok := args["content"].(string)
	if !ok || strings.TrimSpace(content) == "" {
		return ErrorResult("save_memory: 'content' is required and must be a non-empty string.")
	}

	description, _ := args["description"].(string)

	var tags []string
	if raw, ok := args["tags"]; ok {
		switch v := raw.(type) {
		case []string:
			tags = v
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					tags = append(tags, s)
				}
			}
		}
	}

	m := memory.AgentMemory{
		ID:          uuid.New().String(),
		AgentID:     t.agentID,
		Type:        memory.MemoryType(memType),
		Content:     content,
		Description: description,
		Tags:        tags,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := t.store.Save(ctx, m); err != nil {
		return ErrorResult(fmt.Sprintf("save_memory: failed to save: %v", err)).WithError(err)
	}

	msg := fmt.Sprintf("Memory saved (id=%s, type=%s).", m.ID, m.Type)
	return SilentResult(msg)
}

// RecallMemoryTool searches the agent memory store for previously saved information.
type RecallMemoryTool struct {
	store   *memory.SQLiteAgentMemoryStore
	agentID string
}

// NewRecallMemoryTool creates a RecallMemoryTool bound to the given store and agent.
func NewRecallMemoryTool(store *memory.SQLiteAgentMemoryStore, agentID string) *RecallMemoryTool {
	return &RecallMemoryTool{store: store, agentID: agentID}
}

func (t *RecallMemoryTool) Name() string { return "recall_memory" }

func (t *RecallMemoryTool) Description() string {
	return "Search your persistent memory for previously saved information."
}

func (t *RecallMemoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Keywords to search for across memory content, descriptions, and tags.",
			},
			"type_filter": map[string]any{
				"type":        "string",
				"description": "Optional memory type to restrict results (user, project, feedback, reference, fact).",
				"enum":        []string{"user", "project", "feedback", "reference", "fact"},
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default 5).",
				"default":     5,
			},
		},
		"required": []string{"query"},
	}
}

func (t *RecallMemoryTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, ok := args["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return ErrorResult("recall_memory: 'query' is required and must be a non-empty string.")
	}

	limit := 5
	switch v := args["limit"].(type) {
	case int:
		limit = v
	case float64:
		limit = int(v)
	case int64:
		limit = int(v)
	}
	if limit <= 0 {
		limit = 5
	}

	typeFilter, _ := args["type_filter"].(string)

	memories, err := t.store.Search(ctx, t.agentID, query, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("recall_memory: search failed: %v", err)).WithError(err)
	}

	// Apply optional type filter in-process (search already uses SQL LIKE, type
	// filtering is cheap and avoids a more complex query builder).
	if typeFilter != "" {
		filtered := memories[:0]
		for _, m := range memories {
			if string(m.Type) == typeFilter {
				filtered = append(filtered, m)
			}
		}
		memories = filtered
	}

	if len(memories) == 0 {
		return SilentResult(fmt.Sprintf("recall_memory: no memories found for query %q.", query))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d memor", len(memories))
	if len(memories) == 1 {
		sb.WriteString("y")
	} else {
		sb.WriteString("ies")
	}
	fmt.Fprintf(&sb, " matching %q:\n\n", query)

	for i, m := range memories {
		fmt.Fprintf(&sb, "[%d] id=%s type=%s updated=%s\n",
			i+1, m.ID, m.Type, m.UpdatedAt.Format(time.RFC3339))
		if m.Description != "" {
			fmt.Fprintf(&sb, "    Summary: %s\n", m.Description)
		}
		if len(m.Tags) > 0 {
			fmt.Fprintf(&sb, "    Tags: %s\n", strings.Join(m.Tags, ", "))
		}
		fmt.Fprintf(&sb, "    Content: %s\n\n", m.Content)
	}

	return SilentResult(strings.TrimRight(sb.String(), "\n"))
}
