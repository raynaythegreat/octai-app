// OctAi - Agent Role System
// Defines built-in agent roles with default tool sets and metadata.
package roles

// Role identifies the specialization of an agent within a team.
type Role string

const (
	// RoleOrchestrator coordinates the team, delegates tasks, and aggregates results.
	RoleOrchestrator Role = "orchestrator"
	// RoleSales handles CRM operations, lead qualification, and sales pipeline management.
	RoleSales Role = "sales"
	// RoleSupport handles customer support, ticket management, and issue resolution.
	RoleSupport Role = "support"
	// RoleResearch performs web research, data gathering, and information synthesis.
	RoleResearch Role = "research"
	// RoleContent creates content, copywriting, and social media posts.
	RoleContent Role = "content"
	// RoleAnalytics analyzes data, generates reports, and tracks metrics.
	RoleAnalytics Role = "analytics"
	// RoleAdmin handles system configuration, user management, and billing.
	RoleAdmin Role = "admin"
	// RoleCustom is a user-defined role with a fully custom tool set.
	RoleCustom Role = "custom"
)

// RoleInfo describes a built-in agent role.
type RoleInfo struct {
	// Name is the human-readable display name.
	Name string
	// Description explains the role's purpose.
	Description string
	// DefaultTools lists tool names enabled by default for this role.
	DefaultTools []string
	// PreferHeavyModel hints whether this role benefits from a larger/smarter model.
	PreferHeavyModel bool
}

// registry maps known role identifiers to their metadata.
var registry = map[Role]RoleInfo{
	RoleOrchestrator: {
		Name:        "Orchestrator",
		Description: "Coordinates the agent team, delegates tasks to specialists, and aggregates results into coherent responses.",
		DefaultTools: []string{
			"team",
			"spawn",
			"subagent",
			"web_search",
			"knowledge_search",
		},
		PreferHeavyModel: true,
	},
	RoleSales: {
		Name:        "Sales",
		Description: "Manages CRM data, qualifies leads, tracks pipeline stages, and drafts outreach messages.",
		DefaultTools: []string{
			"web_search",
			"web_fetch",
			"knowledge_search",
			"knowledge_add",
		},
		PreferHeavyModel: true,
	},
	RoleSupport: {
		Name:        "Support",
		Description: "Resolves customer issues, manages support tickets, and retrieves knowledge base articles.",
		DefaultTools: []string{
			"knowledge_search",
			"web_fetch",
		},
		PreferHeavyModel: false,
	},
	RoleResearch: {
		Name:        "Research",
		Description: "Conducts web research, gathers competitive intelligence, and synthesizes findings.",
		DefaultTools: []string{
			"web_search",
			"web_fetch",
			"read_file",
			"knowledge_add",
		},
		PreferHeavyModel: false,
	},
	RoleContent: {
		Name:        "Content",
		Description: "Creates blog posts, social media content, marketing copy, and other written materials.",
		DefaultTools: []string{
			"web_fetch",
			"read_file",
			"write_file",
			"knowledge_search",
		},
		PreferHeavyModel: true,
	},
	RoleAnalytics: {
		Name:        "Analytics",
		Description: "Analyzes business data, generates performance reports, and tracks key metrics.",
		DefaultTools: []string{
			"read_file",
			"knowledge_search",
		},
		PreferHeavyModel: false,
	},
	RoleAdmin: {
		Name:        "Admin",
		Description: "Handles system configuration, user management, billing operations, and compliance checks.",
		DefaultTools: []string{
			"knowledge_search",
		},
		PreferHeavyModel: false,
	},
	RoleCustom: {
		Name:        "Custom",
		Description: "User-defined role with a fully configurable tool set and system prompt.",
		DefaultTools: []string{},
		PreferHeavyModel: false,
	},
}

// Lookup returns the RoleInfo for a known role, plus an existence flag.
func Lookup(r Role) (RoleInfo, bool) {
	info, ok := registry[r]
	return info, ok
}

// DefaultTools returns the default tool names for a role.
// Returns an empty slice for unknown roles.
func DefaultTools(r Role) []string {
	if info, ok := registry[r]; ok {
		out := make([]string, len(info.DefaultTools))
		copy(out, info.DefaultTools)
		return out
	}
	return []string{}
}

// IsKnown reports whether r is a recognized built-in role.
func IsKnown(r Role) bool {
	_, ok := registry[r]
	return ok
}

// All returns every built-in role.
func All() []Role {
	roles := make([]Role, 0, len(registry))
	for r := range registry {
		roles = append(roles, r)
	}
	return roles
}
