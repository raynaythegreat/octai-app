package jira

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ActionsHandler struct {
	client *Client
}

func NewActionsHandler(client *Client) *ActionsHandler {
	return &ActionsHandler{
		client: client,
	}
}

type CreateIssueResult struct {
	Issue   *JiraIssue
	Message string
	Success bool
	Error   error
}

func (h *ActionsHandler) CreateIssueFromMessage(ctx context.Context, req *MessageToIssueRequest) (*CreateIssueResult, error) {
	summary := h.extractSummary(req.Text, 255)
	description := h.formatDescription(req.Text, req)

	createReq := &JiraCreateIssueRequest{
		Fields: JiraCreateFields{
			Project: JiraProject{
				Key: req.ProjectKey,
			},
			Summary:     summary,
			Description: description,
			IssueType: JiraIssueType{
				ID: req.IssueType,
			},
		},
	}

	if createReq.Fields.IssueType.ID == "" {
		createReq.Fields.IssueType.Name = "Task"
	}

	if req.Priority != "" {
		createReq.Fields.Priority = &JiraPriority{
			Name: req.Priority,
		}
	}

	if len(req.Labels) > 0 {
		createReq.Fields.Labels = req.Labels
	}

	issue, err := h.client.CreateIssue(ctx, createReq)
	if err != nil {
		return &CreateIssueResult{
			Success: false,
			Error:   err,
			Message: fmt.Sprintf("Failed to create issue: %v", err),
		}, err
	}

	return &CreateIssueResult{
		Issue:   issue,
		Success: true,
		Message: fmt.Sprintf("Created issue %s: %s", issue.Key, issue.Summary),
	}, nil
}

func (h *ActionsHandler) extractSummary(text string, maxLen int) string {
	lines := strings.Split(text, "\n")
	summary := strings.TrimSpace(lines[0])

	if len(summary) > maxLen {
		summary = summary[:maxLen-3] + "..."
	}

	return summary
}

func (h *ActionsHandler) formatDescription(text string, req *MessageToIssueRequest) any {
	var sb strings.Builder
	sb.WriteString(text)
	sb.WriteString("\n\n---\n")
	sb.WriteString(fmt.Sprintf("Created from message in channel: %s\n", req.ChannelID))
	sb.WriteString(fmt.Sprintf("Message ID: %s\n", req.MessageID))
	sb.WriteString(fmt.Sprintf("User ID: %s\n", req.UserID))
	sb.WriteString(fmt.Sprintf("Created: %s\n", time.Now().Format(time.RFC3339)))

	if len(req.Metadata) > 0 {
		sb.WriteString("\nMetadata:\n")
		for k, v := range req.Metadata {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
		}
	}

	return sb.String()
}

type LinkResult struct {
	Success bool
	Message string
	Error   error
}

func (h *ActionsHandler) LinkMessageToIssue(ctx context.Context, req *IssueLinkRequest) (*LinkResult, error) {
	issue, err := h.client.GetIssue(ctx, req.IssueKey)
	if err != nil {
		return &LinkResult{
			Success: false,
			Message: fmt.Sprintf("Issue %s not found", req.IssueKey),
			Error:   err,
		}, err
	}

	comment := &JiraAddCommentRequest{
		Body: fmt.Sprintf("Linked to message: %s", req.MessageID),
	}

	if _, err := h.client.AddComment(ctx, req.IssueKey, comment); err != nil {
		return &LinkResult{
			Success: false,
			Message: fmt.Sprintf("Failed to add link comment: %v", err),
			Error:   err,
		}, err
	}

	return &LinkResult{
		Success: true,
		Message: fmt.Sprintf("Message linked to issue %s: %s", issue.Key, issue.Summary),
	}, nil
}

type StatusUpdateResult struct {
	Success   bool
	Message   string
	Issue     *JiraIssue
	OldStatus string
	NewStatus string
	Error     error
}

func (h *ActionsHandler) UpdateIssueStatus(ctx context.Context, issueKey string, targetStatus string) (*StatusUpdateResult, error) {
	issue, err := h.client.GetIssue(ctx, issueKey)
	if err != nil {
		return &StatusUpdateResult{
			Success: false,
			Message: fmt.Sprintf("Issue %s not found", issueKey),
			Error:   err,
		}, err
	}

	oldStatus := issue.Status.Name

	if strings.EqualFold(issue.Status.Name, targetStatus) {
		return &StatusUpdateResult{
			Success:   true,
			Message:   fmt.Sprintf("Issue %s is already in status %s", issueKey, targetStatus),
			Issue:     issue,
			OldStatus: oldStatus,
			NewStatus: targetStatus,
		}, nil
	}

	transitions, err := h.client.GetTransitions(ctx, issueKey)
	if err != nil {
		return &StatusUpdateResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get transitions: %v", err),
			Error:   err,
		}, err
	}

	var targetTransition *JiraTransition
	for i := range transitions {
		if strings.EqualFold(transitions[i].ToStatus.Name, targetStatus) ||
			strings.EqualFold(transitions[i].Name, targetStatus) {
			targetTransition = &transitions[i]
			break
		}
	}

	if targetTransition == nil {
		var available []string
		for _, t := range transitions {
			available = append(available, t.ToStatus.Name)
		}
		return &StatusUpdateResult{
			Success: false,
			Message: fmt.Sprintf("No transition found to status '%s'. Available statuses: %s", targetStatus, strings.Join(available, ", ")),
			Error:   fmt.Errorf("no transition to status %s", targetStatus),
		}, nil
	}

	if err := h.client.TransitionIssue(ctx, issueKey, targetTransition.ID); err != nil {
		return &StatusUpdateResult{
			Success: false,
			Message: fmt.Sprintf("Failed to transition issue: %v", err),
			Error:   err,
		}, err
	}

	updatedIssue, err := h.client.GetIssue(ctx, issueKey)
	if err != nil {
		return &StatusUpdateResult{
			Success:   true,
			Message:   fmt.Sprintf("Issue %s transitioned to %s", issueKey, targetStatus),
			OldStatus: oldStatus,
			NewStatus: targetStatus,
		}, nil
	}

	return &StatusUpdateResult{
		Success:   true,
		Message:   fmt.Sprintf("Issue %s transitioned from %s to %s", issueKey, oldStatus, updatedIssue.Status.Name),
		Issue:     updatedIssue,
		OldStatus: oldStatus,
		NewStatus: updatedIssue.Status.Name,
	}, nil
}

type CommentResult struct {
	Success bool
	Message string
	Comment *JiraComment
	Error   error
}

func (h *ActionsHandler) AddCommentFromMessage(ctx context.Context, issueKey string, text string, author string) (*CommentResult, error) {
	var body strings.Builder
	body.WriteString(text)
	if author != "" {
		body.WriteString(fmt.Sprintf("\n\n— %s", author))
	}

	comment := &JiraAddCommentRequest{
		Body: body.String(),
	}

	created, err := h.client.AddComment(ctx, issueKey, comment)
	if err != nil {
		return &CommentResult{
			Success: false,
			Message: fmt.Sprintf("Failed to add comment: %v", err),
			Error:   err,
		}, err
	}

	return &CommentResult{
		Success: true,
		Message: fmt.Sprintf("Comment added to issue %s", issueKey),
		Comment: created,
	}, nil
}

func (h *ActionsHandler) AssignIssue(ctx context.Context, issueKey string, assignee string) (*StatusUpdateResult, error) {
	if err := h.client.AssignIssue(ctx, issueKey, assignee); err != nil {
		return &StatusUpdateResult{
			Success: false,
			Message: fmt.Sprintf("Failed to assign issue: %v", err),
			Error:   err,
		}, err
	}

	issue, err := h.client.GetIssue(ctx, issueKey)
	if err != nil {
		return &StatusUpdateResult{
			Success: true,
			Message: fmt.Sprintf("Issue %s assigned to %s", issueKey, assignee),
		}, nil
	}

	assigneeName := assignee
	if issue.Assignee != nil {
		assigneeName = issue.Assignee.DisplayName
	}

	return &StatusUpdateResult{
		Success: true,
		Message: fmt.Sprintf("Issue %s assigned to %s", issueKey, assigneeName),
		Issue:   issue,
	}, nil
}

func (h *ActionsHandler) SearchIssuesByUser(ctx context.Context, accountID string, maxResults int) ([]JiraIssue, error) {
	jql := fmt.Sprintf("assignee = \"%s\" AND resolution = Unresolved ORDER BY updated DESC", accountID)
	return h.client.SearchIssuesWithOpts(ctx, jql, 0, maxResults)
}

func (h *ActionsHandler) SearchIssuesByProject(ctx context.Context, projectKey string, maxResults int) ([]JiraIssue, error) {
	jql := fmt.Sprintf("project = \"%s\" AND resolution = Unresolved ORDER BY updated DESC", projectKey)
	return h.client.SearchIssuesWithOpts(ctx, jql, 0, maxResults)
}

func (h *ActionsHandler) SearchRecentIssues(ctx context.Context, days int, maxResults int) ([]JiraIssue, error) {
	jql := fmt.Sprintf("updated >= -%dd ORDER BY updated DESC", days)
	return h.client.SearchIssuesWithOpts(ctx, jql, 0, maxResults)
}

func (h *ActionsHandler) GetIssueStats(ctx context.Context, projectKey string) (*IssueStats, error) {
	jql := fmt.Sprintf("project = \"%s\"", projectKey)

	openJQL := jql + " AND resolution = Unresolved"
	openIssues, err := h.client.SearchIssuesWithOpts(ctx, openJQL, 0, 0)
	if err != nil {
		return nil, err
	}

	inProgressJQL := jql + " AND status in (\"In Progress\", \"In Progress\") AND resolution = Unresolved"
	inProgressIssues, err := h.client.SearchIssuesWithOpts(ctx, inProgressJQL, 0, 0)
	if err != nil {
		return nil, err
	}

	doneJQL := jql + " AND resolution != Unresolved"
	doneIssues, err := h.client.SearchIssuesWithOpts(ctx, doneJQL, 0, 0)
	if err != nil {
		return nil, err
	}

	return &IssueStats{
		ProjectKey:  projectKey,
		TotalOpen:   len(openIssues),
		InProgress:  len(inProgressIssues),
		Done:        len(doneIssues),
		LastUpdated: time.Now(),
	}, nil
}

type IssueStats struct {
	ProjectKey  string    `json:"project_key"`
	TotalOpen   int       `json:"total_open"`
	InProgress  int       `json:"in_progress"`
	Done        int       `json:"done"`
	LastUpdated time.Time `json:"last_updated"`
}
