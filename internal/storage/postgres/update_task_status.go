package postgres

import (
	"context"

	"github.com/amitbasuri/taskqueue-go/internal/models"
	"github.com/amitbasuri/taskqueue-go/internal/storage"
)

// UpdateTaskStatus updates the status of a task
func (s *Store) UpdateTaskStatus(ctx context.Context, taskID int64, status models.TaskStatus, errorMessage *string) error {
	query := `
		UPDATE tasks
		SET status = $1, last_error = $2, updated_at = NOW()
		WHERE id = $3
	`

	result, err := s.pool.Exec(ctx, query, status, errorMessage, taskID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrTaskNotFound
	}

	return nil
}
