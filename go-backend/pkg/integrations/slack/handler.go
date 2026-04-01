package slack

import (
	"context"
	"encoding/json"
	"fmt"
)

type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{
		client: client,
	}
}

func (h *Handler) HandleEvent(ctx context.Context, event SlackEvent) error {
	switch event.Type {
	case "message":
		return h.HandleMessageEvent(ctx, event)
	case "command":
		return h.HandleCommandEvent(ctx, event)
	case "interaction":
		return h.HandleInteractionEvent(ctx, event)
	case "url_verification":
		return nil
	default:
		return fmt.Errorf("unhandled event type: %s", event.Type)
	}
}

func (h *Handler) HandleMessageEvent(ctx context.Context, event SlackEvent) error {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	var msgEvent struct {
		Type       string `json:"type"`
		Channel    string `json:"channel"`
		User       string `json:"user"`
		Text       string `json:"text"`
		Ts         string `json:"ts"`
		ThreadTs   string `json:"thread_ts"`
		SubType    string `json:"subtype"`
		BotID      string `json:"bot_id"`
		BotProfile struct {
			AppID string `json:"app_id"`
		} `json:"bot_profile"`
		Files []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			URL  string `json:"url_private_download"`
		} `json:"files"`
	}

	if err := json.Unmarshal(data, &msgEvent); err != nil {
		return fmt.Errorf("failed to unmarshal message event: %w", err)
	}

	if msgEvent.BotID != "" {
		return nil
	}

	if msgEvent.SubType != "" && msgEvent.SubType != "file_share" {
		return nil
	}

	return nil
}

func (h *Handler) HandleCommandEvent(ctx context.Context, event SlackEvent) error {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	var cmd SlackCommandEvent
	if err := json.Unmarshal(data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command event: %w", err)
	}

	return nil
}

func (h *Handler) HandleInteractionEvent(ctx context.Context, event SlackEvent) error {
	data, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	var interaction SlackInteractionEvent
	if err := json.Unmarshal(data, &interaction); err != nil {
		return fmt.Errorf("failed to unmarshal interaction event: %w", err)
	}

	switch interaction.Type {
	case "block_actions":
		return h.handleBlockAction(ctx, &interaction)
	case "view_submission":
		return h.handleViewSubmission(ctx, &interaction)
	case "view_closed":
		return h.handleViewClosed(ctx, &interaction)
	case "shortcut":
		return h.handleShortcut(ctx, &interaction)
	default:
		return fmt.Errorf("unhandled interaction type: %s", interaction.Type)
	}
}

func (h *Handler) handleBlockAction(ctx context.Context, interaction *SlackInteractionEvent) error {
	return nil
}

func (h *Handler) handleViewSubmission(ctx context.Context, interaction *SlackInteractionEvent) error {
	return nil
}

func (h *Handler) handleViewClosed(ctx context.Context, interaction *SlackInteractionEvent) error {
	return nil
}

func (h *Handler) handleShortcut(ctx context.Context, interaction *SlackInteractionEvent) error {
	return nil
}

type MessageHandlerFunc func(ctx context.Context, msg *SlackMessage) error
type CommandHandlerFunc func(ctx context.Context, cmd *SlackCommandEvent) error
type InteractionHandlerFunc func(ctx context.Context, interaction *SlackInteractionEvent) error

type HandlerRegistry struct {
	messageHandlers     map[string]MessageHandlerFunc
	commandHandlers     map[string]CommandHandlerFunc
	interactionHandlers map[string]InteractionHandlerFunc
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		messageHandlers:     make(map[string]MessageHandlerFunc),
		commandHandlers:     make(map[string]CommandHandlerFunc),
		interactionHandlers: make(map[string]InteractionHandlerFunc),
	}
}

func (r *HandlerRegistry) RegisterMessageHandler(eventType string, handler MessageHandlerFunc) {
	r.messageHandlers[eventType] = handler
}

func (r *HandlerRegistry) RegisterCommandHandler(command string, handler CommandHandlerFunc) {
	r.commandHandlers[command] = handler
}

func (r *HandlerRegistry) RegisterInteractionHandler(actionID string, handler InteractionHandlerFunc) {
	r.interactionHandlers[actionID] = handler
}

func (r *HandlerRegistry) GetMessageHandler(eventType string) (MessageHandlerFunc, bool) {
	handler, ok := r.messageHandlers[eventType]
	return handler, ok
}

func (r *HandlerRegistry) GetCommandHandler(command string) (CommandHandlerFunc, bool) {
	handler, ok := r.commandHandlers[command]
	return handler, ok
}

func (r *HandlerRegistry) GetInteractionHandler(actionID string) (InteractionHandlerFunc, bool) {
	handler, ok := r.interactionHandlers[actionID]
	return handler, ok
}
