package audit

import (
	"time"
)

type Action string

const (
	ActionCreate         Action = "create"
	ActionRead           Action = "read"
	ActionUpdate         Action = "update"
	ActionDelete         Action = "delete"
	ActionLogin          Action = "login"
	ActionLogout         Action = "logout"
	ActionLoginFailed    Action = "login_failed"
	ActionPasswordChange Action = "password_change"
	ActionPasswordReset  Action = "password_reset"
	ActionAPIKeyCreate   Action = "api_key_create"
	ActionAPIKeyRevoke   Action = "api_key_revoke"
	ActionExport         Action = "export"
	ActionImport         Action = "import"
	ActionShare          Action = "share"
	ActionUnshare        Action = "unshare"
	ActionArchive        Action = "archive"
	ActionRestore        Action = "restore"
	ActionAccess         Action = "access"
	ActionConfigure      Action = "configure"
	ActionEnable         Action = "enable"
	ActionDisable        Action = "disable"
)

type ResourceType string

const (
	ResourceOrganization ResourceType = "organization"
	ResourceMembership   ResourceType = "membership"
	ResourceSubscription ResourceType = "subscription"
	ResourceMessage      ResourceType = "message"
	ResourceAgent        ResourceType = "agent"
	ResourceChannel      ResourceType = "channel"
	ResourceSession      ResourceType = "session"
	ResourceAPIKey       ResourceType = "api_key"
	ResourceUser         ResourceType = "user"
	ResourceBilling      ResourceType = "billing"
	ResourceSettings     ResourceType = "settings"
	ResourceSkill        ResourceType = "skill"
	ResourceTool         ResourceType = "tool"
	ResourceWebhook      ResourceType = "webhook"
	ResourceIntegration  ResourceType = "integration"
	ResourceDocument     ResourceType = "document"
	ResourceReport       ResourceType = "report"
	ResourceAuditLog     ResourceType = "audit_log"
)

type AuditLog struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	UserID         string                 `json:"user_id"`
	Action         Action                 `json:"action"`
	ResourceType   ResourceType           `json:"resource_type"`
	ResourceID     string                 `json:"resource_id"`
	Changes        map[string]interface{} `json:"changes,omitempty"`
	IPAddress      string                 `json:"ip_address,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`
	Location       *Location              `json:"location,omitempty"`
	Status         string                 `json:"status,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

type Location struct {
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Region      string `json:"region,omitempty"`
	City        string `json:"city,omitempty"`
	Latitude    string `json:"latitude,omitempty"`
	Longitude   string `json:"longitude,omitempty"`
}

type AuditEntry struct {
	OrganizationID string
	UserID         string
	Action         Action
	ResourceType   ResourceType
	ResourceID     string
	Changes        map[string]interface{}
	IPAddress      string
	UserAgent      string
	Status         string
	ErrorMessage   string
	Metadata       map[string]interface{}
}

type QueryFilters struct {
	UserID       string
	Action       Action
	ResourceType ResourceType
	ResourceID   string
	StartDate    time.Time
	EndDate      time.Time
	Status       string
	IPAddress    string
	Limit        int
	Offset       int
	SortOrder    string
}

func (f *QueryFilters) HasDateFilter() bool {
	return !f.StartDate.IsZero() || !f.EndDate.IsZero()
}

type QueryResult struct {
	Logs    []*AuditLog `json:"logs"`
	Total   int64       `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
	HasMore bool        `json:"has_more"`
}

const (
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusDenied  = "denied"
)

var DefaultRetentionDays = 90

var EnterpriseRetentionDays = map[string]int{
	"standard":  90,
	"extended":  365,
	"unlimited": 2555,
}

type PIIField string

const (
	PIIFieldEmail      PIIField = "email"
	PIIFieldName       PIIField = "name"
	PIIFieldPhone      PIIField = "phone"
	PIIFieldAddress    PIIField = "address"
	PIIFieldPassword   PIIField = "password"
	PIIFieldAPIKey     PIIField = "api_key"
	PIIFieldToken      PIIField = "token"
	PIIFieldCreditCard PIIField = "credit_card"
	PIIFieldSSN        PIIField = "ssn"
)

var SensitiveFields = map[PIIField]bool{
	PIIFieldEmail:      true,
	PIIFieldName:       true,
	PIIFieldPhone:      true,
	PIIFieldAddress:    true,
	PIIFieldPassword:   true,
	PIIFieldAPIKey:     true,
	PIIFieldToken:      true,
	PIIFieldCreditCard: true,
	PIIFieldSSN:        true,
}
