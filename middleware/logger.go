package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// After response
		duration := time.Since(start)
		status := c.Writer.Status()

		log.Printf(
			"[HTTP] %s %s - %d (%s)",
			c.Request.Method,
			c.Request.URL.Path,
			status,
			duration,
		)
	}
}
