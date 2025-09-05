package handlers

import (
    "context"
    "net/http"
    "strconv"
    "time"
    "fmt"
   
    "sudo/internal/database"
    "sudo/internal/models"
    "sudo/internal/realtime"
    "sudo/templates/components"
   
    "github.com/gin-gonic/gin"
    "github.com/gin-contrib/sessions"
    "github.com/google/uuid"
    "github.com/a-h/templ"
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
        session.Save()
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
   
    if title == "" || columnIDStr == "" || boardIDStr == "" {
        c.String(http.StatusBadRequest, "Title, column ID, and board ID are required")
        return
    }
    
    if priority == "" {
        priority = "Medium"
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
   
    task, err := h.db.CreateTask(context.Background(), title, description, columnID, boardID, priority)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to create task: %v", err)
        return
    }
    
    // Handle deadline if provided
    if deadlineStr != "" {
        if deadline, err := time.Parse("2006-01-02T15:04", deadlineStr); err == nil {
            updates := map[string]interface{}{
                "deadline": deadline,
            }
            h.db.UpdateTask(context.Background(), task.ID, updates)
            task.Deadline = &deadline
        }
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
   
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "task_id": taskID,
        "column_id": columnID,
        "position": position,
        "message": "Task moved successfully",
    })
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
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
        if deadlineTime, err := time.Parse("2006-01-02T15:04", deadline); err == nil {
            updates["deadline"] = deadlineTime
        }
    }
    
    // Handle completion status
    if completed := c.PostForm("completed"); completed != "" {
        updates["completed"] = completed == "true"
    }
    
    // Handle assignee
    if assigneeStr := c.PostForm("assignee_id"); assigneeStr != "" {
        if assigneeStr == "unassign" {
            updates["assignee_id"] = nil
        } else if assigneeID, err := uuid.Parse(assigneeStr); err == nil {
            updates["assignee_id"] = assigneeID
        }
    }
   
    // Update task in database
    err = h.db.UpdateTask(context.Background(), taskID, updates)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to update task: %v", err)
        return
    }
   
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
    err = h.db.DeleteTask(context.Background(), taskID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to delete task: %v", err)
        return
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
    users, err := h.db.GetBoardMembers(context.Background(), task.BoardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to get board members: %v", err)
        return
    }
    
    // Convert []User to []BoardMember for the template
    var members []models.BoardMember
    for _, user := range users {
        member := models.BoardMember{
            UserID: user.ID,
            User:   &user,
        }
        members = append(members, member)
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
        "completed": true,
        "completed_at": time.Now(),
    }
    
    err = h.db.UpdateTask(context.Background(), taskID, updates)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to complete task: %v", err)
        return
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
        "completed": false,
        "completed_at": nil,
    }
    
    err = h.db.UpdateTask(context.Background(), taskID, updates)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to reopen task: %v", err)
        return
    }
    
    c.Status(http.StatusOK)
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
        if assigneeID, err := uuid.Parse(assignedTo); err == nil {
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
    taskComponent.Render(c.Request.Context(), c.Writer)
}