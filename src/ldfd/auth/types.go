// Package auth provides authentication and authorization functionality for ldfd.
package auth

import (
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Default role IDs for the three system roles (fixed UUIDs for consistency)
const (
	RoleIDRoot      = "908b291e-61fb-4d95-98db-0b76c0afd6b4"
	RoleIDDeveloper = "91db9f27-b8a2-4452-9b80-5f6ab1096da8"
	RoleIDAnonymous = "e8fcda13-fea4-4a1f-9e60-e4c9b882e0d0"
)

// RefreshTokenDuration is the lifetime of a refresh token (7 days)
const RefreshTokenDuration = 7 * 24 * time.Hour

// Role represents a user role name for backward compatibility
// New code should use RoleRecord for full role information
type Role string

const (
	// RoleRoot is the admin role with full read/write access
	RoleRoot Role = "root"
	// RoleDeveloper is the default role with read/write access to owned resources
	RoleDeveloper Role = "developer"
	// RoleAnonymous is a read-only role for public resources
	RoleAnonymous Role = "anonymous"
)

// RolePermissions defines the permission flags for a role
type RolePermissions struct {
	CanRead   bool `json:"can_read"`
	CanWrite  bool `json:"can_write"`
	CanDelete bool `json:"can_delete"`
	CanAdmin  bool `json:"can_admin"`
}

// RoleRecord represents a role stored in the database
type RoleRecord struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description,omitempty"`
	Permissions  RolePermissions `json:"permissions"`
	IsSystem     bool            `json:"is_system"`
	ParentRoleID string          `json:"parent_role_id,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// HasWriteAccess returns true if the role has write permissions
func (r *RoleRecord) HasWriteAccess() bool {
	return r.Permissions.CanWrite
}

// HasDeleteAccess returns true if the role has delete permissions
func (r *RoleRecord) HasDeleteAccess() bool {
	return r.Permissions.CanDelete
}

// HasAdminAccess returns true if the role has admin permissions
func (r *RoleRecord) HasAdminAccess() bool {
	return r.Permissions.CanAdmin
}

// User represents a user account
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	RoleID       string    `json:"role_id"`
	RoleName     string    `json:"role"` // Populated from joined role data
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserManager handles user, role, and token persistence
type UserManager struct {
	db *sql.DB
}

// RevokedToken represents a revoked JWT token
type RevokedToken struct {
	TokenID   string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	RevokedAt time.Time `json:"revoked_at"`
	ExpiresAt time.Time `json:"expires_at"` // Keep until original expiry for cleanup
}

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

// AuthRequest represents the authentication request body structure
type AuthRequest struct {
	Auth AuthBody `json:"auth"`
}

// AuthBody contains the identity information
type AuthBody struct {
	Identity Identity `json:"identity"`
}

// Identity contains the authentication methods
type Identity struct {
	Methods  []string         `json:"methods"`
	Password PasswordIdentity `json:"password"`
}

// PasswordIdentity contains user credentials
type PasswordIdentity struct {
	User UserCredentials `json:"user"`
}

// UserCredentials contains the user's login credentials
type UserCredentials struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"` // Only for registration
	Role     string `json:"role,omitempty"`  // Only for registration
}

// TokenClaims represents the JWT token claims
type TokenClaims struct {
	UserID      string          `json:"user_id"`
	UserName    string          `json:"user_name"`
	Email       string          `json:"email"`
	RoleID      string          `json:"role_id"`
	RoleName    string          `json:"role"`
	Permissions RolePermissions `json:"permissions"`
	TokenID     string          `json:"jti"` // JWT ID for revocation tracking
}

// HasWriteAccess returns true if the token has write permissions
func (c *TokenClaims) HasWriteAccess() bool {
	return c.Permissions.CanWrite
}

// HasDeleteAccess returns true if the token has delete permissions
func (c *TokenClaims) HasDeleteAccess() bool {
	return c.Permissions.CanDelete
}

// HasAdminAccess returns true if the token has admin permissions
func (c *TokenClaims) HasAdminAccess() bool {
	return c.Permissions.CanAdmin
}

// JWTService handles JWT token generation and validation
type JWTService struct {
	secretKey            []byte
	issuer               string
	tokenDuration        time.Duration
	refreshTokenDuration time.Duration
	userManager          *UserManager
}

// JWTConfig holds JWT service configuration
type JWTConfig struct {
	Issuer               string
	TokenDuration        time.Duration
	RefreshTokenDuration time.Duration
}

// SettingsStore interface for getting/setting persistent settings
type SettingsStore interface {
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
}

// jwtClaims represents the full JWT claims structure (internal use)
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
