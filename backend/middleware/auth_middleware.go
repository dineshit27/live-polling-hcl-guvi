package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"live-polling-backend/services"
)

const (
	ContextUserIDKey = "auth_user_id"
	ContextEmailKey  = "auth_email"
	ContextNameKey   = "auth_name"
)

func respondUnauthorized(c *gin.Context, code, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error":   code,
		"message": message,
	})
}

// AuthMiddleware validates the Authorization Bearer JWT header.
func AuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondUnauthorized(c, "unauthorized", "Missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			respondUnauthorized(c, "unauthorized", "Invalid Authorization header format. Expected 'Bearer <token>'")
			return
		}

		tokenString := strings.TrimSpace(parts[1])
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			respondUnauthorized(c, "unauthorized", "Invalid, expired, or tampered token")
			return
		}

		// Store verified user claims in request context
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextEmailKey, claims.Email)
		c.Set(ContextNameKey, claims.Name)

		c.Next()
	}
}

// OptionalAuthMiddleware inspects Authorization header if present, but does not reject anonymous visitors.
func OptionalAuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString := strings.TrimSpace(parts[1])
				if claims, err := authService.ValidateToken(tokenString); err == nil {
					c.Set(ContextUserIDKey, claims.UserID)
					c.Set(ContextEmailKey, claims.Email)
					c.Set(ContextNameKey, claims.Name)
				}
			}
		}
		c.Next()
	}
}

// GetAuthUserID extracts the authenticated user's ID from gin.Context.
func GetAuthUserID(c *gin.Context) (string, bool) {
	val, exists := c.Get(ContextUserIDKey)
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}
