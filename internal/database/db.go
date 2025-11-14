package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"

	"sudo/internal/models"
	"sudo/internal/security"
)

type DB struct {
	client *supabase.Client
	crypto *security.CryptoService
}

func NewDB() *DB {
	url := os.Getenv("SUPABASE_URL")
	key := os.Getenv("SUPABASE_SERVICE_KEY")

	if url == "" || key == "" {
		log.Fatal("SUPABASE_URL and SUPABASE_SERVICE_KEY must be set")
	}

	// Configure client with explicit schema
	opts := &supabase.ClientOptions{
		Schema: "public", // Explicitly set schema to public
	}

	client, err := supabase.NewClient(url, key, opts)
	if err != nil {
		log.Fatal("Failed to initialize Supabase client:", err)
	}

	// Initialize crypto service
	crypto, err := security.NewCryptoService()
	if err != nil {
		log.Fatal("Failed to initialize crypto service:", err)
	}

	return &DB{
		client: client,
		crypto: crypto,
	}
}

// User operations
func (db *DB) CreateUser(ctx context.Context, email, name string) (*models.User, error) {
	// Encrypt email before storing
	encryptedEmail, err := db.crypto.EncryptEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt email: %w", err)
	}

	// Use map instead of struct to avoid UUID issues
	userData := map[string]interface{}{
		"email": encryptedEmail,
	}

	// Only add name if it's not empty
	if strings.TrimSpace(name) != "" {
		userData["name"] = name
	}

	var result []models.User
	_, err = db.client.From("users").Insert(userData, false, "", "", "").ExecuteTo(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if len(result) > 0 {
		// Decrypt email for return value
		user := result[0]
		user.DecryptedEmail = email // We already have the plaintext
		log.Printf("User created with ID: %s", user.ID.String())
		return &user, nil
	}

	return nil, fmt.Errorf("failed to get created user data")
}

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	// Encrypt email to search for it (deterministic encryption allows this)
	encryptedEmail, err := db.crypto.EncryptEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt email for search: %w", err)
	}

	var users []models.User
	_, err = db.client.From("users").Select("*", "", false).Eq("email", encryptedEmail).ExecuteTo(&users)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	// Decrypt email for return value
	user := users[0]
	user.DecryptedEmail = email // We already have the plaintext

	return &user, nil
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

	// Decrypt email for return value
	user := users[0]
	if user.Email != "" {
		decryptedEmail, err := db.crypto.DecryptEmail(user.Email)
		if err != nil {
			log.Printf("Warning: Failed to decrypt email for user %s: %v", userID, err)
			// Don't fail the entire operation, just leave email encrypted
		} else {
			user.DecryptedEmail = decryptedEmail
		}
	}

	return &user, nil
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
		boardID := result[0].ID

		// Add the owner as a board member with 'owner' role
		// NOTE: This is redundant with the database trigger (trg_ensure_board_owner_membership)
		// but kept for defensive programming. The trigger should fire first, and this will
		// silently handle duplicates via ON CONFLICT (once Upsert is implemented).
		err := db.AddBoardMember(ctx, boardID, ownerID, "owner")
		if err != nil {
			log.Printf("Warning: Failed to explicitly add owner as board member (trigger should have handled this): %v", err)
			// Continue anyway - the database trigger should have added the owner
		}

		// Create default columns
		defaultColumns := []string{"To Do", "In Progress", "Review", "Done"}
		for i, colTitle := range defaultColumns {
			_, err := db.CreateColumn(ctx, boardID, colTitle, i)
			if err != nil {
				log.Printf("Failed to create default column %s: %v", colTitle, err)
			}
		}

		return &result[0], nil
	}

	return nil, fmt.Errorf("failed to get created board data")
}

func (db *DB) GetUserBoards(ctx context.Context, userID uuid.UUID) ([]models.Board, error) {
	var ownedBoards []models.Board
	var memberBoards []models.Board

	// Get boards where user is owner
	_, err := db.client.From("boards").
		Select("*", "", false).
		Eq("owner_id", userID.String()).
		Order("created_at", nil).
		ExecuteTo(&ownedBoards)

	if err != nil {
		return nil, fmt.Errorf("failed to get owned boards: %w", err)
	}

	// Get board IDs where user is a member
	var memberships []models.BoardMember
	_, err = db.client.From("board_members").
		Select("board_id", "", false).
		Eq("user_id", userID.String()).
		ExecuteTo(&memberships)

	if err != nil {
		return nil, fmt.Errorf("failed to get board memberships: %w", err)
	}

	// Create a map of owned board IDs to avoid duplicates
	ownedBoardIDs := make(map[string]bool)
	for _, board := range ownedBoards {
		ownedBoardIDs[board.ID.String()] = true
	}

	// Get board details for each membership (excluding owned boards)
	for _, membership := range memberships {
		if !ownedBoardIDs[membership.BoardID.String()] {
			var board []models.Board
			_, err = db.client.From("boards").
				Select("*", "", false).
				Eq("id", membership.BoardID.String()).
				ExecuteTo(&board)

			if err != nil {
				log.Printf("Failed to get member board %s: %v", membership.BoardID.String(), err)
				continue
			}

			if len(board) > 0 {
				memberBoards = append(memberBoards, board[0])
			}
		}
	}

	// Combine owned and member boards
	allBoards := append(ownedBoards, memberBoards...)

	return allBoards, nil
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

	// Get board members - CRITICAL for assignee dropdowns in task forms
	members, err := db.GetBoardMembers(ctx, boardID)
	if err != nil {
		log.Printf("Warning: Failed to get board members for board %s: %v", boardID.String(), err)
		// Create empty slice instead of nil to prevent template errors
		members = []models.BoardMember{}
	}

	// Ensure board owner is always in the members list (defensive programming)
	ownerInList := false
	for _, member := range members {
		if member.User != nil && member.User.ID == board.OwnerID {
			ownerInList = true
			break
		}
	}

	if !ownerInList {
		// Get owner user data
		owner, err := db.GetUserByID(ctx, board.OwnerID)
		if err != nil {
			log.Printf("Warning: Failed to get owner user data: %v", err)
			// Create placeholder owner
			owner = &models.User{
				ID:   board.OwnerID,
				Name: "Board Owner",
			}
		}

		// Prepend owner to members list
		members = append([]models.BoardMember{{
			BoardID: boardID,
			UserID:  board.OwnerID,
			Role:    "owner",
			User:    owner,
		}}, members...)
	}

	board.Members = members

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
		Select("id, parent_board_id", "", false).
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
	isMember, err := db.IsBoardMember(ctx, boardID, userID)
	if err != nil {
		return false, err
	}

	if isMember {
		return true, nil
	}

	// If this is a sub-board, check if user has access to the parent board
	_, err = db.client.From("boards").
		Select("id, parent_board_id", "", false).
		Eq("id", boardID.String()).
		ExecuteTo(&boards)

	if err != nil {
		return false, fmt.Errorf("failed to get board info: %w", err)
	}

	if len(boards) > 0 && boards[0].ParentBoardID != nil {
		// Recursively check access to parent board
		return db.HasBoardAccess(ctx, userID, *boards[0].ParentBoardID)
	}

	return false, nil
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
	member := map[string]interface{}{
		"board_id": boardID.String(),
		"user_id":  userID.String(),
		"role":     role,
	}

	// Use Upsert to handle ON CONFLICT - if member already exists, update their role
	// This prevents errors when the database trigger has already added the owner,
	// or when re-inviting a member who was previously removed
	_, err := db.client.From("board_members").Upsert(member, "board_id,user_id", "", "").ExecuteTo(nil)
	if err != nil {
		return fmt.Errorf("failed to add/update board member: %w", err)
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

func (db *DB) GetBoardMembers(ctx context.Context, boardID uuid.UUID) ([]models.BoardMember, error) {
	// First get all board members with their user info
	var members []models.BoardMember
	_, err := db.client.From("board_members").
		Select("id, user_id, board_id, role, joined_at", "", false).
		Eq("board_id", boardID.String()).
		ExecuteTo(&members)

	if err != nil {
		return nil, fmt.Errorf("failed to get board members: %w", err)
	}

	// Populate user details for each member
	for i := range members {
		user, err := db.GetUserByID(context.Background(), members[i].UserID)
		if err != nil {
			log.Printf("Failed to get user %s: %v", members[i].UserID, err)
			// Create placeholder user instead of skipping - ensures member remains assignable
			members[i].User = &models.User{
				ID:   members[i].UserID,
				Name: "Unknown User",
			}
			continue
		}
		members[i].User = user
	}

	return members, nil
}

// GetBoardMembersWithRoles returns board members with their roles in API format
func (db *DB) GetBoardMembersWithRoles(ctx context.Context, boardID uuid.UUID) ([]map[string]interface{}, error) {
	var members []models.BoardMember
	_, err := db.client.From("board_members").
		Select("user_id, role", "", false).
		Eq("board_id", boardID.String()).
		ExecuteTo(&members)

	if err != nil {
		return nil, fmt.Errorf("failed to get board members: %w", err)
	}

	// Get user details for each member and format for API
	var membersData []map[string]interface{}
	for _, member := range members {
		user, err := db.GetUserByID(ctx, member.UserID)
		if err != nil {
			log.Printf("Failed to get user %s: %v", member.UserID, err)
			continue
		}

		membersData = append(membersData, map[string]interface{}{
			"user_id":   member.UserID.String(),
			"user_name": user.GetDisplayName(),
			"role":      member.Role,
		})
	}

	return membersData, nil
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

	task := &tasks[0]

	// Load multiple assignees
	assignees, err := db.GetTaskAssignees(ctx, task.ID)
	if err != nil {
		log.Printf("Warning: Failed to get assignees for task %s: %v", task.ID.String(), err)
	} else {
		task.Assignees = assignees
	}

	// Keep backward compatibility with single assignee
	if task.AssignedTo != nil {
		assignee, err := db.GetUserByID(ctx, *task.AssignedTo)
		if err != nil {
			log.Printf("Warning: Failed to get assignee for task %s: %v", task.ID.String(), err)
		} else {
			task.Assignee = assignee
		}
	}

	return task, nil
}

func (db *DB) GetTaskByNestedBoardID(ctx context.Context, nestedBoardID uuid.UUID) (*models.Task, error) {
	var tasks []models.Task
	_, err := db.client.From("tasks").
		Select("*", "", false).
		Eq("nested_board_id", nestedBoardID.String()).
		ExecuteTo(&tasks)

	if err != nil {
		return nil, fmt.Errorf("failed to get task by nested board ID: %w", err)
	}

	if len(tasks) == 0 {
		return nil, nil // No task found (this is okay, board might not have a parent task)
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

	// Populate assignee information for each task
	for i := range tasks {
		// Load multiple assignees
		assignees, err := db.GetTaskAssignees(ctx, tasks[i].ID)
		if err != nil {
			log.Printf("Warning: Failed to get assignees for task %s: %v", tasks[i].ID.String(), err)
		} else {
			tasks[i].Assignees = assignees
		}

		// Keep backward compatibility with single assignee
		if tasks[i].AssignedTo != nil {
			assignee, err := db.GetUserByID(ctx, *tasks[i].AssignedTo)
			if err != nil {
				log.Printf("Warning: Failed to get assignee %s for task %s: %v",
					tasks[i].AssignedTo.String(), tasks[i].ID.String(), err)
				// Continue without assignee data rather than failing
				continue
			}
			tasks[i].Assignee = assignee
		}
	}

	return tasks, nil
}

func (db *DB) UpdateTask(ctx context.Context, taskID uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	// First get current version, then increment it
	var currentTask []models.Task
	_, err := db.client.From("tasks").
		Select("version", "", false).
		Eq("id", taskID.String()).
		ExecuteTo(&currentTask)

	if err == nil && len(currentTask) > 0 {
		updates["version"] = currentTask[0].Version + 1
	}

	_, err = db.client.From("tasks").
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
		"column_id":  newColumnID,
		"position":   newPosition,
		"updated_at": time.Now(),
	}

	// Get current version and increment it
	var currentTask []models.Task
	_, err := db.client.From("tasks").
		Select("version", "", false).
		Eq("id", taskID.String()).
		ExecuteTo(&currentTask)

	if err == nil && len(currentTask) > 0 {
		updates["version"] = currentTask[0].Version + 1
	}

	_, err = db.client.From("tasks").
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

	// Get current version and increment it
	var currentTask []models.Task
	_, err := db.client.From("tasks").
		Select("version", "", false).
		Eq("id", taskID.String()).
		ExecuteTo(&currentTask)

	if err == nil && len(currentTask) > 0 {
		updates["version"] = currentTask[0].Version + 1
	}

	_, err = db.client.From("tasks").
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

	// Get current version and increment it
	var currentTask []models.Task
	_, err := db.client.From("tasks").
		Select("version", "", false).
		Eq("id", taskID.String()).
		ExecuteTo(&currentTask)

	if err == nil && len(currentTask) > 0 {
		updates["version"] = currentTask[0].Version + 1
	}

	_, err = db.client.From("tasks").
		Update(updates, "", "").
		Eq("id", taskID.String()).
		ExecuteTo(nil)

	if err != nil {
		return fmt.Errorf("failed to unassign task: %w", err)
	}

	return nil
}

// Task Assignees operations (multiple assignees support)
func (db *DB) AddTaskAssignee(ctx context.Context, taskID, userID, assignedBy uuid.UUID) error {
	assigneeData := map[string]interface{}{
		"task_id":     taskID.String(),
		"user_id":     userID.String(),
		"assigned_by": assignedBy.String(),
		"completed":   false,
	}

	log.Printf("DEBUG AddTaskAssignee: Adding assignee %s to task %s", userID.String(), taskID.String())

	_, err := db.client.From("task_assignees").Insert(assigneeData, false, "", "", "").ExecuteTo(nil)
	if err != nil {
		log.Printf("ERROR AddTaskAssignee: Failed to add assignee: %v", err)
		return fmt.Errorf("failed to add task assignee: %w", err)
	}

	log.Printf("DEBUG AddTaskAssignee: Successfully added assignee %s to task %s", userID.String(), taskID.String())
	return nil
}

func (db *DB) RemoveTaskAssignee(ctx context.Context, taskID, userID uuid.UUID) error {
	_, err := db.client.From("task_assignees").
		Delete("", "").
		Eq("task_id", taskID.String()).
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		return fmt.Errorf("failed to remove task assignee: %w", err)
	}

	return nil
}

func (db *DB) GetTaskAssignees(ctx context.Context, taskID uuid.UUID) ([]models.TaskAssignee, error) {
	var assignees []models.TaskAssignee
	_, err := db.client.From("task_assignees").
		Select("*", "", false).
		Eq("task_id", taskID.String()).
		Order("assigned_at", &postgrest.OrderOpts{Ascending: true}).
		ExecuteTo(&assignees)

	if err != nil {
		return nil, fmt.Errorf("failed to get task assignees: %w", err)
	}

	log.Printf("DEBUG GetTaskAssignees: Found %d raw assignee records for task %s", len(assignees), taskID.String())

	// Load user info for each assignee
	for i := range assignees {
		log.Printf("DEBUG GetTaskAssignees: Loading user %s for assignee %d", assignees[i].UserID.String(), i)
		user, err := db.GetUserByID(ctx, assignees[i].UserID)
		if err != nil {
			log.Printf("Warning: Failed to get user %s for task assignee: %v", assignees[i].UserID, err)
			continue
		}
		assignees[i].User = user
		log.Printf("DEBUG GetTaskAssignees: Loaded user %s", user.GetDisplayName())
	}

	log.Printf("DEBUG GetTaskAssignees: Returning %d assignees with user data", len(assignees))
	return assignees, nil
}

func (db *DB) UpdateTaskAssigneeCompletion(ctx context.Context, taskID, userID uuid.UUID, completed bool) error {
	updates := map[string]interface{}{
		"completed":  completed,
		"updated_at": time.Now(),
	}

	if completed {
		updates["completed_at"] = time.Now()
	} else {
		updates["completed_at"] = nil
	}

	_, err := db.client.From("task_assignees").
		Update(updates, "", "").
		Eq("task_id", taskID.String()).
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		return fmt.Errorf("failed to update task assignee completion: %w", err)
	}

	return nil
}

// OTP operations
func (db *DB) CreateOTP(ctx context.Context, email, token string, expiresAt time.Time) error {
	log.Printf("CreateOTP called with email=%s, token=%s", email, token)

	// Encrypt email for storage
	encryptedEmail, err := db.crypto.EncryptEmail(email)
	if err != nil {
		log.Printf("Failed to encrypt email: %v", err)
		return fmt.Errorf("failed to encrypt email: %w", err)
	}
	log.Printf("Email encrypted successfully")

	// Hash OTP for storage
	hashedToken, err := db.crypto.HashOTP(token)
	if err != nil {
		log.Printf("Failed to hash OTP: %v", err)
		return fmt.Errorf("failed to hash OTP: %w", err)
	}
	log.Printf("OTP hashed successfully")

	// Create a map instead of struct to avoid UUID issues
	otp := map[string]interface{}{
		"email":      encryptedEmail,
		"token":      hashedToken,
		"expires_at": expiresAt.UTC(),
	}

	log.Printf("Attempting to insert OTP to database")
	_, err = db.client.From("otp_tokens").Insert(otp, false, "", "", "").ExecuteTo(nil)
	if err != nil {
		log.Printf("Database insert failed: %v", err)
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	log.Printf("OTP created successfully")
	return nil
}

func (db *DB) ValidateOTP(ctx context.Context, email, token string) (*models.User, error) {
	log.Printf("ValidateOTP called with email=%s, token=%s", email, token)

	// Encrypt email to search for matching OTPs
	encryptedEmail, err := db.crypto.EncryptEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt email for search: %w", err)
	}

	var otps []models.OTPToken

	// Get all OTPs for this email (we'll verify the token hash manually)
	_, err = db.client.From("otp_tokens").
		Select("*", "", false).
		Eq("email", encryptedEmail).
		ExecuteTo(&otps)

	if err != nil {
		log.Printf("Database query failed: %v", err)
		return nil, fmt.Errorf("failed to validate OTP: %w", err)
	}

	log.Printf("Found %d OTP records for email=%s", len(otps), email)

	if len(otps) == 0 {
		log.Printf("No OTP found for email=%s", email)
		return nil, fmt.Errorf("no OTP found for email")
	}

	// Check each OTP to find a valid one that matches the provided token
	validOTP := (*models.OTPToken)(nil)
	now := time.Now().UTC()

	for i := range otps {
		log.Printf("Checking OTP[%d]: ID=%s, Used=%v, ExpiresAt=%v, Now=%v",
			i, otps[i].ID.String(), otps[i].Used, otps[i].ExpiresAt, now)

		// Skip if already used or expired
		if otps[i].Used || otps[i].ExpiresAt.Before(now) {
			continue
		}

		// Verify the token hash
		isValid, verifyErr := db.crypto.VerifyOTP(token, otps[i].Token)
		if verifyErr != nil {
			log.Printf("Failed to verify OTP hash: %v", verifyErr)
			continue
		}

		if isValid {
			validOTP = &otps[i]
			log.Printf("Found valid OTP: %s", validOTP.ID.String())
			break
		}
	}

	if validOTP == nil {
		log.Printf("No valid OTP found - all are either used, expired, or don't match")
		return nil, fmt.Errorf("OTP is invalid, used, or expired")
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
		log.Printf("New user created: %s (ID: %s)", user.DecryptedEmail, user.ID.String())
	} else {
		log.Printf("Existing user found: %s (ID: %s)", user.DecryptedEmail, user.ID.String())

		// Check if user has zero UUID and fix it
		if user.ID.String() == "00000000-0000-0000-0000-000000000000" {
			log.Printf("User has zero UUID, recreating user record")
			// Delete the broken user record by encrypted email
			_, deleteErr := db.client.From("users").
				Delete("", "").
				Eq("email", encryptedEmail).
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

	log.Printf("OTP validation completed successfully for user: %s", user.DecryptedEmail)
	return user, nil
}

// User profile operations
func (db *DB) UpdateUserProfile(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := db.client.From("users").
		Update(updates, "", "").
		Eq("id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	return nil
}

// Contact management operations
// GetUserContacts returns all users that have been invited to any board by the current user
func (db *DB) GetUserContacts(ctx context.Context, userID uuid.UUID) ([]map[string]interface{}, error) {
	// Get all boards owned by the user
	var ownedBoards []models.Board
	_, err := db.client.From("boards").
		Select("id", "", false).
		Eq("owner_id", userID.String()).
		ExecuteTo(&ownedBoards)

	if err != nil {
		return nil, fmt.Errorf("failed to get owned boards: %w", err)
	}

	if len(ownedBoards) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Collect all unique contacts across all owned boards
	contactsMap := make(map[string]map[string]interface{})

	for _, board := range ownedBoards {
		var members []models.BoardMember
		_, err := db.client.From("board_members").
			Select("user_id, role, invited_by, joined_at", "", false).
			Eq("board_id", board.ID.String()).
			ExecuteTo(&members)

		if err != nil {
			log.Printf("Failed to get members for board %s: %v", board.ID, err)
			continue
		}

		// Process each member
		for _, member := range members {
			// Skip if it's the owner themselves
			if member.UserID == userID {
				continue
			}

			// Get user details
			user, err := db.GetUserByID(ctx, member.UserID)
			if err != nil {
				log.Printf("Failed to get user %s: %v", member.UserID, err)
				continue
			}

			// Check if contact already exists in map
			contactKey := member.UserID.String()
			if existing, ok := contactsMap[contactKey]; ok {
				// Update board count
				boardsCount, ok := existing["boards_count"].(int)
				if !ok {
					boardsCount = 0
				}
				boardsCount++
				existing["boards_count"] = boardsCount

				// Add board info to boards list
				boards, ok := existing["boards"].([]map[string]interface{})
				if !ok {
					boards = []map[string]interface{}{}
				}
				boards = append(boards, map[string]interface{}{
					"board_id":    board.ID.String(),
					"board_title": "", // We'll fetch this if needed
					"role":        member.Role,
				})
				existing["boards"] = boards
			} else {
				// New contact
				contactsMap[contactKey] = map[string]interface{}{
					"user_id":      member.UserID.String(),
					"email":        user.DecryptedEmail,
					"name":         user.Name,
					"avatar_url":   user.AvatarURL,
					"boards_count": 1,
					"added_at":     member.JoinedAt,
					"boards": []map[string]interface{}{
						{
							"board_id":    board.ID.String(),
							"board_title": "",
							"role":        member.Role,
						},
					},
				}
			}
		}
	}

	// Convert map to slice
	contacts := make([]map[string]interface{}, 0, len(contactsMap))
	for _, contact := range contactsMap {
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// GetContactBoards returns all boards where a specific contact is a member, owned by the current user
func (db *DB) GetContactBoards(ctx context.Context, userID, contactID uuid.UUID) ([]map[string]interface{}, error) {
	// Get all boards owned by the user
	var ownedBoards []models.Board
	_, err := db.client.From("boards").
		Select("id, title", "", false).
		Eq("owner_id", userID.String()).
		ExecuteTo(&ownedBoards)

	if err != nil {
		return nil, fmt.Errorf("failed to get owned boards: %w", err)
	}

	var contactBoards []map[string]interface{}

	// Check which boards the contact is a member of
	for _, board := range ownedBoards {
		var members []models.BoardMember
		_, err := db.client.From("board_members").
			Select("role", "", false).
			Eq("board_id", board.ID.String()).
			Eq("user_id", contactID.String()).
			ExecuteTo(&members)

		if err != nil {
			log.Printf("Failed to check membership for board %s: %v", board.ID, err)
			continue
		}

		if len(members) > 0 {
			contactBoards = append(contactBoards, map[string]interface{}{
				"board_id":    board.ID.String(),
				"board_title": board.Title,
				"role":        members[0].Role,
			})
		}
	}

	return contactBoards, nil
}

// RemoveContactFromAllBoards removes a contact from all boards owned by the current user
func (db *DB) RemoveContactFromAllBoards(ctx context.Context, userID, contactID uuid.UUID) error {
	// Get all boards owned by the user
	var ownedBoards []models.Board
	_, err := db.client.From("boards").
		Select("id", "", false).
		Eq("owner_id", userID.String()).
		ExecuteTo(&ownedBoards)

	if err != nil {
		return fmt.Errorf("failed to get owned boards: %w", err)
	}

	// Remove contact from each board
	for _, board := range ownedBoards {
		err := db.RemoveBoardMember(ctx, board.ID, contactID)
		if err != nil {
			log.Printf("Failed to remove contact from board %s: %v", board.ID, err)
			// Continue with other boards even if one fails
		}
	}

	return nil
}

// DeleteUserAccount permanently deletes a user and ALL associated data
// This is a destructive operation that cannot be undone
func (db *DB) DeleteUserAccount(ctx context.Context, userID uuid.UUID) error {
	log.Printf("Starting account deletion for user %s", userID.String())

	// Step 1: Get all boards owned by the user (need to delete these first)
	var ownedBoards []models.Board
	_, err := db.client.From("boards").
		Select("id", "", false).
		Eq("owner_id", userID.String()).
		ExecuteTo(&ownedBoards)

	if err != nil {
		log.Printf("Failed to get owned boards for deletion: %v", err)
		// Continue anyway
	}

	// Step 2: Delete all owned boards (this will CASCADE delete columns, tasks, etc.)
	for _, board := range ownedBoards {
		log.Printf("Deleting owned board %s", board.ID.String())
		err = db.DeleteBoard(ctx, board.ID)
		if err != nil {
			log.Printf("Failed to delete board %s: %v", board.ID.String(), err)
			// Continue deleting other boards
		}
	}

	// Step 3: Remove user from all board memberships where they're not the owner
	_, err = db.client.From("board_members").
		Delete("", "").
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete board memberships: %v", err)
	}

	// Step 4: Delete all OTP tokens for this user
	// First get the encrypted email
	user, err := db.GetUserByID(ctx, userID)
	if err == nil && user != nil && user.Email != "" {
		_, err = db.client.From("otp_tokens").
			Delete("", "").
			Eq("email", user.Email).
			ExecuteTo(nil)

		if err != nil {
			log.Printf("Failed to delete OTP tokens: %v", err)
		}
	}

	// Step 5: Delete all user presence records
	_, err = db.client.From("user_presence").
		Delete("", "").
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete user presence: %v", err)
	}

	// Step 6: Delete all realtime sessions
	_, err = db.client.From("realtime_sessions").
		Delete("", "").
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete realtime sessions: %v", err)
	}

	// Step 7: Delete all activity logs
	_, err = db.client.From("activity_log").
		Delete("", "").
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete activity logs: %v", err)
	}

	// Step 8: Delete all comments by this user
	_, err = db.client.From("comments").
		Delete("", "").
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete comments: %v", err)
	}

	// Step 9: Delete all task assignees for this user
	_, err = db.client.From("task_assignees").
		Delete("", "").
		Eq("user_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete task assignees: %v", err)
	}

	// Step 10: Delete all proposed edits by this user
	_, err = db.client.From("proposed_edits").
		Delete("", "").
		Eq("proposed_by", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete proposed edits: %v", err)
	}

	// Step 11: Delete all approval notifications for this user
	_, err = db.client.From("approval_notifications").
		Delete("", "").
		Eq("recipient_id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		log.Printf("Failed to delete approval notifications: %v", err)
	}

	// Step 12: Finally, delete the user record itself
	// This will CASCADE delete any remaining references
	_, err = db.client.From("users").
		Delete("", "").
		Eq("id", userID.String()).
		ExecuteTo(nil)

	if err != nil {
		return fmt.Errorf("failed to delete user account: %w", err)
	}

	log.Printf("Successfully deleted all data for user %s", userID.String())
	return nil
}
