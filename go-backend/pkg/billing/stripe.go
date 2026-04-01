package billing

import (
	"context"
	"errors"
	"time"
)

var (
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrInvoiceNotFound       = errors.New("invoice not found")
	ErrWebhookVerification   = errors.New("webhook verification failed")
	ErrInvalidWebhookPayload = errors.New("invalid webhook payload")
)

type StripeClient interface {
	CreateCustomer(ctx context.Context, email, orgID string) (*Customer, error)
	GetCustomer(ctx context.Context, customerID string) (*Customer, error)
	UpdateCustomer(ctx context.Context, customerID string, params map[string]interface{}) error

	CreateSubscription(ctx context.Context, customerID, priceID string, interval BillingInterval) (*SubscriptionResult, error)
	GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionResult, error)
	CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error
	UpdateSubscription(ctx context.Context, subscriptionID string, newPriceID string) error

	GetInvoice(ctx context.Context, invoiceID string) (*InvoiceResult, error)
	ListInvoices(ctx context.Context, customerID string, limit int) ([]InvoiceResult, error)

	CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (*CheckoutSessionResult, error)
	CreatePortalSession(ctx context.Context, customerID, returnUrl string) (*BillingPortalSession, error)

	HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
}

type Customer struct {
	ID       string            `json:"id"`
	Email    string            `json:"email"`
	Metadata map[string]string `json:"metadata"`
}

type SubscriptionResult struct {
	ID                 string             `json:"id"`
	CustomerID         string             `json:"customer_id"`
	Status             SubscriptionStatus `json:"status"`
	PriceID            string             `json:"price_id"`
	CurrentPeriodStart int64              `json:"current_period_start"`
	CurrentPeriodEnd   int64              `json:"current_period_end"`
	CancelAtPeriodEnd  bool               `json:"cancel_at_period_end"`
}

type InvoiceResult struct {
	ID             string `json:"id"`
	CustomerID     string `json:"customer_id"`
	SubscriptionID string `json:"subscription_id"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	DueDate        int64  `json:"due_date"`
	PaidDate       int64  `json:"paid_date"`
}

type CheckoutSessionResult struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Customer string `json:"customer"`
}

type WebhookEvent struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Created int64       `json:"created"`
}

type MockStripeClient struct {
	customers     map[string]*Customer
	subscriptions map[string]*SubscriptionResult
	invoices      map[string]*InvoiceResult
}

func NewMockStripeClient() *MockStripeClient {
	return &MockStripeClient{
		customers:     make(map[string]*Customer),
		subscriptions: make(map[string]*SubscriptionResult),
		invoices:      make(map[string]*InvoiceResult),
	}
}

func (m *MockStripeClient) CreateCustomer(ctx context.Context, email, orgID string) (*Customer, error) {
	id := "cus_" + generateID()
	customer := &Customer{
		ID:    id,
		Email: email,
		Metadata: map[string]string{
			"organization_id": orgID,
		},
	}
	m.customers[id] = customer
	return customer, nil
}

func (m *MockStripeClient) GetCustomer(ctx context.Context, customerID string) (*Customer, error) {
	customer, ok := m.customers[customerID]
	if !ok {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

func (m *MockStripeClient) UpdateCustomer(ctx context.Context, customerID string, params map[string]interface{}) error {
	customer, ok := m.customers[customerID]
	if !ok {
		return ErrCustomerNotFound
	}
	if email, ok := params["email"].(string); ok {
		customer.Email = email
	}
	return nil
}

func (m *MockStripeClient) CreateSubscription(ctx context.Context, customerID, priceID string, interval BillingInterval) (*SubscriptionResult, error) {
	if _, ok := m.customers[customerID]; !ok {
		return nil, ErrCustomerNotFound
	}

	now := time.Now().Unix()
	var periodEnd int64
	if interval == BillingIntervalYearly {
		periodEnd = now + 365*24*60*60
	} else {
		periodEnd = now + 30*24*60*60
	}

	id := "sub_" + generateID()
	sub := &SubscriptionResult{
		ID:                 id,
		CustomerID:         customerID,
		Status:             SubscriptionStatusActive,
		PriceID:            priceID,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		CancelAtPeriodEnd:  false,
	}
	m.subscriptions[id] = sub
	return sub, nil
}

func (m *MockStripeClient) GetSubscription(ctx context.Context, subscriptionID string) (*SubscriptionResult, error) {
	sub, ok := m.subscriptions[subscriptionID]
	if !ok {
		return nil, ErrSubscriptionNotFound
	}
	return sub, nil
}

func (m *MockStripeClient) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	sub, ok := m.subscriptions[subscriptionID]
	if !ok {
		return ErrSubscriptionNotFound
	}

	if immediately {
		sub.Status = SubscriptionStatusCanceled
	} else {
		sub.CancelAtPeriodEnd = true
	}
	return nil
}

func (m *MockStripeClient) UpdateSubscription(ctx context.Context, subscriptionID string, newPriceID string) error {
	sub, ok := m.subscriptions[subscriptionID]
	if !ok {
		return ErrSubscriptionNotFound
	}
	sub.PriceID = newPriceID
	return nil
}

func (m *MockStripeClient) GetInvoice(ctx context.Context, invoiceID string) (*InvoiceResult, error) {
	inv, ok := m.invoices[invoiceID]
	if !ok {
		return nil, ErrInvoiceNotFound
	}
	return inv, nil
}

func (m *MockStripeClient) ListInvoices(ctx context.Context, customerID string, limit int) ([]InvoiceResult, error) {
	var results []InvoiceResult
	for _, inv := range m.invoices {
		if inv.CustomerID == customerID {
			results = append(results, *inv)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *MockStripeClient) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (*CheckoutSessionResult, error) {
	if _, ok := m.customers[customerID]; !ok {
		return nil, ErrCustomerNotFound
	}

	id := "cs_" + generateID()
	return &CheckoutSessionResult{
		ID:       id,
		URL:      "https://checkout.stripe.com/mock/" + id,
		Customer: customerID,
	}, nil
}

func (m *MockStripeClient) CreatePortalSession(ctx context.Context, customerID, returnUrl string) (*BillingPortalSession, error) {
	if _, ok := m.customers[customerID]; !ok {
		return nil, ErrCustomerNotFound
	}

	return &BillingPortalSession{
		URL: "https://billing.stripe.com/mock/" + customerID,
	}, nil
}

func (m *MockStripeClient) HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	if signature == "" {
		return nil, ErrWebhookVerification
	}

	return &WebhookEvent{
		ID:      "evt_" + generateID(),
		Type:    "mock.event",
		Data:    payload,
		Created: time.Now().Unix(),
	}, nil
}

type StripeService struct {
	client StripeClient
	store  BillingStore
}

type BillingStore interface {
	GetSubscription(ctx context.Context, orgID string) (*Subscription, error)
	CreateSubscription(ctx context.Context, sub *Subscription) error
	UpdateSubscription(ctx context.Context, sub *Subscription) error
	GetOrganization(ctx context.Context, orgID string) (*OrganizationInfo, error)
	UpdateStripeCustomerID(ctx context.Context, orgID, customerID string) error
	CreateInvoice(ctx context.Context, invoice *Invoice) error
}

type OrganizationInfo struct {
	ID               string
	StripeCustomerID *string
}

func NewStripeService(client StripeClient, store BillingStore) *StripeService {
	return &StripeService{
		client: client,
		store:  store,
	}
}

func (s *StripeService) CreateCustomer(ctx context.Context, email, orgID string) (string, error) {
	customer, err := s.client.CreateCustomer(ctx, email, orgID)
	if err != nil {
		return "", err
	}

	if err := s.store.UpdateStripeCustomerID(ctx, orgID, customer.ID); err != nil {
		return "", err
	}

	return customer.ID, nil
}

func (s *StripeService) CreateSubscription(ctx context.Context, customerID, priceID string, interval BillingInterval, orgID string) (*Subscription, error) {
	result, err := s.client.CreateSubscription(ctx, customerID, priceID, interval)
	if err != nil {
		return nil, err
	}

	periodStart := time.Unix(result.CurrentPeriodStart, 0)
	periodEnd := time.Unix(result.CurrentPeriodEnd, 0)

	planID := priceToPlanID(priceID)

	sub := &Subscription{
		ID:                   generateID(),
		OrgID:                orgID,
		PlanID:               planID,
		Status:               result.Status,
		Interval:             interval,
		CurrentPeriodStart:   &periodStart,
		CurrentPeriodEnd:     &periodEnd,
		StripeSubscriptionID: result.ID,
		StripeCustomerID:     customerID,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	if err := s.store.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *StripeService) CancelSubscription(ctx context.Context, subscriptionID string, immediately bool) error {
	return s.client.CancelSubscription(ctx, subscriptionID, immediately)
}

func (s *StripeService) GetInvoice(ctx context.Context, invoiceID string) (*InvoiceResult, error) {
	return s.client.GetInvoice(ctx, invoiceID)
}

func (s *StripeService) HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	return s.client.HandleWebhook(ctx, payload, signature)
}

func (s *StripeService) CreateCheckoutSession(ctx context.Context, orgID, priceID, successURL, cancelURL string) (*CheckoutSession, error) {
	org, err := s.store.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var customerID string
	if org.StripeCustomerID != nil {
		customerID = *org.StripeCustomerID
	}

	if customerID == "" {
		return nil, ErrCustomerNotFound
	}

	result, err := s.client.CreateCheckoutSession(ctx, customerID, priceID, successURL, cancelURL)
	if err != nil {
		return nil, err
	}

	return &CheckoutSession{
		ID:       result.ID,
		URL:      result.URL,
		Customer: result.Customer,
	}, nil
}

func (s *StripeService) CreatePortalSession(ctx context.Context, orgID, returnUrl string) (*BillingPortalSession, error) {
	org, err := s.store.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if org.StripeCustomerID == nil {
		return nil, ErrCustomerNotFound
	}

	return s.client.CreatePortalSession(ctx, *org.StripeCustomerID, returnUrl)
}

func priceToPlanID(priceID string) string {
	switch {
	case contains(priceID, "pro"):
		return PlanPro
	case contains(priceID, "business"):
		return PlanBusiness
	case contains(priceID, "enterprise"):
		return PlanEnterprise
	default:
		return PlanFree
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func generateID() string {
	return time.Now().Format("20060102150405")
}
