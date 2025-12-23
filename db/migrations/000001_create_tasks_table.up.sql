-- Create enum type for task status (only 4 essential statuses)
CREATE TYPE task_status AS ENUM ('queued', 'running', 'succeeded', 'failed');

-- Create tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    status task_status NOT NULL DEFAULT 'queued',
    priority INTEGER NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    next_run_at TIMESTAMP NOT NULL DEFAULT NOW(),
    backoff_seconds INTEGER NOT NULL DEFAULT 5,
    timeout_seconds INTEGER NOT NULL DEFAULT 30,
    locked_at TIMESTAMP,
    lock_expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for efficient queue operations
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_next_run ON tasks(next_run_at) WHERE status = 'queued';
CREATE INDEX idx_tasks_priority_created ON tasks(priority DESC, created_at ASC);
CREATE INDEX idx_tasks_lock_expires ON tasks(lock_expires_at) WHERE status = 'running';

-- Documentation
COMMENT ON TABLE tasks IS 'Background tasks queue with retry, timeout, and backoff support';
COMMENT ON COLUMN tasks.status IS 'Public task status: queued, running, succeeded, failed';
COMMENT ON COLUMN tasks.retry_count IS 'Number of retry attempts that have occurred';
COMMENT ON COLUMN tasks.max_retries IS 'Maximum number of retry attempts allowed';
COMMENT ON COLUMN tasks.backoff_seconds IS 'Base backoff value used to compute exponential retry delays';
COMMENT ON COLUMN tasks.next_run_at IS 'Earliest timestamp the task may be executed (for delays, scheduling, retry backoff)';
COMMENT ON COLUMN tasks.timeout_seconds IS 'Maximum time a worker may hold a task before it is considered stalled';
COMMENT ON COLUMN tasks.locked_at IS 'Timestamp when a worker claimed this task';
COMMENT ON COLUMN tasks.lock_expires_at IS 'Timestamp when worker lock expires; if passed, task becomes eligible for retry';
