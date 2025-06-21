package storage

import (
	"database/sql"
	"time"
)

// User represents a user in the system
type User struct {
	ID             string
	Email          string
	TelegramUserID sql.NullInt64
	TelegramChatID sql.NullInt64
	CreatedAt      time.Time
	UpdatedAt      time.Time
} 