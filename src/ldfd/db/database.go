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

// loadSchemaMigrations loads only the schema_migrations table from disk
// This must be called BEFORE running migrations to prevent re-seeding
func (d *Database) loadSchemaMigrations() error {
	if d.persistPath == "" {
		return nil
	}

	// Open the disk database
	diskDB, err := sql.Open("sqlite3", d.persistPath)
	if err != nil {
		return fmt.Errorf("failed to open disk database: %w", err)
	}
	defer diskDB.Close()

	// Check if schema_migrations table exists in disk DB
	var tableName string
	err = diskDB.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='schema_migrations'
	`).Scan(&tableName)
	if err != nil {
		// Table doesn't exist - this is a fresh database
		return nil
	}

	// Create schema_migrations table in memory if it doesn't exist
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Copy schema_migrations from disk to memory
	attachQuery := fmt.Sprintf("ATTACH DATABASE '%s' AS disk_db", d.persistPath)
	if _, err := d.db.Exec(attachQuery); err != nil {
		return fmt.Errorf("failed to attach disk database: %w", err)
	}
	defer d.db.Exec("DETACH DATABASE disk_db")

	_, err = d.db.Exec(`
		INSERT OR REPLACE INTO schema_migrations
		SELECT * FROM disk_db.schema_migrations
	`)
	if err != nil {
		return fmt.Errorf("failed to copy schema_migrations: %w", err)
	}

	return nil
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

	// Copy components table FIRST (source_defaults and user_sources have FK references to components)
	// Must be loaded before sources to satisfy foreign key constraints
	// We DELETE then INSERT to ensure deleted components stay deleted (not re-seeded)
	if d.tableExistsInDiskDB("components") {
		// Check if the disk schema is compatible by looking for the 'is_optional' column
		// (this is a key column in the current components schema)
		var hasIsOptionalCol int
		d.db.QueryRow(`
			SELECT COUNT(*) FROM disk_db.pragma_table_info('components') WHERE name = 'is_optional'
		`).Scan(&hasIsOptionalCol)

		if hasIsOptionalCol > 0 {
			// Compatible schema - delete seeded components and load from disk
			// ON DELETE CASCADE handles cleanup of related tables (download_jobs),
			// and ON DELETE SET NULL handles upstream_sources
			if _, err := d.db.Exec(`DELETE FROM components`); err != nil {
				loadErrors = append(loadErrors, fmt.Sprintf("components delete: %v", err))
			}
			// Insert from disk (handle schema migration for is_system and owner_id)
			// Schema: id, name, category, display_name, description, artifact_pattern,
			//         default_url_template, github_normalized_template, is_optional,
			//         created_at, updated_at, is_system, owner_id
			result, err := d.db.Exec(`
				INSERT INTO components
				(id, name, category, display_name, description, artifact_pattern,
				 default_url_template, github_normalized_template, is_optional,
				 created_at, updated_at, is_system, owner_id)
				SELECT id, name, category, display_name, description, artifact_pattern,
				       default_url_template, github_normalized_template, is_optional,
				       created_at, updated_at,
				       COALESCE(is_system, 0) as is_system,
				       NULLIF(owner_id, '') as owner_id
				FROM disk_db.components
			`)
			if err != nil {
				// Fallback: try with old schema (no is_system, owner_id columns on disk)
				result, err = d.db.Exec(`
					INSERT INTO components
					(id, name, category, display_name, description, artifact_pattern,
					 default_url_template, github_normalized_template, is_optional,
					 created_at, updated_at, is_system, owner_id)
					SELECT id, name, category, display_name, description, artifact_pattern,
					       default_url_template, github_normalized_template, is_optional,
					       created_at, updated_at,
					       0 as is_system,
					       NULL as owner_id
					FROM disk_db.components
				`)
			}
			if err != nil {
				loadErrors = append(loadErrors, fmt.Sprintf("components: %v", err))
			} else if rows, _ := result.RowsAffected(); rows > 0 {
				loadedTables = append(loadedTables, fmt.Sprintf("components(%d)", rows))
			}
		} else {
			// Incompatible schema - keep seeded components from migrations
			fmt.Fprintf(os.Stderr, "INFO: Components table schema changed, using default components\n")
		}
	}

	// Copy upstream_sources table (unified sources table - new schema)
	if d.tableExistsInDiskDB("upstream_sources") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO upstream_sources
			(id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
			SELECT id, name, url, COALESCE(component_ids, '[]'), retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at
			FROM disk_db.upstream_sources
		`)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("upstream_sources: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("upstream_sources(%d)", rows))
		}
	} else {
		// Fallback: migrate from old source_defaults and user_sources tables if they exist
		// This handles loading databases created before migration 004

		// Load from source_defaults (system sources)
		if d.tableExistsInDiskDB("source_defaults") {
			// Try with component_ids column first (new schema)
			result, err := d.db.Exec(`
				INSERT OR REPLACE INTO upstream_sources
				(id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
				SELECT id, name, url, COALESCE(component_ids, '[]'), retrieval_method, url_template, priority, enabled, 1, NULL, created_at, updated_at
				FROM disk_db.source_defaults
			`)
			if err != nil {
				// Fallback: try with component_id column (migrate to component_ids)
				result, err = d.db.Exec(`
					INSERT OR REPLACE INTO upstream_sources
					(id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
					SELECT id, name, url,
					       CASE WHEN component_id IS NOT NULL AND component_id != '' THEN '["' || component_id || '"]' ELSE '[]' END,
					       retrieval_method, url_template, priority, enabled, 1, NULL, created_at, updated_at
					FROM disk_db.source_defaults
				`)
			}
			if err != nil {
				// Fallback: try without component columns (old schema)
				result, err = d.db.Exec(`
					INSERT OR REPLACE INTO upstream_sources
					(id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
					SELECT id, name, url, '[]', 'release', NULL, priority, enabled, 1, NULL, created_at, updated_at
					FROM disk_db.source_defaults
				`)
			}
			if err != nil {
				loadErrors = append(loadErrors, fmt.Sprintf("source_defaults->upstream_sources: %v", err))
			} else if rows, _ := result.RowsAffected(); rows > 0 {
				loadedTables = append(loadedTables, fmt.Sprintf("source_defaults->upstream_sources(%d)", rows))
			}
		}

		// Load from user_sources (user sources)
		if d.tableExistsInDiskDB("user_sources") {
			// Try with component_ids column first (new schema)
			result, err := d.db.Exec(`
				INSERT OR REPLACE INTO upstream_sources
				(id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
				SELECT id, name, url, COALESCE(component_ids, '[]'), retrieval_method, url_template, priority, enabled, 0, owner_id, created_at, updated_at
				FROM disk_db.user_sources
			`)
			if err != nil {
				// Fallback: try with component_id column (migrate to component_ids)
				result, err = d.db.Exec(`
					INSERT OR REPLACE INTO upstream_sources
					(id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
					SELECT id, name, url,
					       CASE WHEN component_id IS NOT NULL AND component_id != '' THEN '["' || component_id || '"]' ELSE '[]' END,
					       retrieval_method, url_template, priority, enabled, 0, owner_id, created_at, updated_at
					FROM disk_db.user_sources
				`)
			}
			if err != nil {
				// Fallback: try without component columns (old schema)
				result, err = d.db.Exec(`
					INSERT OR REPLACE INTO upstream_sources
					(id, name, url, component_ids, retrieval_method, url_template, priority, enabled, is_system, owner_id, created_at, updated_at)
					SELECT id, name, url, '[]', 'release', NULL, priority, enabled, 0, owner_id, created_at, updated_at
					FROM disk_db.user_sources
				`)
			}
			if err != nil {
				loadErrors = append(loadErrors, fmt.Sprintf("user_sources->upstream_sources: %v", err))
			} else if rows, _ := result.RowsAffected(); rows > 0 {
				loadedTables = append(loadedTables, fmt.Sprintf("user_sources->upstream_sources(%d)", rows))
			}
		}
	}

	// Copy download_jobs table (handle schema migration for new columns)
	if d.tableExistsInDiskDB("download_jobs") {
		result, err := d.db.Exec(`
			INSERT OR REPLACE INTO download_jobs
			(id, distribution_id, owner_id, component_id, source_id, source_type,
			 retrieval_method, resolved_url, version, status, progress_bytes, total_bytes,
			 created_at, started_at, completed_at, artifact_path, checksum, error_message,
			 retry_count, max_retries)
			SELECT id, distribution_id,
			       COALESCE(owner_id, '') as owner_id,
			       component_id,
			       source_id, source_type,
			       COALESCE(retrieval_method, 'release') as retrieval_method,
			       resolved_url, version, status, progress_bytes, total_bytes,
			       created_at, started_at, completed_at, artifact_path, checksum, error_message,
			       retry_count, max_retries
			FROM disk_db.download_jobs
		`)
		if err != nil {
			// Fallback: try without new columns
			result, err = d.db.Exec(`
				INSERT OR REPLACE INTO download_jobs
				(id, distribution_id, owner_id, component_id, source_id, source_type,
				 retrieval_method, resolved_url, version, status, progress_bytes, total_bytes,
				 created_at, started_at, completed_at, artifact_path, checksum, error_message,
				 retry_count, max_retries)
				SELECT id, distribution_id, '', component_id, source_id, source_type,
				       'release', resolved_url, version, status, progress_bytes, total_bytes,
				       created_at, started_at, completed_at, artifact_path, checksum, error_message,
				       retry_count, max_retries
				FROM disk_db.download_jobs
			`)
		}
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("download_jobs: %v", err))
		} else if rows, _ := result.RowsAffected(); rows > 0 {
			loadedTables = append(loadedTables, fmt.Sprintf("download_jobs(%d)", rows))
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

// ResetToDefaults resets the database to its default state by:
// 1. Clearing all tables except schema_migrations
// 2. Re-running seeding migrations
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

	// Get list of all tables except sqlite internal tables and schema_migrations
	rows, err := d.db.Query(`
		SELECT name FROM sqlite_master
		WHERE type='table'
		AND name NOT LIKE 'sqlite_%'
		AND name != 'schema_migrations'
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

	// Disable foreign keys temporarily to allow deletion in any order
	if _, err := d.db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	// Delete all data from all tables
	for _, table := range tables {
		if _, err := d.db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			// Re-enable foreign keys before returning
			d.db.Exec("PRAGMA foreign_keys = ON")
			return fmt.Errorf("failed to clear table %s: %w", table, err)
		}
	}

	// Re-enable foreign keys
	if _, err := d.db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to re-enable foreign keys: %w", err)
	}

	// Clear schema_migrations to force re-running all migrations
	if _, err := d.db.Exec("DELETE FROM schema_migrations"); err != nil {
		return fmt.Errorf("failed to clear schema_migrations: %w", err)
	}

	// Re-run all migrations to seed default data
	runner := migrations.NewRunner(d.db)
	if err := runner.Run(); err != nil {
		return fmt.Errorf("failed to run migrations after reset: %w", err)
	}

	return nil
}
