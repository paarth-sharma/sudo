package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"sudo/internal/database"
	"sudo/internal/models"
	"sudo/templates/components"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket upgrader with security settings
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate against allowed origins
		return true // TODO: Restrict to your domain in production
	},
}

// Message types for WebSocket communication
const (
	MessageTypeTaskMove       = "task_move"
	MessageTypeTaskCreate     = "task_create"
	MessageTypeTaskUpdate     = "task_update"
	MessageTypeTaskDelete     = "task_delete"
	MessageTypeUserPresence   = "user_presence"
	MessageTypeCursorMove     = "cursor_move"
	MessageTypeError          = "error"
	MessageTypeHTMXUpdate     = "htmx_update"
	MessageTypeMemberAdded    = "member_added"
	MessageTypeMemberRemoved  = "member_removed"
	MessageTypePresenceUpdate = "presence_update"
)

// WebSocket message structure
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	UserID    string                 `json:"user_id"`
	BoardID   string                 `json:"board_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// HTMX-specific message for DOM updates
type HTMXUpdateMessage struct {
	Type         string `json:"type"`
	Target       string `json:"target"`        // CSS selector
	HTMLContent  string `json:"html_content"`  // Rendered HTML
	SwapStrategy string `json:"swap_strategy"` // innerHTML, outerHTML, etc.
	UserID       string `json:"user_id"`
}

// Client represents a WebSocket connection
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	boardID  string
	userID   uuid.UUID
	user     *models.User
	lastSeen time.Time
}

// RealtimeService manages all WebSocket connections
type RealtimeService struct {
	// Board ID -> Client connections map
	clients    map[string]map[*Client]bool
	broadcast  chan *WebSocketMessage
	register   chan *Client
	unregister chan *Client
	db         *database.DB
	mu         sync.RWMutex
}

// NewRealtimeService creates a new real-time service
func NewRealtimeService(db *database.DB) *RealtimeService {
	return &RealtimeService{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *WebSocketMessage, 256),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		db:         db,
	}
}

// Run starts the WebSocket service hub
func (s *RealtimeService) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-s.register:
			s.registerClient(client)

		case client := <-s.unregister:
			s.unregisterClient(client)

		case message := <-s.broadcast:
			s.broadcastToBoard(message)

		case <-ticker.C:
			s.cleanupStaleConnections()
		}
	}
}

// HandleWebSocketConnection upgrades HTTP to WebSocket
func (s *RealtimeService) HandleWebSocketConnection(c *gin.Context) {
	// Validate authentication using existing session
	user, err := s.validateWebSocketAuth(c)
	if err != nil {
		log.Printf("WebSocket auth failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	boardID := c.Param("boardId")
	if boardID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Board ID required"})
		return
	}

	boardUUID, err := uuid.Parse(boardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid board ID"})
		return
	}

	// Verify user has access to this board
	hasAccess, err := s.db.HasBoardAccess(context.Background(), user.ID, boardUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this board"})
		return
	}

	// Upgrade connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client and register
	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		boardID:  boardID,
		userID:   user.ID,
		user:     user,
		lastSeen: time.Now(),
	}

	s.register <- client

	// Start client goroutines
	go s.handleClientWrite(client)
	go s.handleClientRead(client)
}

// validateWebSocketAuth checks session authentication
func (s *RealtimeService) validateWebSocketAuth(c *gin.Context) (*models.User, error) {
	session := sessions.Default(c)
	userIDStr := session.Get("user_id")
	if userIDStr == nil {
		return nil, fmt.Errorf("no valid session")
	}

	userIDStrVal, ok := userIDStr.(string)
	if !ok {
		return nil, fmt.Errorf("user ID in session is not a string")
	}

	userID, err := uuid.Parse(userIDStrVal)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in session")
	}

	// Verify user exists and is valid
	user, err := s.db.GetUserByID(context.Background(), userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return user, nil
}

// registerClient adds a new client connection
func (s *RealtimeService) registerClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Initialize board client map if needed
	if s.clients[client.boardID] == nil {
		s.clients[client.boardID] = make(map[*Client]bool)
	}

	s.clients[client.boardID][client] = true

	log.Printf("User %s connected to board %s. Total connections: %d",
		client.user.Name, client.boardID, len(s.clients[client.boardID]))

	// Update user presence in database
	s.updateUserPresence(client, true)

	// Notify other users of new presence
	s.broadcastPresenceUpdate(client.boardID, client.userID, "joined")

	// Send current board state to new client
	s.sendBoardSnapshot(client)
}

// unregisterClient removes a client connection
func (s *RealtimeService) unregisterClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if clients, exists := s.clients[client.boardID]; exists {
		if _, clientExists := clients[client]; clientExists {
			delete(clients, client)
			close(client.send)

			// Clean up empty board maps
			if len(clients) == 0 {
				delete(s.clients, client.boardID)
			}

			log.Printf("User %s disconnected from board %s",
				client.user.Name, client.boardID)

			// Update presence and notify others
			s.updateUserPresence(client, false)
			s.broadcastPresenceUpdate(client.boardID, client.userID, "left")
		}
	}
}

// broadcastToBoard sends message to all clients in a board
func (s *RealtimeService) broadcastToBoard(message *WebSocketMessage) {
	s.mu.RLock()
	clients := s.clients[message.BoardID]
	s.mu.RUnlock()

	if clients == nil {
		return
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	for client := range clients {
		// Don't send message back to sender
		if client.userID.String() == message.UserID {
			continue
		}

		select {
		case client.send <- messageBytes:
		default:
			// Client buffer full, disconnect
			s.unregister <- client
		}
	}
}

// handleClientRead processes incoming messages from client
func (s *RealtimeService) handleClientRead(client *Client) {
	defer func() {
		s.unregister <- client
		client.conn.Close()
	}()

	// Set read deadline and pong handler
	if err := client.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
		log.Printf("Failed to set read deadline: %v", err)
		return
	}
	client.conn.SetPongHandler(func(string) error {
		if err := client.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			log.Printf("Failed to set read deadline in pong handler: %v", err)
		}
		client.lastSeen = time.Now()
		return nil
	})

	for {
		var message WebSocketMessage
		err := client.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Process message based on type
		s.handleClientMessage(client, &message)
	}
}

// handleClientWrite sends messages to client
func (s *RealtimeService) handleClientWrite(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			if err := client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("Failed to set write deadline: %v", err)
				return
			}
			if !ok {
				_ = client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			if err := client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
				log.Printf("Failed to set write deadline for ping: %v", err)
				return
			}
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage processes different message types
func (s *RealtimeService) handleClientMessage(client *Client, message *WebSocketMessage) {
	// Validate message
	if message.BoardID != client.boardID {
		s.sendErrorToClient(client, "Board ID mismatch")
		return
	}

	// Set message metadata
	message.UserID = client.userID.String()
	message.Timestamp = time.Now()

	switch message.Type {
	case MessageTypeTaskMove:
		s.handleTaskMove(client, message)
	case MessageTypeTaskUpdate:
		s.handleTaskUpdate(client, message)
	case MessageTypeCursorMove:
		s.handleCursorMove(client, message)
	case MessageTypeUserPresence:
		s.handlePresenceUpdate(client, message)
	default:
		s.sendErrorToClient(client, "Unknown message type")
	}
}

// handleTaskMove processes task movement between columns
func (s *RealtimeService) handleTaskMove(client *Client, message *WebSocketMessage) {
	// Extract task movement data
	taskID, _ := message.Data["task_id"].(string)
	columnID, _ := message.Data["column_id"].(string)
	position, _ := message.Data["position"].(float64)
	version, _ := message.Data["version"].(float64)

	if taskID == "" || columnID == "" {
		s.sendErrorToClient(client, "Invalid task move data")
		return
	}

	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		s.sendErrorToClient(client, "Invalid task ID")
		return
	}

	columnUUID, err := uuid.Parse(columnID)
	if err != nil {
		s.sendErrorToClient(client, "Invalid column ID")
		return
	}

	// Update task in database
	updates := map[string]interface{}{
		"column_id": columnUUID,
		"position":  int(position),
		"version":   int(version) + 1,
	}

	err = s.db.UpdateTask(context.Background(), taskUUID, updates)
	if err != nil {
		s.sendErrorToClient(client, fmt.Sprintf("Failed to move task: %v", err))
		return
	}

	// Get updated task for broadcasting
	task, err := s.db.GetTask(context.Background(), taskUUID)
	if err != nil {
		log.Printf("Failed to get updated task: %v", err)
		return
	}

	// Render updated task HTML using existing Templ component
	taskHTML, err := s.renderTaskCard(task)
	if err != nil {
		log.Printf("Failed to render task HTML: %v", err)
		return
	}

	// Broadcast HTMX update to all board clients
	htmxMessage := &WebSocketMessage{
		Type:      MessageTypeHTMXUpdate,
		BoardID:   client.boardID,
		UserID:    client.userID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"target":        fmt.Sprintf("#task-%s", taskID),
			"html_content":  taskHTML,
			"swap_strategy": "outerHTML",
			"task_id":       taskID,
		},
	}

	s.broadcast <- htmxMessage
}

// handleTaskUpdate processes task property updates
func (s *RealtimeService) handleTaskUpdate(client *Client, message *WebSocketMessage) {
	taskID, _ := message.Data["task_id"].(string)
	updates, _ := message.Data["updates"].(map[string]interface{})

	if taskID == "" || updates == nil {
		s.sendErrorToClient(client, "Invalid task update data")
		return
	}

	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		s.sendErrorToClient(client, "Invalid task ID")
		return
	}

	// Validate and sanitize updates
	validatedUpdates := make(map[string]interface{})
	for key, value := range updates {
		switch key {
		case "title", "description", "priority":
			if str, ok := value.(string); ok && str != "" {
				validatedUpdates[key] = str
			}
		case "deadline":
			// Handle deadline string parsing
			if str, ok := value.(string); ok {
				if deadline, parseErr := time.Parse(time.RFC3339, str); parseErr == nil {
					validatedUpdates[key] = deadline
				}
			}
		case "assigned_to":
			if userIDStr, ok := value.(string); ok {
				if userID, parseErr := uuid.Parse(userIDStr); parseErr == nil {
					validatedUpdates[key] = userID
				}
			}
		}
	}

	if len(validatedUpdates) == 0 {
		s.sendErrorToClient(client, "No valid updates provided")
		return
	}

	// Update task in database
	err = s.db.UpdateTask(context.Background(), taskUUID, validatedUpdates)
	if err != nil {
		s.sendErrorToClient(client, fmt.Sprintf("Failed to update task: %v", err))
		return
	}

	// Broadcast update to other clients
	broadcastMessage := &WebSocketMessage{
		Type:      MessageTypeTaskUpdate,
		BoardID:   client.boardID,
		UserID:    client.userID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"task_id": taskID,
			"updates": validatedUpdates,
		},
	}

	s.broadcast <- broadcastMessage
}

// handleCursorMove processes live cursor position updates
func (s *RealtimeService) handleCursorMove(client *Client, message *WebSocketMessage) {
	cursorX, _ := message.Data["x"].(float64)
	cursorY, _ := message.Data["y"].(float64)
	element, _ := message.Data["element"].(string)

	// Update presence in database (non-blocking)
	go s.updateCursorPosition(client.userID, client.boardID, int(cursorX), int(cursorY), element)

	// Broadcast cursor position to other clients (rate limited)
	broadcastMessage := &WebSocketMessage{
		Type:      MessageTypeCursorMove,
		BoardID:   client.boardID,
		UserID:    client.userID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_name": client.user.Name,
			"x":         int(cursorX),
			"y":         int(cursorY),
			"element":   element,
		},
	}

	s.broadcast <- broadcastMessage
}

// handlePresenceUpdate processes user presence changes
func (s *RealtimeService) handlePresenceUpdate(client *Client, message *WebSocketMessage) {
	isTyping, _ := message.Data["is_typing"].(bool)
	activeTaskID, _ := message.Data["active_task_id"].(string)

	// Update presence in database
	go s.updateTypingStatus(client.userID, client.boardID, isTyping, activeTaskID)

	// Broadcast presence update
	broadcastMessage := &WebSocketMessage{
		Type:      MessageTypeUserPresence,
		BoardID:   client.boardID,
		UserID:    client.userID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"user_name":      client.user.Name,
			"user_initials":  client.user.GetInitials(),
			"is_typing":      isTyping,
			"active_task_id": activeTaskID,
		},
	}

	s.broadcast <- broadcastMessage
}

// Database helper methods
func (s *RealtimeService) updateUserPresence(client *Client, isOnline bool) {
	boardUUID, err := uuid.Parse(client.boardID)
	if err != nil {
		log.Printf("Invalid board ID: %v", err)
		return
	}

	if isOnline {
		// Update user presence using database method
		err := s.db.UpdateUserPresence(context.Background(), client.userID, boardUUID, nil, nil, nil, false)
		if err != nil {
			log.Printf("Failed to update presence: %v", err)
		}
	} else {
		// Remove presence on disconnect
		err := s.db.RemoveUserPresence(context.Background(), client.userID, boardUUID)
		if err != nil {
			log.Printf("Failed to remove presence: %v", err)
		}
	}
}

func (s *RealtimeService) updateCursorPosition(userID uuid.UUID, boardID string, x, y int, element string) {
	boardUUID, err := uuid.Parse(boardID)
	if err != nil {
		log.Printf("Invalid board ID: %v", err)
		return
	}

	err = s.db.UpdateUserPresence(context.Background(), userID, boardUUID, &x, &y, &element, false)

	if err != nil {
		log.Printf("Failed to update cursor position: %v", err)
	}
}

func (s *RealtimeService) updateTypingStatus(userID uuid.UUID, boardID string, isTyping bool, _ string) {
	boardUUID, err := uuid.Parse(boardID)
	if err != nil {
		log.Printf("Invalid board ID: %v", err)
		return
	}

	// Use the database method with typing status
	err = s.db.UpdateUserPresence(context.Background(), userID, boardUUID, nil, nil, nil, isTyping)
	if err != nil {
		log.Printf("Failed to update typing status: %v", err)
	}
}

// broadcastPresenceUpdate notifies clients of user presence changes
func (s *RealtimeService) broadcastPresenceUpdate(boardID string, userID uuid.UUID, action string) {
	// Get current online users for this board
	onlineUsers := s.getOnlineUsers(boardID)

	message := &WebSocketMessage{
		Type:      MessageTypeUserPresence,
		BoardID:   boardID,
		UserID:    userID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"action":       action,
			"online_users": onlineUsers,
		},
	}

	s.broadcast <- message
}

// sendBoardSnapshot sends current board state to newly connected client
func (s *RealtimeService) sendBoardSnapshot(client *Client) {
	boardUUID, err := uuid.Parse(client.boardID)
	if err != nil {
		return
	}

	// Get current board state
	board, err := s.db.GetBoardWithColumns(context.Background(), boardUUID)
	if err != nil {
		log.Printf("Failed to get board snapshot: %v", err)
		return
	}

	// Send online users list
	onlineUsers := s.getOnlineUsers(client.boardID)

	snapshotMessage := &WebSocketMessage{
		Type:      "board_snapshot",
		BoardID:   client.boardID,
		UserID:    client.userID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"board":        board,
			"online_users": onlineUsers,
		},
	}

	messageBytes, err := json.Marshal(snapshotMessage)
	if err != nil {
		log.Printf("Failed to marshal snapshot: %v", err)
		return
	}

	select {
	case client.send <- messageBytes:
	default:
		log.Printf("Failed to send snapshot to client")
	}
}

// getOnlineUsers returns list of users currently connected to board
// Also includes recently offline users (within last 5 minutes) with status
func (s *RealtimeService) getOnlineUsers(boardID string) []map[string]interface{} {
	s.mu.RLock()
	clients := s.clients[boardID]
	s.mu.RUnlock()

	var users []map[string]interface{}
	seen := make(map[uuid.UUID]bool)
	now := time.Now()

	// First, add actively connected users (green dot - online)
	for client := range clients {
		if !seen[client.userID] {
			timeSinceLastSeen := now.Sub(client.lastSeen)
			status := "online"

			// If last seen > 30 seconds, mark as offline (will have gray dot)
			if timeSinceLastSeen > 30*time.Second {
				status = "offline"
			}

			users = append(users, map[string]interface{}{
				"id":         client.userID.String(),
				"name":       client.user.Name,
				"initials":   client.user.GetInitials(),
				"email":      client.user.Email,
				"avatar_url": client.user.AvatarURL,
				"last_seen":  client.lastSeen,
				"status":     status,
			})
			seen[client.userID] = true
		}
	}

	// Then, query database for recently disconnected users (within last 5 minutes)
	// These will be shown with gray dots to indicate they were recently online
	boardUUID, err := uuid.Parse(boardID)
	if err == nil {
		presences, err := s.db.GetBoardPresence(context.Background(), boardUUID)
		if err == nil {
			for _, presence := range presences {
				// Skip if already in active users list
				if seen[presence.UserID] {
					continue
				}

				// Only show users who were active in the last 5 minutes
				timeSinceActive := now.Sub(presence.LastActivity)
				if timeSinceActive <= 5*time.Minute {
					// Get user details
					user, err := s.db.GetUserByID(context.Background(), presence.UserID)
					if err == nil && user != nil {
						users = append(users, map[string]interface{}{
							"id":         user.ID.String(),
							"name":       user.Name,
							"initials":   user.GetInitials(),
							"email":      user.Email,
							"avatar_url": user.AvatarURL,
							"last_seen":  presence.LastActivity,
							"status":     "offline", // Gray dot for recently offline
						})
						seen[presence.UserID] = true
					}
				}
			}
		}
	}

	return users
}

// sendErrorToClient sends error message to specific client
func (s *RealtimeService) sendErrorToClient(client *Client, errorMsg string) {
	errorMessage := &WebSocketMessage{
		Type:      MessageTypeError,
		BoardID:   client.boardID,
		UserID:    client.userID.String(),
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"error": errorMsg,
		},
	}

	messageBytes, err := json.Marshal(errorMessage)
	if err != nil {
		return
	}

	select {
	case client.send <- messageBytes:
	default:
		// Client buffer full, will be disconnected by main loop
	}
}

// renderTaskCard renders task HTML using existing Templ components
func (s *RealtimeService) renderTaskCard(task *models.Task) (string, error) {
	// Use your existing task card component
	component := components.TaskCard(*task)

	// Render to string
	var htmlBuilder strings.Builder
	err := component.Render(context.Background(), &htmlBuilder)
	if err != nil {
		return "", fmt.Errorf("failed to render task card: %w", err)
	}

	return htmlBuilder.String(), nil
}

// cleanupStaleConnections removes inactive connections (30 second timeout)
func (s *RealtimeService) cleanupStaleConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Changed from 1 minute to 30 seconds for faster offline detection
	staleThreshold := time.Now().Add(-30 * time.Second)

	for boardID, clients := range s.clients {
		for client := range clients {
			if client.lastSeen.Before(staleThreshold) {
				log.Printf("Cleaning up stale connection for user %s in board %s (last seen: %s)",
					client.user.Name, boardID, client.lastSeen.Format(time.RFC3339))

				// Update presence to offline before removing
				s.updateUserPresence(client, false)

				// Broadcast that user went offline
				s.broadcastPresenceUpdate(boardID, client.userID, "left")

				delete(clients, client)
				close(client.send)

				if len(clients) == 0 {
					delete(s.clients, boardID)
				}
			}
		}
	}
}

// GetConnectedUsersCount returns number of users connected to a board
func (s *RealtimeService) GetConnectedUsersCount(boardID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := s.clients[boardID]
	if clients == nil {
		return 0
	}

	userSet := make(map[uuid.UUID]bool)
	for client := range clients {
		userSet[client.userID] = true
	}

	return len(userSet)
}

// BroadcastTaskUpdate sends task updates to all board clients
func (s *RealtimeService) BroadcastTaskUpdate(boardID string, task *models.Task, updateType string) {
	message := &WebSocketMessage{
		Type:      MessageTypeHTMXUpdate,
		BoardID:   boardID,
		UserID:    "system", // System-generated update
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"update_type": updateType,
			"task_id":     task.ID.String(),
		},
	}

	// For deleted tasks, don't try to render HTML - just send delete notification
	if updateType == "deleted" {
		message.Data["target"] = fmt.Sprintf("#task-%s", task.ID.String())
		message.Data["swap_strategy"] = "delete"
		log.Printf("Broadcasting delete notification for task %s", task.ID.String())
	} else {
		// For other updates, render the task card HTML
		taskHTML, err := s.renderTaskCard(task)
		if err != nil {
			log.Printf("Failed to render task for broadcast: %v", err)
			return
		}
		message.Data["target"] = fmt.Sprintf("#task-%s", task.ID.String())
		message.Data["html_content"] = taskHTML
		message.Data["swap_strategy"] = "outerHTML"
	}

	// Use non-blocking send to prevent handler from hanging
	select {
	case s.broadcast <- message:
		// Message sent successfully
	default:
		log.Printf("Warning: Broadcast channel full, skipping broadcast for task %s", task.ID.String())
	}
}

// BroadcastMemberAdded notifies all clients when a new member is added to the board
func (s *RealtimeService) BroadcastMemberAdded(boardID string, member *models.User, role string) {
	// Get updated online users list
	onlineUsers := s.getOnlineUsers(boardID)

	// Render presence indicator HTML
	presenceHTML, err := s.renderPresenceIndicator(onlineUsers)
	if err != nil {
		log.Printf("Failed to render presence indicator: %v", err)
		return
	}

	message := &WebSocketMessage{
		Type:      MessageTypeMemberAdded,
		BoardID:   boardID,
		UserID:    "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"member_id":     member.ID.String(),
			"member_name":   member.GetDisplayName(),
			"member_email":  member.DecryptedEmail,
			"member_avatar": member.AvatarURL,
			"role":          role,
			"online_users":  onlineUsers,
			"presence_html": presenceHTML,
		},
	}

	s.broadcast <- message
	log.Printf("Broadcast member added to board %s: %s", boardID, member.GetDisplayName())
}

// BroadcastMemberRemoved notifies all clients when a member is removed from the board
func (s *RealtimeService) BroadcastMemberRemoved(boardID string, memberID uuid.UUID) {
	boardUUID, err := uuid.Parse(boardID)
	if err != nil {
		log.Printf("Invalid board ID for member removed broadcast: %v", err)
		return
	}

	// Remove user presence from database
	err = s.db.RemoveUserPresence(context.Background(), memberID, boardUUID)
	if err != nil {
		log.Printf("Failed to remove user presence: %v", err)
	}

	// Get updated online users list
	onlineUsers := s.getOnlineUsers(boardID)

	// Render presence indicator HTML
	presenceHTML, err := s.renderPresenceIndicator(onlineUsers)
	if err != nil {
		log.Printf("Failed to render presence indicator: %v", err)
		return
	}

	message := &WebSocketMessage{
		Type:      MessageTypeMemberRemoved,
		BoardID:   boardID,
		UserID:    "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"member_id":     memberID.String(),
			"online_users":  onlineUsers,
			"presence_html": presenceHTML,
		},
	}

	s.broadcast <- message
	log.Printf("Broadcast member removed from board %s: %s", boardID, memberID.String())
}

// BroadcastPresenceUpdate sends updated presence indicator to all board clients
func (s *RealtimeService) BroadcastPresenceUpdate(boardID string) {
	onlineUsers := s.getOnlineUsers(boardID)

	presenceHTML, err := s.renderPresenceIndicator(onlineUsers)
	if err != nil {
		log.Printf("Failed to render presence indicator: %v", err)
		return
	}

	message := &WebSocketMessage{
		Type:      MessageTypePresenceUpdate,
		BoardID:   boardID,
		UserID:    "system",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"online_users":  onlineUsers,
			"presence_html": presenceHTML,
		},
	}

	s.broadcast <- message
}

// renderPresenceIndicator renders presence indicator HTML with online/offline status
func (s *RealtimeService) renderPresenceIndicator(onlineUsers []map[string]interface{}) (string, error) {
	// Convert to UserPresenceStatus slice with status information
	userPresences := make([]models.UserPresenceStatus, 0, len(onlineUsers))
	for _, userData := range onlineUsers {
		idStr, ok := userData["id"].(string)
		if !ok {
			continue
		}
		userID, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}

		status := "online" // default
		if statusStr, ok := userData["status"].(string); ok {
			status = statusStr
		}

		name, _ := userData["name"].(string)
		email, _ := userData["email"].(string)
		avatarURL, _ := userData["avatar_url"].(string)

		userPresences = append(userPresences, models.UserPresenceStatus{
			User: models.User{
				ID:             userID,
				Name:           name,
				Email:          email,
				AvatarURL:      avatarURL,
				DecryptedEmail: email,
			},
			Status: status,
		})
	}

	component := components.PresenceIndicatorsWithStatus(userPresences)

	var htmlBuilder strings.Builder
	err := component.Render(context.Background(), &htmlBuilder)
	if err != nil {
		return "", fmt.Errorf("failed to render presence indicator: %w", err)
	}

	return htmlBuilder.String(), nil
}
