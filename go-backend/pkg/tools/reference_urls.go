package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// referenceURLEntry mirrors the JSON structure saved by the web API.
type referenceURLEntry struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Notes       string   `json:"notes,omitempty"`
	AddedAt     string   `json:"added_at"`
}

// ReferenceURLSearchTool lets agents search the saved reference URL library.
type ReferenceURLSearchTool struct {
	dataPath string
}

// NewReferenceURLSearchTool creates a tool that searches reference_urls.json.
// dataPath should be the workspace directory; the tool appends "reference_urls.json".
func NewReferenceURLSearchTool(workspaceDir string) *ReferenceURLSearchTool {
	return &ReferenceURLSearchTool{
		dataPath: filepath.Join(workspaceDir, "reference_urls.json"),
	}
}

func (t *ReferenceURLSearchTool) Name() string { return "search_references" }

func (t *ReferenceURLSearchTool) Description() string {
	return "Search the saved reference URL library for relevant links, docs, APIs, or tools. " +
		"Returns matching references with URLs, descriptions, and categories. " +
		"Use this when you need relevant documentation, API references, tutorials, or tool links for a task."
}

func (t *ReferenceURLSearchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query — keywords, topic, or URL fragment to match against saved references.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional: filter by category (documentation, api-reference, tutorial, tool, library, blog, example, specification, dataset, video, other).",
			},
		},
		"required": []string{"query"},
	}
}

func (t *ReferenceURLSearchTool) Execute(_ context.Context, args map[string]any) *ToolResult {
	query, _ := args["query"].(string)
	if query == "" {
		return ErrorResult("query parameter is required").WithError(fmt.Errorf("query is required"))
	}
	filterCategory, _ := args["category"].(string)
	queryLower := strings.ToLower(query)

	data, err := os.ReadFile(t.dataPath)
	if os.IsNotExist(err) {
		return &ToolResult{
			ForLLM:  "No reference URLs have been saved yet. Ask the user to add references via the Reference URLs page in the OctAi web dashboard.",
			ForUser: "",
			Silent:  true,
		}
	}
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to read reference URLs: %v", err)).WithError(err)
	}

	var entries []referenceURLEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return ErrorResult(fmt.Sprintf("Failed to parse reference URLs: %v", err)).WithError(err)
	}

	// Score and filter entries
	type scored struct {
		entry referenceURLEntry
		score int
	}
	var results []scored

	for _, e := range entries {
		if filterCategory != "" && e.Category != filterCategory {
			continue
		}

		score := 0
		title := strings.ToLower(e.Title)
		desc := strings.ToLower(e.Description)
		url := strings.ToLower(e.URL)
		notes := strings.ToLower(e.Notes)

		// Exact matches score highest
		if strings.Contains(title, queryLower) {
			score += 10
		}
		if strings.Contains(desc, queryLower) {
			score += 5
		}
		if strings.Contains(url, queryLower) {
			score += 4
		}
		if strings.Contains(notes, queryLower) {
			score += 3
		}
		for _, tag := range e.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				score += 6
			}
		}
		// Category match
		if strings.Contains(strings.ToLower(e.Category), queryLower) {
			score += 2
		}
		// Word-by-word matching
		for _, word := range strings.Fields(queryLower) {
			if len(word) < 3 {
				continue
			}
			if strings.Contains(title, word) {
				score += 3
			}
			if strings.Contains(desc, word) {
				score += 2
			}
			for _, tag := range e.Tags {
				if strings.Contains(strings.ToLower(tag), word) {
					score += 2
				}
			}
		}

		if score > 0 {
			results = append(results, scored{entry: e, score: score})
		}
	}

	if len(results) == 0 {
		msg := fmt.Sprintf("No reference URLs found matching '%s'", query)
		if filterCategory != "" {
			msg += " in category '" + filterCategory + "'"
		}
		return &ToolResult{
			ForLLM:  msg + ". The library may not have relevant entries yet.",
			ForUser: "",
			Silent:  true,
		}
	}

	// Sort by score descending (simple insertion)
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	// Cap at 8 results
	if len(results) > 8 {
		results = results[:8]
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d reference(s) matching '%s':\n\n", len(results), query))
	for i, r := range results {
		e := r.entry
		sb.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, e.Title))
		sb.WriteString(fmt.Sprintf("   URL: %s\n", e.URL))
		sb.WriteString(fmt.Sprintf("   Category: %s\n", e.Category))
		if e.Description != "" {
			sb.WriteString(fmt.Sprintf("   Description: %s\n", e.Description))
		}
		if len(e.Tags) > 0 {
			sb.WriteString(fmt.Sprintf("   Tags: %s\n", strings.Join(e.Tags, ", ")))
		}
		if e.Notes != "" {
			sb.WriteString(fmt.Sprintf("   Notes: %s\n", e.Notes))
		}
		sb.WriteString("\n")
	}

	return &ToolResult{
		ForLLM:  sb.String(),
		ForUser: fmt.Sprintf("Found %d reference URL(s) for '%s'.", len(results), query),
		Silent:  false,
	}
}
