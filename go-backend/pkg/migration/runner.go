package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type MigrationStatus struct {
	Version   string    `json:"version"`
	Name      string    `json:"name"`
	Applied   bool      `json:"applied"`
	AppliedAt time.Time `json:"applied_at,omitempty"`
}

type Migration struct {
	Version string
	Name    string
	SQL     string
}

type MigrationRunner struct {
	db         *sql.DB
	migrations []Migration
}

type Option func(*MigrationRunner) error

func WithEmbeddedMigrations(fsys embed.FS, dir string) Option {
	return func(r *MigrationRunner) error {
		return r.loadFromFS(fsys, dir)
	}
}

func WithMigrationsDir(dir string) Option {
	return func(r *MigrationRunner) error {
		return r.loadFromDirectory(dir)
	}
}

func NewMigrationRunner(db *sql.DB, opts ...Option) (*MigrationRunner, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	runner := &MigrationRunner{
		db: db,
	}

	for _, opt := range opts {
		if err := opt(runner); err != nil {
			return nil, err
		}
	}

	if len(runner.migrations) == 0 {
		if err := runner.loadFromDirectory("migrations"); err != nil {
			return nil, fmt.Errorf("failed to load migrations: %w", err)
		}
	}

	return runner, nil
}

func (r *MigrationRunner) loadFromDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		if err := r.addMigration(entry.Name(), string(content)); err != nil {
			return err
		}
	}

	r.sortMigrations()
	return nil
}

func (r *MigrationRunner) loadFromFS(fsys embed.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations from embedded FS: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		if err := r.addMigration(entry.Name(), string(content)); err != nil {
			return err
		}
	}

	r.sortMigrations()
	return nil
}

func (r *MigrationRunner) addMigration(filename, content string) error {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return nil
	}

	version := parts[0]
	name := strings.TrimSuffix(parts[1], ".sql")

	r.migrations = append(r.migrations, Migration{
		Version: version,
		Name:    name,
		SQL:     content,
	})

	return nil
}

func (r *MigrationRunner) sortMigrations() {
	sort.Slice(r.migrations, func(i, j int) bool {
		vi, _ := strconv.Atoi(r.migrations[i].Version)
		vj, _ := strconv.Atoi(r.migrations[j].Version)
		return vi < vj
	})
}

func (r *MigrationRunner) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

func (r *MigrationRunner) isApplied(ctx context.Context, version string) (bool, time.Time, error) {
	var appliedAt sql.NullTime
	query := `SELECT applied_at FROM schema_migrations WHERE version = ?`
	err := r.db.QueryRowContext(ctx, query, version).Scan(&appliedAt)
	if err == sql.ErrNoRows {
		return false, time.Time{}, nil
	}
	if err != nil {
		return false, time.Time{}, err
	}
	return true, appliedAt.Time, nil
}

func (r *MigrationRunner) Run(ctx context.Context) error {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	for _, m := range r.migrations {
		applied, _, err := r.isApplied(ctx, m.Version)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", m.Version, err)
		}

		if applied {
			continue
		}

		fmt.Printf("Running migration %s: %s\n", m.Version, m.Name)

		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", m.Version, err)
		}

		if err := r.recordMigrationInTx(ctx, tx, m.Version, m.Name); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", m.Version, err)
		}

		fmt.Printf("Completed migration %s\n", m.Version)
	}

	return nil
}

func (r *MigrationRunner) recordMigrationInTx(ctx context.Context, tx *sql.Tx, version, name string) error {
	query := `INSERT INTO schema_migrations (version, name) VALUES (?, ?)`
	_, err := tx.ExecContext(ctx, query, version, name)
	return err
}

func (r *MigrationRunner) Rollback(ctx context.Context, version string) error {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, _, err := r.isApplied(ctx, version)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if !applied {
		return fmt.Errorf("migration %s is not applied", version)
	}

	var migration *Migration
	for _, m := range r.migrations {
		if m.Version == version {
			migration = &m
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration %s not found", version)
	}

	fmt.Printf("Warning: Rollback for %s requires manual down migration\n", version)
	fmt.Printf("Removing migration record for %s\n", version)

	if err := r.removeMigrationRecord(ctx, version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return nil
}

func (r *MigrationRunner) removeMigrationRecord(ctx context.Context, version string) error {
	query := `DELETE FROM schema_migrations WHERE version = ?`
	_, err := r.db.ExecContext(ctx, query, version)
	return err
}

func (r *MigrationRunner) GetStatus(ctx context.Context) ([]MigrationStatus, error) {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	statuses := make([]MigrationStatus, 0, len(r.migrations))

	for _, m := range r.migrations {
		applied, appliedAt, err := r.isApplied(ctx, m.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to check migration %s: %w", m.Version, err)
		}

		statuses = append(statuses, MigrationStatus{
			Version:   m.Version,
			Name:      m.Name,
			Applied:   applied,
			AppliedAt: appliedAt,
		})
	}

	return statuses, nil
}

func (r *MigrationRunner) GetPendingCount(ctx context.Context) (int, error) {
	if err := r.ensureMigrationsTable(ctx); err != nil {
		return 0, err
	}

	count := 0
	for _, m := range r.migrations {
		applied, _, err := r.isApplied(ctx, m.Version)
		if err != nil {
			return 0, err
		}
		if !applied {
			count++
		}
	}

	return count, nil
}

func (r *MigrationRunner) GetMigrations() []Migration {
	return r.migrations
}
