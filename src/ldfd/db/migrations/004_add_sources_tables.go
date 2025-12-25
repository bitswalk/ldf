package migrations

import (
	"database/sql"
	"fmt"
)

// migration004AddSourcesTables creates the source_defaults and user_sources tables
func migration004AddSourcesTables() Migration {
	return Migration{
		Version:     4,
		Description: "Add source_defaults and user_sources tables for upstream source management",
		Up:          migration004Up,
	}
}

func migration004Up(tx *sql.Tx) error {
	// Create source_defaults table for system-wide default sources (admin-managed)
	if _, err := tx.Exec(`
		CREATE TABLE source_defaults (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL UNIQUE,
			priority INTEGER NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create source_defaults table: %w", err)
	}

	// Create index for priority sorting
	if _, err := tx.Exec(`CREATE INDEX idx_source_defaults_priority ON source_defaults(priority)`); err != nil {
		return fmt.Errorf("failed to create source_defaults priority index: %w", err)
	}

	// Create user_sources table for user-specific sources
	if _, err := tx.Exec(`
		CREATE TABLE user_sources (
			id TEXT PRIMARY KEY,
			owner_id TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(owner_id, url)
		)
	`); err != nil {
		return fmt.Errorf("failed to create user_sources table: %w", err)
	}

	// Create indexes for user_sources
	if _, err := tx.Exec(`CREATE INDEX idx_user_sources_owner ON user_sources(owner_id)`); err != nil {
		return fmt.Errorf("failed to create user_sources owner index: %w", err)
	}
	if _, err := tx.Exec(`CREATE INDEX idx_user_sources_priority ON user_sources(priority)`); err != nil {
		return fmt.Errorf("failed to create user_sources priority index: %w", err)
	}

	return nil
}
