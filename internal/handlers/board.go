package handlers

import (
    "context"
    "net/http"
    "fmt"
    "strings"
    
    "sudo/internal/database"
    "sudo/internal/email"
    "sudo/internal/models"
    "sudo/templates/pages"
    "sudo/templates/components"
    
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/gin-contrib/sessions"
    "github.com/a-h/templ"
)

type BoardHandler struct {
    db           *database.DB
    emailService *email.EmailService
}

func NewBoardHandler(db *database.DB) *BoardHandler {
    return &BoardHandler{
        db:           db,
        emailService: email.NewEmailService(),
    }
}

func (h *BoardHandler) Dashboard(c *gin.Context) {
    user, err := h.validateUserSession(c)
    if err != nil {
        fmt.Printf("Dashboard session error: %v\n", err)
        c.Redirect(http.StatusSeeOther, "/")
        return
    }
    
    fmt.Printf("Loading dashboard for user: %s (%s)\n", user.Email, user.ID.String())
    
    boards, err := h.db.GetUserBoards(context.Background(), user.ID)
    if err != nil {
        fmt.Printf("Failed to load boards: %v\n", err)
        c.String(http.StatusInternalServerError, "Failed to load boards: %v", err)
        return
    }
    
    fmt.Printf("Found %d boards for user\n", len(boards))
    
    // Fetch full board data with columns and tasks for metrics calculation
    var mainBoards []models.Board
    var nestedBoards []models.Board
    
    for _, board := range boards {
        // Get full board data with columns and tasks
        fullBoard, err := h.db.GetBoardWithColumns(context.Background(), board.ID)
        if err != nil {
            fmt.Printf("Warning: Failed to get full data for board %s: %v\n", board.ID.String(), err)
            // Use basic board data if full data fetch fails
            if board.ParentBoardID == nil {
                mainBoards = append(mainBoards, board)
            } else {
                nestedBoards = append(nestedBoards, board)
            }
            continue
        }
        
        // Get board members for metrics calculation
        members, err := h.db.GetBoardMembers(context.Background(), board.ID)
        if err != nil {
            fmt.Printf("Warning: Failed to get members for board %s: %v\n", board.ID.String(), err)
        } else {
            fullBoard.Members = members
        }
        
        // Categorize the full board data
        if fullBoard.ParentBoardID == nil {
            mainBoards = append(mainBoards, *fullBoard)
        } else {
            nestedBoards = append(nestedBoards, *fullBoard)
        }
    }
    
    fmt.Printf("Separated into %d main boards and %d nested boards\n", len(mainBoards), len(nestedBoards))
    
    component := pages.DashboardWithNested(mainBoards, nestedBoards)
    handler := templ.Handler(component)
    handler.ServeHTTP(c.Writer, c.Request)
}

func (h *BoardHandler) CreateBoard(c *gin.Context) {
    fmt.Printf("CreateBoard handler called\n")
    
    user, err := h.validateUserSession(c)
    if err != nil {
        fmt.Printf("Session error: %v\n", err)
        c.Header("HX-Redirect", "/")
        c.Status(http.StatusUnauthorized)
        return
    }
    
    fmt.Printf("Valid user session: %s (%s)\n", user.Email, user.ID.String())
    
    title := c.PostForm("title")
    description := c.PostForm("description")
    parentBoardIDStr := c.PostForm("parent_board_id")
    
    fmt.Printf("Form data - Title: %s, Description: %s\n", title, description)
    
    if title == "" {
        fmt.Printf("No title provided\n")
        c.String(http.StatusBadRequest, "Board title is required")
        return
    }
    
    var parentBoardID *uuid.UUID
    if parentBoardIDStr != "" {
        id, err := uuid.Parse(parentBoardIDStr)
        if err == nil {
            parentBoardID = &id
        }
    }
    
    board, err := h.db.CreateBoard(context.Background(), title, description, user.ID, parentBoardID)
    if err != nil {
        fmt.Printf("Database error: %v\n", err)
        c.String(http.StatusInternalServerError, "Failed to create board: %v", err)
        return
    }
    
    fmt.Printf("Board created successfully: %s (ID: %s)\n", board.Title, board.ID.String())
    
    component := pages.BoardCard(*board)
    handler := templ.Handler(component)
    handler.ServeHTTP(c.Writer, c.Request)
}

func (h *BoardHandler) ViewBoard(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusSeeOther, "/")
        return
    }
    
    boardIDStr := c.Param("id")
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    board, err := h.db.GetBoardWithColumns(context.Background(), boardID)
    if err != nil {
        c.String(http.StatusNotFound, "Board not found: %v", err)
        return
    }
    
    // Check if user has access to this board
    hasAccess, err := h.checkBoardAccess(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
        return
    }
    
    if !hasAccess {
        c.String(http.StatusForbidden, "You don't have access to this board")
        return
    }
    
    // Get parent board if this is a nested board
    var parentBoard *models.Board
    if board.ParentBoardID != nil {
        parentBoard, err = h.db.GetBoardWithColumns(context.Background(), *board.ParentBoardID)
        if err != nil {
            fmt.Printf("Warning: Failed to get parent board: %v\n", err)
            // Continue without parent board info
        }
    }
    
    // Get nested boards for this board
    nestedBoards, err := h.db.GetNestedBoards(context.Background(), boardID)
    if err != nil {
        fmt.Printf("Warning: Failed to get nested boards: %v\n", err)
        // Continue with empty nested boards list
        nestedBoards = []models.Board{}
    }
    
    component := pages.BoardWithNested(*board, parentBoard, nestedBoards)
    handler := templ.Handler(component)
    handler.ServeHTTP(c.Writer, c.Request)
}

func (h *BoardHandler) CreateColumn(c *gin.Context) {
    user, err := h.validateUserSession(c)
    if err != nil {
        fmt.Printf("Column creation session error: %v\n", err)
        c.Header("HX-Redirect", "/")
        c.Status(http.StatusUnauthorized)
        return
    }
    
    fmt.Printf("Valid user creating column: %s (%s)\n", user.Email, user.ID.String())
    
    boardIDStr := c.Param("id")
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    title := c.PostForm("title")
    if title == "" {
        c.String(http.StatusBadRequest, "Column title is required")
        return
    }
    
    // Check if user has access to this board
    hasAccess, err := h.checkBoardAccess(user.ID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
        return
    }
    
    if !hasAccess {
        c.String(http.StatusForbidden, "You don't have access to this board")
        return
    }
    
    // Get position (add to end)
    columns, err := h.db.GetBoardColumns(context.Background(), boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to get columns: %v", err)
        return
    }
    
    position := len(columns)
    
    column, err := h.db.CreateColumn(context.Background(), boardID, title, position)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to create column: %v", err)
        return
    }
    
    component := components.Column(*column, boardIDStr)
    handler := templ.Handler(component)
    handler.ServeHTTP(c.Writer, c.Request)
}

func (h *BoardHandler) InviteMember(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    email := c.PostForm("email")
    role := c.PostForm("role")
    boardIDStr := c.PostForm("board_id")
    
    if email == "" || role == "" || boardIDStr == "" {
        c.String(http.StatusBadRequest, "Email, role, and board ID are required")
        return
    }
    
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    // Check if user owns this board
    isOwner, err := h.checkBoardOwnership(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board ownership: %v", err)
        return
    }
    
    if !isOwner {
        c.String(http.StatusForbidden, "Only board owners can invite members")
        return
    }
    
    // Get board details for email
    board, err := h.db.GetBoardWithColumns(context.Background(), boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to get board details: %v", err)
        return
    }
    
    // Get current user details
    currentUser, err := h.db.GetUserByID(context.Background(), userID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to get user details: %v", err)
        return
    }
    
    // Check if user already exists
    invitedUser, err := h.db.GetUserByEmail(context.Background(), email)
    if err != nil {
        // User doesn't exist, create them
        invitedUser, err = h.db.CreateUser(context.Background(), email, "")
        if err != nil {
            c.String(http.StatusInternalServerError, "Failed to create user: %v", err)
            return
        }
    }
    
    // Check if user is already a member
    isMember, err := h.db.IsBoardMember(context.Background(), boardID, invitedUser.ID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check membership: %v", err)
        return
    }
    
    if isMember {
        c.String(http.StatusBadRequest, "User is already a member of this board")
        return
    }
    
    // Add user to board
    err = h.db.AddBoardMember(context.Background(), boardID, invitedUser.ID, role)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to add board member: %v", err)
        return
    }
    
    // Send invitation email
    inviteURL := fmt.Sprintf("%s/boards/%s", getBaseURL(c), boardID.String())
    err = h.emailService.SendInvitation(email, currentUser.Name, board.Title, inviteURL)
    if err != nil {
        // Log error but don't fail the request
        fmt.Printf("Failed to send invitation email: %v\n", err)
    }
    
    c.String(http.StatusOK, "Invitation sent successfully")
}

func (h *BoardHandler) UpdateBoard(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    boardIDStr := c.Param("id")
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    // Check if user has access to this board
    hasAccess, err := h.checkBoardAccess(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
        return
    }
    
    if !hasAccess {
        c.String(http.StatusForbidden, "You don't have access to this board")
        return
    }
    
    title := c.PostForm("title")
    description := c.PostForm("description")
    
    updates := map[string]interface{}{}
    if title != "" {
        updates["title"] = title
    }
    if description != "" {
        updates["description"] = description
    }
    
    if len(updates) == 0 {
        c.String(http.StatusBadRequest, "No updates provided")
        return
    }
    
    err = h.db.UpdateBoard(context.Background(), boardID, updates)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to update board: %v", err)
        return
    }
    
    c.Status(http.StatusOK)
}

func (h *BoardHandler) DeleteBoard(c *gin.Context) {
    fmt.Printf("DeleteBoard handler called\n")
    
    user, err := h.validateUserSession(c)
    if err != nil {
        fmt.Printf("Board deletion session error: %v\n", err)
        c.Status(http.StatusUnauthorized)
        return
    }
    
    boardIDStr := c.Param("id")
    fmt.Printf("Attempting to delete board: %s\n", boardIDStr)
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        fmt.Printf("Invalid board ID: %s\n", boardIDStr)
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    // Check if user owns this board
    fmt.Printf("Checking ownership for user %s on board %s\n", user.ID.String(), boardID.String())
    isOwner, err := h.checkBoardOwnership(user.ID, boardID)
    if err != nil {
        fmt.Printf("Error checking board ownership: %v\n", err)
        c.String(http.StatusInternalServerError, "Failed to check board ownership: %v", err)
        return
    }
    
    fmt.Printf("Ownership check result: %v\n", isOwner)
    if !isOwner {
        fmt.Printf("User %s does not own board %s\n", user.ID.String(), boardID.String())
        c.String(http.StatusForbidden, "Only board owners can delete the board")
        return
    }
    
    fmt.Printf("Deleting board: %s by user %s\n", boardID.String(), user.Email)
    err = h.db.DeleteBoard(context.Background(), boardID)
    if err != nil {
        fmt.Printf("Failed to delete board: %v\n", err)
        c.String(http.StatusInternalServerError, "Failed to delete board: %v", err)
        return
    }
    
    fmt.Printf("Board deleted successfully, redirecting to dashboard\n")
    c.Header("HX-Redirect", "/dashboard")
    c.Status(http.StatusOK)
}

func (h *BoardHandler) GetBoardMembers(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    boardIDStr := c.Param("id")
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    // Check if user has access to this board
    hasAccess, err := h.checkBoardAccess(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
        return
    }
    
    if !hasAccess {
        c.String(http.StatusForbidden, "You don't have access to this board")
        return
    }
    
    members, err := h.db.GetBoardMembers(context.Background(), boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to get board members: %v", err)
        return
    }
    
    c.JSON(http.StatusOK, members)
}

func (h *BoardHandler) UpdateColumn(c *gin.Context) {
    _, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    columnIDStr := c.Param("id")
    columnID, err := uuid.Parse(columnIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid column ID")
        return
    }
    
    title := c.PostForm("title")
    if title == "" {
        c.String(http.StatusBadRequest, "Column title is required")
        return
    }
    
    updates := map[string]interface{}{
        "title": title,
    }
    
    err = h.db.UpdateColumn(context.Background(), columnID, updates)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to update column: %v", err)
        return
    }
    
    c.Status(http.StatusOK)
}

func (h *BoardHandler) DeleteColumn(c *gin.Context) {
    user, err := h.validateUserSession(c)
    if err != nil {
        fmt.Printf("Column deletion session error: %v\n", err)
        c.Status(http.StatusUnauthorized)
        return
    }
    
    columnIDStr := c.Param("id")
    columnID, err := uuid.Parse(columnIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid column ID")
        return
    }
    
    fmt.Printf("Deleting column: %s by user %s\n", columnID.String(), user.Email)
    err = h.db.DeleteColumn(context.Background(), columnID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to delete column: %v", err)
        return
    }
    
    c.Status(http.StatusOK)
}

func (h *BoardHandler) RemoveBoardMember(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    boardIDStr := c.PostForm("board_id")
    memberIDStr := c.PostForm("member_id")
    
    if boardIDStr == "" || memberIDStr == "" {
        c.String(http.StatusBadRequest, "Board ID and member ID are required")
        return
    }
    
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    memberID, err := uuid.Parse(memberIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid member ID")
        return
    }
    
    // Check if user is admin or owner of this board
    isAdmin, err := h.checkBoardAdmin(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board permissions: %v", err)
        return
    }
    
    if !isAdmin {
        c.String(http.StatusForbidden, "You don't have permission to remove members")
        return
    }
    
    err = h.db.RemoveBoardMember(context.Background(), boardID, memberID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to remove board member: %v", err)
        return
    }
    
    c.Status(http.StatusOK)
}

func (h *BoardHandler) GetBoardTasks(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    boardIDStr := c.Param("id")
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    // Check if user has access to this board
    hasAccess, err := h.checkBoardAccess(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
        return
    }
    
    if !hasAccess {
        c.String(http.StatusForbidden, "You don't have access to this board")
        return
    }
    
    board, err := h.db.GetBoardWithColumns(context.Background(), boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to get board: %v", err)
        return
    }
    
    // Flatten all tasks from all columns
    var allTasks []models.Task
    for _, column := range board.Columns {
        allTasks = append(allTasks, column.Tasks...)
    }
    
    c.JSON(http.StatusOK, allTasks)
}

func (h *BoardHandler) GetNestedBoards(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    boardIDStr := c.Param("id")
    boardID, err := uuid.Parse(boardIDStr)
    if err != nil {
        c.String(http.StatusBadRequest, "Invalid board ID")
        return
    }
    
    // Check if user has access to this board
    hasAccess, err := h.checkBoardAccess(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board access: %v", err)
        return
    }
    
    if !hasAccess {
        c.String(http.StatusForbidden, "You don't have access to this board")
        return
    }
    
    nestedBoards, err := h.db.GetNestedBoards(context.Background(), boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to get nested boards: %v", err)
        return
    }
    
    c.JSON(http.StatusOK, nestedBoards)
}

func (h *BoardHandler) SearchContent(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusOK, gin.H{"boards": []models.Board{}, "tasks": []models.Task{}})
        return
    }
    
    // Get user's boards
    boards, err := h.db.GetUserBoards(context.Background(), userID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to search: %v", err)
        return
    }
    
    // Simple search implementation - in production, use full-text search
    var matchingBoards []models.Board
    var matchingTasks []models.Task
    
    for _, board := range boards {
        // Search board titles and descriptions
        if strings.Contains(strings.ToLower(board.Title), strings.ToLower(query)) ||
           strings.Contains(strings.ToLower(board.Description), strings.ToLower(query)) {
            matchingBoards = append(matchingBoards, board)
        }
        
        // Search tasks in this board
        boardWithColumns, err := h.db.GetBoardWithColumns(context.Background(), board.ID)
        if err != nil {
            continue
        }
        
        for _, column := range boardWithColumns.Columns {
            for _, task := range column.Tasks {
                if strings.Contains(strings.ToLower(task.Title), strings.ToLower(query)) ||
                   strings.Contains(strings.ToLower(task.Description), strings.ToLower(query)) {
                    matchingTasks = append(matchingTasks, task)
                }
            }
        }
    }
    
    c.JSON(http.StatusOK, gin.H{
        "boards": matchingBoards,
        "tasks":  matchingTasks,
    })
}

func (h *BoardHandler) HandleWebSocket(c *gin.Context) {
    // WebSocket implementation for real-time updates
    // This is a placeholder - implement with gorilla/websocket or similar
    c.String(http.StatusNotImplemented, "WebSocket endpoint not implemented yet")
}

// Helper functions
func (h *BoardHandler) checkBoardAccess(userID, boardID uuid.UUID) (bool, error) {
    return h.db.HasBoardAccess(context.Background(), userID, boardID)
}

func (h *BoardHandler) checkBoardOwnership(userID, boardID uuid.UUID) (bool, error) {
    return h.db.IsBoardOwner(context.Background(), userID, boardID)
}

func (h *BoardHandler) checkBoardAdmin(userID, boardID uuid.UUID) (bool, error) {
    return h.db.IsBoardAdmin(context.Background(), userID, boardID)
}

func getUserFromSession(c *gin.Context) (uuid.UUID, error) {
    session := sessions.Default(c)
    userIDStr := session.Get("user_id")
    if userIDStr == nil {
        return uuid.Nil, fmt.Errorf("user not logged in")
    }
    
    return uuid.Parse(userIDStr.(string))
}

func (h *BoardHandler) validateUserSession(c *gin.Context) (*models.User, error) {
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

func getBaseURL(c *gin.Context) string {
    scheme := "http"
    if c.Request.Header.Get("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
        scheme = "https"
    }
    return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}