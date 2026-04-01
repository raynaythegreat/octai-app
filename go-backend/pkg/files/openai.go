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
	openAIDefaultBaseURL = "https://api.openai.com/v1"
	openAIFilePurpose    = "assistants"
)

// OpenAIFileStore implements FileStore using the OpenAI Files API.
type OpenAIFileStore struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIFileStore creates a new OpenAIFileStore with a 30-second timeout.
func NewOpenAIFileStore(apiKey string) *OpenAIFileStore {
	return &OpenAIFileStore{
		apiKey:  apiKey,
		baseURL: openAIDefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// openAIFileResponse is the JSON shape returned by POST/GET /v1/files.
type openAIFileResponse struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Bytes     int64  `json:"bytes"`
	CreatedAt int64  `json:"created_at"`
	Purpose   string `json:"purpose"`
	Object    string `json:"object"`
}

// openAIListResponse is the JSON shape returned by GET /v1/files.
type openAIListResponse struct {
	Data   []openAIFileResponse `json:"data"`
	Object string               `json:"object"`
}

// openAIDeleteResponse is the JSON shape returned by DELETE /v1/files/{id}.
type openAIDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
	Object  string `json:"object"`
}

func (o *OpenAIFileStore) addHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
}

func openAIToFileInfo(r openAIFileResponse) FileInfo {
	return FileInfo{
		ID:        r.ID,
		Name:      r.Filename,
		SizeBytes: r.Bytes,
		Provider:  "openai",
		CreatedAt: time.Unix(r.CreatedAt, 0).UTC(),
	}
}

// Upload implements FileStore.Upload via POST /v1/files.
func (o *OpenAIFileStore) Upload(ctx context.Context, name, mimeType string, content io.Reader) (FileInfo, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: create form file: %w", err)
	}
	if _, err = io.Copy(fw, content); err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: copy content: %w", err)
	}
	if err = mw.WriteField("purpose", openAIFilePurpose); err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: write purpose field: %w", err)
	}
	if err = mw.Close(); err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/files", &body)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: build request: %w", err)
	}
	o.addHeaders(req)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: upload request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		raw, _ := io.ReadAll(resp.Body)
		return FileInfo{}, fmt.Errorf("files/openai: upload status %d: %s", resp.StatusCode, raw)
	}

	var fr openAIFileResponse
	if err = json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: decode upload response: %w", err)
	}
	fi := openAIToFileInfo(fr)
	fi.MimeType = mimeType
	return fi, nil
}

// Get implements FileStore.Get via GET /v1/files/{id}.
func (o *OpenAIFileStore) Get(ctx context.Context, fileID string) (FileInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/files/"+fileID, nil)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: build get request: %w", err)
	}
	o.addHeaders(req)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: get request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return FileInfo{}, fmt.Errorf("files/openai: get status %d: %s", resp.StatusCode, raw)
	}

	var fr openAIFileResponse
	if err = json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return FileInfo{}, fmt.Errorf("files/openai: decode get response: %w", err)
	}
	return openAIToFileInfo(fr), nil
}

// Delete implements FileStore.Delete via DELETE /v1/files/{id}.
func (o *OpenAIFileStore) Delete(ctx context.Context, fileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, o.baseURL+"/files/"+fileID, nil)
	if err != nil {
		return fmt.Errorf("files/openai: build delete request: %w", err)
	}
	o.addHeaders(req)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("files/openai: delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("files/openai: delete status %d: %s", resp.StatusCode, raw)
	}

	// Optionally decode the deletion confirmation
	var dr openAIDeleteResponse
	_ = json.NewDecoder(resp.Body).Decode(&dr)
	return nil
}

// List implements FileStore.List via GET /v1/files.
func (o *OpenAIFileStore) List(ctx context.Context) ([]FileInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.baseURL+"/files", nil)
	if err != nil {
		return nil, fmt.Errorf("files/openai: build list request: %w", err)
	}
	o.addHeaders(req)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("files/openai: list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("files/openai: list status %d: %s", resp.StatusCode, raw)
	}

	var lr openAIListResponse
	if err = json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("files/openai: decode list response: %w", err)
	}

	infos := make([]FileInfo, len(lr.Data))
	for i, fr := range lr.Data {
		infos[i] = openAIToFileInfo(fr)
	}
	return infos, nil
}
