package billing

import (
	"time"
)

type BillingInterval string

const (
	BillingIntervalMonthly BillingInterval = "monthly"
	BillingIntervalYearly  BillingInterval = "yearly"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusTrial    SubscriptionStatus = "trialing"
)

type Plan struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Price               int64           `json:"price"`
	PriceYearly         int64           `json:"price_yearly"`
	Interval            BillingInterval `json:"interval"`
	Features            []string        `json:"features"`
	Limits              PlanLimits      `json:"limits"`
	StripePriceID       string          `json:"stripe_price_id,omitempty"`
	StripePriceIDYearly string          `json:"stripe_price_id_yearly,omitempty"`
}

type PlanLimits struct {
	MaxUsers        int   `json:"max_users"`
	MaxAgents       int   `json:"max_agents"`
	MaxMessages     int   `json:"max_messages"`
	MaxChannels     int   `json:"max_channels"`
	MaxStorageBytes int64 `json:"max_storage_bytes"`
}

type Subscription struct {
	ID                   string             `json:"id"`
	OrgID                string             `json:"org_id"`
	PlanID               string             `json:"plan_id"`
	Status               SubscriptionStatus `json:"status"`
	Interval             BillingInterval    `json:"interval"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool               `json:"cancel_at_period_end"`
	StripeSubscriptionID string             `json:"stripe_subscription_id,omitempty"`
	StripeCustomerID     string             `json:"stripe_customer_id,omitempty"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
	CanceledAt           *time.Time         `json:"canceled_at,omitempty"`
}

type Invoice struct {
	ID              string     `json:"id"`
	OrgID           string     `json:"org_id"`
	SubscriptionID  string     `json:"subscription_id,omitempty"`
	Amount          int64      `json:"amount"`
	Currency        string     `json:"currency"`
	Status          string     `json:"status"`
	DueDate         time.Time  `json:"due_date"`
	PaidDate        *time.Time `json:"paid_date,omitempty"`
	StripeInvoiceID string     `json:"stripe_invoice_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type UsageSummary struct {
	Messages     int64   `json:"messages"`
	Tokens       int64   `json:"tokens"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	Storage      int64   `json:"storage_bytes"`
	Overages     Overage `json:"overages"`
}

type Overage struct {
	Messages int64 `json:"messages"`
	Tokens   int64 `json:"tokens"`
	Storage  int64 `json:"storage_bytes"`
}

type UsageEvent struct {
	ID        string                 `json:"id"`
	OrgID     string                 `json:"org_id"`
	EventType string                 `json:"event_type"`
	Quantity  int64                  `json:"quantity"`
	AgentID   *string                `json:"agent_id,omitempty"`
	Channel   *string                `json:"channel,omitempty"`
	Model     *string                `json:"model,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type CheckoutSession struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Customer string `json:"customer,omitempty"`
}

type BillingPortalSession struct {
	URL string `json:"url"`
}
