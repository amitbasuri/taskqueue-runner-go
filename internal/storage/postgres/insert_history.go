package postgres

import (
	"context"

	"github.com/amitbasuri/taskqueue-runner-go/internal/models"
)

// InsertHistory adds a new detailed event entry to task history
func (s *Store) InsertHistory(ctx context.Context, history models.TaskHistory) error {
	query := `
		INSERT INTO task_history (
			task_id, status, event_type, 
			retry_count, max_retries, backoff_seconds, next_run_at,
			error_message, worker_id, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
	`

	_, err := s.pool.Exec(ctx, query,
		history.TaskID,
		history.Status,
		history.EventType,
		history.RetryCount,
		history.MaxRetries,
		history.BackoffSeconds,
		history.NextRunAt,
		history.ErrorMessage,
		history.WorkerID,
	)
	return err
}
