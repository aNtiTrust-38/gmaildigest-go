package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
)

// User represents a user in the system
type User struct {
	TelegramID     int64
	GmailUserID    string
	DigestInterval time.Duration
	LastDigestSent *time.Time
	TokenValid     bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SQLiteStorage handles all database operations
type SQLiteStorage struct {
	db   *sql.DB
	path string
}

// NewSQLiteStorage creates a new SQLiteStorage instance
func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &SQLiteStorage{db: db, path: path}, nil
}

// validateInput checks if the input parameters are valid
func validateInput(telegramID int64, gmailUserID string, digestInterval time.Duration) error {
	if telegramID <= 0 {
		return fmt.Errorf("%w: telegram ID must be positive", ErrInvalidInput)
	}
	if gmailUserID == "" {
		return fmt.Errorf("%w: gmail user ID cannot be empty", ErrInvalidInput)
	}
	if digestInterval <= 0 {
		return fmt.Errorf("%w: digest interval must be positive", ErrInvalidInput)
	}
	return nil
}

// validateTokenInput checks if the token input parameters are valid
func validateTokenInput(userID string, token, nonce []byte) error {
	if userID == "" {
		return fmt.Errorf("%w: user ID cannot be empty", ErrInvalidInput)
	}
	if len(token) == 0 {
		return fmt.Errorf("%w: token cannot be empty", ErrInvalidInput)
	}
	if len(nonce) == 0 {
		return fmt.Errorf("%w: nonce cannot be empty", ErrInvalidInput)
	}
	return nil
}

// validateEmailInput checks if the email input parameters are valid
func validateEmailInput(messageID, userID string) error {
	if messageID == "" {
		return fmt.Errorf("%w: message ID cannot be empty", ErrInvalidInput)
	}
	if userID == "" {
		return fmt.Errorf("%w: user ID cannot be empty", ErrInvalidInput)
	}
	return nil
}

// StoreToken stores or updates an encrypted token and its nonce
func (s *SQLiteStorage) StoreToken(ctx context.Context, userID string, token, nonce []byte) error {
	if err := validateTokenInput(userID, token, nonce); err != nil {
		return err
	}

	query := `INSERT OR REPLACE INTO tokens (user_id, encrypted_token, nonce) VALUES (?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, userID, token, nonce)
	return err
}

// DeleteToken removes a token from the database.
func (s *SQLiteStorage) DeleteToken(ctx context.Context, userID string) error {
	query := `DELETE FROM tokens WHERE user_id = ?`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// GetToken retrieves an encrypted token and its nonce
func (s *SQLiteStorage) GetToken(ctx context.Context, userID string) ([]byte, []byte, error) {
	if userID == "" {
		return nil, nil, fmt.Errorf("%w: user ID cannot be empty", ErrInvalidInput)
	}

	var token, nonce []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT encrypted_token, nonce FROM tokens WHERE user_id = ?",
		userID).Scan(&token, &nonce)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf("%w: token not found for user %s", ErrNotFound, userID)
		}
		return nil, nil, fmt.Errorf("failed to get token: %w", err)
	}
	return token, nonce, nil
}

// CreateUser creates a new user
func (s *SQLiteStorage) CreateUser(ctx context.Context, telegramID int64, gmailUserID string, digestInterval time.Duration) error {
	if err := validateInput(telegramID, gmailUserID, digestInterval); err != nil {
		return err
	}

	query := `
		INSERT INTO users (
			telegram_id, gmail_user_id, digest_interval
		) VALUES (?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query, telegramID, gmailUserID, int64(digestInterval.Seconds()))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by their Telegram ID
func (s *SQLiteStorage) GetUser(ctx context.Context, telegramID int64) (*User, error) {
	if telegramID <= 0 {
		return nil, fmt.Errorf("%w: telegram ID must be positive", ErrInvalidInput)
	}

	user := &User{}
	var digestIntervalSecs int64
	var lastDigestSent sql.NullTime

	err := s.db.QueryRowContext(ctx, `
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
			return nil, fmt.Errorf("%w: user not found with ID %d", ErrNotFound, telegramID)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.DigestInterval = time.Duration(digestIntervalSecs) * time.Second
	if lastDigestSent.Valid {
		user.LastDigestSent = &lastDigestSent.Time
	}

	return user, nil
}

// UpdateUser updates a user's digest interval
func (s *SQLiteStorage) UpdateUser(ctx context.Context, telegramID int64, digestInterval time.Duration) error {
	if telegramID <= 0 {
		return fmt.Errorf("%w: telegram ID must be positive", ErrInvalidInput)
	}
	if digestInterval <= 0 {
		return fmt.Errorf("%w: digest interval must be positive", ErrInvalidInput)
	}

	query := `
		UPDATE users 
		SET digest_interval = ?, updated_at = CURRENT_TIMESTAMP
		WHERE telegram_id = ?
	`
	result, err := s.db.ExecContext(ctx, query, int64(digestInterval.Seconds()), telegramID)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: user not found with ID %d", ErrNotFound, telegramID)
	}

	return nil
}

// MarkEmailProcessed marks an email as processed for a user
func (s *SQLiteStorage) MarkEmailProcessed(ctx context.Context, messageID, userID string) error {
	if err := validateEmailInput(messageID, userID); err != nil {
		return err
	}

	query := `
		INSERT OR REPLACE INTO processed_emails (
			message_id, user_id
		) VALUES (?, ?)
	`
	_, err := s.db.ExecContext(ctx, query, messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark email as processed: %w", err)
	}
	return nil
}

// IsEmailProcessed checks if an email has been processed
func (s *SQLiteStorage) IsEmailProcessed(ctx context.Context, messageID, userID string) (bool, error) {
	if err := validateEmailInput(messageID, userID); err != nil {
		return false, err
	}

	var exists bool
	err := s.db.QueryRowContext(ctx, `
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

func (s *SQLiteStorage) UpdateUserTelegramDetails(ctx context.Context, userID string, telegramUserID, telegramChatID int64) error {
	query := `UPDATE users SET telegram_user_id = ?, telegram_chat_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := s.db.ExecContext(ctx, query, telegramUserID, telegramChatID, userID)
	if err != nil {
		return fmt.Errorf("failed to update user telegram details: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
} 