package storage

import (
	"context"
	"errors"

	"github.com/amitbasuri/taskqueue-runner-go/internal/models"
)

// Common errors
var (
	ErrTaskNotFound = errors.New("task not found")
)

// Store defines the interface for task storage operations
// This allows for different implementations (PostgreSQL, in-memory, etc.)
type Store interface {
	// CreateTask creates a new task and returns it
	CreateTask(ctx context.Context, req models.CreateTaskRequest) (*models.Task, error)

	// GetTask retrieves a task by its ID
	GetTask(ctx context.Context, id int64) (*models.Task, error)

	// GetTaskHistory retrieves the status change history for a task
	GetTaskHistory(ctx context.Context, taskID int64) ([]models.TaskHistory, error)

	// InsertHistory adds a new detailed event entry to task history
	InsertHistory(ctx context.Context, history models.TaskHistory) error

	// UpdateTaskStatus updates the status of a task
	UpdateTaskStatus(ctx context.Context, taskID int64, status models.TaskStatus, errorMessage *string) error

	// ClaimNextTask atomically claims the next available task for processing
	// Handles timeout recovery and respects next_run_at scheduling
	// Returns nil if no tasks are available
	ClaimNextTask(ctx context.Context, workerID string) (*models.Task, error)

	// ScheduleRetry marks a task for retry with exponential backoff
	ScheduleRetry(ctx context.Context, taskID int64, errorMessage string) error

	// MarkTaskFailed permanently marks a task as failed (no more retries)
	MarkTaskFailed(ctx context.Context, taskID int64, errorMessage string) error

	// CompleteTask marks a task as succeeded
	CompleteTask(ctx context.Context, taskID int64) error

	// GetStats retrieves system statistics for dashboard
	GetStats(ctx context.Context) (*models.TaskStatsResponse, error)
}
