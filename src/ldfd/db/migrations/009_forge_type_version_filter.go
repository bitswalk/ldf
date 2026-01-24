package migrations

import (
	"database/sql"
)

func migration009ForgeTypeVersionFilter() Migration {
	return Migration{
		Version:     9,
		Description: "Add forge_type and version_filter columns to upstream sources",
		Up: func(tx *sql.Tx) error {
			// Add forge_type column to upstream_sources
			_, err := tx.Exec(`ALTER TABLE upstream_sources ADD COLUMN forge_type TEXT NOT NULL DEFAULT 'generic'`)
			if err != nil {
				return err
			}

			// Add version_filter column to upstream_sources
			_, err = tx.Exec(`ALTER TABLE upstream_sources ADD COLUMN version_filter TEXT NOT NULL DEFAULT ''`)
			if err != nil {
				return err
			}

			// Update existing GitHub sources to have the correct forge type
			_, err = tx.Exec(`UPDATE upstream_sources SET forge_type = 'github' WHERE url LIKE '%github.com%'`)
			if err != nil {
				return err
			}

			// Update existing GitLab sources
			_, err = tx.Exec(`UPDATE upstream_sources SET forge_type = 'gitlab' WHERE url LIKE '%gitlab.com%' OR url LIKE '%gitlab.%'`)
			if err != nil {
				return err
			}

			// Update existing Codeberg sources
			_, err = tx.Exec(`UPDATE upstream_sources SET forge_type = 'codeberg' WHERE url LIKE '%codeberg.org%'`)
			if err != nil {
				return err
			}

			// Create index for efficient filtering
			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_upstream_sources_forge_type ON upstream_sources(forge_type)`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
