package auth

import (
	coreauth "github.com/bitswalk/ldf/src/ldfd/auth"
)

// Handler handles authentication HTTP requests
type Handler struct {
	userManager *coreauth.UserManager
	jwtService  *coreauth.JWTService
}

// Config contains configuration options for the Handler
type Config struct {
	UserManager *coreauth.UserManager
	JWTService  *coreauth.JWTService
}

// AuthRequest represents the authentication request structure
type AuthRequest struct {
	Auth struct {
		Identity struct {
			Methods  []string `json:"methods"`
			Password struct {
				User struct {
					Name     string `json:"name"`
					Password string `json:"password"`
					Email    string `json:"email"`
					Role     string `json:"role"`
				} `json:"user"`
			} `json:"password"`
		} `json:"identity"`
	} `json:"auth"`
}

// RefreshRequest represents the refresh token request body
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RoleCreateRequest represents the request body for creating a new role
type RoleCreateRequest struct {
	Name         string `json:"name" binding:"required"`
	Description  string `json:"description"`
	ParentRoleID string `json:"parent_role_id"`
	Permissions  struct {
		CanRead   bool `json:"can_read"`
		CanWrite  bool `json:"can_write"`
		CanDelete bool `json:"can_delete"`
		CanAdmin  bool `json:"can_admin"`
	} `json:"permissions"`
}

// RoleUpdateRequest represents the request body for updating a role
type RoleUpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Permissions *struct {
		CanRead   bool `json:"can_read"`
		CanWrite  bool `json:"can_write"`
		CanDelete bool `json:"can_delete"`
		CanAdmin  bool `json:"can_admin"`
	} `json:"permissions"`
}
