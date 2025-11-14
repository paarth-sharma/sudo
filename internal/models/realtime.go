package models

import (
	"github.com/google/uuid"
	"time"
)

type RealtimeSession struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	UserID         uuid.UUID              `json:"user_id" db:"user_id"`
	BoardID        uuid.UUID              `json:"board_id" db:"board_id"`
	ConnectionID   string                 `json:"connection_id" db:"connection_id"`
	SocketMetadata map[string]interface{} `json:"socket_metadata" db:"socket_metadata"`
	LastPing       time.Time              `json:"last_ping" db:"last_ping"`
	UserAgent      string                 `json:"user_agent" db:"user_agent"`
	IPAddress      string                 `json:"ip_address" db:"ip_address"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

type UserPresence struct {
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	BoardID         uuid.UUID  `json:"board_id" db:"board_id"`
	CursorX         *int       `json:"cursor_x" db:"cursor_x"`
	CursorY         *int       `json:"cursor_y" db:"cursor_y"`
	FocusedElement  *string    `json:"focused_element" db:"focused_element"`
	ActiveTaskID    *uuid.UUID `json:"active_task_id" db:"active_task_id"`
	IsTyping        bool       `json:"is_typing" db:"is_typing"`
	TypingInElement *string    `json:"typing_in_element" db:"typing_in_element"`
	LastActivity    time.Time  `json:"last_activity" db:"last_activity"`

	// Relationships
	User *User `json:"user,omitempty"`
	Task *Task `json:"task,omitempty"`
}

// UserPresenceStatus represents a user with their online status for UI display
type UserPresenceStatus struct {
	User   User   `json:"user"`
	Status string `json:"status"` // "online", "offline", "away"
}

// WebSocket event constants
const (
	RealtimeEventTaskMove     = "task_move"
	RealtimeEventTaskCreate   = "task_create"
	RealtimeEventTaskUpdate   = "task_update"
	RealtimeEventTaskDelete   = "task_delete"
	RealtimeEventUserPresence = "user_presence"
	RealtimeEventCursorMove   = "cursor_move"
	RealtimeEventTyping       = "typing"
	RealtimeEventError        = "error"
)
