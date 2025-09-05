package models

import (
    "fmt"
    "strings"
    "time"
    
    "github.com/google/uuid"
)

type User struct {
    ID        uuid.UUID `json:"id" db:"id"`
    Email     string    `json:"email" db:"email"`
    Name      string    `json:"name" db:"name"`
    AvatarURL string    `json:"avatar_url" db:"avatar_url"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Board struct {
    ID            uuid.UUID  `json:"id" db:"id"`
    Title         string     `json:"title" db:"title"`
    Description   string     `json:"description" db:"description"`
    OwnerID       uuid.UUID  `json:"owner_id" db:"owner_id"`
    ParentBoardID *uuid.UUID `json:"parent_board_id" db:"parent_board_id"`
    Settings      map[string]interface{} `json:"settings" db:"settings"`
    CreatedAt     time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
    
    // Relationships
    Owner   *User    `json:"owner,omitempty"`
    Columns []Column `json:"columns,omitempty"`
    Members []User   `json:"members,omitempty"`
}

type BoardMember struct {
    ID       uuid.UUID `json:"id" db:"id"`
    BoardID  uuid.UUID `json:"board_id" db:"board_id"`
    UserID   uuid.UUID `json:"user_id" db:"user_id"`
    Role     string    `json:"role" db:"role"`
    JoinedAt time.Time `json:"joined_at" db:"joined_at"`
    
    // Relationships
    Board *Board `json:"board,omitempty"`
    User  *User  `json:"user,omitempty"`
}

type Column struct {
    ID       uuid.UUID `json:"id" db:"id"`
    BoardID  uuid.UUID `json:"board_id" db:"board_id"`
    Title    string    `json:"title" db:"title"`
    Position int       `json:"position" db:"position"`
    Settings map[string]interface{} `json:"settings" db:"settings"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    
    // Relationships
    Board *Board `json:"board,omitempty"`
    Tasks []Task `json:"tasks,omitempty"`
}

type Task struct {
    ID            uuid.UUID  `json:"id" db:"id"`
    Title         string     `json:"title" db:"title"`
    Description   string     `json:"description" db:"description"`
    ColumnID      uuid.UUID  `json:"column_id" db:"column_id"`
    BoardID       uuid.UUID  `json:"board_id" db:"board_id"`
    AssignedTo    *uuid.UUID `json:"assigned_to" db:"assigned_to"`
    Priority      string     `json:"priority" db:"priority"`
    Position      int        `json:"position" db:"position"`
    Deadline      *time.Time `json:"deadline" db:"deadline"`
    Completed     bool       `json:"completed" db:"completed"`
    CompletedAt   *time.Time `json:"completed_at" db:"completed_at"`
    Tags          []string   `json:"tags" db:"tags"`
    Attachments   []map[string]interface{} `json:"attachments" db:"attachments"`
    NestedBoardID *uuid.UUID `json:"nested_board_id" db:"nested_board_id"`
    CreatedAt     time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
    
    // Relationships
    Column      *Column   `json:"column,omitempty"`
    Board       *Board    `json:"board,omitempty"`
    Assignee    *User     `json:"assignee,omitempty"`
    Comments    []Comment `json:"comments,omitempty"`
    NestedBoard *Board    `json:"nested_board,omitempty"`
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
    ID          uuid.UUID              `json:"id" db:"id"`
    UserID      uuid.UUID              `json:"user_id" db:"user_id"`
    BoardID     uuid.UUID              `json:"board_id" db:"board_id"`
    TaskID      *uuid.UUID             `json:"task_id" db:"task_id"`
    Action      string                 `json:"action" db:"action"`
    Description string                 `json:"description" db:"description"`
    Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
    CreatedAt   time.Time              `json:"created_at" db:"created_at"`
    
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
            return strings.ToUpper(string(u.Email[0]))
        }
        return "?"
    }
    
    initials := ""
    words := splitName(u.Name)
    for i, word := range words {
        if i < 2 && len(word) > 0 {
            initials += strings.ToUpper(string(word[0]))
        }
    }
    
    if initials == "" {
        return "?"
    }
    
    return initials
}

func (u *User) GetDisplayName() string {
    if u.Name != "" {
        return u.Name
    }
    return u.Email
}

func (t *Task) GetPriorityColor() string {
    switch t.Priority {
    case PriorityUrgent:
        return "red"
    case PriorityHigh:
        return "orange"
    case PriorityMedium:
        return "yellow"
    case PriorityLow:
        return "green"
    default:
        return "gray"
    }
}

func (t *Task) IsOverdue() bool {
    if t.Deadline == nil || t.Completed {
        return false
    }
    return time.Now().After(*t.Deadline)
}

func (t *Task) GetDeadlineStatus() string {
    if t.Deadline == nil {
        return ""
    }
    
    if t.Completed {
        return "completed"
    }
    
    now := time.Now()
    diff := t.Deadline.Sub(now)
    
    if diff < 0 {
        return "overdue"
    } else if diff < 24*time.Hour {
        return "due-soon"
    } else if diff < 7*24*time.Hour {
        return "due-this-week"
    }
    
    return "due-later"
}

func (t *Task) HasNestedBoard() bool {
    return t.NestedBoardID != nil
}

func (b *Board) IsSubBoard() bool {
    return b.ParentBoardID != nil
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

func (bm *BoardMember) CanEdit() bool {
    return bm.Role == RoleOwner || bm.Role == RoleAdmin
}

func (bm *BoardMember) CanDelete() bool {
    return bm.Role == RoleOwner
}

func (bm *BoardMember) CanInvite() bool {
    return bm.Role == RoleOwner || bm.Role == RoleAdmin
}

// Helper functions
func splitName(name string) []string {
    return strings.Fields(strings.TrimSpace(name))
}

func ValidatePriority(priority string) bool {
    validPriorities := []string{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}
    for _, p := range validPriorities {
        if priority == p {
            return true
        }
    }
    return false
}

func ValidateRole(role string) bool {
    validRoles := []string{RoleOwner, RoleAdmin, RoleMember}
    for _, r := range validRoles {
        if role == r {
            return true
        }
    }
    return false
}

func GetPriorityList() []string {
    return []string{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent}
}

func GetRoleList() []string {
    return []string{RoleMember, RoleAdmin, RoleOwner}
}

// Format time helpers
func FormatRelativeTime(t time.Time) string {
    now := time.Now()
    diff := now.Sub(t)
    
    if diff < time.Minute {
        return "just now"
    } else if diff < time.Hour {
        minutes := int(diff.Minutes())
        if minutes == 1 {
            return "1 minute ago"
        }
        return fmt.Sprintf("%d minutes ago", minutes)
    } else if diff < 24*time.Hour {
        hours := int(diff.Hours())
        if hours == 1 {
            return "1 hour ago"
        }
        return fmt.Sprintf("%d hours ago", hours)
    } else if diff < 7*24*time.Hour {
        days := int(diff.Hours() / 24)
        if days == 1 {
            return "1 day ago"
        }
        return fmt.Sprintf("%d days ago", days)
    } else {
        return t.Format("Jan 2, 2006")
    }
}

func FormatDeadline(deadline *time.Time) string {
    if deadline == nil {
        return ""
    }
    
    now := time.Now()
    diff := deadline.Sub(now)
    
    if diff < 0 {
        return "Overdue"
    } else if diff < 24*time.Hour {
        hours := int(diff.Hours())
        if hours < 1 {
            return "Due very soon"
        }
        return fmt.Sprintf("Due in %d hours", hours)
    } else if diff < 7*24*time.Hour {
        days := int(diff.Hours() / 24)
        return fmt.Sprintf("Due in %d days", days)
    } else {
        return deadline.Format("Due Jan 2")
    }
}