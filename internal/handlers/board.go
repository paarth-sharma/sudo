package handlers

import (
    "context"
    "net/http"
    "fmt"
    
    "sudo/internal/database"
    "sudo/internal/email"
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
    userID, err := getUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusSeeOther, "/")
        return
    }
    
    boards, err := h.db.GetUserBoards(context.Background(), userID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to load boards: %v", err)
        return
    }
    
    component := pages.Dashboard(boards)
    handler := templ.Handler(component)
    handler.ServeHTTP(c.Writer, c.Request)
}

func (h *BoardHandler) CreateBoard(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    title := c.PostForm("title")
    description := c.PostForm("description")
    parentBoardIDStr := c.PostForm("parent_board_id")
    
    if title == "" {
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
    
    board, err := h.db.CreateBoard(context.Background(), title, description, userID, parentBoardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to create board: %v", err)
        return
    }
    
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
    
    component := pages.Board(*board)
    handler := templ.Handler(component)
    handler.ServeHTTP(c.Writer, c.Request)
}

func (h *BoardHandler) CreateColumn(c *gin.Context) {
    userID, err := getUserFromSession(c)
    if err != nil {
        c.String(http.StatusUnauthorized, "Unauthorized")
        return
    }
    
    boardIDStr := c.PostForm("board_id")
    title := c.PostForm("title")
    
    if boardIDStr == "" || title == "" {
        c.String(http.StatusBadRequest, "Board ID and title are required")
        return
    }
    
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
    
    // Get current column count for position
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
    err = h.emailService.SendInvitationEmail(email, board.Title, currentUser.Name, inviteURL)
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
    
    // Check if user owns this board
    isOwner, err := h.checkBoardOwnership(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board ownership: %v", err)
        return
    }
    
    if !isOwner {
        c.String(http.StatusForbidden, "Only board owners can update the board")
        return
    }
    
    updates := make(map[string]interface{})
    
    if title := c.PostForm("title"); title != "" {
        updates["title"] = title
    }
    
    if description := c.PostForm("description"); description != "" {
        updates["description"] = description
    }
    
    err = h.db.UpdateBoard(context.Background(), boardID, updates)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to update board: %v", err)
        return
    }
    
    c.Status(http.StatusOK)
}

func (h *BoardHandler) DeleteBoard(c *gin.Context) {
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
    
    // Check if user owns this board
    isOwner, err := h.checkBoardOwnership(userID, boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to check board ownership: %v", err)
        return
    }
    
    if !isOwner {
        c.String(http.StatusForbidden, "Only board owners can delete the board")
        return
    }
    
    err = h.db.DeleteBoard(context.Background(), boardID)
    if err != nil {
        c.String(http.StatusInternalServerError, "Failed to delete board: %v", err)
        return
    }
    
    c.Header("HX-Redirect", "/dashboard")
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

func getBaseURL(c *gin.Context) string {
    scheme := "http"
    if c.Request.Header.Get("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
        scheme = "https"
    }
    return fmt.Sprintf("%s://%s", scheme, c.Request.Host)
}