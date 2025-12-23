package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/amitbasuri/taskqueue-go/internal/models"
	"github.com/amitbasuri/taskqueue-go/internal/storage"
	"github.com/gin-gonic/gin"
)

// GetTaskHistory handles GET /tasks/:id/history
// Returns the complete history of status changes for the task
func (h *Handler) GetTaskHistory(c *gin.Context) {
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

	// Verify task exists first
	_, err = h.store.GetTask(c.Request.Context(), taskID)
	if err != nil {
		if errors.Is(err, storage.ErrTaskNotFound) {
			slog.Warn("Task not found", "task_id", taskID)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}

		slog.Error("Failed to verify task existence", "task_id", taskID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve task",
		})
		return
	}

	// Retrieve task history from storage
	history, err := h.store.GetTaskHistory(c.Request.Context(), taskID)
	if err != nil {
		slog.Error("Failed to get task history", "task_id", taskID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve task history",
		})
		return
	}

	// Return history
	c.JSON(http.StatusOK, models.TaskHistoryResponse{
		History: history,
	})
}
