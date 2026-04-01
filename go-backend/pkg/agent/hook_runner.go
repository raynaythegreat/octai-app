package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/logger"
)

// HookEvent defines when a hook fires.
type HookEvent string

const (
	HookEventPreToolUse  HookEvent = "pre_tool_use"
	HookEventPostToolUse HookEvent = "post_tool_use"
	HookEventPreLLMCall  HookEvent = "pre_llm_call"
	HookEventPostLLMCall HookEvent = "post_llm_call"
	HookEventAgentStart  HookEvent = "agent_start"
	HookEventAgentEnd    HookEvent = "agent_end"
	HookEventOnError     HookEvent = "on_error"
)

// HookType defines the kind of hook handler.
type HookType string

const (
	HookTypeScript HookType = "script" // shell command
	HookTypeHTTP   HookType = "http"   // POST to URL
)

// RunnerHookConfig configures a single hook for the HookRunner.
type RunnerHookConfig struct {
	Event   HookEvent         `json:"event"`
	Type    HookType          `json:"type"`
	Command string            `json:"command,omitempty"` // for script hooks
	URL     string            `json:"url,omitempty"`     // for http hooks
	Timeout int               `json:"timeout,omitempty"` // seconds, default 30
	Match   map[string]string `json:"match,omitempty"`   // filter: {"tool": "bash"} etc
}

// RunnerAction is what the hook runner tells the agent loop to do.
type RunnerAction string

const (
	RunnerActionAllow  RunnerAction = "allow"  // proceed normally
	RunnerActionBlock  RunnerAction = "block"  // block the action
	RunnerActionModify RunnerAction = "modify" // use modified input
)

// HookPayload is the data sent to a hook handler.
type HookPayload struct {
	Event      HookEvent      `json:"event"`
	AgentID    string         `json:"agent_id,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`   // for tool hooks
	ToolArgs   map[string]any `json:"tool_args,omitempty"`
	ToolResult string         `json:"tool_result,omitempty"` // for post_tool_use
	Message    string         `json:"message,omitempty"`     // for LLM hooks
	Error      string         `json:"error,omitempty"`       // for error hooks
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// HookRunnerResult is what the hook returns.
type HookRunnerResult struct {
	Action       RunnerAction   `json:"action"`
	ModifiedArgs map[string]any `json:"modified_args,omitempty"` // for modify action
	Message      string         `json:"message,omitempty"`       // block reason or info
}

const defaultRunnerHookTimeout = 30 * time.Second

// HookRunner executes shell-script and HTTP webhook hooks.
type HookRunner struct {
	hooks []RunnerHookConfig
}

// NewHookRunner creates a HookRunner with the given hook configurations.
func NewHookRunner(hooks []RunnerHookConfig) *HookRunner {
	return &HookRunner{hooks: hooks}
}

// RunHooks fires all hooks matching the event and payload.
// It returns the first non-allow result; if no hook blocks or modifies, it returns allow.
// Hook execution errors are logged but do not block execution (fail open).
func (h *HookRunner) RunHooks(ctx context.Context, payload HookPayload) HookRunnerResult {
	for _, hook := range h.hooks {
		if hook.Event != payload.Event {
			continue
		}
		if !h.matchesFilter(hook, payload) {
			continue
		}

		result, err := h.runHook(ctx, hook, payload)
		if err != nil {
			logger.WarnCF("hook_runner", "Hook execution error (fail open)", map[string]any{
				"event":   string(payload.Event),
				"type":    string(hook.Type),
				"command": hook.Command,
				"url":     hook.URL,
				"error":   err.Error(),
			})
			continue
		}

		if result.Action != RunnerActionAllow && result.Action != "" {
			return result
		}
	}
	return HookRunnerResult{Action: RunnerActionAllow}
}

func (h *HookRunner) runHook(ctx context.Context, hook RunnerHookConfig, payload HookPayload) (HookRunnerResult, error) {
	switch hook.Type {
	case HookTypeScript:
		return h.runScript(ctx, hook, payload)
	case HookTypeHTTP:
		return h.runHTTP(ctx, hook, payload)
	default:
		return HookRunnerResult{Action: RunnerActionAllow}, fmt.Errorf("unknown hook type: %q", hook.Type)
	}
}

// matchesFilter checks if a hook's Match map applies to the payload.
// Supported keys: "tool" (comma-separated list of tool names), "agent" (agent ID).
// An empty/nil Match fires for all payloads.
func (h *HookRunner) matchesFilter(hook RunnerHookConfig, payload HookPayload) bool {
	if len(hook.Match) == 0 {
		return true
	}
	for key, val := range hook.Match {
		switch key {
		case "tool":
			if !matchesCommaSeparated(val, payload.ToolName) {
				return false
			}
		case "agent":
			if val != payload.AgentID {
				return false
			}
		}
	}
	return true
}

func matchesCommaSeparated(pattern, value string) bool {
	if pattern == "" {
		return true
	}
	for _, part := range strings.Split(pattern, ",") {
		if strings.TrimSpace(part) == value {
			return true
		}
	}
	return false
}

// runScript executes a shell script hook, passing the payload as JSON via stdin.
// The script can output a JSON HookRunnerResult to stdout, or just exit 0 (allow) / non-zero (block).
func (h *HookRunner) runScript(ctx context.Context, hook RunnerHookConfig, payload HookPayload) (HookRunnerResult, error) {
	timeout := hookTimeout(hook.Timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return HookRunnerResult{}, fmt.Errorf("marshal hook payload: %w", err)
	}

	cmd := exec.Command("sh", "-c", hook.Command) //nolint:gosec
	cmd.Stdin = bytes.NewReader(payloadJSON)
	// Put the child in its own process group so we can kill all descendants on timeout.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if startErr := cmd.Start(); startErr != nil {
		return HookRunnerResult{}, fmt.Errorf("start script hook: %w", startErr)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var runErr error
	select {
	case runErr = <-done:
		// Command finished normally.
	case <-ctx.Done():
		// Kill the entire process group.
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-done // wait for cmd.Wait() to return
		runErr = ctx.Err()
	}

	stdoutBytes := bytes.TrimSpace(stdout.Bytes())

	// Try to parse stdout as JSON HookRunnerResult.
	if len(stdoutBytes) > 0 {
		var result HookRunnerResult
		if jsonErr := json.Unmarshal(stdoutBytes, &result); jsonErr == nil {
			return result, nil
		}
	}

	// No valid JSON output — use exit code to determine action.
	if runErr != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		reason := stderrStr
		if reason == "" {
			reason = runErr.Error()
		}
		return HookRunnerResult{Action: RunnerActionBlock, Message: reason}, nil
	}

	return HookRunnerResult{Action: RunnerActionAllow}, nil
}

// runHTTP sends the payload as a JSON POST to the hook URL and reads the HookRunnerResult response.
// Fails open on HTTP errors (4xx/5xx) or network errors.
func (h *HookRunner) runHTTP(ctx context.Context, hook RunnerHookConfig, payload HookPayload) (HookRunnerResult, error) {
	timeout := hookTimeout(hook.Timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return HookRunnerResult{}, fmt.Errorf("marshal hook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook.URL, bytes.NewReader(payloadJSON))
	if err != nil {
		return HookRunnerResult{}, fmt.Errorf("create HTTP hook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// Fail open on network errors.
		logger.WarnCF("hook_runner", "HTTP hook request failed (fail open)", map[string]any{
			"url":   hook.URL,
			"error": err.Error(),
		})
		return HookRunnerResult{Action: RunnerActionAllow}, nil
	}
	defer func() { _ = resp.Body.Close() }()

	// Fail open on 4xx/5xx.
	if resp.StatusCode >= 400 {
		logger.WarnCF("hook_runner", "HTTP hook returned error status (fail open)", map[string]any{
			"url":    hook.URL,
			"status": resp.StatusCode,
		})
		return HookRunnerResult{Action: RunnerActionAllow}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return HookRunnerResult{Action: RunnerActionAllow}, nil
	}

	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return HookRunnerResult{Action: RunnerActionAllow}, nil
	}

	var result HookRunnerResult
	if err := json.Unmarshal(body, &result); err != nil {
		// Invalid JSON body — fail open.
		logger.WarnCF("hook_runner", "HTTP hook returned invalid JSON (fail open)", map[string]any{
			"url":   hook.URL,
			"error": err.Error(),
		})
		return HookRunnerResult{Action: RunnerActionAllow}, nil
	}

	return result, nil
}

func hookTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultRunnerHookTimeout
	}
	return time.Duration(seconds) * time.Second
}
