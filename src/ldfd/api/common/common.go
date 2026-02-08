package common

import (
	"net/http"
	"strconv"

	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
)

const (
	DefaultPaginationLimit = 50
	MaxPaginationLimit     = 500
)

// GetClaimsFromContext retrieves the token claims stored by auth middleware
func GetClaimsFromContext(c *gin.Context) *auth.TokenClaims {
	if claims, exists := c.Get("claims"); exists {
		if tokenClaims, ok := claims.(*auth.TokenClaims); ok {
			return tokenClaims
		}
	}
	return nil
}

// GetTokenClaimsFromRequest extracts and validates JWT claims from the request headers
func GetTokenClaimsFromRequest(c *gin.Context, jwtService *auth.JWTService) *auth.TokenClaims {
	token := c.GetHeader("X-Subject-Token")
	if token == "" {
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	// Fallback to query parameter for SSE/EventSource connections
	// (browser EventSource API cannot set custom headers)
	if token == "" {
		token = c.Query("token")
	}

	if token == "" {
		return nil
	}

	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		return nil
	}

	return claims
}

// GetPaginationParams extracts limit and offset from query parameters
func GetPaginationParams(c *gin.Context, maxLimit int) (int, int) {
	limit := DefaultPaginationLimit
	offset := 0

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= maxLimit {
			limit = parsed
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	return limit, offset
}

// BadRequest sends a 400 Bad Request response
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   "Bad request",
		Code:    http.StatusBadRequest,
		Message: message,
	})
}

// NotFound sends a 404 Not Found response
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   "Not found",
		Code:    http.StatusNotFound,
		Message: message,
	})
}

// InternalError sends a 500 Internal Server Error response
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "Internal server error",
		Code:    http.StatusInternalServerError,
		Message: message,
	})
}

// Unauthorized sends a 401 Unauthorized response
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, ErrorResponse{
		Error:   "Unauthorized",
		Code:    http.StatusUnauthorized,
		Message: message,
	})
}

// Forbidden sends a 403 Forbidden response
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, ErrorResponse{
		Error:   "Forbidden",
		Code:    http.StatusForbidden,
		Message: message,
	})
}

// Conflict sends a 409 Conflict response
func Conflict(c *gin.Context, message string) {
	c.JSON(http.StatusConflict, ErrorResponse{
		Error:   "Conflict",
		Code:    http.StatusConflict,
		Message: message,
	})
}

// ServiceUnavailable sends a 503 Service Unavailable response
func ServiceUnavailable(c *gin.Context, message string) {
	c.JSON(http.StatusServiceUnavailable, ErrorResponse{
		Error:   "Service unavailable",
		Code:    http.StatusServiceUnavailable,
		Message: message,
	})
}

// AbortUnauthorized aborts the request with a 401 Unauthorized response
func AbortUnauthorized(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{
		Error:   "Unauthorized",
		Code:    http.StatusUnauthorized,
		Message: message,
	})
}

// AbortForbidden aborts the request with a 403 Forbidden response
func AbortForbidden(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{
		Error:   "Forbidden",
		Code:    http.StatusForbidden,
		Message: message,
	})
}

// BadGateway sends a 502 Bad Gateway response
func BadGateway(c *gin.Context, message string) {
	c.JSON(http.StatusBadGateway, ErrorResponse{
		Error:   "Bad gateway",
		Code:    http.StatusBadGateway,
		Message: message,
	})
}

// AbortTooManyRequests aborts the request with a 429 Too Many Requests response
func AbortTooManyRequests(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{
		Error:   "Too many requests",
		Code:    http.StatusTooManyRequests,
		Message: message,
	})
}
