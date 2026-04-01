package commands

import (
	"context"
	"fmt"
)

func thinkCommand() Definition {
	return Definition{
		Name:        "think",
		Description: "Set thinking level for the current session",
		Usage:       "/think [off|low|medium|high|xhigh|adaptive]",
		Handler: func(_ context.Context, req Request, rt *Runtime) error {
			if rt == nil || rt.SetThinkingLevel == nil {
				return req.Reply(unavailableMsg)
			}
			level := nthToken(req.Text, 1)
			if level == "" {
				level = "adaptive"
			}
			validLevels := map[string]bool{
				"off": true, "low": true, "medium": true,
				"high": true, "xhigh": true, "adaptive": true,
			}
			if !validLevels[level] {
				return req.Reply(fmt.Sprintf(
					"Unknown thinking level %q. Valid values: off, low, medium, high, xhigh, adaptive",
					level,
				))
			}
			if err := rt.SetThinkingLevel(level); err != nil {
				return req.Reply(fmt.Sprintf("Failed to set thinking level: %v", err))
			}
			return req.Reply(fmt.Sprintf("Thinking level set to: %s", level))
		},
	}
}
