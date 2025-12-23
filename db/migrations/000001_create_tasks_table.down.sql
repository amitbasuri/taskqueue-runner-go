-- Drop tasks table-- Drop indexes

DROP TABLE IF EXISTS tasks;DROP INDEX IF EXISTS idx_tasks_priority_created;

DROP INDEX IF EXISTS idx_tasks_status;

-- Drop task status enum

DROP TYPE IF EXISTS task_status;-- Drop tasks table

DROP TABLE IF EXISTS tasks;

-- Drop enum type
DROP TYPE IF EXISTS task_status;
