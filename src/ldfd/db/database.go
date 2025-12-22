// Package db provides database functionality for ldfd with in-memory SQLite
// and automatic persistence to disk on shutdown or crash.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bitswalk/ldf/src/common/paths"
	"github.com/bitswalk/ldf/src/ldfd/db/migrations"
	_ "github.com/mattn/go-sqlite3"
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

	// Open in-memory database with shared cache mode
	// This ensures all connections from the pool share the same in-memory database
	// Without this, each connection from sql.DB's pool would get a separate empty database!
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}

	// For in-memory SQLite with shared cache, we need to ensure at least one connection
	// stays open to prevent the database from being destroyed
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Connections don't expire

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	database := &Database{
		db:          db,
		persistPath: persistPath,
	}

	// Run migrations to initialize schema
	runner := migrations.NewRunner(db)
	if err := runner.Run(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Load existing data from disk if configured and file exists
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

// persistToDisk saves the in-memory database to the configured file path
// Uses atomic write pattern: write to temp file, then rename to target
func (d *Database) persistToDisk() error {
	if d.persistPath == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(d.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create a temporary file in the same directory for atomic rename
	tempPath := d.persistPath + ".tmp"

	// Remove any existing temp file from a previous failed attempt
	os.Remove(tempPath)

	// Use SQLite VACUUM INTO to write to temp file
	query := fmt.Sprintf("VACUUM INTO '%s'", tempPath)
	if _, err := d.db.Exec(query); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to vacuum database to disk: %w", err)
	}

	// Atomically rename temp file to target path
	// This overwrites the existing file if present
	if err := os.Rename(tempPath, d.persistPath); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename database file: %w", err)
	}

	return nil
}

// tableExistsInDiskDB checks if a table exists in the attached disk_db
func (d *Database) tableExistsInDiskDB(tableName string) bool {
	var count int
	err := d.db.QueryRow(`
		SELECT COUNT(*) FROM disk_db.sqlite_master
		WHERE type='table' AND name=?
	`, tableName).Scan(&count)
	return err == nil && count > 0
}

// LoadFromDisk loads data from the persisted database file into memory
func (d *Database) LoadFromDisk() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.persistPath == "" {
		return nil
	}

	// Open the disk database to verify it's valid
	diskDB, err := sql.Open("sqlite3", d.persistPath)
	if err != nil {
		return fmt.Errorf("failed to open disk database: %w", err)
	}
	defer diskDB.Close()

	// Verify disk database is valid
	if err := diskDB.Ping(); err != nil {
		return fmt.Errorf("disk database ping failed: %w", err)
	}

	// Copy data from disk to memory using attach
	attachQuery := fmt.Sprintf("ATTACH DATABASE '%s' AS disk_db", d.persistPath)
	if _, err := d.db.Exec(attachQuery); err != nil {
		return fmt.Errorf("failed to attach disk database: %w", err)
	}
	defer d.db.Exec("DETACH DATABASE disk_db")

	var loadedTables []string
	var loadErrors []string

	// Copy settings table first (no dependencies)
	if d.tableExistsInDiskDB("settings") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO settings
			SELECT * FROM disk_db.settings
		`)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("settings: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("settings(%d)", rows))
		}
	}

	// Copy custom roles (non-system roles) - before users (users reference roles)
	if d.tableExistsInDiskDB("roles") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO roles
			SELECT * FROM disk_db.roles WHERE is_system = 0
		`)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("roles: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("roles(%d)", rows))
		}
	}

	// Copy users table - before distributions (distributions reference users)
	if d.tableExistsInDiskDB("users") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO users
			SELECT * FROM disk_db.users
		`)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("users: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("users(%d)", rows))
		}
	}

	// Copy distributions table (handle schema migration for visibility column)
	if d.tableExistsInDiskDB("distributions") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO distributions
			(id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id, created_at, updated_at, started_at, completed_at, error_message)
			SELECT id, name, version, status,
			       COALESCE(visibility, 'private') as visibility,
			       config, source_url, checksum, size_bytes, owner_id, created_at, updated_at, started_at, completed_at, error_message
			FROM disk_db.distributions
		`)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("distributions: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("distributions(%d)", rows))
		}
	}

	// Copy distribution_logs table
	if d.tableExistsInDiskDB("distribution_logs") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO distribution_logs
			SELECT * FROM disk_db.distribution_logs
		`)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("distribution_logs: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("distribution_logs(%d)", rows))
		}
	}

	// Copy revoked_tokens table (references users)
	if d.tableExistsInDiskDB("revoked_tokens") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO revoked_tokens
			SELECT * FROM disk_db.revoked_tokens
		`)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("revoked_tokens: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("revoked_tokens(%d)", rows))
		}
	}

	// Log what was loaded
	if len(loadedTables) > 0 {
		fmt.Fprintf(os.Stderr, "INFO: Loaded from disk: %v\n", loadedTables)
	}

	// Log any errors
	if len(loadErrors) > 0 {
		for _, e := range loadErrors {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to load table: %s\n", e)
		}
	}

	return nil
}

// SaveToDisk manually triggers a save to disk (for periodic backups)
func (d *Database) SaveToDisk() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.persistToDisk()
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
