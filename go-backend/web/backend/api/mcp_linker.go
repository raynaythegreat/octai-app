package api

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/logger"
)

type mcpAutoEnableRule struct {
	mcpServer  string
	envVar     string
	envFileKey string
	checkCmd   string
}

var providerMCPRules = map[string][]mcpAutoEnableRule{
	"openai": {
		{mcpServer: "fetch"},
	},
	"anthropic": {
		{mcpServer: "sequential-thinking"},
		{mcpServer: "memory"},
	},
	"google-antigravity": {
		{mcpServer: "fetch"},
		{mcpServer: "puppeteer"},
	},
	"brave": {
		{mcpServer: "brave-search", envVar: "BRAVE_API_KEY", envFileKey: "BRAVE_API_KEY", checkCmd: "npx"},
	},
	"github": {
		{mcpServer: "github", envVar: "GITHUB_PERSONAL_ACCESS_TOKEN", envFileKey: "GITHUB_TOKEN", checkCmd: "npx"},
	},
}

func autoEnableMCPServers(configPath string, provider string, apiKey string) {
	rules, ok := providerMCPRules[provider]
	if !ok {
		return
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.ErrorC("mcp_linker", fmt.Sprintf("failed to load config for MCP auto-enable: %v", err))
		return
	}

	changed := false
	for _, rule := range rules {
		if cfg.Tools.MCP.Servers == nil {
			continue
		}
		serverCfg, exists := cfg.Tools.MCP.Servers[rule.mcpServer]
		if !exists || serverCfg.Enabled {
			continue
		}

		if rule.checkCmd != "" {
			if _, err := exec.LookPath(rule.checkCmd); err != nil {
				logger.DebugC("mcp_linker", fmt.Sprintf("skipping %s: %s not found in PATH", rule.mcpServer, rule.checkCmd))
				continue
			}
		}

		if rule.envFileKey != "" && apiKey != "" {
			workspacePath := cfg.WorkspacePath()
			if workspacePath != "" {
				envFilePath := ".env." + strings.TrimPrefix(rule.mcpServer, "@")
				if serverCfg.EnvFile != "" {
					envFilePath = serverCfg.EnvFile
				}
				if !strings.HasPrefix(envFilePath, ".") {
					envFilePath = "." + envFilePath
				}
				fullPath := workspacePath + "/" + envFilePath
				line := fmt.Sprintf("%s=%s\n", rule.envFileKey, apiKey)
				f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
				if err == nil {
					f.WriteString(line)
					f.Close()
				}
			}
		}

		serverCfg.Enabled = true
		cfg.Tools.MCP.Servers[rule.mcpServer] = serverCfg
		changed = true
		logger.InfoC("mcp_linker", fmt.Sprintf("auto-enabled MCP server: %s (triggered by %s)", rule.mcpServer, provider))
	}

	if changed {
		if err := config.SaveConfig(configPath, cfg); err != nil {
			logger.ErrorC("mcp_linker", fmt.Sprintf("failed to save config after MCP auto-enable: %v", err))
		}
	}
}

func triggerMCPLinker(model string, apiKey string) {
	provider := providerFromModel(model)
	if provider == "" {
		return
	}
	homePath := os.Getenv("OCTAI_HOME")
	if homePath == "" {
		home, _ := os.UserHomeDir()
		homePath = home + "/.octai"
	}
	configPath := homePath + "/config.json"
	autoEnableMCPServers(configPath, provider, apiKey)
}

func providerFromModel(model string) string {
	lower := strings.ToLower(model)
	if idx := strings.Index(lower, "/"); idx >= 0 {
		lower = lower[:idx]
	}
	switch lower {
	case "openai", "anthropic", "grok", "gemini", "deepseek", "groq", "mistral":
		return lower
	case "antigravity", "google-antigravity":
		return "google-antigravity"
	case "github-copilot":
		return "github"
	default:
		return ""
	}
}
