package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"sudo/internal/database"
	"sudo/internal/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RealtimeAuthMiddleware validates WebSocket connections
func RealtimeAuthMiddleware(db *database.DB) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// For WebSocket endpoints, validate session
		if c.Request.Header.Get("Upgrade") == "websocket" {
			user, err := validateWebSocketSession(c, db)
			if err != nil {
				log.Printf("WebSocket auth failed: %v", err)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Authentication required for WebSocket connection",
				})
				return
			}

			// Store user in context for WebSocket handler
			c.Set("ws_user", user)
		}

		c.Next()
	})
}

func validateWebSocketSession(c *gin.Context, db *database.DB) (*models.User, error) {
	session := sessions.Default(c)
	userIDStr := session.Get("user_id")
	if userIDStr == nil {
		return nil, fmt.Errorf("no valid session")
	}

	userIDString, ok := userIDStr.(string)
	if !ok {
		return nil, fmt.Errorf("invalid user ID type in session")
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in session")
	}

	user, err := db.GetUserByID(context.Background(), userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}
