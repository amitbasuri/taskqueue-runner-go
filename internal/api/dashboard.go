package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// StreamTasks streams task updates using Server-Sent Events (SSE)
func (h *Handler) StreamTasks(c *gin.Context) {
	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	ctx := c.Request.Context()
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	// Send updates every 2 seconds
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get latest stats
			stats, err := h.store.GetStats(context.Background())
			if err != nil {
				slog.Error("Failed to get stats for SSE", "error", err)
				continue
			}

			data, err := json.Marshal(stats)
			if err != nil {
				slog.Error("Failed to marshal stats", "error", err)
				continue
			}

			// SSE format: "event: stats\ndata: <json>\n\n"
			fmt.Fprintf(c.Writer, "event: stats\ndata: %s\n\n", string(data))
			flusher.Flush()
		}
	}
}

// ServeDashboard serves the HTML dashboard
func (h *Handler) ServeDashboard(c *gin.Context) {
	c.File("web/templates/dashboard.html")
}
