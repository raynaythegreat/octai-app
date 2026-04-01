package commands

import (
	"context"
	"fmt"
	"strings"
)

func memorySearchCommand() Definition {
	return Definition{
		Name:        "memory",
		Description: "Search agent memory",
		Usage:       "/memory <query>",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SearchMemory == nil {
				return req.Reply(unavailableMsg)
			}
			// Extract everything after "/memory "
			text := strings.TrimSpace(req.Text)
			prefix := "/memory"
			if !strings.HasPrefix(text, prefix) {
				prefix = "!memory"
			}
			query := strings.TrimSpace(strings.TrimPrefix(text, prefix))
			if query == "" {
				return req.Reply("Usage: /memory <search query>")
			}
			result, err := rt.SearchMemory(query)
			if err != nil {
				return req.Reply(fmt.Sprintf("Memory search failed: %v", err))
			}
			if result == "" {
				return req.Reply(fmt.Sprintf("No memory entries found for: %s", query))
			}
			return req.Reply(result)
		},
	}
}
