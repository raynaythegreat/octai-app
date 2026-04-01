// OctAi - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 OctAi contributors

package agent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// AutoModeDecision is the result of the classifier.
type AutoModeDecision int

const (
	AutoModeAllow  AutoModeDecision = iota // proceed automatically
	AutoModeBlock                          // block and ask agent to retry
	AutoModePrompt                         // ask the human for confirmation
)

// AutoModeSensitivity controls how aggressive the classifier is.
type AutoModeSensitivity string

const (
	AutoModeSensitivityStrict     AutoModeSensitivity = "strict"
	AutoModeSensitivityBalanced   AutoModeSensitivity = "balanced"
	AutoModeSensitivityPermissive AutoModeSensitivity = "permissive"
)

// AutoModeConfig holds configuration for the classifier.
type AutoModeConfig struct {
	Enabled         bool                `json:"enabled"`
	Sensitivity     AutoModeSensitivity `json:"sensitivity"`
	BlockedPatterns []string            `json:"blocked_patterns,omitempty"`
	AllowedPatterns []string            `json:"allowed_patterns,omitempty"`
}

// DefaultAutoModeConfig returns sensible defaults.
func DefaultAutoModeConfig() AutoModeConfig {
	return AutoModeConfig{
		Enabled:     false,
		Sensitivity: AutoModeSensitivityBalanced,
	}
}

// RiskCategory describes what kind of risk was detected.
type RiskCategory string

const (
	RiskMassFileDeletion   RiskCategory = "mass_file_deletion"
	RiskDataExfiltration   RiskCategory = "data_exfiltration"
	RiskMaliciousExec      RiskCategory = "malicious_exec"
	RiskCredentialAccess   RiskCategory = "credential_access"
	RiskSystemModification RiskCategory = "system_modification"
	RiskNetworkOperation   RiskCategory = "network_operation"
)

// riskLevel represents how severe a matched rule is.
type riskLevel int

const (
	riskLow    riskLevel = iota
	riskMedium riskLevel = iota
	riskHigh   riskLevel = iota
)

// ClassificationResult holds the classifier's decision and reasoning.
type ClassificationResult struct {
	Decision    AutoModeDecision
	Risk        RiskCategory
	Reason      string
	MatchedRule string
}

// ruleMatch is the result of evaluating a single rule.
type ruleMatch struct {
	risk     RiskCategory
	level    riskLevel
	reason   string
	ruleName string
}

// AutoModeClassifier evaluates tool calls for risk.
type AutoModeClassifier struct {
	cfg AutoModeConfig
}

// NewAutoModeClassifier creates a classifier with the given config.
func NewAutoModeClassifier(cfg AutoModeConfig) *AutoModeClassifier {
	return &AutoModeClassifier{cfg: cfg}
}

// Classify evaluates a tool call and returns a decision.
func (c AutoModeClassifier) Classify(toolName string, toolArgs map[string]any) ClassificationResult {
	combined := c.combinedArgs(toolName, toolArgs)

	// Custom allowed patterns take priority over everything.
	for _, pat := range c.cfg.AllowedPatterns {
		if matchesPattern(pat, combined) {
			return ClassificationResult{
				Decision:    AutoModeAllow,
				MatchedRule: fmt.Sprintf("allowed_pattern:%s", pat),
			}
		}
	}

	// Custom blocked patterns override built-in rules.
	for _, pat := range c.cfg.BlockedPatterns {
		if matchesPattern(pat, combined) {
			return ClassificationResult{
				Decision:    AutoModeBlock,
				Reason:      "matched custom blocked pattern",
				MatchedRule: fmt.Sprintf("blocked_pattern:%s", pat),
			}
		}
	}

	match, ok := c.evaluate(toolName, toolArgs)
	if !ok {
		return ClassificationResult{Decision: AutoModeAllow}
	}

	return c.decide(match)
}

// combinedArgs produces a single lowercase string from the tool name and all
// argument values, used for custom-pattern matching.
func (c AutoModeClassifier) combinedArgs(toolName string, toolArgs map[string]any) string {
	parts := []string{strings.ToLower(toolName)}
	for _, v := range toolArgs {
		parts = append(parts, strings.ToLower(fmt.Sprintf("%v", v)))
	}
	return strings.Join(parts, " ")
}

// decide converts a ruleMatch into a ClassificationResult based on sensitivity.
func (c AutoModeClassifier) decide(m ruleMatch) ClassificationResult {
	var decision AutoModeDecision
	switch c.cfg.Sensitivity {
	case AutoModeSensitivityStrict:
		switch m.level {
		case riskHigh:
			decision = AutoModeBlock
		case riskMedium:
			decision = AutoModeBlock
		default:
			decision = AutoModePrompt
		}
	case AutoModeSensitivityPermissive:
		switch m.level {
		case riskHigh:
			decision = AutoModePrompt
		default:
			decision = AutoModeAllow
		}
	default: // balanced
		switch m.level {
		case riskHigh:
			decision = AutoModeBlock
		case riskMedium:
			decision = AutoModePrompt
		default:
			decision = AutoModeAllow
		}
	}
	return ClassificationResult{
		Decision:    decision,
		Risk:        m.risk,
		Reason:      m.reason,
		MatchedRule: m.ruleName,
	}
}

// isShellTool reports whether the tool name is a shell/exec variant.
func isShellTool(toolName string) bool {
	lower := strings.ToLower(toolName)
	shellTools := []string{
		"bash", "shell_exec", "exec", "run_command",
		"computer", "terminal", "shell",
	}
	for _, t := range shellTools {
		if lower == t || strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

// isWriteTool reports whether the tool name is a file-write variant.
func isWriteTool(toolName string) bool {
	lower := strings.ToLower(toolName)
	writeTools := []string{
		"file_write", "write_file", "edit_file", "create_file",
		"file_edit", "write", "create",
	}
	for _, t := range writeTools {
		if lower == t || strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

// isReadTool reports whether the tool name is a file-read variant.
func isReadTool(toolName string) bool {
	lower := strings.ToLower(toolName)
	readTools := []string{
		"read_file", "cat", "file_read", "read",
	}
	for _, t := range readTools {
		if lower == t || strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

// argsToString extracts all string values from toolArgs into one lowercase blob.
func argsToString(toolArgs map[string]any) string {
	raw, _ := json.Marshal(toolArgs)
	return strings.ToLower(string(raw))
}

// evaluate runs all rule categories and returns the first (worst) match.
func (c AutoModeClassifier) evaluate(toolName string, toolArgs map[string]any) (ruleMatch, bool) {
	lower := strings.ToLower(toolName)
	args := argsToString(toolArgs)

	if isShellTool(lower) {
		if m, ok := classifyShellHigh(args); ok {
			return m, true
		}
		if m, ok := classifyShellMedium(args); ok {
			return m, true
		}
		if m, ok := classifyShellNetwork(args); ok {
			return m, true
		}
	}

	if isWriteTool(lower) {
		if m, ok := classifyWriteHigh(args); ok {
			return m, true
		}
		if m, ok := classifyWriteMedium(args); ok {
			return m, true
		}
	}

	if isReadTool(lower) {
		if m, ok := classifyReadCredential(args); ok {
			return m, true
		}
	}

	return ruleMatch{}, false
}

// matchesPattern does a simple case-insensitive substring or glob-style match.
func matchesPattern(pattern, target string) bool {
	pattern = strings.ToLower(pattern)
	target = strings.ToLower(target)
	if strings.Contains(pattern, "*") {
		// Treat * as a wildcard: split on * and check each part appears in order.
		parts := strings.Split(pattern, "*")
		idx := 0
		for _, part := range parts {
			if part == "" {
				continue
			}
			pos := strings.Index(target[idx:], part)
			if pos < 0 {
				return false
			}
			idx += pos + len(part)
		}
		return true
	}
	return strings.Contains(target, pattern)
}

// reMatch is a helper for case-insensitive regexp matching against args.
func reMatch(pattern, s string) bool {
	matched, _ := regexp.MatchString("(?i)"+pattern, s)
	return matched
}

// --- Shell HIGH risk rules ---

func classifyShellHigh(args string) (ruleMatch, bool) {
	rules := []struct {
		pattern  string
		ruleName string
		reason   string
		risk     RiskCategory
	}{
		// rm -rf targeting root-like paths or wildcards
		{`rm\s+-[^\s]*r[^\s]*f[^\s]*\s+(\/|~|\/home|\/etc|\/usr|\/var|\/tmp|\/opt|\*)`, "rm_rf_root", "rm -rf targeting a root-like or broad path", RiskMassFileDeletion},
		{`rm\s+-[^\s]*f[^\s]*r[^\s]*\s+(\/|~|\/home|\/etc|\/usr|\/var|\/tmp|\/opt|\*)`, "rm_rf_root_reversed_flags", "rm -rf targeting a root-like or broad path", RiskMassFileDeletion},
		// rm -rf . or rm -r . (dot at root-like context)
		{`rm\s+-[^\s]*r[^\s]*\s+\.`, "rm_rf_dot", "rm -r targeting current directory", RiskMassFileDeletion},
		// find . -delete or find / -delete
		{`find\s+(\/|\.)\s+.*-delete`, "find_delete", "find -delete on root or current dir", RiskMassFileDeletion},
		// write to /etc/ via redirection
		{`>\s*\/etc\/`, "redirect_etc", "redirecting output to /etc/", RiskSystemModification},
		// chmod 777 /etc or broad system chmod
		{`chmod\s+[0-9]*7[0-9]*\s+\/etc`, "chmod_etc", "broad chmod on /etc", RiskSystemModification},
		{`chmod\s+777\s+\/`, "chmod_777_root", "chmod 777 on root path", RiskSystemModification},
		// sudo rm patterns
		{`sudo\s+rm\s+`, "sudo_rm", "sudo rm command", RiskMassFileDeletion},
		// curl/wget piped to shell
		{`(curl|wget)\s+.*\|\s*(bash|sh)`, "pipe_to_shell", "remote code execution via curl/wget pipe", RiskMaliciousExec},
		// python -c with os.remove
		{`python[23]?\s+-c\s+.*os\.remove`, "python_os_remove", "Python one-liner calling os.remove", RiskMassFileDeletion},
		// dd writing to block device
		{`dd\s+.*if=.*\s+of=\/dev\/(sd|nvme|hd)`, "dd_block_device", "dd writing to raw block device", RiskMassFileDeletion},
		// base64 decode pipe to shell
		{`echo\s+.*\|\s*base64\s+-d\s*\|\s*(bash|sh)`, "base64_pipe_shell", "hex/base64 encoded payload piped to shell", RiskMaliciousExec},
		// fork bomb
		{`:\(\)\s*\{`, "fork_bomb", "fork bomb pattern detected", RiskMaliciousExec},
		// Database destructive operations without WHERE
		{`drop\s+(table|database)`, "db_drop", "DROP TABLE or DROP DATABASE", RiskMassFileDeletion},
		{`delete\s+from\s+\w+\s*(;|$|")`, "db_delete_no_where", "DELETE FROM without WHERE clause", RiskMassFileDeletion},
	}

	for _, r := range rules {
		if reMatch(r.pattern, args) {
			return ruleMatch{
				risk:     r.risk,
				level:    riskHigh,
				reason:   r.reason,
				ruleName: r.ruleName,
			}, true
		}
	}
	return ruleMatch{}, false
}

// --- Shell MEDIUM risk rules ---

func classifyShellMedium(args string) (ruleMatch, bool) {
	rules := []struct {
		pattern  string
		ruleName string
		reason   string
		risk     RiskCategory
	}{
		// Reading /etc/passwd or /etc/shadow
		{`cat\s+\/etc\/(passwd|shadow)`, "cat_passwd_shadow", "reading /etc/passwd or /etc/shadow", RiskCredentialAccess},
		// env/printenv grepping for sensitive data
		{`(env|printenv)\s*\|?\s*grep\s+-i`, "env_grep_sensitive", "grepping env for sensitive keys", RiskCredentialAccess},
		// Reading .env or key files
		{`cat\s+.*\.(env|pem|key|p12)`, "cat_env_key", "reading .env or key/cert files", RiskCredentialAccess},
		{`cat\s+.*(id_rsa|id_ed25519)`, "cat_ssh_key", "reading SSH private key", RiskCredentialAccess},
		{`cat\s+.*\.env(\.local|\.production)?`, "cat_env_variants", "reading .env file variant", RiskCredentialAccess},
		// git push --force
		{`git\s+push\s+.*--force`, "git_push_force", "force push to git remote", RiskSystemModification},
		// git reset --hard with remote
		{`git\s+reset\s+--hard\s+\w+\/`, "git_reset_hard_remote", "git reset --hard with remote ref", RiskSystemModification},
		// kill -9 PID 1 or kill -9 $(pgrep...)
		{`kill\s+-9\s+(1\b|\$\()`, "kill_9_pid1_or_pgrep", "kill -9 targeting PID 1 or pgrep output", RiskMaliciousExec},
		// pkill -9 all or critical daemon
		{`pkill\s+-9`, "pkill_9", "pkill -9 matching processes", RiskMaliciousExec},
		// Sending data to external host
		{`aws\s+s3\s+sync\s+.*s3:\/\/`, "aws_s3_sync_upload", "aws s3 sync uploading data", RiskDataExfiltration},
		{`rsync\s+-r?\s+.*@`, "rsync_external", "rsync to external host", RiskDataExfiltration},
		{`(scp|sftp)\s+.*(:\/)`, "scp_sftp_external", "scp/sftp sending files externally", RiskDataExfiltration},
	}

	for _, r := range rules {
		if reMatch(r.pattern, args) {
			return ruleMatch{
				risk:     r.risk,
				level:    riskMedium,
				reason:   r.reason,
				ruleName: r.ruleName,
			}, true
		}
	}
	return ruleMatch{}, false
}

// --- Shell NETWORK risk rules (medium level) ---

func classifyShellNetwork(args string) (ruleMatch, bool) {
	rules := []struct {
		pattern  string
		ruleName string
		reason   string
	}{
		{`curl\s+-X\s+POST\s+`, "curl_post_external", "curl POST sending data externally"},
		{`nc\s+-e\s+`, "netcat_exec", "netcat execute mode"},
		{`nmap\s+`, "nmap_scan", "nmap network scanning"},
	}

	for _, r := range rules {
		if reMatch(r.pattern, args) {
			return ruleMatch{
				risk:     RiskNetworkOperation,
				level:    riskMedium,
				reason:   r.reason,
				ruleName: r.ruleName,
			}, true
		}
	}
	return ruleMatch{}, false
}

// --- File write HIGH risk rules ---

func classifyWriteHigh(args string) (ruleMatch, bool) {
	rules := []struct {
		pattern  string
		ruleName string
		reason   string
		risk     RiskCategory
	}{
		{`(\/etc\/|\/usr\/bin\/|\/usr\/local\/bin\/|\/boot\/)`, "write_system_path", "writing to a protected system path", RiskSystemModification},
		{`\~\/(\.bashrc|\.zshrc|\.profile|\.bash_profile)`, "write_shell_profile", "writing to shell profile/rc file", RiskSystemModification},
		{`\/home\/[^\/]+\/(\.bashrc|\.zshrc|\.profile|\.bash_profile)`, "write_shell_profile_abs", "writing to shell profile/rc file", RiskSystemModification},
		{`\~\/\.config\/systemd\/`, "write_systemd_user", "writing to user systemd config", RiskSystemModification},
		{`\/home\/[^\/]+\/\.config\/systemd\/`, "write_systemd_user_abs", "writing to user systemd config", RiskSystemModification},
	}

	for _, r := range rules {
		if reMatch(r.pattern, args) {
			return ruleMatch{
				risk:     r.risk,
				level:    riskHigh,
				reason:   r.reason,
				ruleName: r.ruleName,
			}, true
		}
	}
	return ruleMatch{}, false
}

// --- File write MEDIUM risk rules ---

func classifyWriteMedium(args string) (ruleMatch, bool) {
	rules := []struct {
		pattern  string
		ruleName string
		reason   string
		risk     RiskCategory
	}{
		// Overwriting with empty content — detect content key with empty/blank value
		{`"content"\s*:\s*""`, "write_empty_content", "overwriting file with empty content (data loss)", RiskMassFileDeletion},
		{`"data"\s*:\s*""`, "write_empty_data", "overwriting file with empty data (data loss)", RiskMassFileDeletion},
		// Writing to SSH authorized_keys or config
		{`\.ssh\/(authorized_keys|config)`, "write_ssh_config", "writing to SSH authorized_keys or config", RiskCredentialAccess},
	}

	for _, r := range rules {
		if reMatch(r.pattern, args) {
			return ruleMatch{
				risk:     r.risk,
				level:    riskMedium,
				reason:   r.reason,
				ruleName: r.ruleName,
			}, true
		}
	}
	return ruleMatch{}, false
}

// --- File read CREDENTIAL risk rules ---

func classifyReadCredential(args string) (ruleMatch, bool) {
	rules := []struct {
		pattern  string
		ruleName string
		reason   string
	}{
		{`\.ssh\/(id_rsa|id_ed25519)`, "read_ssh_private_key", "reading SSH private key file"},
		{`\.(env|env\.local|env\.production)`, "read_env_file", "reading .env or production env file"},
		{`\.(key|pem|p12)`, "read_key_cert", "reading key or certificate file"},
		{`(secret|password|credential)`, "read_sensitive_path", "reading file with sensitive name in path"},
	}

	for _, r := range rules {
		if reMatch(r.pattern, args) {
			return ruleMatch{
				risk:     RiskCredentialAccess,
				level:    riskMedium,
				reason:   r.reason,
				ruleName: r.ruleName,
			}, true
		}
	}
	return ruleMatch{}, false
}
