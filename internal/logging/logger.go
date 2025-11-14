package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger levels
const (
	DEBUG = "DEBUG"
	INFO  = "INFO"
	WARN  = "WARN"
	ERROR = "ERROR"
	FATAL = "FATAL"
)

// AppLogger provides structured logging
type AppLogger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	warnLogger  *log.Logger
}

// NewAppLogger creates a new application logger
func NewAppLogger() *AppLogger {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
	}

	// Open log files
	infoFile, err := os.OpenFile(
		filepath.Join("logs", "app.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Printf("Failed to open info log file: %v", err)
		infoFile = os.Stdout
	}

	errorFile, err := os.OpenFile(
		filepath.Join("logs", "error.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Printf("Failed to open error log file: %v", err)
		errorFile = os.Stderr
	}

	debugFile, err := os.OpenFile(
		filepath.Join("logs", "debug.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Printf("Failed to open debug log file: %v", err)
		debugFile = os.Stdout
	}

	// Create multi-writers to write to both file and console in development
	var infoWriter, errorWriter, debugWriter io.Writer

	if os.Getenv("APP_ENV") == "production" {
		infoWriter = infoFile
		errorWriter = errorFile
		debugWriter = debugFile
	} else {
		infoWriter = io.MultiWriter(os.Stdout, infoFile)
		errorWriter = io.MultiWriter(os.Stderr, errorFile)
		debugWriter = io.MultiWriter(os.Stdout, debugFile)
	}

	return &AppLogger{
		infoLogger:  log.New(infoWriter, "[INFO] ", log.LstdFlags|log.Lshortfile),
		errorLogger: log.New(errorWriter, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		debugLogger: log.New(debugWriter, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		warnLogger:  log.New(infoWriter, "[WARN] ", log.LstdFlags|log.Lshortfile),
	}
}

// Info logs info level messages
func (l *AppLogger) Info(format string, v ...interface{}) {
	l.infoLogger.Printf(format, v...)
}

// Error logs error level messages
func (l *AppLogger) Error(format string, v ...interface{}) {
	l.errorLogger.Printf(format, v...)
}

// Debug logs debug level messages
func (l *AppLogger) Debug(format string, v ...interface{}) {
	if os.Getenv("LOG_LEVEL") == "debug" || os.Getenv("APP_ENV") != "production" {
		l.debugLogger.Printf(format, v...)
	}
}

// Warn logs warning level messages
func (l *AppLogger) Warn(format string, v ...interface{}) {
	l.warnLogger.Printf(format, v...)
}

// Fatal logs fatal level messages and exits
func (l *AppLogger) Fatal(format string, v ...interface{}) {
	l.errorLogger.Printf("[FATAL] "+format, v...)
	os.Exit(1)
}

// WebSocket specific logging methods
func (l *AppLogger) LogWebSocketConnection(userID, boardID string, action string) {
	l.Info("WebSocket %s: User %s, Board %s", action, userID, boardID)
}

func (l *AppLogger) LogWebSocketMessage(userID, boardID, messageType string) {
	l.Debug("WebSocket message: User %s, Board %s, Type %s", userID, boardID, messageType)
}

func (l *AppLogger) LogWebSocketError(userID, boardID string, err error) {
	l.Error("WebSocket error: User %s, Board %s, Error: %v", userID, boardID, err)
}

// Database operation logging
func (l *AppLogger) LogDatabaseOperation(operation string, duration time.Duration, err error) {
	if err != nil {
		l.Error("Database %s failed after %v: %v", operation, duration, err)
	} else {
		l.Debug("Database %s completed in %v", operation, duration)
	}
}

// Security logging
func (l *AppLogger) LogSecurityEvent(userID, event, details string) {
	l.Warn("Security event: User %s, Event %s, Details: %s", userID, event, details)
}

// Performance logging
func (l *AppLogger) LogPerformance(operation string, duration time.Duration, metadata map[string]interface{}) {
	if duration > 1*time.Second {
		l.Warn("Slow operation: %s took %v, metadata: %+v", operation, duration, metadata)
	} else {
		l.Debug("Performance: %s took %v", operation, duration)
	}
}

// Request logging middleware for Gin
func (l *AppLogger) RequestLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format
		return fmt.Sprintf("[%s] %s %s %d %v \"%s\" \"%s\" %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.ClientIP,
			param.Method,
			param.StatusCode,
			param.Latency,
			param.Path,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// Global logger instance
var AppLog = NewAppLogger()

// Package-level convenience functions
func Info(format string, v ...interface{}) {
	AppLog.Info(format, v...)
}

func Error(format string, v ...interface{}) {
	AppLog.Error(format, v...)
}

func Debug(format string, v ...interface{}) {
	AppLog.Debug(format, v...)
}

func Warn(format string, v ...interface{}) {
	AppLog.Warn(format, v...)
}

func Fatal(format string, v ...interface{}) {
	AppLog.Fatal(format, v...)
}

func LogWebSocketConnection(userID, boardID string, action string) {
	AppLog.LogWebSocketConnection(userID, boardID, action)
}

func LogWebSocketMessage(userID, boardID, messageType string) {
	AppLog.LogWebSocketMessage(userID, boardID, messageType)
}

func LogWebSocketError(userID, boardID string, err error) {
	AppLog.LogWebSocketError(userID, boardID, err)
}

func LogDatabaseOperation(operation string, duration time.Duration, err error) {
	AppLog.LogDatabaseOperation(operation, duration, err)
}

func LogSecurityEvent(userID, event, details string) {
	AppLog.LogSecurityEvent(userID, event, details)
}

func LogPerformance(operation string, duration time.Duration, metadata map[string]interface{}) {
	AppLog.LogPerformance(operation, duration, metadata)
}
