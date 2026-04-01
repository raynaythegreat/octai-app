package commands

import (
	"context"
	"fmt"
)

func statusCommand() Definition {
	return Definition{
		Name:        "status",
		Description: "Show current agent settings",
		Usage:       "/status",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetModelInfo == nil {
				return req.Reply(unavailableMsg)
			}
			name, provider := rt.GetModelInfo()
			return req.Reply(fmt.Sprintf("Model: %s\nProvider: %s", name, provider))
		},
	}
}
