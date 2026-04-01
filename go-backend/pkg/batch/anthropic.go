package batch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/providers/protocoltypes"
)

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com/v1"
	anthropicVersion        = "2023-06-01"
	anthropicBatchBeta      = "message-batches-2024-09-24"
)

// AnthropicBatchClient implements BatchClient for the Anthropic Message Batches API.
type AnthropicBatchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewAnthropicBatchClient creates a new Anthropic batch client with the given API key.
func NewAnthropicBatchClient(apiKey string) *AnthropicBatchClient {
	return &AnthropicBatchClient{
		apiKey:  apiKey,
		baseURL: anthropicDefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ---- wire types for Anthropic Batches API ----

type anthropicBatchCreateRequest struct {
	Requests []anthropicBatchRequestItem `json:"requests"`
}

type anthropicBatchRequestItem struct {
	CustomID string                 `json:"custom_id"`
	Params   anthropicMessageParams `json:"params"`
}

type anthropicMessageParams struct {
	Model     string               `json:"model"`
	MaxTokens int                  `json:"max_tokens"`
	Messages  []anthropicMessage   `json:"messages"`
	System    []anthropicTextBlock `json:"system,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicBatchResponse struct {
	ID             string                    `json:"id"`
	Type           string                    `json:"type"`
	ProcessingStatus string                  `json:"processing_status"`
	RequestCounts  anthropicRequestCounts    `json:"request_counts"`
	CreatedAt      time.Time                 `json:"created_at"`
	ExpiresAt      time.Time                 `json:"expires_at"`
	ResultsURL     string                    `json:"results_url,omitempty"`
}

type anthropicRequestCounts struct {
	Processing int `json:"processing"`
	Succeeded  int `json:"succeeded"`
	Errored    int `json:"errored"`
	Canceled   int `json:"canceled"`
	Expired    int `json:"expired"`
}

type anthropicResultLine struct {
	CustomID string               `json:"custom_id"`
	Result   anthropicResultEntry `json:"result"`
}

type anthropicResultEntry struct {
	Type    string           `json:"type"` // "succeeded", "errored", "expired"
	Message *anthropicMsg    `json:"message,omitempty"`
	Error   *anthropicError  `json:"error,omitempty"`
}

type anthropicMsg struct {
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      anthropicUsage          `json:"usage"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// ---- BatchClient implementation ----

// Submit sends a batch of requests to the Anthropic Message Batches API.
func (c *AnthropicBatchClient) Submit(ctx context.Context, requests []BatchRequest) (BatchStatus, error) {
	items := make([]anthropicBatchRequestItem, 0, len(requests))
	for _, req := range requests {
		item, err := c.buildRequestItem(req)
		if err != nil {
			return BatchStatus{}, fmt.Errorf("building request item %q: %w", req.CustomID, err)
		}
		items = append(items, item)
	}

	body, err := json.Marshal(anthropicBatchCreateRequest{Requests: items})
	if err != nil {
		return BatchStatus{}, fmt.Errorf("marshaling batch request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/messages/batches", body, 10*time.Second)
	if err != nil {
		return BatchStatus{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return BatchStatus{}, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return BatchStatus{}, fmt.Errorf("anthropic batch submit: status %d: %s", resp.StatusCode, string(data))
	}

	var br anthropicBatchResponse
	if err := json.Unmarshal(data, &br); err != nil {
		return BatchStatus{}, fmt.Errorf("parsing batch response: %w", err)
	}

	return c.toBatchStatus(br), nil
}

// Poll returns the current status of a batch.
func (c *AnthropicBatchClient) Poll(ctx context.Context, batchID string) (BatchStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/messages/batches/"+batchID, nil, 10*time.Second)
	if err != nil {
		return BatchStatus{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return BatchStatus{}, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return BatchStatus{}, fmt.Errorf("anthropic batch poll: status %d: %s", resp.StatusCode, string(data))
	}

	var br anthropicBatchResponse
	if err := json.Unmarshal(data, &br); err != nil {
		return BatchStatus{}, fmt.Errorf("parsing batch status: %w", err)
	}

	return c.toBatchStatus(br), nil
}

// GetResults retrieves completed results for a batch (call only when status == "ended").
// Results are returned as newline-delimited JSON (JSONL).
func (c *AnthropicBatchClient) GetResults(ctx context.Context, batchID string) ([]BatchResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/messages/batches/"+batchID+"/results", nil, 60*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic batch results: status %d: %s", resp.StatusCode, string(data))
	}

	var results []BatchResponse
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry anthropicResultLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing result line: %w", err)
		}
		results = append(results, c.toResponseEntry(entry))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading JSONL results: %w", err)
	}

	return results, nil
}

// Cancel cancels an in-progress batch.
func (c *AnthropicBatchClient) Cancel(ctx context.Context, batchID string) error {
	resp, err := c.doRequest(ctx, http.MethodPost, "/messages/batches/"+batchID+"/cancel", nil, 10*time.Second)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic batch cancel: status %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// ---- helpers ----

func (c *AnthropicBatchClient) doRequest(ctx context.Context, method, path string, body []byte, timeout time.Duration) (*http.Response, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(reqCtx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("anthropic-beta", anthropicBatchBeta)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}

func (c *AnthropicBatchClient) buildRequestItem(req BatchRequest) (anthropicBatchRequestItem, error) {
	maxTokens := 1024
	if mt, ok := req.Options["max_tokens"].(int); ok {
		maxTokens = mt
	}

	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	// Anthropic API uses hyphens; config may use dots.
	model = strings.ReplaceAll(model, ".", "-")

	var system []anthropicTextBlock
	var msgs []anthropicMessage

	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			system = append(system, anthropicTextBlock{Type: "text", Text: m.Content})
		case "user", "assistant":
			msgs = append(msgs, anthropicMessage{Role: m.Role, Content: m.Content})
		}
	}

	params := anthropicMessageParams{
		Model:     model,
		MaxTokens: maxTokens,
		Messages:  msgs,
	}
	if len(system) > 0 {
		params.System = system
	}

	return anthropicBatchRequestItem{
		CustomID: req.CustomID,
		Params:   params,
	}, nil
}

func (c *AnthropicBatchClient) toBatchStatus(br anthropicBatchResponse) BatchStatus {
	status := br.ProcessingStatus
	// Map Anthropic's processing_status to our canonical status values.
	// Anthropic uses: "in_progress", "canceling", "ended"
	// Our interface uses: "in_progress", "ended", "canceling", "canceled", "expired"
	// "ended" covers succeeded/errored/expired results — keep as-is.

	return BatchStatus{
		ID:       br.ID,
		Provider: "anthropic",
		Status:   status,
		RequestCounts: RequestCounts{
			Processing: br.RequestCounts.Processing,
			Succeeded:  br.RequestCounts.Succeeded,
			Errored:    br.RequestCounts.Errored,
			Canceled:   br.RequestCounts.Canceled,
			Expired:    br.RequestCounts.Expired,
		},
		CreatedAt:  br.CreatedAt,
		ExpiresAt:  br.ExpiresAt,
		ResultsURL: br.ResultsURL,
	}
}

func (c *AnthropicBatchClient) toResponseEntry(entry anthropicResultLine) BatchResponse {
	resp := BatchResponse{
		CustomID: entry.CustomID,
		Status:   entry.Result.Type,
	}

	switch entry.Result.Type {
	case "succeeded":
		if entry.Result.Message != nil {
			resp.Response = parseAnthropicMessage(entry.Result.Message)
		}
	case "errored":
		if entry.Result.Error != nil {
			resp.Error = entry.Result.Error.Message
		}
	}

	return resp
}

func parseAnthropicMessage(msg *anthropicMsg) *protocoltypes.LLMResponse {
	var sb strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	finishReason := "stop"
	switch msg.StopReason {
	case "tool_use":
		finishReason = "tool_calls"
	case "max_tokens":
		finishReason = "length"
	}

	return &protocoltypes.LLMResponse{
		Content:      sb.String(),
		FinishReason: finishReason,
		Usage: &protocoltypes.UsageInfo{
			PromptTokens:     msg.Usage.InputTokens,
			CompletionTokens: msg.Usage.OutputTokens,
			TotalTokens:      msg.Usage.InputTokens + msg.Usage.OutputTokens,
		},
	}
}
