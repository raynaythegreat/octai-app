// OctAi - Workflow Store
// Persists workflow definitions and run history to SQLite.
package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store persists workflow definitions and run history.
type Store interface {
	// SaveWorkflow upserts a workflow definition.
	SaveWorkflow(ctx context.Context, def WorkflowDefinition) error
	// GetWorkflow retrieves a workflow definition by ID.
	GetWorkflow(ctx context.Context, id string) (*WorkflowDefinition, error)
	// ListWorkflows returns all workflow definitions for a team.
	ListWorkflows(ctx context.Context, teamID string) ([]WorkflowDefinition, error)
	// DeleteWorkflow removes a workflow definition and its run history.
	DeleteWorkflow(ctx context.Context, id string) error

	// SaveRun upserts a workflow run.
	SaveRun(ctx context.Context, run *WorkflowRun) error
	// GetRun retrieves a workflow run by ID.
	GetRun(ctx context.Context, id string) (*WorkflowRun, error)
	// ListRuns returns recent runs for a workflow.
	ListRuns(ctx context.Context, workflowID string, limit int) ([]WorkflowRun, error)
}

// SQLiteWorkflowStore persists workflows and runs to SQLite.
type SQLiteWorkflowStore struct {
	db *sql.DB
}

// NewSQLiteWorkflowStore opens (or creates) the SQLite workflow store at dbPath.
func NewSQLiteWorkflowStore(dbPath string) (*SQLiteWorkflowStore, error) {
	dsn := fmt.Sprintf("file:%s?_journal=WAL&_foreign_keys=on", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("workflow store: open: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &SQLiteWorkflowStore{db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("workflow store: migrate: %w", err)
	}
	return store, nil
}

const workflowSchema = `
CREATE TABLE IF NOT EXISTS workflows (
    id          TEXT PRIMARY KEY,
    team_id     TEXT,
    org_id      TEXT,
    name        TEXT NOT NULL,
    definition  TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workflow_runs (
    id          TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status      TEXT NOT NULL,
    node_states TEXT,
    started_at  DATETIME,
    ended_at    DATETIME,
    error       TEXT
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_workflow ON workflow_runs(workflow_id, started_at DESC);
`

func (s *SQLiteWorkflowStore) migrate() error {
	_, err := s.db.Exec(workflowSchema)
	return err
}

func (s *SQLiteWorkflowStore) SaveWorkflow(ctx context.Context, def WorkflowDefinition) error {
	defJSON, err := json.Marshal(def)
	if err != nil {
		return fmt.Errorf("workflow store: marshal: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO workflows (id, team_id, org_id, name, definition, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, 'active', COALESCE((SELECT created_at FROM workflows WHERE id=?), ?), ?)`,
		def.ID, def.TeamID, def.OrgID, def.Name, string(defJSON),
		def.ID, now, now,
	)
	return err
}

func (s *SQLiteWorkflowStore) GetWorkflow(ctx context.Context, id string) (*WorkflowDefinition, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT definition FROM workflows WHERE id = ?`, id)
	var defJSON string
	if err := row.Scan(&defJSON); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var def WorkflowDefinition
	if err := json.Unmarshal([]byte(defJSON), &def); err != nil {
		return nil, err
	}
	return &def, nil
}

func (s *SQLiteWorkflowStore) ListWorkflows(ctx context.Context, teamID string) ([]WorkflowDefinition, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT definition FROM workflows WHERE team_id = ? ORDER BY updated_at DESC`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []WorkflowDefinition
	for rows.Next() {
		var defJSON string
		if err := rows.Scan(&defJSON); err != nil {
			return nil, err
		}
		var def WorkflowDefinition
		if err := json.Unmarshal([]byte(defJSON), &def); err != nil {
			continue
		}
		defs = append(defs, def)
	}
	return defs, rows.Err()
}

func (s *SQLiteWorkflowStore) DeleteWorkflow(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workflows WHERE id = ?`, id)
	return err
}

func (s *SQLiteWorkflowStore) SaveRun(ctx context.Context, run *WorkflowRun) error {
	statesJSON, _ := json.Marshal(run.NodeStates)
	startedStr := run.StartedAt.UTC().Format(time.RFC3339)
	var endedStr string
	if !run.EndedAt.IsZero() {
		endedStr = run.EndedAt.UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO workflow_runs
		 (id, workflow_id, status, node_states, started_at, ended_at, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.WorkflowID, string(run.Status),
		string(statesJSON), startedStr, endedStr, run.Error,
	)
	return err
}

func (s *SQLiteWorkflowStore) GetRun(ctx context.Context, id string) (*WorkflowRun, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, workflow_id, status, node_states, started_at, ended_at, error
		 FROM workflow_runs WHERE id = ?`, id)
	return scanRun(row)
}

func (s *SQLiteWorkflowStore) ListRuns(ctx context.Context, workflowID string, limit int) ([]WorkflowRun, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, workflow_id, status, node_states, started_at, ended_at, error
		 FROM workflow_runs WHERE workflow_id = ?
		 ORDER BY started_at DESC LIMIT ?`, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []WorkflowRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		if run != nil {
			runs = append(runs, *run)
		}
	}
	return runs, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanRun(s scanner) (*WorkflowRun, error) {
	var run WorkflowRun
	var statesJSON, startedStr, endedStr string
	err := s.Scan(&run.ID, &run.WorkflowID, &run.Status,
		&statesJSON, &startedStr, &endedStr, &run.Error)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(statesJSON), &run.NodeStates)
	run.StartedAt, _ = time.Parse(time.RFC3339, startedStr)
	if endedStr != "" {
		run.EndedAt, _ = time.Parse(time.RFC3339, endedStr)
	}
	return &run, nil
}
