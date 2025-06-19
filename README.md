Gmail Digest Assistant v3.0 is a background-processing Go application designed to fetch, summarize, and deliver Gmail digest emails via Telegram. It is a milestone-driven, TDD-based project with a focus on concurrency safety, OAuth integration, metrics and observability, and production readiness. See `instructions.md` for detailed roadmap.

## Background Job Scheduling & Persistence (Milestone 7)

### Minimal Cron Parser
- Implements a 5-field cron syntax: `minute hour day month weekday` (e.g., `0 9 * * 1-5` for 9am on weekdays).
- Supports `*`, single values, comma-separated lists, and ranges (e.g., `1,15,30` or `1-5`).
- No support for step values (e.g., `*/5`) or named days/months.
- Used to schedule recurring jobs for digest delivery, token refresh, and maintenance.
- See `internal/scheduler/cron.go` for implementation.

### Job Persistence Schema
- Jobs are persisted in a SQLite table (`jobs`) to ensure reliability and recovery after restart.
- Schema includes:
  - `id` (UUID), `user_id`, `type`, `schedule`, `payload` (JSON), `status`, `retry_count`, `last_error`, `next_run`, `last_run`, `created_at`, `updated_at`
  - Unique constraint on (`user_id`, `type`, `schedule`) for deduplication
  - Dead letter handling: jobs with `retry_count >= 10` and `status = 'dead'`
- Migration logic is in `internal/scheduler/persistence.go`.

### WorkerPool: Concurrency, Retry, and Dead Letter Logic
- The WorkerPool manages concurrent processing of background jobs using a configurable number of worker goroutines.
- Tasks are submitted to a buffered queue; if the queue is full, submission fails (backpressure).
- Each task is retried up to 10 times on failure. If it still fails, it is moved to a dead letter queue for inspection.
- See `internal/worker/pool.go` for implementation and `internal/worker/pool_test.go` for tests.

These components provide the foundation for robust, concurrency-safe background processing and reliable job management in Gmail Digest Assistant v3.0.