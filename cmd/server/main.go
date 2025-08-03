package main

import (
    "log"
    "os"
    
    "sudo/internal/database"
    "sudo/internal/email"
    "sudo/internal/handlers"
    "sudo/templates/pages"
    
    "github.com/gin-contrib/sessions"
    "github.com/gin-contrib/sessions/cookie"
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "github.com/a-h/templ"
)

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using system environment variables")
    }
    
    // Initialize services
    db := database.NewDB()
    emailService := email.NewEmailService()
    
    // Initialize handlers
    authHandler := handlers.NewAuthHandler(db, emailService)
    boardHandler := handlers.NewBoardHandler(db)
    taskHandler := handlers.NewTaskHandler(db)
    
    // Setup Gin
    if os.Getenv("APP_ENV") == "production" {
        gin.SetMode(gin.ReleaseMode)
    }
    r := gin.Default()
    
    // Setup sessions
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        jwtSecret = "your-secret-key-change-in-production"
        log.Println("Warning: Using default JWT secret. Set JWT_SECRET in production!")
    }
    store := cookie.NewStore([]byte(jwtSecret))
    r.Use(sessions.Sessions("kanban-session", store))
    
    // Serve static files
    r.Static("/static", "./static")
    r.StaticFile("/favicon.ico", "./static/favicon.ico")
    
    // Auth routes (public)
    r.GET("/", func(c *gin.Context) {
        // Check if user is already logged in
        session := sessions.Default(c)
        if userID := session.Get("user_id"); userID != nil {
            c.Redirect(302, "/dashboard")
            return
        }
        
        component := pages.Login()
        handler := templ.Handler(component)
        handler.ServeHTTP(c.Writer, c.Request)
    })
    
    r.POST("/auth/send-otp", authHandler.SendOTP)
    r.POST("/auth/verify-otp", authHandler.VerifyOTP)
    r.POST("/auth/logout", authHandler.Logout)
    
    // Protected routes
}