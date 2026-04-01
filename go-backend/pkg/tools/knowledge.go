// OctAi - Knowledge Base Tools
// Provides agents with access to the team's shared knowledge base.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/raynaythegreat/octai-app/pkg/knowledge"
)

// KnowledgeSearchTool searches the team knowledge base using BM25.
type KnowledgeSearchTool struct {
	store  knowledge.KnowledgeStore
	teamID string
}

// NewKnowledgeSearchTool creates a search tool for the given team's knowledge base.
func NewKnowledgeSearchTool(store knowledge.KnowledgeStore, teamID string) *KnowledgeSearchTool {
	return &KnowledgeSearchTool{store: store, teamID: teamID}
}

func (t *KnowledgeSearchTool) Name() string { return "knowledge_search" }

func (t *KnowledgeSearchTool) Description() string {
	return "Search the team's shared knowledge base for relevant information. " +
		"Returns the most relevant excerpts ranked by relevance. " +
		"Use this before web_search when the answer might already be in the team's knowledge base."
}

func (t *KnowledgeSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query — natural language or keywords.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 5, max: 20).",
			},
		},
		"required": []string{"query"},
	}
}

func (t *KnowledgeSearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	query, _ := args["query"].(string)
	if query == "" {
		return ErrorResult("query parameter is required").WithError(fmt.Errorf("query is required"))
	}

	limit := 5
	if l, ok := args["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 20 {
			limit = 20
		}
	}

	results, err := t.store.Search(ctx, t.teamID, query, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Knowledge search failed: %v", err)).WithError(err)
	}

	formatted := knowledge.FormatSearchResults(results, 4000)
	if len(results) == 0 {
		return &ToolResult{
			ForLLM:  "No results found in the knowledge base for: " + query,
			ForUser: "",
			Silent:  true,
		}
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Knowledge search results for '%s':\n\n%s", query, formatted),
		ForUser: fmt.Sprintf("Found %d knowledge base entries for '%s'.", len(results), query),
		Silent:  false,
	}
}

// KnowledgeAddTool ingests a new document into the team knowledge base.
type KnowledgeAddTool struct {
	store  knowledge.KnowledgeStore
	teamID string
}

// NewKnowledgeAddTool creates a tool for adding documents to the knowledge base.
func NewKnowledgeAddTool(store knowledge.KnowledgeStore, teamID string) *KnowledgeAddTool {
	return &KnowledgeAddTool{store: store, teamID: teamID}
}

func (t *KnowledgeAddTool) Name() string { return "knowledge_add" }

func (t *KnowledgeAddTool) Description() string {
	return "Add a document or note to the team's shared knowledge base. " +
		"Use this to preserve research findings, customer notes, or any information " +
		"the team may need in future conversations."
}

func (t *KnowledgeAddTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Short descriptive title for this knowledge entry.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to store. Can be Markdown, plain text, or structured notes.",
			},
			"source_url": map[string]any{
				"type":        "string",
				"description": "Optional source URL if the content was fetched from the web.",
			},
		},
		"required": []string{"title", "content"},
	}
}

func (t *KnowledgeAddTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	sourceURL, _ := args["source_url"].(string)

	if title == "" || content == "" {
		return ErrorResult("title and content are required").WithError(fmt.Errorf("title and content are required"))
	}

	docID, err := knowledge.IngestMarkdown(ctx, t.store, t.teamID, title, content, sourceURL)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to add to knowledge base: %v", err)).WithError(err)
	}

	return &ToolResult{
		ForLLM:  fmt.Sprintf("Document '%s' added to knowledge base with ID: %s", title, docID),
		ForUser: fmt.Sprintf("Added '%s' to the team knowledge base.", title),
		Silent:  false,
	}
}

// KnowledgeListTool lists documents in the team knowledge base.
type KnowledgeListTool struct {
	store  knowledge.KnowledgeStore
	teamID string
}

// NewKnowledgeListTool creates a tool for listing knowledge base documents.
func NewKnowledgeListTool(store knowledge.KnowledgeStore, teamID string) *KnowledgeListTool {
	return &KnowledgeListTool{store: store, teamID: teamID}
}

func (t *KnowledgeListTool) Name() string { return "knowledge_list" }

func (t *KnowledgeListTool) Description() string {
	return "List documents available in the team's shared knowledge base."
}

func (t *KnowledgeListTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *KnowledgeListTool) Execute(ctx context.Context, _ map[string]any) *ToolResult {
	docs, err := t.store.ListDocuments(ctx, t.teamID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to list knowledge base: %v", err)).WithError(err)
	}

	if len(docs) == 0 {
		return &ToolResult{
			ForLLM:  "Knowledge base is empty. Use knowledge_add to add documents.",
			ForUser: "Knowledge base is empty.",
			Silent:  false,
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Knowledge base contains %d documents:\n\n", len(docs)))
	for i, doc := range docs {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s", i+1, doc.ID, doc.Title))
		if doc.SourceURL != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", doc.SourceURL))
		}
		sb.WriteString(fmt.Sprintf(" — %d chars, updated %s\n",
			len(doc.Content),
			doc.UpdatedAt.Format("2006-01-02")))
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: fmt.Sprintf("Knowledge base: %d documents.", len(docs)),
		Silent:  false,
	}
}
