package migrations

import (
	"database/sql"
)

func migration006RefreshTokens() Migration {
	return Migration{
		Version:     6,
		Description: "Add refresh tokens table for JWT token renewal",
		Up: func(tx *sql.Tx) error {
			// Create refresh_tokens table
			_, err := tx.Exec(`
				CREATE TABLE refresh_tokens (
					id TEXT PRIMARY KEY,
					user_id TEXT NOT NULL,
					token_hash TEXT NOT NULL UNIQUE,
					expires_at DATETIME NOT NULL,
					created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
					last_used_at DATETIME,
					revoked BOOLEAN DEFAULT FALSE,
					FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
				)
			`)
			if err != nil {
				return err
			}

			// Create indexes for efficient lookups
			_, err = tx.Exec(`CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id)`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash)`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at)`)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
