// Package middleware holds gin middleware shared by every module's
// presentation/http layer (as opposed to module-specific middleware,
// which lives under modules/<name>/presentation/http/middleware).
package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs method, path, status and latency for every request.
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		logger.Info("http_request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(start).String(),
		)
	}
}

// Recovery converts panics into a 500 JSON response instead of crashing
// the process, logging the recovered value for diagnostics.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic_recovered", "error", r)
				c.AbortWithStatusJSON(500, gin.H{"success": false, "error": "internal server error"})
			}
		}()
		c.Next()
	}
}

// CORS applies a permissive CORS policy suitable for local development.
// Tighten AllowOrigin before deploying to production.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
