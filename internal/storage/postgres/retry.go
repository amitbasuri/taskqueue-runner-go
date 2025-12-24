package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"time"

	"github.com/amitbasuri/taskqueue-runner-go/internal/models"
	"github.com/amitbasuri/taskqueue-runner-go/internal/storage"
)

// ScheduleRetry marks a task for retry with exponential backoff
func (s *Store) ScheduleRetry(ctx context.Context, taskID int64, errorMessage string) error {
	// Get current task state
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	// Check if retries are exhausted
	if task.RetryCount >= task.MaxRetries {
		return s.MarkTaskFailed(ctx, taskID, fmt.Sprintf("max retries exceeded: %s", errorMessage))
	}

	// Calculate exponential backoff with jitter
	retryCount := task.RetryCount + 1
	backoffDuration := calculateBackoff(task.BackoffSeconds, retryCount)
	nextRunAt := time.Now().Add(backoffDuration)

	query := `
		UPDATE tasks
		SET 
			status = $1,
			retry_count = $2,
			last_error = $3,
			next_run_at = $4,
			locked_at = NULL,
			lock_expires_at = NULL,
			updated_at = NOW()
		WHERE id = $5
	`

	result, err := s.pool.Exec(ctx, query,
		models.TaskStatusQueued,
		retryCount,
		errorMessage,
		nextRunAt,
		taskID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrTaskNotFound
	}

	// Best-effort history logging
	history := models.TaskHistory{
		TaskID:         taskID,
		Status:         models.TaskStatusQueued,
		EventType:      models.EventRetryScheduled,
		RetryCount:     &retryCount,
		MaxRetries:     &task.MaxRetries,
		BackoffSeconds: &task.BackoffSeconds,
		NextRunAt:      &nextRunAt,
		ErrorMessage:   &errorMessage,
	}

	if err := s.InsertHistory(ctx, history); err != nil {
		slog.Error("Failed to insert retry history", "task_id", taskID, "error", err)
	}

	return nil
}

// calculateBackoff computes exponential backoff with jitter
// Formula: backoff_seconds * (2 ^ retry_count) with random jitter
func calculateBackoff(baseSeconds int, retryCount int) time.Duration {
	// Exponential backoff: base * 2^(retry_count-1)
	// Cap the exponent to prevent overflow (2^20 = ~1M seconds = 11 days)
	exponent := retryCount - 1
	if exponent > 20 {
		exponent = 20
	}

	exponential := float64(baseSeconds) * math.Pow(2, float64(exponent))

	// Hard cap at 1 hour to prevent runaway delays
	if exponential > 3600 {
		exponential = 3600
	}

	// Add proper uniform jitter (±25%)
	// Using math/rand is sufficient for backoff jitter (crypto/rand is overkill)
	jitterPercent := (rand.Float64() * 0.5) - 0.25 // Range: -0.25 to +0.25
	jitter := exponential * jitterPercent

	backoff := exponential + jitter

	// Ensure minimum backoff of 1 second
	if backoff < 1 {
		backoff = 1
	}

	return time.Duration(backoff) * time.Second
}
