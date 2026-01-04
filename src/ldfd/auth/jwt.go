package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	secretKey            []byte
	issuer               string
	tokenDuration        time.Duration
	refreshTokenDuration time.Duration
	repo                 *Repository
}

// JWTConfig holds JWT service configuration
type JWTConfig struct {
	Issuer               string
	TokenDuration        time.Duration
	RefreshTokenDuration time.Duration
}

// DefaultJWTConfig returns default JWT configuration
func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		Issuer:               "ldfd",
		TokenDuration:        15 * time.Minute,   // Short-lived access tokens
		RefreshTokenDuration: 7 * 24 * time.Hour, // 7 days for refresh tokens
	}
}

// generateSecretKey generates a random 256-bit secret key
func generateSecretKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a default (not recommended for production)
		return "ldfd-default-secret-key-change-me"
	}
	return hex.EncodeToString(bytes)
}

// SettingsStore interface for getting/setting persistent settings
type SettingsStore interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
}

// NewJWTService creates a new JWT service with persistent secret key
func NewJWTService(cfg JWTConfig, repo *Repository, settings SettingsStore) *JWTService {
	// Try to get existing secret key from settings
	secretKey, err := settings.GetSetting("jwt_secret")
	if err != nil || secretKey == "" {
		// Generate new secret key and persist it
		secretKey = generateSecretKey()
		settings.SetSetting("jwt_secret", secretKey)
	}

	return &JWTService{
		secretKey:            []byte(secretKey),
		issuer:               cfg.Issuer,
		tokenDuration:        cfg.TokenDuration,
		refreshTokenDuration: cfg.RefreshTokenDuration,
		repo:                 repo,
	}
}

// jwtClaims represents the full JWT claims structure
type jwtClaims struct {
	jwt.RegisteredClaims
	UserID      string          `json:"user_id"`
	UserName    string          `json:"user_name"`
	Email       string          `json:"email"`
	RoleID      string          `json:"role_id"`
	RoleName    string          `json:"role"`
	Permissions RolePermissions `json:"permissions"`
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	ExpiresIn    int64     `json:"expires_in"` // seconds until access token expiry
}

// GenerateToken generates a new JWT access token for a user
func (s *JWTService) GenerateToken(user *User) (string, error) {
	// Fetch the role to get permissions
	role, err := s.repo.GetRoleByID(user.RoleID)
	if err != nil {
		return "", fmt.Errorf("failed to get user role: %w", err)
	}

	tokenID := uuid.New().String()
	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenDuration)

	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Issuer:    s.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:      user.ID,
		UserName:    user.Name,
		Email:       user.Email,
		RoleID:      user.RoleID,
		RoleName:    role.Name,
		Permissions: role.Permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// GenerateTokenPair generates both an access token and a refresh token for a user
func (s *JWTService) GenerateTokenPair(user *User) (*TokenPair, error) {
	// Generate access token
	accessToken, err := s.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, _, err := s.repo.CreateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenDuration)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		ExpiresIn:    int64(s.tokenDuration.Seconds()),
	}, nil
}

// RefreshAccessToken validates a refresh token and generates a new access token
func (s *JWTService) RefreshAccessToken(refreshToken string) (*TokenPair, *User, error) {
	// Validate the refresh token
	rt, err := s.repo.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, nil, err
	}

	// Update last used timestamp
	s.repo.UpdateRefreshTokenLastUsed(rt.ID)

	// Get the user
	user, err := s.repo.GetUserByID(rt.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new access token
	accessToken, err := s.GenerateToken(user)
	if err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(s.tokenDuration)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken, // Return the same refresh token
		ExpiresAt:    expiresAt,
		ExpiresIn:    int64(s.tokenDuration.Seconds()),
	}, user, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, errors.ErrTokenExpired
		}
		return nil, errors.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.ErrTokenInvalid
	}

	// Check if token has been revoked
	if s.repo != nil {
		revoked, err := s.repo.IsTokenRevoked(claims.ID)
		if err != nil {
			return nil, errors.ErrDatabaseQuery.WithCause(err)
		}
		if revoked {
			return nil, errors.ErrTokenRevoked
		}
	}

	return &TokenClaims{
		UserID:      claims.UserID,
		UserName:    claims.UserName,
		Email:       claims.Email,
		RoleID:      claims.RoleID,
		RoleName:    claims.RoleName,
		Permissions: claims.Permissions,
		TokenID:     claims.ID,
	}, nil
}

// RevokeToken revokes a JWT token
func (s *JWTService) RevokeToken(tokenString string) error {
	// Parse the token to get claims (without full validation since we're revoking)
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})

	if err != nil && err != jwt.ErrTokenExpired {
		return errors.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return errors.ErrTokenInvalid
	}

	// Add token to revoked list
	expiresAt := time.Now().UTC().Add(s.tokenDuration) // Default expiry
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return s.repo.RevokeToken(claims.ID, claims.UserID, expiresAt)
}

// GetTokenExpiry returns the token expiry time from a token string
func (s *JWTService) GetTokenExpiry(tokenString string) (time.Time, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return s.secretKey, nil
	})

	if err != nil && err != jwt.ErrTokenExpired {
		return time.Time{}, errors.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return time.Time{}, errors.ErrTokenInvalid
	}

	if claims.ExpiresAt != nil {
		return claims.ExpiresAt.Time, nil
	}

	return time.Time{}, errors.ErrTokenInvalid
}
