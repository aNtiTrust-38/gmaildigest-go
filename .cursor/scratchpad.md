# Background and Motivation
We are now proceeding in a linear, milestone-by-milestone fashion, starting from Milestone 1. The goal is to build a robust, maintainable, and test-driven foundation for Gmail Digest Assistant v3.0, ensuring each layer is solid before moving to the next.

# Key Challenges and Analysis
- Ensuring configuration is flexible, secure, and testable
- Supporting environment variable overrides for config
- Defining data models that are serializable and validated
- Achieving high test coverage from the start

# High-level Task Breakdown (Milestone 1: Foundation & Configuration System)
1. Write failing tests for configuration loading from JSON (`internal/config/config_test.go`)
2. Write failing tests for environment variable override
3. Write failing tests for configuration validation
4. Write failing tests for data model serialization:
   - Email (`pkg/models/email_test.go`)
   - User (`pkg/models/user_test.go`)
   - Digest (`pkg/models/digest_test.go`)
5. Implement minimal code to make each test pass, one feature at a time
6. Refactor for clarity and performance
7. Validate integration with existing components (if any)
8. Commit and push after each major milestone

# Project Status Board
- [ ] Write config loading test
- [ ] Write env var override test
- [ ] Write config validation test
- [ ] Write Email model serialization test
- [ ] Write User model serialization test
- [ ] Write Digest model serialization test
- [ ] Implement config loading
- [ ] Implement env var override
- [ ] Implement config validation
- [ ] Implement Email model
- [ ] Implement User model
- [ ] Implement Digest model
- [ ] Refactor and validate
- [ ] Pass all tests
- [ ] Final review and documentation

# Executor's Feedback or Assistance Requests
- Ready to begin Milestone 1 with TDD, starting with configuration tests.

# Lessons
- Proceeding linearly ensures a solid foundation and reduces technical debt.
- Always start with failing tests and build up functionality incrementally.

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
- [x] Implement cron parser tests and implementation
- [x] Implement job persistence schema:
  - [x] Write tests for job persistence:
    - [x] Test schema migration
    - [x] Test job CRUD operations
    - [x] Test unique constraint enforcement
    - [x] Test dead letter handling
    - [x] Test job status transitions
  - [x] Implement schema migration:
    - [x] Create jobs table with all required columns
    - [x] Add unique constraint
    - [x] Add indexes for performance
  - [x] Implement job persistence operations:
    - [x] CreateJob
    - [x] GetJob
    - [x] UpdateJob
    - [x] ListJobs
    - [x] DeleteJob
- [x] Implement Scheduler core logic:
  - [x] Job scheduling and deduplication
  - [x] Job dispatch to WorkerPool
  - [x] Job status management
  - [x] Job retry handling
  - [x] Dead letter queue handling
- [ ] Implement TokenRefreshService:
  - [ ] Design TokenRefreshService interface
  - [ ] Implement token refresh job type
  - [ ] Add token refresh scheduling logic
  - [ ] Add token refresh error handling
- [ ] Final review and documentation

# Executor's Feedback or Assistance Requests
- All tests are passing for the job persistence and scheduler implementations
- Next step is to implement the TokenRefreshService

# Key Challenges and Analysis
1. Job Persistence:
   - Successfully implemented SQLite-based job persistence with proper schema
   - Added support for generic JSON payloads
   - Implemented deduplication and retry tracking
   - Added dead letter queue handling

2. Scheduler Core:
   - Integrated with WorkerPool for job execution
   - Implemented job scheduling with cron expressions
   - Added support for job deduplication and retry logic
   - Implemented graceful shutdown handling

3. Next Steps:
   - Design and implement TokenRefreshService
   - Add token refresh job type and scheduling logic
   - Add comprehensive error handling for token refresh failures

# Lessons
- Use proper type definitions and interfaces for better code organization
- Implement comprehensive test coverage for all components
- Handle edge cases in job scheduling and execution
- Use proper error handling and logging
- Implement proper cleanup and shutdown handling

# Background and Motivation
The Gmail Digest Assistant v3.0 requires robust background scheduling and services to handle token refresh, email fetching, and digest generation. The current milestone focuses on implementing these background services with proper job scheduling, persistence, and error handling.

# Current Status / Progress Tracking
- [x] Milestone 1-6 (to be reviewed after completing Milestone 7)
- [x] Milestone 7 - Background Scheduling & Services (In Progress)
  - [x] Job Persistence Implementation
  - [x] Scheduler Core Implementation
  - [ ] TokenRefreshService Implementation 