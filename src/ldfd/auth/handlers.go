package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Handler handles authentication HTTP requests
type Handler struct {
	repo       *Repository
	jwtService *JWTService
}

// NewHandler creates a new auth handler
func NewHandler(repo *Repository, jwtService *JWTService) *Handler {
	return &Handler{
		repo:       repo,
		jwtService: jwtService,
	}
}

// authErrorResponse represents an authentication error response
type authErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HandleCreate handles user registration and creates a new user account with the provided credentials
func (h *Handler) HandleCreate(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Validate request structure
	if len(req.Auth.Identity.Methods) == 0 || req.Auth.Identity.Methods[0] != "password" {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Only password authentication method is supported",
		})
		return
	}

	creds := req.Auth.Identity.Password.User
	if creds.Name == "" || creds.Password == "" || creds.Email == "" {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Name, password, and email are required",
		})
		return
	}

	// Determine role ID (default to developer)
	roleID := RoleIDDeveloper
	if creds.Role != "" {
		// Look up the role by name
		role, err := h.repo.GetRoleByName(creds.Role)
		if err != nil {
			if errors.Is(err, ErrRoleNotFound) {
				c.JSON(http.StatusBadRequest, authErrorResponse{
					Error:   "invalid_request",
					Message: "Invalid role specified",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, authErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate role",
			})
			return
		}
		roleID = role.ID
	}

	// If requesting root role, check if root already exists
	if roleID == RoleIDRoot {
		hasRoot, err := h.repo.HasRootUser()
		if err != nil {
			c.JSON(http.StatusInternalServerError, authErrorResponse{
				Error:   "internal_error",
				Message: "Failed to check root user existence",
			})
			return
		}
		if hasRoot {
			c.JSON(http.StatusUnauthorized, authErrorResponse{
				Error:   "unauthorized",
				Message: "Root user already exists",
			})
			return
		}
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to hash password",
		})
		return
	}

	// Create the user
	user := NewUser(creds.Name, creds.Email, string(passwordHash), roleID)
	if err := h.repo.CreateUser(user); err != nil {
		if errors.Is(err, ErrEmailExists) {
			c.JSON(http.StatusUnauthorized, authErrorResponse{
				Error:   "unauthorized",
				Message: "Email already exists",
			})
			return
		}
		if errors.Is(err, ErrUserExists) {
			c.JSON(http.StatusUnauthorized, authErrorResponse{
				Error:   "unauthorized",
				Message: "Username already exists",
			})
			return
		}
		if errors.Is(err, ErrRootExists) {
			c.JSON(http.StatusUnauthorized, authErrorResponse{
				Error:   "unauthorized",
				Message: "Root user already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create user",
		})
		return
	}

	// Fetch the user with role information for the response and token generation
	user, err = h.repo.GetUserByID(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to retrieve created user",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate token",
		})
		return
	}

	// Set token in header
	c.Header("X-Subject-Token", token)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"role":       user.RoleName,
			"role_id":    user.RoleID,
			"created_at": user.CreatedAt,
		},
	})
}

// HandleLogin handles user authentication with username and password
func (h *Handler) HandleLogin(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Validate request structure
	if len(req.Auth.Identity.Methods) == 0 || req.Auth.Identity.Methods[0] != "password" {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Only password authentication method is supported",
		})
		return
	}

	creds := req.Auth.Identity.Password.User
	if creds.Name == "" || creds.Password == "" {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Name and password are required",
		})
		return
	}

	// Find user by name
	user, err := h.repo.GetUserByName(creds.Name)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, authErrorResponse{
				Error:   "unauthorized",
				Message: "Invalid credentials",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to authenticate",
		})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, authErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid credentials",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate token",
		})
		return
	}

	// Set token in header
	c.Header("X-Subject-Token", token)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":      user.ID,
			"name":    user.Name,
			"email":   user.Email,
			"role":    user.RoleName,
			"role_id": user.RoleID,
		},
	})
}

// HandleLogout handles user logout and revokes the current JWT token
func (h *Handler) HandleLogout(c *gin.Context) {
	// Get token from header
	token := c.GetHeader("X-Subject-Token")
	if token == "" {
		// Also check Authorization header
		authHeader := c.GetHeader("Authorization")
		if t, found := strings.CutPrefix(authHeader, "Bearer "); found {
			token = t
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, authErrorResponse{
			Error:   "unauthorized",
			Message: "No token provided",
		})
		return
	}

	// Validate the token first
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		if errors.Is(err, ErrTokenRevoked) {
			c.JSON(http.StatusUnauthorized, authErrorResponse{
				Error:   "unauthorized",
				Message: "Token already revoked",
			})
			return
		}
		c.JSON(http.StatusUnauthorized, authErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid token",
		})
		return
	}

	// Revoke the token
	if err := h.jwtService.RevokeToken(token); err != nil {
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to revoke token",
		})
		return
	}

	// Return 498 as specified
	c.JSON(498, gin.H{
		"message": "Token revoked successfully",
		"user_id": claims.UserID,
	})
}

// ExtractTokenFromRequest extracts JWT token from request headers
func ExtractTokenFromRequest(c *gin.Context) string {
	// Check X-Subject-Token header first
	token := c.GetHeader("X-Subject-Token")
	if token != "" {
		return token
	}

	// Check Authorization header
	if token, found := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer "); found {
		return token
	}

	return ""
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

// HandleListRoles returns all available roles
func (h *Handler) HandleListRoles(c *gin.Context) {
	roles, err := h.repo.ListRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list roles",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"roles": roles,
	})
}

// HandleGetRole returns a specific role by ID
func (h *Handler) HandleGetRole(c *gin.Context) {
	id := c.Param("id")

	role, err := h.repo.GetRoleByID(id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, authErrorResponse{
				Error:   "not_found",
				Message: "Role not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get role",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role": role,
	})
}

// HandleCreateRole creates a new custom role
func (h *Handler) HandleCreateRole(c *gin.Context) {
	var req RoleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Validate parent role if specified
	if req.ParentRoleID != "" {
		_, err := h.repo.GetRoleByID(req.ParentRoleID)
		if err != nil {
			if errors.Is(err, ErrRoleNotFound) {
				c.JSON(http.StatusBadRequest, authErrorResponse{
					Error:   "invalid_request",
					Message: "Parent role not found",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, authErrorResponse{
				Error:   "internal_error",
				Message: "Failed to validate parent role",
			})
			return
		}
	}

	role := NewRole(req.Name, req.Description, RolePermissions{
		CanRead:   req.Permissions.CanRead,
		CanWrite:  req.Permissions.CanWrite,
		CanDelete: req.Permissions.CanDelete,
		CanAdmin:  req.Permissions.CanAdmin,
	}, req.ParentRoleID)

	if err := h.repo.CreateRole(role); err != nil {
		if errors.Is(err, ErrRoleNameExists) {
			c.JSON(http.StatusConflict, authErrorResponse{
				Error:   "conflict",
				Message: "Role name already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create role",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"role": role,
	})
}

// HandleUpdateRole updates an existing custom role
func (h *Handler) HandleUpdateRole(c *gin.Context) {
	id := c.Param("id")

	var req RoleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, authErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Fetch existing role
	role, err := h.repo.GetRoleByID(id)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, authErrorResponse{
				Error:   "not_found",
				Message: "Role not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get role",
		})
		return
	}

	// Apply updates
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.Permissions != nil {
		role.Permissions = RolePermissions{
			CanRead:   req.Permissions.CanRead,
			CanWrite:  req.Permissions.CanWrite,
			CanDelete: req.Permissions.CanDelete,
			CanAdmin:  req.Permissions.CanAdmin,
		}
	}

	if err := h.repo.UpdateRole(role); err != nil {
		if errors.Is(err, ErrSystemRole) {
			c.JSON(http.StatusForbidden, authErrorResponse{
				Error:   "forbidden",
				Message: "Cannot modify system role",
			})
			return
		}
		if errors.Is(err, ErrRoleNameExists) {
			c.JSON(http.StatusConflict, authErrorResponse{
				Error:   "conflict",
				Message: "Role name already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update role",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role": role,
	})
}

// HandleDeleteRole deletes a custom role
func (h *Handler) HandleDeleteRole(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.DeleteRole(id); err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			c.JSON(http.StatusNotFound, authErrorResponse{
				Error:   "not_found",
				Message: "Role not found",
			})
			return
		}
		if errors.Is(err, ErrSystemRole) {
			c.JSON(http.StatusForbidden, authErrorResponse{
				Error:   "forbidden",
				Message: "Cannot delete system role",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, authErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete role",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role deleted successfully",
	})
}
