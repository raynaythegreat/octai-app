package batch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/providers/protocoltypes"
)

const (
	openaiDefaultBaseURL = "https://api.openai.com/v1"
)

// OpenAIBatchClient implements BatchClient for the OpenAI Batch API.
//
// OpenAI Batch API flow:
//  1. Upload a JSONL file of requests via the Files API.
//  2. Create a batch referencing that file.
//  3. Poll the batch status until it reaches "completed" / "failed" / "expired" / "cancelled".
//  4. Download the output file and parse results.
type OpenAIBatchClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIBatchClient creates a new OpenAI batch client with the given API key.
func NewOpenAIBatchClient(apiKey string) *OpenAIBatchClient {
	return &OpenAIBatchClient{
		apiKey:  apiKey,
		baseURL: openaiDefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ---- wire types for OpenAI Batch / Files APIs ----

// openaiJSONLRequest is one line in the uploaded JSONL file.
type openaiJSONLRequest struct {
	CustomID string              `json:"custom_id"`
	Method   string              `json:"method"`
	URL      string              `json:"url"`
	Body     openaiChatBody      `json:"body"`
}

type openaiChatBody struct {
	Model     string          `json:"model"`
	Messages  []openaiMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens,omitempty"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openaiFileResponse is the response from the Files API upload.
type openaiFileResponse struct {
	ID       string `json:"id"`
	Object   string `json:"object"`
	Filename string `json:"filename"`
}

// openaiCreateBatchRequest is the body for POST /v1/batches.
type openaiCreateBatchRequest struct {
	InputFileID      string `json:"input_file_id"`
	Endpoint         string `json:"endpoint"`
	CompletionWindow string `json:"completion_window"`
}

// openaiBatchResponse is the response from POST /v1/batches and GET /v1/batches/{id}.
type openaiBatchResponse struct {
	ID             string               `json:"id"`
	Object         string               `json:"object"`
	Status         string               `json:"status"` // "validating", "failed", "in_progress", "finalizing", "completed", "expired", "cancelling", "cancelled"
	RequestCounts  openaiRequestCounts  `json:"request_counts"`
	CreatedAt      int64                `json:"created_at"`
	ExpiresAt      int64                `json:"expires_at"`
	OutputFileID   string               `json:"output_file_id,omitempty"`
	ErrorFileID    string               `json:"error_file_id,omitempty"`
}

type openaiRequestCounts struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
}

// openaiResultLine is one JSONL line in the output file.
type openaiResultLine struct {
	ID       string             `json:"id"`
	CustomID string             `json:"custom_id"`
	Response *openaiResult      `json:"response,omitempty"`
	Error    *openaiResultError `json:"error,omitempty"`
}

type openaiResult struct {
	StatusCode int             `json:"status_code"`
	Body       openaiChatResp  `json:"body"`
}

type openaiChatResp struct {
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
}

type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiResultError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ---- BatchClient implementation ----

// Submit uploads a JSONL file and creates a batch.
func (c *OpenAIBatchClient) Submit(ctx context.Context, requests []BatchRequest) (BatchStatus, error) {
	// Build the JSONL payload.
	var jsonlBuf bytes.Buffer
	for _, req := range requests {
		line, err := c.buildJSONLLine(req)
		if err != nil {
			return BatchStatus{}, fmt.Errorf("building JSONL line for %q: %w", req.CustomID, err)
		}
		jsonlBuf.Write(line)
		jsonlBuf.WriteByte('\n')
	}

	// Step 1: Upload the file.
	fileID, err := c.uploadFile(ctx, jsonlBuf.Bytes())
	if err != nil {
		return BatchStatus{}, fmt.Errorf("uploading batch file: %w", err)
	}

	// Step 2: Create the batch.
	createReq := openaiCreateBatchRequest{
		InputFileID:      fileID,
		Endpoint:         "/v1/chat/completions",
		CompletionWindow: "24h",
	}
	body, err := json.Marshal(createReq)
	if err != nil {
		return BatchStatus{}, fmt.Errorf("marshaling batch create request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/batches", body, "application/json", 10*time.Second)
	if err != nil {
		return BatchStatus{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return BatchStatus{}, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return BatchStatus{}, fmt.Errorf("openai batch create: status %d: %s", resp.StatusCode, string(data))
	}

	var br openaiBatchResponse
	if err := json.Unmarshal(data, &br); err != nil {
		return BatchStatus{}, fmt.Errorf("parsing batch response: %w", err)
	}

	return c.toBatchStatus(br), nil
}

// Poll returns the current status of a batch.
func (c *OpenAIBatchClient) Poll(ctx context.Context, batchID string) (BatchStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/batches/"+batchID, nil, "application/json", 10*time.Second)
	if err != nil {
		return BatchStatus{}, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return BatchStatus{}, fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return BatchStatus{}, fmt.Errorf("openai batch poll: status %d: %s", resp.StatusCode, string(data))
	}

	var br openaiBatchResponse
	if err := json.Unmarshal(data, &br); err != nil {
		return BatchStatus{}, fmt.Errorf("parsing batch status: %w", err)
	}

	return c.toBatchStatus(br), nil
}

// GetResults retrieves completed results for a batch (call only when status == "ended" / "completed").
func (c *OpenAIBatchClient) GetResults(ctx context.Context, batchID string) ([]BatchResponse, error) {
	// First poll to get the output_file_id.
	status, err := c.Poll(ctx, batchID)
	if err != nil {
		return nil, err
	}
	if status.ResultsURL == "" {
		return nil, fmt.Errorf("openai batch %q has no output file (status: %s)", batchID, status.Status)
	}

	// Download the output file.
	// status.ResultsURL holds the file ID for OpenAI.
	resp, err := c.doRequest(ctx, http.MethodGet, "/files/"+status.ResultsURL+"/content", nil, "application/json", 60*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai file download: status %d: %s", resp.StatusCode, string(data))
	}

	var results []BatchResponse
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry openaiResultLine
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
func (c *OpenAIBatchClient) Cancel(ctx context.Context, batchID string) error {
	resp, err := c.doRequest(ctx, http.MethodPost, "/batches/"+batchID+"/cancel", nil, "application/json", 10*time.Second)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai batch cancel: status %d: %s", resp.StatusCode, string(data))
	}
	return nil
}

// ---- helpers ----

func (c *OpenAIBatchClient) doRequest(ctx context.Context, method, path string, body []byte, contentType string, timeout time.Duration) (*http.Response, error) {
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

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	return resp, nil
}

// uploadFile uploads a JSONL payload to the OpenAI Files API using multipart/form-data.
func (c *OpenAIBatchClient) uploadFile(ctx context.Context, jsonlData []byte) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// purpose field
	if err := mw.WriteField("purpose", "batch"); err != nil {
		return "", fmt.Errorf("writing purpose field: %w", err)
	}

	// file field
	fw, err := mw.CreateFormFile("file", "batch.jsonl")
	if err != nil {
		return "", fmt.Errorf("creating form file: %w", err)
	}
	if _, err := fw.Write(jsonlData); err != nil {
		return "", fmt.Errorf("writing JSONL to form: %w", err)
	}
	if err := mw.Close(); err != nil {
		return "", fmt.Errorf("closing multipart writer: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.baseURL+"/files", &buf)
	if err != nil {
		return "", fmt.Errorf("creating file upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("uploading file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading file upload response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai file upload: status %d: %s", resp.StatusCode, string(data))
	}

	var fr openaiFileResponse
	if err := json.Unmarshal(data, &fr); err != nil {
		return "", fmt.Errorf("parsing file upload response: %w", err)
	}
	return fr.ID, nil
}

func (c *OpenAIBatchClient) buildJSONLLine(req BatchRequest) ([]byte, error) {
	model := req.Model
	if model == "" {
		model = "gpt-4o"
	}

	maxTokens := 0
	if mt, ok := req.Options["max_tokens"].(int); ok {
		maxTokens = mt
	}

	var msgs []openaiMessage
	for _, m := range req.Messages {
		if m.Role == "system" || m.Role == "user" || m.Role == "assistant" {
			msgs = append(msgs, openaiMessage{Role: m.Role, Content: m.Content})
		}
	}

	line := openaiJSONLRequest{
		CustomID: req.CustomID,
		Method:   "POST",
		URL:      "/v1/chat/completions",
		Body: openaiChatBody{
			Model:     model,
			Messages:  msgs,
			MaxTokens: maxTokens,
		},
	}

	return json.Marshal(line)
}

// toBatchStatus converts an OpenAI batch response to our canonical BatchStatus.
// OpenAI statuses: "validating", "failed", "in_progress", "finalizing", "completed", "expired", "cancelling", "cancelled"
// Our interface status: "in_progress", "ended", "canceling", "canceled", "expired"
func (c *OpenAIBatchClient) toBatchStatus(br openaiBatchResponse) BatchStatus {
	status := br.Status
	switch br.Status {
	case "completed", "failed", "finalizing":
		status = "ended"
	case "cancelling":
		status = "canceling"
	case "cancelled":
		status = "canceled"
	case "validating":
		status = "in_progress"
	}

	return BatchStatus{
		ID:       br.ID,
		Provider: "openai",
		Status:   status,
		RequestCounts: RequestCounts{
			Processing: br.RequestCounts.Total - br.RequestCounts.Completed - br.RequestCounts.Failed,
			Succeeded:  br.RequestCounts.Completed,
			Errored:    br.RequestCounts.Failed,
		},
		CreatedAt:  time.Unix(br.CreatedAt, 0),
		ExpiresAt:  time.Unix(br.ExpiresAt, 0),
		ResultsURL: br.OutputFileID, // file ID used to download results
	}
}

func (c *OpenAIBatchClient) toResponseEntry(entry openaiResultLine) BatchResponse {
	resp := BatchResponse{
		CustomID: entry.CustomID,
	}

	if entry.Error != nil {
		resp.Status = "errored"
		resp.Error = entry.Error.Message
		return resp
	}

	if entry.Response == nil {
		resp.Status = "errored"
		resp.Error = "empty response"
		return resp
	}

	if entry.Response.StatusCode != http.StatusOK {
		resp.Status = "errored"
		resp.Error = fmt.Sprintf("status code %d", entry.Response.StatusCode)
		return resp
	}

	resp.Status = "succeeded"
	resp.Response = parseOpenAIResponse(&entry.Response.Body)
	return resp
}

func parseOpenAIResponse(body *openaiChatResp) *protocoltypes.LLMResponse {
	var content strings.Builder
	finishReason := "stop"

	for _, choice := range body.Choices {
		content.WriteString(choice.Message.Content)
		if choice.FinishReason != "" {
			finishReason = choice.FinishReason
		}
	}

	return &protocoltypes.LLMResponse{
		Content:      content.String(),
		FinishReason: finishReason,
		Usage: &protocoltypes.UsageInfo{
			PromptTokens:     body.Usage.PromptTokens,
			CompletionTokens: body.Usage.CompletionTokens,
			TotalTokens:      body.Usage.TotalTokens,
		},
	}
}
