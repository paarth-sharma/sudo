package metrics

import (
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics holds application performance metrics
// Thread-safety is handled by MetricsCollector's mutex, not this struct
type Metrics struct {
	// WebSocket metrics
	ConnectedUsers   int64 `json:"connected_users"`
	TotalConnections int64 `json:"total_connections"`
	MessagesSent     int64 `json:"messages_sent"`
	MessagesReceived int64 `json:"messages_received"`
	WebSocketErrors  int64 `json:"websocket_errors"`
	ConnectionDrops  int64 `json:"connection_drops"`

	// Performance metrics
	AverageLatency  float64 `json:"average_latency_ms"`
	DatabaseLatency float64 `json:"database_latency_ms"`
	RequestCount    int64   `json:"request_count"`
	ErrorCount      int64   `json:"error_count"`

	// System metrics
	MemoryUsage    float64 `json:"memory_usage_mb"`
	CPUUsage       float64 `json:"cpu_usage_percent"`
	GoroutineCount int     `json:"goroutine_count"`

	// Business metrics
	ActiveBoards int64 `json:"active_boards"`
	TasksCreated int64 `json:"tasks_created"`
	TasksMoved   int64 `json:"tasks_moved"`

	// Timestamps
	LastUpdated time.Time `json:"last_updated"`
	StartTime   time.Time `json:"start_time"`
	Uptime      float64   `json:"uptime_seconds"`
}

// MetricsCollector manages metrics collection
type MetricsCollector struct {
	metrics *Metrics
	mu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: &Metrics{
			StartTime:   time.Now(),
			LastUpdated: time.Now(),
		},
	}
}

// GetMetrics returns current metrics (thread-safe)
func (mc *MetricsCollector) GetMetrics() *Metrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := *mc.metrics
	metrics.LastUpdated = time.Now()
	metrics.Uptime = time.Since(metrics.StartTime).Seconds()
	metrics.GoroutineCount = runtime.NumGoroutine()

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics.MemoryUsage = float64(m.Alloc) / 1024 / 1024 // Convert to MB

	return &metrics
}

// Increment methods (thread-safe)
func (mc *MetricsCollector) IncrementConnectedUsers() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.ConnectedUsers++
	mc.metrics.TotalConnections++
}

func (mc *MetricsCollector) DecrementConnectedUsers() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if mc.metrics.ConnectedUsers > 0 {
		mc.metrics.ConnectedUsers--
	}
}

func (mc *MetricsCollector) IncrementMessagesSent() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.MessagesSent++
}

func (mc *MetricsCollector) IncrementMessagesReceived() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.MessagesReceived++
}

func (mc *MetricsCollector) IncrementWebSocketErrors() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.WebSocketErrors++
}

func (mc *MetricsCollector) IncrementConnectionDrops() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.ConnectionDrops++
}

func (mc *MetricsCollector) IncrementRequestCount() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.RequestCount++
}

func (mc *MetricsCollector) IncrementErrorCount() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.ErrorCount++
}

func (mc *MetricsCollector) IncrementTasksCreated() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.TasksCreated++
}

func (mc *MetricsCollector) IncrementTasksMoved() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.TasksMoved++
}

func (mc *MetricsCollector) SetActiveBoards(count int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics.ActiveBoards = count
}

// RecordLatency records request latency
func (mc *MetricsCollector) RecordLatency(latency time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	latencyMs := float64(latency.Nanoseconds()) / 1000000

	// Simple moving average (can be improved with proper windowing)
	if mc.metrics.AverageLatency == 0 {
		mc.metrics.AverageLatency = latencyMs
	} else {
		mc.metrics.AverageLatency = (mc.metrics.AverageLatency*0.9 + latencyMs*0.1)
	}
}

// RecordDatabaseLatency records database operation latency
func (mc *MetricsCollector) RecordDatabaseLatency(latency time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	latencyMs := float64(latency.Nanoseconds()) / 1000000

	if mc.metrics.DatabaseLatency == 0 {
		mc.metrics.DatabaseLatency = latencyMs
	} else {
		mc.metrics.DatabaseLatency = (mc.metrics.DatabaseLatency*0.9 + latencyMs*0.1)
	}
}

// Middleware for Gin to collect HTTP metrics
func (mc *MetricsCollector) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		mc.IncrementRequestCount()

		c.Next()

		// Record latency
		latency := time.Since(startTime)
		mc.RecordLatency(latency)

		// Count errors
		if c.Writer.Status() >= 400 {
			mc.IncrementErrorCount()
		}
	}
}

// HTTP handler to expose metrics
func (mc *MetricsCollector) MetricsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics := mc.GetMetrics()
		c.JSON(http.StatusOK, metrics)
	}
}

// Global metrics collector instance
var GlobalMetrics = NewMetricsCollector()

// Helper functions for easy access
func IncrementConnectedUsers() {
	GlobalMetrics.IncrementConnectedUsers()
}

func DecrementConnectedUsers() {
	GlobalMetrics.DecrementConnectedUsers()
}

func IncrementMessagesSent() {
	GlobalMetrics.IncrementMessagesSent()
}

func IncrementMessagesReceived() {
	GlobalMetrics.IncrementMessagesReceived()
}

func IncrementWebSocketErrors() {
	GlobalMetrics.IncrementWebSocketErrors()
}

func IncrementConnectionDrops() {
	GlobalMetrics.IncrementConnectionDrops()
}

func IncrementTasksCreated() {
	GlobalMetrics.IncrementTasksCreated()
}

func IncrementTasksMoved() {
	GlobalMetrics.IncrementTasksMoved()
}

func RecordDatabaseLatency(latency time.Duration) {
	GlobalMetrics.RecordDatabaseLatency(latency)
}

func SetActiveBoards(count int64) {
	GlobalMetrics.SetActiveBoards(count)
}

// StartMetricsServer starts a separate metrics server
func StartMetricsServer(port string) {
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/metrics", GlobalMetrics.MetricsHandler())
	r.GET("/health", func(c *gin.Context) {
		metrics := GlobalMetrics.GetMetrics()

		health := gin.H{
			"status":          "healthy",
			"timestamp":       time.Now(),
			"uptime":          metrics.Uptime,
			"memory_usage_mb": metrics.MemoryUsage,
			"goroutines":      metrics.GoroutineCount,
			"connected_users": metrics.ConnectedUsers,
		}

		// Determine health status based on metrics
		if metrics.ErrorCount > 0 && float64(metrics.ErrorCount)/float64(metrics.RequestCount) > 0.1 {
			health["status"] = "degraded"
			c.JSON(http.StatusServiceUnavailable, health)
		} else {
			c.JSON(http.StatusOK, health)
		}
	})

	log.Printf("Metrics server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Printf("Failed to start metrics server: %v", err)
	}
}
