package handlers

import (
    "context"
    "crypto/rand"
    "math/big"
    "net/http"
    "time"
    
    "sudo/internal/database"
    "sudo/internal/email"
    "sudo/templates/components"
    
    "github.com/gin-contrib/sessions"
    "github.com/gin-gonic/gin"
    "github.com/a-h/templ"
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
    email := c.PostForm("email")
    if email == "" {
        component := components.AuthError("Email is required")
        handler := templ.Handler(component)
        handler.ServeHTTP(c.Writer, c.Request)
        return
    }
    
    // Validate email format
    if !isValidEmail(email) {
        component := components.AuthError("Please enter a valid email address")
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
    
    session.Options(sessions.Options{
        MaxAge:   86400 * 30, // 30 days
        HttpOnly: true,
        Secure:   false, // Disable for local development
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
        Secure:   false,
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
    // Simple email validation
    if len(email) < 3 || len(email) > 254 {
        return false
    }
    
    atCount := 0
    lastAt := -1
    
    for i, ch := range email {
        if ch == '@' {
            atCount++
            lastAt = i
        }
    }
    
    if atCount != 1 || lastAt == 0 || lastAt == len(email)-1 {
        return false
    }
    
    // Check for dot after @
    hasDotAfterAt := false
    for i := lastAt + 1; i < len(email); i++ {
        if email[i] == '.' && i != lastAt+1 && i != len(email)-1 {
            hasDotAfterAt = true
            break
        }
    }
    
    return hasDotAfterAt
}