package slack

import "time"

type SlackConfig struct {
	BotToken      string
	AppToken      string
	SigningSecret string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
}

type SlackChannel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsPrivate bool   `json:"is_private"`
	IsChannel bool   `json:"is_channel"`
	IsGroup   bool   `json:"is_group"`
	IsIM      bool   `json:"is_im"`
}

type SlackUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	RealName string `json:"real_name"`
	Email    string `json:"email"`
	IsAdmin  bool   `json:"is_admin"`
	IsOwner  bool   `json:"is_owner"`
	IsBot    bool   `json:"is_bot"`
	Deleted  bool   `json:"deleted"`
}

type SlackMessage struct {
	Channel     string            `json:"channel"`
	User        string            `json:"user"`
	Text        string            `json:"text"`
	Timestamp   string            `json:"ts"`
	ThreadTS    string            `json:"thread_ts,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type SlackBlock struct {
	Type     string              `json:"type"`
	Text     *SlackTextBlock     `json:"text,omitempty"`
	Elements []SlackBlockElement `json:"elements,omitempty"`
}

type SlackTextBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

type SlackBlockElement struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ActionID string `json:"action_id,omitempty"`
	URL      string `json:"url,omitempty"`
	Value    string `json:"value,omitempty"`
}

type SlackAttachment struct {
	ID      int                     `json:"id"`
	Color   string                  `json:"color,omitempty"`
	Title   string                  `json:"title,omitempty"`
	Text    string                  `json:"text,omitempty"`
	Fields  []SlackAttachmentField  `json:"fields,omitempty"`
	Actions []SlackAttachmentAction `json:"actions,omitempty"`
}

type SlackAttachmentField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type SlackAttachmentAction struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	URL   string `json:"url,omitempty"`
	Style string `json:"style,omitempty"`
}

type SlackEvent struct {
	Type      string         `json:"type"`
	Data      map[string]any `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
	TenantID  string         `json:"tenant_id,omitempty"`
}

type SlackCommandEvent struct {
	Command     string `json:"command"`
	Text        string `json:"text"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	TeamID      string `json:"team_id"`
	TeamDomain  string `json:"team_domain"`
	TriggerID   string `json:"trigger_id"`
	ResponseURL string `json:"response_url"`
}

type SlackInteractionEvent struct {
	Type        string       `json:"type"`
	User        SlackUser    `json:"user"`
	Channel     SlackChannel `json:"channel"`
	Message     SlackMessage `json:"message"`
	ActionID    string       `json:"action_id"`
	BlockID     string       `json:"block_id"`
	TriggerID   string       `json:"trigger_id"`
	ResponseURL string       `json:"response_url"`
	View        *SlackView   `json:"view,omitempty"`
}

type SlackView struct {
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	Title           SlackTextBlock `json:"title"`
	SubmitLabel     string         `json:"submit_label,omitempty"`
	Blocks          []SlackBlock   `json:"blocks"`
	PrivateMetadata string         `json:"private_metadata,omitempty"`
	State           map[string]any `json:"state,omitempty"`
	CallbackID      string         `json:"callback_id"`
}

type SlackViewSubmission struct {
	Type string    `json:"type"`
	User SlackUser `json:"user"`
	View SlackView `json:"view"`
	Team SlackTeam `json:"team"`
}

type SlackTeam struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

type SlackAPIResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type SlackPostMessageResponse struct {
	SlackAPIResponse
	TS      string       `json:"ts"`
	Channel string       `json:"channel"`
	Message SlackMessage `json:"message"`
}

type SlackConversationListResponse struct {
	SlackAPIResponse
	Channels         []SlackChannel         `json:"channels"`
	ResponseMetadata *SlackResponseMetadata `json:"response_metadata,omitempty"`
}

type SlackUsersListResponse struct {
	SlackAPIResponse
	Members          []SlackUser            `json:"members"`
	ResponseMetadata *SlackResponseMetadata `json:"response_metadata,omitempty"`
}

type SlackResponseMetadata struct {
	NextCursor string `json:"next_cursor"`
}

type SlackOAuthResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	BotUserID    string    `json:"bot_user_id"`
	AppID        string    `json:"app_id"`
	Team         SlackTeam `json:"team"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
}

type SlackModalView struct {
	Type            string          `json:"type"`
	Title           SlackTextBlock  `json:"title"`
	Blocks          []SlackBlock    `json:"blocks"`
	Submit          *SlackTextBlock `json:"submit,omitempty"`
	Close           *SlackTextBlock `json:"close,omitempty"`
	PrivateMetadata string          `json:"private_metadata,omitempty"`
	CallbackID      string          `json:"callback_id,omitempty"`
}
