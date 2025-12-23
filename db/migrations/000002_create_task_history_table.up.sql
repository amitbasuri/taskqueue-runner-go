-- Create task_history table for audit trail
CREATE TABLE IF NOT EXISTS task_history (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    status task_status NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 0,
    backoff_seconds INTEGER,
    next_run_at TIMESTAMP,
    error_message TEXT,
    worker_id VARCHAR(100),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index for fast task history lookup
CREATE INDEX idx_task_history_task_id ON task_history(task_id, created_at DESC);

-- Add comment
COMMENT ON TABLE task_history IS 'Audit trail of task status changes';
