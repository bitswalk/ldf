package migrations

import (
	"database/sql"
)

func migration011SourceDefaultVersion() Migration {
	return Migration{
		Version:     11,
		Description: "Add default_version column to upstream sources",
		Up: func(tx *sql.Tx) error {
			// Add default_version column to upstream_sources
			_, err := tx.Exec(`ALTER TABLE upstream_sources ADD COLUMN default_version TEXT NOT NULL DEFAULT ''`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
