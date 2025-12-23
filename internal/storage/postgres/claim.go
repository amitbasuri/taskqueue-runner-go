package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/amitbasuri/taskqueue-go/internal/models"
	"github.com/jackc/pgx/v5"
)

// ClaimNextTask atomically claims the next available task for processing
// Handles timeout recovery and respects next_run_at scheduling
// Prioritizes tasks with expired locks to prevent starvation
func (s *Store) ClaimNextTask(ctx context.Context, workerID string) (*models.Task, error) {
	now := time.Now()

	query := `
		UPDATE tasks
		SET 
			status = $1,
			locked_at = $2,
			lock_expires_at = $2 + (timeout_seconds || ' seconds')::interval,
			updated_at = $2
		WHERE id = (
			SELECT id
			FROM tasks
			WHERE status = $3
			  AND next_run_at <= $2
			  AND (lock_expires_at IS NULL OR lock_expires_at <= $2)
			ORDER BY 
			  -- Prioritize tasks with expired locks (stalled tasks)
			  CASE WHEN lock_expires_at IS NOT NULL AND lock_expires_at <= $2 THEN 0 ELSE 1 END,
			  -- Then by priority (higher first)
			  priority DESC, 
			  -- Then by creation time (FIFO)
			  created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, name, type, payload, status, priority, 
		          retry_count, max_retries, last_error, 
		          next_run_at, backoff_seconds, timeout_seconds, 
		          locked_at, lock_expires_at, created_at, updated_at
	`

	var task models.Task
	err := s.pool.QueryRow(ctx, query,
		models.TaskStatusRunning,
		now,
		models.TaskStatusQueued,
	).Scan(
		&task.ID,
		&task.Name,
		&task.Type,
		&task.Payload,
		&task.Status,
		&task.Priority,
		&task.RetryCount,
		&task.MaxRetries,
		&task.LastError,
		&task.NextRunAt,
		&task.BackoffSeconds,
		&task.TimeoutSeconds,
		&task.LockedAt,
		&task.LockExpiresAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No tasks available
		}
		return nil, err
	}

	return &task, nil
}
