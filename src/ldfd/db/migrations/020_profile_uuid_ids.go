package migrations

import (
	"database/sql"

	"github.com/google/uuid"
)

func migration020ProfileUUIDIDs() Migration {
	return Migration{
		Version:     20,
		Description: "Replace board/toolchain profile seed IDs with UUIDs",
		Up: func(tx *sql.Tx) error {
			// Board profiles: bp-* -> UUID
			rows, err := tx.Query(`SELECT id FROM board_profiles WHERE id LIKE 'bp-%'`)
			if err != nil {
				return err
			}
			defer rows.Close()

			var boardIDs []string
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					return err
				}
				boardIDs = append(boardIDs, id)
			}
			if err := rows.Err(); err != nil {
				return err
			}

			for _, oldID := range boardIDs {
				newID := uuid.New().String()
				if _, err := tx.Exec(`UPDATE board_profiles SET id = ? WHERE id = ?`, newID, oldID); err != nil {
					return err
				}
			}

			// Toolchain profiles: tp-* -> UUID
			rows2, err := tx.Query(`SELECT id FROM toolchain_profiles WHERE id LIKE 'tp-%'`)
			if err != nil {
				return err
			}
			defer rows2.Close()

			var toolchainIDs []string
			for rows2.Next() {
				var id string
				if err := rows2.Scan(&id); err != nil {
					return err
				}
				toolchainIDs = append(toolchainIDs, id)
			}
			if err := rows2.Err(); err != nil {
				return err
			}

			for _, oldID := range toolchainIDs {
				newID := uuid.New().String()
				if _, err := tx.Exec(`UPDATE toolchain_profiles SET id = ? WHERE id = ?`, newID, oldID); err != nil {
					return err
				}
			}

			return nil
		},
	}
}
