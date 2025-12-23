package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetStats handles GET /stats
// Returns system statistics for dashboard visualization
func (h *Handler) GetStats(c *gin.Context) {
	// Retrieve statistics from storage
	stats, err := h.store.GetStats(c.Request.Context())
	if err != nil {
		slog.Error("Failed to get stats", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve statistics",
		})
		return
	}

	// Return statistics
	c.JSON(http.StatusOK, stats)
}
