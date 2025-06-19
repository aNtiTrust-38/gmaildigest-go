# Gmail Digest Assistant v3.0 - Go Implementation Instructions

## Project Overview

This document provides detailed implementation instructions for migrating Gmail Digest Assistant from Python v2.0 to Go v3.0. The goal is to achieve 100% feature parity while leveraging Go's superior concurrency, single binary deployment, and eliminating Python dependency issues.

## Development Methodology

### Test-Driven Development (TDD) Approach
Each milestone follows a strict Test-Fail-Pass-Refactor cycle:
1. **Write failing tests** that define the expected behavior
2. **Implement minimal code** to make tests pass
3. **Refactor** for clarity and performance
4. **Validate** integration with existing components

### Milestone Structure
Development is organized into focused milestones, each building upon the previous with clear success criteria and comprehensive testing.

## Project Scope & Requirements

### Core Objectives
- **Complete rewrite** from Python to Go
- **100% feature parity** with Python v2.0
- **Single binary deployment** (no Docker complexity)
- **Test-driven development** throughout
- **Production-ready implementation**

### Key Features to Implement
- Gmail OAuth 2.0 authentication with token encryption
- Email fetching, parsing, and deduplication
- AI-powered summarization (Anthropic Claude API)
- Event detection and calendar integration
- Telegram bot with rich formatting and interactive features
- Background scheduling and worker pool system
- SQLite persistence layer
- Health monitoring and logging

## Architecture Design

### Project Structure
```
gmail-digest-go/
├── cmd/
│   └── gda/
│       └── main.go              # Application entry point
├── internal/
│   ├── app/                     # Application orchestration
│   ├── config/                  # Configuration management
│   ├── auth/                    # OAuth 2.0 implementation
│   ├── gmail/                   # Gmail API integration
│   ├── telegram/                # Telegram bot implementation
│   ├── summary/                 # AI summarization service
│   ├── calendar/                # Calendar integration
│   ├── storage/                 # Database layer
│   ├── scheduler/               # Background job scheduling
│   └── worker/                  # Worker pool system
├── pkg/
│   └── models/                  # Shared data models
├── test/
│   ├── integration/             # Integration tests
│   ├── fixtures/                # Test data and mocks
│   └── testutils/               # Testing utilities
├── configs/
│   └── config.example.json      # Example configuration
├── scripts/
│   └── deploy.sh               # Deployment scripts
└── docs/
    └── api.md                  # API documentation
```

### Technology Stack
- **Language**: Go 1.21+
- **Testing**: Go testing package + testify
- **Database**: SQLite with encryption
- **Authentication**: Google OAuth 2.0 with PKCE
- **APIs**: Gmail API, Telegram Bot API, Anthropic Claude API
- **Concurrency**: Goroutines and channels
- **Build**: Single static binary

## Milestone 1: Foundation & Configuration System

### Objective
Establish project foundation with configuration management and data models.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Configuration loading from JSON
func TestConfig_LoadFromFile(t *testing.T)

// Test: Environment variable override
func TestConfig_EnvironmentOverride(t *testing.T)

// Test: Configuration validation
func TestConfig_Validation(t *testing.T)

// Test: Data model serialization
func TestEmail_JSONSerialization(t *testing.T)
func TestUser_JSONSerialization(t *testing.T)
func TestDigest_JSONSerialization(t *testing.T)
```

#### Implementation Tasks
1. **Project Initialization**
   - Initialize Go module
   - Create directory structure
   - Set up testing framework
   - Implement basic Makefile with test targets

2. **Configuration System Implementation**
   - JSON-based configuration with environment variable overrides
   - Configuration validation with struct tags
   - Default value assignment
   - Error handling for missing/invalid configurations

3. **Data Models Definition**
   - Email, User, Digest, and Token models
   - JSON serialization/deserialization
   - Validation rules and constraints
   - Type safety and immutability where appropriate

#### Success Criteria
- [ ] All configuration tests pass
- [ ] Environment variable overrides work correctly
- [ ] Data models serialize/deserialize properly
- [ ] Configuration validation catches all error cases
- [ ] Test coverage > 90% for configuration and models

#### Implementation Details

**Configuration Structure:**
```go
type Config struct {
    Telegram struct {
        BotToken              string        `json:"bot_token" validate:"required"`
        DefaultDigestInterval time.Duration `json:"default_digest_interval" validate:"min=1h"`
    } `json:"telegram"`
    
    Auth struct {
        CredentialsPath    string `json:"credentials_path" validate:"required,file"`
        TokenDBPath        string `json:"token_db_path" validate:"required"`
        TokenEncryptionKey string `json:"token_encryption_key" validate:"required,min=32"`
    } `json:"auth"`
    
    Gmail struct {
        ForwardEmail string `json:"forward_email" validate:"email"`
        BatchSize    int    `json:"batch_size" validate:"min=1,max=100"`
    } `json:"gmail"`
    
    Summary struct {
        AnthropicAPIKey string        `json:"anthropic_api_key"`
        OpenAIAPIKey    string        `json:"openai_api_key"`
        Timeout         time.Duration `json:"timeout" validate:"min=5s"`
    } `json:"summary"`
}
```

**Required Test Files:**
- `internal/config/config_test.go`
- `pkg/models/email_test.go`
- `pkg/models/user_test.go`
- `pkg/models/digest_test.go`

## Milestone 2: Database Layer & Storage

### Objective
Implement SQLite storage layer with encryption and migration system.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Database connection and migration
func TestSQLiteStorage_Migrate(t *testing.T)

// Test: Token storage with encryption
func TestSQLiteStorage_StoreToken(t *testing.T)
func TestSQLiteStorage_GetToken(t *testing.T)

// Test: User management
func TestSQLiteStorage_CreateUser(t *testing.T)
func TestSQLiteStorage_GetUser(t *testing.T)
func TestSQLiteStorage_UpdateUser(t *testing.T)

// Test: Email processing tracking
func TestSQLiteStorage_MarkEmailProcessed(t *testing.T)
func TestSQLiteStorage_IsEmailProcessed(t *testing.T)

// Test: Token encryption/decryption
func TestTokenEncryption_RoundTrip(t *testing.T)

// Test: Database transaction handling
func TestSQLiteStorage_TransactionRollback(t *testing.T)
```

#### Implementation Tasks
1. **Database Schema Design**
   - Define tables for tokens, users, and processed emails
   - Create migration system with version tracking
   - Implement database connection management
   - Add transaction support

2. **Token Encryption System**
   - AES-256-GCM encryption implementation
   - Secure key derivation
   - Nonce generation and management
   - Error handling for encryption failures

3. **Storage Interface Implementation**
   - CRUD operations for all entities
   - Query optimization and indexing
   - Connection pooling
   - Graceful error handling

#### Success Criteria
- [ ] All storage tests pass with in-memory SQLite
- [ ] Token encryption/decryption works correctly
- [ ] Database migrations execute successfully
- [ ] Concurrent access handled safely
- [ ] Test coverage > 95% for storage layer

#### Implementation Details

**Database Schema:**
```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tokens (
    user_id TEXT PRIMARY KEY,
    encrypted_token BLOB NOT NULL,
    nonce BLOB NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    telegram_id INTEGER PRIMARY KEY,
    gmail_user_id TEXT UNIQUE,
    digest_interval INTEGER DEFAULT 7200,
    last_digest_sent DATETIME,
    google_token_valid BOOLEAN DEFAULT FALSE,
    preferences TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS processed_emails (
    email_id TEXT,
    user_id TEXT,
    processed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (email_id, user_id)
);

CREATE INDEX idx_users_gmail_id ON users(gmail_user_id);
CREATE INDEX idx_processed_emails_user ON processed_emails(user_id);
```

**Storage Interface:**
```go
type Storage interface {
    // Migration and lifecycle
    Migrate(ctx context.Context) error
    Close() error
    
    // Token management
    StoreToken(ctx context.Context, userID string, token *oauth2.Token) error
    GetToken(ctx context.Context, userID string) (*oauth2.Token, error)
    DeleteToken(ctx context.Context, userID string) error
    
    // User management
    CreateUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, telegramID int64) (*User, error)
    GetUserByGmailID(ctx context.Context, gmailID string) (*User, error)
    UpdateUser(ctx context.Context, user *User) error
    DeleteUser(ctx context.Context, telegramID int64) error
    
    // Email processing tracking
    IsEmailProcessed(ctx context.Context, emailID, userID string) (bool, error)
    MarkEmailProcessed(ctx context.Context, emailID, userID string) error
    GetProcessedEmailsCount(ctx context.Context, userID string) (int, error)
    
    // Cleanup operations
    CleanupOldProcessedEmails(ctx context.Context, olderThan time.Time) error
}
```

**Required Test Files:**
- `internal/storage/sqlite_test.go`
- `internal/storage/encryption_test.go`
- `internal/storage/migration_test.go`

## Milestone 3: OAuth Authentication System

### Objective
Implement Google OAuth 2.0 flow with token management and refresh logic.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: OAuth configuration setup
func TestOAuthManager_NewManager(t *testing.T)

// Test: Authorization URL generation
func TestOAuthManager_GetAuthURL(t *testing.T)

// Test: Token exchange
func TestOAuthManager_HandleCallback(t *testing.T)

// Test: Token refresh
func TestOAuthManager_RefreshToken(t *testing.T)

// Test: Token validation
func TestOAuthManager_ValidateToken(t *testing.T)

// Test: Error handling for invalid credentials
func TestOAuthManager_InvalidCredentials(t *testing.T)

// Test: Concurrent token operations
func TestOAuthManager_ConcurrentAccess(t *testing.T)
```

#### Implementation Tasks
1. **OAuth Configuration Setup**
   - Load Google OAuth credentials from JSON file
   - Configure OAuth scopes for Gmail and Calendar
   - Implement PKCE for enhanced security
   - Set up redirect URL handling

2. **Token Exchange and Management**
   - Authorization URL generation with state parameter
   - Token exchange from authorization code
   - Token validation and expiry checking
   - Automatic token refresh with retry logic

3. **Integration with Storage Layer**
   - Secure token persistence
   - Token retrieval and caching
   - Error handling for storage failures
   - Background token refresh scheduling

#### Success Criteria
- [ ] All OAuth tests pass with mocked HTTP responses
- [ ] Token refresh logic works correctly
- [ ] PKCE implementation follows security best practices
- [ ] Error scenarios handled gracefully
- [ ] Test coverage > 90% for auth package

#### Implementation Details

**OAuth Manager Interface:**
```go
type Manager interface {
    // OAuth flow
    GetAuthURL(userID string) (string, error)
    HandleCallback(ctx context.Context, code, state, userID string) error
    
    // Token management
    GetValidToken(ctx context.Context, userID string) (*oauth2.Token, error)
    RefreshToken(ctx context.Context, userID string) error
    RevokeToken(ctx context.Context, userID string) error
    
    // Token validation
    ValidateToken(ctx context.Context, token *oauth2.Token) error
    IsTokenExpired(token *oauth2.Token) bool
    
    // Background services
    StartTokenRefreshService(ctx context.Context) error
    StopTokenRefreshService() error
}
```

**Required Test Files:**
- `internal/auth/oauth_test.go`
- `internal/auth/pkce_test.go`
- `internal/auth/refresh_test.go`

## Milestone 4: Gmail API Integration

### Objective
Implement Gmail API client with email fetching, parsing, and processing capabilities.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Gmail service initialization
func TestGmailService_NewService(t *testing.T)

// Test: Email fetching with pagination
func TestGmailService_FetchUnreadEmails(t *testing.T)

// Test: Email content extraction
func TestGmailService_ExtractEmailContent(t *testing.T)

// Test: HTML content parsing
func TestGmailService_ParseHTMLContent(t *testing.T)

// Test: Attachment handling
func TestGmailService_HandleAttachments(t *testing.T)

// Test: Rate limiting
func TestGmailService_RateLimit(t *testing.T)

// Test: Error handling for API failures
func TestGmailService_APIError(t *testing.T)

// Test: Email deduplication
func TestGmailService_Deduplication(t *testing.T)
```

#### Implementation Tasks
1. **Gmail API Client Setup**
   - Initialize Gmail service with authenticated client
   - Configure API quotas and rate limiting
   - Implement retry logic with exponential backoff
   - Error handling for various API failure modes

2. **Email Fetching and Processing**
   - Fetch unread emails with pagination
   - Extract text and HTML content from messages
   - Handle multipart messages and attachments
   - Implement incremental sync using history API

3. **Content Processing Pipeline**
   - Clean and normalize email content
   - Extract metadata (sender, subject, timestamp)
   - Group emails by thread/conversation
   - Filter out spam and low-priority emails

#### Success Criteria
- [ ] All Gmail tests pass with mocked API responses
- [ ] Email content extraction handles all formats
- [ ] Rate limiting prevents API quota exceeded errors
- [ ] Pagination works correctly for large mailboxes
- [ ] Test coverage > 85% for Gmail package

#### Implementation Details

**Gmail Service Interface:**
```go
type GmailService interface {
    // Email fetching
    FetchUnreadEmails(ctx context.Context, userID string, maxResults int) ([]Email, error)
    GetEmail(ctx context.Context, userID, emailID string) (*Email, error)
    GetEmailsByThread(ctx context.Context, userID, threadID string) ([]Email, error)
    
    // Email operations
    MarkAsRead(ctx context.Context, userID, emailID string) error
    MarkAsUnread(ctx context.Context, userID, emailID string) error
    ForwardEmail(ctx context.Context, userID, emailID, forwardTo string) error
    
    // Batch operations
    ProcessBatch(ctx context.Context, userID string) (*ProcessingResult, error)
    GetUnreadCount(ctx context.Context, userID string) (int, error)
    
    // History and sync
    GetHistorySince(ctx context.Context, userID string, since time.Time) ([]Email, error)
}

type Email struct {
    ID           string            `json:"id"`
    ThreadID     string            `json:"thread_id"`
    Subject      string            `json:"subject"`
    From         string            `json:"from"`
    To           []string          `json:"to"`
    CC           []string          `json:"cc"`
    BCC          []string          `json:"bcc"`
    Body         string            `json:"body"`
    HTMLBody     string            `json:"html_body"`
    Attachments  []Attachment      `json:"attachments"`
    Timestamp    time.Time         `json:"timestamp"`
    Labels       []string          `json:"labels"`
    Headers      map[string]string `json:"headers"`
    InReplyTo    string            `json:"in_reply_to"`
    References   []string          `json:"references"`
}

type Attachment struct {
    ID          string `json:"id"`
    Filename    string `json:"filename"`
    MimeType    string `json:"mime_type"`
    Size        int64  `json:"size"`
    ContentID   string `json:"content_id"`
    Disposition string `json:"disposition"`
}
```

**Required Test Files:**
- `internal/gmail/service_test.go`
- `internal/gmail/parser_test.go`
- `internal/gmail/ratelimit_test.go`
- `test/fixtures/gmail_responses.json`

## Milestone 5: AI Summarization Service

### Objective
Implement AI-powered email summarization with multiple provider support and fallback logic.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Anthropic provider initialization
func TestAnthropicProvider_NewProvider(t *testing.T)

// Test: Email summarization
func TestAnthropicProvider_Summarize(t *testing.T)

// Test: Provider fallback chain
func TestSummaryService_ProviderFallback(t *testing.T)

// Test: Urgency detection
func TestSummaryService_DetectUrgency(t *testing.T)

// Test: Batch processing
func TestSummaryService_ProcessBatch(t *testing.T)

// Test: Error handling and retries
func TestSummaryService_ErrorHandling(t *testing.T)

// Test: Rate limiting
func TestSummaryService_RateLimit(t *testing.T)

// Test: Content preprocessing
func TestSummaryService_ContentPreprocessing(t *testing.T)
```

#### Implementation Tasks
1. **Provider Implementation**
   - Anthropic Claude API client
   - OpenAI GPT API client (fallback)
   - Local extractive summarization (final fallback)
   - Provider health monitoring and selection

2. **Summarization Engine**
   - Multi-email batch processing
   - Context-aware prompt engineering
   - Urgency detection using keywords and ML
   - Reading time estimation
   - Summary quality scoring

3. **Content Processing**
   - Email content preprocessing and cleaning
   - HTML tag removal and text extraction
   - Content length optimization for API limits
   - Thread conversation context preservation

#### Success Criteria
- [ ] All summarization tests pass with mocked API responses
- [ ] Provider fallback logic works correctly
- [ ] Urgency detection has >80% accuracy on test data
- [ ] Batch processing handles edge cases
- [ ] Test coverage > 90% for summary package

#### Implementation Details

**Summary Service Interface:**
```go
type SummaryService interface {
    // Core summarization
    SummarizeEmails(ctx context.Context, emails []Email) (*Digest, error)
    SummarizeEmail(ctx context.Context, email Email) (*Summary, error)
    
    // Analysis features
    DetectUrgency(ctx context.Context, email Email) (UrgencyLevel, error)
    ExtractActionItems(ctx context.Context, email Email) ([]ActionItem, error)
    EstimateReadingTime(content string) time.Duration
    
    // Provider management
    GetHealthyProvider(ctx context.Context) (Provider, error)
    CheckProviderHealth(ctx context.Context, provider Provider) error
}

type Provider interface {
    Name() string
    Summarize(ctx context.Context, content string, opts SummarizeOptions) (string, error)
    IsHealthy(ctx context.Context) bool
    Priority() int
    MaxContentLength() int
}

type Digest struct {
    UserID      string        `json:"user_id"`
    Emails      []Email       `json:"emails"`
    Summary     string        `json:"summary"`
    Urgency     UrgencyLevel  `json:"urgency"`
    ActionItems []ActionItem  `json:"action_items"`
    ReadingTime time.Duration `json:"reading_time"`
    Created     time.Time     `json:"created"`
    Provider    string        `json:"provider"`
}

type UrgencyLevel int

const (
    UrgencyLow UrgencyLevel = iota
    UrgencyMedium
    UrgencyHigh
    UrgencyUrgent
)

type ActionItem struct {
    Description string    `json:"description"`
    DueDate     time.Time `json:"due_date,omitempty"`
    Priority    int       `json:"priority"`
    Context     string    `json:"context"`
}
```

**Required Test Files:**
- `internal/summary/service_test.go`
- `internal/summary/anthropic_test.go`
- `internal/summary/openai_test.go`
- `internal/summary/urgency_test.go`
- `test/fixtures/email_samples.json`

## Milestone 6: Telegram Bot Implementation

### Objective
Implement comprehensive Telegram bot with command handling, rich formatting, and interactive features.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Bot initialization and configuration
func TestTelegramBot_NewBot(t *testing.T)

// Test: Command routing
func TestTelegramBot_CommandRouting(t *testing.T)

// Test: Start command flow
func TestStartCommand_Handle(t *testing.T)

// Test: Digest command
func TestDigestCommand_Handle(t *testing.T)

// Test: Settings command
func TestSettingsCommand_Handle(t *testing.T)

// Test: Message formatting
func TestMessageFormatter_FormatDigest(t *testing.T)

// Test: Inline keyboard generation
func TestKeyboardBuilder_BuildDigestKeyboard(t *testing.T)

// Test: Callback handling
func TestCallbackHandler_HandleCallback(t *testing.T)

// Test: Error message formatting
func TestTelegramBot_ErrorHandling(t *testing.T)
```

#### Implementation Tasks
1. **Bot Framework Setup**
   - Telegram Bot API client configuration
   - Command routing and middleware system
   - User session and state management
   - Error handling and recovery mechanisms

2. **Command Implementation**
   - `/start` - User onboarding and OAuth initiation
   - `/digest` - Immediate digest generation with progress
   - `/menu` - Interactive main menu with buttons
   - `/settings` - User preference management
   - `/reauthorize` - Force Google OAuth refresh
   - `/version` - System version and health information

3. **Message Formatting and UI**
   - HTML message formatting with email content preservation
   - Inline keyboard generation for actions
   - Message pagination for long content
   - Progress indicators for long-running operations
   - Attachment handling and file forwarding

#### Success Criteria
- [ ] All bot tests pass with mocked Telegram API
- [ ] Command routing works correctly
- [ ] Message formatting preserves HTML content
- [ ] Inline keyboards respond to callbacks
- [ ] Test coverage > 85% for telegram package

#### Implementation Details

**Bot Interface:**
```go
type Bot interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Message handling
    HandleUpdate(ctx context.Context, update tgbotapi.Update) error
    SendMessage(ctx context.Context, chatID int64, message Message) error
    EditMessage(ctx context.Context, chatID int64, messageID int, message Message) error
    
    // User management
    RegisterUser(ctx context.Context, telegramID int64) error
    GetUserState(ctx context.Context, telegramID int64) (UserState, error)
    SetUserState(ctx context.Context, telegramID int64, state UserState) error
}

type CommandHandler interface {
    Handle(ctx context.Context, update tgbotapi.Update) error
    Description() string
    Usage() string
    RequiresAuth() bool
}

type CallbackHandler interface {
    HandleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery) error
    CallbackPrefix() string
}

type Message struct {
    Text      string                `json:"text"`
    ParseMode string                `json:"parse_mode"`
    Keyboard  *tgbotapi.InlineKeyboardMarkup `json:"keyboard,omitempty"`
    Photo     *PhotoAttachment      `json:"photo,omitempty"`
    Document  *DocumentAttachment   `json:"document,omitempty"`
}

type UserState struct {
    Step         string                 `json:"step"`
    Data         map[string]interface{} `json:"data"`
    LastActivity time.Time              `json:"last_activity"`
}
```

**Required Test Files:**
- `internal/telegram/bot_test.go`
- `internal/telegram/commands_test.go`
- `internal/telegram/formatter_test.go`
- `internal/telegram/callbacks_test.go`

## Milestone 7: Background Services & Scheduling

### Objective
Implement background job scheduling, worker pools, and automatic digest delivery.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Scheduler initialization
func TestScheduler_NewScheduler(t *testing.T)

// Test: Job scheduling and execution
func TestScheduler_ScheduleJob(t *testing.T)

// Test: Recurring job handling
func TestScheduler_RecurringJobs(t *testing.T)

// Test: Worker pool management
func TestWorkerPool_ProcessJobs(t *testing.T)

// Test: Graceful shutdown
func TestScheduler_GracefulShutdown(t *testing.T)

// Test: Job persistence and recovery
func TestScheduler_JobPersistence(t *testing.T)

// Test: Error handling and retries
func TestScheduler_ErrorHandling(t *testing.T)

// Test: Token refresh background service
func TestTokenRefreshService_BackgroundRefresh(t *testing.T)
```

#### Implementation Tasks
1. **Job Scheduler Implementation**
   - Cron-style job scheduling with timezone support
   - Job persistence and recovery after restart
   - Priority queue for job execution
   - Dynamic job rescheduling

2. **Worker Pool System**
   - Configurable worker count with auto-scaling
   - Job queuing with backpressure handling
   - Rate limiting across workers
   - Dead letter queue for failed jobs

3. **Background Services**
   - Automatic digest generation and delivery
   - OAuth token refresh service
   - Database cleanup and maintenance
   - Health monitoring and alerting

#### Success Criteria
- [ ] All scheduler tests pass with time-mocked scenarios
- [ ] Worker pool handles concurrent jobs correctly
- [ ] Job persistence survives application restart
- [ ] Background services run without blocking main thread
- [ ] Test coverage > 90% for scheduler and worker packages

#### Implementation Details

**Scheduler Interface:**
```go
type Scheduler interface {
    // Job management
    ScheduleJob(ctx context.Context, job Job) error
    CancelJob(ctx context.Context, jobID string) error
    RescheduleJob(ctx context.Context, jobID string, newSchedule string) error
    
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Status and monitoring
    GetJobStatus(ctx context.Context, jobID string) (JobStatus, error)
    ListJobs(ctx context.Context) ([]JobInfo, error)
}

type Job interface {
    ID() string
    UserID() string
    Type() JobType
    Schedule() string  // Cron expression
    Execute(ctx context.Context) error
    Retry() RetryPolicy
}

type WorkerPool interface {
    // Pool management
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Resize(newSize int) error
    
    // Job submission
    Submit(ctx context.Context, task Task) error
    SubmitWithPriority(ctx context.Context, task Task, priority int) error
    
    // Monitoring
    Stats() PoolStats
}

type DigestJob struct {
    JobID     string    `json:"job_id"`
    UserID    string    `json:"user_id"`
    Schedule  string    `json:"schedule"`
    LastRun   time.Time `json:"last_run"`
    NextRun   time.Time `json:"next_run"`
    Enabled   bool      `json:"enabled"`
}
```

**Required Test Files:**
- `internal/scheduler/scheduler_test.go`
- `internal/worker/pool_test.go`
- `internal/scheduler/jobs_test.go`
- `internal/scheduler/persistence_test.go`

## Milestone 8: Integration & Application Orchestration

### Objective
Integrate all components into a cohesive application with health monitoring and graceful lifecycle management.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Application initialization
func TestApplication_Initialize(t *testing.T)

// Test: Service dependency injection
func TestApplication_ServiceWiring(t *testing.T)

// Test: Health check endpoints
func TestHealthMonitor_HealthChecks(t *testing.T)

// Test: Graceful shutdown
func TestApplication_GracefulShutdown(t *testing.T)

// Test: Signal handling
func TestApplication_SignalHandling(t *testing.T)

// Test: End-to-end digest flow
func TestIntegration_FullDigestFlow(t *testing.T)

// Test: Configuration hot reload
func TestApplication_ConfigReload(t *testing.T)

// Test: Error recovery
func TestApplication_ErrorRecovery(t *testing.T)
```

#### Implementation Tasks
1. **Application Orchestration**
   - Service initialization and dependency injection
   - Configuration validation and loading
   - Component lifecycle management
   - Signal handling for graceful shutdown

2. **Health Monitoring System**
   - HTTP health check endpoints
   - Service health monitoring
   - Metrics collection and reporting
   - Alert system for critical failures

3. **Integration Testing**
   - End-to-end workflow testing
   - Component integration validation
   - Performance testing under load
   - Error scenario testing

#### Success Criteria
- [ ] All integration tests pass
- [ ] Application starts and stops gracefully
- [ ] Health endpoints return accurate status
- [ ] End-to-end digest flow works correctly
- [ ] Test coverage > 80% for integration tests

#### Implementation Details

**Application Interface:**
```go
type Application interface {
    // Lifecycle
    Initialize(ctx context.Context) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Health and monitoring
    HealthCheck(ctx context.Context) (HealthStatus, error)
    GetMetrics(ctx context.Context) (Metrics, error)
    
    // Configuration
    ReloadConfig(ctx context.Context) error
    GetConfig() *config.Config
}

type HealthStatus struct {
    Status    string                    `json:"status"`
    Services  map[string]ServiceHealth  `json:"services"`
    Timestamp time.Time                 `json:"timestamp"`
    Version   string                    `json:"version"`
}

type ServiceHealth struct {
    Status      string        `json:"status"`
    LastCheck   time.Time     `json:"last_check"`
    ResponseTime time.Duration `json:"response_time"`
    Error       string        `json:"error,omitempty"`
}
```

**Health Check Endpoints:**
- `GET /health` - Overall system health
- `GET /health/gmail` - Gmail API connectivity
- `GET /health/telegram` - Telegram bot status
- `GET /health/database` - Database connectivity
- `GET /health/summary` - AI service availability
- `GET /metrics` - Prometheus-compatible metrics

**Required Test Files:**
- `test/integration/full_flow_test.go`
- `test/integration/health_test.go`
- `internal/app/app_test.go`
- `internal/health/monitor_test.go`

## Milestone 9: Build, Deployment & Production Readiness

### Objective
Create production-ready build system with deployment automation and monitoring.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Binary compilation and version embedding
func TestBuild_BinaryCreation(t *testing.T)

// Test: Configuration validation in production mode
func TestDeployment_ConfigValidation(t *testing.T)

// Test: Systemd service integration
func TestDeployment_SystemdIntegration(t *testing.T)

// Test: Migration from development to production
func TestDeployment_DataMigration(t *testing.T)

// Test: Binary size and performance
func TestBuild_PerformanceMetrics(t *testing.T)

// Test: Cross-platform compilation
func TestBuild_CrossPlatform(t *testing.T)

// Test: Security hardening
func TestSecurity_FilePermissions(t *testing.T)
func TestSecurity_EnvironmentVariables(t *testing.T)

// Test: Backup and recovery
func TestOperations_BackupRestore(t *testing.T)
```

#### Implementation Tasks
1. **Build System**
   - Multi-platform binary compilation
   - Version embedding and build metadata
   - Static linking for portable binaries
   - Build optimization and size reduction

2. **Deployment Automation**
   - Systemd service configuration
   - Installation and upgrade scripts
   - Configuration management
   - Database migration automation

3. **Production Hardening**
   - Security best practices implementation
   - File permission management
   - Environment variable security
   - Logging and audit trails

4. **Operational Tools**
   - Backup and recovery procedures
   - Monitoring and alerting setup
   - Performance tuning guidelines
   - Troubleshooting documentation

#### Success Criteria
- [ ] Single binary builds successfully for all target platforms
- [ ] Deployment scripts work on clean systems
- [ ] Security audit passes all checks
- [ ] Performance meets specified requirements
- [ ] Operational procedures documented and tested

#### Implementation Details

**Build Configuration:**
```makefile
# Makefile
VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

.PHONY: test build build-static cross-compile docker clean

# Development build
build:
	go build -ldflags="$(LDFLAGS)" -o bin/gda cmd/gda/main.go

# Production build with optimizations
build-prod:
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -trimpath -o bin/gda cmd/gda/main.go

# Static binary for containerless deployment
build-static:
	CGO_ENABLED=1 go build -ldflags="$(LDFLAGS) -extldflags=-static" -trimpath -o bin/gda-static cmd/gda/main.go

# Cross-compilation targets
cross-compile:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-linux-amd64 cmd/gda/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-linux-arm64 cmd/gda/main.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-darwin-amd64 cmd/gda/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="$(LDFLAGS)" -o bin/gda-darwin-arm64 cmd/gda/main.go

# Testing targets
test:
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	go test -v -tags=integration ./test/integration/...

benchmark:
	go test -bench=. -benchmem ./...

# Linting and quality checks
lint:
	golangci-lint run ./...

security-scan:
	gosec ./...

# Deployment helpers
install: build-prod
	sudo ./scripts/install.sh

uninstall:
	sudo ./scripts/uninstall.sh

clean:
	rm -rf bin/ coverage.out
```

**Systemd Service Configuration:**
```ini
# /etc/systemd/system/gmail-digest.service
[Unit]
Description=Gmail Digest Assistant v3.0
Documentation=https://github.com/yourusername/gmail-digest-go
After=network-online.target
Wants=network-online.target

[Service]
Type=notify
User=gda
Group=gda
ExecStart=/opt/gda/bin/gda
ExecReload=/bin/kill -HUP $MAINPID
WorkingDirectory=/opt/gda
Restart=always
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=30

# Security hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/gda/data
CapabilityBoundingSet=
AmbientCapabilities=
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM

# Environment
Environment=GDA_CONFIG_PATH=/opt/gda/config.json
EnvironmentFile=-/opt/gda/environment

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=gmail-digest

[Install]
WantedBy=multi-user.target
```

**Installation Script:**
```bash
#!/bin/bash
# scripts/install.sh

set -euo pipefail

INSTALL_DIR="/opt/gda"
SERVICE_USER="gda"
SERVICE_FILE="/etc/systemd/system/gmail-digest.service"

# Check if running as root
if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root" 
   exit 1
fi

# Create service user
if ! id "$SERVICE_USER" &>/dev/null; then
    useradd --system --home-dir "$INSTALL_DIR" --shell /bin/false "$SERVICE_USER"
fi

# Create directory structure
mkdir -p "$INSTALL_DIR"/{bin,data,logs,configs}
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
chmod 755 "$INSTALL_DIR"
chmod 700 "$INSTALL_DIR"/data

# Copy binary
cp bin/gda "$INSTALL_DIR/bin/"
chmod 755 "$INSTALL_DIR/bin/gda"
chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/bin/gda"

# Copy configuration
if [[ ! -f "$INSTALL_DIR/config.json" ]]; then
    cp configs/config.example.json "$INSTALL_DIR/config.json"
    chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR/config.json"
    chmod 600 "$INSTALL_DIR/config.json"
fi

# Install systemd service
cp scripts/gmail-digest.service "$SERVICE_FILE"
systemctl daemon-reload

# Enable but don't start service
systemctl enable gmail-digest

echo "Gmail Digest Assistant installed successfully!"
echo "1. Edit /opt/gda/config.json with your settings"
echo "2. Set environment variables in /opt/gda/environment"
echo "3. Start the service: systemctl start gmail-digest"
```

**Required Test Files:**
- `test/build/build_test.go`
- `test/deployment/systemd_test.go`
- `test/security/security_test.go`
- `scripts/test-install.sh`

## Final Milestone: Documentation & Quality Assurance

### Objective
Complete comprehensive documentation, final testing, and quality assurance.

### TDD Requirements

#### Test Cases to Implement First
```go
// Test: Documentation completeness
func TestDocumentation_APIReference(t *testing.T)

// Test: Example configurations
func TestDocumentation_ExampleConfigs(t *testing.T)

// Test: Performance benchmarks
func TestPerformance_Benchmarks(t *testing.T)

// Test: Load testing
func TestPerformance_LoadTest(t *testing.T)

// Test: Memory leak detection
func TestPerformance_MemoryLeaks(t *testing.T)

// Test: Error scenario coverage
func TestReliability_ErrorScenarios(t *testing.T)

// Test: Upgrade path validation
func TestUpgrade_MigrationPath(t *testing.T)
```

#### Implementation Tasks
1. **Documentation Creation**
   - API reference documentation
   - Installation and configuration guides
   - Troubleshooting documentation
   - Performance tuning guides

2. **Quality Assurance**
   - Comprehensive test suite validation
   - Performance benchmarking
   - Security audit and penetration testing
   - Code quality review

3. **Final Validation**
   - End-to-end system testing
   - Load testing and stress testing
   - Production deployment validation
   - User acceptance testing

#### Success Criteria
- [ ] All documentation complete and accurate
- [ ] Test coverage > 85% across all packages
- [ ] Performance benchmarks meet requirements
- [ ] Security audit passes
- [ ] Production deployment successful

## Performance Requirements

### Baseline Performance Targets
- **Startup Time**: < 5 seconds from process start to ready
- **Memory Usage**: < 100MB during normal operation
- **Email Processing**: > 100 emails/minute per user
- **Telegram Response**: < 2 seconds for command responses
- **Concurrent Users**: Support 100+ active users
- **Database Operations**: < 100ms for 95th percentile queries
- **API Calls**: Respect rate limits with <1% error rate

### Benchmarking Requirements
```go
// Example benchmark tests
func BenchmarkEmailProcessing(b *testing.B)
func BenchmarkDigestGeneration(b *testing.B)
func BenchmarkTelegramFormatting(b *testing.B)
func BenchmarkDatabaseOperations(b *testing.B)
```

## Security Requirements

### Security Checklist
- [ ] Token encryption using AES-256-GCM
- [ ] Secure environment variable handling
- [ ] Input validation and sanitization
- [ ] SQL injection prevention
- [ ] Rate limiting on all external APIs
- [ ] Secure file permissions
- [ ] Audit logging for sensitive operations
- [ ] Regular security dependency updates

### Security Testing
```go
// Security test examples
func TestSecurity_TokenEncryption(t *testing.T)
func TestSecurity_InputValidation(t *testing.T)
func TestSecurity_SQLInjection(t *testing.T)
func TestSecurity_RateLimiting(t *testing.T)
```

## Monitoring and Observability

### Metrics Collection
```go
type Metrics struct {
    // Application metrics
    EmailsProcessed   prometheus.Counter
    DigestsSent      prometheus.Counter
    ErrorsTotal      prometheus.CounterVec
    
    // Performance metrics
    RequestDuration  prometheus.HistogramVec
    MemoryUsage     prometheus.Gauge
    GoroutineCount  prometheus.Gauge
    
    // Business metrics
    ActiveUsers     prometheus.Gauge
    APICallsTotal   prometheus.CounterVec
    DatabaseQueries prometheus.HistogramVec
}
```

### Health Check Endpoints
- `GET /health` - Overall system health
- `GET /health/ready` - Readiness probe
- `GET /health/live` - Liveness probe
- `GET /metrics` - Prometheus metrics
- `GET /version` - Version information

## Error Handling Strategy

### Error Categories
1. **Recoverable Errors**: Retry with exponential backoff
2. **User Errors**: Return helpful error messages
3. **System Errors**: Log and alert, graceful degradation
4. **Security Errors**: Log and alert, block request

### Error Response Format
```go
type ErrorResponse struct {
    Error   string            `json:"error"`
    Code    string            `json:"code"`
    Details map[string]string `json:"details,omitempty"`
    TraceID string            `json:"trace_id"`
}
```

## Testing Strategy Summary

### Test Coverage Requirements
- **Unit Tests**: > 90% coverage for business logic
- **Integration Tests**: > 80% coverage for component interactions
- **End-to-End Tests**: Cover all major user workflows
- **Performance Tests**: Validate all performance requirements
- **Security Tests**: Cover all security requirements

### Test Execution
```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Run benchmarks
make benchmark

# Run security scans
make security-scan
```

## Risk Mitigation

### Technical Risks
1. **OAuth Complexity**: Comprehensive error handling and user guidance
2. **API Rate Limits**: Implement proper rate limiting and backoff
3. **Memory Leaks**: Regular profiling and monitoring
4. **Database Corruption**: Implement backup and recovery

### Timeline Risks
1. **Scope Creep**: Stick to defined milestone objectives
2. **Integration Issues**: Continuous integration testing
3. **Performance Problems**: Early performance testing
4. **Security Vulnerabilities**: Regular security reviews

## Success Criteria Summary

### Functional Requirements
- [ ] All Python v2.0 features implemented with 100% parity
- [ ] Gmail OAuth authentication working reliably
- [ ] Email fetching and processing functional
- [ ] AI summarization operational with fallback
- [ ] Telegram bot responding to all commands
- [ ] Background scheduling and delivery working
- [ ] Single binary deployment successful

### Non-Functional Requirements
- [ ] Performance targets met under load
- [ ] Security requirements satisfied
- [ ] Test coverage above specified thresholds
- [ ] Documentation complete and accurate
- [ ] Production deployment successful
- [ ] Operational procedures validated

## Milestone Dependencies

```
M1 (Foundation) → M2 (Database) → M3 (Auth) → M4 (Gmail) → M5 (AI) → M6 (Telegram) → M7 (Background) → M8 (Integration) → M9 (Deploy) → Final (QA)
```

### Critical Path
The critical path runs through: Foundation → Database → Auth → Gmail → AI → Telegram, as each component depends on the previous ones. Background services, deployment, and documentation can be developed in parallel during later milestones.

### Milestone Flexibility
- Milestones 1-6 must be completed sequentially
- Milestones 7-9 can overlap with careful coordination
- Final milestone can begin once milestone 8 core features are complete

---

This milestone-based implementation plan provides a structured approach to developing Gmail Digest Assistant v3.0 in Go using test-driven development methodology. Each milestone builds upon the previous one while maintaining clear success criteria and comprehensive testing throughout the development process.