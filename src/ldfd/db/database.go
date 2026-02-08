// Package db provides database functionality for ldfd with in-memory SQLite
// and automatic persistence to disk on shutdown or crash.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/common/paths"
	"github.com/bitswalk/ldf/src/ldfd/db/migrations"
	_ "github.com/mattn/go-sqlite3"
)

var log = logs.NewDefault()

const (
	SQLiteBusyTimeout  = 5000
	SQLiteMaxOpenConns = 1
	SQLiteMaxIdleConns = 1
)

// Database wraps the SQLite connection with persistence capabilities
type Database struct {
	db           *sql.DB
	persistPath  string
	mu           sync.RWMutex
	shutdownOnce sync.Once
}

// Config holds the database configuration
type Config struct {
	// PersistPath is the file path where the database will be saved on shutdown
	PersistPath string
	// LoadOnStart determines whether to load existing data from disk on startup
	LoadOnStart bool
}

// DefaultConfig returns a default database configuration
func DefaultConfig() Config {
	return Config{
		PersistPath: "~/.ldfd/ldfd.db",
		LoadOnStart: true,
	}
}

// New creates a new in-memory database with persistence support
func New(cfg Config) (*Database, error) {
	// Expand ~ and env vars in persist path
	persistPath := paths.Expand(cfg.PersistPath)

	// Open in-memory database with shared cache mode.
	// Shared cache is required so the single in-memory database survives
	// connection recycling by Go's database/sql pool.
	//
	// DSN parameters applied to the connection:
	//   _foreign_keys=1    - enforce FK constraints
	//   _busy_timeout=5000 - wait up to 5s for locks before SQLITE_BUSY
	//
	// IMPORTANT: We use exactly ONE connection (MaxOpenConns=1) to avoid
	// SQLite shared-cache table-level lock contention. With multiple
	// connections, concurrent readers (SSE polling) and writers (build
	// worker) deadlock on table locks, causing "database table is locked"
	// errors that silently drop stage status updates.
	dsn := fmt.Sprintf("file::memory:?cache=shared&_busy_timeout=%d&_foreign_keys=1", SQLiteBusyTimeout)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}

	// Single connection eliminates shared-cache table-level lock contention.
	// All DB operations are serialized, which is fine for SQLite performance.
	db.SetMaxOpenConns(SQLiteMaxOpenConns)
	db.SetMaxIdleConns(SQLiteMaxIdleConns)
	db.SetConnMaxLifetime(0) // Connection doesn't expire

	database := &Database{
		db:          db,
		persistPath: persistPath,
	}

	// Run ALL migrations to initialize schema in-memory
	// This creates all tables and seeds default data
	runner := migrations.NewRunner(db)
	if err := runner.Run(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Load existing data from disk if configured and file exists
	// This will REPLACE the seeded data with actual user data from disk
	// The LoadFromDisk method handles this by deleting seeded data first
	if cfg.LoadOnStart && persistPath != "" {
		if _, err := os.Stat(persistPath); err == nil {
			if err := database.LoadFromDisk(); err != nil {
				// Log warning but don't fail - start fresh
				fmt.Fprintf(os.Stderr, "warning: failed to load database from disk: %v\n", err)
			}
		}
	}

	// Note: Signal handling for graceful shutdown is managed by the server (core/server.go)
	// to avoid race conditions with multiple signal handlers

	return database, nil
}

// DB returns the underlying sql.DB for direct queries
func (d *Database) DB() *sql.DB {
	return d.db
}

// WithTransaction executes fn within a database transaction.
// The transaction is committed if fn returns nil, rolled back otherwise.
func (d *Database) WithTransaction(fn func(*sql.Tx) error) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// Shutdown persists the database to disk and closes the connection
func (d *Database) Shutdown() error {
	var shutdownErr error

	d.shutdownOnce.Do(func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		if d.persistPath != "" {
			if err := d.persistToDisk(); err != nil {
				shutdownErr = fmt.Errorf("failed to persist database: %w", err)
			}
		}

		if err := d.db.Close(); err != nil {
			if shutdownErr != nil {
				shutdownErr = fmt.Errorf("%v; also failed to close database: %w", shutdownErr, err)
			} else {
				shutdownErr = fmt.Errorf("failed to close database: %w", err)
			}
		}
	})

	return shutdownErr
}

// GetSetting retrieves a setting value by key
func (d *Database) GetSetting(key string) (string, error) {
	var value string
	err := d.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetSetting stores or updates a setting value
func (d *Database) SetSetting(key, value string) error {
	_, err := d.db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`, key, value)
	return err
}

// GetAllSettings retrieves all settings as a map
func (d *Database) GetAllSettings() (map[string]string, error) {
	rows, err := d.db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return settings, nil
}

// ResetToDefaults resets the database to its default state by:
// 1. Dropping all tables (including schema_migrations)
// 2. Re-running all migrations from scratch
// This is a destructive operation that should only be performed by root users.
func (d *Database) ResetToDefaults() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Delete the disk database file if it exists
	if d.persistPath != "" {
		if _, err := os.Stat(d.persistPath); err == nil {
			if err := os.Remove(d.persistPath); err != nil {
				return fmt.Errorf("failed to delete disk database: %w", err)
			}
		}
	}

	// Get list of all tables except sqlite internal tables
	rows, err := d.db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table'
		AND name NOT LIKE 'sqlite_%'
	`)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	rows.Close()

	// Disable foreign keys temporarily to allow dropping in any order
	if _, err := d.db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	// Drop all tables (not just delete data) so migrations can recreate them
	for _, table := range tables {
		if _, err := d.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			// Re-enable foreign keys before returning
			if _, fkErr := d.db.Exec("PRAGMA foreign_keys = ON"); fkErr != nil {
				log.Warn("Failed to re-enable foreign keys", "error", fkErr)
			}
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Re-enable foreign keys
	if _, err := d.db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to re-enable foreign keys: %w", err)
	}

	// Re-run all migrations from scratch to recreate tables and seed default data
	runner := migrations.NewRunner(d.db)
	if err := runner.Run(); err != nil {
		return fmt.Errorf("failed to run migrations after reset: %w", err)
	}

	return nil
}
