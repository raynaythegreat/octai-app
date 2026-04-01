package commands

import (
	"context"
	"fmt"
)

func fastCommand() Definition {
	return Definition{
		Name:        "fast",
		Description: "Toggle fast mode (provider-specific optimizations)",
		Usage:       "/fast",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.ToggleFastMode == nil {
				return req.Reply(unavailableMsg)
			}
			enabled, err := rt.ToggleFastMode()
			if err != nil {
				return req.Reply(fmt.Sprintf("Failed to toggle fast mode: %v", err))
			}
			state := "off"
			if enabled {
				state = "on"
			}
			return req.Reply(fmt.Sprintf("Fast mode: %s", state))
		},
	}
}
