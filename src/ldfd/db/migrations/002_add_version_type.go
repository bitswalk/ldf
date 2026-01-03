package migrations

import (
	"database/sql"
	"fmt"
)

// migration002AddVersionType adds the version_type column to source_versions table
func migration002AddVersionType() Migration {
	return Migration{
		Version:     2,
		Description: "Add version_type column to source_versions for kernel version classification",
		Up:          migration002Up,
	}
}

func migration002Up(tx *sql.Tx) error {
	// Add version_type column with default value 'stable'
	_, err := tx.Exec(`
		ALTER TABLE source_versions ADD COLUMN version_type TEXT NOT NULL DEFAULT 'stable'
	`)
	if err != nil {
		return fmt.Errorf("failed to add version_type column: %w", err)
	}

	// Update existing records: set version_type based on is_stable and version patterns
	// - RC versions (contains '-rc') -> 'mainline'
	// - linux-next versions -> 'linux-next'
	// - LTS version patterns (6.12.x, 6.6.x, 6.1.x, 5.15.x, 5.10.x, 5.4.x, 4.19.x, 4.14.x) -> 'longterm'
	// - Others with is_stable = 1 -> 'stable'

	// First, mark mainline (RC) versions
	_, err = tx.Exec(`
		UPDATE source_versions
		SET version_type = 'mainline'
		WHERE version LIKE '%-rc%'
	`)
	if err != nil {
		return fmt.Errorf("failed to update mainline versions: %w", err)
	}

	// Mark linux-next versions
	_, err = tx.Exec(`
		UPDATE source_versions
		SET version_type = 'linux-next'
		WHERE version LIKE 'next-%'
	`)
	if err != nil {
		return fmt.Errorf("failed to update linux-next versions: %w", err)
	}

	// Mark known LTS kernel versions
	// LTS versions as of 2024: 6.12, 6.6, 6.1, 5.15, 5.10, 5.4, 4.19, 4.14
	_, err = tx.Exec(`
		UPDATE source_versions
		SET version_type = 'longterm'
		WHERE version_type = 'stable' AND (
			version LIKE '6.12.%' OR version = '6.12' OR
			version LIKE '6.6.%' OR version = '6.6' OR
			version LIKE '6.1.%' OR version = '6.1' OR
			version LIKE '5.15.%' OR version = '5.15' OR
			version LIKE '5.10.%' OR version = '5.10' OR
			version LIKE '5.4.%' OR version = '5.4' OR
			version LIKE '4.19.%' OR version = '4.19' OR
			version LIKE '4.14.%' OR version = '4.14'
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to update longterm versions: %w", err)
	}

	// Create index for version_type filtering
	_, err = tx.Exec(`
		CREATE INDEX idx_source_versions_type ON source_versions(source_id, source_type, version_type)
	`)
	if err != nil {
		return fmt.Errorf("failed to create version_type index: %w", err)
	}

	return nil
}
