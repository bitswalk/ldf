package api

import (
	"net/http"

	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
)

// getTokenClaims extracts and validates JWT claims from the request
// Returns nil if no valid token is present (for optional auth)
func (a *API) getTokenClaims(c *gin.Context) *auth.TokenClaims {
	return common.GetTokenClaimsFromRequest(c, a.jwtService)
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
