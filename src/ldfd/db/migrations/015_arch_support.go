package migrations

import (
	"database/sql"
)

func migration015ArchSupport() Migration {
	return Migration{
		Version:     15,
		Description: "Add supported_architectures column to components",
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				ALTER TABLE components ADD COLUMN supported_architectures TEXT DEFAULT ''
			`)
			return err
		},
	}
}
