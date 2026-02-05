package migrations

import (
	"database/sql"
)

func migration013BuildClearCache() Migration {
	return Migration{
		Version:     13,
		Description: "Add clear_cache column to build_jobs table",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`ALTER TABLE build_jobs ADD COLUMN clear_cache INTEGER DEFAULT 0`)
			return err
		},
	}
}
