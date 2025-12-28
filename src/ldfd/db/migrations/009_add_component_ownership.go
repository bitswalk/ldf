package migrations

import (
	"database/sql"
	"fmt"
)

// migration009AddComponentOwnership adds ownership fields to components table
// - is_system: true for default/seeded components, false for user-created
// - owner_id: NULL for system components, user ID for user-created components
func migration009AddComponentOwnership() Migration {
	return Migration{
		Version:     9,
		Description: "Add is_system and owner_id fields to components for ownership tracking",
		Up:          migration009Up,
	}
}

func migration009Up(tx *sql.Tx) error {
	// Add is_system column (default true for existing components since they were seeded)
	if _, err := tx.Exec(`ALTER TABLE components ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT 1`); err != nil {
		return fmt.Errorf("failed to add is_system column: %w", err)
	}

	// Add owner_id column (NULL for system components)
	if _, err := tx.Exec(`ALTER TABLE components ADD COLUMN owner_id TEXT REFERENCES users(id) ON DELETE CASCADE`); err != nil {
		return fmt.Errorf("failed to add owner_id column: %w", err)
	}

	// Create index for owner lookups
	if _, err := tx.Exec(`CREATE INDEX idx_components_owner ON components(owner_id)`); err != nil {
		return fmt.Errorf("failed to create components owner index: %w", err)
	}

	// Create index for system component lookups
	if _, err := tx.Exec(`CREATE INDEX idx_components_is_system ON components(is_system)`); err != nil {
		return fmt.Errorf("failed to create components is_system index: %w", err)
	}

	return nil
}
