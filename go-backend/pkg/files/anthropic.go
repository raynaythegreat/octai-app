package files

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com/v1"
	anthropicVersion        = "2023-06-01"
	anthropicBeta           = "files-api-2025-04-14"
)

// AnthropicFileStore implements FileStore using the Anthropic Files API.
type AnthropicFileStore struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewAnthropicFileStore creates a new AnthropicFileStore with a 30-second timeout.
func NewAnthropicFileStore(apiKey string) *AnthropicFileStore {
	return &AnthropicFileStore{
		apiKey:  apiKey,
		baseURL: anthropicDefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// anthropicFileResponse is the JSON shape returned by POST/GET /v1/files.
type anthropicFileResponse struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Size      int64  `json:"size"`
	CreatedAt int64  `json:"created_at"`
	Purpose   string `json:"purpose"`
}

// anthropicListResponse is the JSON shape returned by GET /v1/files.
type anthropicListResponse struct {
	Data []anthropicFileResponse `json:"data"`
}

func (a *AnthropicFileStore) addHeaders(req *http.Request) {
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("anthropic-beta", anthropicBeta)
}

func anthropicToFileInfo(r anthropicFileResponse) FileInfo {
	return FileInfo{
		ID:        r.ID,
		Name:      r.Filename,
		SizeBytes: r.Size,
		Provider:  "anthropic",
		CreatedAt: time.Unix(r.CreatedAt, 0).UTC(),
	}
}

// Upload implements FileStore.Upload via POST /v1/files.
func (a *AnthropicFileStore) Upload(ctx context.Context, name, mimeType string, content io.Reader) (FileInfo, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: create form file: %w", err)
	}
	if _, err = io.Copy(fw, content); err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: copy content: %w", err)
	}
	if err = mw.Close(); err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/files", &body)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: build request: %w", err)
	}
	a.addHeaders(req)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return FileInfo{}, fmt.Errorf("files/anthropic: upload status %d: %s", resp.StatusCode, raw)
	}

	var fr anthropicFileResponse
	if err = json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: decode upload response: %w", err)
	}
	fi := anthropicToFileInfo(fr)
	// Preserve the caller's MIME type since the API doesn't echo it back.
	fi.MimeType = mimeType
	return fi, nil
}

// Get implements FileStore.Get via GET /v1/files/{id}.
func (a *AnthropicFileStore) Get(ctx context.Context, fileID string) (FileInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/files/"+fileID, nil)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: build get request: %w", err)
	}
	a.addHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: get request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return FileInfo{}, fmt.Errorf("files/anthropic: get status %d: %s", resp.StatusCode, raw)
	}

	var fr anthropicFileResponse
	if err = json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return FileInfo{}, fmt.Errorf("files/anthropic: decode get response: %w", err)
	}
	return anthropicToFileInfo(fr), nil
}

// Delete implements FileStore.Delete via DELETE /v1/files/{id}.
func (a *AnthropicFileStore) Delete(ctx context.Context, fileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, a.baseURL+"/files/"+fileID, nil)
	if err != nil {
		return fmt.Errorf("files/anthropic: build delete request: %w", err)
	}
	a.addHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("files/anthropic: delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("files/anthropic: delete status %d: %s", resp.StatusCode, raw)
	}
	return nil
}

// List implements FileStore.List via GET /v1/files.
func (a *AnthropicFileStore) List(ctx context.Context) ([]FileInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/files", nil)
	if err != nil {
		return nil, fmt.Errorf("files/anthropic: build list request: %w", err)
	}
	a.addHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("files/anthropic: list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("files/anthropic: list status %d: %s", resp.StatusCode, raw)
	}

	var lr anthropicListResponse
	if err = json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("files/anthropic: decode list response: %w", err)
	}

	infos := make([]FileInfo, len(lr.Data))
	for i, fr := range lr.Data {
		infos[i] = anthropicToFileInfo(fr)
	}
	return infos, nil
}
