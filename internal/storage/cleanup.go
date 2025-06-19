package storage

import (
	"context"
	"fmt"
	"time"
)

// CleanupProcessedEmails removes processed email records older than the retention period
func (s *SQLiteStorage) CleanupProcessedEmails(ctx context.Context, retentionPeriod time.Duration) (int64, error) {
	if retentionPeriod <= 0 {
		return 0, fmt.Errorf("%w: retention period must be positive", ErrInvalidInput)
	}

	query := `
		DELETE FROM processed_emails
		WHERE processed_at < datetime('now', ?)
	`
	result, err := s.db.ExecContext(ctx, query, fmt.Sprintf("-%d seconds", int64(retentionPeriod.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup processed emails: %w", err)
	}

	return result.RowsAffected()
}

// CleanupInvalidTokens removes tokens for users whose tokens are marked as invalid
func (s *SQLiteStorage) CleanupInvalidTokens(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM tokens
		WHERE user_id IN (
			SELECT gmail_user_id
			FROM users
			WHERE google_token_valid = FALSE
		)
	`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup invalid tokens: %w", err)
	}

	return result.RowsAffected()
}

// CleanupInactiveUsers removes users who haven't been active for longer than the inactivity period
func (s *SQLiteStorage) CleanupInactiveUsers(ctx context.Context, inactivityPeriod time.Duration) (int64, error) {
	if inactivityPeriod <= 0 {
		return 0, fmt.Errorf("%w: inactivity period must be positive", ErrInvalidInput)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete tokens first (due to foreign key constraints)
	_, err = tx.ExecContext(ctx, `
		DELETE FROM tokens
		WHERE user_id IN (
			SELECT gmail_user_id
			FROM users
			WHERE updated_at < datetime('now', ?)
		)`,
		fmt.Sprintf("-%d seconds", int64(inactivityPeriod.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup tokens for inactive users: %w", err)
	}

	// Then delete users
	result, err := tx.ExecContext(ctx, `
		DELETE FROM users
		WHERE updated_at < datetime('now', ?)`,
		fmt.Sprintf("-%d seconds", int64(inactivityPeriod.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup inactive users: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return deleted, nil
}

// Transaction cleanup methods

// CleanupProcessedEmails removes processed email records older than the retention period within a transaction
func (t *Transaction) CleanupProcessedEmails(retentionPeriod time.Duration) (int64, error) {
	if t.closed {
		return 0, ErrTransactionClosed
	}

	if retentionPeriod <= 0 {
		return 0, fmt.Errorf("%w: retention period must be positive", ErrInvalidInput)
	}

	query := `
		DELETE FROM processed_emails
		WHERE processed_at < datetime('now', ?)
	`
	result, err := t.tx.Exec(query, fmt.Sprintf("-%d seconds", int64(retentionPeriod.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup processed emails: %w", err)
	}

	return result.RowsAffected()
}

// CleanupInvalidTokens removes tokens for users whose tokens are marked as invalid within a transaction
func (t *Transaction) CleanupInvalidTokens() (int64, error) {
	if t.closed {
		return 0, ErrTransactionClosed
	}

	query := `
		DELETE FROM tokens
		WHERE user_id IN (
			SELECT gmail_user_id
			FROM users
			WHERE google_token_valid = FALSE
		)
	`
	result, err := t.tx.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup invalid tokens: %w", err)
	}

	return result.RowsAffected()
}

// CleanupInactiveUsers removes users who haven't been active for longer than the inactivity period within a transaction
func (t *Transaction) CleanupInactiveUsers(inactivityPeriod time.Duration) (int64, error) {
	if t.closed {
		return 0, ErrTransactionClosed
	}

	if inactivityPeriod <= 0 {
		return 0, fmt.Errorf("%w: inactivity period must be positive", ErrInvalidInput)
	}

	// Delete tokens first (due to foreign key constraints)
	_, err := t.tx.Exec(`
		DELETE FROM tokens
		WHERE user_id IN (
			SELECT gmail_user_id
			FROM users
			WHERE updated_at < datetime('now', ?)
		)`,
		fmt.Sprintf("-%d seconds", int64(inactivityPeriod.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup tokens for inactive users: %w", err)
	}

	// Then delete users
	result, err := t.tx.Exec(`
		DELETE FROM users
		WHERE updated_at < datetime('now', ?)`,
		fmt.Sprintf("-%d seconds", int64(inactivityPeriod.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup inactive users: %w", err)
	}

	return result.RowsAffected()
} 