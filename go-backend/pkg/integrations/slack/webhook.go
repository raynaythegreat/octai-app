package slack

import (
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
	signingSecret string
	handler       *Handler
	registry      *HandlerRegistry
}

func NewWebhookHandler(client *Client, registry *HandlerRegistry) *WebhookHandler {
	return &WebhookHandler{
		client:        client,
		signingSecret: client.signingSecret,
		handler:       NewHandler(client),
		registry:      registry,
	}
}

func (wh *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	if !wh.VerifySignature(body, timestamp, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := wh.ParseEvent(body)
	if err != nil {
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	if event.Type == "url_verification" {
		wh.handleURLVerification(w, body)
		return
	}

	w.WriteHeader(http.StatusOK)

	go func() {
		ctx := r.Context()
		if err := wh.handler.HandleEvent(ctx, *event); err != nil {
			fmt.Printf("Error handling event: %v\n", err)
		}
	}()
}

func (wh *WebhookHandler) VerifySignature(payload []byte, timestamp, signature string) bool {
	if timestamp == "" || signature == "" {
		return false
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return false
	}

	if !strings.HasPrefix(signature, "v0=") {
		return false
	}

	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(wh.signingSecret))
	mac.Write([]byte(baseString))
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (wh *WebhookHandler) ParseEvent(payload []byte) (*SlackEvent, error) {
	var rawEvent struct {
		Type           string          `json:"type"`
		Token          string          `json:"token"`
		Challenge      string          `json:"challenge"`
		TeamID         string          `json:"team_id"`
		APIAppID       string          `json:"api_app_id"`
		Event          json.RawMessage `json:"event"`
		Authorizations []struct {
			EnterpriseID        string `json:"enterprise_id"`
			TeamID              string `json:"team_id"`
			UserID              string `json:"user_id"`
			IsBot               bool   `json:"is_bot"`
			IsEnterpriseInstall bool   `json:"is_enterprise_install"`
		} `json:"authorizations"`
		EventContext string `json:"event_context"`
		EventTime    int64  `json:"event_time"`
	}

	if err := json.Unmarshal(payload, &rawEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	event := &SlackEvent{
		Type:      rawEvent.Type,
		Timestamp: time.Now(),
		TenantID:  rawEvent.TeamID,
		Data:      make(map[string]any),
	}

	if rawEvent.Event != nil {
		var eventData map[string]any
		if err := json.Unmarshal(rawEvent.Event, &eventData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
		}
		event.Data = eventData
		event.Data["event_type"] = eventData["type"]
		if eventType, ok := eventData["type"].(string); ok {
			event.Type = eventType
		}
	}

	return event, nil
}

func (wh *WebhookHandler) handleURLVerification(w http.ResponseWriter, body []byte) {
	var challenge struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &challenge); err != nil {
		http.Error(w, "Invalid challenge", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(challenge.Challenge))
}

func (wh *WebhookHandler) HandleSlashCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	body := []byte(r.Form.Encode())
	if !wh.VerifySignature(body, timestamp, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	cmd := SlackCommandEvent{
		Command:     r.FormValue("command"),
		Text:        r.FormValue("text"),
		UserID:      r.FormValue("user_id"),
		UserName:    r.FormValue("user_name"),
		ChannelID:   r.FormValue("channel_id"),
		ChannelName: r.FormValue("channel_name"),
		TeamID:      r.FormValue("team_id"),
		TeamDomain:  r.FormValue("team_domain"),
		TriggerID:   r.FormValue("trigger_id"),
		ResponseURL: r.FormValue("response_url"),
	}

	if handler, ok := wh.registry.GetCommandHandler(cmd.Command); ok {
		go func() {
			if err := handler(r.Context(), &cmd); err != nil {
				fmt.Printf("Error handling command %s: %v\n", cmd.Command, err)
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
}

func (wh *WebhookHandler) HandleInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	timestamp := r.Header.Get("X-Slack-Request-Timestamp")
	signature := r.Header.Get("X-Slack-Signature")

	body := []byte(r.Form.Encode())
	if !wh.VerifySignature(body, timestamp, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	payloadStr := r.FormValue("payload")
	if payloadStr == "" {
		http.Error(w, "Missing payload", http.StatusBadRequest)
		return
	}

	var payload struct {
		Type        string       `json:"type"`
		User        SlackUser    `json:"user"`
		Channel     SlackChannel `json:"channel"`
		Message     SlackMessage `json:"message"`
		TriggerID   string       `json:"trigger_id"`
		ResponseURL string       `json:"response_url"`
		View        *SlackView   `json:"view"`
		Actions     []struct {
			ActionID string `json:"action_id"`
			BlockID  string `json:"block_id"`
			Type     string `json:"type"`
			Text     struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"text"`
			Value string `json:"value"`
			URL   string `json:"url"`
		} `json:"actions"`
		Team      SlackTeam `json:"team"`
		Container struct {
			Type        string `json:"type"`
			MessageTs   string `json:"message_ts"`
			ChannelID   string `json:"channel_id"`
			IsEphemeral bool   `json:"is_ephemeral"`
		} `json:"container"`
		CallbackID string `json:"callback_id"`
	}

	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	interaction := &SlackInteractionEvent{
		Type:        payload.Type,
		User:        payload.User,
		Channel:     payload.Channel,
		Message:     payload.Message,
		TriggerID:   payload.TriggerID,
		ResponseURL: payload.ResponseURL,
		View:        payload.View,
	}

	if len(payload.Actions) > 0 {
		interaction.ActionID = payload.Actions[0].ActionID
		interaction.BlockID = payload.Actions[0].BlockID
	}

	if handler, ok := wh.registry.GetInteractionHandler(interaction.ActionID); ok {
		go func() {
			if err := handler(r.Context(), interaction); err != nil {
				fmt.Printf("Error handling interaction %s: %v\n", interaction.ActionID, err)
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{})
}

type WebhookResponse struct {
	ResponseType    string            `json:"response_type,omitempty"`
	Text            string            `json:"text,omitempty"`
	Blocks          []SlackBlock      `json:"blocks,omitempty"`
	Attachments     []SlackAttachment `json:"attachments,omitempty"`
	ReplaceOriginal bool              `json:"replace_original,omitempty"`
	DeleteOriginal  bool              `json:"delete_original,omitempty"`
}

func (wh *WebhookHandler) RespondToCommand(responseURL string, response WebhookResponse) error {
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	resp, err := wh.client.httpClient.Post(responseURL, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to send response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("response failed with status: %d", resp.StatusCode)
	}

	return nil
}
