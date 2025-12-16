// Package auth provides authentication and authorization functionality for ldfd.
package auth

import (
	"time"

	"github.com/google/uuid"
)

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

// NewUser creates a new user with a generated UUID
func NewUser(name, email, passwordHash, roleID string) *User {
	now := time.Now().UTC()
	return &User{
		ID:           uuid.New().String(),
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		RoleID:       roleID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// RevokedToken represents a revoked JWT token
type RevokedToken struct {
	TokenID   string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	RevokedAt time.Time `json:"revoked_at"`
	ExpiresAt time.Time `json:"expires_at"` // Keep until original expiry for cleanup
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
