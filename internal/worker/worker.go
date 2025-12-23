package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amitbasuri/taskqueue-go/internal/models"
	"github.com/amitbasuri/taskqueue-go/internal/storage"
)

// Worker processes tasks from the queue
type Worker struct {
	store             storage.Store
	handlerRegistry   *HandlerRegistry
	pollInterval      time.Duration
	taskTimeout       time.Duration
	simulatedTaskTime time.Duration
	maxConcurrency    int
	workerID          string
}

// Config holds worker configuration
type Config struct {
	PollInterval      time.Duration // How often to check for new tasks
	TaskTimeout       time.Duration // Maximum time for a task to execute
	SimulatedTaskTime time.Duration // Simulated task processing time
	MaxConcurrency    int           // Maximum number of concurrent tasks
}

// NewWorker creates a new worker instance
func NewWorker(store storage.Store, handlerRegistry *HandlerRegistry, config Config) *Worker {
	if config.PollInterval == 0 {
		config.PollInterval = 1 * time.Second
	}
	if config.TaskTimeout == 0 {
		config.TaskTimeout = 30 * time.Second
	}
	if config.SimulatedTaskTime == 0 {
		config.SimulatedTaskTime = 3 * time.Second // Default 3 second task processing time
	}
	if config.MaxConcurrency == 0 {
		config.MaxConcurrency = 5 // Default 5 concurrent tasks
	}

	// Generate stable worker ID: hostname + PID + timestamp
	// In Kubernetes, all pods have PID=1, so we add timestamp for uniqueness
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	workerID := fmt.Sprintf("%s-%d-%d", hostname, os.Getpid(), time.Now().UnixNano())

	return &Worker{
		store:             store,
		handlerRegistry:   handlerRegistry,
		pollInterval:      config.PollInterval,
		taskTimeout:       config.TaskTimeout,
		simulatedTaskTime: config.SimulatedTaskTime,
		maxConcurrency:    config.MaxConcurrency,
		workerID:          workerID,
	}
}

// Start begins the worker with a dispatcher model to prevent DB thundering herd
func (w *Worker) Start(ctx context.Context) error {
	slog.Info("Worker started",
		"poll_interval", w.pollInterval,
		"task_timeout", w.taskTimeout,
		"simulated_task_time", w.simulatedTaskTime,
		"max_concurrency", w.maxConcurrency,
	)

	// Task channel acts as a buffer between fetcher and workers
	taskChan := make(chan *models.Task, w.maxConcurrency)

	// Start a single dispatcher goroutine that fetches tasks
	go w.dispatcherLoop(ctx, taskChan)

	// Start worker pool to process tasks from channel
	for i := 0; i < w.maxConcurrency; i++ {
		workerNum := i + 1
		go w.workerLoop(ctx, workerNum, taskChan)
	}

	// Wait for context cancellation
	<-ctx.Done()
	slog.Info("Worker stopping due to context cancellation")
	close(taskChan)
	return ctx.Err()
}

// dispatcherLoop continuously fetches tasks and sends them to worker pool
// This prevents the DB thundering herd problem
func (w *Worker) dispatcherLoop(ctx context.Context, taskChan chan<- *models.Task) {
	slog.Info("Dispatcher started")
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Dispatcher stopping")
			return
		case <-ticker.C:
			// Try to claim a task
			task, err := w.store.ClaimNextTask(ctx, w.workerID)
			if err != nil {
				slog.Error("Error claiming task", "error", err)
				continue
			}

			// No task available
			if task == nil {
				continue
			}

			// Log lock acquisition event
			// Task status is now 'running' (ClaimNextTask already updated it in the database)
			lockHistory := models.TaskHistory{
				TaskID:    task.ID,
				Status:    models.TaskStatusRunning,
				EventType: models.EventWorkerLockAcquired,
				WorkerID:  &w.workerID,
			}
			if err := w.store.InsertHistory(ctx, lockHistory); err != nil {
				slog.Error("Failed to insert lock acquired history", "task_id", task.ID, "error", err)
			}

			// Send task to worker pool (blocking)
			// This ensures tasks are never silently dropped
			// Backpressure naturally slows down polling when workers are busy
			select {
			case taskChan <- task:
				// Task sent successfully
			case <-ctx.Done():
				// Context cancelled while trying to send task
				return
			}
		}
	}
}

// workerLoop processes tasks from the task channel
func (w *Worker) workerLoop(ctx context.Context, workerNum int, taskChan <-chan *models.Task) {
	slog.Info("Worker goroutine started", "worker_num", workerNum)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Worker goroutine stopping", "worker_num", workerNum)
			return
		case task, ok := <-taskChan:
			if !ok {
				// Channel closed
				slog.Info("Task channel closed", "worker_num", workerNum)
				return
			}

			// Process the task
			if err := w.processTask(ctx, workerNum, task); err != nil {
				slog.Error("Error processing task",
					"worker_num", workerNum,
					"task_id", task.ID,
					"error", err)
			}
		}
	}
}

// processTask processes a single claimed task
func (w *Worker) processTask(ctx context.Context, workerNum int, task *models.Task) error {
	slog.Info("Claimed task",
		"worker_num", workerNum,
		"task_id", task.ID,
		"task_name", task.Name,
		"task_type", task.Type,
		"retry_count", task.RetryCount,
		"max_retries", task.MaxRetries,
	)

	// Record history: task is now running
	history := models.TaskHistory{
		TaskID:    task.ID,
		Status:    models.TaskStatusRunning,
		EventType: models.EventTaskStarted,
		WorkerID:  &w.workerID,
	}
	if err := w.store.InsertHistory(ctx, history); err != nil {
		slog.Error("Failed to insert task_started history", "task_id", task.ID, "error", err)
	}

	// Execute the task
	if err := w.executeTask(ctx, task); err != nil {
		return w.handleTaskFailure(ctx, task, err)
	}

	return w.handleTaskSuccess(ctx, task)
}

// executeTask executes the task handler with timeout
func (w *Worker) executeTask(ctx context.Context, task *models.Task) error {
	// Get the handler for this task type
	h, err := w.handlerRegistry.Get(task.Type)
	if err != nil {
		return fmt.Errorf("handler not found for type %s: %w", task.Type, err)
	}

	// Create context with timeout
	taskCtx, cancel := context.WithTimeout(ctx, w.taskTimeout)
	defer cancel()

	// Execute the handler
	slog.Info("Executing task",
		"task_id", task.ID,
		"task_type", task.Type,
		"handler_type", h.Type(),
	)

	if err := h.Execute(taskCtx, task.Payload); err != nil {
		return fmt.Errorf("task execution failed: %w", err)
	}

	return nil
}

// handleTaskSuccess handles successful task completion
func (w *Worker) handleTaskSuccess(ctx context.Context, task *models.Task) error {
	slog.Info("Task succeeded",
		"task_id", task.ID,
		"task_name", task.Name,
		"retry_count", task.RetryCount,
	)

	// Mark task as completed
	if err := w.store.CompleteTask(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	return nil
}

// handleTaskFailure handles task execution failure with retry logic
func (w *Worker) handleTaskFailure(ctx context.Context, task *models.Task, execErr error) error {
	errorMsg := execErr.Error()

	slog.Warn("Task failed",
		"task_id", task.ID,
		"task_name", task.Name,
		"retry_count", task.RetryCount,
		"max_retries", task.MaxRetries,
		"error", errorMsg,
	)

	// Schedule retry (storage layer handles retry exhaustion logic)
	if err := w.store.ScheduleRetry(ctx, task.ID, errorMsg); err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	return nil
}
