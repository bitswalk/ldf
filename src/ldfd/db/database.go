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

	// Open in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open in-memory database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	database := &Database{
		db:          db,
		persistPath: persistPath,
	}

	// Initialize schema
	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
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

// initSchema creates the database tables
func (d *Database) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS distributions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		version TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		visibility TEXT NOT NULL DEFAULT 'private',
		config TEXT,
		source_url TEXT,
		checksum TEXT,
		size_bytes INTEGER DEFAULT 0,
		owner_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		started_at DATETIME,
		completed_at DATETIME,
		error_message TEXT,
		FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_distributions_status ON distributions(status);
	CREATE INDEX IF NOT EXISTS idx_distributions_name ON distributions(name);
	CREATE INDEX IF NOT EXISTS idx_distributions_owner ON distributions(owner_id);
	CREATE INDEX IF NOT EXISTS idx_distributions_visibility ON distributions(visibility);

	CREATE TABLE IF NOT EXISTS distribution_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		distribution_id INTEGER NOT NULL,
		level TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (distribution_id) REFERENCES distributions(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_distribution_logs_dist_id ON distribution_logs(distribution_id);

	CREATE TABLE IF NOT EXISTS roles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		description TEXT,
		can_read BOOLEAN NOT NULL DEFAULT 1,
		can_write BOOLEAN NOT NULL DEFAULT 0,
		can_delete BOOLEAN NOT NULL DEFAULT 0,
		can_admin BOOLEAN NOT NULL DEFAULT 0,
		is_system BOOLEAN NOT NULL DEFAULT 0,
		parent_role_id TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (parent_role_id) REFERENCES roles(id) ON DELETE SET NULL
	);

	CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
	CREATE INDEX IF NOT EXISTS idx_roles_parent ON roles(parent_role_id);

	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role_id TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
	);

	CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role_id);

	CREATE TABLE IF NOT EXISTS revoked_tokens (
		token_id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		revoked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_revoked_tokens_user_id ON revoked_tokens(user_id);
	CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := d.db.Exec(schema)
	if err != nil {
		return err
	}

	// Seed default roles if they don't exist
	return d.seedDefaultRoles()
}

// seedDefaultRoles creates the three default system roles if they don't exist
func (d *Database) seedDefaultRoles() error {
	// Fixed UUIDs for system roles - must match constants in auth/role.go
	defaultRoles := []struct {
		id          string
		name        string
		description string
		canRead     bool
		canWrite    bool
		canDelete   bool
		canAdmin    bool
	}{
		{
			id:          "908b291e-61fb-4d95-98db-0b76c0afd6b4", // RoleIDRoot
			name:        "root",
			description: "Administrator role with full system access",
			canRead:     true,
			canWrite:    true,
			canDelete:   true,
			canAdmin:    true,
		},
		{
			id:          "91db9f27-b8a2-4452-9b80-5f6ab1096da8", // RoleIDDeveloper
			name:        "developer",
			description: "Standard user with read/write access to owned resources",
			canRead:     true,
			canWrite:    true,
			canDelete:   true,
			canAdmin:    false,
		},
		{
			id:          "e8fcda13-fea4-4a1f-9e60-e4c9b882e0d0", // RoleIDAnonymous
			name:        "anonymous",
			description: "Read-only access to public resources",
			canRead:     true,
			canWrite:    false,
			canDelete:   false,
			canAdmin:    false,
		},
	}

	for _, role := range defaultRoles {
		_, err := d.db.Exec(`
			INSERT OR IGNORE INTO roles (id, name, description, can_read, can_write, can_delete, can_admin, is_system)
			VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		`, role.id, role.name, role.description, role.canRead, role.canWrite, role.canDelete, role.canAdmin)
		if err != nil {
			return fmt.Errorf("failed to seed role %s: %w", role.name, err)
		}
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

	// Open the disk database
	diskDB, err := sql.Open("sqlite3", d.persistPath)
	if err != nil {
		return fmt.Errorf("failed to open disk database: %w", err)
	}
	defer diskDB.Close()

	// Copy data from disk to memory using attach
	attachQuery := fmt.Sprintf("ATTACH DATABASE '%s' AS disk_db", d.persistPath)
	if _, err := d.db.Exec(attachQuery); err != nil {
		return fmt.Errorf("failed to attach disk database: %w", err)
	}
	defer d.db.Exec("DETACH DATABASE disk_db")

	// Copy settings table first (no dependencies)
	if d.tableExistsInDiskDB("settings") {
		if _, err := d.db.Exec(`
			INSERT OR REPLACE INTO settings
			SELECT * FROM disk_db.settings
		`); err != nil {
			// Ignore error - table structure may have changed
		}
	}

	// Copy custom roles (non-system roles) - before users (users reference roles)
	if d.tableExistsInDiskDB("roles") {
		if _, err := d.db.Exec(`
			INSERT OR REPLACE INTO roles
			SELECT * FROM disk_db.roles WHERE is_system = 0
		`); err != nil {
			// Ignore error - table structure may have changed
		}
	}

	// Copy users table - before distributions (distributions reference users)
	if d.tableExistsInDiskDB("users") {
		if _, err := d.db.Exec(`
			INSERT OR REPLACE INTO users
			SELECT * FROM disk_db.users
		`); err != nil {
			// Ignore error - table structure may have changed
		}
	}

	// Copy distributions table (handle schema migration for visibility column)
	if d.tableExistsInDiskDB("distributions") {
		if _, err := d.db.Exec(`
			INSERT OR REPLACE INTO distributions
			(id, name, version, status, visibility, config, source_url, checksum, size_bytes, owner_id, created_at, updated_at, started_at, completed_at, error_message)
			SELECT id, name, version, status,
			       COALESCE(visibility, 'private') as visibility,
			       config, source_url, checksum, size_bytes, owner_id, created_at, updated_at, started_at, completed_at, error_message
			FROM disk_db.distributions
		`); err != nil {
			// Ignore error - table structure may have changed
		}
	}

	// Copy distribution_logs table
	if d.tableExistsInDiskDB("distribution_logs") {
		if _, err := d.db.Exec(`
			INSERT OR REPLACE INTO distribution_logs
			SELECT * FROM disk_db.distribution_logs
		`); err != nil {
			// Ignore error - table structure may have changed
		}
	}

	// Copy revoked_tokens table (references users)
	if d.tableExistsInDiskDB("revoked_tokens") {
		if _, err := d.db.Exec(`
			INSERT OR REPLACE INTO revoked_tokens
			SELECT * FROM disk_db.revoked_tokens
		`); err != nil {
			// Ignore error - table structure may have changed
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
