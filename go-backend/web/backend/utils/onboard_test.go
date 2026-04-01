package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raynaythegreat/octai-app/pkg/config"
)

func TestEnsureOnboardedSkipsWhenConfigExists(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	called := false
	execCommand = func(name string, args ...string) *exec.Cmd {
		called = true
		return exec.Command("sh", "-c", "exit 1")
	}

	if err := EnsureOnboarded(configPath); err != nil {
		t.Fatalf("EnsureOnboarded() error = %v", err)
	}
	if called {
		t.Fatal("expected onboard command not to run when config already exists")
	}
}

func TestEnsureOnboardedRunsOnboardWhenConfigMissing(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("EXPECTED_CONFIG_PATH", configPath)

	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	var gotName string
	var gotArgs []string
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return exec.Command(
			"sh",
			"-c",
			`test "$OCTAI_CONFIG" = "$EXPECTED_CONFIG_PATH" &&
mkdir -p "$(dirname "$OCTAI_CONFIG")" &&
printf '{}' > "$OCTAI_CONFIG"`,
		)
	}

	if err := EnsureOnboarded(configPath); err != nil {
		t.Fatalf("EnsureOnboarded() error = %v", err)
	}
	if gotName == "" {
		t.Fatal("expected onboard command to run")
	}
	if len(gotArgs) != 1 || gotArgs[0] != "onboard" {
		t.Fatalf("command args = %#v, want []string{\"onboard\"}", gotArgs)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config to be created: %v", err)
	}
}

func TestEnsureOnboardedFailsWhenOnboardDoesNotCreateConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "exit 0")
	}

	if err := EnsureOnboarded(configPath); err == nil {
		t.Fatal("EnsureOnboarded() error = nil, want failure when onboard does not create config")
	}
}

func TestEnsureOnboardedIncludesOnboardOutputOnFailure(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")

	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo onboarding failed >&2; exit 2")
	}

	err := EnsureOnboarded(configPath)
	if err == nil {
		t.Fatal("EnsureOnboarded() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "onboarding failed") {
		t.Fatalf("error = %q, want onboard output included", err)
	}
}

func TestImportLegacyConfigIfNeededCopiesLegacyModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	currentPath := filepath.Join(home, ".octai", "config.json")
	if err := os.MkdirAll(filepath.Dir(currentPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		currentPath,
		[]byte(`{"version":1,"model_list":[],"image_model_list":[],"video_model_list":[]}`),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile(current) error = %v", err)
	}

	legacyPath := filepath.Join(home, ".picoclaw", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	legacy := config.DefaultConfig()
	legacy.ModelList = []*config.ModelConfig{
		{
			ModelName: "llama3",
			Model:     "ollama/llama3",
			APIBase:   "http://localhost:11434/v1",
		},
	}
	legacy.ImageModelList = []*config.ModelConfig{}
	legacy.VideoModelList = []*config.ModelConfig{}
	legacy.Agents.Defaults.Provider = "ollama"
	legacy.Agents.Defaults.ModelName = "llama3"
	legacy.Agents.Defaults.ModelFallbacks = []string{"llama3.1"}
	legacy.Agents.Defaults.Routing = &config.RoutingConfig{
		Enabled:    true,
		LightModel: "llama3",
		Threshold:  0.2,
	}
	if err := config.SaveConfig(legacyPath, legacy); err != nil {
		t.Fatalf("SaveConfig(legacy) error = %v", err)
	}

	if err := ImportLegacyConfigIfNeeded(currentPath); err != nil {
		t.Fatalf("ImportLegacyConfigIfNeeded() error = %v", err)
	}

	imported, err := config.LoadConfig(currentPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(imported.ModelList) != 1 {
		t.Fatalf("len(ModelList) = %d, want 1", len(imported.ModelList))
	}
	if imported.ModelList[0].ModelName != "llama3" {
		t.Fatalf("ModelList[0].ModelName = %q, want %q", imported.ModelList[0].ModelName, "llama3")
	}
	if imported.ModelList[0].APIBase != "http://localhost:11434/v1" {
		t.Fatalf("ModelList[0].APIBase = %q, want %q", imported.ModelList[0].APIBase, "http://localhost:11434/v1")
	}
	if imported.Agents.Defaults.ModelName != "llama3" {
		t.Fatalf("Defaults.ModelName = %q, want %q", imported.Agents.Defaults.ModelName, "llama3")
	}
	if imported.Agents.Defaults.Provider != "ollama" {
		t.Fatalf("Defaults.Provider = %q, want %q", imported.Agents.Defaults.Provider, "ollama")
	}
	if imported.Agents.Defaults.Routing == nil || imported.Agents.Defaults.Routing.LightModel != "llama3" {
		t.Fatalf("Defaults.Routing = %#v, want LightModel %q", imported.Agents.Defaults.Routing, "llama3")
	}
}

func TestImportLegacyConfigIfNeededSkipsWhenTargetAlreadyHasModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	currentPath := filepath.Join(home, ".octai", "config.json")
	if err := os.MkdirAll(filepath.Dir(currentPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(
		currentPath,
		[]byte(`{"version":1,"model_list":[{"model_name":"current-model","model":"ollama/current-model","api_base":"http://localhost:11434/v1"}],"image_model_list":[],"video_model_list":[],"agents":{"defaults":{"model_name":"current-model"}}}`),
		0o600,
	); err != nil {
		t.Fatalf("WriteFile(current) error = %v", err)
	}

	legacyPath := filepath.Join(home, ".picoclaw", "config.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	legacy := config.DefaultConfig()
	legacy.ModelList = []*config.ModelConfig{
		{
			ModelName: "legacy-model",
			Model:     "ollama/legacy-model",
			APIBase:   "http://localhost:11434/v1",
		},
	}
	legacy.Agents.Defaults.ModelName = "legacy-model"
	if err := config.SaveConfig(legacyPath, legacy); err != nil {
		t.Fatalf("SaveConfig(legacy) error = %v", err)
	}

	if err := ImportLegacyConfigIfNeeded(currentPath); err != nil {
		t.Fatalf("ImportLegacyConfigIfNeeded() error = %v", err)
	}

	imported, err := config.LoadConfig(currentPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if len(imported.ModelList) != 1 {
		t.Fatalf("len(ModelList) = %d, want 1", len(imported.ModelList))
	}
	if imported.ModelList[0].ModelName != "current-model" {
		t.Fatalf("ModelList[0].ModelName = %q, want %q", imported.ModelList[0].ModelName, "current-model")
	}
	if imported.Agents.Defaults.ModelName != "current-model" {
		t.Fatalf("Defaults.ModelName = %q, want %q", imported.Agents.Defaults.ModelName, "current-model")
	}
}
