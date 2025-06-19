package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_LoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	credentialsPath := filepath.Join(tmpDir, "credentials.json")

	// Create dummy credentials file
	err := os.WriteFile(credentialsPath, []byte("{}"), 0644)
	require.NoError(t, err)

	// Create valid config file
	configJSON := `{
		"telegram": {
			"bot_token": "test-token",
			"default_digest_interval": "2h"
		},
		"auth": {
			"credentials_path": "` + credentialsPath + `",
			"token_db_path": "/path/to/tokens.db",
			"token_encryption_key": "0123456789abcdef0123456789abcdef"
		},
		"gmail": {
			"forward_email": "test@example.com",
			"batch_size": 50
		},
		"summary": {
			"anthropic_api_key": "test-key",
			"timeout": "10s"
		}
	}`
	err = os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	// Test loading valid config
	cfg, err := LoadFromFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "test-token", cfg.Telegram.BotToken)
	assert.Equal(t, 2*time.Hour, cfg.Telegram.DefaultDigestInterval.Duration)
	assert.Equal(t, credentialsPath, cfg.Auth.CredentialsPath)
	assert.Equal(t, "test@example.com", cfg.Gmail.ForwardEmail)
	assert.Equal(t, 50, cfg.Gmail.BatchSize)
	assert.Equal(t, "test-key", cfg.Summary.AnthropicAPIKey)
	assert.Equal(t, 10*time.Second, cfg.Summary.Timeout.Duration)

	// Test loading non-existent file
	_, err = LoadFromFile("non-existent.json")
	assert.Error(t, err)

	// Test loading invalid JSON
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("{invalid json}"), 0644)
	require.NoError(t, err)
	_, err = LoadFromFile(invalidPath)
	assert.Error(t, err)
}

func TestConfig_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	credentialsPath := filepath.Join(tmpDir, "credentials.json")
	err := os.WriteFile(credentialsPath, []byte("{}"), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		config      Config
		shouldError bool
	}{
		{
			name: "valid config",
			config: Config{
				Telegram: struct {
					BotToken              string   `json:"bot_token" validate:"required" env:"TELEGRAM_BOT_TOKEN"`
					DefaultDigestInterval Duration `json:"default_digest_interval" validate:"min=1h" env:"TELEGRAM_DEFAULT_DIGEST_INTERVAL"`
				}{
					BotToken:              "test-token",
					DefaultDigestInterval: Duration{2 * time.Hour},
				},
				Auth: struct {
					CredentialsPath    string `json:"credentials_path" validate:"required,file" env:"AUTH_CREDENTIALS_PATH"`
					TokenDBPath        string `json:"token_db_path" validate:"required" env:"AUTH_TOKEN_DB_PATH"`
					TokenEncryptionKey string `json:"token_encryption_key" validate:"required,min=32" env:"AUTH_TOKEN_ENCRYPTION_KEY"`
				}{
					CredentialsPath:    credentialsPath,
					TokenDBPath:        "/path/to/tokens.db",
					TokenEncryptionKey: "0123456789abcdef0123456789abcdef",
				},
				Gmail: struct {
					ForwardEmail string `json:"forward_email" validate:"email" env:"GMAIL_FORWARD_EMAIL"`
					BatchSize    int    `json:"batch_size" validate:"min=1,max=100" env:"GMAIL_BATCH_SIZE"`
				}{
					ForwardEmail: "test@example.com",
					BatchSize:    50,
				},
				Summary: struct {
					AnthropicAPIKey string   `json:"anthropic_api_key" env:"SUMMARY_ANTHROPIC_API_KEY"`
					OpenAIAPIKey    string   `json:"openai_api_key" env:"SUMMARY_OPENAI_API_KEY"`
					Timeout         Duration `json:"timeout" validate:"required,min=5s" env:"SUMMARY_TIMEOUT"`
				}{
					Timeout: Duration{10 * time.Second},
				},
			},
			shouldError: false,
		},
		{
			name: "missing bot token",
			config: Config{
				Telegram: struct {
					BotToken              string   `json:"bot_token" validate:"required" env:"TELEGRAM_BOT_TOKEN"`
					DefaultDigestInterval Duration `json:"default_digest_interval" validate:"min=1h" env:"TELEGRAM_DEFAULT_DIGEST_INTERVAL"`
				}{
					DefaultDigestInterval: Duration{2 * time.Hour},
				},
			},
			shouldError: true,
		},
		{
			name: "invalid digest interval",
			config: Config{
				Telegram: struct {
					BotToken              string   `json:"bot_token" validate:"required" env:"TELEGRAM_BOT_TOKEN"`
					DefaultDigestInterval Duration `json:"default_digest_interval" validate:"min=1h" env:"TELEGRAM_DEFAULT_DIGEST_INTERVAL"`
				}{
					BotToken:              "test-token",
					DefaultDigestInterval: Duration{30 * time.Minute},
				},
			},
			shouldError: true,
		},
		{
			name: "invalid email",
			config: Config{
				Gmail: struct {
					ForwardEmail string `json:"forward_email" validate:"email" env:"GMAIL_FORWARD_EMAIL"`
					BatchSize    int    `json:"batch_size" validate:"min=1,max=100" env:"GMAIL_BATCH_SIZE"`
				}{
					ForwardEmail: "not-an-email",
					BatchSize:    50,
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_EnvironmentOverrides(t *testing.T) {
	// Set up environment variables
	os.Setenv("TELEGRAM_BOT_TOKEN", "env-token")
	os.Setenv("TELEGRAM_DEFAULT_DIGEST_INTERVAL", "3h")
	os.Setenv("GMAIL_BATCH_SIZE", "75")
	os.Setenv("SUMMARY_TIMEOUT", "15s")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("TELEGRAM_DEFAULT_DIGEST_INTERVAL")
		os.Unsetenv("GMAIL_BATCH_SIZE")
		os.Unsetenv("SUMMARY_TIMEOUT")
	}()

	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	credentialsPath := filepath.Join(tmpDir, "credentials.json")
	err := os.WriteFile(credentialsPath, []byte("{}"), 0644)
	require.NoError(t, err)

	configJSON := `{
		"telegram": {
			"bot_token": "file-token",
			"default_digest_interval": "2h"
		},
		"auth": {
			"credentials_path": "` + credentialsPath + `",
			"token_db_path": "/path/to/tokens.db",
			"token_encryption_key": "0123456789abcdef0123456789abcdef"
		},
		"gmail": {
			"forward_email": "test@example.com",
			"batch_size": 50
		},
		"summary": {
			"timeout": "10s"
		}
	}`
	err = os.WriteFile(configPath, []byte(configJSON), 0644)
	require.NoError(t, err)

	// Load config and verify environment overrides
	cfg, err := LoadFromFile(configPath)
	require.NoError(t, err)

	// Check that environment variables override file values
	assert.Equal(t, "env-token", cfg.Telegram.BotToken)
	assert.Equal(t, 3*time.Hour, cfg.Telegram.DefaultDigestInterval.Duration)
	assert.Equal(t, 75, cfg.Gmail.BatchSize)
	assert.Equal(t, 15*time.Second, cfg.Summary.Timeout.Duration)

	// Check that non-overridden values remain
	assert.Equal(t, "test@example.com", cfg.Gmail.ForwardEmail)
	assert.Equal(t, credentialsPath, cfg.Auth.CredentialsPath)
}
