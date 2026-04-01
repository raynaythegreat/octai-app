package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ComplianceChecker interface {
	Check(ctx context.Context, orgID string, rule ComplianceRule) (*ComplianceCheckResult, error)
	Type() RuleType
}

type CheckerConfig struct {
	Store ComplianceStore
}

type ComplianceStore interface {
	GetOrgConfig(ctx context.Context, orgID string, configType string) (json.RawMessage, error)
	GetMemberships(ctx context.Context, orgID string) ([]map[string]interface{}, error)
	GetDataRegion(ctx context.Context, orgID string) (string, error)
}

type DataRetentionChecker struct {
	store ComplianceStore
}

func NewDataRetentionChecker(store ComplianceStore) *DataRetentionChecker {
	return &DataRetentionChecker{store: store}
}

func (c *DataRetentionChecker) Type() RuleType {
	return RuleTypeDataRetention
}

func (c *DataRetentionChecker) Check(ctx context.Context, orgID string, rule ComplianceRule) (*ComplianceCheckResult, error) {
	result := &ComplianceCheckResult{
		ID:             uuid.New().String(),
		RuleID:         rule.ID,
		OrganizationID: orgID,
		CheckedAt:      time.Now(),
		MaxScore:       100,
	}

	var ruleConfig DataRetentionConfig
	if len(rule.Config) > 0 {
		if err := json.Unmarshal(rule.Config, &ruleConfig); err != nil {
			return nil, fmt.Errorf("failed to parse rule config: %w", err)
		}
	}

	orgConfig, err := c.store.GetOrgConfig(ctx, orgID, "data_retention")
	if err != nil {
		result.Status = CheckStatusWarning
		result.Findings = append(result.Findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Data Retention Configuration Not Found",
			Description: "Organization has not configured data retention settings",
			Severity:    SeverityWarning,
		})
		result.Score = 0
		return result, nil
	}

	var orgRetention DataRetentionConfig
	if err := json.Unmarshal(orgConfig, &orgRetention); err != nil {
		return nil, fmt.Errorf("failed to parse org config: %w", err)
	}

	findings := []Finding{}
	score := 100

	if !orgRetention.Enabled {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Data Retention Not Enabled",
			Description: "Data retention policy is not enabled for this organization",
			Severity:    rule.Severity,
		})
		score -= 40
	}

	if ruleConfig.RetentionDays > 0 && orgRetention.RetentionDays < ruleConfig.RetentionDays {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Insufficient Retention Period",
			Description: fmt.Sprintf("Retention period (%d days) is less than required (%d days)", orgRetention.RetentionDays, ruleConfig.RetentionDays),
			Severity:    SeverityWarning,
			Details: map[string]interface{}{
				"current_days":  orgRetention.RetentionDays,
				"required_days": ruleConfig.RetentionDays,
			},
		})
		score -= 20
	}

	if ruleConfig.IncludeMessages && !orgRetention.IncludeMessages {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Messages Not Included in Retention",
			Description: "Message data is not included in retention policy",
			Severity:    SeverityInfo,
		})
		score -= 10
	}

	if ruleConfig.AutoDelete && !orgRetention.AutoDelete {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Auto-Delete Not Enabled",
			Description: "Automatic deletion of expired data is not enabled",
			Severity:    SeverityWarning,
		})
		score -= 15
	}

	result.Findings = findings
	result.Score = score

	if len(findings) == 0 {
		result.Status = CheckStatusPass
	} else if score < 50 {
		result.Status = CheckStatusFail
	} else {
		result.Status = CheckStatusWarning
	}

	return result, nil
}

type AccessControlChecker struct {
	store ComplianceStore
}

func NewAccessControlChecker(store ComplianceStore) *AccessControlChecker {
	return &AccessControlChecker{store: store}
}

func (c *AccessControlChecker) Type() RuleType {
	return RuleTypeAccessControl
}

func (c *AccessControlChecker) Check(ctx context.Context, orgID string, rule ComplianceRule) (*ComplianceCheckResult, error) {
	result := &ComplianceCheckResult{
		ID:             uuid.New().String(),
		RuleID:         rule.ID,
		OrganizationID: orgID,
		CheckedAt:      time.Now(),
		MaxScore:       100,
	}

	var ruleConfig AccessControlConfig
	if len(rule.Config) > 0 {
		if err := json.Unmarshal(rule.Config, &ruleConfig); err != nil {
			return nil, fmt.Errorf("failed to parse rule config: %w", err)
		}
	}

	orgConfig, err := c.store.GetOrgConfig(ctx, orgID, "access_control")
	if err != nil {
		result.Status = CheckStatusWarning
		result.Findings = append(result.Findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Access Control Configuration Not Found",
			Description: "Organization has not configured access control settings",
			Severity:    SeverityWarning,
		})
		result.Score = 0
		return result, nil
	}

	var orgAccess AccessControlConfig
	if err := json.Unmarshal(orgConfig, &orgAccess); err != nil {
		return nil, fmt.Errorf("failed to parse org config: %w", err)
	}

	findings := []Finding{}
	score := 100

	if ruleConfig.RBACEnabled && !orgAccess.RBACEnabled {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "RBAC Not Enabled",
			Description: "Role-based access control is not enabled",
			Severity:    SeverityCritical,
		})
		score -= 35
	}

	if ruleConfig.MFARequired && !orgAccess.MFARequired {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "MFA Not Required",
			Description: "Multi-factor authentication is not required for users",
			Severity:    SeverityCritical,
		})
		score -= 30
	}

	if ruleConfig.SSOEnabled && !orgAccess.SSOEnabled {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "SSO Not Enabled",
			Description: "Single sign-on is not configured",
			Severity:    SeverityWarning,
		})
		score -= 15
	}

	if ruleConfig.SessionTimeoutMin > 0 && (orgAccess.SessionTimeoutMin == 0 || orgAccess.SessionTimeoutMin > ruleConfig.SessionTimeoutMin) {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Session Timeout Too Long",
			Description: fmt.Sprintf("Session timeout (%d min) exceeds maximum allowed (%d min)", orgAccess.SessionTimeoutMin, ruleConfig.SessionTimeoutMin),
			Severity:    SeverityWarning,
			Details: map[string]interface{}{
				"current_timeout": orgAccess.SessionTimeoutMin,
				"max_timeout":     ruleConfig.SessionTimeoutMin,
			},
		})
		score -= 10
	}

	memberships, err := c.store.GetMemberships(ctx, orgID)
	if err == nil {
		if len(memberships) == 0 {
			findings = append(findings, Finding{
				ID:          uuid.New().String(),
				Title:       "No Members Found",
				Description: "Organization has no members configured",
				Severity:    SeverityInfo,
			})
		}
	}

	result.Findings = findings
	result.Score = score

	if len(findings) == 0 {
		result.Status = CheckStatusPass
	} else if score < 50 {
		result.Status = CheckStatusFail
	} else {
		result.Status = CheckStatusWarning
	}

	return result, nil
}

type EncryptionChecker struct {
	store ComplianceStore
}

func NewEncryptionChecker(store ComplianceStore) *EncryptionChecker {
	return &EncryptionChecker{store: store}
}

func (c *EncryptionChecker) Type() RuleType {
	return RuleTypeEncryption
}

func (c *EncryptionChecker) Check(ctx context.Context, orgID string, rule ComplianceRule) (*ComplianceCheckResult, error) {
	result := &ComplianceCheckResult{
		ID:             uuid.New().String(),
		RuleID:         rule.ID,
		OrganizationID: orgID,
		CheckedAt:      time.Now(),
		MaxScore:       100,
	}

	var ruleConfig EncryptionConfig
	if len(rule.Config) > 0 {
		if err := json.Unmarshal(rule.Config, &ruleConfig); err != nil {
			return nil, fmt.Errorf("failed to parse rule config: %w", err)
		}
	}

	orgConfig, err := c.store.GetOrgConfig(ctx, orgID, "encryption")
	if err != nil {
		result.Status = CheckStatusWarning
		result.Findings = append(result.Findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Encryption Configuration Not Found",
			Description: "Organization has not configured encryption settings",
			Severity:    SeverityWarning,
		})
		result.Score = 0
		return result, nil
	}

	var orgEncryption EncryptionConfig
	if err := json.Unmarshal(orgConfig, &orgEncryption); err != nil {
		return nil, fmt.Errorf("failed to parse org config: %w", err)
	}

	findings := []Finding{}
	score := 100

	if ruleConfig.AtRestEnabled && !orgEncryption.AtRestEnabled {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Encryption at Rest Not Enabled",
			Description: "Data at rest encryption is not enabled",
			Severity:    SeverityCritical,
		})
		score -= 40
	}

	if ruleConfig.InTransitEnabled && !orgEncryption.InTransitEnabled {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Encryption in Transit Not Enabled",
			Description: "Data in transit encryption is not enabled (TLS)",
			Severity:    SeverityCritical,
		})
		score -= 35
	}

	if ruleConfig.Algorithm != "" && orgEncryption.Algorithm != ruleConfig.Algorithm {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Encryption Algorithm Mismatch",
			Description: fmt.Sprintf("Encryption algorithm (%s) does not match required (%s)", orgEncryption.Algorithm, ruleConfig.Algorithm),
			Severity:    SeverityWarning,
			Details: map[string]interface{}{
				"current_algorithm":  orgEncryption.Algorithm,
				"required_algorithm": ruleConfig.Algorithm,
			},
		})
		score -= 15
	}

	if ruleConfig.KeyRotationDays > 0 && (orgEncryption.KeyRotationDays == 0 || orgEncryption.KeyRotationDays > ruleConfig.KeyRotationDays) {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Key Rotation Not Configured Properly",
			Description: fmt.Sprintf("Key rotation period (%d days) exceeds recommended (%d days)", orgEncryption.KeyRotationDays, ruleConfig.KeyRotationDays),
			Severity:    SeverityWarning,
		})
		score -= 10
	}

	result.Findings = findings
	result.Score = score

	if len(findings) == 0 {
		result.Status = CheckStatusPass
	} else if score < 50 {
		result.Status = CheckStatusFail
	} else {
		result.Status = CheckStatusWarning
	}

	return result, nil
}

type PasswordPolicyChecker struct {
	store ComplianceStore
}

func NewPasswordPolicyChecker(store ComplianceStore) *PasswordPolicyChecker {
	return &PasswordPolicyChecker{store: store}
}

func (c *PasswordPolicyChecker) Type() RuleType {
	return RuleTypePasswordPolicy
}

func (c *PasswordPolicyChecker) Check(ctx context.Context, orgID string, rule ComplianceRule) (*ComplianceCheckResult, error) {
	result := &ComplianceCheckResult{
		ID:             uuid.New().String(),
		RuleID:         rule.ID,
		OrganizationID: orgID,
		CheckedAt:      time.Now(),
		MaxScore:       100,
	}

	var ruleConfig PasswordPolicyConfig
	if len(rule.Config) > 0 {
		if err := json.Unmarshal(rule.Config, &ruleConfig); err != nil {
			return nil, fmt.Errorf("failed to parse rule config: %w", err)
		}
	}

	orgConfig, err := c.store.GetOrgConfig(ctx, orgID, "password_policy")
	if err != nil {
		result.Status = CheckStatusWarning
		result.Findings = append(result.Findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Password Policy Configuration Not Found",
			Description: "Organization has not configured password policy settings",
			Severity:    SeverityWarning,
		})
		result.Score = 0
		return result, nil
	}

	var orgPolicy PasswordPolicyConfig
	if err := json.Unmarshal(orgConfig, &orgPolicy); err != nil {
		return nil, fmt.Errorf("failed to parse org config: %w", err)
	}

	findings := []Finding{}
	score := 100

	if ruleConfig.MinLength > 0 && orgPolicy.MinLength < ruleConfig.MinLength {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Password Minimum Length Too Short",
			Description: fmt.Sprintf("Minimum password length (%d) is less than required (%d)", orgPolicy.MinLength, ruleConfig.MinLength),
			Severity:    SeverityWarning,
			Details: map[string]interface{}{
				"current_length":  orgPolicy.MinLength,
				"required_length": ruleConfig.MinLength,
			},
		})
		score -= 20
	}

	if ruleConfig.RequireUppercase && !orgPolicy.RequireUppercase {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Uppercase Requirement Not Enforced",
			Description: "Passwords do not require uppercase letters",
			Severity:    SeverityInfo,
		})
		score -= 10
	}

	if ruleConfig.RequireLowercase && !orgPolicy.RequireLowercase {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Lowercase Requirement Not Enforced",
			Description: "Passwords do not require lowercase letters",
			Severity:    SeverityInfo,
		})
		score -= 10
	}

	if ruleConfig.RequireNumbers && !orgPolicy.RequireNumbers {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Number Requirement Not Enforced",
			Description: "Passwords do not require numeric characters",
			Severity:    SeverityInfo,
		})
		score -= 10
	}

	if ruleConfig.RequireSymbols && !orgPolicy.RequireSymbols {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Symbol Requirement Not Enforced",
			Description: "Passwords do not require special characters",
			Severity:    SeverityInfo,
		})
		score -= 10
	}

	if ruleConfig.MaxAgeDays > 0 && (orgPolicy.MaxAgeDays == 0 || orgPolicy.MaxAgeDays > ruleConfig.MaxAgeDays) {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Password Expiration Too Long",
			Description: fmt.Sprintf("Password max age (%d days) exceeds recommended (%d days)", orgPolicy.MaxAgeDays, ruleConfig.MaxAgeDays),
			Severity:    SeverityWarning,
		})
		score -= 15
	}

	if ruleConfig.HistoryCount > 0 && orgPolicy.HistoryCount < ruleConfig.HistoryCount {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Password History Too Short",
			Description: fmt.Sprintf("Password history (%d) is less than required (%d)", orgPolicy.HistoryCount, ruleConfig.HistoryCount),
			Severity:    SeverityInfo,
		})
		score -= 10
	}

	result.Findings = findings
	result.Score = score

	if len(findings) == 0 {
		result.Status = CheckStatusPass
	} else if score < 50 {
		result.Status = CheckStatusFail
	} else {
		result.Status = CheckStatusWarning
	}

	return result, nil
}

type DataLocationChecker struct {
	store ComplianceStore
}

func NewDataLocationChecker(store ComplianceStore) *DataLocationChecker {
	return &DataLocationChecker{store: store}
}

func (c *DataLocationChecker) Type() RuleType {
	return RuleTypeDataLocation
}

func (c *DataLocationChecker) Check(ctx context.Context, orgID string, rule ComplianceRule) (*ComplianceCheckResult, error) {
	result := &ComplianceCheckResult{
		ID:             uuid.New().String(),
		RuleID:         rule.ID,
		OrganizationID: orgID,
		CheckedAt:      time.Now(),
		MaxScore:       100,
	}

	var ruleConfig DataLocationConfig
	if len(rule.Config) > 0 {
		if err := json.Unmarshal(rule.Config, &ruleConfig); err != nil {
			return nil, fmt.Errorf("failed to parse rule config: %w", err)
		}
	}

	orgConfig, err := c.store.GetOrgConfig(ctx, orgID, "data_location")
	if err != nil {
		result.Status = CheckStatusWarning
		result.Findings = append(result.Findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Data Location Configuration Not Found",
			Description: "Organization has not configured data location settings",
			Severity:    SeverityWarning,
		})
		result.Score = 0
		return result, nil
	}

	var orgLocation DataLocationConfig
	if err := json.Unmarshal(orgConfig, &orgLocation); err != nil {
		return nil, fmt.Errorf("failed to parse org config: %w", err)
	}

	findings := []Finding{}
	score := 100

	if ruleConfig.DataResidency && !orgLocation.DataResidency {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Data Residency Not Enforced",
			Description: "Data residency requirements are not configured",
			Severity:    SeverityCritical,
		})
		score -= 40
	}

	if len(ruleConfig.AllowedRegions) > 0 {
		regionAllowed := false
		for _, allowed := range ruleConfig.AllowedRegions {
			if orgLocation.PrimaryRegion == allowed {
				regionAllowed = true
				break
			}
		}
		if !regionAllowed && orgLocation.PrimaryRegion != "" {
			findings = append(findings, Finding{
				ID:          uuid.New().String(),
				Title:       "Primary Region Not in Allowed List",
				Description: fmt.Sprintf("Primary region (%s) is not in the list of allowed regions", orgLocation.PrimaryRegion),
				Severity:    SeverityCritical,
				Details: map[string]interface{}{
					"current_region":  orgLocation.PrimaryRegion,
					"allowed_regions": ruleConfig.AllowedRegions,
				},
			})
			score -= 35
		}
	}

	if orgLocation.PrimaryRegion == "" {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Primary Region Not Configured",
			Description: "Organization has not specified a primary data region",
			Severity:    SeverityWarning,
		})
		score -= 20
	}

	actualRegion, err := c.store.GetDataRegion(ctx, orgID)
	if err == nil && actualRegion != "" && actualRegion != orgLocation.PrimaryRegion {
		findings = append(findings, Finding{
			ID:          uuid.New().String(),
			Title:       "Data Region Mismatch",
			Description: fmt.Sprintf("Actual data region (%s) differs from configured region (%s)", actualRegion, orgLocation.PrimaryRegion),
			Severity:    SeverityWarning,
			Resource:    "data_storage",
		})
		score -= 15
	}

	result.Findings = findings
	result.Score = score

	if len(findings) == 0 {
		result.Status = CheckStatusPass
	} else if score < 50 {
		result.Status = CheckStatusFail
	} else {
		result.Status = CheckStatusWarning
	}

	return result, nil
}

type CheckerRegistry struct {
	checkers map[RuleType]ComplianceChecker
	store    ComplianceStore
}

func NewCheckerRegistry(store ComplianceStore) *CheckerRegistry {
	registry := &CheckerRegistry{
		checkers: make(map[RuleType]ComplianceChecker),
		store:    store,
	}
	registry.Register(NewDataRetentionChecker(store))
	registry.Register(NewAccessControlChecker(store))
	registry.Register(NewEncryptionChecker(store))
	registry.Register(NewPasswordPolicyChecker(store))
	registry.Register(NewDataLocationChecker(store))
	return registry
}

func (r *CheckerRegistry) Register(checker ComplianceChecker) {
	r.checkers[checker.Type()] = checker
}

func (r *CheckerRegistry) Get(ruleType RuleType) (ComplianceChecker, bool) {
	checker, ok := r.checkers[ruleType]
	return checker, ok
}

func (r *CheckerRegistry) CheckAll(ctx context.Context, orgID string, rules []ComplianceRule) ([]*ComplianceCheckResult, error) {
	results := make([]*ComplianceCheckResult, 0, len(rules))
	for _, rule := range rules {
		checker, ok := r.checkers[rule.Type]
		if !ok {
			continue
		}
		result, err := checker.Check(ctx, orgID, rule)
		if err != nil {
			return nil, fmt.Errorf("check failed for rule %s: %w", rule.ID, err)
		}
		results = append(results, result)
	}
	return results, nil
}
