package migrations

import (
	"database/sql"
)

func migration007ComponentVersions() Migration {
	return Migration{
		Version:     7,
		Description: "Add default version fields to components table",
		Up: func(tx *sql.Tx) error {
			// Add default_version column - stores the pinned version or resolved version
			_, err := tx.Exec(`ALTER TABLE components ADD COLUMN default_version TEXT`)
			if err != nil {
				return err
			}

			// Add default_version_rule column - "pinned", "latest-stable", "latest-lts"
			_, err = tx.Exec(`ALTER TABLE components ADD COLUMN default_version_rule TEXT DEFAULT 'latest-stable'`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
