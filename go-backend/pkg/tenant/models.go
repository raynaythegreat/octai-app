package tenant

import (
	"errors"
	"regexp"
	"time"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
	RoleViewer Role = "viewer"
)

func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierBusiness   Tier = "business"
	TierEnterprise Tier = "enterprise"
)

func (t Tier) Valid() bool {
	switch t {
	case TierFree, TierPro, TierBusiness, TierEnterprise:
		return true
	default:
		return false
	}
}

type SubscriptionStatus string

const (
	SubStatusActive     SubscriptionStatus = "active"
	SubStatusPastDue    SubscriptionStatus = "past_due"
	SubStatusCanceled   SubscriptionStatus = "canceled"
	SubStatusIncomplete SubscriptionStatus = "incomplete"
	SubStatusTrialing   SubscriptionStatus = "trialing"
)

func (s SubscriptionStatus) Valid() bool {
	switch s {
	case SubStatusActive, SubStatusPastDue, SubStatusCanceled, SubStatusIncomplete, SubStatusTrialing:
		return true
	default:
		return false
	}
}

type Organization struct {
	ID               string     `json:"id"`
	Slug             string     `json:"slug"`
	Name             string     `json:"name"`
	LogoURL          *string    `json:"logo_url,omitempty"`
	Settings         JSONB      `json:"settings,omitempty"`
	StripeCustomerID *string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

type Membership struct {
	ID             string     `json:"id"`
	OrganizationID string     `json:"organization_id"`
	UserID         string     `json:"user_id"`
	Role           Role       `json:"role"`
	InvitedBy      *string    `json:"invited_by,omitempty"`
	InvitedAt      *time.Time `json:"invited_at,omitempty"`
	AcceptedAt     *time.Time `json:"accepted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Subscription struct {
	ID                   string             `json:"id"`
	OrganizationID       string             `json:"organization_id"`
	StripeSubscriptionID *string            `json:"stripe_subscription_id,omitempty"`
	StripePriceID        *string            `json:"stripe_price_id,omitempty"`
	Status               SubscriptionStatus `json:"status"`
	Tier                 Tier               `json:"tier"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
}

type UsageRecord struct {
	ID             string                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	EventType      string                 `json:"event_type"`
	Quantity       int64                  `json:"quantity"`
	AgentID        *string                `json:"agent_id,omitempty"`
	Channel        *string                `json:"channel,omitempty"`
	Model          *string                `json:"model,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type JSONB map[string]interface{}

type TierLimits struct {
	MaxUsers        int
	MaxAgents       int
	MaxMessages     int
	MaxChannels     int
	MaxStorageBytes int64
	Features        []string
}

var TierConfig = map[Tier]TierLimits{
	TierFree: {
		MaxUsers:        1,
		MaxAgents:       1,
		MaxMessages:     500,
		MaxChannels:     2,
		MaxStorageBytes: 100 * 1024 * 1024,
		Features:        []string{"basic_skills"},
	},
	TierPro: {
		MaxUsers:        5,
		MaxAgents:       5,
		MaxMessages:     10000,
		MaxChannels:     5,
		MaxStorageBytes: 5 * 1024 * 1024 * 1024,
		Features:        []string{"basic_skills", "custom_skills"},
	},
	TierBusiness: {
		MaxUsers:        25,
		MaxAgents:       20,
		MaxMessages:     100000,
		MaxChannels:     15,
		MaxStorageBytes: 50 * 1024 * 1024 * 1024,
		Features:        []string{"basic_skills", "custom_skills", "marketplace", "sso", "audit_logs"},
	},
	TierEnterprise: {
		MaxUsers:        -1,
		MaxAgents:       -1,
		MaxMessages:     -1,
		MaxChannels:     -1,
		MaxStorageBytes: -1,
		Features:        []string{"basic_skills", "custom_skills", "marketplace", "sso", "audit_logs", "dedicated_support", "sla"},
	},
}

type Permission string

const (
	PermOrgRead       Permission = "org:read"
	PermOrgWrite      Permission = "org:write"
	PermOrgDelete     Permission = "org:delete"
	PermBillingManage Permission = "billing:manage"
	PermMembersManage Permission = "members:manage"
	PermAgentsManage  Permission = "agents:manage"
	PermAgentsUse     Permission = "agents:use"
	PermSessionsAll   Permission = "sessions:all"
	PermChannelsCfg   Permission = "channels:configure"
	PermAnalytics     Permission = "analytics:view"
)

var RolePermissions = map[Role][]Permission{
	RoleOwner: {
		PermOrgRead, PermOrgWrite, PermOrgDelete, PermBillingManage,
		PermMembersManage, PermAgentsManage, PermAgentsUse,
		PermSessionsAll, PermChannelsCfg, PermAnalytics,
	},
	RoleAdmin: {
		PermOrgRead, PermOrgWrite,
		PermMembersManage, PermAgentsManage, PermAgentsUse,
		PermSessionsAll, PermChannelsCfg, PermAnalytics,
	},
	RoleMember: {
		PermOrgRead, PermAgentsManage, PermAgentsUse, PermAnalytics,
	},
	RoleViewer: {
		PermOrgRead, PermAnalytics,
	},
}

var (
	ErrInvalidSlug          = errors.New("invalid slug format")
	ErrInvalidRole          = errors.New("invalid role")
	ErrInvalidTier          = errors.New("invalid tier")
	ErrOrgNotFound          = errors.New("organization not found")
	ErrMembershipNotFound   = errors.New("membership not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrAlreadyMember        = errors.New("user is already a member")
	ErrCannotRemoveOwner    = errors.New("cannot remove the organization owner")
	ErrInsufficientPerms    = errors.New("insufficient permissions")
	ErrLimitExceeded        = errors.New("tier limit exceeded")
)

var slugRegex = regexp.MustCompile(`^[a-z0-9](-?[a-z0-9])*$`)

func ValidateSlug(slug string) error {
	if len(slug) < 3 || len(slug) > 63 {
		return ErrInvalidSlug
	}
	if !slugRegex.MatchString(slug) {
		return ErrInvalidSlug
	}
	return nil
}

type TenantContext struct {
	OrganizationID   string
	OrganizationSlug string
	UserID           string
	Role             Role
	SubscriptionTier Tier
}

func (t *TenantContext) CanAccessResource(resourceOrgID string) bool {
	return t.OrganizationID == resourceOrgID
}

func (t *TenantContext) HasPermission(perm Permission) bool {
	perms, ok := RolePermissions[t.Role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

func (t *TenantContext) GetLimits() TierLimits {
	limits, ok := TierConfig[t.SubscriptionTier]
	if !ok {
		return TierConfig[TierFree]
	}
	return limits
}

func (t *TenantContext) HasFeature(feature string) bool {
	limits := t.GetLimits()
	for _, f := range limits.Features {
		if f == feature {
			return true
		}
	}
	return false
}
