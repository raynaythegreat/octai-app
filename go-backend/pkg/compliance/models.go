package compliance

import (
	"encoding/json"
	"time"
)

type RuleType string

const (
	RuleTypeDataRetention  RuleType = "data_retention"
	RuleTypeAccessControl  RuleType = "access_control"
	RuleTypeEncryption     RuleType = "encryption"
	RuleTypePasswordPolicy RuleType = "password_policy"
	RuleTypeDataLocation   RuleType = "data_location"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

func (s Severity) Valid() bool {
	switch s {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}

func (s Severity) Weight() int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	default:
		return 1
	}
}

type ComplianceFramework string

const (
	FrameworkSOC2  ComplianceFramework = "soc2"
	FrameworkHIPAA ComplianceFramework = "hipaa"
	FrameworkGDPR  ComplianceFramework = "gdpr"
)

type ComplianceRule struct {
	ID             string                `json:"id"`
	OrganizationID string                `json:"organization_id"`
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	Type           RuleType              `json:"type"`
	Severity       Severity              `json:"severity"`
	Config         json.RawMessage       `json:"config,omitempty"`
	Frameworks     []ComplianceFramework `json:"frameworks,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at,omitempty"`
}

type CheckStatus string

const (
	CheckStatusPass    CheckStatus = "pass"
	CheckStatusFail    CheckStatus = "fail"
	CheckStatusWarning CheckStatus = "warning"
	CheckStatusSkipped CheckStatus = "skipped"
)

func (s CheckStatus) Valid() bool {
	switch s {
	case CheckStatusPass, CheckStatusFail, CheckStatusWarning, CheckStatusSkipped:
		return true
	default:
		return false
	}
}

type Finding struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    Severity               `json:"severity"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Resource    string                 `json:"resource,omitempty"`
}

type ComplianceCheckResult struct {
	ID              string      `json:"id"`
	RuleID          string      `json:"rule_id"`
	OrganizationID  string      `json:"organization_id"`
	Status          CheckStatus `json:"status"`
	Findings        []Finding   `json:"findings,omitempty"`
	Score           int         `json:"score"`
	MaxScore        int         `json:"max_score"`
	CheckedAt       time.Time   `json:"checked_at"`
	ResolvedAt      *time.Time  `json:"resolved_at,omitempty"`
	ResolutionNotes string      `json:"resolution_notes,omitempty"`
}

func (r *ComplianceCheckResult) IsResolved() bool {
	return r.ResolvedAt != nil
}

func (r *ComplianceCheckResult) HasFindings() bool {
	return len(r.Findings) > 0
}

func (r *ComplianceCheckResult) CriticalFindings() []Finding {
	var critical []Finding
	for _, f := range r.Findings {
		if f.Severity == SeverityCritical {
			critical = append(critical, f)
		}
	}
	return critical
}

func (r *ComplianceCheckResult) WarningFindings() []Finding {
	var warnings []Finding
	for _, f := range r.Findings {
		if f.Severity == SeverityWarning {
			warnings = append(warnings, f)
		}
	}
	return warnings
}

type DataRetentionConfig struct {
	Enabled          bool `json:"enabled"`
	RetentionDays    int  `json:"retention_days"`
	IncludeMessages  bool `json:"include_messages"`
	IncludeAuditLogs bool `json:"include_audit_logs"`
	AutoDelete       bool `json:"auto_delete"`
}

type AccessControlConfig struct {
	RBACEnabled       bool     `json:"rbac_enabled"`
	MFARequired       bool     `json:"mfa_required"`
	SSOEnabled        bool     `json:"sso_enabled"`
	AllowedRoles      []string `json:"allowed_roles,omitempty"`
	SessionTimeoutMin int      `json:"session_timeout_min"`
}

type EncryptionConfig struct {
	AtRestEnabled    bool   `json:"at_rest_enabled"`
	InTransitEnabled bool   `json:"in_transit_enabled"`
	KeyRotationDays  int    `json:"key_rotation_days"`
	Algorithm        string `json:"algorithm,omitempty"`
}

type PasswordPolicyConfig struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSymbols   bool `json:"require_symbols"`
	MaxAgeDays       int  `json:"max_age_days"`
	HistoryCount     int  `json:"history_count"`
}

type DataLocationConfig struct {
	PrimaryRegion  string   `json:"primary_region"`
	AllowedRegions []string `json:"allowed_regions,omitempty"`
	DRRegion       string   `json:"dr_region,omitempty"`
	DataResidency  bool     `json:"data_residency"`
}

func DefaultSOC2Rules() []ComplianceRule {
	return []ComplianceRule{
		{
			ID:          "soc2-data-retention",
			Name:        "SOC 2 Data Retention Policy",
			Description: "Ensure data retention policy is configured and enforced",
			Type:        RuleTypeDataRetention,
			Severity:    SeverityWarning,
			Config:      json.RawMessage(`{"retention_days": 365}`),
			Frameworks:  []ComplianceFramework{FrameworkSOC2},
		},
		{
			ID:          "soc2-access-control",
			Name:        "SOC 2 Access Control",
			Description: "Verify RBAC is configured with appropriate permissions",
			Type:        RuleTypeAccessControl,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"rbac_enabled": true, "mfa_required": true}`),
			Frameworks:  []ComplianceFramework{FrameworkSOC2},
		},
		{
			ID:          "soc2-encryption",
			Name:        "SOC 2 Encryption at Rest",
			Description: "Validate data encryption at rest",
			Type:        RuleTypeEncryption,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"at_rest_enabled": true, "in_transit_enabled": true}`),
			Frameworks:  []ComplianceFramework{FrameworkSOC2},
		},
	}
}

func DefaultHIPAARules() []ComplianceRule {
	return []ComplianceRule{
		{
			ID:          "hipaa-data-retention",
			Name:        "HIPAA Data Retention",
			Description: "Healthcare data must be retained for 6 years minimum",
			Type:        RuleTypeDataRetention,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"retention_days": 2190, "auto_delete": false}`),
			Frameworks:  []ComplianceFramework{FrameworkHIPAA},
		},
		{
			ID:          "hipaa-encryption",
			Name:        "HIPAA Encryption Requirements",
			Description: "PHI must be encrypted at rest and in transit",
			Type:        RuleTypeEncryption,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"at_rest_enabled": true, "in_transit_enabled": true, "algorithm": "AES-256"}`),
			Frameworks:  []ComplianceFramework{FrameworkHIPAA},
		},
		{
			ID:          "hipaa-access-control",
			Name:        "HIPAA Access Control",
			Description: "Access to PHI must be strictly controlled and audited",
			Type:        RuleTypeAccessControl,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"rbac_enabled": true, "mfa_required": true, "session_timeout_min": 30}`),
			Frameworks:  []ComplianceFramework{FrameworkHIPAA},
		},
		{
			ID:          "hipaa-data-location",
			Name:        "HIPAA Data Location",
			Description: "PHI must be stored in compliant regions",
			Type:        RuleTypeDataLocation,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"data_residency": true, "allowed_regions": ["us-east-1", "us-west-2"]}`),
			Frameworks:  []ComplianceFramework{FrameworkHIPAA},
		},
	}
}

func DefaultGDPRRules() []ComplianceRule {
	return []ComplianceRule{
		{
			ID:          "gdpr-data-retention",
			Name:        "GDPR Data Retention",
			Description: "Personal data must not be kept longer than necessary",
			Type:        RuleTypeDataRetention,
			Severity:    SeverityWarning,
			Config:      json.RawMessage(`{"retention_days": 90, "auto_delete": true}`),
			Frameworks:  []ComplianceFramework{FrameworkGDPR},
		},
		{
			ID:          "gdpr-data-location",
			Name:        "GDPR Data Location",
			Description: "Personal data must remain in EU or approved regions",
			Type:        RuleTypeDataLocation,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"data_residency": true, "allowed_regions": ["eu-west-1", "eu-central-1"]}`),
			Frameworks:  []ComplianceFramework{FrameworkGDPR},
		},
		{
			ID:          "gdpr-encryption",
			Name:        "GDPR Encryption",
			Description: "Personal data must be protected with appropriate encryption",
			Type:        RuleTypeEncryption,
			Severity:    SeverityCritical,
			Config:      json.RawMessage(`{"at_rest_enabled": true, "in_transit_enabled": true}`),
			Frameworks:  []ComplianceFramework{FrameworkGDPR},
		},
	}
}
