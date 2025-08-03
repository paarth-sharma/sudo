package models

import (
    "fmt"
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    Name      string    `json:"name" db:"name"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Board struct {
    ID            uuid.UUID  `json:"id" db:"id"`
    Title         string     `json:"title" db:"title"`
    Description   string     `json:"description" db:"description"`
    OwnerID       uuid.UUID  `json:"owner_id" db:"owner_id"`
    ParentBoardID *uuid.UUID `json:"parent_board_id" db:"parent_board_id"`
    CreatedAt     time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
    
    // Relationships
    Columns []Column `json:"columns,omitempty"`
    Owner   *User    `json:"owner,omitempty"`
    Members []User   `json:"members,omitempty"`
}

type Column struct {
    ID        uuid.UUID `json:"id" db:"id"`
    BoardID   uuid.UUID `json:"board_id" db:"board_id"`
    Title     string    `json:"title" db:"title"`
    Position  int       `json:"position" db:"position"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    
    // Relationships
    Tasks []Task `json:"tasks,omitempty"`
    Board *Board `json:"board,omitempty"`
}

type Task struct {
    ID          uuid.UUID  `json:"id" db:"id"`
    Title       string     `json:"title" db:"title"`
    Description string     `json:"description" db:"description"`
    ColumnID    uuid.UUID  `json:"column_id" db:"column_id"`
    BoardID     uuid.UUID  `json:"board_id" db:"board_id"`
    AssigneeID  *uuid.UUID `json:"assignee_id" db:"assignee_id"`
    Priority    string     `json:"priority" db:"priority"`
    Deadline    *time.Time `json:"deadline" db:"deadline"`
    Position    int        `json:"position" db:"position"`
    Completed   bool       `json:"completed" db:"completed"`
    CreatedAt   time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
    
    // Relationships
    Column   *Column `json:"column,omitempty"`
    Board    *Board  `json:"board,omitempty"`
    Assignee *User   `json:"assignee,omitempty"`
}

type BoardMember struct {
    ID        uuid.UUID `json:"id" db:"id"`
    BoardID   uuid.UUID `json:"board_id" db:"board_id"`
    UserID    uuid.UUID `json:"user_id" db:"user_id"`
    Role      string    `json:"role" db:"role"` // owner, admin, member
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    
    // Relationships
    Board *Board `json:"board,omitempty"`
    User  *User  `json:"user,omitempty"`
}

type OTPToken struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    Token     string    `json:"token" db:"token"`
    ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
    Used      bool      `json:"used" db:"used"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Comment struct {
    ID        uuid.UUID `json:"id" db:"id"`
    TaskID    uuid.UUID `json:"task_id" db:"task_id"`
    UserID    uuid.UUID `json:"user_id" db:"user_id"`
    Content   string    `json:"content" db:"content"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    
    // Relationships
    Task *Task `json:"task,omitempty"`
    User *User `json:"user,omitempty"`
}

type Activity struct {
    ID          uuid.UUID `json:"id" db:"id"`
    UserID      uuid.UUID `json:"user_id" db:"user_id"`
    BoardID     uuid.UUID `json:"board_id" db:"board_id"`
    TaskID      *uuid.UUID `json:"task_id" db:"task_id"`
    Action      string    `json:"action" db:"action"` // created, updated, moved, deleted, etc.
    Description string    `json:"description" db:"description"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
    
    // Relationships
    User  *User  `json:"user,omitempty"`
    Board *Board `json:"board,omitempty"`
    Task  *Task  `json:"task,omitempty"`
}

// Priority constants
const (
    PriorityLow    = "Low"
    PriorityMedium = "Medium"
    PriorityHigh   = "High"
    PriorityUrgent = "Urgent"
)

// Role constants
const (
    RoleOwner  = "owner"
    RoleAdmin  = "admin"
    RoleMember = "member"
)

// Activity action constants
const (
    ActionCreated     = "created"
    ActionUpdated     = "updated"
    ActionMoved       = "moved"
    ActionDeleted     = "deleted"
    ActionAssigned    = "assigned"
    ActionUnassigned  = "unassigned"
    ActionCompleted   = "completed"
    ActionReopened    = "reopened"
    ActionCommented   = "commented"
    ActionInvited     = "invited"
    ActionJoined      = "joined"
    ActionLeft        = "left"
)

// Helper methods
func (u *User) GetInitials() string {
    if u.Name == "" {
        if u.Email != "" {
            return string(u.Email[0])
        }
        return "?"
    }
    
    initials := ""
    words := splitName(u.Name)
    for i, word := range words {
        if i < 2 && len(word) > 0 {
            initials += string(word[0])
        }
    }
    
    if initials == "" {
        return "?"
    }
    
    return initials
}

func (t *Task) IsOverdue() bool {
    if t.Deadline == nil || t.Completed {
        return false
    }
    return t.Deadline.Before(time.Now())
}

func (t *Task) IsDueSoon() bool {
    if t.Deadline == nil || t.Completed {
        return false
    }
    return t.Deadline.Before(time.Now().Add(24 * time.Hour))
}

func (t *Task) GetPriorityWeight() int {
    switch t.Priority {
    case PriorityUrgent:
        return 4
    case PriorityHigh:
        return 3
    case PriorityMedium:
        return 2
    case PriorityLow:
        return 1
    default:
        return 2
    }
}

func (b *Board) GetTaskCount() int {
    count := 0
    for _, column := range b.Columns {
        count += len(column.Tasks)
    }
    return count
}

func (b *Board) GetCompletedTaskCount() int {
    count := 0
    for _, column := range b.Columns {
        for _, task := range column.Tasks {
            if task.Completed {
                count++
            }
        }
    }
    return count
}

func (c *Column) GetTaskCount() int {
    return len(c.Tasks)
}

func (c *Column) GetCompletedTaskCount() int {
    count := 0
    for _, task := range c.Tasks {
        if task.Completed {
            count++
        }
    }
    return count
}

// Helper functions
func splitName(name string) []string {
    var words []string
    var current string
    
    for _, r := range name {
        if r == ' ' || r == '\t' || r == '\n' {
            if current != "" {
                words = append(words, current)
                current = ""
            }
        } else {
            current += string(r)
        }
    }
    
    if current != "" {
        words = append(words, current)
    }
    
    return words
}

// Validation helpers
func (u *User) Validate() error {
    if u.Email == "" {
        return fmt.Errorf("email is required")
    }
    return nil
}

func (b *Board) Validate() error {
    if b.Title == "" {
        return fmt.Errorf("board title is required")
    }
    return nil
}

func (c *Column) Validate() error {
    if c.Title == "" {
        return fmt.Errorf("column title is required")
    }
    return nil
}

func (t *Task) Validate() error {
    if t.Title == "" {
        return fmt.Errorf("task title is required")
    }
    
    validPriorities := []string{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}
    valid := false
    for _, p := range validPriorities {
        if t.Priority == p {
            valid = true
            break
        }
    }
    
    if !valid {
        return fmt.Errorf("invalid priority: %s", t.Priority)
    }
    
    return nil
}