package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient    *http.Client
	botToken      string
	appToken      string
	signingSecret string
	clientID      string
	clientSecret  string
	redirectURL   string
	baseURL       string
}

type ClientOption func(*Client)

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

func NewClient(cfg SlackConfig, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		botToken:      cfg.BotToken,
		appToken:      cfg.AppToken,
		signingSecret: cfg.SigningSecret,
		clientID:      cfg.ClientID,
		clientSecret:  cfg.ClientSecret,
		redirectURL:   cfg.RedirectURL,
		baseURL:       "https://slack.com/api",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) SendMessage(ctx context.Context, channel, text string) error {
	_, err := c.PostMessage(ctx, channel, text, nil)
	return err
}

func (c *Client) PostMessage(ctx context.Context, channel, text string, blocks []SlackBlock) (*SlackPostMessageResponse, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("text", text)

	if len(blocks) > 0 {
		blocksJSON, err := json.Marshal(blocks)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal blocks: %w", err)
		}
		params.Set("blocks", string(blocksJSON))
	}

	resp, err := c.doRequest(ctx, "POST", "/chat.postMessage", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SlackPostMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return &result, nil
}

func (c *Client) SendEphemeral(ctx context.Context, channel, user, text string) error {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("user", user)
	params.Set("text", text)

	resp, err := c.doRequest(ctx, "POST", "/chat.postEphemeral", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result SlackAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

func (c *Client) GetChannels(ctx context.Context) ([]SlackChannel, error) {
	return c.GetChannelsWithTypes(ctx, "public_channel", "private_channel")
}

func (c *Client) GetChannelsWithTypes(ctx context.Context, types ...string) ([]SlackChannel, error) {
	var allChannels []SlackChannel
	cursor := ""

	for {
		params := url.Values{}
		params.Set("types", strings.Join(types, ","))
		params.Set("limit", "200")
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		resp, err := c.doRequest(ctx, "GET", "/conversations.list", params)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result SlackConversationListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if !result.OK {
			return nil, fmt.Errorf("slack API error: %s", result.Error)
		}

		allChannels = append(allChannels, result.Channels...)

		if result.ResponseMetadata == nil || result.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = result.ResponseMetadata.NextCursor
	}

	return allChannels, nil
}

func (c *Client) GetUsers(ctx context.Context) ([]SlackUser, error) {
	var allUsers []SlackUser
	cursor := ""

	for {
		params := url.Values{}
		params.Set("limit", "200")
		if cursor != "" {
			params.Set("cursor", cursor)
		}

		resp, err := c.doRequest(ctx, "GET", "/users.list", params)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result SlackUsersListResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if !result.OK {
			return nil, fmt.Errorf("slack API error: %s", result.Error)
		}

		allUsers = append(allUsers, result.Members...)

		if result.ResponseMetadata == nil || result.ResponseMetadata.NextCursor == "" {
			break
		}
		cursor = result.ResponseMetadata.NextCursor
	}

	return allUsers, nil
}

func (c *Client) OpenModal(ctx context.Context, triggerID string, view SlackModalView) error {
	viewJSON, err := json.Marshal(view)
	if err != nil {
		return fmt.Errorf("failed to marshal view: %w", err)
	}

	params := url.Values{}
	params.Set("trigger_id", triggerID)
	params.Set("view", string(viewJSON))

	resp, err := c.doRequest(ctx, "POST", "/views.open", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result SlackAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

func (c *Client) UpdateModal(ctx context.Context, viewID string, view SlackModalView) error {
	viewJSON, err := json.Marshal(view)
	if err != nil {
		return fmt.Errorf("failed to marshal view: %w", err)
	}

	params := url.Values{}
	params.Set("view_id", viewID)
	params.Set("view", string(viewJSON))

	resp, err := c.doRequest(ctx, "POST", "/views.update", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result SlackAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

func (c *Client) PushModal(ctx context.Context, triggerID string, view SlackModalView) error {
	viewJSON, err := json.Marshal(view)
	if err != nil {
		return fmt.Errorf("failed to marshal view: %w", err)
	}

	params := url.Values{}
	params.Set("trigger_id", triggerID)
	params.Set("view", string(viewJSON))

	resp, err := c.doRequest(ctx, "POST", "/views.push", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result SlackAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

func (c *Client) AddReaction(ctx context.Context, channel, timestamp, reaction string) error {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("timestamp", timestamp)
	params.Set("name", reaction)

	resp, err := c.doRequest(ctx, "POST", "/reactions.add", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result SlackAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

func (c *Client) RemoveReaction(ctx context.Context, channel, timestamp, reaction string) error {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("timestamp", timestamp)
	params.Set("name", reaction)

	resp, err := c.doRequest(ctx, "POST", "/reactions.remove", params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result SlackAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

func (c *Client) GetUserInfo(ctx context.Context, userID string) (*SlackUser, error) {
	params := url.Values{}
	params.Set("user", userID)

	resp, err := c.doRequest(ctx, "GET", "/users.info", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		SlackAPIResponse
		User SlackUser `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return &result.User, nil
}

func (c *Client) GetConversationInfo(ctx context.Context, channelID string) (*SlackChannel, error) {
	params := url.Values{}
	params.Set("channel", channelID)

	resp, err := c.doRequest(ctx, "GET", "/conversations.info", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		SlackAPIResponse
		Channel SlackChannel `json:"channel"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return &result.Channel, nil
}

func (c *Client) AuthTest(ctx context.Context) (*SlackAuthTestResponse, error) {
	resp, err := c.doRequest(ctx, "GET", "/auth.test", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result SlackAuthTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return &result, nil
}

type SlackAuthTestResponse struct {
	SlackAPIResponse
	URL    string `json:"url"`
	Team   string `json:"team"`
	User   string `json:"user"`
	TeamID string `json:"team_id"`
	UserID string `json:"user_id"`
	BotID  string `json:"bot_id,omitempty"`
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values) (*http.Response, error) {
	var req *http.Request
	var err error

	endpointURL := c.baseURL + endpoint

	if method == "GET" && params != nil {
		endpointURL = endpointURL + "?" + params.Encode()
		req, err = http.NewRequestWithContext(ctx, method, endpointURL, nil)
	} else if method == "POST" && params != nil {
		req, err = http.NewRequestWithContext(ctx, method, endpointURL, strings.NewReader(params.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, endpointURL, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.botToken)

	return c.httpClient.Do(req)
}

func (c *Client) doJSONRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	endpointURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, endpointURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.botToken)
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}
