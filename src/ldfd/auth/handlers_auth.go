package auth

import (
	"net/http"
	"strings"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Package-level logger, must be initialized via SetLogger
var log *logs.Logger

// SetLogger sets the package-level logger
func SetLogger(l *logs.Logger) {
	log = l
}

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

// HandleCreate handles user registration and creates a new user account with the provided credentials
func (h *Handler) HandleCreate(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidJSON.ToResponse())
		return
	}

	// Validate request structure
	if len(req.Auth.Identity.Methods) == 0 || req.Auth.Identity.Methods[0] != "password" {
		c.JSON(http.StatusBadRequest, errors.ErrValidationFailed.WithMessage("Only password authentication method is supported").ToResponse())
		return
	}

	creds := req.Auth.Identity.Password.User
	if creds.Name == "" || creds.Password == "" || creds.Email == "" {
		c.JSON(http.StatusBadRequest, errors.ErrMissingRequiredField.WithMessage("Name, password, and email are required").ToResponse())
		return
	}

	// Determine role ID (default to developer)
	roleID := RoleIDDeveloper
	if creds.Role != "" {
		// Look up the role by name
		role, err := h.repo.GetRoleByName(creds.Role)
		if err != nil {
			if errors.Is(err, errors.ErrRoleNotFound) {
				c.JSON(http.StatusBadRequest, errors.ErrRoleNotFound.WithMessage("Invalid role specified").ToResponse())
				return
			}
			c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
			return
		}
		roleID = role.ID
	}

	// If requesting root role, check if root already exists
	if roleID == RoleIDRoot {
		hasRoot, err := h.repo.HasRootUser()
		if err != nil {
			c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
			return
		}
		if hasRoot {
			c.JSON(http.StatusConflict, errors.ErrRootUserExists.ToResponse())
			return
		}
	}

	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	// Create the user
	user := NewUser(creds.Name, creds.Email, string(passwordHash), roleID)
	if err := h.repo.CreateUser(user); err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	// Fetch the user with role information for the response and token generation
	user, err = h.repo.GetUserByID(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
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
		c.JSON(http.StatusBadRequest, errors.ErrInvalidJSON.ToResponse())
		return
	}

	// Validate request structure
	if len(req.Auth.Identity.Methods) == 0 || req.Auth.Identity.Methods[0] != "password" {
		c.JSON(http.StatusBadRequest, errors.ErrValidationFailed.WithMessage("Only password authentication method is supported").ToResponse())
		return
	}

	creds := req.Auth.Identity.Password.User
	if creds.Name == "" || creds.Password == "" {
		c.JSON(http.StatusBadRequest, errors.ErrMissingRequiredField.WithMessage("Name and password are required").ToResponse())
		return
	}

	// Find user by name
	user, err := h.repo.GetUserByName(creds.Name)
	if err != nil {
		if errors.Is(err, errors.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, errors.ErrInvalidCredentials.ToResponse())
			return
		}
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, errors.ErrInvalidCredentials.ToResponse())
		return
	}

	// Generate JWT token
	token, err := h.jwtService.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
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
		c.JSON(http.StatusUnauthorized, errors.ErrNoToken.ToResponse())
		return
	}

	// Validate the token first
	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		if errors.Is(err, errors.ErrTokenRevoked) {
			c.JSON(http.StatusUnauthorized, errors.ErrTokenRevoked.WithMessage("Token already revoked").ToResponse())
			return
		}
		c.JSON(http.StatusUnauthorized, errors.ErrTokenInvalid.ToResponse())
		return
	}

	// Revoke the token
	if err := h.jwtService.RevokeToken(token); err != nil {
		if log != nil {
			log.Error("Failed to revoke token", "user", claims.UserName, "user_id", claims.UserID, "error", err)
		}
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	if log != nil {
		log.Info("User logged out", "user", claims.UserName, "user_id", claims.UserID)
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
