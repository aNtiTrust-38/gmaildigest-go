package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTransactionClosed = errors.New("transaction is already closed")
)

// Transaction represents a database transaction
type Transaction struct {
	tx     *sql.Tx
	closed bool
}

// BeginTx starts a new database transaction
func (s *SQLiteStorage) BeginTx(ctx context.Context) (*Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Transaction{tx: tx}, nil
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	if t.closed {
		return ErrTransactionClosed
	}
	t.closed = true
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	if t.closed {
		return ErrTransactionClosed
	}
	t.closed = true
	return t.tx.Rollback()
}

// CreateUser creates a new user within the transaction
func (t *Transaction) CreateUser(telegramID int64, gmailUserID string, digestInterval time.Duration) error {
	query := `
		INSERT INTO users (
			telegram_id, gmail_user_id, digest_interval
		) VALUES (?, ?, ?)
	`
	_, err := t.tx.Exec(query, telegramID, gmailUserID, int64(digestInterval.Seconds()))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by their Telegram ID within the transaction
func (t *Transaction) GetUser(telegramID int64) (*User, error) {
	user := &User{}
	var digestIntervalSecs int64
	var lastDigestSent sql.NullTime

	err := t.tx.QueryRow(`
		SELECT 
			telegram_id, gmail_user_id, digest_interval,
			last_digest_sent, google_token_valid,
			created_at, updated_at
		FROM users 
		WHERE telegram_id = ?`,
		telegramID).Scan(
		&user.TelegramID,
		&user.GmailUserID,
		&digestIntervalSecs,
		&lastDigestSent,
		&user.TokenValid,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %d", telegramID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.DigestInterval = time.Duration(digestIntervalSecs) * time.Second
	if lastDigestSent.Valid {
		user.LastDigestSent = &lastDigestSent.Time
	}

	return user, nil
}

// UpdateUser updates a user's digest interval within the transaction
func (t *Transaction) UpdateUser(telegramID int64, digestInterval time.Duration) error {
	query := `
		UPDATE users 
		SET digest_interval = ?, updated_at = CURRENT_TIMESTAMP
		WHERE telegram_id = ?
	`
	result, err := t.tx.Exec(query, int64(digestInterval.Seconds()), telegramID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %d", telegramID)
	}

	return nil
}

// StoreToken stores or updates an encrypted token and its nonce within the transaction
func (t *Transaction) StoreToken(userID string, token, nonce []byte) error {
	query := `
		INSERT INTO tokens (user_id, encrypted_token, nonce, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			encrypted_token = excluded.encrypted_token,
			nonce = excluded.nonce,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := t.tx.Exec(query, userID, token, nonce)
	if err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}
	return nil
}

// GetToken retrieves an encrypted token and its nonce within the transaction
func (t *Transaction) GetToken(userID string) ([]byte, []byte, error) {
	var token, nonce []byte
	err := t.tx.QueryRow(
		"SELECT encrypted_token, nonce FROM tokens WHERE user_id = ?",
		userID).Scan(&token, &nonce)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf("token not found for user %s", userID)
		}
		return nil, nil, fmt.Errorf("failed to get token: %w", err)
	}
	return token, nonce, nil
}

// MarkEmailProcessed marks an email as processed within the transaction
func (t *Transaction) MarkEmailProcessed(messageID, userID string) error {
	query := `
		INSERT OR REPLACE INTO processed_emails (
			message_id, user_id
		) VALUES (?, ?)
	`
	_, err := t.tx.Exec(query, messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}
	return nil
}

// IsEmailProcessed checks if an email has been processed within the transaction
func (t *Transaction) IsEmailProcessed(messageID, userID string) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM processed_emails 
			WHERE message_id = ? AND user_id = ?
		)`,
		messageID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email status: %w", err)
	}
	return exists, nil
} 