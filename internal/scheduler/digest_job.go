package scheduler

import (
	"context"
	"fmt"
	"log"

	"gmaildigest-go/internal/gmail"
	"gmaildigest-go/internal/storage"
	"gmaildigest-go/internal/summary"
	"gmaildigest-go/internal/telegram"

	"golang.org/x/oauth2"
)

// DigestJob holds the dependencies for creating and sending a digest.
type DigestJob struct {
	logger          *log.Logger
	storage         storage.Storage
	tokenStore      *storage.TokenStore
	summaryService  *summary.Service
	telegramService *telegram.Service
}

// NewDigestJob creates a new DigestJob.
func NewDigestJob(
	logger *log.Logger,
	storage storage.Storage,
	tokenStore *storage.TokenStore,
	summaryService *summary.Service,
	telegramService *telegram.Service,
) *DigestJob {
	return &DigestJob{
		logger:          logger,
		storage:         storage,
		tokenStore:      tokenStore,
		summaryService:  summaryService,
		telegramService: telegramService,
	}
}

// Run executes the digest creation and delivery process for a given user.
func (j *DigestJob) Run(userID string) error {
	j.logger.Printf("Running digest job for user %s", userID)
	ctx := context.Background()

	// 1. Get user's token from token store
	oauthToken, err := j.tokenStore.GetToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get token for user %s: %w", userID, err)
	}

	// 2. Get user from storage (for telegram details)
	user, err := j.storage.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user %s: %w", userID, err)
	}

	// 3. Create Gmail service
	gmailService, err := gmail.NewService(ctx, oauthToken, j.logger)
	if err != nil {
		return fmt.Errorf("failed to create gmail service for user %s: %w", userID, err)
	}

	// 4. Fetch unread emails
	emails, err := gmailService.FetchUnreadEmails(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch emails for user %s: %w", userID, err)
	}

	// 5. Create summary
	digest, err := j.summaryService.Summarize(ctx, emails)
	if err != nil {
		return fmt.Errorf("failed to summarize emails for user %s: %w", userID, err)
	}

	// 6. Get telegram chat ID
	if !user.TelegramChatID.Valid {
		return fmt.Errorf("user %s has not connected their telegram account", userID)
	}
	chatID := user.TelegramChatID.Int64

	// 7. Send digest
	if err := j.telegramService.SendMessage(chatID, digest); err != nil {
		return fmt.Errorf("failed to send digest to user %s: %w", userID, err)
	}

	j.logger.Printf("Successfully sent digest to user %s", userID)
	return nil
} 