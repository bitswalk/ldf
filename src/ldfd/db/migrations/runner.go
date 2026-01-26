// Package migrations provides database schema versioning and migration support.
package migrations

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/bitswalk/ldf/src/common/logs"
)

// package-level logger, can be set via SetLogger
var log *logs.Logger

// SetLogger sets the logger for the migrations package
func SetLogger(l *logs.Logger) {
	log = l
}

// Migration represents a single database migration
type Migration struct {
	Version     int
	Description string
	Up          func(tx *sql.Tx) error
}

// Runner handles database migrations
type Runner struct {
	db         *sql.DB
	migrations []Migration
}

// NewRunner creates a new migration runner
func NewRunner(db *sql.DB) *Runner {
	r := &Runner{
		db:         db,
		migrations: []Migration{},
	}
	r.registerAll()
	return r
}

// registerAll registers all available migrations in order
func (r *Runner) registerAll() {
	r.migrations = []Migration{
		migration001InitialSchema(),
		migration002AddVersionType(),
		migration003MultiComponentSources(),
		migration004MergeSources(),
		migration005RemoveDistributionOverrides(),
		migration006RefreshTokens(),
		migration007ComponentVersions(),
		migration008ComponentBuildTypes(),
		migration009ForgeTypeVersionFilter(),
		migration010DownloadJobSourceDedup(),
		migration011SourceDefaultVersion(),
	}

	// Sort by version to ensure correct order
	sort.Slice(r.migrations, func(i, j int) bool {
		return r.migrations[i].Version < r.migrations[j].Version
	})
}

// ensureMigrationsTable creates the migrations tracking table if it doesn't exist
func (r *Runner) ensureMigrationsTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// getAppliedVersions returns a set of already applied migration versions
func (r *Runner) getAppliedVersions() (map[int]bool, error) {
	rows, err := r.db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// Run executes all pending migrations
func (r *Runner) Run() error {
	if err := r.ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, err := r.getAppliedVersions()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	for _, m := range r.migrations {
		if applied[m.Version] {
			continue
		}

		if err := r.runMigration(m); err != nil {
			if log != nil {
				log.Error("Migration failed", "version", m.Version, "description", m.Description, "error", err)
			}
			return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Description, err)
		}
	}

	return nil
}

// runMigration executes a single migration within a transaction
func (r *Runner) runMigration(m Migration) error {
	if log != nil {
		log.Debug("Applying migration", "version", m.Version, "description", m.Description)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute the migration
	if err := m.Up(tx); err != nil {
		tx.Rollback()
		return err
	}

	// Record the migration
	_, err = tx.Exec(
		"INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)",
		m.Version, m.Description, time.Now().UTC(),
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if log != nil {
		log.Debug("Migration applied successfully", "version", m.Version)
	}

	return nil
}

// CurrentVersion returns the highest applied migration version
func (r *Runner) CurrentVersion() (int, error) {
	if err := r.ensureMigrationsTable(); err != nil {
		return 0, err
	}

	var version int
	err := r.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	return version, err
}

// PendingCount returns the number of pending migrations
func (r *Runner) PendingCount() (int, error) {
	if err := r.ensureMigrationsTable(); err != nil {
		return 0, err
	}

	applied, err := r.getAppliedVersions()
	if err != nil {
		return 0, err
	}

	pending := 0
	for _, m := range r.migrations {
		if !applied[m.Version] {
			pending++
		}
	}

	return pending, nil
}
