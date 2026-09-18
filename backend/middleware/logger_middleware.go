package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggerMiddleware logs incoming requests with latency, client IP, method, and status.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[API] %3d | %13v | %15s | %-7s %s",
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

// RecoveryMiddleware gracefully catches panics and responds with 500 JSON.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERED] %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal server error",
					"message": "An unexpected server error occurred. Please try again later.",
				})
			}
		}()
		c.Next()
	}
}
