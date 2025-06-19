package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// Backup creates a backup of the database at the specified path
func (s *SQLiteStorage) Backup(ctx context.Context, backupPath string) error {
	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Begin transaction to ensure consistent backup
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create backup database
	backupDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup database: %w", err)
	}
	defer backupDB.Close()

	// Enable WAL mode for better concurrency
	_, err = backupDB.ExecContext(ctx, "PRAGMA journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Backup using SQLite's backup API
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		ATTACH DATABASE '%s' AS backup;
		
		-- Create tables in backup database
		CREATE TABLE backup.schema_migrations AS SELECT * FROM schema_migrations;
		CREATE TABLE backup.users AS SELECT * FROM users;
		CREATE TABLE backup.tokens AS SELECT * FROM tokens;
		CREATE TABLE backup.processed_emails AS SELECT * FROM processed_emails;
		
		-- Create indexes in backup database
		CREATE INDEX backup.idx_users_gmail_user_id ON users(gmail_user_id);
		CREATE INDEX backup.idx_processed_emails_user_id ON processed_emails(user_id);
		CREATE INDEX backup.idx_processed_emails_processed_at ON processed_emails(processed_at);
		
		DETACH DATABASE backup;
	`, backupPath))
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Verify backup
	err = s.verifyBackup(ctx, backupPath)
	if err != nil {
		// If verification fails, try to remove the corrupted backup
		os.Remove(backupPath)
		return fmt.Errorf("backup verification failed: %w", err)
	}

	return nil
}

// verifyBackup checks if the backup database is valid and contains all tables
func (s *SQLiteStorage) verifyBackup(ctx context.Context, backupPath string) error {
	backupDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}
	defer backupDB.Close()

	// Check if all tables exist and have data
	tables := []string{
		"schema_migrations",
		"users",
		"tokens",
		"processed_emails",
	}

	for _, table := range tables {
		var count int64
		err := backupDB.QueryRowContext(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to verify table %s: %w", table, err)
		}
	}

	// Compare row counts between source and backup
	for _, table := range tables {
		var sourceCount, backupCount int64

		err := s.db.QueryRowContext(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&sourceCount)
		if err != nil {
			return fmt.Errorf("failed to get source count for table %s: %w", table, err)
		}

		err = backupDB.QueryRowContext(ctx,
			fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&backupCount)
		if err != nil {
			return fmt.Errorf("failed to get backup count for table %s: %w", table, err)
		}

		if sourceCount != backupCount {
			return fmt.Errorf("row count mismatch for table %s: source=%d, backup=%d",
				table, sourceCount, backupCount)
		}
	}

	return nil
}

// Restore restores the database from a backup file
func (s *SQLiteStorage) Restore(ctx context.Context, backupPath string) error {
	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Drop existing tables
	_, err = tx.ExecContext(ctx, `
		DROP TABLE IF EXISTS schema_migrations;
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS tokens;
		DROP TABLE IF EXISTS processed_emails;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop existing tables: %w", err)
	}

	// Restore from backup
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		ATTACH DATABASE '%s' AS backup;
		
		-- Restore tables
		CREATE TABLE schema_migrations AS SELECT * FROM backup.schema_migrations;
		CREATE TABLE users AS SELECT * FROM backup.users;
		CREATE TABLE tokens AS SELECT * FROM backup.tokens;
		CREATE TABLE processed_emails AS SELECT * FROM backup.processed_emails;
		
		-- Restore indexes
		CREATE INDEX idx_users_gmail_user_id ON users(gmail_user_id);
		CREATE INDEX idx_processed_emails_user_id ON processed_emails(user_id);
		CREATE INDEX idx_processed_emails_processed_at ON processed_emails(processed_at);
		
		DETACH DATABASE backup;
	`, backupPath))
	if err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Transaction backup methods

// Backup creates a backup of the database at the specified path within a transaction
func (t *Transaction) Backup(backupPath string) error {
	if t.closed {
		return ErrTransactionClosed
	}

	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup database
	backupDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup database: %w", err)
	}
	defer backupDB.Close()

	// Enable WAL mode for better concurrency
	_, err = backupDB.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Backup using SQLite's backup API
	_, err = t.tx.Exec(fmt.Sprintf(`
		ATTACH DATABASE '%s' AS backup;
		
		-- Create tables in backup database
		CREATE TABLE backup.schema_migrations AS SELECT * FROM schema_migrations;
		CREATE TABLE backup.users AS SELECT * FROM users;
		CREATE TABLE backup.tokens AS SELECT * FROM tokens;
		CREATE TABLE backup.processed_emails AS SELECT * FROM processed_emails;
		
		-- Create indexes in backup database
		CREATE INDEX backup.idx_users_gmail_user_id ON users(gmail_user_id);
		CREATE INDEX backup.idx_processed_emails_user_id ON processed_emails(user_id);
		CREATE INDEX backup.idx_processed_emails_processed_at ON processed_emails(processed_at);
		
		DETACH DATABASE backup;
	`, backupPath))
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	return nil
} 