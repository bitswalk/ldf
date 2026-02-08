package auth

import (
	"net/http"
	"strings"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/bitswalk/ldf/src/common/logs"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	coreauth "github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var log = logs.NewDefault()

// SetLogger sets the logger for the auth api package
func SetLogger(l *logs.Logger) {
	if l != nil {
		log = l
	}
}

// NewHandler creates a new auth handler
func NewHandler(cfg Config) *Handler {
	return &Handler{
		userManager: cfg.UserManager,
		jwtService:  cfg.JWTService,
	}
}

// HandleCreate handles user registration and creates a new user account
// @Summary      Create a new user account
// @Description  Registers a new user with username, password, and email. Returns JWT tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      AuthRequest  true  "Authentication request"
// @Success      200      {object}  object       "User info with JWT tokens"
// @Failure      400      {object}  common.ErrorResponse
// @Failure      409      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Router       /auth/create [post]
func (h *Handler) HandleCreate(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidJSON.ToResponse())
		return
	}

	if len(req.Auth.Identity.Methods) == 0 || req.Auth.Identity.Methods[0] != "password" {
		c.JSON(http.StatusBadRequest, errors.ErrValidationFailed.WithMessage("Only password authentication method is supported").ToResponse())
		return
	}

	creds := req.Auth.Identity.Password.User
	if creds.Name == "" || creds.Password == "" || creds.Email == "" {
		c.JSON(http.StatusBadRequest, errors.ErrMissingRequiredField.WithMessage("Name, password, and email are required").ToResponse())
		return
	}

	roleID := coreauth.RoleIDDeveloper
	if creds.Role != "" {
		role, err := h.userManager.GetRoleByName(creds.Role)
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

	if roleID == coreauth.RoleIDRoot {
		hasRoot, err := h.userManager.HasRootUser()
		if err != nil {
			c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
			return
		}
		if hasRoot {
			c.JSON(http.StatusConflict, errors.ErrRootUserExists.ToResponse())
			return
		}
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	user := coreauth.NewUser(creds.Name, creds.Email, string(passwordHash), roleID)
	if err := h.userManager.CreateUser(user); err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	user, err = h.userManager.GetUserByID(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	tokenPair, err := h.jwtService.GenerateTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "auth.create_user", UserID: user.ID, UserName: user.Name, Success: true})

	c.Header("X-Subject-Token", tokenPair.AccessToken)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"role":       user.RoleName,
			"role_id":    user.RoleID,
			"created_at": user.CreatedAt,
		},
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"expires_in":    tokenPair.ExpiresIn,
	})
}

// HandleLogin handles user authentication with username and password
// @Summary      Authenticate a user
// @Description  Authenticates a user with username and password. Returns JWT tokens.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      AuthRequest  true  "Authentication request"
// @Success      200      {object}  object       "User info with JWT tokens"
// @Failure      400      {object}  common.ErrorResponse
// @Failure      401      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Router       /auth/login [post]
func (h *Handler) HandleLogin(c *gin.Context) {
	var req AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidJSON.ToResponse())
		return
	}

	if len(req.Auth.Identity.Methods) == 0 || req.Auth.Identity.Methods[0] != "password" {
		c.JSON(http.StatusBadRequest, errors.ErrValidationFailed.WithMessage("Only password authentication method is supported").ToResponse())
		return
	}

	creds := req.Auth.Identity.Password.User
	if creds.Name == "" || creds.Password == "" {
		c.JSON(http.StatusBadRequest, errors.ErrMissingRequiredField.WithMessage("Name and password are required").ToResponse())
		return
	}

	user, err := h.userManager.GetUserByName(creds.Name)
	if err != nil {
		if errors.Is(err, errors.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, errors.ErrInvalidCredentials.ToResponse())
			return
		}
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(creds.Password)); err != nil {
		common.AuditLog(c, common.AuditEvent{Action: "auth.login", UserName: creds.Name, Detail: "invalid password", Success: false})
		c.JSON(http.StatusUnauthorized, errors.ErrInvalidCredentials.ToResponse())
		return
	}

	tokenPair, err := h.jwtService.GenerateTokenPair(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "auth.login", UserID: user.ID, UserName: user.Name, Success: true})

	c.Header("X-Subject-Token", tokenPair.AccessToken)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":      user.ID,
			"name":    user.Name,
			"email":   user.Email,
			"role":    user.RoleName,
			"role_id": user.RoleID,
		},
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"expires_in":    tokenPair.ExpiresIn,
	})
}

// HandleLogout handles user logout and revokes the current JWT token
// @Summary      Logout and revoke token
// @Description  Revokes the current JWT token, effectively logging the user out.
// @Tags         Auth
// @Produce      json
// @Success      498  {object}  object  "Token revoked successfully"
// @Failure      401  {object}  common.ErrorResponse
// @Failure      500  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (h *Handler) HandleLogout(c *gin.Context) {
	token := c.GetHeader("X-Subject-Token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if t, found := strings.CutPrefix(authHeader, "Bearer "); found {
			token = t
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, errors.ErrNoToken.ToResponse())
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		if errors.Is(err, errors.ErrTokenRevoked) {
			c.JSON(http.StatusUnauthorized, errors.ErrTokenRevoked.WithMessage("Token already revoked").ToResponse())
			return
		}
		c.JSON(http.StatusUnauthorized, errors.ErrTokenInvalid.ToResponse())
		return
	}

	if err := h.jwtService.RevokeToken(token); err != nil {
		if log != nil {
			log.Error("Failed to revoke token", "user", claims.UserName, "user_id", claims.UserID, "error", err)
		}
		c.JSON(http.StatusInternalServerError, errors.ErrInternal.ToResponse())
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "auth.logout", UserID: claims.UserID, UserName: claims.UserName, Success: true})

	c.JSON(498, gin.H{
		"message": "Token revoked successfully",
		"user_id": claims.UserID,
	})
}

// HandleRefresh handles token refresh requests
// @Summary      Refresh access token
// @Description  Refreshes an expired access token using a valid refresh token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      RefreshRequest  true  "Refresh token request"
// @Success      200      {object}  object          "New JWT tokens with user info"
// @Failure      400      {object}  common.ErrorResponse
// @Failure      401      {object}  common.ErrorResponse
// @Failure      500      {object}  common.ErrorResponse
// @Router       /auth/refresh [post]
func (h *Handler) HandleRefresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ErrInvalidJSON.ToResponse())
		return
	}

	tokenPair, user, err := h.jwtService.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		common.AuditLog(c, common.AuditEvent{Action: "auth.refresh", Detail: "invalid refresh token", Success: false})
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	common.AuditLog(c, common.AuditEvent{Action: "auth.refresh", UserID: user.ID, UserName: user.Name, Success: true})

	c.Header("X-Subject-Token", tokenPair.AccessToken)
	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_at":    tokenPair.ExpiresAt,
		"expires_in":    tokenPair.ExpiresIn,
		"user": gin.H{
			"id":      user.ID,
			"name":    user.Name,
			"email":   user.Email,
			"role":    user.RoleName,
			"role_id": user.RoleID,
		},
	})
}

// HandleValidate validates the current access token and returns user info
// @Summary      Validate access token
// @Description  Validates the current access token and returns the associated user information.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  object  "Token validation result with user info"
// @Failure      401  {object}  common.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/validate [get]
func (h *Handler) HandleValidate(c *gin.Context) {
	token := c.GetHeader("X-Subject-Token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if t, found := strings.CutPrefix(authHeader, "Bearer "); found {
			token = t
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, errors.ErrNoToken.ToResponse())
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		status := errors.GetHTTPStatus(err)
		c.JSON(status, errors.NewResponse(err))
		return
	}

	if log != nil {
		log.Debug("Token validated", "user", claims.UserName, "user_id", claims.UserID)
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"user": gin.H{
			"id":          claims.UserID,
			"name":        claims.UserName,
			"email":       claims.Email,
			"role":        claims.RoleName,
			"role_id":     claims.RoleID,
			"permissions": claims.Permissions,
		},
	})
}
