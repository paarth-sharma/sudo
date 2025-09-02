package database

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/google/uuid"
    "github.com/supabase-community/supabase-go"
    "github.com/supabase-community/postgrest-go"

    "sudo/internal/models"
)

type DB struct {
    client *supabase.Client
}

func NewDB() *DB {
    url := os.Getenv("SUPABASE_URL")
    key := os.Getenv("SUPABASE_SERVICE_KEY")
    
    if url == "" || key == "" {
        log.Fatal("SUPABASE_URL and SUPABASE_SERVICE_KEY must be set")
    }
    
    client, err := supabase.NewClient(url, key, nil)
    if err != nil {
        log.Fatal("Failed to initialize Supabase client:", err)
    }
    
    return &DB{client: client}
}

// User operations
func (db *DB) CreateUser(ctx context.Context, email, name string) (*models.User, error) {
    // Use map instead of struct to avoid UUID issues
    userData := map[string]interface{}{
        "email": email,
        "name":  name,
    }
    
    var result []models.User
    _, err := db.client.From("users").Insert(userData, false, "", "", "").ExecuteTo(&result)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    if len(result) > 0 {
        log.Printf("User created with ID: %s", result[0].ID.String())
        return &result[0], nil
    }
    
    return nil, fmt.Errorf("failed to get created user data")
}

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    var users []models.User
    _, err := db.client.From("users").Select("*", "", false).Eq("email", email).ExecuteTo(&users)
    if err != nil {
        return nil, fmt.Errorf("failed to get user by email: %w", err)
    }
    
    if len(users) == 0 {
        return nil, fmt.Errorf("user not found")
    }
    
    return &users[0], nil
}

func (db *DB) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
    var users []models.User
    _, err := db.client.From("users").Select("*", "", false).Eq("id", userID.String()).ExecuteTo(&users)
    if err != nil {
        return nil, fmt.Errorf("failed to get user by ID: %w", err)
    }
    
    if len(users) == 0 {
        return nil, fmt.Errorf("user not found")
    }
    
    return &users[0], nil
}

// Board operations
func (db *DB) CreateBoard(ctx context.Context, title, description string, ownerID uuid.UUID, parentBoardID *uuid.UUID) (*models.Board, error) {
    // Use map instead of struct to let database generate UUID
    boardData := map[string]interface{}{
        "title":       title,
        "description": description,
        "owner_id":    ownerID.String(),
    }
    
    if parentBoardID != nil {
        boardData["parent_board_id"] = parentBoardID.String()
    }
    
    var result []models.Board
    _, err := db.client.From("boards").Insert(boardData, false, "", "", "").ExecuteTo(&result)
    if err != nil {
        return nil, fmt.Errorf("failed to create board: %w", err)
    }
    
    if len(result) > 0 {
        // Create default columns
        defaultColumns := []string{"To Do", "In Progress", "Review", "Done"}
        for i, colTitle := range defaultColumns {
            _, err := db.CreateColumn(ctx, result[0].ID, colTitle, i)
            if err != nil {
                log.Printf("Failed to create default column %s: %v", colTitle, err)
            }
        }
        
        return &result[0], nil
    }
    
    return nil, fmt.Errorf("failed to get created board data")
}

func (db *DB) GetUserBoards(ctx context.Context, userID uuid.UUID) ([]models.Board, error) {
    var boards []models.Board
    
    // Get boards where user is owner
    _, err := db.client.From("boards").
        Select("*", "", false).
        Eq("owner_id", userID.String()).
        Order("created_at", nil).
        ExecuteTo(&boards)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get user boards: %w", err)
    }
    
    return boards, nil
}

func (db *DB) GetNestedBoards(ctx context.Context, parentBoardID uuid.UUID) ([]models.Board, error) {
    var nestedBoards []models.Board
    
    // Get boards where parent_board_id matches the given ID
    _, err := db.client.From("boards").
        Select("*", "", false).
        Eq("parent_board_id", parentBoardID.String()).
        Order("created_at", nil).
        ExecuteTo(&nestedBoards)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get nested boards: %w", err)
    }
    
    return nestedBoards, nil
}

func (db *DB) GetBoardWithColumns(ctx context.Context, boardID uuid.UUID) (*models.Board, error) {
    var boards []models.Board
    _, err := db.client.From("boards").Select("*", "", false).Eq("id", boardID.String()).ExecuteTo(&boards)
    if err != nil {
        return nil, fmt.Errorf("failed to get board: %w", err)
    }
    
    if len(boards) == 0 {
        return nil, fmt.Errorf("board not found")
    }
    
    board := boards[0]
    
    // Get columns
    columns, err := db.GetBoardColumns(ctx, boardID)
    if err != nil {
        return nil, fmt.Errorf("failed to get board columns: %w", err)
    }
    
    board.Columns = columns
    return &board, nil
}

func (db *DB) UpdateBoard(ctx context.Context, boardID uuid.UUID, updates map[string]interface{}) error {
    updates["updated_at"] = time.Now()
    
    _, err := db.client.From("boards").
        Update(updates, "", "").
        Eq("id", boardID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to update board: %w", err)
    }
    
    return nil
}

func (db *DB) DeleteBoard(ctx context.Context, boardID uuid.UUID) error {
    _, err := db.client.From("boards").
        Delete("", "").
        Eq("id", boardID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to delete board: %w", err)
    }
    
    return nil
}

func (db *DB) HasBoardAccess(ctx context.Context, userID, boardID uuid.UUID) (bool, error) {
    // Check if user is owner
    var boards []models.Board
    _, err := db.client.From("boards").
        Select("id", "", false).
        Eq("id", boardID.String()).
        Eq("owner_id", userID.String()).
        ExecuteTo(&boards)
    
    if err != nil {
        return false, fmt.Errorf("failed to check board ownership: %w", err)
    }
    
    if len(boards) > 0 {
        return true, nil
    }
    
    // Check if user is a board member
    return db.IsBoardMember(ctx, boardID, userID)
}

func (db *DB) IsBoardOwner(ctx context.Context, userID, boardID uuid.UUID) (bool, error) {
    var boards []models.Board
    _, err := db.client.From("boards").
        Select("id", "", false).
        Eq("id", boardID.String()).
        Eq("owner_id", userID.String()).
        ExecuteTo(&boards)
    
    if err != nil {
        return false, fmt.Errorf("failed to check board ownership: %w", err)
    }
    
    return len(boards) > 0, nil
}

func (db *DB) IsBoardAdmin(ctx context.Context, userID, boardID uuid.UUID) (bool, error) {
    // Check if user is owner first
    isOwner, err := db.IsBoardOwner(ctx, userID, boardID)
    if err != nil {
        return false, err
    }
    
    if isOwner {
        return true, nil
    }
    
    // Check if user is admin member
    var members []models.BoardMember
    _, err = db.client.From("board_members").
        Select("id", "", false).
        Eq("board_id", boardID.String()).
        Eq("user_id", userID.String()).
        Eq("role", "admin").
        ExecuteTo(&members)
    
    if err != nil {
        return false, fmt.Errorf("failed to check board admin status: %w", err)
    }
    
    return len(members) > 0, nil
}

// Board member operations
func (db *DB) IsBoardMember(ctx context.Context, boardID, userID uuid.UUID) (bool, error) {
    var members []models.BoardMember
    _, err := db.client.From("board_members").
        Select("id", "", false).
        Eq("board_id", boardID.String()).
        Eq("user_id", userID.String()).
        ExecuteTo(&members)
    
    if err != nil {
        return false, fmt.Errorf("failed to check board membership: %w", err)
    }
    
    return len(members) > 0, nil
}

func (db *DB) AddBoardMember(ctx context.Context, boardID, userID uuid.UUID, role string) error {
    member := models.BoardMember{
        BoardID: boardID,
        UserID:  userID,
        Role:    role,
    }
    
    _, err := db.client.From("board_members").Insert(member, false, "", "", "").ExecuteTo(nil)
    if err != nil {
        return fmt.Errorf("failed to add board member: %w", err)
    }
    
    return nil
}

func (db *DB) RemoveBoardMember(ctx context.Context, boardID, userID uuid.UUID) error {
    _, err := db.client.From("board_members").
        Delete("", "").
        Eq("board_id", boardID.String()).
        Eq("user_id", userID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to remove board member: %w", err)
    }
    
    return nil
}

func (db *DB) GetBoardMembers(ctx context.Context, boardID uuid.UUID) ([]models.User, error) {
    // First get all board members with their user info
    var members []models.BoardMember
    _, err := db.client.From("board_members").
        Select("user_id, role", "", false).
        Eq("board_id", boardID.String()).
        ExecuteTo(&members)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get board members: %w", err)
    }
    
    // Get user details for each member
    var users []models.User
    for _, member := range members {
        user, err := db.GetUserByID(context.Background(), member.UserID)
        if err != nil {
            log.Printf("Failed to get user %s: %v", member.UserID, err)
            continue
        }
        users = append(users, *user)
    }
    
    return users, nil
}

// Column operations
func (db *DB) CreateColumn(ctx context.Context, boardID uuid.UUID, title string, position int) (*models.Column, error) {
    // Use map instead of struct to let database generate UUID
    columnData := map[string]interface{}{
        "board_id": boardID.String(),
        "title":    title,
        "position": position,
    }
    
    var result []models.Column
    _, err := db.client.From("columns").Insert(columnData, false, "", "", "").ExecuteTo(&result)
    if err != nil {
        return nil, fmt.Errorf("failed to create column: %w", err)
    }
    
    if len(result) > 0 {
        return &result[0], nil
    }
    
    return nil, fmt.Errorf("failed to get created column data")
}

func (db *DB) GetBoardColumns(ctx context.Context, boardID uuid.UUID) ([]models.Column, error) {
    var columns []models.Column
    _, err := db.client.From("columns").
        Select("*", "", false).
        Eq("board_id", boardID.String()).
        Order("position", &postgrest.OrderOpts{Ascending: true}).
        ExecuteTo(&columns)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get board columns: %w", err)
    }
    
    // Get tasks for each column
    for i := range columns {
        tasks, err := db.GetColumnTasks(ctx, columns[i].ID)
        if err != nil {
            log.Printf("Failed to get tasks for column %s: %v", columns[i].ID, err)
            continue
        }
        columns[i].Tasks = tasks
    }
    
    return columns, nil
}

func (db *DB) UpdateColumn(ctx context.Context, columnID uuid.UUID, updates map[string]interface{}) error {
    updates["updated_at"] = time.Now()
    
    _, err := db.client.From("columns").
        Update(updates, "", "").
        Eq("id", columnID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to update column: %w", err)
    }
    
    return nil
}

func (db *DB) DeleteColumn(ctx context.Context, columnID uuid.UUID) error {
    _, err := db.client.From("columns").
        Delete("", "").
        Eq("id", columnID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to delete column: %w", err)
    }
    
    return nil
}

// Task operations
func (db *DB) CreateTask(ctx context.Context, title, description string, columnID, boardID uuid.UUID, priority string) (*models.Task, error) {
    // Get next position
    var tasks []models.Task
    _, _ = db.client.From("tasks").
        Select("position", "", false).
        Eq("column_id", columnID.String()).
        Order("position", &postgrest.OrderOpts{Ascending: false}).
        Limit(1, "").
        ExecuteTo(&tasks)
    
    position := 0
    if len(tasks) > 0 {
        position = tasks[0].Position + 1
    }
    
    // Use map instead of struct to let database generate UUID
    taskData := map[string]interface{}{
        "title":       title,
        "description": description,
        "column_id":   columnID.String(),
        "board_id":    boardID.String(),
        "priority":    priority,
        "position":    position,
    }
    
    var result []models.Task
    _, err := db.client.From("tasks").Insert(taskData, false, "", "", "").ExecuteTo(&result)
    if err != nil {
        return nil, fmt.Errorf("failed to create task: %w", err)
    }
    
    if len(result) > 0 {
        return &result[0], nil
    }
    
    return nil, fmt.Errorf("failed to get created task data")
}

func (db *DB) GetTask(ctx context.Context, taskID uuid.UUID) (*models.Task, error) {
    var tasks []models.Task
    _, err := db.client.From("tasks").
        Select("*", "", false).
        Eq("id", taskID.String()).
        ExecuteTo(&tasks)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get task: %w", err)
    }
    
    if len(tasks) == 0 {
        return nil, fmt.Errorf("task not found")
    }
    
    return &tasks[0], nil
}

func (db *DB) GetColumnTasks(ctx context.Context, columnID uuid.UUID) ([]models.Task, error) {
    var tasks []models.Task
    _, err := db.client.From("tasks").
        Select("*", "", false).
        Eq("column_id", columnID.String()).
        Order("position", &postgrest.OrderOpts{Ascending: true}).
        ExecuteTo(&tasks)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get column tasks: %w", err)
    }
    
    return tasks, nil
}

func (db *DB) UpdateTask(ctx context.Context, taskID uuid.UUID, updates map[string]interface{}) error {
    updates["updated_at"] = time.Now()
    
    _, err := db.client.From("tasks").
        Update(updates, "", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to update task: %w", err)
    }
    
    return nil
}

func (db *DB) MoveTask(ctx context.Context, taskID, newColumnID uuid.UUID, newPosition int) error {
    updates := map[string]interface{}{
        "column_id": newColumnID,
        "position":  newPosition,
        "updated_at": time.Now(),
    }
    
    _, err := db.client.From("tasks").
        Update(updates, "", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to move task: %w", err)
    }
    
    return nil
}

func (db *DB) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
    _, err := db.client.From("tasks").
        Delete("", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to delete task: %w", err)
    }
    
    return nil
}

func (db *DB) AssignTask(ctx context.Context, taskID, userID uuid.UUID) error {
    updates := map[string]interface{}{
        "assigned_to": userID,
        "updated_at":  time.Now(),
    }
    
    _, err := db.client.From("tasks").
        Update(updates, "", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to assign task: %w", err)
    }
    
    return nil
}

func (db *DB) UnassignTask(ctx context.Context, taskID uuid.UUID) error {
    updates := map[string]interface{}{
        "assigned_to": nil,
        "updated_at":  time.Now(),
    }
    
    _, err := db.client.From("tasks").
        Update(updates, "", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to unassign task: %w", err)
    }
    
    return nil
}

// OTP operations
func (db *DB) CreateOTP(ctx context.Context, email, token string, expiresAt time.Time) error {
    // Create a map instead of struct to avoid UUID issues
    otp := map[string]interface{}{
        "email":      email,
        "token":      token,
        "expires_at": expiresAt.UTC(),
    }
    
    _, err := db.client.From("otp_tokens").Insert(otp, false, "", "", "").ExecuteTo(nil)
    if err != nil {
        return fmt.Errorf("failed to create OTP: %w", err)
    }
    
    return nil
}

func (db *DB) ValidateOTP(ctx context.Context, email, token string) (*models.User, error) {
    log.Printf("ValidateOTP called with email=%s, token=%s", email, token)
    
    var otps []models.OTPToken
    
    // First, try to find any OTP with matching email and token (ignore used status temporarily)
    _, err := db.client.From("otp_tokens").
        Select("*", "", false).
        Eq("email", email).
        Eq("token", token).
        ExecuteTo(&otps)
    
    if err != nil {
        log.Printf("Database query failed: %v", err)
        return nil, fmt.Errorf("failed to validate OTP: %w", err)
    }
    
    log.Printf("Found %d OTP records for email=%s, token=%s", len(otps), email, token)
    
    if len(otps) == 0 {
        log.Printf("No OTP found for email=%s, token=%s", email, token)
        return nil, fmt.Errorf("no OTP found for email and token")
    }
    
    // Check if any valid (unused and not expired) OTP exists
    validOTP := (*models.OTPToken)(nil)
    now := time.Now().UTC()
    
    for i := range otps {
        log.Printf("Checking OTP[%d]: ID=%s, Used=%v, ExpiresAt=%v, Now=%v", 
            i, otps[i].ID.String(), otps[i].Used, otps[i].ExpiresAt, now)
            
        if !otps[i].Used && otps[i].ExpiresAt.After(now) {
            validOTP = &otps[i]
            log.Printf("Found valid OTP: %s", validOTP.ID.String())
            break
        }
    }
    
    if validOTP == nil {
        log.Printf("No valid OTP found - all are either used or expired")
        return nil, fmt.Errorf("OTP is used or expired")
    }
    
    // Mark OTP as used
    log.Printf("Marking OTP as used: %s", validOTP.ID.String())
    _, err = db.client.From("otp_tokens").
        Update(map[string]interface{}{"used": true}, "", "").
        Eq("id", validOTP.ID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        log.Printf("Failed to mark OTP as used: %v", err)
        return nil, fmt.Errorf("failed to mark OTP as used: %w", err)
    }
    
    log.Printf("OTP marked as used successfully")
    
    // Get or create user
    log.Printf("Looking up user by email: %s", email)
    user, err := db.GetUserByEmail(ctx, email)
    if err != nil {
        log.Printf("User not found, creating new user: %s", email)
        // Create new user
        user, err = db.CreateUser(ctx, email, "")
        if err != nil {
            log.Printf("Failed to create user: %v", err)
            return nil, fmt.Errorf("failed to create user: %w", err)
        }
        log.Printf("New user created: %s (ID: %s)", user.Email, user.ID.String())
    } else {
        log.Printf("Existing user found: %s (ID: %s)", user.Email, user.ID.String())
        
        // Check if user has zero UUID and fix it
        if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
            log.Printf("User has zero UUID, recreating user record")
            // Delete the broken user record
            _, deleteErr := db.client.From("users").
                Delete("", "").
                Eq("email", email).
                ExecuteTo(nil)
            if deleteErr != nil {
                log.Printf("Failed to delete broken user record: %v", deleteErr)
            }
            
            // Create new user
            user, err = db.CreateUser(ctx, email, "")
            if err != nil {
                log.Printf("Failed to recreate user: %v", err)
                return nil, fmt.Errorf("failed to recreate user: %w", err)
            }
            log.Printf("User recreated with new ID: %s", user.ID.String())
        }
    }
    
    log.Printf("OTP validation completed successfully for user: %s", user.Email)
    return user, nil
}