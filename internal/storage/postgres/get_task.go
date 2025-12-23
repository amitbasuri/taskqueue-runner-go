package postgres

import (
	"context"
	"errors"

	"github.com/amitbasuri/taskqueue-go/internal/models"
	"github.com/amitbasuri/taskqueue-go/internal/storage"
	"github.com/jackc/pgx/v5"
)

// GetTask retrieves a task by ID
func (s *Store) GetTask(ctx context.Context, id int64) (*models.Task, error) {
	query := `
		SELECT id, name, type, payload, status, priority, 
		       retry_count, max_retries, last_error, 
		       next_run_at, backoff_seconds, timeout_seconds, 
		       locked_at, lock_expires_at, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`

	var task models.Task
	err := s.pool.QueryRow(ctx, query, id).Scan(
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
			return nil, storage.ErrTaskNotFound
		}
		return nil, err
	}

	return &task, nil
}
