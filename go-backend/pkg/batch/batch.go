package batch

import (
	"context"
	"time"

	"github.com/raynaythegreat/octai-app/pkg/providers/protocoltypes"
)

// BatchRequest is a single request in a batch
type BatchRequest struct {
	CustomID string               `json:"custom_id"` // your identifier
	Messages []protocoltypes.Message
	Model    string
	Options  map[string]any
}

// BatchResponse is the result of a single request in a batch
type BatchResponse struct {
	CustomID string                     `json:"custom_id"`
	Status   string                     `json:"status"` // "succeeded", "errored", "expired"
	Response *protocoltypes.LLMResponse `json:"response,omitempty"`
	Error    string                     `json:"error,omitempty"`
}

// BatchStatus tracks a submitted batch
type BatchStatus struct {
	ID            string        `json:"id"`
	Provider      string        `json:"provider"` // "anthropic" or "openai"
	Status        string        `json:"status"`   // "in_progress", "ended", "canceling", "canceled", "expired"
	RequestCounts RequestCounts `json:"request_counts"`
	CreatedAt     time.Time     `json:"created_at"`
	ExpiresAt     time.Time     `json:"expires_at"`
	ResultsURL    string        `json:"results_url,omitempty"`
}

type RequestCounts struct {
	Processing int `json:"processing"`
	Succeeded  int `json:"succeeded"`
	Errored    int `json:"errored"`
	Canceled   int `json:"canceled"`
	Expired    int `json:"expired"`
}

// BatchClient is the interface all batch providers implement
type BatchClient interface {
	// Submit sends a batch of requests, returns batch ID
	Submit(ctx context.Context, requests []BatchRequest) (BatchStatus, error)
	// Poll returns current status of a batch
	Poll(ctx context.Context, batchID string) (BatchStatus, error)
	// GetResults retrieves completed results (only call when status == "ended")
	GetResults(ctx context.Context, batchID string) ([]BatchResponse, error)
	// Cancel cancels an in-progress batch
	Cancel(ctx context.Context, batchID string) error
}
