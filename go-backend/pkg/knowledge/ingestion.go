// OctAi - Knowledge Base Ingestion Pipeline
// Provides helpers for ingesting content from different sources into the store.
package knowledge

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// IngestMarkdown ingests a Markdown document into the knowledge store.
func IngestMarkdown(ctx context.Context, store KnowledgeStore, teamID, title, content, sourceURL string) (string, error) {
	doc := Document{
		ID:        newID(),
		TeamID:    teamID,
		Title:     title,
		Content:   content,
		SourceURL: sourceURL,
		Metadata: map[string]string{
			"type":       "markdown",
			"ingested":   time.Now().UTC().Format(time.RFC3339),
		},
	}
	return store.AddDocument(ctx, doc)
}

// IngestText ingests a plain-text note into the knowledge store.
func IngestText(ctx context.Context, store KnowledgeStore, teamID, title, content string) (string, error) {
	doc := Document{
		ID:     newID(),
		TeamID: teamID,
		Title:  title,
		Content: content,
		Metadata: map[string]string{
			"type":     "text",
			"ingested": time.Now().UTC().Format(time.RFC3339),
		},
	}
	return store.AddDocument(ctx, doc)
}

// IngestURL ingests content from a URL (content must be pre-fetched by the caller).
func IngestURL(ctx context.Context, store KnowledgeStore, teamID, url, content string) (string, error) {
	// Derive a title from the URL.
	title := url
	if idx := strings.LastIndex(url, "/"); idx >= 0 && idx < len(url)-1 {
		title = url[idx+1:]
	}
	title = strings.TrimSuffix(title, ".html")
	title = strings.TrimSuffix(title, ".htm")
	if title == "" {
		title = url
	}

	doc := Document{
		ID:        newID(),
		TeamID:    teamID,
		Title:     title,
		Content:   content,
		SourceURL: url,
		Metadata: map[string]string{
			"type":     "web",
			"url":      url,
			"ingested": time.Now().UTC().Format(time.RFC3339),
		},
	}
	return store.AddDocument(ctx, doc)
}

// FormatSearchResults formats search results into a prompt-friendly string.
func FormatSearchResults(results []SearchResult, maxChars int) string {
	if len(results) == 0 {
		return "No relevant knowledge base entries found."
	}

	var sb strings.Builder
	sb.WriteString("## Relevant knowledge base entries\n\n")

	remaining := maxChars
	for i, r := range results {
		entry := fmt.Sprintf("### [%d] %s\n%s\n\n", i+1, r.Title, r.Content)
		if remaining > 0 && len(entry) > remaining {
			entry = entry[:remaining] + "...\n\n"
		}
		sb.WriteString(entry)
		remaining -= len(entry)
		if remaining <= 0 {
			break
		}
	}
	return sb.String()
}
