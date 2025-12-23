package postgres

import (
	"context"
	"log/slog"

	"github.com/amitbasuri/taskqueue-go/internal/models"
	"github.com/amitbasuri/taskqueue-go/internal/storage"
)

// MarkTaskFailed permanently marks a task as failed (no more retries)
func (s *Store) MarkTaskFailed(ctx context.Context, taskID int64, errorMessage string) error {
	query := `
		UPDATE tasks
		SET 
			status = $1,
			last_error = $2,
			locked_at = NULL,
			lock_expires_at = NULL,
			updated_at = NOW()
		WHERE id = $3
	`

	result, err := s.pool.Exec(ctx, query,
		models.TaskStatusFailed,
		errorMessage,
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
		TaskID:       taskID,
		Status:       models.TaskStatusFailed,
		EventType:    models.EventTaskFailedFinal,
		ErrorMessage: &errorMessage,
	}

	if err := s.InsertHistory(ctx, history); err != nil {
		slog.Error("Failed to insert failure history", "task_id", taskID, "error", err)
	}

	return nil
}
