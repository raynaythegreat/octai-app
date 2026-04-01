package jira

import "time"

type JiraConfig struct {
	Host     string
	Email    string
	APIToken string
	CloudID  string
	IsCloud  bool
	PAT      string
}

type JiraIssue struct {
	ID          string           `json:"id"`
	Key         string           `json:"key"`
	Summary     string           `json:"summary"`
	Description string           `json:"description"`
	Status      JiraStatus       `json:"status"`
	Priority    JiraPriority     `json:"priority"`
	Assignee    *JiraUser        `json:"assignee,omitempty"`
	Reporter    *JiraUser        `json:"reporter,omitempty"`
	Project     JiraProject      `json:"project"`
	IssueType   JiraIssueType    `json:"issuetype"`
	Labels      []string         `json:"labels,omitempty"`
	Components  []JiraComponent  `json:"components,omitempty"`
	Created     time.Time        `json:"created"`
	Updated     time.Time        `json:"updated"`
	Resolution  *JiraResolution  `json:"resolution,omitempty"`
	DueDate     *time.Time       `json:"duedate,omitempty"`
	Parent      *JiraIssueParent `json:"parent,omitempty"`
	Subtasks    []JiraSubtask    `json:"subtasks,omitempty"`
	Comments    []JiraComment    `json:"comments,omitempty"`
	Attachments []JiraAttachment `json:"attachments,omitempty"`
	Links       []JiraIssueLink  `json:"links,omitempty"`
	Fields      map[string]any   `json:"fields,omitempty"`
}

type JiraIssueParent struct {
	ID  string `json:"id"`
	Key string `json:"key"`
}

type JiraSubtask struct {
	ID     string     `json:"id"`
	Key    string     `json:"key"`
	Fields JiraFields `json:"fields"`
}

type JiraFields struct {
	Summary   string        `json:"summary"`
	Status    JiraStatus    `json:"status"`
	Priority  JiraPriority  `json:"priority"`
	IssueType JiraIssueType `json:"issuetype"`
	Assignee  *JiraUser     `json:"assignee,omitempty"`
	Project   JiraProject   `json:"project"`
}

type JiraStatus struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Category    JiraCategory `json:"statusCategory"`
}

type JiraCategory struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type JiraPriority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type JiraIssueType struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Subtask     bool   `json:"subtask,omitempty"`
}

type JiraComponent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type JiraResolution struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type JiraProject struct {
	ID          string        `json:"id"`
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Lead        *JiraUser     `json:"lead,omitempty"`
	Description string        `json:"description,omitempty"`
	ProjectType string        `json:"projectTypeKey,omitempty"`
	Category    *JiraCategory `json:"projectCategory,omitempty"`
}

type JiraUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"emailAddress,omitempty"`
	DisplayName string `json:"displayName"`
	Active      bool   `json:"active"`
	AccountID   string `json:"accountId,omitempty"`
	AccountType string `json:"accountType,omitempty"`
	AvatarURL   string `json:"avatarUrls,omitempty"`
}

type JiraComment struct {
	ID           string    `json:"id"`
	Body         string    `json:"body"`
	Author       JiraUser  `json:"author"`
	UpdateAuthor JiraUser  `json:"updateAuthor,omitempty"`
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated,omitempty"`
}

type JiraAttachment struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	Content   string    `json:"content"`
	MimeType  string    `json:"mimeType"`
	Size      int64     `json:"size"`
	Author    JiraUser  `json:"author"`
	Created   time.Time `json:"created"`
	Thumbnail string    `json:"thumbnail,omitempty"`
}

type JiraIssueLink struct {
	ID      string            `json:"id"`
	Type    JiraIssueLinkType `json:"type"`
	Inward  *JiraLinkedIssue  `json:"inwardIssue,omitempty"`
	Outward *JiraLinkedIssue  `json:"outwardIssue,omitempty"`
}

type JiraIssueLinkType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Inward  string `json:"inward"`
	Outward string `json:"outward"`
}

type JiraLinkedIssue struct {
	ID     string     `json:"id"`
	Key    string     `json:"key"`
	Fields JiraFields `json:"fields"`
}

type JiraTransition struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	ToStatus      JiraStatus `json:"to"`
	HasScreen     bool       `json:"hasScreen,omitempty"`
	IsGlobal      bool       `json:"isGlobal,omitempty"`
	IsInitial     bool       `json:"isInitial,omitempty"`
	IsConditional bool       `json:"isConditional,omitempty"`
	Fields        JiraFields `json:"fields,omitempty"`
}

type JiraSearchResult struct {
	StartAt    int         `json:"startAt"`
	MaxResults int         `json:"maxResults"`
	Total      int         `json:"total"`
	Issues     []JiraIssue `json:"issues"`
}

type JiraCreateIssueRequest struct {
	Fields JiraCreateFields `json:"fields"`
}

type JiraCreateFields struct {
	Project     JiraProject      `json:"project"`
	Summary     string           `json:"summary"`
	Description any              `json:"description,omitempty"`
	IssueType   JiraIssueType    `json:"issuetype"`
	Priority    *JiraPriority    `json:"priority,omitempty"`
	Assignee    *JiraUser        `json:"assignee,omitempty"`
	Labels      []string         `json:"labels,omitempty"`
	Components  []JiraComponent  `json:"components,omitempty"`
	Parent      *JiraIssueParent `json:"parent,omitempty"`
	DueDate     string           `json:"duedate,omitempty"`
}

type JiraUpdateIssueRequest struct {
	Fields map[string]any `json:"fields"`
}

type JiraAddCommentRequest struct {
	Body string `json:"body"`
}

type JiraTransitionRequest struct {
	Transition JiraTransitionID `json:"transition"`
	Fields     map[string]any   `json:"fields,omitempty"`
}

type JiraTransitionID struct {
	ID string `json:"id"`
}

type JiraErrorResponse struct {
	ErrorMessages []string          `json:"errorMessages"`
	Errors        map[string]string `json:"errors"`
}

type JiraWebhookEvent struct {
	Timestamp  int64           `json:"timestamp"`
	WebhookID  string          `json:"webhookId"`
	Event      string          `json:"webhookEvent"`
	IssueEvent string          `json:"issue_event_type_name,omitempty"`
	User       JiraUser        `json:"user"`
	Issue      *JiraIssue      `json:"issue,omitempty"`
	Comment    *JiraComment    `json:"comment,omitempty"`
	Changelog  *JiraChangelog  `json:"changelog,omitempty"`
	Transition *JiraTransition `json:"transition,omitempty"`
	Project    *JiraProject    `json:"project,omitempty"`
	Version    string          `json:"version,omitempty"`
	EventID    string          `json:"event_id,omitempty"`
}

type JiraChangelog struct {
	ID    string           `json:"id"`
	Items []JiraChangeItem `json:"items"`
}

type JiraChangeItem struct {
	Field      string `json:"field"`
	FieldType  string `json:"fieldtype"`
	From       string `json:"from"`
	FromString string `json:"fromString"`
	To         string `json:"to"`
	ToString   string `json:"toString"`
}

type MessageToIssueRequest struct {
	MessageID  string            `json:"message_id"`
	ChannelID  string            `json:"channel_id"`
	UserID     string            `json:"user_id"`
	Text       string            `json:"text"`
	ProjectKey string            `json:"project_key"`
	IssueType  string            `json:"issue_type,omitempty"`
	Priority   string            `json:"priority,omitempty"`
	Labels     []string          `json:"labels,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type IssueLinkRequest struct {
	MessageID string `json:"message_id"`
	IssueKey  string `json:"issue_key"`
	LinkType  string `json:"link_type,omitempty"`
}
