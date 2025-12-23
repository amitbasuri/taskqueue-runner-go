package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/amitbasuri/taskqueue-go/internal/models"
	"github.com/amitbasuri/taskqueue-go/internal/storage"
	"github.com/gin-gonic/gin"
)

// CreateTask handles POST /tasks
// Creates a new task that will be processed by background workers
func (h *Handler) CreateTask(c *gin.Context) {
	var req models.CreateTaskRequest

	// Bind and validate JSON request body
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required field: type
	if req.Type == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task type is required",
		})
		return
	}

	// If payload is not provided or empty, set to empty JSON object
	if len(req.Payload) == 0 {
		req.Payload = json.RawMessage("{}")
	}

	// Create the task in storage
	task, err := h.store.CreateTask(c.Request.Context(), req)
	if err != nil {
		slog.Error("Failed to create task", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create task",
		})
		return
	}

	slog.Info("Task created",
		"task_id", task.ID,
		"task_name", task.Name,
		"task_type", task.Type,
		"priority", task.Priority,
		"max_retries", task.MaxRetries,
	)

	// Return success response
	c.JSON(http.StatusCreated, models.CreateTaskResponse{
		ID:     task.ID,
		Status: task.Status.String(),
	})
}

// GetTask handles GET /tasks/:id
// Returns the status and details of the task with the given ID
func (h *Handler) GetTask(c *gin.Context) {
	// Parse task ID from URL parameter
	idParam := c.Param("id")
	taskID, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil {
		slog.Warn("Invalid task ID", "id", idParam, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid task ID",
		})
		return
	}

	// Retrieve task from storage
	task, err := h.store.GetTask(c.Request.Context(), taskID)
	if err != nil {
		if errors.Is(err, storage.ErrTaskNotFound) {
			slog.Warn("Task not found", "task_id", taskID)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}

		slog.Error("Failed to get task", "task_id", taskID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve task",
		})
		return
	}

	// Return task details
	c.JSON(http.StatusOK, task.ToTaskResponse())
}
