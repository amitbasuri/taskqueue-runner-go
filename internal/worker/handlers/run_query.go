package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/amitbasuri/taskqueue-runner-go/internal/models"
)

// RunQueryHandler handles database query execution tasks
type RunQueryHandler struct {
	rng *rand.Rand
}

// NewRunQueryHandler creates a new query handler
func NewRunQueryHandler() *RunQueryHandler {
	return &RunQueryHandler{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (h *RunQueryHandler) Type() models.TaskType {
	return models.TaskTypeRunQuery
}

func (h *RunQueryHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var req struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if req.Query == "" {
		return fmt.Errorf("missing required field: query")
	}

	// Check for cancellation before starting work
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// TODO: Integrate with actual database query execution
	slog.Info("Running query",
		"query", req.Query,
		"query_length", len(req.Query),
	)

	// Simulate different failure scenarios for testing:
	// - 20% regular failures (1-2 out of 10)
	// - 20% timeouts (3-4 out of 10) - exceeds worker's 30s timeout
	// - 60% success (5-10 out of 10)
	scenario := h.rng.Intn(10) + 1

	switch {
	case scenario <= 2:
		// Regular failure (20%)
		slog.Warn("Query execution failed (simulated)", "query", req.Query, "scenario", "regular_failure")
		return fmt.Errorf("query execution failed: database connection error")

	case scenario <= 4:
		// Timeout scenario (20%) - use context-aware sleep
		slog.Warn("Query execution timing out (simulated)", "query", req.Query, "scenario", "timeout", "sleep_duration", "5s")
		select {
		case <-time.After(5 * time.Second):
			return fmt.Errorf("query execution failed: database timeout")
		case <-ctx.Done():
			slog.Warn("Query cancelled during timeout simulation", "query", req.Query)
			return ctx.Err()
		}

	default:
		// Success (60%) - with context-aware sleep
		select {
		case <-time.After(3 * time.Second):
			slog.Info("Query executed successfully", "query", req.Query, "scenario", "success")
			return nil
		case <-ctx.Done():
			slog.Warn("Query cancelled during execution", "query", req.Query)
			return ctx.Err()
		}
	}
}
