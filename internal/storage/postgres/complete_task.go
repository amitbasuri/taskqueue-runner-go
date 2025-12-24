package postgres

import (
	"context"
	"log/slog"

	"github.com/amitbasuri/taskqueue-runner-go/internal/models"
	"github.com/amitbasuri/taskqueue-runner-go/internal/storage"
)

// CompleteTask marks a task as successfully completed
func (s *Store) CompleteTask(ctx context.Context, taskID int64) error {
	query := `
		UPDATE tasks
		SET 
			status = $1,
			last_error = NULL,
			locked_at = NULL,
			lock_expires_at = NULL,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := s.pool.Exec(ctx, query, models.TaskStatusSucceeded, taskID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrTaskNotFound
	}

	// Best-effort history logging
	history := models.TaskHistory{
		TaskID:    taskID,
		Status:    models.TaskStatusSucceeded,
		EventType: models.EventTaskSucceeded,
	}

	if err := s.InsertHistory(ctx, history); err != nil {
		slog.Error("Failed to insert success history", "task_id", taskID, "error", err)
	}

	return nil
}
