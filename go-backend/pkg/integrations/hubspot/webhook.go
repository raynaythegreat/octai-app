package hubspot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type EventHandlerFunc func(ctx context.Context, event *WebhookEvent) error

type WebhookHandler struct {
	client        *Client
	syncService   *SyncService
	clientSecret  string
	eventHandlers map[string]EventHandlerFunc
	mu            sync.RWMutex
}

func NewWebhookHandler(client *Client, syncService *SyncService, clientSecret string) *WebhookHandler {
	wh := &WebhookHandler{
		client:        client,
		syncService:   syncService,
		clientSecret:  clientSecret,
		eventHandlers: make(map[string]EventHandlerFunc),
	}

	wh.registerDefaultHandlers()

	return wh
}

func (wh *WebhookHandler) registerDefaultHandlers() {
	wh.RegisterHandler("contact.creation", wh.HandleContactCreated)
	wh.RegisterHandler("contact.deletion", wh.HandleContactDeleted)
	wh.RegisterHandler("contact.propertyChange", wh.HandleContactUpdated)
	wh.RegisterHandler("deal.creation", wh.HandleDealCreated)
	wh.RegisterHandler("deal.deletion", wh.HandleDealDeleted)
	wh.RegisterHandler("deal.propertyChange", wh.HandleDealUpdated)
}

func (wh *WebhookHandler) RegisterHandler(eventType string, handler EventHandlerFunc) {
	wh.mu.Lock()
	defer wh.mu.Unlock()
	wh.eventHandlers[eventType] = handler
}

func (wh *WebhookHandler) HandleWebhook(ctx context.Context, payload []byte) error {
	var events []WebhookEvent
	if err := json.Unmarshal(payload, &events); err != nil {
		return fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	for _, event := range events {
		if handler := wh.getHandler(event.SubscriptionType); handler != nil {
			if err := handler(ctx, &event); err != nil {
				fmt.Printf("error handling webhook event %s: %v\n", event.SubscriptionType, err)
				continue
			}
		}
	}

	return nil
}

func (wh *WebhookHandler) getHandler(eventType string) EventHandlerFunc {
	wh.mu.RLock()
	defer wh.mu.RUnlock()
	return wh.eventHandlers[eventType]
}

func (wh *WebhookHandler) VerifySignature(payload []byte, signature string) bool {
	if signature == "" || wh.clientSecret == "" {
		return false
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSig := strings.TrimPrefix(signature, "sha256=")

	hash := sha256.New()
	hash.Write(payload)
	actualSig := hex.EncodeToString(hash.Sum(nil))

	return expectedSig == actualSig
}

func (wh *WebhookHandler) VerifySignatureV2(payload []byte, signature string, timestamp string, maxAge time.Duration) bool {
	if signature == "" || timestamp == "" || wh.clientSecret == "" {
		return false
	}

	ts, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		if tsInt, err := time.ParseDuration(timestamp + "s"); err == nil {
			ts = time.Now().Add(-tsInt)
		} else {
			return false
		}
	}

	if time.Since(ts) > maxAge {
		return false
	}

	return wh.VerifySignature(payload, signature)
}

func (wh *WebhookHandler) HandleContactCreated(ctx context.Context, event *WebhookEvent) error {
	contactID := event.ObjectID
	if contactID == "" {
		return fmt.Errorf("contact creation event missing object ID")
	}

	if wh.syncService != nil {
		fmt.Printf("Contact created: %s (event ID: %s)\n", contactID, event.EventID)
	}

	return nil
}

func (wh *WebhookHandler) HandleContactDeleted(ctx context.Context, event *WebhookEvent) error {
	contactID := event.ObjectID
	if contactID == "" {
		return fmt.Errorf("contact deletion event missing object ID")
	}

	fmt.Printf("Contact deleted: %s (event ID: %s)\n", contactID, event.EventID)

	return nil
}

func (wh *WebhookHandler) HandleContactUpdated(ctx context.Context, event *WebhookEvent) error {
	contactID := event.ObjectID
	if contactID == "" {
		return fmt.Errorf("contact update event missing object ID")
	}

	property := event.PropertyName
	oldValue := event.OldValue
	newValue := event.NewValue

	fmt.Printf("Contact updated: %s - %s changed from '%s' to '%s'\n", contactID, property, oldValue, newValue)

	return nil
}

func (wh *WebhookHandler) HandleDealCreated(ctx context.Context, event *WebhookEvent) error {
	dealID := event.ObjectID
	if dealID == "" {
		return fmt.Errorf("deal creation event missing object ID")
	}

	if wh.syncService != nil {
		fmt.Printf("Deal created: %s (event ID: %s)\n", dealID, event.EventID)
	}

	return nil
}

func (wh *WebhookHandler) HandleDealDeleted(ctx context.Context, event *WebhookEvent) error {
	dealID := event.ObjectID
	if dealID == "" {
		return fmt.Errorf("deal deletion event missing object ID")
	}

	fmt.Printf("Deal deleted: %s (event ID: %s)\n", dealID, event.EventID)

	return nil
}

func (wh *WebhookHandler) HandleDealUpdated(ctx context.Context, event *WebhookEvent) error {
	dealID := event.ObjectID
	if dealID == "" {
		return fmt.Errorf("deal update event missing object ID")
	}

	property := event.PropertyName
	oldValue := event.OldValue
	newValue := event.NewValue

	fmt.Printf("Deal updated: %s - %s changed from '%s' to '%s'\n", dealID, property, oldValue, newValue)

	if property == "dealstage" {
		return wh.HandleDealStageChanged(ctx, event)
	}

	return nil
}

func (wh *WebhookHandler) HandleDealStageChanged(ctx context.Context, event *WebhookEvent) error {
	dealID := event.ObjectID
	oldStage := event.OldValue
	newStage := event.NewValue

	fmt.Printf("Deal stage changed: %s - '%s' -> '%s'\n", dealID, oldStage, newStage)

	return nil
}

func (wh *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	signature := r.Header.Get("X-HubSpot-Signature")
	if signature == "" {
		signature = r.Header.Get("X-HubSpot-Signature-v3")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if wh.clientSecret != "" && !wh.VerifySignature(body, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	if err := wh.HandleWebhook(ctx, body); err != nil {
		fmt.Printf("error handling webhook: %v\n", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (wh *WebhookHandler) ParseEvent(payload []byte) ([]WebhookEvent, error) {
	var events []WebhookEvent
	if err := json.Unmarshal(payload, &events); err != nil {
		return nil, fmt.Errorf("failed to parse webhook events: %w", err)
	}
	return events, nil
}

func (wh *WebhookHandler) ProcessBatch(ctx context.Context, payloads [][]byte) error {
	for _, payload := range payloads {
		if err := wh.HandleWebhook(ctx, payload); err != nil {
			fmt.Printf("error processing batch payload: %v\n", err)
			continue
		}
	}
	return nil
}

type WebhookSubscription struct {
	ID              string `json:"id"`
	EventType       string `json:"eventType"`
	PropertyName    string `json:"propertyName,omitempty"`
	Active          bool   `json:"active"`
	CreationDetails struct {
		CreatedAt       int64  `json:"createdAt"`
		CreatedByUserID string `json:"createdByUserId"`
	} `json:"creationDetails"`
}

type WebhookSubscriptionRequest struct {
	EventType    string `json:"eventType"`
	PropertyName string `json:"propertyName,omitempty"`
	TargetURL    string `json:"targetUrl"`
}

func (c *Client) GetWebhookSubscriptions(ctx context.Context) ([]WebhookSubscription, error) {
	resp, err := c.doRequest(ctx, "GET", "/webhooks/v3/app/subscriptions", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		Results []WebhookSubscription `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode subscriptions: %w", err)
	}

	return result.Results, nil
}

func (c *Client) CreateWebhookSubscription(ctx context.Context, req *WebhookSubscriptionRequest) (*WebhookSubscription, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", "/webhooks/v3/app/subscriptions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var subscription WebhookSubscription
	if err := json.NewDecoder(resp.Body).Decode(&subscription); err != nil {
		return nil, fmt.Errorf("failed to decode subscription: %w", err)
	}

	return &subscription, nil
}

func (c *Client) UpdateWebhookSubscription(ctx context.Context, subscriptionID string, active bool) error {
	req := map[string]bool{"active": active}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "PATCH", fmt.Sprintf("/webhooks/v3/app/subscriptions/%s", subscriptionID), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) DeleteWebhookSubscription(ctx context.Context, subscriptionID string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/webhooks/v3/app/subscriptions/%s", subscriptionID), nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

type WebhookSettings struct {
	TargetURL             string            `json:"targetUrl"`
	Throttling            *ThrottlingConfig `json:"throttling,omitempty"`
	Concurrency           int               `json:"concurrency,omitempty"`
	MaxConcurrentRequests int               `json:"maxConcurrentRequests,omitempty"`
}

type ThrottlingConfig struct {
	MaxAtOnce    int `json:"maxAtOnce"`
	PeriodMillis int `json:"periodMillis"`
	MaxPerSecond int `json:"maxPerSecond,omitempty"`
}

func (c *Client) GetWebhookSettings(ctx context.Context) (*WebhookSettings, error) {
	resp, err := c.doRequest(ctx, "GET", "/webhooks/v3/app/settings", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var settings WebhookSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("failed to decode settings: %w", err)
	}

	return &settings, nil
}

func (c *Client) ConfigureWebhooks(ctx context.Context, settings *WebhookSettings) error {
	body, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "PUT", "/webhooks/v3/app/settings", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}
