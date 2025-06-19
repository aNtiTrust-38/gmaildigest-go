package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Config holds the database configuration
type Config struct {
	Path            string        // Path to the SQLite database file
	MaxOpenConns    int          // Maximum number of open connections
	MaxIdleConns    int          // Maximum number of idle connections
	ConnMaxLifetime time.Duration // Maximum lifetime of a connection
	ConnMaxIdleTime time.Duration // Maximum idle time of a connection
	BusyTimeout     time.Duration // SQLite busy timeout
}

// DefaultConfig returns a default database configuration
func DefaultConfig() Config {
	return Config{
		Path:            "gmail_digest.db",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
		BusyTimeout:     5 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("%w: database path cannot be empty", ErrInvalidInput)
	}

	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("%w: max open connections must be positive", ErrInvalidInput)
	}

	if c.MaxIdleConns < 0 {
		return fmt.Errorf("%w: max idle connections cannot be negative", ErrInvalidInput)
	}

	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("%w: max idle connections cannot be greater than max open connections", ErrInvalidInput)
	}

	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("%w: connection max lifetime must be positive", ErrInvalidInput)
	}

	if c.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("%w: connection max idle time must be positive", ErrInvalidInput)
	}

	if c.ConnMaxIdleTime > c.ConnMaxLifetime {
		return fmt.Errorf("%w: connection max idle time cannot be greater than max lifetime", ErrInvalidInput)
	}

	if c.BusyTimeout <= 0 {
		return fmt.Errorf("%w: busy timeout must be positive", ErrInvalidInput)
	}

	return nil
}

// OpenDatabase opens a SQLite database with the given configuration
func OpenDatabase(cfg Config) (*SQLiteStorage, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Build DSN with busy timeout and other pragmas
	dsn := fmt.Sprintf("%s?_busy_timeout=%d&_journal_mode=WAL&_synchronous=NORMAL",
		cfg.Path,
		int(cfg.BusyTimeout.Milliseconds()))

	// Open database connection
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Create storage instance
	storage := NewSQLiteStorage(db)

	// Test connection and run migrations
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := storage.Migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
} 