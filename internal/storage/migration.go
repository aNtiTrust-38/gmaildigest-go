package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3" // Import the sqlite3 driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrationLock ensures only one migration can run at a time
var migrationLock sync.Mutex

// Migration represents a database migration
type Migration struct {
	Version     int64
	Description string
	SQL         string
}

// migrations contains all database migrations in order
var migrations = []Migration{
	{
		Version:     1,
		Description: "Create initial schema",
		SQL: `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS users (
				telegram_id INTEGER PRIMARY KEY,
				gmail_user_id TEXT UNIQUE NOT NULL,
				google_token_valid BOOLEAN NOT NULL DEFAULT FALSE,
				digest_interval INTEGER NOT NULL,
				last_digest_sent DATETIME,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);

			CREATE INDEX IF NOT EXISTS idx_users_gmail_user_id ON users(gmail_user_id);

			CREATE TABLE IF NOT EXISTS tokens (
				user_id TEXT PRIMARY KEY REFERENCES users(gmail_user_id) ON DELETE CASCADE,
				encrypted_token BLOB NOT NULL,
				nonce BLOB NOT NULL,
				created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
			);

			CREATE TABLE IF NOT EXISTS processed_emails (
				message_id TEXT NOT NULL,
				user_id TEXT NOT NULL REFERENCES users(gmail_user_id) ON DELETE CASCADE,
				processed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (message_id, user_id)
			);

			CREATE INDEX IF NOT EXISTS idx_processed_emails_user_id ON processed_emails(user_id);
			CREATE INDEX IF NOT EXISTS idx_processed_emails_processed_at ON processed_emails(processed_at);
		`,
	},
	{
		Version:     2,
		Description: "Add triggers for updated_at",
		SQL: `
			CREATE TRIGGER IF NOT EXISTS users_updated_at
			AFTER UPDATE ON users
			BEGIN
				UPDATE users SET updated_at = CURRENT_TIMESTAMP
				WHERE telegram_id = NEW.telegram_id;
			END;

			CREATE TRIGGER IF NOT EXISTS tokens_updated_at
			AFTER UPDATE ON tokens
			BEGIN
				UPDATE tokens SET updated_at = CURRENT_TIMESTAMP
				WHERE user_id = NEW.user_id;
			END;
		`,
	},
}

// Migrate applies all pending database migrations
func (s *SQLiteStorage) Migrate() error {
	sourceInstance, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Re-open a connection for the migration tool, as the driver needs it.
	db, err := sql.Open("sqlite3", s.path)
	if err != nil {
		return fmt.Errorf("failed to open db for migration: %w", err)
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"iofs",
		sourceInstance,
		"sqlite3", // The name of the database driver
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// GetMigrationStatus returns the current migration status
func (s *SQLiteStorage) GetMigrationStatus(ctx context.Context) ([]MigrationStatus, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT version, applied_at
		FROM schema_migrations
		ORDER BY version
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var status []MigrationStatus
	for rows.Next() {
		var s MigrationStatus
		err := rows.Scan(&s.Version, &s.AppliedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		status = append(status, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate migrations: %w", err)
	}

	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version   int64
	AppliedAt time.Time
}

// Transaction migration methods

// GetMigrationStatus returns the current migration status within a transaction
func (t *Transaction) GetMigrationStatus() ([]MigrationStatus, error) {
	if t.closed {
		return nil, ErrTransactionClosed
	}

	rows, err := t.tx.Query(`
		SELECT version, applied_at
		FROM schema_migrations
		ORDER BY version
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	var status []MigrationStatus
	for rows.Next() {
		var s MigrationStatus
		err := rows.Scan(&s.Version, &s.AppliedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		status = append(status, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate migrations: %w", err)
	}

	return status, nil
} 