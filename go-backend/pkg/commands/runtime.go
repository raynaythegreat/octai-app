package commands

import "github.com/raynaythegreat/octai-app/pkg/config"

// Runtime provides runtime dependencies to command handlers. It is constructed
// per-request by the agent loop so that per-request state (like session scope)
// can coexist with long-lived callbacks (like GetModelInfo).
type Runtime struct {
	Config             *config.Config
	GetModelInfo       func() (name, provider string)
	ListAgentIDs       func() []string
	ListDefinitions    func() []Definition
	ListSkillNames     func() []string
	GetEnabledChannels func() []string
	GetActiveTurn      func() any // Returning any to avoid circular dependency with agent package
	SwitchModel        func(value string) (oldModel string, err error)
	SwitchChannel      func(value string) error
	ClearHistory       func() error
	ReloadConfig       func() error
	// SetThinkingLevel changes the agent's thinking level for the current session.
	SetThinkingLevel func(level string) error
	// ToggleFastMode toggles fast mode on the current agent, returning the new state.
	ToggleFastMode func() (enabled bool, err error)
	// SearchMemory performs a keyword search over agent memory.
	SearchMemory func(query string) (string, error)
	// ListModels returns all configured models with their auth status.
	ListModels func() []ModelStatus
}

// ModelStatus holds display info for a single configured model.
type ModelStatus struct {
	Name      string
	ModelID   string
	Provider  string
	HasAPIKey bool
}
