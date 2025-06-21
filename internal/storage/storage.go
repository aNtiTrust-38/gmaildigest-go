package storage

import (
	"context"
)

// Storage defines the interface for low-level database operations
// required by the higher-level TokenStore.
type Storage interface {
	GetToken(ctx context.Context, userID string) ([]byte, []byte, error)
	StoreToken(ctx context.Context, userID string, token, nonce []byte) error
	DeleteToken(ctx context.Context, userID string) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	UpdateUserTelegramDetails(ctx context.Context, userID string, telegramUserID, telegramChatID int64) error
} 