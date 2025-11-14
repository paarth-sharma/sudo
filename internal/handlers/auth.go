package handlers

import (
	"context"
	"crypto/rand"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"sudo/internal/database"
	"sudo/internal/email"
	"sudo/templates/components"

	"github.com/a-h/templ"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	db           *database.DB
	emailService *email.EmailService
}

func NewAuthHandler(db *database.DB, emailService *email.EmailService) *AuthHandler {
	return &AuthHandler{
		db:           db,
		emailService: emailService,
	}
}

func (h *AuthHandler) SendOTP(c *gin.Context) {
	email := strings.TrimSpace(strings.ToLower(c.PostForm("email")))
	if email == "" {
		component := components.AuthError("Email is required")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Enhanced email validation
	if !isValidEmail(email) {
		component := components.AuthError("Please enter a valid email address")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Input sanitization - prevent injection
	if containsSuspiciousChars(email) {
		component := components.AuthError("Invalid email format")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Generate 6-digit OTP
	otp, err := generateOTP()
	if err != nil {
		component := components.AuthError("Failed to generate OTP. Please try again.")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Save OTP to database (expires in 10 minutes)
	expiresAt := time.Now().Add(10 * time.Minute)
	err = h.db.CreateOTP(context.Background(), email, otp, expiresAt)
	if err != nil {
		component := components.AuthError("Failed to create OTP. Please try again.")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Send OTP email using the email service
	err = h.emailService.SendOTP(email, otp)
	if err != nil {
		component := components.AuthError("Failed to send email. Please check your email address and try again.")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Return OTP input form
	component := components.OTPForm(email)
	handler := templ.Handler(component)
	handler.ServeHTTP(c.Writer, c.Request)
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	email := c.PostForm("email")
	otp := c.PostForm("otp")

	if email == "" || otp == "" {
		component := components.AuthError("Email and OTP are required")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Validate OTP
	user, err := h.db.ValidateOTP(context.Background(), email, otp)
	if err != nil {
		component := components.AuthError("Invalid or expired OTP. Please try again.")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Create session
	session := sessions.Default(c)
	session.Set("user_id", user.ID.String())
	session.Set("user_email", user.Email)
	session.Set("user_name", user.Name)
	session.Set("onboarding_completed", user.OnboardingCompleted)

	session.Options(sessions.Options{
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production", // Enable HTTPS in production
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	err = session.Save()
	if err != nil {
		component := components.AuthError("Failed to create session. Please try again.")
		handler := templ.Handler(component)
		handler.ServeHTTP(c.Writer, c.Request)
		return
	}

	// Use window.location instead of HX-Redirect to ensure session cookies are sent
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `<script>window.location.href = "/dashboard";</script>`)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	err := session.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout"})
		return
	}

	// Use window.location with a slight delay to ensure session is cleared
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, `<script>setTimeout(function(){ window.location.href = "/"; }, 100);</script>`)
}

func generateOTP() (string, error) {
	const digits = "0123456789"
	otp := make([]byte, 6)

	for i := range otp {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[num.Int64()]
	}

	return string(otp), nil
}

func isValidEmail(email string) bool {
	// Enhanced email validation with regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	if len(email) < 3 || len(email) > 254 {
		return false
	}

	return emailRegex.MatchString(email)
}

func containsSuspiciousChars(input string) bool {
	// Check for potential injection patterns
	suspiciousPatterns := []string{
		"<script", "</script>", "javascript:", "data:", "vbscript:",
		"onload=", "onerror=", "onclick=", "onmouseover=", "<iframe",
		"eval(", "alert(", "document.cookie", "window.location",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}

	return false
}
