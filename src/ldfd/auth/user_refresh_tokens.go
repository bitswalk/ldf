package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/google/uuid"
)

// RefreshToken represents a refresh token stored in the database
type RefreshToken struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	TokenHash  string    `json:"-"` // Never expose the hash
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
	Revoked    bool      `json:"revoked"`
}

// RefreshTokenDuration is the lifetime of a refresh token (7 days)
const RefreshTokenDuration = 7 * 24 * time.Hour

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

// CleanupExpiredRefreshTokens removes expired refresh tokens
func (m *UserManager) CleanupExpiredRefreshTokens() error {
	_, err := m.db.Exec(`
		DELETE FROM refresh_tokens
		WHERE expires_at < ? OR revoked = TRUE
	`, time.Now().UTC())

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
