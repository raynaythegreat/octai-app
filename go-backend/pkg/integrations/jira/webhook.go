package jira

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type WebhookHandler struct {
	client        *Client
	actions       *ActionsHandler
	secret        string
	eventHandlers map[string]EventHandlerFunc
}

type EventHandlerFunc func(ctx context.Context, event *JiraWebhookEvent) error

func NewWebhookHandler(client *Client, secret string) *WebhookHandler {
	wh := &WebhookHandler{
		client:        client,
		actions:       NewActionsHandler(client),
		secret:        secret,
		eventHandlers: make(map[string]EventHandlerFunc),
	}

	wh.registerDefaultHandlers()

	return wh
}

func (wh *WebhookHandler) registerDefaultHandlers() {
	wh.RegisterHandler("jira:issue_created", wh.HandleIssueCreated)
	wh.RegisterHandler("jira:issue_updated", wh.HandleIssueUpdated)
	wh.RegisterHandler("jira:issue_deleted", wh.HandleIssueDeleted)
	wh.RegisterHandler("issue_created", wh.HandleIssueCreated)
	wh.RegisterHandler("issue_updated", wh.HandleIssueUpdated)
	wh.RegisterHandler("issue_deleted", wh.HandleIssueDeleted)
	wh.RegisterHandler("comment_created", wh.HandleCommentAdded)
	wh.RegisterHandler("comment_updated", wh.HandleCommentUpdated)
	wh.RegisterHandler("comment_deleted", wh.HandleCommentDeleted)
}

func (wh *WebhookHandler) RegisterHandler(eventType string, handler EventHandlerFunc) {
	wh.eventHandlers[eventType] = handler
}

func (wh *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !wh.VerifyQueryParams(r) {
		http.Error(w, "Invalid query parameters", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if wh.secret != "" {
		signature := r.Header.Get("X-Hub-Signature")
		if !wh.VerifySignature(body, signature) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	event, err := wh.ParseEvent(body)
	if err != nil {
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))

	go func() {
		ctx := r.Context()
		if handler, ok := wh.eventHandlers[event.Event]; ok {
			if err := handler(ctx, event); err != nil {
				fmt.Printf("Error handling Jira event %s: %v\n", event.Event, err)
			}
		}
	}()
}

func (wh *WebhookHandler) VerifyQueryParams(r *http.Request) bool {
	webhookID := r.URL.Query().Get("webhook_id")
	if webhookID == "" {
		return true
	}

	timestamp := r.URL.Query().Get("t")
	if timestamp != "" {
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return false
		}

		if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
			return false
		}
	}

	return true
}

func (wh *WebhookHandler) VerifySignature(payload []byte, signature string) bool {
	if signature == "" {
		return false
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedSig := strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(wh.secret))
	mac.Write(payload)
	actualSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(actualSig))
}

func (wh *WebhookHandler) ParseEvent(payload []byte) (*JiraWebhookEvent, error) {
	var event JiraWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &event, nil
}

func (wh *WebhookHandler) HandleIssueCreated(ctx context.Context, event *JiraWebhookEvent) error {
	if event.Issue == nil {
		return fmt.Errorf("issue_created event missing issue data")
	}

	issue := event.Issue
	fmt.Printf("Issue created: %s - %s by %s\n", issue.Key, issue.Summary, event.User.DisplayName)

	return nil
}

func (wh *WebhookHandler) HandleIssueUpdated(ctx context.Context, event *JiraWebhookEvent) error {
	if event.Issue == nil {
		return fmt.Errorf("issue_updated event missing issue data")
	}

	issue := event.Issue

	if event.Changelog != nil {
		for _, item := range event.Changelog.Items {
			fmt.Printf("Issue %s: %s changed from '%s' to '%s'\n",
				issue.Key, item.Field, item.FromString, item.ToString)
		}
	} else {
		fmt.Printf("Issue updated: %s - %s by %s\n", issue.Key, issue.Summary, event.User.DisplayName)
	}

	return nil
}

func (wh *WebhookHandler) HandleIssueDeleted(ctx context.Context, event *JiraWebhookEvent) error {
	if event.Issue == nil {
		return fmt.Errorf("issue_deleted event missing issue data")
	}

	issue := event.Issue
	fmt.Printf("Issue deleted: %s - %s by %s\n", issue.Key, issue.Summary, event.User.DisplayName)

	return nil
}

func (wh *WebhookHandler) HandleCommentAdded(ctx context.Context, event *JiraWebhookEvent) error {
	if event.Comment == nil || event.Issue == nil {
		return fmt.Errorf("comment_created event missing data")
	}

	comment := event.Comment
	issue := event.Issue

	fmt.Printf("Comment added to %s by %s: %s\n", issue.Key, comment.Author.DisplayName, truncate(comment.Body, 100))

	return nil
}

func (wh *WebhookHandler) HandleCommentUpdated(ctx context.Context, event *JiraWebhookEvent) error {
	if event.Comment == nil || event.Issue == nil {
		return fmt.Errorf("comment_updated event missing data")
	}

	comment := event.Comment
	issue := event.Issue

	fmt.Printf("Comment updated on %s by %s\n", issue.Key, comment.UpdateAuthor.DisplayName)

	return nil
}

func (wh *WebhookHandler) HandleCommentDeleted(ctx context.Context, event *JiraWebhookEvent) error {
	if event.Comment == nil || event.Issue == nil {
		return fmt.Errorf("comment_deleted event missing data")
	}

	issue := event.Issue

	fmt.Printf("Comment deleted on %s\n", issue.Key)

	return nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

type WebhookEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

func (wh *WebhookHandler) HandleEvents(ctx context.Context, events []WebhookEvent) error {
	for _, event := range events {
		var jiraEvent JiraWebhookEvent
		if err := json.Unmarshal(event.Data, &jiraEvent); err != nil {
			fmt.Printf("Failed to unmarshal event %s: %v\n", event.ID, err)
			continue
		}

		jiraEvent.Event = event.Type

		if handler, ok := wh.eventHandlers[event.Type]; ok {
			if err := handler(ctx, &jiraEvent); err != nil {
				fmt.Printf("Error handling event %s: %v\n", event.ID, err)
			}
		}
	}

	return nil
}

func (wh *WebhookHandler) ProcessBatch(ctx context.Context, payloads [][]byte) error {
	for _, payload := range payloads {
		event, err := wh.ParseEvent(payload)
		if err != nil {
			fmt.Printf("Failed to parse event: %v\n", err)
			continue
		}

		if handler, ok := wh.eventHandlers[event.Event]; ok {
			if err := handler(ctx, event); err != nil {
				fmt.Printf("Error handling event: %v\n", err)
			}
		}
	}

	return nil
}

type IssueEventPayload struct {
	WebhookID string         `json:"webhookId"`
	Event     string         `json:"webhookEvent"`
	User      JiraUser       `json:"user"`
	Issue     JiraIssue      `json:"issue"`
	Comment   *JiraComment   `json:"comment,omitempty"`
	Changelog *JiraChangelog `json:"changelog,omitempty"`
}

func (wh *WebhookHandler) ParseIssueEvent(payload []byte) (*IssueEventPayload, error) {
	var event IssueEventPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse issue event: %w", err)
	}
	return &event, nil
}

func (wh *WebhookHandler) GetClient() *Client {
	return wh.client
}

func (wh *WebhookHandler) GetActions() *ActionsHandler {
	return wh.actions
}
