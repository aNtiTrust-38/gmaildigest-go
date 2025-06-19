package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty path",
			config: Config{
				Path:            "",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero max open conns",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    0,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative max open conns",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    -1,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "max idle conns greater than max open conns",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    5,
				MaxIdleConns:    10,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative max idle conns",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    -1,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero conn max lifetime",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 0,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative conn max lifetime",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: -1 * time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero conn max idle time",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 0,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative conn max idle time",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: -1 * time.Minute,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero busy timeout",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     0,
			},
			wantErr: true,
		},
		{
			name: "negative busy timeout",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				BusyTimeout:     -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "conn max idle time greater than conn max lifetime",
			config: Config{
				Path:            "test.db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Hour,
				ConnMaxIdleTime: 2 * time.Hour,
				BusyTimeout:     5 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
} 