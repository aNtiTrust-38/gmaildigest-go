package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the application configuration
type Config struct {
	Telegram struct {
		BotToken              string        `json:"bot_token" validate:"required" env:"TELEGRAM_BOT_TOKEN"`
		DefaultDigestInterval Duration      `json:"default_digest_interval" validate:"min=1h" env:"TELEGRAM_DEFAULT_DIGEST_INTERVAL"`
	} `json:"telegram"`

	Auth struct {
		CredentialsPath    string `json:"credentials_path" validate:"required,file" env:"AUTH_CREDENTIALS_PATH"`
		TokenDBPath        string `json:"token_db_path" validate:"required" env:"AUTH_TOKEN_DB_PATH"`
		TokenEncryptionKey string `json:"token_encryption_key" validate:"required,min=32" env:"AUTH_TOKEN_ENCRYPTION_KEY"`
	} `json:"auth"`

	Gmail struct {
		ForwardEmail string `json:"forward_email" validate:"email" env:"GMAIL_FORWARD_EMAIL"`
		BatchSize    int    `json:"batch_size" validate:"min=1,max=100" env:"GMAIL_BATCH_SIZE"`
	} `json:"gmail"`

	Summary struct {
		AnthropicAPIKey string   `json:"anthropic_api_key" env:"SUMMARY_ANTHROPIC_API_KEY"`
		OpenAIAPIKey    string   `json:"openai_api_key" env:"SUMMARY_OPENAI_API_KEY"`
		Timeout         Duration `json:"timeout" validate:"required,min=5s" env:"SUMMARY_TIMEOUT"`
	} `json:"summary"`

	DBPath string `json:"db_path" validate:"required"`

	Worker struct {
		PoolSize int `json:"pool_size" validate:"min=1"`
	} `json:"worker"`

	Server struct {
		Port int `json:"port" validate:"min=1024,max=65535"`
	} `json:"server"`
}

// Duration is a wrapper around time.Duration that implements JSON marshaling/unmarshaling
type Duration struct {
	time.Duration
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

// MarshalJSON implements json.Marshaler
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// LoadFromFile loads configuration from a JSON file and applies environment variable overrides
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.applyEnvironmentOverrides(); err != nil {
		return nil, fmt.Errorf("applying environment overrides: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// Validate performs validation on the configuration values
func (c *Config) Validate() error {
	validate := validator.New()

	// Register custom validation for Duration
	validate.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
		if duration, ok := field.Interface().(Duration); ok {
			return duration.Duration
		}
		return nil
	}, Duration{})

	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Additional custom validations
	if _, err := os.Stat(c.Auth.CredentialsPath); err != nil {
		return fmt.Errorf("credentials file not found at %s", c.Auth.CredentialsPath)
	}

	return nil
}

// applyEnvironmentOverrides checks for environment variables and overrides config values
func (c *Config) applyEnvironmentOverrides() error {
	// Telegram overrides
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		c.Telegram.BotToken = v
	}
	if v := os.Getenv("TELEGRAM_DEFAULT_DIGEST_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing TELEGRAM_DEFAULT_DIGEST_INTERVAL: %w", err)
		}
		c.Telegram.DefaultDigestInterval = Duration{d}
	}

	// Auth overrides
	if v := os.Getenv("AUTH_CREDENTIALS_PATH"); v != "" {
		c.Auth.CredentialsPath = v
	}
	if v := os.Getenv("AUTH_TOKEN_DB_PATH"); v != "" {
		c.Auth.TokenDBPath = v
	}
	if v := os.Getenv("AUTH_TOKEN_ENCRYPTION_KEY"); v != "" {
		c.Auth.TokenEncryptionKey = v
	}

	// Gmail overrides
	if v := os.Getenv("GMAIL_FORWARD_EMAIL"); v != "" {
		c.Gmail.ForwardEmail = v
	}
	if v := os.Getenv("GMAIL_BATCH_SIZE"); v != "" {
		var err error
		c.Gmail.BatchSize, err = parseInt(v)
		if err != nil {
			return fmt.Errorf("parsing GMAIL_BATCH_SIZE: %w", err)
		}
	}

	// Summary overrides
	if v := os.Getenv("SUMMARY_ANTHROPIC_API_KEY"); v != "" {
		c.Summary.AnthropicAPIKey = v
	}
	if v := os.Getenv("SUMMARY_OPENAI_API_KEY"); v != "" {
		c.Summary.OpenAIAPIKey = v
	}
	if v := os.Getenv("SUMMARY_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing SUMMARY_TIMEOUT: %w", err)
		}
		c.Summary.Timeout = Duration{d}
	}

	// DBPath overrides
	if v := os.Getenv("DB_PATH"); v != "" {
		c.DBPath = v
	}

	// Worker overrides
	if v := os.Getenv("WORKER_POOL_SIZE"); v != "" {
		var err error
		c.Worker.PoolSize, err = parseInt(v)
		if err != nil {
			return fmt.Errorf("parsing WORKER_POOL_SIZE: %w", err)
		}
	}

	// Server overrides
	if v := os.Getenv("SERVER_PORT"); v != "" {
		var err error
		c.Server.Port, err = parseInt(v)
		if err != nil {
			return fmt.Errorf("parsing SERVER_PORT: %w", err)
		}
	}

	return nil
}

// parseInt parses a string to an integer with error handling
func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	if err != nil {
		return 0, err
	}
	return i, nil
} 