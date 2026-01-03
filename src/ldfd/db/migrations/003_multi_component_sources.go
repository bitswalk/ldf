package migrations

import (
	"database/sql"
	"fmt"
)

// migration003MultiComponentSources migrates source tables from single component_id to component_ids JSON array
func migration003MultiComponentSources() Migration {
	return Migration{
		Version:     3,
		Description: "Migrate source tables from component_id FK to component_ids JSON array",
		Up:          migration003Up,
	}
}

func migration003Up(tx *sql.Tx) error {
	// Step 1: Add component_ids column to source_defaults
	_, err := tx.Exec(`
		ALTER TABLE source_defaults ADD COLUMN component_ids TEXT DEFAULT '[]'
	`)
	if err != nil {
		return fmt.Errorf("failed to add component_ids to source_defaults: %w", err)
	}

	// Step 2: Migrate existing component_id data to component_ids JSON array
	_, err = tx.Exec(`
		UPDATE source_defaults
		SET component_ids = CASE
			WHEN component_id IS NOT NULL AND component_id != '' THEN '["' || component_id || '"]'
			ELSE '[]'
		END
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate source_defaults component_id to component_ids: %w", err)
	}

	// Step 3: Add component_ids column to user_sources
	_, err = tx.Exec(`
		ALTER TABLE user_sources ADD COLUMN component_ids TEXT DEFAULT '[]'
	`)
	if err != nil {
		return fmt.Errorf("failed to add component_ids to user_sources: %w", err)
	}

	// Step 4: Migrate existing component_id data to component_ids JSON array
	_, err = tx.Exec(`
		UPDATE user_sources
		SET component_ids = CASE
			WHEN component_id IS NOT NULL AND component_id != '' THEN '["' || component_id || '"]'
			ELSE '[]'
		END
	`)
	if err != nil {
		return fmt.Errorf("failed to migrate user_sources component_id to component_ids: %w", err)
	}

	// Note: We keep the old component_id column for backwards compatibility during transition
	// SQLite doesn't support DROP COLUMN in older versions, and it's safer to keep it
	// The repository code will use component_ids going forward

	// Step 5: Create index for component_ids lookups (JSON contains query optimization)
	// SQLite doesn't have native JSON indexing, but we can create a virtual table or use LIKE queries
	// For now, queries will use json_each() which is reasonably fast for small arrays

	return nil
}
