package migrations

import (
	"database/sql"
	"fmt"
)

// migration007AddLanguagePacksTable creates the language_packs table
// for storing custom language packs uploaded by administrators
func migration007AddLanguagePacksTable() Migration {
	return Migration{
		Version:     7,
		Description: "Add language_packs table for custom i18n support",
		Up:          migration007Up,
	}
}

func migration007Up(tx *sql.Tx) error {
	// Create language_packs table - stores custom translation packs
	if _, err := tx.Exec(`
		CREATE TABLE language_packs (
			locale TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			author TEXT,
			dictionary TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create language_packs table: %w", err)
	}

	// Create index for listing
	if _, err := tx.Exec(`CREATE INDEX idx_language_packs_created ON language_packs(created_at DESC)`); err != nil {
		return fmt.Errorf("failed to create language_packs created index: %w", err)
	}

	return nil
}
