// OctAi - Integration Registry
// Manages active integrations and provides role-based tool registration.
package integrations

import (
	"context"
	"fmt"
	"sync"

	"github.com/raynaythegreat/octai-app/pkg/logger"
	"github.com/raynaythegreat/octai-app/pkg/tools"
)

// Registry manages all active integrations and auto-registers their tools to agents.
type Registry struct {
	integrations map[string]Integration
	mu           sync.RWMutex
}

// NewRegistry creates an empty integration registry.
func NewRegistry() *Registry {
	return &Registry{
		integrations: make(map[string]Integration),
	}
}

// Register adds an integration to the registry.
func (r *Registry) Register(i Integration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.integrations[i.Name()] = i
}

// Get returns an integration by name.
func (r *Registry) Get(name string) (Integration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i, ok := r.integrations[name]
	return i, ok
}

// ConnectAll connects all registered integrations using their respective configs.
func (r *Registry) ConnectAll(ctx context.Context, configs map[string]IntegrationConfig) {
	r.mu.RLock()
	integrations := make([]Integration, 0, len(r.integrations))
	for _, i := range r.integrations {
		integrations = append(integrations, i)
	}
	r.mu.RUnlock()

	for _, i := range integrations {
		cfg := configs[i.Name()]
		if err := i.Connect(ctx, cfg); err != nil {
			logger.WarnCF("integrations", fmt.Sprintf("Failed to connect %s integration", i.Name()),
				map[string]any{"error": err.Error()})
		} else {
			logger.InfoCF("integrations", fmt.Sprintf("Connected %s integration", i.Name()), nil)
		}
	}
}

// HealthCheckAll checks the health of all integrations.
func (r *Registry) HealthCheckAll(ctx context.Context) map[string]HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make(map[string]HealthStatus, len(r.integrations))
	for name, i := range r.integrations {
		statuses[name] = i.Health(ctx)
	}
	return statuses
}

// ToolsForRole returns all tools from integrations that target the given role.
// Integrations with an empty ForRoles() list provide tools for all roles.
func (r *Registry) ToolsForRole(role string) []tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []tools.Tool
	for _, i := range r.integrations {
		roles := i.ForRoles()
		if len(roles) == 0 || containsRole(roles, role) {
			result = append(result, i.Tools()...)
		}
	}
	return result
}

// AllTools returns tools from every integration regardless of role targeting.
func (r *Registry) AllTools() []tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []tools.Tool
	for _, i := range r.integrations {
		result = append(result, i.Tools()...)
	}
	return result
}

// List returns all registered integration names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.integrations))
	for n := range r.integrations {
		names = append(names, n)
	}
	return names
}

func containsRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
