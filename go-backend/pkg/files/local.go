package files

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// LocalFileStore implements FileStore by persisting files on the local filesystem.
// It is intended for providers that do not have a native Files API; callers can
// inline the file content when constructing messages by reading the stored file.
type LocalFileStore struct {
	baseDir string // root directory, e.g. ~/.octai/files/
}

// NewLocalFileStore creates a LocalFileStore rooted at baseDir.
// The directory is created (including any parents) if it does not already exist.
func NewLocalFileStore(baseDir string) (*LocalFileStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("files/local: create base dir %q: %w", baseDir, err)
	}
	return &LocalFileStore{baseDir: baseDir}, nil
}

// Upload writes content to baseDir/<uuid>/<name> and returns a FileInfo whose
// ID is "<uuid>/<name>" and Provider is "local".
func (l *LocalFileStore) Upload(_ context.Context, name, mimeType string, content io.Reader) (FileInfo, error) {
	id := uuid.New().String()
	dir := filepath.Join(l.baseDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileInfo{}, fmt.Errorf("files/local: create dir %q: %w", dir, err)
	}

	dst := filepath.Join(dir, name)
	f, err := os.Create(dst)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/local: create file %q: %w", dst, err)
	}
	defer f.Close()

	n, err := io.Copy(f, content)
	if err != nil {
		return FileInfo{}, fmt.Errorf("files/local: write file %q: %w", dst, err)
	}

	now := time.Now().UTC()
	return FileInfo{
		ID:        id + "/" + name,
		Name:      name,
		MimeType:  mimeType,
		SizeBytes: n,
		Provider:  "local",
		CreatedAt: now,
	}, nil
}

// Get returns metadata for the file identified by fileID ("<uuid>/<name>").
func (l *LocalFileStore) Get(_ context.Context, fileID string) (FileInfo, error) {
	path := filepath.Join(l.baseDir, fileID)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return FileInfo{}, fmt.Errorf("files/local: file %q not found", fileID)
		}
		return FileInfo{}, fmt.Errorf("files/local: stat %q: %w", path, err)
	}

	return FileInfo{
		ID:        fileID,
		Name:      info.Name(),
		SizeBytes: info.Size(),
		Provider:  "local",
		CreatedAt: info.ModTime().UTC(),
	}, nil
}

// Delete removes the file and its parent uuid directory.
func (l *LocalFileStore) Delete(_ context.Context, fileID string) error {
	path := filepath.Join(l.baseDir, fileID)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("files/local: remove %q: %w", path, err)
	}
	// Best-effort: remove the uuid sub-directory if it is now empty.
	dir := filepath.Dir(path)
	_ = os.Remove(dir)
	return nil
}

// List walks the base directory and returns a FileInfo for every stored file.
func (l *LocalFileStore) List(_ context.Context) ([]FileInfo, error) {
	var infos []FileInfo

	err := filepath.Walk(l.baseDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		// Reconstruct the fileID as the path relative to baseDir.
		rel, err := filepath.Rel(l.baseDir, path)
		if err != nil {
			return err
		}

		infos = append(infos, FileInfo{
			ID:        rel,
			Name:      info.Name(),
			SizeBytes: info.Size(),
			Provider:  "local",
			CreatedAt: info.ModTime().UTC(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("files/local: walk %q: %w", l.baseDir, err)
	}
	return infos, nil
}
