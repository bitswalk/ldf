package api

import (
	"net/http"
	"strings"

	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
)

// getTokenClaims extracts and validates JWT claims from the request
// Returns nil if no valid token is present (for optional auth)
func (a *API) getTokenClaims(c *gin.Context) *auth.TokenClaims {
	// Check X-Subject-Token header first
	token := c.GetHeader("X-Subject-Token")
	if token == "" {
		// Check Authorization header
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		return nil
	}

	claims, err := a.jwtService.ValidateToken(token)
	if err != nil {
		return nil
	}

	return claims
}

// authRequired is a middleware that requires a valid JWT token
func (a *API) authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := a.getTokenClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Unauthorized",
				Code:    http.StatusUnauthorized,
				Message: "Authentication required",
			})
			return
		}

		// Store claims in context for handlers to use
		c.Set("claims", claims)
		c.Next()
	}
}

// writeAccessRequired is a middleware that requires authenticated user with write access
func (a *API) writeAccessRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := a.getTokenClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Unauthorized",
				Code:    http.StatusUnauthorized,
				Message: "Authentication required",
			})
			return
		}

		// Check if user has write permission
		if !claims.HasWriteAccess() {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Code:    http.StatusForbidden,
				Message: "Write access denied",
			})
			return
		}

		// Store claims in context for handlers to use
		c.Set("claims", claims)
		c.Next()
	}
}

// deleteAccessRequired is a middleware that requires authenticated user with delete access
func (a *API) deleteAccessRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := a.getTokenClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Unauthorized",
				Code:    http.StatusUnauthorized,
				Message: "Authentication required",
			})
			return
		}

		// Check if user has delete permission
		if !claims.HasDeleteAccess() {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Code:    http.StatusForbidden,
				Message: "Delete access denied",
			})
			return
		}

		// Store claims in context for handlers to use
		c.Set("claims", claims)
		c.Next()
	}
}

// adminAccessRequired is a middleware that requires authenticated user with admin access
func (a *API) adminAccessRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := a.getTokenClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Unauthorized",
				Code:    http.StatusUnauthorized,
				Message: "Authentication required",
			})
			return
		}

		// Check if user has admin permission
		if !claims.HasAdminAccess() {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Code:    http.StatusForbidden,
				Message: "Admin access denied",
			})
			return
		}

		// Store claims in context for handlers to use
		c.Set("claims", claims)
		c.Next()
	}
}

// rootAccessRequired is a middleware that requires authenticated user with root role
func (a *API) rootAccessRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := a.getTokenClaims(c)
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "Unauthorized",
				Code:    http.StatusUnauthorized,
				Message: "Authentication required",
			})
			return
		}

		// Must be root role specifically
		if claims.RoleID != auth.RoleIDRoot {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
				Error:   "Forbidden",
				Code:    http.StatusForbidden,
				Message: "Root access required",
			})
			return
		}

		// Store claims in context for handlers to use
		c.Set("claims", claims)
		c.Next()
	}
}

// getClaimsFromContext retrieves the token claims stored by auth middleware
func getClaimsFromContext(c *gin.Context) *auth.TokenClaims {
	if claims, exists := c.Get("claims"); exists {
		if tokenClaims, ok := claims.(*auth.TokenClaims); ok {
			return tokenClaims
		}
	}
	return nil
}
