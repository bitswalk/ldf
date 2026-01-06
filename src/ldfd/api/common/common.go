package common

import (
	"github.com/bitswalk/ldf/src/ldfd/auth"
	"github.com/gin-gonic/gin"
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
