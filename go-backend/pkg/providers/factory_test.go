package providers

import (
	"fmt"
	"testing"

	"github.com/raynaythegreat/octai-app/pkg/auth"
	"github.com/raynaythegreat/octai-app/pkg/config"
)

func TestCreateProviderReturnsHTTPProviderForOpenRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-openrouter"
	modelCfg := &config.ModelConfig{
		ModelName: "test-openrouter",
		Model:     "openrouter/auto",
		APIBase:   "https://openrouter.ai/api/v1",
	}
	modelCfg.SetAPIKey("sk-or-test")
	cfg.ModelList = []*config.ModelConfig{modelCfg}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*HTTPProvider); !ok {
		t.Fatalf("provider type = %T, want *HTTPProvider", provider)
	}
}

func TestCreateProviderReturnsCodexCliProviderForCodexCode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-codex"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-codex",
			Model:     "codex-cli/codex-model",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexCliProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeCliProviderForClaudeCli(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-claude-cli"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-claude-cli",
			Model:     "claude-cli/claude-sonnet",
			Workspace: "/tmp/workspace",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeCliProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeCliProvider", provider)
	}
}

func TestCreateProviderReturnsClaudeProviderForAnthropicOAuth(t *testing.T) {
	originalGetCredential := getCredential
	originalSetCredential := setCredential
	originalAnthropicOAuthConfig := anthropicOAuthConfig
	originalRefreshAccessToken := refreshAccessToken
	t.Cleanup(func() {
		getCredential = originalGetCredential
		setCredential = originalSetCredential
		anthropicOAuthConfig = originalAnthropicOAuthConfig
		refreshAccessToken = originalRefreshAccessToken
	})

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "anthropic" {
			t.Fatalf("provider = %q, want anthropic", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "anthropic-token",
		}, nil
	}
	setCredential = func(string, *auth.AuthCredential) error { return nil }
	anthropicOAuthConfig = func() (auth.OAuthProviderConfig, error) {
		return auth.OAuthProviderConfig{}, fmt.Errorf("not configured")
	}
	refreshAccessToken = func(*auth.AuthCredential, auth.OAuthProviderConfig) (*auth.AuthCredential, error) {
		return nil, fmt.Errorf("no refresh token")
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-claude-oauth"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "test-claude-oauth",
			Model:      "anthropic/claude-sonnet-4.6",
			AuthMethod: "oauth",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*ClaudeProvider); !ok {
		t.Fatalf("provider type = %T, want *ClaudeProvider", provider)
	}
	// TODO: Test custom APIBase when createClaudeAuthProvider supports it
}

func TestCreateProviderReturnsCodexProviderForOpenAIOAuth(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() {
		getCredential = originalGetCredential
	})

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "openai" {
			t.Fatalf("provider = %q, want openai", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "openai-token",
			AccountID:   "acct_123",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-openai-oauth"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName:  "test-openai-oauth",
			Model:      "openai/gpt-5.4",
			AuthMethod: "oauth",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexProvider", provider)
	}
}

func TestCreateProviderReturnsCodexProviderForImplicitOpenAICredential(t *testing.T) {
	originalGetCredential := getCredential
	t.Cleanup(func() {
		getCredential = originalGetCredential
	})

	getCredential = func(provider string) (*auth.AuthCredential, error) {
		if provider != "openai" {
			t.Fatalf("provider = %q, want openai", provider)
		}
		return &auth.AuthCredential{
			AccessToken: "openai-token",
			AccountID:   "acct_123",
		}, nil
	}

	cfg := config.DefaultConfig()
	cfg.Agents.Defaults.ModelName = "test-openai-implicit"
	cfg.ModelList = []*config.ModelConfig{
		{
			ModelName: "test-openai-implicit",
			Model:     "openai/gpt-5.4",
			APIBase:   "https://api.openai.com/v1",
		},
	}

	provider, _, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("CreateProvider() error = %v", err)
	}

	if _, ok := provider.(*CodexProvider); !ok {
		t.Fatalf("provider type = %T, want *CodexProvider", provider)
	}
}
