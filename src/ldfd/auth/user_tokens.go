package auth

import (
	"time"

	"github.com/bitswalk/ldf/src/common/errors"
)

// RevokeToken adds a token to the revoked tokens list
func (m *UserManager) RevokeToken(tokenID, userID string, expiresAt time.Time) error {
	_, err := m.db.Exec(`
		INSERT OR REPLACE INTO revoked_tokens (token_id, user_id, revoked_at, expires_at)
		VALUES (?, ?, ?, ?)
	`, tokenID, userID, time.Now().UTC(), expiresAt)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	return nil
}

// IsTokenRevoked checks if a token has been revoked
func (m *UserManager) IsTokenRevoked(tokenID string) (bool, error) {
	var count int
	if err := m.db.QueryRow("SELECT COUNT(*) FROM revoked_tokens WHERE token_id = ?", tokenID).Scan(&count); err != nil {
		return false, errors.ErrDatabaseQuery.WithCause(err)
	}
	return count > 0, nil
}

// CleanupExpiredTokens removes revoked tokens that have passed their expiry time
func (m *UserManager) CleanupExpiredTokens() error {
	_, err := m.db.Exec("DELETE FROM revoked_tokens WHERE expires_at < ?", time.Now().UTC())
	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}
	return nil
}
