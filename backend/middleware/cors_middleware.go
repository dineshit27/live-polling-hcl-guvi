package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware configures CORS headers allowing safe cross-origin access.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowAll := false
		matched := false

		for _, o := range allowedOrigins {
			trimmed := strings.TrimSpace(o)
			if trimmed == "*" {
				allowAll = true
				break
			}
			if origin != "" && (trimmed == origin || strings.TrimRight(trimmed, "/") == strings.TrimRight(origin, "/")) {
				matched = true
				break
			}
		}

		if matched && origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if allowAll {
			// RFC 6454 / W3C CORS: Wildcard '*' MUST NOT be combined with Allow-Credentials: true
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		} else if len(allowedOrigins) > 0 && origin == "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigins[0])
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
