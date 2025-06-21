package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config holds all configuration for the application.
type Config struct {
	HTTPPort      int    `json:"http_port" validate:"gte=0"`
	MetricsPort   int    `json:"metrics_port" validate:"gte=0"`
	LogLevel      string `json:"log_level" validate:"oneof=debug info warn error"`
	NumWorkers    int    `json:"num_workers" validate:"min=1"`
	DBPath        string `json:"db_path" validate:"required"`
	EncryptionKey string `json:"encryption_key" validate:"required,min=32"`

	Auth struct {
		ClientID       string `json:"client_id" validate:"required"`
		ClientSecret   string `json:"client_secret" validate:"required"`
		CredentialsPath string `json:"credentials_path" validate:"required,file"`
	} `json:"auth"`

	Telegram struct {
		BotToken string `json:"bot_token" validate:"required"`
	} `json:"telegram"`

	Scheduler struct {
		DefaultInterval Duration `json:"default_interval" validate:"min=1m"`
	} `json:"scheduler"`
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

// Load reads configuration from a file and overrides with environment variables.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if err := cfg.applyEnvOverrides(); err != nil {
		return nil, fmt.Errorf("applying environment overrides: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// applyEnvOverrides overrides config fields with environment variables.
func (c *Config) applyEnvOverrides() error {
	// Telegram overrides
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		c.Telegram.BotToken = v
	}

	// Auth overrides
	if v := os.Getenv("AUTH_CLIENT_ID"); v != "" {
		c.Auth.ClientID = v
	}
	if v := os.Getenv("AUTH_CLIENT_SECRET"); v != "" {
		c.Auth.ClientSecret = v
	}

	// HTTPPort overrides
	if v := os.Getenv("HTTP_PORT"); v != "" {
		var err error
		c.HTTPPort, err = parseInt(v)
		if err != nil {
			return fmt.Errorf("parsing HTTP_PORT: %w", err)
		}
	}

	// MetricsPort overrides
	if v := os.Getenv("METRICS_PORT"); v != "" {
		var err error
		c.MetricsPort, err = parseInt(v)
		if err != nil {
			return fmt.Errorf("parsing METRICS_PORT: %w", err)
		}
	}

	// LogLevel overrides
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}

	// DBPath overrides
	if v := os.Getenv("DB_PATH"); v != "" {
		c.DBPath = v
	}

	// EncryptionKey overrides
	if v := os.Getenv("ENCRYPTION_KEY"); v != "" {
		c.EncryptionKey = v
	}

	// Scheduler overrides
	if v := os.Getenv("SCHEDULER_DEFAULT_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing SCHEDULER_DEFAULT_INTERVAL: %w", err)
		}
		c.Scheduler.DefaultInterval = Duration{d}
	}

	return nil
}

// validate checks the configuration for errors.
func (c *Config) validate() error {
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

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
} 