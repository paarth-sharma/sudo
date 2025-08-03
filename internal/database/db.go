package database

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "github.com/google/uuid"
    "github.com/supabase-community/supabase-go"

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
    user := models.User{
        Email: email,
        Name:  name,
    }
    
    result, _, err := db.client.From("users").Insert(user, false, "", "", "").ExecuteTo(&user)
    if err != nil {
        return nil, fmt.Errorf("failed to create user: %w", err)
    }
    
    if len(result) > 0 {
        return &result[0], nil
    }
    
    return &user, nil
}

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
    var users []models.User
    _, _, err := db.client.From("users").Select("*", "", false).Eq("email", email).ExecuteTo(&users)
    if err != nil {
        return nil, fmt.Errorf("failed to get user by email: %w", err)
    }
    
    if len(users) == 0 {
        return nil, fmt.Errorf("user not found")
    }
    
    return &users[0], nil
}

// Board operations
func (db *DB) CreateBoard(ctx context.Context, title, description string, ownerID uuid.UUID, parentBoardID *uuid.UUID) (*models.Board, error) {
    board := models.Board{
        Title:         title,
        Description:   description,
        OwnerID:       ownerID,
        ParentBoardID: parentBoardID,
    }
    
    var result []models.Board
    _, _, err := db.client.From("boards").Insert(board, false, "", "", "").ExecuteTo(&result)
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
    
    return &board, nil
}

func (db *DB) GetUserBoards(ctx context.Context, userID uuid.UUID) ([]models.Board, error) {
    var boards []models.Board
    
    // Get boards where user is owner
    _, _, err := db.client.From("boards").
        Select("*", "", false).
        Eq("owner_id", userID.String()).
        Order("created_at", &supabase.OrderOpts{Ascending: false}).
        ExecuteTo(&boards)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get user boards: %w", err)
    }
    
    return boards, nil
}

func (db *DB) GetBoardWithColumns(ctx context.Context, boardID uuid.UUID) (*models.Board, error) {
    var boards []models.Board
    _, _, err := db.client.From("boards").Select("*").Eq("id", boardID.String()).ExecuteTo(&boards)
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

// Column operations
func (db *DB) CreateColumn(ctx context.Context, boardID uuid.UUID, title string, position int) (*models.Column, error) {
    column := models.Column{
        BoardID:  boardID,
        Title:    title,
        Position: position,
    }
    
    var result []models.Column
    _, _, err := db.client.From("columns").Insert(column, false, "", "", "").ExecuteTo(&result)
    if err != nil {
        return nil, fmt.Errorf("failed to create column: %w", err)
    }
    
    if len(result) > 0 {
        return &result[0], nil
    }
    
    return &column, nil
}

func (db *DB) GetBoardColumns(ctx context.Context, boardID uuid.UUID) ([]models.Column, error) {
    var columns []models.Column
    _, _, err := db.client.From("columns").
        Select("*").
        Eq("board_id", boardID.String()).
        Order("position", &supabase.OrderOpts{Ascending: true}).
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

// Task operations
func (db *DB) CreateTask(ctx context.Context, title, description string, columnID, boardID uuid.UUID, priority string) (*models.Task, error) {
    // Get next position
    var tasks []models.Task
    db.client.From("tasks").
        Select("position").
        Eq("column_id", columnID.String()).
        Order("position", &supabase.OrderOpts{Ascending: false}).
        Limit(1, "").
        ExecuteTo(&tasks)
    
    position := 0
    if len(tasks) > 0 {
        position = tasks[0].Position + 1
    }
    
    task := models.Task{
        Title:       title,
        Description: description,
        ColumnID:    columnID,
        BoardID:     boardID,
        Priority:    priority,
        Position:    position,
    }
    
    var result []models.Task
    _, _, err := db.client.From("tasks").Insert(task, false, "", "", "").ExecuteTo(&result)
    if err != nil {
        return nil, fmt.Errorf("failed to create task: %w", err)
    }
    
    if len(result) > 0 {
        return &result[0], nil
    }
    
    return &task, nil
}

func (db *DB) GetColumnTasks(ctx context.Context, columnID uuid.UUID) ([]models.Task, error) {
    var tasks []models.Task
    _, _, err := db.client.From("tasks").
        Select("*").
        Eq("column_id", columnID.String()).
        Order("position", &supabase.OrderOpts{Ascending: true}).
        ExecuteTo(&tasks)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get column tasks: %w", err)
    }
    
    return tasks, nil
}

func (db *DB) MoveTask(ctx context.Context, taskID, newColumnID uuid.UUID, newPosition int) error {
    updates := map[string]interface{}{
        "column_id": newColumnID,
        "position":  newPosition,
        "updated_at": time.Now(),
    }
    
    _, _, err := db.client.From("tasks").
        Update(updates, "", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to move task: %w", err)
    }
    
    return nil
}

func (db *DB) UpdateTask(ctx context.Context, taskID uuid.UUID, updates map[string]interface{}) error {
    updates["updated_at"] = time.Now()
    
    _, _, err := db.client.From("tasks").
        Update(updates, "", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to update task: %w", err)
    }
    
    return nil
}

// OTP operations
func (db *DB) CreateOTP(ctx context.Context, email, token string, expiresAt time.Time) error {
    otp := models.OTPToken{
        Email:     email,
        Token:     token,
        ExpiresAt: expiresAt,
    }
    
    _, _, err := db.client.From("otp_tokens").Insert(otp, false, "", "", "").ExecuteTo(nil)
    if err != nil {
        return fmt.Errorf("failed to create OTP: %w", err)
    }
    
    return nil
}

func (db *DB) ValidateOTP(ctx context.Context, email, token string) (*models.User, error) {
    var otps []models.OTPToken
    _, _, err := db.client.From("otp_tokens").
        Select("*").
        Eq("email", email).
        Eq("token", token).
        Eq("used", false).
        Gt("expires_at", time.Now().Format(time.RFC3339)).
        ExecuteTo(&otps)
    
    if err != nil {
        return nil, fmt.Errorf("failed to validate OTP: %w", err)
    }
    
    if len(otps) == 0 {
        return nil, fmt.Errorf("invalid or expired OTP")
    }
    
    // Mark OTP as used
    _, _, err = db.client.From("otp_tokens").
        Update(map[string]interface{}{"used": true}, "", "").
        Eq("id", otps[0].ID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return nil, fmt.Errorf("failed to mark OTP as used: %w", err)
    }
    
    // Get or create user
    user, err := db.GetUserByEmail(ctx, email)
    if err != nil {
        // Create new user
        user, err = db.CreateUser(ctx, email, "")
        if err != nil {
            return nil, fmt.Errorf("failed to create user: %w", err)
        }
    }
    
    return user, nil
}
// Add these methods to internal/database/db.go

func (db *DB) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
    var users []models.User
    _, _, err := db.client.From("users").Select("*", "", false).Eq("id", userID.String()).ExecuteTo(&users)
    if err != nil {
        return nil, fmt.Errorf("failed to get user by ID: %w", err)
    }
    
    if len(users) == 0 {
        return nil, fmt.Errorf("user not found")
    }
    
    return &users[0], nil
}

func (db *DB) IsBoardMember(ctx context.Context, boardID, userID uuid.UUID) (bool, error) {
    var members []models.BoardMember
    _, _, err := db.client.From("board_members").
        Select("id").
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
    
    _, _, err := db.client.From("board_members").Insert(member, false, "", "", "").ExecuteTo(nil)
    if err != nil {
        return fmt.Errorf("failed to add board member: %w", err)
    }
    
    return nil
}

func (db *DB) RemoveBoardMember(ctx context.Context, boardID, userID uuid.UUID) error {
    _, _, err := db.client.From("board_members").
        Delete("", "").
        Eq("board_id", boardID.String()).
        Eq("user_id", userID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to remove board member: %w", err)
    }
    
    return nil
}

func (db *DB) HasBoardAccess(ctx context.Context, userID, boardID uuid.UUID) (bool, error) {
    // Check if user is owner
    var boards []models.Board
    _, _, err := db.client.From("boards").
        Select("id").
        Eq("id", boardID.String()).
        Eq("owner_id", userID.String()).
        ExecuteTo(&boards)
    
    if err != nil {
        return false, fmt.Errorf("failed to check board ownership: %w", err)
    }
    
    if len(boards) > 0 {
        return true, nil
    }
    
    // Check if user is a member
    return db.IsBoardMember(ctx, boardID, userID)
}

func (db *DB) IsBoardOwner(ctx context.Context, userID, boardID uuid.UUID) (bool, error) {
    var boards []models.Board
    _, _, err := db.client.From("boards").
        Select("id").
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
    _, _, err = db.client.From("board_members").
        Select("id").
        Eq("board_id", boardID.String()).
        Eq("user_id", userID.String()).
        Eq("role", "admin").
        ExecuteTo(&members)
    
    if err != nil {
        return false, fmt.Errorf("failed to check board admin status: %w", err)
    }
    
    return len(members) > 0, nil
}

func (db *DB) UpdateBoard(ctx context.Context, boardID uuid.UUID, updates map[string]interface{}) error {
    updates["updated_at"] = time.Now()
    
    _, _, err := db.client.From("boards").
        Update(updates, "", "").
        Eq("id", boardID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to update board: %w", err)
    }
    
    return nil
}

func (db *DB) DeleteBoard(ctx context.Context, boardID uuid.UUID) error {
    _, _, err := db.client.From("boards").
        Delete("", "").
        Eq("id", boardID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to delete board: %w", err)
    }
    
    return nil
}

func (db *DB) UpdateColumn(ctx context.Context, columnID uuid.UUID, updates map[string]interface{}) error {
    updates["updated_at"] = time.Now()
    
    _, _, err := db.client.From("columns").
        Update(updates, "", "").
        Eq("id", columnID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to update column: %w", err)
    }
    
    return nil
}

func (db *DB) DeleteColumn(ctx context.Context, columnID uuid.UUID) error {
    _, _, err := db.client.From("columns").
        Delete("", "").
        Eq("id", columnID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to delete column: %w", err)
    }
    
    return nil
}

func (db *DB) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
    _, _, err := db.client.From("tasks").
        Delete("", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to delete task: %w", err)
    }
    
    return nil
}

func (db *DB) GetTask(ctx context.Context, taskID uuid.UUID) (*models.Task, error) {
    var tasks []models.Task
    _, _, err := db.client.From("tasks").Select("*").Eq("id", taskID.String()).ExecuteTo(&tasks)
    if err != nil {
        return nil, fmt.Errorf("failed to get task: %w", err)
    }
    
    if len(tasks) == 0 {
        return nil, fmt.Errorf("task not found")
    }
    
    return &tasks[0], nil
}

func (db *DB) GetBoardMembers(ctx context.Context, boardID uuid.UUID) ([]models.BoardMember, error) {
    var members []models.BoardMember
    _, _, err := db.client.From("board_members").
        Select("*, users(*)").
        Eq("board_id", boardID.String()).
        ExecuteTo(&members)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get board members: %w", err)
    }
    
    return members, nil
}

func (db *DB) AssignTask(ctx context.Context, taskID, userID uuid.UUID) error {
    updates := map[string]interface{}{
        "assignee_id": userID,
        "updated_at":  time.Now(),
    }
    
    _, _, err := db.client.From("tasks").
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
        "assignee_id": nil,
        "updated_at":  time.Now(),
    }
    
    _, _, err := db.client.From("tasks").
        Update(updates, "", "").
        Eq("id", taskID.String()).
        ExecuteTo(nil)
    
    if err != nil {
        return fmt.Errorf("failed to unassign task: %w", err)
    }
    
    return nil
}

func (db *DB) GetTasksWithDeadlines(ctx context.Context, userID uuid.UUID, daysBefore int) ([]models.Task, error) {
    deadline := time.Now().AddDate(0, 0, daysBefore)
    
    var tasks []models.Task
    _, _, err := db.client.From("tasks").
        Select("*, boards(title)").
        Eq("assignee_id", userID.String()).
        Lte("deadline", deadline.Format(time.RFC3339)).
        Eq("completed", false).
        ExecuteTo(&tasks)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get tasks with deadlines: %w", err)
    }
    
    return tasks, nil
}