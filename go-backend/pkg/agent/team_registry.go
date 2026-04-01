// OctAi - Team Registry
// Tracks team membership and provides role-based agent lookup.
package agent

import (
	"fmt"
	"sync"

	agentroles "github.com/raynaythegreat/octai-app/pkg/agent/roles"
	"github.com/raynaythegreat/octai-app/pkg/config"
	"github.com/raynaythegreat/octai-app/pkg/routing"
)

// TeamMember describes an agent's membership in a team.
type TeamMember struct {
	AgentID string
	Role    agentroles.Role
}

// Team holds the runtime state of an agent team.
type Team struct {
	cfg           config.TeamConfig
	orchestratorID string
	members       []TeamMember // ordered list of member agents (excluding orchestrator)
}

// ID returns the team's config identifier.
func (t *Team) ID() string { return t.cfg.ID }

// Name returns the team's human-readable name.
func (t *Team) Name() string {
	if t.cfg.Name != "" {
		return t.cfg.Name
	}
	return t.cfg.ID
}

// OrchestratorID returns the agent ID that leads this team.
func (t *Team) OrchestratorID() string { return t.orchestratorID }

// Members returns a snapshot of the team's member list.
func (t *Team) Members() []TeamMember {
	out := make([]TeamMember, len(t.members))
	copy(out, t.members)
	return out
}

// Config returns the underlying TeamConfig.
func (t *Team) Config() config.TeamConfig { return t.cfg }

// TeamRegistry maps team IDs to their runtime Team state and provides
// role-based agent lookup for the TeamTool.
type TeamRegistry struct {
	teams    map[string]*Team          // team ID → Team
	agentTeam map[string]string        // agent ID → team ID (for reverse lookup)
	agentRole map[string]agentroles.Role    // agent ID → role
	mu       sync.RWMutex
}

// NewTeamRegistry builds a TeamRegistry from the config's Teams list.
// It cross-references AgentConfig entries to populate role information.
func NewTeamRegistry(cfg *config.Config) *TeamRegistry {
	tr := &TeamRegistry{
		teams:     make(map[string]*Team),
		agentTeam: make(map[string]string),
		agentRole: make(map[string]agentroles.Role),
	}

	// Build agent ID → role map from AgentConfig entries.
	for i := range cfg.Agents.List {
		ac := &cfg.Agents.List[i]
		id := routing.NormalizeAgentID(ac.ID)
		if ac.Role != "" {
			tr.agentRole[id] = agentroles.Role(ac.Role)
		}
	}

	// Build teams.
	for _, tc := range cfg.Agents.Teams {
		orchestratorID := routing.NormalizeAgentID(tc.OrchestratorID)

		team := &Team{
			cfg:            tc,
			orchestratorID: orchestratorID,
		}

		// Mark orchestrator's team membership.
		tr.agentTeam[orchestratorID] = tc.ID
		// Ensure orchestrator has a role recorded.
		if _, ok := tr.agentRole[orchestratorID]; !ok {
			tr.agentRole[orchestratorID] = agentroles.RoleOrchestrator
		}

		// Register members.
		for _, memberID := range tc.MemberIDs {
			normID := routing.NormalizeAgentID(memberID)
			role := tr.agentRole[normID] // may be empty → RoleCustom
			if role == "" {
				role = agentroles.RoleCustom
			}
			team.members = append(team.members, TeamMember{
				AgentID: normID,
				Role:    role,
			})
			tr.agentTeam[normID] = tc.ID
		}

		tr.teams[tc.ID] = team
	}

	return tr
}

// GetTeam returns the Team for a given team ID.
func (tr *TeamRegistry) GetTeam(teamID string) (*Team, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	t, ok := tr.teams[teamID]
	return t, ok
}

// TeamOf returns the team ID that the given agent belongs to.
func (tr *TeamRegistry) TeamOf(agentID string) (string, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	id := routing.NormalizeAgentID(agentID)
	teamID, ok := tr.agentTeam[id]
	return teamID, ok
}

// RoleOf returns the role of the given agent.
func (tr *TeamRegistry) RoleOf(agentID string) (agentroles.Role, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	id := routing.NormalizeAgentID(agentID)
	r, ok := tr.agentRole[id]
	return r, ok
}

// FindByRole returns the first agent ID in the team that matches the requested role string.
// This satisfies the tools.TeamResolver interface. Returns an error if no matching member is found.
func (tr *TeamRegistry) FindByRole(teamID string, role string) (string, error) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	team, ok := tr.teams[teamID]
	if !ok {
		return "", fmt.Errorf("team %q not found", teamID)
	}

	r := agentroles.Role(role)
	for _, m := range team.members {
		if m.Role == r {
			return m.AgentID, nil
		}
	}
	return "", fmt.Errorf("no agent with role %q found in team %q", role, teamID)
}

// ListTeamIDs returns all registered team IDs.
func (tr *TeamRegistry) ListTeamIDs() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	ids := make([]string, 0, len(tr.teams))
	for id := range tr.teams {
		ids = append(ids, id)
	}
	return ids
}
