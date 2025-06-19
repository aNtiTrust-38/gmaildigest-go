package storage

import (
	"context"
	"fmt"
	"time"
)

// Metrics represents system-wide metrics
type Metrics struct {
	TotalUsers      int64     // Total number of users
	ActiveUsers     int64     // Number of users with valid tokens
	ProcessedEmails int64     // Total number of processed emails
	ValidTokens     int64     // Number of valid tokens
	CollectedAt     time.Time // When these metrics were collected
}

// UserMetrics represents user-specific metrics
type UserMetrics struct {
	TelegramID      int64     // User's Telegram ID
	GmailUserID     string    // User's Gmail ID
	ProcessedEmails int64     // Number of processed emails
	HasValidToken   bool      // Whether the user has a valid token
	LastActive      time.Time // Last activity timestamp
	DigestInterval  time.Duration // User's digest interval
}

// GetMetrics retrieves system-wide metrics
func (s *SQLiteStorage) GetMetrics(ctx context.Context) (*Metrics, error) {
	metrics := &Metrics{
		CollectedAt: time.Now(),
	}

	// Get total users and active users
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COUNT(CASE WHEN google_token_valid = TRUE THEN 1 END)
		FROM users
	`).Scan(&metrics.TotalUsers, &metrics.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	// Get total processed emails
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM processed_emails
	`).Scan(&metrics.ProcessedEmails)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed emails count: %w", err)
	}

	// Get valid tokens count
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM tokens t
		JOIN users u ON t.user_id = u.gmail_user_id
		WHERE u.google_token_valid = TRUE
	`).Scan(&metrics.ValidTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid tokens count: %w", err)
	}

	return metrics, nil
}

// GetMetricsWithinTimeRange retrieves system-wide metrics within a specific time range
func (s *SQLiteStorage) GetMetricsWithinTimeRange(ctx context.Context, start, end time.Time) (*Metrics, error) {
	if end.Before(start) {
		return nil, fmt.Errorf("%w: end time cannot be before start time", ErrInvalidInput)
	}

	metrics := &Metrics{
		CollectedAt: time.Now(),
	}

	// Get total users and active users as of end time
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*),
			COUNT(CASE WHEN google_token_valid = TRUE THEN 1 END)
		FROM users
		WHERE created_at <= ? AND (updated_at >= ? OR updated_at >= ?)
	`, end, start, end).Scan(&metrics.TotalUsers, &metrics.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	// Get processed emails within time range
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM processed_emails
		WHERE processed_at BETWEEN ? AND ?
	`, start, end).Scan(&metrics.ProcessedEmails)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed emails count: %w", err)
	}

	// Get valid tokens count as of end time
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM tokens t
		JOIN users u ON t.user_id = u.gmail_user_id
		WHERE u.google_token_valid = TRUE
		AND t.created_at <= ?
		AND (t.updated_at >= ? OR t.updated_at >= ?)
	`, end, start, end).Scan(&metrics.ValidTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid tokens count: %w", err)
	}

	return metrics, nil
}

// GetUserMetrics retrieves metrics for a specific user
func (s *SQLiteStorage) GetUserMetrics(ctx context.Context, telegramID int64) (*UserMetrics, error) {
	if telegramID <= 0 {
		return nil, fmt.Errorf("%w: telegram ID must be positive", ErrInvalidInput)
	}

	metrics := &UserMetrics{
		TelegramID: telegramID,
	}

	// Get user information
	var digestIntervalSecs int64
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			gmail_user_id,
			google_token_valid,
			digest_interval,
			updated_at
		FROM users
		WHERE telegram_id = ?
	`, telegramID).Scan(
		&metrics.GmailUserID,
		&metrics.HasValidToken,
		&digestIntervalSecs,
		&metrics.LastActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user information: %w", err)
	}

	metrics.DigestInterval = time.Duration(digestIntervalSecs) * time.Second

	// Get processed emails count
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM processed_emails
		WHERE user_id = ?
	`, metrics.GmailUserID).Scan(&metrics.ProcessedEmails)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed emails count: %w", err)
	}

	return metrics, nil
}

// Transaction metrics methods

// GetMetrics retrieves system-wide metrics within a transaction
func (t *Transaction) GetMetrics() (*Metrics, error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}

	metrics := &Metrics{
		CollectedAt: time.Now(),
	}

	// Get total users and active users
	err := t.tx.QueryRow(`
		SELECT 
			COUNT(*),
			COUNT(CASE WHEN google_token_valid = TRUE THEN 1 END)
		FROM users
	`).Scan(&metrics.TotalUsers, &metrics.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get user metrics: %w", err)
	}

	// Get total processed emails
	err = t.tx.QueryRow(`
		SELECT COUNT(*)
		FROM processed_emails
	`).Scan(&metrics.ProcessedEmails)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed emails count: %w", err)
	}

	// Get valid tokens count
	err = t.tx.QueryRow(`
		SELECT COUNT(*)
		FROM tokens t
		JOIN users u ON t.user_id = u.gmail_user_id
		WHERE u.google_token_valid = TRUE
	`).Scan(&metrics.ValidTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid tokens count: %w", err)
	}

	return metrics, nil
}

// GetUserMetrics retrieves metrics for a specific user within a transaction
func (t *Transaction) GetUserMetrics(telegramID int64) (*UserMetrics, error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}

	if telegramID <= 0 {
		return nil, fmt.Errorf("%w: telegram ID must be positive", ErrInvalidInput)
	}

	metrics := &UserMetrics{
		TelegramID: telegramID,
	}

	// Get user information
	var digestIntervalSecs int64
	err := t.tx.QueryRow(`
		SELECT 
			gmail_user_id,
			google_token_valid,
			digest_interval,
			updated_at
		FROM users
		WHERE telegram_id = ?
	`, telegramID).Scan(
		&metrics.GmailUserID,
		&metrics.HasValidToken,
		&digestIntervalSecs,
		&metrics.LastActive,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user information: %w", err)
	}

	metrics.DigestInterval = time.Duration(digestIntervalSecs) * time.Second

	// Get processed emails count
	err = t.tx.QueryRow(`
		SELECT COUNT(*)
		FROM processed_emails
		WHERE user_id = ?
	`, metrics.GmailUserID).Scan(&metrics.ProcessedEmails)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed emails count: %w", err)
	}

	return metrics, nil
} 