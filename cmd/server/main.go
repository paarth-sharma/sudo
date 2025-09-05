package main

import (
    "fmt"
    "log"
    "os"
    "strings"
    
    "sudo/internal/database"
    "sudo/internal/email"
    "sudo/internal/handlers"
    "sudo/internal/realtime"
    "sudo/templates/pages"
    
    "github.com/gin-contrib/sessions"
    "github.com/gin-contrib/sessions/cookie"
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "github.com/a-h/templ"
)

// AuthMiddleware checks if user is authenticated
func AuthMiddleware() gin.HandlerFunc {
    return gin.HandlerFunc(func(c *gin.Context) {
        session := sessions.Default(c)
        userID := session.Get("user_id")
        
        if userID == nil {
            // For HTMX requests, redirect via header
            if c.GetHeader("HX-Request") == "true" {
                c.Header("HX-Redirect", "/")
                c.AbortWithStatus(401)
                return
            }
            // For regular requests, redirect normally
            c.Redirect(302, "/")
            c.Abort()
            return
        }
        
        c.Next()
    })
}

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found, using system environment variables")
    }
    
    // Initialize services
    db := database.NewDB()
    emailService := email.NewEmailService()
    
    // Add real-time service initialization
    realtimeService := realtime.NewRealtimeService(db)
    go realtimeService.Run() // Start the real-time hub
    
    // Initialize handlers
    authHandler := handlers.NewAuthHandler(db, emailService)
    boardHandler := handlers.NewBoardHandler(db, realtimeService) // Pass realtime service
    taskHandler := handlers.NewTaskHandler(db, realtimeService) // Pass realtime service
    
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
    
    // Serve static files with cache control headers for development
    if os.Getenv("GIN_MODE") != "release" {
        r.Use(func(c *gin.Context) {
            // Add no-cache headers for development to ensure fresh assets
            if strings.HasPrefix(c.Request.URL.Path, "/static/") {
                c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
                c.Header("Pragma", "no-cache")
                c.Header("Expires", "0")
            }
            c.Next()
        })
    }
    r.Static("/static", "./static")
    r.StaticFile("/favicon.ico", "./static/favicon.ico")
    
    // Public routes (no auth required)
    public := r.Group("/")
    {
        // Auth routes
        public.GET("/", func(c *gin.Context) {
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
        
        public.POST("/auth/send-otp", authHandler.SendOTP)
        public.POST("/auth/verify-otp", authHandler.VerifyOTP)
        public.POST("/auth/logout", authHandler.Logout)
    }
    
    // Add request logging middleware
    r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %#v\n",
            param.TimeStamp.Format("2006/01/02 - 15:04:05"),
            param.StatusCode,
            param.Latency,
            param.ClientIP,
            param.Method,
            param.Path,
        )
    }))

    // Protected routes (auth required)
    protected := r.Group("/")
    protected.Use(AuthMiddleware())
    {
        // Dashboard
        protected.GET("/dashboard", boardHandler.Dashboard)
        
        // Board routes
        protected.POST("/boards", boardHandler.CreateBoard)
        protected.GET("/boards/:id", boardHandler.ViewBoard)
        protected.PUT("/boards/:id", boardHandler.UpdateBoard)
        protected.DELETE("/boards/:id", boardHandler.DeleteBoard)
        protected.POST("/boards/:id/invite", boardHandler.InviteMember)
        protected.DELETE("/boards/:id/members/:memberId", boardHandler.RemoveBoardMember)
        protected.GET("/boards/:id/members", boardHandler.GetBoardMembers)
        protected.POST("/boards/:id/columns", boardHandler.CreateColumn)
        protected.PUT("/columns/:id", boardHandler.UpdateColumn)
        protected.DELETE("/columns/:id", boardHandler.DeleteColumn)
        
        // Task routes
        protected.POST("/tasks", taskHandler.CreateTask)
        protected.GET("/tasks/:id", taskHandler.GetTask)
        protected.PUT("/tasks/:id", taskHandler.UpdateTask)
        protected.DELETE("/tasks/:id", taskHandler.DeleteTask)
        protected.POST("/tasks/move", taskHandler.MoveTask)
        protected.POST("/tasks/:id/assign", taskHandler.AssignTask)
        protected.DELETE("/tasks/:id/assign", taskHandler.UnassignTask)
        protected.POST("/tasks/:id/convert-to-board", taskHandler.CreateNestedBoard)
        
        // Additional API endpoints for HTMX
        protected.GET("/api/boards/:id/tasks", func(c *gin.Context) {
            // Get all tasks for a board (for filtering/searching)
            boardHandler.GetBoardTasks(c)
        })
        protected.GET("/api/boards/:id/nested", func(c *gin.Context) {
            // Get all nested boards for a parent board
            boardHandler.GetNestedBoards(c)
        })
        
        protected.POST("/api/tasks/:id/complete", func(c *gin.Context) {
            // Mark task as complete
            taskHandler.CompleteTask(c)
        })
        
        protected.POST("/api/tasks/:id/reopen", func(c *gin.Context) {
            // Reopen completed task
            taskHandler.ReopenTask(c)
        })
        
        protected.GET("/api/search", func(c *gin.Context) {
            // Global search across boards and tasks
            boardHandler.SearchContent(c)
        })
        
        // WebSocket endpoint for real-time updates
        protected.GET("/ws/:boardId", func(c *gin.Context) {
            // WebSocket connection for real-time collaboration
            boardHandler.HandleWebSocket(c)
        })
    }
    
    // Health check endpoint
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "ok",
            "version": "1.0.0",
        })
    })
    
    // Get port from environment
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    log.Printf("Server starting on port %s", port)
    log.Printf("Dashboard: http://localhost:%s/dashboard", port)
    
    if err := r.Run(":" + port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}