package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/google/uuid"
)

// generateRefreshToken generates a cryptographically secure random token
func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of a token for secure storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// CreateRefreshToken creates a new refresh token for a user
// Returns the plain token (to send to client) and the stored record
func (m *UserManager) CreateRefreshToken(userID string) (string, *RefreshToken, error) {
	plainToken, err := generateRefreshToken()
	if err != nil {
		return "", nil, errors.ErrInternal.WithCause(err)
	}

	tokenHash := hashToken(plainToken)
	now := time.Now().UTC()

	refreshToken := &RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(RefreshTokenDuration),
		CreatedAt: now,
		Revoked:   false,
	}

	_, err = m.db.Exec(`
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, revoked)
		VALUES (?, ?, ?, ?, ?, ?)
	`, refreshToken.ID, refreshToken.UserID, refreshToken.TokenHash,
		refreshToken.ExpiresAt, refreshToken.CreatedAt, refreshToken.Revoked)

	if err != nil {
		return "", nil, errors.ErrDatabaseQuery.WithCause(err)
	}

	return plainToken, refreshToken, nil
}

// ValidateRefreshToken validates a refresh token and returns the associated record
func (m *UserManager) ValidateRefreshToken(plainToken string) (*RefreshToken, error) {
	tokenHash := hashToken(plainToken)

	var rt RefreshToken
	var lastUsedAt *time.Time

	err := m.db.QueryRow(`
		SELECT id, user_id, token_hash, expires_at, created_at, last_used_at, revoked
		FROM refresh_tokens
		WHERE token_hash = ?
	`, tokenHash).Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt,
		&rt.CreatedAt, &lastUsedAt, &rt.Revoked)

	if err != nil {
		return nil, errors.ErrRefreshTokenInvalid
	}

	if lastUsedAt != nil {
		rt.LastUsedAt = *lastUsedAt
	}

	// Check if token is revoked
	if rt.Revoked {
		return nil, errors.ErrRefreshTokenRevoked
	}

	// Check if token is expired
	if time.Now().UTC().After(rt.ExpiresAt) {
		return nil, errors.ErrRefreshTokenExpired
	}

	return &rt, nil
}

// UpdateRefreshTokenLastUsed updates the last_used_at timestamp
func (m *UserManager) UpdateRefreshTokenLastUsed(tokenID string) error {
	_, err := m.db.Exec(`
		UPDATE refresh_tokens
		SET last_used_at = ?
		WHERE id = ?
	`, time.Now().UTC(), tokenID)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	return nil
}

// RevokeRefreshToken revokes a specific refresh token
func (m *UserManager) RevokeRefreshToken(tokenID string) error {
	_, err := m.db.Exec(`
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE id = ?
	`, tokenID)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	return nil
}

// RevokeRefreshTokenByHash revokes a refresh token by its plain token value
func (m *UserManager) RevokeRefreshTokenByHash(plainToken string) error {
	tokenHash := hashToken(plainToken)

	result, err := m.db.Exec(`
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE token_hash = ?
	`, tokenHash)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	if rows == 0 {
		return errors.ErrRefreshTokenInvalid
	}

	return nil
}

// RevokeAllUserRefreshTokens revokes all refresh tokens for a user
func (m *UserManager) RevokeAllUserRefreshTokens(userID string) error {
	_, err := m.db.Exec(`
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE user_id = ?
	`, userID)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	return nil
}

// GetUserRefreshTokenCount returns the count of active refresh tokens for a user
func (m *UserManager) GetUserRefreshTokenCount(userID string) (int, error) {
	var count int
	err := m.db.QueryRow(`
		SELECT COUNT(*)
		FROM refresh_tokens
		WHERE user_id = ? AND revoked = FALSE AND expires_at > ?
	`, userID, time.Now().UTC()).Scan(&count)

	if err != nil {
		return 0, errors.ErrDatabaseQuery.WithCause(err)
	}

	return count, nil
}

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

// CleanupExcessTokens removes the oldest active refresh tokens for a user,
// keeping only the maxTokens most recent ones. This prevents token accumulation
// from repeated refresh operations.
func (m *UserManager) CleanupExcessTokens(userID string, maxTokens int) error {
	// Delete all but the N most recent active (non-revoked, non-expired) refresh tokens
	_, err := m.db.Exec(`
		DELETE FROM refresh_tokens
		WHERE user_id = ? AND revoked = FALSE AND expires_at > ?
		AND id NOT IN (
			SELECT id FROM refresh_tokens
			WHERE user_id = ? AND revoked = FALSE AND expires_at > ?
			ORDER BY created_at DESC
			LIMIT ?
		)
	`, userID, time.Now().UTC(), userID, time.Now().UTC(), maxTokens)

	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	return nil
}

// CleanupExpiredTokens removes expired and revoked tokens from both tables
func (m *UserManager) CleanupExpiredTokens() error {
	now := time.Now().UTC()

	// Clean up expired/revoked refresh tokens
	_, err := m.db.Exec(`
		DELETE FROM refresh_tokens
		WHERE expires_at < ? OR revoked = TRUE
	`, now)
	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	// Clean up expired revoked tokens
	_, err = m.db.Exec("DELETE FROM revoked_tokens WHERE expires_at < ?", now)
	if err != nil {
		return errors.ErrDatabaseQuery.WithCause(err)
	}

	return nil
}
