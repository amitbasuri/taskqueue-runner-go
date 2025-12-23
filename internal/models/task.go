package models

import (
	"context"
	"encoding/json"
	"time"
)

// TaskType represents the type of task to be executed
type TaskType string

const (
	TaskTypeSendEmail TaskType = "send_email"
	TaskTypeRunQuery  TaskType = "run_query"
)

// TaskStatus represents the lifecycle status of a task (4 essential public-facing statuses)
type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

// EventType represents granular task lifecycle events for history tracking
type EventType string

const (
	EventTaskQueued         EventType = "task_queued"
	EventTaskStarted        EventType = "task_started"
	EventTaskSucceeded      EventType = "task_succeeded"
	EventTaskFailed         EventType = "task_failed"
	EventRetryScheduled     EventType = "retry_scheduled"
	EventTimeoutOccurred    EventType = "timeout_occurred"
	EventWorkerLockAcquired EventType = "worker_lock_acquired"
	EventWorkerLockExpired  EventType = "worker_lock_expired"
	EventTaskFailedFinal    EventType = "task_failed_final"
)

// IsValid checks if the task status is valid
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusQueued, TaskStatusRunning, TaskStatusSucceeded, TaskStatusFailed:
		return true
	}
	return false
}

// String returns the string representation of TaskStatus
func (s TaskStatus) String() string {
	return string(s)
}

// Task represents a background task with retry, timeout, and scheduling support
type Task struct {
	ID       int64           `json:"id" db:"id"`
	Name     string          `json:"name" db:"name"`
	Type     string          `json:"type" db:"type"`
	Payload  json.RawMessage `json:"payload" db:"payload"`
	Status   TaskStatus      `json:"status" db:"status"`
	Priority int             `json:"priority" db:"priority"`

	// Retry metadata
	RetryCount int     `json:"retry_count" db:"retry_count"`
	MaxRetries int     `json:"max_retries" db:"max_retries"`
	LastError  *string `json:"last_error,omitempty" db:"last_error"`

	// Scheduling & backoff
	NextRunAt      time.Time `json:"next_run_at" db:"next_run_at"`
	BackoffSeconds int       `json:"backoff_seconds" db:"backoff_seconds"`

	// Timeout & worker safety
	TimeoutSeconds int        `json:"timeout_seconds" db:"timeout_seconds"`
	LockedAt       *time.Time `json:"locked_at,omitempty" db:"locked_at"`
	LockExpiresAt  *time.Time `json:"lock_expires_at,omitempty" db:"lock_expires_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TaskHistory represents a detailed status change event in a task's lifecycle
type TaskHistory struct {
	ID        int64      `json:"id" db:"id"`
	TaskID    int64      `json:"task_id" db:"task_id"`
	Status    TaskStatus `json:"status" db:"status"`
	EventType EventType  `json:"event_type" db:"event_type"`

	// Retry metadata at time of event
	RetryCount     *int       `json:"retry_count,omitempty" db:"retry_count"`
	MaxRetries     *int       `json:"max_retries,omitempty" db:"max_retries"`
	BackoffSeconds *int       `json:"backoff_seconds,omitempty" db:"backoff_seconds"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`

	ErrorMessage *string   `json:"error_message,omitempty" db:"error_message"`
	WorkerID     *string   `json:"worker_id,omitempty" db:"worker_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// CreateTaskRequest represents the API request to create a new task
type CreateTaskRequest struct {
	Name           string          `json:"name" binding:"required"`
	Type           string          `json:"type" binding:"required"`
	Payload        json.RawMessage `json:"payload"`
	Priority       int             `json:"priority"`
	MaxRetries     *int            `json:"max_retries,omitempty"`
	TimeoutSeconds *int            `json:"timeout_seconds,omitempty"`
	BackoffSeconds *int            `json:"backoff_seconds,omitempty"`
}

// CreateTaskResponse represents the API response when creating a task
type CreateTaskResponse struct {
	ID     int64  `json:"id"`
	Status string `json:"status"`
}

// TaskResponse represents the API response for task details
type TaskResponse struct {
	ID             int64           `json:"id"`
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	Status         string          `json:"status"`
	Priority       int             `json:"priority"`
	RetryCount     int             `json:"retry_count"`
	MaxRetries     int             `json:"max_retries"`
	LastError      *string         `json:"last_error,omitempty"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// TaskHistoryResponse represents the API response for task history
type TaskHistoryResponse struct {
	History []TaskHistory `json:"history"`
}

// TaskStatsResponse represents system statistics for dashboard
type TaskStatsResponse struct {
	TotalTasks       int64   `json:"total_tasks"`
	QueuedTasks      int64   `json:"queued_tasks"`
	RunningTasks     int64   `json:"running_tasks"`
	SucceededTasks   int64   `json:"succeeded_tasks"`
	FailedTasks      int64   `json:"failed_tasks"`
	AvgRetryCount    float64 `json:"avg_retry_count"`
	TasksWithRetries int64   `json:"tasks_with_retries"`
}

// ToTaskResponse converts a Task to TaskResponse
func (t *Task) ToTaskResponse() TaskResponse {
	return TaskResponse{
		ID:             t.ID,
		Name:           t.Name,
		Type:           t.Type,
		Payload:        t.Payload,
		Status:         t.Status.String(),
		Priority:       t.Priority,
		RetryCount:     t.RetryCount,
		MaxRetries:     t.MaxRetries,
		LastError:      t.LastError,
		TimeoutSeconds: t.TimeoutSeconds,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}

// TaskHandler defines the interface that all task handlers must implement
type TaskHandler interface {
	// Execute runs the task with the given payload
	// Returns an error if the task execution fails
	Execute(ctx context.Context, payload json.RawMessage) error

	// Type returns the unique type identifier for this handler
	Type() TaskType
}
