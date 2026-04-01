package compliance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type RemediationStep struct {
	ID            string                 `json:"id"`
	Order         int                    `json:"order"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	Actions       []RemediationAction    `json:"actions"`
	Priority      int                    `json:"priority"`
	EstimatedTime string                 `json:"estimated_time,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
}

type RemediationAction struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	URL         string `json:"url,omitempty"`
	Automatic   bool   `json:"automatic"`
}

type RemediationService struct {
	store RemediationStore
}

type RemediationStore interface {
	GetCheckResult(ctx context.Context, resultID string) (*ComplianceCheckResult, error)
	UpdateCheckResult(ctx context.Context, result *ComplianceCheckResult) error
	GetOpenIssues(ctx context.Context, orgID string) ([]*ComplianceCheckResult, error)
	GetRule(ctx context.Context, ruleID string) (*ComplianceRule, error)
}

func NewRemediationService(store RemediationStore) *RemediationService {
	return &RemediationService{store: store}
}

func (s *RemediationService) GetRemediationSteps(ctx context.Context, result *ComplianceCheckResult) ([]RemediationStep, error) {
	rule, err := s.store.GetRule(ctx, result.RuleID)
	if err != nil {
		rule = &ComplianceRule{Type: RuleTypeDataRetention}
	}

	steps := s.generateRemediationSteps(result, rule)

	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Priority > steps[j].Priority
	})

	for i := range steps {
		steps[i].Order = i + 1
	}

	return steps, nil
}

func (s *RemediationService) generateRemediationSteps(result *ComplianceCheckResult, rule *ComplianceRule) []RemediationStep {
	steps := make([]RemediationStep, 0)

	switch rule.Type {
	case RuleTypeDataRetention:
		steps = s.dataRetentionRemediation(result, rule)
	case RuleTypeAccessControl:
		steps = s.accessControlRemediation(result, rule)
	case RuleTypeEncryption:
		steps = s.encryptionRemediation(result, rule)
	case RuleTypePasswordPolicy:
		steps = s.passwordPolicyRemediation(result, rule)
	case RuleTypeDataLocation:
		steps = s.dataLocationRemediation(result, rule)
	default:
		steps = s.genericRemediation(result, rule)
	}

	return steps
}

func (s *RemediationService) dataRetentionRemediation(result *ComplianceCheckResult, rule *ComplianceRule) []RemediationStep {
	steps := []RemediationStep{}

	for _, finding := range result.Findings {
		switch {
		case strings.Contains(finding.Title, "Not Enabled"):
			steps = append(steps, RemediationStep{
				ID:          "dr-1",
				Title:       "Enable Data Retention Policy",
				Description: "Navigate to organization settings and enable data retention",
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Data Management > Retention",
						URL:         "/settings/data-retention",
						Automatic:   false,
					},
					{
						Type:        "toggle",
						Description: "Enable the 'Data Retention' toggle",
						Automatic:   false,
					},
				},
				Priority:      10,
				EstimatedTime: "5 minutes",
			})

		case strings.Contains(finding.Title, "Retention Period"):
			days := 365
			if d, ok := finding.Details["required_days"].(int); ok {
				days = d
			} else if d, ok := finding.Details["required_days"].(float64); ok {
				days = int(d)
			}
			steps = append(steps, RemediationStep{
				ID:          "dr-2",
				Title:       "Configure Retention Period",
				Description: fmt.Sprintf("Set retention period to at least %d days", days),
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Data Management > Retention",
						URL:         "/settings/data-retention",
						Automatic:   false,
					},
					{
						Type:        "input",
						Description: fmt.Sprintf("Set 'Retention Days' to %d or higher", days),
						Automatic:   false,
					},
				},
				Priority:      9,
				EstimatedTime: "2 minutes",
				Details: map[string]interface{}{
					"required_days": days,
				},
			})

		case strings.Contains(finding.Title, "Messages Not Included"):
			steps = append(steps, RemediationStep{
				ID:          "dr-3",
				Title:       "Include Messages in Retention",
				Description: "Enable message data in retention policy",
				Actions: []RemediationAction{
					{
						Type:        "toggle",
						Description: "Enable 'Include Messages' in retention settings",
						Automatic:   false,
					},
				},
				Priority:      7,
				EstimatedTime: "1 minute",
			})

		case strings.Contains(finding.Title, "Auto-Delete"):
			steps = append(steps, RemediationStep{
				ID:          "dr-4",
				Title:       "Enable Auto-Delete",
				Description: "Enable automatic deletion of expired data",
				Actions: []RemediationAction{
					{
						Type:        "toggle",
						Description: "Enable 'Auto-Delete' in retention settings",
						Automatic:   false,
					},
				},
				Priority:      8,
				EstimatedTime: "1 minute",
			})
		}
	}

	return steps
}

func (s *RemediationService) accessControlRemediation(result *ComplianceCheckResult, rule *ComplianceRule) []RemediationStep {
	steps := []RemediationStep{}

	for _, finding := range result.Findings {
		switch {
		case strings.Contains(finding.Title, "RBAC"):
			steps = append(steps, RemediationStep{
				ID:          "ac-1",
				Title:       "Enable Role-Based Access Control",
				Description: "Configure RBAC for granular permission management",
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Security > Access Control",
						URL:         "/settings/access-control",
						Automatic:   false,
					},
					{
						Type:        "toggle",
						Description: "Enable 'Role-Based Access Control'",
						Automatic:   false,
					},
				},
				Priority:      10,
				EstimatedTime: "5 minutes",
			})

		case strings.Contains(finding.Title, "MFA"):
			steps = append(steps, RemediationStep{
				ID:          "ac-2",
				Title:       "Require Multi-Factor Authentication",
				Description: "Enable MFA requirement for all users",
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Security > Authentication",
						URL:         "/settings/authentication",
						Automatic:   false,
					},
					{
						Type:        "toggle",
						Description: "Enable 'Require MFA for all users'",
						Automatic:   false,
					},
					{
						Type:        "notify",
						Description: "Notify users to set up MFA",
						Automatic:   true,
					},
				},
				Priority:      10,
				EstimatedTime: "10 minutes",
			})

		case strings.Contains(finding.Title, "SSO"):
			steps = append(steps, RemediationStep{
				ID:          "ac-3",
				Title:       "Configure Single Sign-On",
				Description: "Set up SSO integration for centralized authentication",
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Security > SSO",
						URL:         "/settings/sso",
						Automatic:   false,
					},
					{
						Type:        "config",
						Description: "Configure SAML or OIDC provider settings",
						Automatic:   false,
					},
				},
				Priority:      7,
				EstimatedTime: "30 minutes",
			})

		case strings.Contains(finding.Title, "Session Timeout"):
			maxTimeout := 60
			if t, ok := finding.Details["max_timeout"].(int); ok {
				maxTimeout = t
			} else if t, ok := finding.Details["max_timeout"].(float64); ok {
				maxTimeout = int(t)
			}
			steps = append(steps, RemediationStep{
				ID:          "ac-4",
				Title:       "Reduce Session Timeout",
				Description: fmt.Sprintf("Set session timeout to %d minutes or less", maxTimeout),
				Actions: []RemediationAction{
					{
						Type:        "input",
						Description: fmt.Sprintf("Set 'Session Timeout' to %d minutes", maxTimeout),
						Automatic:   false,
					},
				},
				Priority:      8,
				EstimatedTime: "2 minutes",
				Details: map[string]interface{}{
					"max_timeout_minutes": maxTimeout,
				},
			})
		}
	}

	return steps
}

func (s *RemediationService) encryptionRemediation(result *ComplianceCheckResult, rule *ComplianceRule) []RemediationStep {
	steps := []RemediationStep{}

	for _, finding := range result.Findings {
		switch {
		case strings.Contains(finding.Title, "at Rest"):
			steps = append(steps, RemediationStep{
				ID:          "enc-1",
				Title:       "Enable Encryption at Rest",
				Description: "Enable AES-256 encryption for data at rest",
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Security > Encryption",
						URL:         "/settings/encryption",
						Automatic:   false,
					},
					{
						Type:        "toggle",
						Description: "Enable 'Encryption at Rest'",
						Automatic:   false,
					},
				},
				Priority:      10,
				EstimatedTime: "5 minutes",
				Details: map[string]interface{}{
					"algorithm": "AES-256-GCM",
				},
			})

		case strings.Contains(finding.Title, "in Transit"):
			steps = append(steps, RemediationStep{
				ID:          "enc-2",
				Title:       "Enable TLS for Data in Transit",
				Description: "Ensure all data transfers use TLS 1.2+",
				Actions: []RemediationAction{
					{
						Type:        "toggle",
						Description: "Enable 'Enforce TLS' in security settings",
						Automatic:   false,
					},
					{
						Type:        "verify",
						Description: "Verify SSL/TLS certificate is valid",
						Automatic:   true,
					},
				},
				Priority:      10,
				EstimatedTime: "5 minutes",
			})

		case strings.Contains(finding.Title, "Algorithm"):
			algo := "AES-256"
			if a, ok := finding.Details["required_algorithm"].(string); ok {
				algo = a
			}
			steps = append(steps, RemediationStep{
				ID:          "enc-3",
				Title:       "Update Encryption Algorithm",
				Description: fmt.Sprintf("Configure encryption to use %s", algo),
				Actions: []RemediationAction{
					{
						Type:        "select",
						Description: fmt.Sprintf("Select '%s' as encryption algorithm", algo),
						Automatic:   false,
					},
				},
				Priority:      8,
				EstimatedTime: "3 minutes",
			})

		case strings.Contains(finding.Title, "Key Rotation"):
			steps = append(steps, RemediationStep{
				ID:          "enc-4",
				Title:       "Configure Key Rotation",
				Description: "Set up automatic encryption key rotation",
				Actions: []RemediationAction{
					{
						Type:        "toggle",
						Description: "Enable 'Automatic Key Rotation'",
						Automatic:   false,
					},
					{
						Type:        "input",
						Description: "Set rotation period to 90 days or less",
						Automatic:   false,
					},
				},
				Priority:      7,
				EstimatedTime: "10 minutes",
			})
		}
	}

	return steps
}

func (s *RemediationService) passwordPolicyRemediation(result *ComplianceCheckResult, rule *ComplianceRule) []RemediationStep {
	steps := []RemediationStep{}

	var configActions []RemediationAction
	configActions = append(configActions, RemediationAction{
		Type:        "navigate",
		Description: "Go to Settings > Security > Password Policy",
		URL:         "/settings/password-policy",
		Automatic:   false,
	})

	for _, finding := range result.Findings {
		switch {
		case strings.Contains(finding.Title, "Length"):
			minLen := 12
			if l, ok := finding.Details["required_length"].(int); ok {
				minLen = l
			} else if l, ok := finding.Details["required_length"].(float64); ok {
				minLen = int(l)
			}
			configActions = append(configActions, RemediationAction{
				Type:        "input",
				Description: fmt.Sprintf("Set minimum password length to %d", minLen),
				Automatic:   false,
			})

		case strings.Contains(finding.Title, "Uppercase"):
			configActions = append(configActions, RemediationAction{
				Type:        "toggle",
				Description: "Enable 'Require Uppercase Letters'",
				Automatic:   false,
			})

		case strings.Contains(finding.Title, "Lowercase"):
			configActions = append(configActions, RemediationAction{
				Type:        "toggle",
				Description: "Enable 'Require Lowercase Letters'",
				Automatic:   false,
			})

		case strings.Contains(finding.Title, "Number"):
			configActions = append(configActions, RemediationAction{
				Type:        "toggle",
				Description: "Enable 'Require Numbers'",
				Automatic:   false,
			})

		case strings.Contains(finding.Title, "Symbol"):
			configActions = append(configActions, RemediationAction{
				Type:        "toggle",
				Description: "Enable 'Require Special Characters'",
				Automatic:   false,
			})

		case strings.Contains(finding.Title, "Expiration"):
			configActions = append(configActions, RemediationAction{
				Type:        "input",
				Description: "Set password expiration to 90 days or less",
				Automatic:   false,
			})

		case strings.Contains(finding.Title, "History"):
			configActions = append(configActions, RemediationAction{
				Type:        "input",
				Description: "Set password history to at least 5 passwords",
				Automatic:   false,
			})
		}
	}

	if len(configActions) > 1 {
		steps = append(steps, RemediationStep{
			ID:            "pp-1",
			Title:         "Configure Password Policy",
			Description:   "Update password policy to meet compliance requirements",
			Actions:       configActions,
			Priority:      9,
			EstimatedTime: "5 minutes",
		})

		steps = append(steps, RemediationStep{
			ID:          "pp-2",
			Title:       "Notify Users of Policy Change",
			Description: "Inform users about updated password requirements",
			Actions: []RemediationAction{
				{
					Type:        "notify",
					Description: "Send notification to all users about password policy update",
					Automatic:   true,
				},
			},
			Priority:      6,
			EstimatedTime: "Immediate",
		})
	}

	return steps
}

func (s *RemediationService) dataLocationRemediation(result *ComplianceCheckResult, rule *ComplianceRule) []RemediationStep {
	steps := []RemediationStep{}

	for _, finding := range result.Findings {
		switch {
		case strings.Contains(finding.Title, "Residency"):
			steps = append(steps, RemediationStep{
				ID:          "dl-1",
				Title:       "Enable Data Residency",
				Description: "Configure data residency to ensure data stays in approved regions",
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Data Management > Data Location",
						URL:         "/settings/data-location",
						Automatic:   false,
					},
					{
						Type:        "toggle",
						Description: "Enable 'Data Residency Enforcement'",
						Automatic:   false,
					},
				},
				Priority:      10,
				EstimatedTime: "5 minutes",
			})

		case strings.Contains(finding.Title, "Region Not in Allowed"):
			regions := []string{"us-east-1", "us-west-2"}
			if r, ok := finding.Details["allowed_regions"].([]string); ok {
				regions = r
			} else if r, ok := finding.Details["allowed_regions"].([]interface{}); ok {
				regions = make([]string, len(r))
				for i, v := range r {
					regions[i] = fmt.Sprintf("%v", v)
				}
			}
			steps = append(steps, RemediationStep{
				ID:          "dl-2",
				Title:       "Select Compliant Region",
				Description: fmt.Sprintf("Choose a primary region from allowed regions: %v", regions),
				Actions: []RemediationAction{
					{
						Type:        "select",
						Description: fmt.Sprintf("Select primary region from: %s", strings.Join(regions, ", ")),
						Automatic:   false,
					},
				},
				Priority:      10,
				EstimatedTime: "3 minutes",
				Details: map[string]interface{}{
					"allowed_regions": regions,
				},
			})

		case strings.Contains(finding.Title, "Not Configured"):
			steps = append(steps, RemediationStep{
				ID:          "dl-3",
				Title:       "Configure Primary Region",
				Description: "Set a primary data region for your organization",
				Actions: []RemediationAction{
					{
						Type:        "navigate",
						Description: "Go to Settings > Data Management > Data Location",
						URL:         "/settings/data-location",
						Automatic:   false,
					},
					{
						Type:        "select",
						Description: "Select a primary region for data storage",
						Automatic:   false,
					},
				},
				Priority:      9,
				EstimatedTime: "5 minutes",
			})

		case strings.Contains(finding.Title, "Mismatch"):
			steps = append(steps, RemediationStep{
				ID:          "dl-4",
				Title:       "Resolve Data Region Mismatch",
				Description: "Migrate data or update configuration to match regions",
				Actions: []RemediationAction{
					{
						Type:        "migrate",
						Description: "Initiate data migration to configured region",
						Automatic:   false,
					},
					{
						Type:        "verify",
						Description: "Verify all data is in the correct region",
						Automatic:   true,
					},
				},
				Priority:      8,
				EstimatedTime: "Varies by data size",
			})
		}
	}

	return steps
}

func (s *RemediationService) genericRemediation(result *ComplianceCheckResult, rule *ComplianceRule) []RemediationStep {
	steps := []RemediationStep{
		{
			ID:          "gen-1",
			Title:       "Review Compliance Finding",
			Description: "Review the compliance finding details and take appropriate action",
			Actions: []RemediationAction{
				{
					Type:        "review",
					Description: fmt.Sprintf("Review finding for rule: %s", rule.Name),
					Automatic:   false,
				},
			},
			Priority:      5,
			EstimatedTime: "Varies",
		},
	}
	return steps
}

func (s *RemediationService) MarkAsResolved(ctx context.Context, resultID string, notes string) error {
	result, err := s.store.GetCheckResult(ctx, resultID)
	if err != nil {
		return fmt.Errorf("failed to get check result: %w", err)
	}

	now := time.Now()
	result.ResolvedAt = &now
	result.ResolutionNotes = notes

	if err := s.store.UpdateCheckResult(ctx, result); err != nil {
		return fmt.Errorf("failed to update check result: %w", err)
	}

	return nil
}

func (s *RemediationService) GetOpenIssues(ctx context.Context, orgID string) ([]*ComplianceCheckResult, error) {
	issues, err := s.store.GetOpenIssues(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get open issues: %w", err)
	}

	sort.Slice(issues, func(i, j int) bool {
		criticalI := len(issues[i].CriticalFindings())
		criticalJ := len(issues[j].CriticalFindings())
		if criticalI != criticalJ {
			return criticalI > criticalJ
		}
		return issues[i].CheckedAt.After(issues[j].CheckedAt)
	})

	return issues, nil
}

func (s *RemediationService) GetIssueSummary(ctx context.Context, orgID string) (*IssueSummary, error) {
	issues, err := s.GetOpenIssues(ctx, orgID)
	if err != nil {
		return nil, err
	}

	summary := &IssueSummary{
		TotalOpen:    len(issues),
		BySeverity:   make(map[Severity]int),
		ByType:       make(map[RuleType]int),
		ByFramework:  make(map[ComplianceFramework]int),
		OldestIssue:  nil,
		MostCritical: nil,
	}

	for _, issue := range issues {
		summary.BySeverity[SeverityCritical] += len(issue.CriticalFindings())
		summary.BySeverity[SeverityWarning] += len(issue.WarningFindings())

		if summary.OldestIssue == nil || issue.CheckedAt.Before(summary.OldestIssue.CheckedAt) {
			summary.OldestIssue = issue
		}

		if summary.MostCritical == nil || len(issue.CriticalFindings()) > len(summary.MostCritical.CriticalFindings()) {
			summary.MostCritical = issue
		}
	}

	return summary, nil
}

type IssueSummary struct {
	TotalOpen    int                         `json:"total_open"`
	BySeverity   map[Severity]int            `json:"by_severity"`
	ByType       map[RuleType]int            `json:"by_type"`
	ByFramework  map[ComplianceFramework]int `json:"by_framework"`
	OldestIssue  *ComplianceCheckResult      `json:"oldest_issue,omitempty"`
	MostCritical *ComplianceCheckResult      `json:"most_critical,omitempty"`
}
