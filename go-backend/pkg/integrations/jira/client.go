package jira

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	config     JiraConfig
	baseURL    string
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

func NewClient(config JiraConfig, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		config:  config,
		baseURL: buildBaseURL(config),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func buildBaseURL(config JiraConfig) string {
	host := strings.TrimSuffix(config.Host, "/")
	if config.IsCloud {
		return fmt.Sprintf("https://api.atlassian.com/ex/jira/%s/rest/api/3", config.CloudID)
	}
	return fmt.Sprintf("%s/rest/api/2", host)
}

func (c *Client) GetIssue(ctx context.Context, key string) (*JiraIssue, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/issue/%s", key), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var issue JiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode issue: %w", err)
	}

	return &issue, nil
}

func (c *Client) SearchIssues(ctx context.Context, jql string) ([]JiraIssue, error) {
	return c.SearchIssuesWithOpts(ctx, jql, 0, 50)
}

func (c *Client) SearchIssuesWithOpts(ctx context.Context, jql string, startAt, maxResults int) ([]JiraIssue, error) {
	params := url.Values{}
	params.Set("jql", jql)
	params.Set("startAt", fmt.Sprintf("%d", startAt))
	params.Set("maxResults", fmt.Sprintf("%d", maxResults))

	resp, err := c.doRequest(ctx, "GET", "/search", params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result JiraSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search result: %w", err)
	}

	return result.Issues, nil
}

func (c *Client) SearchAllIssues(ctx context.Context, jql string) ([]JiraIssue, error) {
	var allIssues []JiraIssue
	startAt := 0
	maxResults := 100

	for {
		issues, err := c.SearchIssuesWithOpts(ctx, jql, startAt, maxResults)
		if err != nil {
			return nil, err
		}

		if len(issues) == 0 {
			break
		}

		allIssues = append(allIssues, issues...)

		if len(issues) < maxResults {
			break
		}

		startAt += maxResults
	}

	return allIssues, nil
}

func (c *Client) CreateIssue(ctx context.Context, req *JiraCreateIssueRequest) (*JiraIssue, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", "/issue", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var created struct {
		ID   string `json:"id"`
		Key  string `json:"key"`
		Self string `json:"self"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.GetIssue(ctx, created.Key)
}

func (c *Client) UpdateIssue(ctx context.Context, key string, req *JiraUpdateIssueRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "PUT", fmt.Sprintf("/issue/%s", key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) DeleteIssue(ctx context.Context, key string) error {
	resp, err := c.doRequest(ctx, "DELETE", fmt.Sprintf("/issue/%s", key), nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) AddComment(ctx context.Context, key string, comment *JiraAddCommentRequest) (*JiraComment, error) {
	body, err := json.Marshal(comment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", fmt.Sprintf("/issue/%s/comment", key), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var createdComment JiraComment
	if err := json.NewDecoder(resp.Body).Decode(&createdComment); err != nil {
		return nil, fmt.Errorf("failed to decode comment: %w", err)
	}

	return &createdComment, nil
}

func (c *Client) GetComments(ctx context.Context, key string) ([]JiraComment, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/issue/%s/comment", key), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		Comments []JiraComment `json:"comments"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode comments: %w", err)
	}

	return result.Comments, nil
}

func (c *Client) TransitionIssue(ctx context.Context, key string, transitionID string) error {
	req := JiraTransitionRequest{
		Transition: JiraTransitionID{ID: transitionID},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", fmt.Sprintf("/issue/%s/transitions", key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) TransitionIssueWithFields(ctx context.Context, key string, transitionID string, fields map[string]any) error {
	req := JiraTransitionRequest{
		Transition: JiraTransitionID{ID: transitionID},
		Fields:     fields,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "POST", fmt.Sprintf("/issue/%s/transitions", key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) GetTransitions(ctx context.Context, key string) ([]JiraTransition, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/issue/%s/transitions", key), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var result struct {
		Transitions []JiraTransition `json:"transitions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode transitions: %w", err)
	}

	return result.Transitions, nil
}

func (c *Client) GetProjects(ctx context.Context) ([]JiraProject, error) {
	resp, err := c.doRequest(ctx, "GET", "/project", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var projects []JiraProject
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects: %w", err)
	}

	return projects, nil
}

func (c *Client) GetProject(ctx context.Context, key string) (*JiraProject, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/project/%s", key), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var project JiraProject
	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, fmt.Errorf("failed to decode project: %w", err)
	}

	return &project, nil
}

func (c *Client) GetIssueTypes(ctx context.Context, projectKey string) ([]JiraIssueType, error) {
	resp, err := c.doRequest(ctx, "GET", fmt.Sprintf("/project/%s/statuses", projectKey), nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var issueTypes []JiraIssueType
	if err := json.NewDecoder(resp.Body).Decode(&issueTypes); err != nil {
		return nil, fmt.Errorf("failed to decode issue types: %w", err)
	}

	return issueTypes, nil
}

func (c *Client) GetUser(ctx context.Context, accountID string) (*JiraUser, error) {
	params := url.Values{}
	params.Set("accountId", accountID)

	resp, err := c.doRequest(ctx, "GET", "/user", params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var user JiraUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	return &user, nil
}

func (c *Client) SearchUsers(ctx context.Context, query string) ([]JiraUser, error) {
	params := url.Values{}
	params.Set("query", query)

	resp, err := c.doRequest(ctx, "GET", "/user/search", params, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := c.checkErrorResponse(resp); err != nil {
		return nil, err
	}

	var users []JiraUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

func (c *Client) AssignIssue(ctx context.Context, key string, accountID string) error {
	req := map[string]any{
		"name": accountID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.doJSONRequest(ctx, "PUT", fmt.Sprintf("/issue/%s/assignee", key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) AddWatcher(ctx context.Context, key string, accountID string) error {
	body := []byte(fmt.Sprintf(`"%s"`, accountID))

	resp, err := c.doJSONRequest(ctx, "POST", fmt.Sprintf("/issue/%s/watchers", key), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return c.checkErrorResponse(resp)
}

func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.GetProjects(ctx)
	return err
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values, body io.Reader) (*http.Response, error) {
	endpointURL := c.baseURL + endpoint

	if params != nil && len(params) > 0 {
		endpointURL = endpointURL + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpointURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) doJSONRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	endpointURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, method, endpointURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) setAuth(req *http.Request) {
	if c.config.IsCloud {
		auth := base64.StdEncoding.EncodeToString([]byte(c.config.Email + ":" + c.config.APIToken))
		req.Header.Set("Authorization", "Basic "+auth)
	} else {
		if c.config.PAT != "" {
			req.Header.Set("Authorization", "Bearer "+c.config.PAT)
		} else {
			auth := base64.StdEncoding.EncodeToString([]byte(c.config.Email + ":" + c.config.APIToken))
			req.Header.Set("Authorization", "Basic "+auth)
		}
	}
}

func (c *Client) checkErrorResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("request failed with status %d: failed to read error body", resp.StatusCode)
	}

	var jiraErr JiraErrorResponse
	if err := json.Unmarshal(body, &jiraErr); err == nil {
		if len(jiraErr.ErrorMessages) > 0 || len(jiraErr.Errors) > 0 {
			return fmt.Errorf("jira error (status %d): %v %v", resp.StatusCode, jiraErr.ErrorMessages, jiraErr.Errors)
		}
	}

	return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
}
