package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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

	// Update paths to point to our dummy files
	authMap := cfgMap["auth"].(map[string]interface{})
	authMap["credentials_path"] = credsFile
	cfgMap["auth"] = authMap
	dbMap := cfgMap["db"].(map[string]interface{})
	dbMap["file_path"] = filepath.Join(tempDir, "test.db")
	cfgMap["db"] = dbMap

	// Write the new config to a temporary file
	tempCfgPath := filepath.Join(tempDir, "config.json")
	newCfgBytes, err := json.Marshal(cfgMap)
	require.NoError(t, err)
	err = os.WriteFile(tempCfgPath, newCfgBytes, 0644)
	require.NoError(t, err)

	cfg, err := config.LoadFromFile(tempCfgPath)
	require.NoError(t, err, "Failed to load test config")

	app, err := New(cfg)
	require.NoError(t, err, "Failed to create app for test")
	require.NotNil(t, app)

	// Teardown: Clean up resources
	err = app.DB.Close()
	assert.NoError(t, err, "Failed to close database connection")
}

func TestApplication_StartStop(t *testing.T) {
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

	// Update paths to point to our dummy files
	authMap := cfgMap["auth"].(map[string]interface{})
	authMap["credentials_path"] = credsFile
	cfgMap["auth"] = authMap
	dbMap := cfgMap["db"].(map[string]interface{})
	dbMap["file_path"] = filepath.Join(tempDir, "test.db")
	cfgMap["db"] = dbMap

	// Write the new config to a temporary file
	tempCfgPath := filepath.Join(tempDir, "config.json")
	newCfgBytes, err := json.Marshal(cfgMap)
	require.NoError(t, err)
	err = os.WriteFile(tempCfgPath, newCfgBytes, 0644)
	require.NoError(t, err)

	cfg, err := config.LoadFromFile(tempCfgPath)
	require.NoError(t, err, "Failed to load test config")

	// Override with test-specific values after validation
	cfg.Server.Port = 0
	cfg.Server.MetricsPort = 0

	app, err := New(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test: Start the application
	err = app.Start(ctx)
	require.NoError(t, err)

	// Allow some time for the server to start
	time.Sleep(100 * time.Millisecond)

	// Test: Stop the application
	err = app.Stop(ctx)
	assert.NoError(t, err)
} 