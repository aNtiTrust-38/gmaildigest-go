package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gmaildigest-go/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewApplication(t *testing.T) {
	// Setup: Create a temporary config file for the test
	tempDir := t.TempDir()
	credsFile := filepath.Join(tempDir, "credentials.json")
	err := os.WriteFile(credsFile, []byte(`{"web": {"client_id": "test", "client_secret": "test", "redirect_uris": ["test"]}}`), 0644)
	require.NoError(t, err)

	exampleCfg, err := os.ReadFile("../../configs/config.example.json")
	require.NoError(t, err)

	var cfgMap map[string]interface{}
	err = json.Unmarshal(exampleCfg, &cfgMap)
	require.NoError(t, err)

	// Update credentials path to point to our dummy file
	authMap := cfgMap["auth"].(map[string]interface{})
	authMap["credentials_path"] = credsFile
	cfgMap["auth"] = authMap

	// Write the new config to a temporary file
	tempCfgPath := filepath.Join(tempDir, "config.json")
	newCfgBytes, err := json.Marshal(cfgMap)
	require.NoError(t, err)
	err = os.WriteFile(tempCfgPath, newCfgBytes, 0644)
	require.NoError(t, err)

	cfg, err := config.LoadFromFile(tempCfgPath)
	require.NoError(t, err, "Failed to load test config")

	// Test: Create a new application
	app, err := New(cfg)
	require.NoError(t, err, "New() should not return an error with a valid config")
	require.NotNil(t, app, "New() should return a non-nil application instance")

	// Assert: Check that all components are initialized
	assert.NotNil(t, app.Config, "Config should be initialized")
	assert.NotNil(t, app.Logger, "Logger should be initialized")
	assert.NotNil(t, app.DB, "Database connection should be initialized")
	assert.NotNil(t, app.WorkerPool, "WorkerPool should be initialized")
	assert.NotNil(t, app.Scheduler, "Scheduler should be initialized")
	assert.NotNil(t, app.HttpServer, "HttpServer should be initialized")

	// Teardown: Clean up resources
	err = app.DB.Close()
	assert.NoError(t, err, "Failed to close database connection")
} 