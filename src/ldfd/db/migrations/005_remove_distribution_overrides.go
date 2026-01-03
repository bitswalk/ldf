package migrations

import (
	"database/sql"
	"fmt"
)

// migration005RemoveDistributionOverrides removes the distribution_source_overrides table
// This table is no longer needed since sources are now unified in upstream_sources
// and source selection is handled by priority-based resolution
func migration005RemoveDistributionOverrides() Migration {
	return Migration{
		Version:     5,
		Description: "Remove distribution_source_overrides table (no longer needed)",
		Up:          migration005Up,
	}
}

func migration005Up(tx *sql.Tx) error {
	// Drop the distribution_source_overrides table
	// The table had:
	// - distribution_id -> component_id -> source_id mapping
	// - This was used to override which source a distribution uses for a component
	// - With unified upstream_sources, we rely on priority-based source selection instead
	_, err := tx.Exec(`DROP TABLE IF EXISTS distribution_source_overrides`)
	if err != nil {
		return fmt.Errorf("failed to drop distribution_source_overrides table: %w", err)
	}

	return nil
}
