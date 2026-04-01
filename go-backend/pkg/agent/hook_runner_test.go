package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// --- Script hook tests ---

func TestHookRunner_Script_ValidJSONAllow(t *testing.T) {
	result := HookRunnerResult{Action: RunnerActionAllow, Message: "ok"}
	resultJSON, _ := json.Marshal(result)

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event:   HookEventPreToolUse,
			Type:    HookTypeScript,
			Command: "echo '" + string(resultJSON) + "'",
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})

	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow, got %q", got.Action)
	}
}

func TestHookRunner_Script_ValidJSONBlock(t *testing.T) {
	result := HookRunnerResult{Action: RunnerActionBlock, Message: "not allowed"}
	resultJSON, _ := json.Marshal(result)

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event:   HookEventPreToolUse,
			Type:    HookTypeScript,
			Command: "echo '" + string(resultJSON) + "'",
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})

	if got.Action != RunnerActionBlock {
		t.Fatalf("expected block, got %q", got.Action)
	}
	if got.Message != "not allowed" {
		t.Fatalf("expected message %q, got %q", "not allowed", got.Message)
	}
}

func TestHookRunner_Script_ExitCode1_Block(t *testing.T) {
	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event:   HookEventPreToolUse,
			Type:    HookTypeScript,
			Command: "echo 'bad thing happened' >&2; exit 1",
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})

	if got.Action != RunnerActionBlock {
		t.Fatalf("expected block on exit code 1, got %q", got.Action)
	}
	if got.Message == "" {
		t.Fatal("expected non-empty block message from stderr")
	}
}

func TestHookRunner_Script_ExitCode0_NoJSON_Allow(t *testing.T) {
	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event:   HookEventPreToolUse,
			Type:    HookTypeScript,
			Command: "true",
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})

	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow on exit 0 with no output, got %q", got.Action)
	}
}

// --- HTTP hook tests ---

func TestHookRunner_HTTP_Allow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json content-type, got %q", ct)
		}
		result := HookRunnerResult{Action: RunnerActionAllow}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPreLLMCall,
			Type:  HookTypeHTTP,
			URL:   srv.URL,
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreLLMCall,
	})

	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow, got %q", got.Action)
	}
}

func TestHookRunner_HTTP_Block(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := HookRunnerResult{Action: RunnerActionBlock, Message: "http blocked"}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPreToolUse,
			Type:  HookTypeHTTP,
			URL:   srv.URL,
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})

	if got.Action != RunnerActionBlock {
		t.Fatalf("expected block, got %q", got.Action)
	}
	if got.Message != "http blocked" {
		t.Fatalf("expected %q, got %q", "http blocked", got.Message)
	}
}

func TestHookRunner_HTTP_ErrorStatus_FailOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPreToolUse,
			Type:  HookTypeHTTP,
			URL:   srv.URL,
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})

	// Must fail open — 5xx does not block execution.
	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow (fail open) on 5xx, got %q", got.Action)
	}
}

func TestHookRunner_HTTP_EmptyBody_Allow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// No body.
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPostToolUse,
			Type:  HookTypeHTTP,
			URL:   srv.URL,
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPostToolUse,
	})

	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow on empty body, got %q", got.Action)
	}
}

// --- Filter matching tests ---

func TestHookRunner_Filter_ToolName_Match(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := HookRunnerResult{Action: RunnerActionBlock, Message: "blocked"}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPreToolUse,
			Type:  HookTypeHTTP,
			URL:   srv.URL,
			Match: map[string]string{"tool": "bash"},
		},
	})

	// Payload with matching tool — should be blocked.
	got := runner.RunHooks(context.Background(), HookPayload{
		Event:    HookEventPreToolUse,
		ToolName: "bash",
	})
	if got.Action != RunnerActionBlock {
		t.Fatalf("expected block for matching tool, got %q", got.Action)
	}

	// Payload with non-matching tool — hook should not fire, allow.
	got2 := runner.RunHooks(context.Background(), HookPayload{
		Event:    HookEventPreToolUse,
		ToolName: "file_read",
	})
	if got2.Action != RunnerActionAllow {
		t.Fatalf("expected allow for non-matching tool, got %q", got2.Action)
	}
}

func TestHookRunner_Filter_ToolName_CommaSeparated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := HookRunnerResult{Action: RunnerActionBlock, Message: "blocked"}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPreToolUse,
			Type:  HookTypeHTTP,
			URL:   srv.URL,
			Match: map[string]string{"tool": "bash,file_write"},
		},
	})

	for _, tool := range []string{"bash", "file_write"} {
		got := runner.RunHooks(context.Background(), HookPayload{
			Event:    HookEventPreToolUse,
			ToolName: tool,
		})
		if got.Action != RunnerActionBlock {
			t.Fatalf("expected block for tool %q, got %q", tool, got.Action)
		}
	}

	got := runner.RunHooks(context.Background(), HookPayload{
		Event:    HookEventPreToolUse,
		ToolName: "echo",
	})
	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow for unmatched tool, got %q", got.Action)
	}
}

func TestHookRunner_Filter_EmptyMatch_FiresForAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := HookRunnerResult{Action: RunnerActionBlock, Message: "always block"}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPreToolUse,
			Type:  HookTypeHTTP,
			URL:   srv.URL,
			// No Match — fires for all.
		},
	})

	for _, tool := range []string{"bash", "file_read", "anything"} {
		got := runner.RunHooks(context.Background(), HookPayload{
			Event:    HookEventPreToolUse,
			ToolName: tool,
		})
		if got.Action != RunnerActionBlock {
			t.Fatalf("expected block for tool %q with empty filter, got %q", tool, got.Action)
		}
	}
}

// --- Multiple hooks: first block wins ---

func TestHookRunner_MultipleHooks_FirstBlockWins(t *testing.T) {
	blockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := HookRunnerResult{Action: RunnerActionBlock, Message: "first block"}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer blockSrv.Close()

	secondCalled := false
	secondSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalled = true
		result := HookRunnerResult{Action: RunnerActionAllow}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer secondSrv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventPreToolUse,
			Type:  HookTypeHTTP,
			URL:   blockSrv.URL,
		},
		{
			Event: HookEventPreToolUse,
			Type:  HookTypeHTTP,
			URL:   secondSrv.URL,
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})

	if got.Action != RunnerActionBlock {
		t.Fatalf("expected block from first hook, got %q", got.Action)
	}
	if got.Message != "first block" {
		t.Fatalf("expected message %q, got %q", "first block", got.Message)
	}
	if secondCalled {
		t.Fatal("expected second hook to not be called after first block")
	}
}

// --- Timeout handling ---

func TestHookRunner_Script_Timeout(t *testing.T) {
	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event:   HookEventPreToolUse,
			Type:    HookTypeScript,
			Command: "sleep 10",
			Timeout: 1, // 1 second — will be killed
		},
	})

	start := time.Now()
	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})
	elapsed := time.Since(start)

	// Should complete well within 5 seconds due to timeout.
	if elapsed > 5*time.Second {
		t.Fatalf("hook took too long (%v), expected timeout to fire", elapsed)
	}
	// Timed-out script with non-zero exit and no valid JSON → block.
	if got.Action != RunnerActionBlock {
		t.Fatalf("expected block on timeout (non-zero exit), got %q", got.Action)
	}
}

func TestHookRunner_HTTP_Timeout(t *testing.T) {
	// Use a channel to unblock the slow handler when the test ends.
	unblock := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-unblock:
		case <-time.After(15 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(unblock)
		slow.Close()
	}()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event:   HookEventPreToolUse,
			Type:    HookTypeHTTP,
			URL:     slow.URL,
			Timeout: 1, // 1 second
		},
	})

	start := time.Now()
	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse,
	})
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Fatalf("HTTP hook took too long (%v), expected timeout", elapsed)
	}
	// Network error from timeout → fail open.
	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow (fail open) on HTTP timeout, got %q", got.Action)
	}
}

// --- Event filtering ---

func TestHookRunner_EventMismatch_DoesNotFire(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		result := HookRunnerResult{Action: RunnerActionBlock}
		_ = json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	runner := NewHookRunner([]RunnerHookConfig{
		{
			Event: HookEventAgentStart, // different event
			Type:  HookTypeHTTP,
			URL:   srv.URL,
		},
	})

	got := runner.RunHooks(context.Background(), HookPayload{
		Event: HookEventPreToolUse, // doesn't match
	})

	if called {
		t.Fatal("hook should not have fired for mismatched event")
	}
	if got.Action != RunnerActionAllow {
		t.Fatalf("expected allow when no hooks match, got %q", got.Action)
	}
}
