package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sudo/internal/database"
	"sudo/internal/models"
	"sudo/internal/realtime"
	"sudo/templates/components"

	"github.com/a-h/templ"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TaskHandler struct {
	db       *database.DB
	realtime *realtime.RealtimeService
}

func NewTaskHandler(db *database.DB, rt *realtime.RealtimeService) *TaskHandler {
	return &TaskHandler{
		db:       db,
		realtime: rt,
	}
}

func (h *TaskHandler) validateUserSession(c *gin.Context) (*models.User, error) {
	userID, err := getUserFromSession(c)
	if err != nil {
		return nil, err
	}

	// Verify user exists in database
	user, err := h.db.GetUserByID(context.Background(), userID)
	if err != nil {
		// User doesn't exist - clear the invalid session
		session := sessions.Default(c)
		session.Clear()
		session.Options(sessions.Options{MaxAge: -1})
		_ = session.Save() // Ignore error when clearing invalid session
		return nil, fmt.Errorf("invalid session - user not found")
	}

	return user, nil
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	user, err := h.validateUserSession(c)
	if err != nil {
		fmt.Printf("Task creation session error: %v\n", err)
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusUnauthorized)
		return
	}

	fmt.Printf("Valid user creating task: %s (%s)\n", user.Email, user.ID.String())

	title := c.PostForm("title")
	description := c.PostForm("description")
	columnIDStr := c.PostForm("column_id")
	boardIDStr := c.PostForm("board_id")
	priority := c.PostForm("priority")
	deadlineStr := c.PostForm("deadline")
	tagsStr := c.PostForm("tags") // Optional field

	// Validate required fields
	if title == "" || columnIDStr == "" || boardIDStr == "" {
		c.String(http.StatusBadRequest, "Title, column ID, and board ID are required")
		return
	}

	if priority == "" {
		c.String(http.StatusBadRequest, "Priority is required")
		return
	}

	if deadlineStr == "" {
		c.String(http.StatusBadRequest, "Deadline is required")
		return
	}

	columnID, err := uuid.Parse(columnIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid column ID")
		return
	}

	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid board ID")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), user.ID, boardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	// Handle multiple assignees from checkbox list - VALIDATE BEFORE TASK CREATION
	assigneeIDs := c.PostFormArray("assignee_ids[]")
	if len(assigneeIDs) == 0 {
		c.String(http.StatusBadRequest, "At least one assignee is required")
		return
	}

	// Parse and validate deadline
	deadline, err := time.Parse("2006-01-02T15:04", deadlineStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid deadline format")
		return
	}

	task, err := h.db.CreateTask(context.Background(), title, description, columnID, boardID, priority)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create task: %v", err)
		return
	}
	for _, assigneeIDStr := range assigneeIDs {
		assigneeID, parseErr := uuid.Parse(assigneeIDStr)
		if parseErr != nil {
			fmt.Printf("Warning: Invalid assignee ID %s: %v\n", assigneeIDStr, parseErr)
			continue
		}

		// Check if assignee has access to the board
		hasAccess, accessErr := h.db.HasBoardAccess(context.Background(), assigneeID, boardID)
		if accessErr != nil || !hasAccess {
			fmt.Printf("Warning: Assignee %s doesn't have board access\n", assigneeIDStr)
			continue
		}

		// Add assignee
		err = h.db.AddTaskAssignee(context.Background(), task.ID, assigneeID, user.ID)
		if err != nil {
			fmt.Printf("Warning: Failed to add assignee %s: %v\n", assigneeIDStr, err)
		}
	}

	// Reload task to get assignees
	task, _ = h.db.GetTask(context.Background(), task.ID)

	// Log activity
	err = h.db.LogActivity(context.Background(), user.ID, boardID, &task.ID, "task_create",
		fmt.Sprintf("Created task: %s", task.Title), map[string]interface{}{
			"task_title":      task.Title,
			"column_id":       columnID.String(),
			"priority":        priority,
			"assignees_count": len(assigneeIDs),
		})
	if err != nil {
		fmt.Printf("Failed to log task creation activity: %v\n", err)
	}

	// Update task with deadline and tags (already validated)
	updates := map[string]interface{}{
		"deadline": deadline,
	}

	// Parse tags from comma-separated string
	if tagsStr != "" {
		tags := []string{}
		for _, tag := range strings.Split(tagsStr, ",") {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag != "" {
				tags = append(tags, trimmedTag)
			}
		}
		if len(tags) > 0 {
			updates["tags"] = tags
		}
	}

	err = h.db.UpdateTask(context.Background(), task.ID, updates)
	if err != nil {
		fmt.Printf("Warning: Failed to update task with deadline and tags: %v\n", err)
	}

	// Reload task to get updated fields
	task, _ = h.db.GetTask(context.Background(), task.ID)

	// Broadcast real-time update
	if h.realtime != nil {
		h.realtime.BroadcastTaskUpdate(boardID.String(), task, "created")
	}

	component := components.TaskCard(*task)
	handler := templ.Handler(component)
	handler.ServeHTTP(c.Writer, c.Request)
}

func (h *TaskHandler) MoveTask(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.PostForm("task_id")
	columnIDStr := c.PostForm("column_id")
	positionStr := c.PostForm("position")

	if taskIDStr == "" || columnIDStr == "" || positionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	columnID, err := uuid.Parse(columnIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid column ID"})
		return
	}

	position, err := strconv.Atoi(positionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid position"})
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check board access"})
		return
	}

	if !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	err = h.db.MoveTask(context.Background(), taskID, columnID, position)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move task"})
		return
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), userID, task.BoardID, &taskID, "task_move",
		fmt.Sprintf("Moved task: %s to position %d", task.Title, position), map[string]interface{}{
			"from_column_id": task.ColumnID.String(),
			"to_column_id":   columnID.String(),
			"new_position":   position,
		})
	if err != nil {
		fmt.Printf("Failed to log task move activity: %v\n", err)
	}

	// Broadcast real-time update
	if h.realtime != nil {
		updatedTask, _ := h.db.GetTask(context.Background(), taskID)
		if updatedTask != nil {
			h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "moved")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"task_id":   taskID,
		"column_id": columnID,
		"position":  position,
		"message":   "Task moved successfully",
	})
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		fmt.Printf("UpdateTask: Session error: %v\n", err)
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	taskIDStr := c.Param("id")
	fmt.Printf("UpdateTask: Task ID: %s\n", taskIDStr)
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		fmt.Printf("UpdateTask: Invalid task ID: %v\n", err)
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		fmt.Printf("UpdateTask: Failed to get task: %v\n", err)
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		fmt.Printf("UpdateTask: Failed to check board access: %v\n", err)
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		fmt.Printf("UpdateTask: User %s has no access to board %s\n", userID.String(), task.BoardID.String())
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	updates := make(map[string]interface{})

	if title := c.PostForm("title"); title != "" {
		updates["title"] = title
	}

	if description := c.PostForm("description"); description != "" {
		updates["description"] = description
	}

	if priority := c.PostForm("priority"); priority != "" {
		updates["priority"] = priority
	}

	if deadline := c.PostForm("deadline"); deadline != "" {
		if deadlineTime, parseErr := time.Parse("2006-01-02T15:04", deadline); parseErr == nil {
			updates["deadline"] = deadlineTime
		}
	}

	// Handle completion status
	if completed := c.PostForm("completed"); completed != "" {
		isCompleted := completed == "true"
		updates["completed"] = isCompleted
		// Ensure completed_at is properly set/cleared to maintain DB constraint
		if isCompleted {
			// If marking as complete and no completed_at is set, set it now
			updates["completed_at"] = time.Now()
		} else {
			// If marking as incomplete, clear completed_at
			updates["completed_at"] = nil
		}
	}

	// Handle assignee
	if assigneeStr := c.PostForm("assignee_id"); assigneeStr != "" {
		if assigneeStr == "unassign" {
			updates["assigned_to"] = nil
		} else if assigneeID, parseErr := uuid.Parse(assigneeStr); parseErr == nil {
			updates["assigned_to"] = assigneeID
		}
	}

	fmt.Printf("UpdateTask: Updates to apply: %+v\n", updates)

	// Update task in database
	err = h.db.UpdateTask(context.Background(), taskID, updates)
	if err != nil {
		fmt.Printf("UpdateTask: Database update error: %v\n", err)
		c.String(http.StatusInternalServerError, "Failed to update task: %v", err)
		return
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), userID, task.BoardID, &taskID, "task_update",
		fmt.Sprintf("Updated task: %s", task.Title), updates)
	if err != nil {
		fmt.Printf("Failed to log task update activity: %v\n", err)
	}

	// Broadcast real-time update
	if h.realtime != nil {
		updatedTask, _ := h.db.GetTask(context.Background(), taskID)
		if updatedTask != nil {
			h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "updated")
		}
	}

	fmt.Printf("UpdateTask: Successfully updated task %s\n", taskID.String())
	c.Status(http.StatusOK)
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	user, err := h.validateUserSession(c)
	if err != nil {
		fmt.Printf("Task deletion session error: %v\n", err)
		c.Status(http.StatusUnauthorized)
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), user.ID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	fmt.Printf("Deleting task: %s by user %s\n", taskID.String(), user.Email)

	var deletedNestedBoardID *uuid.UUID

	// IMPORTANT: Store nested board ID BEFORE deleting the task
	// We must delete the TASK first, then the nested board
	// If we delete the board first, PostgreSQL CASCADE will delete columns and tasks in that board
	if task.HasNestedBoard() {
		deletedNestedBoardID = task.NestedBoardID
		fmt.Printf("Task %s has nested board %s, will delete board after task\n", taskID.String(), task.NestedBoardID.String())
	}

	// Delete the task FIRST
	err = h.db.DeleteTask(context.Background(), taskID)
	if err != nil {
		fmt.Printf("Failed to delete task: %v\n", err)
		c.String(http.StatusInternalServerError, "Failed to delete task: %v", err)
		return
	}
	fmt.Printf("Successfully deleted task %s\n", taskID.String())

	// Now delete the nested board (if it exists)
	// This prevents CASCADE deletion conflicts
	if deletedNestedBoardID != nil {
		err = h.db.DeleteBoard(context.Background(), *deletedNestedBoardID)
		if err != nil {
			fmt.Printf("Failed to delete nested board %s: %v\n", deletedNestedBoardID.String(), err)
			// Don't fail the whole operation since the task is already deleted
			// Just log the error
		} else {
			fmt.Printf("Successfully deleted nested board %s\n", deletedNestedBoardID.String())
		}
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), user.ID, task.BoardID, &taskID, "task_delete",
		fmt.Sprintf("Deleted task: %s", task.Title), map[string]interface{}{
			"task_title":       task.Title,
			"had_nested_board": deletedNestedBoardID != nil,
		})
	if err != nil {
		fmt.Printf("Failed to log task deletion activity: %v\n", err)
	}

	// Broadcast real-time update
	if h.realtime != nil {
		h.realtime.BroadcastTaskUpdate(task.BoardID.String(), task, "deleted")
	}

	fmt.Printf("Successfully deleted task %s\n", taskID.String())

	// Send HTMX trigger with nested board info if applicable
	if deletedNestedBoardID != nil {
		// Send both taskDeleted and nested board deletion events
		triggerString := fmt.Sprintf("taskDeleted, nestedBoardDeleted-%s", deletedNestedBoardID.String())
		c.Header("HX-Trigger", triggerString)
		fmt.Printf("Sending HTMX trigger: %s\n", triggerString)
	} else {
		c.Header("HX-Trigger", "taskDeleted")
		fmt.Printf("Sending HTMX trigger: taskDeleted\n")
	}
	c.Status(http.StatusOK)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// DEBUG: Log task assignees
	fmt.Printf("DEBUG GetTask: Task %s has %d assignees:\n", task.ID.String(), len(task.Assignees))
	for i, assignee := range task.Assignees {
		userName := "nil"
		if assignee.User != nil {
			userName = assignee.User.GetDisplayName()
		}
		fmt.Printf("  [%d] UserID: %s, User: %s, Completed: %v\n", i, assignee.UserID.String(), userName, assignee.Completed)
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	// Get board members for assignment options
	members, err := h.db.GetBoardMembers(context.Background(), task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get board members: %v", err)
		return
	}

	// DEBUG: Log board members
	fmt.Printf("DEBUG GetTask: Board has %d members:\n", len(members))
	for i, member := range members {
		userName := "nil"
		if member.User != nil {
			userName = member.User.GetDisplayName()
		}
		fmt.Printf("  [%d] UserID: %s, User: %s, Role: %s\n", i, member.UserID.String(), userName, member.Role)
	}

	component := components.TaskDetailsModal(*task, members)
	handler := templ.Handler(component)
	handler.ServeHTTP(c.Writer, c.Request)
}

func (h *TaskHandler) AssignTask(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	taskIDStr := c.PostForm("task_id")
	assigneeIDStr := c.PostForm("assignee_id")

	if taskIDStr == "" || assigneeIDStr == "" {
		c.String(http.StatusBadRequest, "Task ID and assignee ID are required")
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	assigneeID, err := uuid.Parse(assigneeIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid assignee ID")
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	// Check if assignee has access to the board
	assigneeAccess, err := h.db.HasBoardAccess(context.Background(), assigneeID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check assignee access: %v", err)
		return
	}

	if !assigneeAccess {
		c.String(http.StatusBadRequest, "Assignee doesn't have access to this board")
		return
	}

	err = h.db.AssignTask(context.Background(), taskID, assigneeID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to assign task: %v", err)
		return
	}

	// Get assignee name for logging
	assignee, _ := h.db.GetUserByID(context.Background(), assigneeID)
	assigneeName := "Unknown User"
	if assignee != nil {
		assigneeName = assignee.GetDisplayName()
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), userID, task.BoardID, &taskID, "task_update",
		fmt.Sprintf("Assigned task: %s to %s", task.Title, assigneeName), map[string]interface{}{
			"action":      "assigned",
			"assigned_to": assigneeID.String(),
		})
	if err != nil {
		fmt.Printf("Failed to log task assignment activity: %v\n", err)
	}

	// Broadcast real-time update
	if h.realtime != nil {
		updatedTask, _ := h.db.GetTask(context.Background(), taskID)
		if updatedTask != nil {
			h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "assigned")
		}
	}

	c.Status(http.StatusOK)
}

func (h *TaskHandler) UnassignTask(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	taskIDStr := c.PostForm("task_id")
	if taskIDStr == "" {
		c.String(http.StatusBadRequest, "Task ID is required")
		return
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	err = h.db.UnassignTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to unassign task: %v", err)
		return
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), userID, task.BoardID, &taskID, "task_update",
		fmt.Sprintf("Unassigned task: %s", task.Title), map[string]interface{}{
			"action": "unassigned",
		})
	if err != nil {
		fmt.Printf("Failed to log task unassignment activity: %v\n", err)
	}

	// Broadcast real-time update
	if h.realtime != nil {
		updatedTask, _ := h.db.GetTask(context.Background(), taskID)
		if updatedTask != nil {
			h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "unassigned")
		}
	}

	c.Status(http.StatusOK)
}

func (h *TaskHandler) CreateNestedBoard(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	taskIDStr := c.Param("id")

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Get task details
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	// Create nested board using task title and description
	boardTitle := fmt.Sprintf("%s - Sub-board", task.Title)
	board, err := h.db.CreateBoard(context.Background(), boardTitle, task.Description, userID, &task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to create nested board: %v", err)
		return
	}

	// Link the task to the newly created nested board
	updates := map[string]interface{}{
		"nested_board_id": board.ID,
	}

	err = h.db.UpdateTask(context.Background(), taskID, updates)
	if err != nil {
		// If we can't link the task, clean up by deleting the board
		_ = h.db.DeleteBoard(context.Background(), board.ID) // Ignore cleanup error
		c.String(http.StatusInternalServerError, "Failed to link task to nested board: %v", err)
		return
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), userID, task.BoardID, &taskID, "task_update",
		fmt.Sprintf("Created nested board for task: %s", task.Title), map[string]interface{}{
			"nested_board_id":    board.ID.String(),
			"nested_board_title": board.Title,
		})
	if err != nil {
		fmt.Printf("Failed to log nested board creation activity: %v\n", err)
	}

	fmt.Printf("Created nested board %s for task %s\n", board.ID.String(), taskID.String())

	// Redirect to the new board
	c.Header("HX-Redirect", fmt.Sprintf("/boards/%s", board.ID.String()))
	c.Status(http.StatusOK)
}

func (h *TaskHandler) CompleteTask(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	updates := map[string]interface{}{
		"completed":    true,
		"completed_at": time.Now(),
	}

	err = h.db.UpdateTask(context.Background(), taskID, updates)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to complete task: %v", err)
		return
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), userID, task.BoardID, &taskID, "task_complete",
		fmt.Sprintf("Completed task: %s", task.Title), map[string]interface{}{
			"completed_at": time.Now(),
		})
	if err != nil {
		fmt.Printf("Failed to log task completion activity: %v\n", err)
	}

	// Broadcast real-time update
	if h.realtime != nil {
		updatedTask, _ := h.db.GetTask(context.Background(), taskID)
		if updatedTask != nil {
			h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "completed")
		}
	}

	c.Status(http.StatusOK)
}

func (h *TaskHandler) ReopenTask(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
		return
	}

	if !hasAccess {
		c.String(http.StatusForbidden, "You don't have access to this board")
		return
	}

	updates := map[string]interface{}{
		"completed":    false,
		"completed_at": nil,
	}

	err = h.db.UpdateTask(context.Background(), taskID, updates)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to reopen task: %v", err)
		return
	}

	// Log activity
	err = h.db.LogActivity(context.Background(), userID, task.BoardID, &taskID, "task_update",
		fmt.Sprintf("Reopened task: %s", task.Title), map[string]interface{}{
			"action": "reopened",
		})
	if err != nil {
		fmt.Printf("Failed to log task reopen activity: %v\n", err)
	}

	// Broadcast real-time update
	if h.realtime != nil {
		updatedTask, _ := h.db.GetTask(context.Background(), taskID)
		if updatedTask != nil {
			h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "reopened")
		}
	}

	c.Status(http.StatusOK)
}

// Add Task Assignee - for multiple assignees support
func (h *TaskHandler) AddTaskAssignee(c *gin.Context) {
	user, err := h.validateUserSession(c)
	if err != nil {
		fmt.Printf("ERROR AddTaskAssignee: Session validation failed: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		fmt.Printf("ERROR AddTaskAssignee: Invalid task ID %s: %v\n", taskIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	assigneeIDStr := c.PostForm("user_id")
	fmt.Printf("DEBUG AddTaskAssignee: Received user_id: %s\n", assigneeIDStr)

	assigneeID, err := uuid.Parse(assigneeIDStr)
	if err != nil {
		fmt.Printf("ERROR AddTaskAssignee: Invalid user ID %s: %v\n", assigneeIDStr, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	fmt.Printf("DEBUG AddTaskAssignee: Task=%s, Assignee=%s, By=%s\n", taskID.String(), assigneeID.String(), user.ID.String())

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		fmt.Printf("ERROR AddTaskAssignee: Failed to get task %s: %v\n", taskID.String(), err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), user.ID, task.BoardID)
	if err != nil {
		fmt.Printf("ERROR AddTaskAssignee: Failed to check user board access: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check access"})
		return
	}
	if !hasAccess {
		fmt.Printf("ERROR AddTaskAssignee: User %s doesn't have access to board %s\n", user.ID.String(), task.BoardID.String())
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if assignee has board access
	assigneeAccess, err := h.db.HasBoardAccess(context.Background(), assigneeID, task.BoardID)
	if err != nil {
		fmt.Printf("ERROR AddTaskAssignee: Failed to check assignee board access: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check assignee access"})
		return
	}
	if !assigneeAccess {
		fmt.Printf("ERROR AddTaskAssignee: Assignee %s doesn't have access to board %s\n", assigneeID.String(), task.BoardID.String())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Assignee doesn't have board access"})
		return
	}

	// Add assignee
	err = h.db.AddTaskAssignee(context.Background(), taskID, assigneeID, user.ID)
	if err != nil {
		fmt.Printf("ERROR AddTaskAssignee: Failed to add assignee to database: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add assignee"})
		return
	}

	fmt.Printf("SUCCESS AddTaskAssignee: Added assignee %s to task %s\n", assigneeID.String(), taskID.String())

	// Get updated task with assignees
	updatedTask, _ := h.db.GetTask(context.Background(), taskID)

	// Broadcast real-time update
	if h.realtime != nil && updatedTask != nil {
		h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "assignee_added")
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Remove Task Assignee
func (h *TaskHandler) RemoveTaskAssignee(c *gin.Context) {
	user, err := h.validateUserSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	assigneeIDStr := c.Param("userId")
	assigneeID, err := uuid.Parse(assigneeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), user.ID, task.BoardID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Remove assignee
	err = h.db.RemoveTaskAssignee(context.Background(), taskID, assigneeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove assignee"})
		return
	}

	// Get updated task with assignees
	updatedTask, _ := h.db.GetTask(context.Background(), taskID)

	// Broadcast real-time update
	if h.realtime != nil && updatedTask != nil {
		h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "assignee_removed")
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Toggle Assignee Completion - for individual completion tracking
func (h *TaskHandler) ToggleAssigneeCompletion(c *gin.Context) {
	user, err := h.validateUserSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	completedStr := c.PostForm("completed")
	completed := completedStr == "true"

	// Get task to check board access
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), user.ID, task.BoardID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update assignee completion status
	err = h.db.UpdateTaskAssigneeCompletion(context.Background(), taskID, user.ID, completed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update completion status"})
		return
	}

	// Get updated task with assignees
	updatedTask, _ := h.db.GetTask(context.Background(), taskID)

	// Broadcast real-time update
	if h.realtime != nil && updatedTask != nil {
		h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "assignee_completion_updated")
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "completed": completed})
}

// GetTaskCard returns the HTML for a single task card (for dynamic updates)
func (h *TaskHandler) GetTaskCard(c *gin.Context) {
	user, err := h.validateUserSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Get task with all related data
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Check if user has access to this board
	hasAccess, err := h.db.HasBoardAccess(context.Background(), user.ID, task.BoardID)
	if err != nil || !hasAccess {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Render the task card component
	component := components.TaskCard(*task)
	c.Header("Content-Type", "text/html")
	templ.Handler(component).ServeHTTP(c.Writer, c.Request)
}

// Enhanced task update with real-time broadcasting
func (h *TaskHandler) UpdateTaskWithBroadcast(c *gin.Context) {
	userID, err := getUserFromSession(c)
	if err != nil {
		c.Header("HX-Redirect", "/")
		c.Status(http.StatusUnauthorized)
		return
	}

	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid task ID")
		return
	}

	// Get current task for board ID
	task, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusNotFound, "Task not found")
		return
	}

	// Check board access
	hasAccess, err := h.db.HasBoardAccess(context.Background(), userID, task.BoardID)
	if err != nil || !hasAccess {
		c.String(http.StatusForbidden, "Access denied")
		return
	}

	// Prepare updates
	updates := make(map[string]interface{})

	if title := c.PostForm("title"); title != "" {
		updates["title"] = title
	}
	if description := c.PostForm("description"); description != "" {
		updates["description"] = description
	}
	if priority := c.PostForm("priority"); priority != "" {
		updates["priority"] = priority
	}
	if assignedTo := c.PostForm("assigned_to"); assignedTo != "" {
		if assigneeID, parseErr := uuid.Parse(assignedTo); parseErr == nil {
			updates["assigned_to"] = assigneeID
		}
	}

	if len(updates) == 0 {
		c.String(http.StatusBadRequest, "No valid updates provided")
		return
	}

	// Update task
	err = h.db.UpdateTask(context.Background(), taskID, updates)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to update task: %v", err)
		return
	}

	// Get updated task
	updatedTask, err := h.db.GetTask(context.Background(), taskID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get updated task")
		return
	}

	// Broadcast real-time update
	if h.realtime != nil {
		h.realtime.BroadcastTaskUpdate(task.BoardID.String(), updatedTask, "updated")
	}

	// Return updated task component
	taskComponent := components.TaskCard(*updatedTask)

	c.Header("Content-Type", "text/html")
	if err := taskComponent.Render(c.Request.Context(), c.Writer); err != nil {
		c.String(http.StatusInternalServerError, "Failed to render task")
	}
}
