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

// getTokenClaims extracts and validates JWT claims from the request.
// Returns nil if no valid token is present (for optional auth).
func (a *API) getTokenClaims(c *gin.Context) *auth.TokenClaims {
	return common.GetTokenClaimsFromRequest(c, a.jwtService)
}

// requireAuth returns middleware that requires authentication and an optional permission check.
// If check is nil, only authentication is required (any valid token).
func (a *API) requireAuth(check func(*auth.TokenClaims) bool, denyMsg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := a.getTokenClaims(c)
		if claims == nil {
			common.AbortUnauthorized(c, "Authentication required")
			return
		}
		if check != nil && !check(claims) {
			common.AbortForbidden(c, denyMsg)
			return
		}
		c.Set("claims", claims)
		c.Next()
	}
}

func (a *API) authRequired() gin.HandlerFunc {
	return a.requireAuth(nil, "")
}

func (a *API) writeAccessRequired() gin.HandlerFunc {
	return a.requireAuth(func(c *auth.TokenClaims) bool { return c.HasWriteAccess() }, "Write access denied")
}

func (a *API) adminAccessRequired() gin.HandlerFunc {
	return a.requireAuth(func(c *auth.TokenClaims) bool { return c.HasAdminAccess() }, "Admin access denied")
}

func (a *API) rootAccessRequired() gin.HandlerFunc {
	return a.requireAuth(func(c *auth.TokenClaims) bool { return c.RoleID == auth.RoleIDRoot }, "Root access required")
}
