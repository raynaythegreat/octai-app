package files

import (
	"context"
	"io"
	"time"
)

// FileInfo describes an uploaded file
type FileInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	MimeType  string     `json:"mime_type"`
	SizeBytes int64      `json:"size_bytes"`
	Provider  string     `json:"provider"` // "anthropic", "openai", "local"
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// FileStore is the interface for uploading and retrieving files
type FileStore interface {
	// Upload sends a file to the provider and returns its FileInfo with ID
	Upload(ctx context.Context, name, mimeType string, content io.Reader) (FileInfo, error)
	// Get retrieves file metadata by ID
	Get(ctx context.Context, fileID string) (FileInfo, error)
	// Delete removes a file
	Delete(ctx context.Context, fileID string) error
	// List returns all uploaded files
	List(ctx context.Context) ([]FileInfo, error)
}
