package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zgsm-ai/oidc-auth/pkg/errs"
	"github.com/zgsm-ai/oidc-auth/pkg/response"
)

// SystemAuth creates a middleware for system API authentication
func SystemAuth(APIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.JSONError(c, http.StatusUnauthorized, errs.ErrAuthentication, "Missing Authorization header")
			c.Abort()
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.JSONError(c, http.StatusUnauthorized, errs.ErrAuthentication, "Invalid Authorization header format")
			c.Abort()
			return
		}

		// Validate API key
		if parts[1] != APIKey {
			response.JSONError(c, http.StatusUnauthorized, errs.ErrAuthentication, "Invalid API key")
			c.Abort()
			return
		}

		c.Next()
	}
}
