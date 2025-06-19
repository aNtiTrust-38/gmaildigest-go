# Background and Motivation
Milestone 7 focuses on implementing background job scheduling, worker pools, and automatic digest delivery for Gmail Digest Assistant v3.0 in Go. The goal is to provide robust, concurrency-safe background processing with persistent job management and reliable token refresh.

# Key Challenges and Analysis
- Ensuring concurrency safety in scheduling and worker pool design
- Implementing a minimal cron parser for flexible scheduling
- Guaranteeing job deduplication by type/user
- Handling job retries and dead letter queue after repeated failures
- Supporting generic job payloads for extensibility
- Persisting jobs in SQLite and recovering on restart

# High-level Task Breakdown
1. Flesh out and expand test cases for Scheduler, WorkerPool, and Persistence, including:
   - Job deduplication
   - Retry and dead letter handling
   - Generic payload support
2. Design and implement a minimal cron parser (Go, no external library)
   - 5-field cron: minute, hour, day, month, weekday
   - Support for *, single values, lists, and ranges
   - No step values or named days/months
   - Data structure: CronSchedule with Next(time.Time) method
3. Define and implement the database schema for job persistence (SQLite, unique constraints)
   - Table: jobs
   - Columns: id, user_id, type, schedule, payload, status, retry_count, last_error, next_run, last_run, created_at, updated_at
   - Unique constraint: (user_id, type, schedule)
   - Dead letter: retry_count >= 10 and status = 'dead'
4. Implement Scheduler with:
   - Deduplication logic
   - Cron-based scheduling
   - Persistence and recovery
   - Graceful shutdown
5. Implement WorkerPool with:
   - Configurable concurrency
   - Retry and dead letter logic
   - Monitoring/stats
6. Integrate TokenRefreshService as a scheduled, deduplicated job
7. Write and run tests for all above functionality
8. Commit and push at each major milestone

# Project Status Board
- [x] Create test skeletons for Scheduler, WorkerPool, Persistence (TDD)
- [x] Expand test cases for deduplication, retry, dead letter, and generic payloads
- [x] Outline minimal cron parser and job persistence schema
- [ ] Implement minimal cron parser
- [ ] Implement job persistence schema
- [ ] Implement Scheduler core logic
- [ ] Implement WorkerPool core logic
- [ ] Integrate TokenRefreshService
- [ ] Pass all tests
- [ ] Final review and documentation

# Executor's Feedback or Assistance Requests
- Test skeletons and expanded cases committed and pushed.
- Minimal cron parser and job persistence schema outlined.
- Ready to implement cron parser and persistence schema next.

# Lessons
- Confirm directory structure before creating files.
- Always clarify requirements for deduplication, retries, and payloads before implementation.
- Commit and push after each major milestone to ensure version control and traceability. 