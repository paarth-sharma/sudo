package handlers

import (
	"context"
	"fmt"
	"net/http"

	"sudo/internal/database"
	"sudo/internal/models"
	"sudo/internal/realtime"
	"sudo/templates/pages"

	"github.com/a-h/templ"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SettingsHandler struct {
	db       *database.DB
	realtime *realtime.RealtimeService
}

func NewSettingsHandler(db *database.DB, rt *realtime.RealtimeService) *SettingsHandler {
	return &SettingsHandler{
		db:       db,
		realtime: rt,
	}
}

// Settings page - displays user settings with profile and contact management
func (h *SettingsHandler) SettingsPage(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// Get current user
	user, err := h.db.GetUserByID(context.Background(), userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get user: %v", err)
		return
	}

	// Get user's boards for invite functionality
	boards, err := h.db.GetUserBoards(context.Background(), userID)
	if err != nil {
		fmt.Printf("Failed to get user boards: %v\n", err)
		boards = []models.Board{} // Continue with empty list
	}

	// Get user contacts
	contacts, err := h.db.GetUserContacts(context.Background(), userID)
	if err != nil {
		fmt.Printf("Failed to get contacts: %v\n", err)
		contacts = []map[string]interface{}{} // Continue with empty list
	}

	component := pages.Settings(*user, boards, contacts)
	handler := templ.Handler(component)
	handler.ServeHTTP(c.Writer, c.Request)
}

// UpdateProfile handles profile updates including name and avatar
func (h *SettingsHandler) UpdateProfile(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	name := c.PostForm("name")
	avatarURL := c.PostForm("avatar_url")

	updates := map[string]interface{}{}

	if name != "" {
		updates["name"] = name
	}

	if avatarURL != "" {
		updates["avatar_url"] = avatarURL
	}

	if len(updates) == 0 {
		c.String(http.StatusBadRequest, "No updates provided")
		return
	}

	err = h.db.UpdateUserProfile(context.Background(), userID, updates)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to update profile: %v", err)
		return
	}

	c.Status(http.StatusOK)
}

// UploadAvatar handles avatar file uploads (base64 images)
func (h *SettingsHandler) UploadAvatar(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get the base64 image data from request body
	var requestData struct {
		ImageData string `json:"image_data"`
	}

	if bindErr := c.ShouldBindJSON(&requestData); bindErr != nil {
		fmt.Printf("Failed to bind JSON: %v\n", bindErr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate we have image data
	if requestData.ImageData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image data provided"})
		return
	}

	// Validate base64 data starts with data:image
	if len(requestData.ImageData) < 22 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image data too short"})
		return
	}

	// Check if it starts with data:image/
	if len(requestData.ImageData) >= 11 && requestData.ImageData[:11] != "data:image/" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image format - must be a data URL"})
		return
	}

	fmt.Printf("Uploading avatar for user %s, data length: %d bytes\n", userID, len(requestData.ImageData))

	// Check if the image is too large (database TEXT field limit)
	// Most databases limit TEXT to ~65KB, so we'll limit to 500KB for base64
	if len(requestData.ImageData) > 500000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image too large. Please use a smaller image or an image hosting service."})
		return
	}

	// Store the base64 data URL directly
	updates := map[string]interface{}{
		"avatar_url": requestData.ImageData,
	}

	err = h.db.UpdateUserProfile(context.Background(), userID, updates)
	if err != nil {
		fmt.Printf("Failed to update profile: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update profile: %v", err)})
		return
	}

	fmt.Printf("Avatar uploaded successfully for user %s\n", userID)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetContacts returns the list of contacts for the current user
func (h *SettingsHandler) GetContacts(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	contacts, err := h.db.GetUserContacts(context.Background(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contacts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"contacts": contacts})
}

// GetContactBoards returns all boards a contact is a member of (owned by current user)
func (h *SettingsHandler) GetContactBoards(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	contactIDStr := c.Param("contactId")
	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid contact ID"})
		return
	}

	boards, err := h.db.GetContactBoards(context.Background(), userID, contactID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get contact boards"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"boards": boards})
}

// RemoveContactFromBoard removes a contact from a specific board
func (h *SettingsHandler) RemoveContactFromBoard(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	contactIDStr := c.PostForm("contact_id")
	boardIDStr := c.PostForm("board_id")

	if contactIDStr == "" || boardIDStr == "" {
		c.String(http.StatusBadRequest, "Contact ID and Board ID are required")
		return
	}

	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid contact ID")
		return
	}

	boardID, err := uuid.Parse(boardIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid board ID")
		return
	}

	// Verify user owns this board
	isOwner, err := h.db.IsBoardOwner(context.Background(), userID, boardID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to check board ownership: %v", err)
		return
	}

	if !isOwner {
		c.String(http.StatusForbidden, "You don't own this board")
		return
	}

	// Remove the contact from the board
	err = h.db.RemoveBoardMember(context.Background(), boardID, contactID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to remove contact from board: %v", err)
		return
	}

	// Broadcast member removed event to all connected clients
	if h.realtime != nil {
		h.realtime.BroadcastMemberRemoved(boardID.String(), contactID)
	}

	c.Status(http.StatusOK)
}

// RemoveContactCompletely removes a contact from all boards owned by current user
func (h *SettingsHandler) RemoveContactCompletely(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.String(http.StatusUnauthorized, "Unauthorized")
		return
	}

	contactIDStr := c.PostForm("contact_id")
	if contactIDStr == "" {
		c.String(http.StatusBadRequest, "Contact ID is required")
		return
	}

	contactID, err := uuid.Parse(contactIDStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid contact ID")
		return
	}

	// Get all boards owned by the user to broadcast member removal
	var ownedBoards []struct{ ID uuid.UUID }
	if h.realtime != nil {
		boards, boardsErr := h.db.GetUserBoards(context.Background(), userID)
		if boardsErr == nil {
			for _, board := range boards {
				isOwner, _ := h.db.IsBoardOwner(context.Background(), userID, board.ID)
				if isOwner {
					ownedBoards = append(ownedBoards, struct{ ID uuid.UUID }{board.ID})
				}
			}
		}
	}

	// Remove contact from all boards
	err = h.db.RemoveContactFromAllBoards(context.Background(), userID, contactID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to remove contact: %v", err)
		return
	}

	// Broadcast member removed event for each board
	if h.realtime != nil {
		for _, board := range ownedBoards {
			h.realtime.BroadcastMemberRemoved(board.ID.String(), contactID)
		}
	}

	c.Status(http.StatusOK)
}

// CompleteOnboarding marks the onboarding walkthrough as completed for the current user
func (h *SettingsHandler) CompleteOnboarding(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	fmt.Printf("Marking onboarding as completed for user %s\n", userID.String())

	// Update the onboarding_completed field
	updates := map[string]interface{}{
		"onboarding_completed": true,
	}

	err = h.db.UpdateUserProfile(context.Background(), userID, updates)
	if err != nil {
		fmt.Printf("Failed to complete onboarding: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete onboarding"})
		return
	}

	// Update the session to reflect the change
	session := sessions.Default(c)
	session.Set("onboarding_completed", true)
	err = session.Save()
	if err != nil {
		fmt.Printf("Failed to update session: %v\n", err)
		// Continue anyway - database is updated
	}

	fmt.Printf("Successfully marked onboarding as completed for user %s\n", userID.String())
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeleteAccount permanently deletes a user account and all associated data
func (h *SettingsHandler) DeleteAccount(c *gin.Context) {
	userID, err := getUserIDFromSession(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get confirmation from request
	var requestData struct {
		Confirmation string `json:"confirmation"`
	}

	if bindErr := c.ShouldBindJSON(&requestData); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Require user to type "DELETE" to confirm
	if requestData.Confirmation != "DELETE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please type DELETE to confirm"})
		return
	}

	fmt.Printf("User %s requested account deletion\n", userID.String())

	// Delete the account and all associated data
	err = h.db.DeleteUserAccount(context.Background(), userID)
	if err != nil {
		fmt.Printf("Failed to delete account: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete account. Please try again."})
		return
	}

	// Clear the session
	session := sessions.Default(c)
	session.Clear()
	err = session.Save()
	if err != nil {
		fmt.Printf("Failed to clear session: %v\n", err)
	}

	fmt.Printf("Successfully deleted account for user %s\n", userID.String())
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Account deleted successfully"})
}

// Helper function to get user ID from session
func getUserIDFromSession(c *gin.Context) (uuid.UUID, error) {
	session := sessions.Default(c)
	userIDStr := session.Get("user_id")
	if userIDStr == nil {
		return uuid.Nil, fmt.Errorf("user not logged in")
	}

	userIDString, ok := userIDStr.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user ID type in session")
	}

	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in session: %w", err)
	}
	return userID, nil
}
