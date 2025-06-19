package auth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func TestOAuthConfig_LoadCredentials(t *testing.T) {
	// Create a temporary credentials file
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials.json")
	
	credentials := map[string]interface{}{
		"web": map[string]interface{}{
			"client_id":     "test-client-id",
			"client_secret": "test-client-secret",
			"redirect_uris": []string{"http://localhost:8080/callback"},
			"auth_uri":      "https://accounts.google.com/o/oauth2/auth",
			"token_uri":     "https://oauth2.googleapis.com/token",
		},
	}
	
	credBytes, err := json.Marshal(credentials)
	require.NoError(t, err)
	
	err = os.WriteFile(credPath, credBytes, 0600)
	require.NoError(t, err)

	tests := []struct {
		name        string
		credPath    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid credentials",
			credPath: credPath,
			wantErr:  false,
		},
		{
			name:        "invalid path",
			credPath:    "/nonexistent/path",
			wantErr:     true,
			errContains: "no such file",
		},
		{
			name:        "empty credentials path",
			credPath:    "",
			wantErr:     true,
			errContains: "credentials path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &OAuthManager{}
			err := manager.LoadCredentials(tt.credPath)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, manager.config)
			assert.Equal(t, "test-client-id", manager.config.ClientID)
			assert.Equal(t, "test-client-secret", manager.config.ClientSecret)
			assert.Contains(t, manager.config.RedirectURL, "callback")
			assert.Contains(t, manager.config.Scopes, "https://www.googleapis.com/auth/gmail.readonly")
		})
	}
}

func TestOAuthManager_GetAuthURL(t *testing.T) {
	manager := &OAuthManager{
		config: &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURL:  "http://localhost:8080/callback",
			Scopes: []string{
				"https://www.googleapis.com/auth/gmail.readonly",
			},
			Endpoint: google.Endpoint,
		},
		pkceStore:  &mockPKCEStore{},
		stateStore: newMockStateStore(),
	}

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "valid user ID",
			userID:  "test-user",
			wantErr: false,
		},
		{
			name:    "empty user ID",
			userID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, state, err := manager.GetAuthURL(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, url)
				assert.Empty(t, state)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, url)
			assert.NotEmpty(t, state)
			assert.Contains(t, url, "accounts.google.com")
			assert.Contains(t, url, "state=")
			assert.Contains(t, url, "code_challenge=")
			assert.Contains(t, url, "code_challenge_method=S256")
		})
	}
}

func TestOAuthManager_ValidateToken(t *testing.T) {
	manager := &OAuthManager{}

	tests := []struct {
		name      string
		token     *oauth2.Token
		wantValid bool
	}{
		{
			name: "valid token",
			token: &oauth2.Token{
				AccessToken:  "valid-token",
				TokenType:    "Bearer",
				Expiry:      time.Now().Add(time.Hour),
				RefreshToken: "refresh-token",
			},
			wantValid: true,
		},
		{
			name: "expired token",
			token: &oauth2.Token{
				AccessToken:  "expired-token",
				TokenType:    "Bearer",
				Expiry:      time.Now().Add(-time.Hour),
				RefreshToken: "refresh-token",
			},
			wantValid: false,
		},
		{
			name:      "nil token",
			token:     nil,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := manager.ValidateToken(tt.token)
			assert.Equal(t, tt.wantValid, valid)
		})
	}
} 