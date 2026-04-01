package commands

import (
	"context"
	"fmt"
	"strings"
)

func modelInfoCommand() Definition {
	return Definition{
		Name:        "model",
		Description: "Show or switch the current model",
		Usage:       "/model [name]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.GetModelInfo == nil {
				return req.Reply(unavailableMsg)
			}
			// /model with no args → show current model
			arg := nthToken(req.Text, 1)
			if arg == "" {
				name, provider := rt.GetModelInfo()
				return req.Reply(fmt.Sprintf("Current model: %s (provider: %s)", name, provider))
			}
			// /model <name> → switch model
			if rt.SwitchModel == nil {
				return req.Reply(unavailableMsg)
			}
			value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(req.Text), "/model"))
			value = strings.TrimSpace(value)
			if value == "" {
				name, provider := rt.GetModelInfo()
				return req.Reply(fmt.Sprintf("Current model: %s (provider: %s)", name, provider))
			}
			oldModel, err := rt.SwitchModel(value)
			if err != nil {
				return req.Reply(fmt.Sprintf("Failed to switch model: %v", err))
			}
			return req.Reply(fmt.Sprintf("Switched model from %s to %s", oldModel, value))
		},
	}
}
