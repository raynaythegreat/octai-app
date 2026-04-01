// OctAi - Integration Framework
// Defines the unified interface all integrations must implement.
// Each integration can provide tools that are automatically registered to agents by role.
package integrations

import (
	"context"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/tools"
)

// HealthStatus describes the current health of an integration.
type HealthStatus struct {
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// IntegrationConfig holds generic connection configuration for an integration.
// Each integration may interpret the fields it needs and ignore the rest.
type IntegrationConfig struct {
	// APIKey is the primary authentication credential.
	APIKey string `json:"api_key,omitempty"`
	// BaseURL overrides the default API endpoint.
	BaseURL string `json:"base_url,omitempty"`
	// ExtraParams holds integration-specific key-value configuration.
	ExtraParams map[string]string `json:"extra_params,omitempty"`
}

// Integration is the common interface for all external service integrations.
// Each integration provides a set of Tools that agents can use.
type Integration interface {
	// Name returns the integration's unique identifier (e.g. "hubspot", "linear").
	Name() string
	// Category describes the integration domain (e.g. "crm", "project_management").
	Category() string
	// Connect establishes the integration's connection using the provided config.
	Connect(ctx context.Context, cfg IntegrationConfig) error
	// Disconnect gracefully tears down the integration's connection.
	Disconnect(ctx context.Context) error
	// Health checks whether the integration is reachable.
	Health(ctx context.Context) HealthStatus
	// Tools returns the set of agent tools this integration provides.
	// These tools are registered to agents with the matching roles.
	Tools() []tools.Tool
	// ForRoles returns the agent roles this integration's tools should be registered to.
	// Empty slice means register for all roles.
	ForRoles() []string
}

// BaseIntegration provides default no-op implementations for optional methods.
// Embed this in concrete integrations to satisfy the full interface easily.
type BaseIntegration struct {
	name     string
	category string
	roles    []string
}

func NewBase(name, category string, roles ...string) BaseIntegration {
	return BaseIntegration{name: name, category: category, roles: roles}
}

func (b BaseIntegration) Name() string      { return b.name }
func (b BaseIntegration) Category() string  { return b.category }
func (b BaseIntegration) ForRoles() []string { return b.roles }

func (b BaseIntegration) Disconnect(_ context.Context) error { return nil }

func (b BaseIntegration) Health(_ context.Context) HealthStatus {
	return HealthStatus{Healthy: true, CheckedAt: time.Now()}
}

func (b BaseIntegration) Tools() []tools.Tool { return nil }
