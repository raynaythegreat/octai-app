// OctAi - Agent Memory Store
// Persistent key-value store backed by SQLite that allows agents to remember
// information across sessions.
package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// MemoryType categorizes memories for better retrieval.
type MemoryType string

const (
	MemoryTypeUser      MemoryType = "user"
	MemoryTypeProject   MemoryType = "project"
	MemoryTypeFeedback  MemoryType = "feedback"
	MemoryTypeReference MemoryType = "reference"
	MemoryTypeFact      MemoryType = "fact"
)

// AgentMemory is a persistent piece of information stored by an agent.
type AgentMemory struct {
	ID          string     `json:"id"`
	AgentID     string     `json:"agent_id"`             // which agent stored it (namespace)
	Type        MemoryType `json:"type"`
	Content     string     `json:"content"`
	Description string     `json:"description,omitempty"` // one-line summary for retrieval
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SQLiteAgentMemoryStore is a SQLite-backed persistent memory store for agents.
type SQLiteAgentMemoryStore struct {
	db *sql.DB
}

const agentMemorySchema = `
CREATE TABLE IF NOT EXISTS agent_memories (
    id          TEXT PRIMARY KEY,
    agent_id    TEXT NOT NULL,
    type        TEXT NOT NULL,
    content     TEXT NOT NULL,
    description TEXT,
    tags        TEXT,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_memories_agent ON agent_memories(agent_id);
CREATE INDEX IF NOT EXISTS idx_memories_type  ON agent_memories(agent_id, type);
`

// NewSQLiteAgentMemoryStore opens (or creates) the SQLite agent memory store at dbPath.
// It uses the same WAL + foreign-keys DSN pattern as the workflow store.
func NewSQLiteAgentMemoryStore(dbPath string) (*SQLiteAgentMemoryStore, error) {
	dsn := fmt.Sprintf("file:%s?_journal=WAL&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("agent memory store: open: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &SQLiteAgentMemoryStore{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("agent memory store: migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteAgentMemoryStore) migrate() error {
	_, err := s.db.Exec(agentMemorySchema)
	return err
}

// Close releases the underlying database connection.
func (s *SQLiteAgentMemoryStore) Close() error {
	return s.db.Close()
}

// Save stores a memory (insert or upsert by ID).
// If a memory with the same ID already exists it is replaced, preserving the
// original created_at timestamp.
func (s *SQLiteAgentMemoryStore) Save(ctx context.Context, m AgentMemory) error {
	tagsJSON, err := json.Marshal(m.Tags)
	if err != nil {
		return fmt.Errorf("agent memory store: marshal tags: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO agent_memories
		 (id, agent_id, type, content, description, tags, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?,
		         COALESCE((SELECT created_at FROM agent_memories WHERE id=?), ?),
		         ?)`,
		m.ID, m.AgentID, string(m.Type), m.Content, m.Description, string(tagsJSON),
		m.ID, now,
		now,
	)
	if err != nil {
		return fmt.Errorf("agent memory store: save: %w", err)
	}
	return nil
}

// Get retrieves a memory by ID. Returns nil, nil when the ID does not exist.
func (s *SQLiteAgentMemoryStore) Get(ctx context.Context, id string) (*AgentMemory, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, agent_id, type, content, description, tags, created_at, updated_at
		 FROM agent_memories WHERE id = ?`, id)
	return scanAgentMemory(row)
}

// Delete removes a memory by ID.
func (s *SQLiteAgentMemoryStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM agent_memories WHERE id = ?`, id)
	return err
}

// Search finds memories matching a query string using keyword search across
// the content, description, and tags columns. All space-separated terms must
// match at least one of those columns (AND logic across terms, OR across
// columns per term).
//
// agentID may be "" to search across all agents.
// limit <= 0 defaults to 10.
func (s *SQLiteAgentMemoryStore) Search(ctx context.Context, agentID, query string, limit int) ([]AgentMemory, error) {
	if limit <= 0 {
		limit = 10
	}

	terms := strings.Fields(query)
	if len(terms) == 0 {
		return s.List(ctx, agentID, limit)
	}

	// Build per-term LIKE clauses:
	//   (content LIKE ? OR description LIKE ? OR tags LIKE ?)
	// All term groups are combined with AND.
	var (
		whereParts []string
		args       []any
	)

	for _, term := range terms {
		like := "%" + term + "%"
		whereParts = append(whereParts,
			"(content LIKE ? OR description LIKE ? OR tags LIKE ?)")
		args = append(args, like, like, like)
	}

	whereClause := strings.Join(whereParts, " AND ")

	var query_str string
	if agentID != "" {
		query_str = fmt.Sprintf(
			`SELECT id, agent_id, type, content, description, tags, created_at, updated_at
			 FROM agent_memories
			 WHERE agent_id = ? AND (%s)
			 ORDER BY updated_at DESC LIMIT ?`,
			whereClause,
		)
		args = append([]any{agentID}, args...)
	} else {
		query_str = fmt.Sprintf(
			`SELECT id, agent_id, type, content, description, tags, created_at, updated_at
			 FROM agent_memories
			 WHERE %s
			 ORDER BY updated_at DESC LIMIT ?`,
			whereClause,
		)
	}
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query_str, args...)
	if err != nil {
		return nil, fmt.Errorf("agent memory store: search: %w", err)
	}
	defer rows.Close()

	return collectAgentMemories(rows)
}

// List returns recent memories for an agent ordered by updated_at DESC.
// agentID may be "" to list across all agents.
// limit <= 0 defaults to 10.
func (s *SQLiteAgentMemoryStore) List(ctx context.Context, agentID string, limit int) ([]AgentMemory, error) {
	if limit <= 0 {
		limit = 10
	}

	var (
		rows *sql.Rows
		err  error
	)
	if agentID != "" {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, agent_id, type, content, description, tags, created_at, updated_at
			 FROM agent_memories
			 WHERE agent_id = ?
			 ORDER BY updated_at DESC LIMIT ?`,
			agentID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx,
			`SELECT id, agent_id, type, content, description, tags, created_at, updated_at
			 FROM agent_memories
			 ORDER BY updated_at DESC LIMIT ?`,
			limit)
	}
	if err != nil {
		return nil, fmt.Errorf("agent memory store: list: %w", err)
	}
	defer rows.Close()

	return collectAgentMemories(rows)
}

// --- helpers ---

type agentMemoryScanner interface {
	Scan(dest ...any) error
}

func scanAgentMemory(s agentMemoryScanner) (*AgentMemory, error) {
	var m AgentMemory
	var tagsJSON, createdStr, updatedStr string
	err := s.Scan(
		&m.ID, &m.AgentID, &m.Type, &m.Content,
		&m.Description, &tagsJSON, &createdStr, &updatedStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(tagsJSON), &m.Tags)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &m, nil
}

func collectAgentMemories(rows *sql.Rows) ([]AgentMemory, error) {
	var memories []AgentMemory
	for rows.Next() {
		m, err := scanAgentMemory(rows)
		if err != nil {
			return nil, err
		}
		if m != nil {
			memories = append(memories, *m)
		}
	}
	return memories, rows.Err()
}
