package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrInvalidToken is returned when the token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when the token has expired
	ErrExpiredToken = errors.New("token has expired")
)

// JWTService handles JWT token generation and validation
type JWTService struct {
	secretKey     []byte
	issuer        string
	tokenDuration time.Duration
	repo          *Repository
}

// JWTConfig holds JWT service configuration
type JWTConfig struct {
	Issuer        string
	TokenDuration time.Duration
}

// DefaultJWTConfig returns default JWT configuration
func DefaultJWTConfig() JWTConfig {
	return JWTConfig{
		Issuer:        "ldfd",
		TokenDuration: 24 * time.Hour,
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
		secretKey:     []byte(secretKey),
		issuer:        cfg.Issuer,
		tokenDuration: cfg.TokenDuration,
		repo:          repo,
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

// GenerateToken generates a new JWT token for a user
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
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if token has been revoked
	if s.repo != nil {
		revoked, err := s.repo.IsTokenRevoked(claims.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check token revocation: %w", err)
		}
		if revoked {
			return nil, ErrTokenRevoked
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

	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return ErrInvalidToken
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

	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return time.Time{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return time.Time{}, ErrInvalidToken
	}

	if claims.ExpiresAt != nil {
		return claims.ExpiresAt.Time, nil
	}

	return time.Time{}, ErrInvalidToken
}
