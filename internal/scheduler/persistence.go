package scheduler

import (
	"context"
	"database/sql"
)

// jobsTableSchema defines the SQL schema for the jobs table
const jobsTableSchema = `
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    type TEXT NOT NULL,
    schedule TEXT NOT NULL,
    payload TEXT,
    status TEXT NOT NULL DEFAULT 'scheduled',
    retry_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    next_run DATETIME NOT NULL,
    last_run DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, type, schedule)
);
`

// MigrateJobsTable creates the jobs table if it does not exist
func MigrateJobsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, jobsTableSchema)
	return err
} 