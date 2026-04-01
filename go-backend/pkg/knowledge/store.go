// OctAi - Knowledge Base Store
// Provides team-shared RAG-powered knowledge management.
// Local mode: SQLite with BM25 text search.
// SaaS mode: PostgreSQL + pgvector (configured externally).
package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Document represents a knowledge base entry.
type Document struct {
	ID        string
	TeamID    string
	Title     string
	Content   string
	SourceURL string
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Chunk is a sub-document segment used for retrieval.
type Chunk struct {
	ID         string
	DocumentID string
	Content    string
	ChunkIndex int
	CreatedAt  time.Time
}

// SearchResult is a chunk returned by a knowledge search.
type SearchResult struct {
	ChunkID    string
	DocumentID string
	Title      string
	Content    string
	Score      float64
}

// KnowledgeStore is the interface for knowledge base operations.
type KnowledgeStore interface {
	// AddDocument ingests a new document and splits it into chunks.
	AddDocument(ctx context.Context, doc Document) (string, error)
	// GetDocument retrieves a document by ID.
	GetDocument(ctx context.Context, id string) (*Document, error)
	// DeleteDocument removes a document and all its chunks.
	DeleteDocument(ctx context.Context, id string) error
	// ListDocuments returns all documents for a team.
	ListDocuments(ctx context.Context, teamID string) ([]Document, error)
	// Search performs BM25 text search over chunks for the given team.
	Search(ctx context.Context, teamID, query string, limit int) ([]SearchResult, error)
	// Close releases resources held by the store.
	Close() error
}

// SQLiteStore is a SQLite-backed KnowledgeStore using full-text search (FTS5/BM25).
type SQLiteStore struct {
	db  *sql.DB
	mu  sync.RWMutex
	dsn string
}

// NewSQLiteStore opens (or creates) a SQLite knowledge base at the given path.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dsn := fmt.Sprintf("file:%s?_journal=WAL&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("knowledge: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite write serialization
	db.SetMaxIdleConns(1)

	store := &SQLiteStore{db: db, dsn: dsn}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("knowledge: migrate: %w", err)
	}
	return store, nil
}

const createTablesSQL = `
CREATE TABLE IF NOT EXISTS kb_documents (
    id          TEXT PRIMARY KEY,
    team_id     TEXT NOT NULL,
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    source_url  TEXT,
    metadata    TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS kb_chunks (
    id          TEXT PRIMARY KEY,
    document_id TEXT NOT NULL REFERENCES kb_documents(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE VIRTUAL TABLE IF NOT EXISTS kb_chunks_fts USING fts5(
    content,
    document_id UNINDEXED,
    chunk_id UNINDEXED,
    title UNINDEXED,
    tokenize='porter unicode61'
);
`

func (s *SQLiteStore) migrate() error {
	_, err := s.db.Exec(createTablesSQL)
	return err
}

// AddDocument ingests a document, splits it into chunks, and indexes them.
func (s *SQLiteStore) AddDocument(ctx context.Context, doc Document) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if doc.ID == "" {
		doc.ID = newID()
	}
	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now

	metaJSON := marshalMetadata(doc.Metadata)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("knowledge: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO kb_documents (id, team_id, title, content, source_url, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.TeamID, doc.Title, doc.Content, doc.SourceURL, metaJSON,
		doc.CreatedAt.UTC().Format(time.RFC3339),
		doc.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("knowledge: insert document: %w", err)
	}

	// Remove stale FTS entries for this document.
	_, _ = tx.ExecContext(ctx,
		`DELETE FROM kb_chunks_fts WHERE document_id = ?`, doc.ID)
	_, _ = tx.ExecContext(ctx,
		`DELETE FROM kb_chunks WHERE document_id = ?`, doc.ID)

	// Split into chunks and index.
	chunks := splitIntoChunks(doc.Content, 512, 64)
	for i, chunkText := range chunks {
		chunkID := fmt.Sprintf("%s-c%d", doc.ID, i)
		_, err = tx.ExecContext(ctx,
			`INSERT INTO kb_chunks (id, document_id, content, chunk_index) VALUES (?, ?, ?, ?)`,
			chunkID, doc.ID, chunkText, i,
		)
		if err != nil {
			return "", fmt.Errorf("knowledge: insert chunk %d: %w", i, err)
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO kb_chunks_fts (content, document_id, chunk_id, title) VALUES (?, ?, ?, ?)`,
			chunkText, doc.ID, chunkID, doc.Title,
		)
		if err != nil {
			return "", fmt.Errorf("knowledge: index chunk %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("knowledge: commit: %w", err)
	}
	return doc.ID, nil
}

// GetDocument retrieves a document by ID.
func (s *SQLiteStore) GetDocument(ctx context.Context, id string) (*Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRowContext(ctx,
		`SELECT id, team_id, title, content, source_url, metadata, created_at, updated_at
		 FROM kb_documents WHERE id = ?`, id)

	var doc Document
	var metaJSON, createdStr, updatedStr string
	err := row.Scan(&doc.ID, &doc.TeamID, &doc.Title, &doc.Content,
		&doc.SourceURL, &metaJSON, &createdStr, &updatedStr)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("knowledge: get document: %w", err)
	}
	doc.Metadata = unmarshalMetadata(metaJSON)
	doc.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	doc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &doc, nil
}

// DeleteDocument removes a document and its chunks from the store and FTS index.
func (s *SQLiteStore) DeleteDocument(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, _ = tx.ExecContext(ctx, `DELETE FROM kb_chunks_fts WHERE document_id = ?`, id)
	_, err = tx.ExecContext(ctx, `DELETE FROM kb_documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("knowledge: delete document: %w", err)
	}
	return tx.Commit()
}

// ListDocuments returns all documents for a team.
func (s *SQLiteStore) ListDocuments(ctx context.Context, teamID string) ([]Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, team_id, title, content, source_url, metadata, created_at, updated_at
		 FROM kb_documents WHERE team_id = ? ORDER BY updated_at DESC`, teamID)
	if err != nil {
		return nil, fmt.Errorf("knowledge: list documents: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		var metaJSON, createdStr, updatedStr string
		if err := rows.Scan(&doc.ID, &doc.TeamID, &doc.Title, &doc.Content,
			&doc.SourceURL, &metaJSON, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		doc.Metadata = unmarshalMetadata(metaJSON)
		doc.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		doc.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// Search performs BM25 full-text search over chunks for the given team.
func (s *SQLiteStore) Search(ctx context.Context, teamID, query string, limit int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 5
	}

	// FTS5 BM25 search: filter by team via JOIN, rank by BM25.
	const searchSQL = `
		SELECT f.chunk_id, f.document_id, f.title, f.content, bm25(kb_chunks_fts) AS score
		FROM kb_chunks_fts f
		JOIN kb_documents d ON d.id = f.document_id
		WHERE kb_chunks_fts MATCH ? AND d.team_id = ?
		ORDER BY score
		LIMIT ?`

	rows, err := s.db.QueryContext(ctx, searchSQL, query, teamID, limit)
	if err != nil {
		return nil, fmt.Errorf("knowledge: search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.Title, &r.Content, &r.Score); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// Close releases the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
