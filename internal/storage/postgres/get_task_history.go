package postgres

import (
	"context"

	"github.com/amitbasuri/taskqueue-go/internal/models"
)

// GetTaskHistory retrieves the history of status changes for a task
func (s *Store) GetTaskHistory(ctx context.Context, taskID int64) ([]models.TaskHistory, error) {
	query := `
		SELECT id, task_id, status, event_type, 
		       retry_count, max_retries, backoff_seconds, next_run_at,
		       error_message, worker_id, created_at
		FROM task_history
		WHERE task_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.TaskHistory
	for rows.Next() {
		var h models.TaskHistory
		err := rows.Scan(
			&h.ID,
			&h.TaskID,
			&h.Status,
			&h.EventType,
			&h.RetryCount,
			&h.MaxRetries,
			&h.BackoffSeconds,
			&h.NextRunAt,
			&h.ErrorMessage,
			&h.WorkerID,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil
	if history == nil {
		history = []models.TaskHistory{}
	}

	return history, nil
}
