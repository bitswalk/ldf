package api

import (
	"fmt"
	"net/http"

	"github.com/bitswalk/ldf/src/common/errors"
	"github.com/bitswalk/ldf/src/ldfd/api/common"
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
)

// rateLimitAuth returns middleware that rate-limits auth endpoints (login/create/refresh).
func (a *API) rateLimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.rateLimiter == nil {
			c.Next()
			return
		}
		key := "ip:" + c.ClientIP()
		if !a.rateLimiter.Allow(key, a.rateLimiter.config.AuthRequestsPerMin) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, errors.ErrRateLimited.ToResponse())
			return
		}
		c.Next()
	}
}

// rateLimitAPI returns middleware that rate-limits general API endpoints.
func (a *API) rateLimitAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.rateLimiter == nil {
			c.Next()
			return
		}
		// Use user ID if authenticated, otherwise fall back to IP
		key := "ip:" + c.ClientIP()
		if claims, ok := c.Get("claims"); ok {
			if tc, ok := claims.(*auth.TokenClaims); ok {
				key = fmt.Sprintf("user:%s", tc.UserID)
			}
		}
		if !a.rateLimiter.Allow(key, a.rateLimiter.config.APIRequestsPerMin) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, errors.ErrRateLimited.ToResponse())
			return
		}
		c.Next()
	}
}

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
