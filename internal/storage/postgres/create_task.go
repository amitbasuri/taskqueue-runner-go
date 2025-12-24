package postgres

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/amitbasuri/taskqueue-runner-go/internal/models"
)

// CreateTask creates a new task in the database
func (s *Store) CreateTask(ctx context.Context, req models.CreateTaskRequest) (*models.Task, error) {
	// Normalize task type to lowercase for consistent handling
	req.Type = strings.ToLower(req.Type)

	// Set defaults
	maxRetries := 3
	if req.MaxRetries != nil {
		maxRetries = *req.MaxRetries
	}

	timeoutSeconds := 30
	if req.TimeoutSeconds != nil {
		timeoutSeconds = *req.TimeoutSeconds
	}

	backoffSeconds := 5
	if req.BackoffSeconds != nil {
		backoffSeconds = *req.BackoffSeconds
	}

	// Default payload to empty JSON object if not provided
	payload := req.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	query := `
		INSERT INTO tasks (
			name, type, payload, priority, status, 
			retry_count, max_retries, backoff_seconds, 
			timeout_seconds, next_run_at, 
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, name, type, payload, status, priority, 
		          retry_count, max_retries, last_error, 
		          next_run_at, backoff_seconds, timeout_seconds, 
		          locked_at, lock_expires_at, created_at, updated_at
	`

	var task models.Task
	err := s.pool.QueryRow(ctx, query,
		req.Name,
		req.Type,
		payload,
		req.Priority,
		models.TaskStatusQueued,
		0, // retry_count starts at 0
		maxRetries,
		backoffSeconds,
		timeoutSeconds,
		time.Now(), // next_run_at - available immediately
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
		return nil, err
	}

	// Best-effort history logging - don't fail task creation if history insert fails
	history := models.TaskHistory{
		TaskID:         task.ID,
		Status:         models.TaskStatusQueued,
		EventType:      models.EventTaskQueued,
		RetryCount:     &task.RetryCount,
		MaxRetries:     &task.MaxRetries,
		BackoffSeconds: &task.BackoffSeconds,
		NextRunAt:      &task.NextRunAt,
	}

	if err := s.InsertHistory(ctx, history); err != nil {
		slog.Error("Failed to insert task creation history", "task_id", task.ID, "error", err)
	}

	return &task, nil
}
