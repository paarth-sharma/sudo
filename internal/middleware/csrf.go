package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// CSRFMiddleware provides CSRF protection
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)

		// Generate CSRF token if not exists
		token := session.Get("csrf_token")
		if token == nil {
			tokenBytes := make([]byte, 32)
			if _, err := rand.Read(tokenBytes); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CSRF token"})
				c.Abort()
				return
			}
			token = hex.EncodeToString(tokenBytes)
			session.Set("csrf_token", token)
			if err := session.Save(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
				c.Abort()
				return
			}
		}

		// For GET requests, just set the token and continue
		if c.Request.Method == "GET" {
			if tokenStr, ok := token.(string); ok {
				c.Header("X-CSRF-Token", tokenStr)
			}
			c.Next()
			return
		}

		// For POST, PUT, DELETE requests, validate CSRF token
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
			// Skip CSRF check for HTMX requests from the same origin
			if c.GetHeader("HX-Request") == "true" {
				c.Next()
				return
			}

			requestToken := c.GetHeader("X-CSRF-Token")
			if requestToken == "" {
				requestToken = c.PostForm("csrf_token")
			}

			tokenStr, ok := token.(string)
			if !ok || requestToken != tokenStr {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "CSRF token mismatch",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
