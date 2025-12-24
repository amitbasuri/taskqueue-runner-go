package api

import (
	"github.com/amitbasuri/taskqueue-runner-go/internal/storage"
	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for the task queue API
type Handler struct {
	store storage.Store
}

// NewHandler creates a new API handler
func NewHandler(store storage.Store) *Handler {
	return &Handler{
		store: store,
	}
}

// RegisterRoutes registers all API routes on the given router
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Health check endpoint
	r.GET("/health", h.Health)

	// Dashboard UI
	r.GET("/", h.ServeDashboard)
	r.Static("/static", "./web/static")

	// API endpoints
	api := r.Group("/api")
	{
		// Task management endpoints
		api.POST("/tasks", h.CreateTask)
		api.GET("/tasks/:id", h.GetTask)
		api.GET("/tasks/:id/history", h.GetTaskHistory)

		// Dashboard statistics endpoint
		api.GET("/stats", h.GetStats)

		// Server-Sent Events stream for real-time updates
		api.GET("/tasks/stream", h.StreamTasks)
	}
}

// Health checks if the service is healthy
func (h *Handler) Health(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy"})
}
