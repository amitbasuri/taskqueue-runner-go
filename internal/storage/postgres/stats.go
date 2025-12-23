package postgres

import (
	"context"

	"github.com/amitbasuri/taskqueue-go/internal/models"
)

// GetStats retrieves system statistics for dashboard
func (s *Store) GetStats(ctx context.Context) (*models.TaskStatsResponse, error) {
	query := `
		SELECT 
			COUNT(*) as total_tasks,
			COUNT(*) FILTER (WHERE status = 'queued') as queued_tasks,
			COUNT(*) FILTER (WHERE status = 'running') as running_tasks,
			COUNT(*) FILTER (WHERE status = 'succeeded') as succeeded_tasks,
			COUNT(*) FILTER (WHERE status = 'failed') as failed_tasks,
			COALESCE(AVG(retry_count), 0) as avg_retry_count,
			COUNT(*) FILTER (WHERE retry_count > 0) as tasks_with_retries
		FROM tasks
	`

	var stats models.TaskStatsResponse
	err := s.pool.QueryRow(ctx, query).Scan(
		&stats.TotalTasks,
		&stats.QueuedTasks,
		&stats.RunningTasks,
		&stats.SucceededTasks,
		&stats.FailedTasks,
		&stats.AvgRetryCount,
		&stats.TasksWithRetries,
	)

	if err != nil {
		return nil, err
	}

	return &stats, nil
}
