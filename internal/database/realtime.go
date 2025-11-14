package database

import (
	"context"
	"fmt"
	"time"

	"sudo/internal/models"

	"github.com/google/uuid"
	"github.com/supabase-community/postgrest-go"
)

// Real-time specific database operations
func (db *DB) CreateRealtimeSession(ctx context.Context, userID uuid.UUID, boardID uuid.UUID, connectionID string, userAgent string) error {
	sessionData := map[string]interface{}{
		"user_id":       userID.String(),
		"board_id":      boardID.String(),
		"connection_id": connectionID,
		"user_agent":    userAgent,
		"last_ping":     time.Now(),
	}

	_, err := db.client.From("realtime_sessions").Insert(sessionData, false, "", "", "").ExecuteTo(nil)
	return err
}

func (db *DB) UpdateSessionPing(ctx context.Context, connectionID string) error {
	updates := map[string]interface{}{
		"last_ping": time.Now(),
	}

	_, err := db.client.From("realtime_sessions").
		Update(updates, "", "").
		Eq("connection_id", connectionID).
		ExecuteTo(nil)

	return err
}

func (db *DB) RemoveRealtimeSession(ctx context.Context, connectionID string) error {
	_, err := db.client.From("realtime_sessions").
		Delete("", "").
		Eq("connection_id", connectionID).
		ExecuteTo(nil)

	return err
}

func (db *DB) LogActivity(ctx context.Context, userID, boardID uuid.UUID, taskID *uuid.UUID, action, description string, metadata map[string]interface{}) error {
	activityData := map[string]interface{}{
		"user_id":     userID.String(),
		"board_id":    boardID.String(),
		"action":      action,
		"description": description,
		"metadata":    metadata,
	}

	if taskID != nil {
		activityData["task_id"] = taskID.String()
	}

	_, err := db.client.From("activity_log").Insert(activityData, false, "", "", "").ExecuteTo(nil)
	return err
}

// Get recent board activities for real-time feed
func (db *DB) GetRecentBoardActivities(ctx context.Context, boardID uuid.UUID, limit int) ([]models.Activity, error) {
	var activities []models.Activity
	_, err := db.client.From("activity_log").
		Select("*, users!inner(name, email)", "", false).
		Eq("board_id", boardID.String()).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(limit, "").
		ExecuteTo(&activities)

	return activities, err
}

// Enhanced task operations with real-time support
func (db *DB) MoveTaskWithOptimisticLock(ctx context.Context, taskID, columnID uuid.UUID, position, expectedVersion int) (*models.Task, error) {
	// For now, use regular task move since we don't have the secure function
	// This would need to be implemented as a database function first
	err := db.MoveTask(ctx, taskID, columnID, position)
	if err != nil {
		return nil, fmt.Errorf("failed to move task: %w", err)
	}

	// Return the updated task
	return db.GetTask(ctx, taskID)
}

// User presence operations
func (db *DB) UpdateUserPresence(ctx context.Context, userID, boardID uuid.UUID, cursorX, cursorY *int, focusedElement *string, isTyping bool) error {
	presenceData := map[string]interface{}{
		"user_id":       userID.String(),
		"board_id":      boardID.String(),
		"is_typing":     isTyping,
		"last_activity": time.Now(),
	}

	if cursorX != nil {
		presenceData["cursor_x"] = *cursorX
	}
	if cursorY != nil {
		presenceData["cursor_y"] = *cursorY
	}
	if focusedElement != nil {
		presenceData["focused_element"] = *focusedElement
	}

	_, err := db.client.From("user_presence").
		Upsert(presenceData, "", "", "").ExecuteTo(nil)

	return err
}

func (db *DB) RemoveUserPresence(ctx context.Context, userID, boardID uuid.UUID) error {
	_, err := db.client.From("user_presence").
		Delete("", "").
		Eq("user_id", userID.String()).
		Eq("board_id", boardID.String()).
		ExecuteTo(nil)

	return err
}

func (db *DB) GetBoardPresence(ctx context.Context, boardID uuid.UUID) ([]models.UserPresence, error) {
	var presence []models.UserPresence
	_, err := db.client.From("user_presence").
		Select("*, users!inner(name, email)", "", false).
		Eq("board_id", boardID.String()).
		ExecuteTo(&presence)

	return presence, err
}

func (db *DB) CleanupStalePresence(ctx context.Context, olderThan time.Time) error {
	_, err := db.client.From("user_presence").
		Delete("", "").
		Lt("last_activity", olderThan.UTC().Format(time.RFC3339)).
		ExecuteTo(nil)

	return err
}
