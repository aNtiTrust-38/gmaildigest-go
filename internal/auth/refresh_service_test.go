package auth

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestTokenRefreshService_RefreshTokens(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	pkceStore := &mockPKCEStore{}
	stateStore := newMockStateStore()

	manager := &OAuthManager{
		storage:    storage,
		pkceStore:  pkceStore,
		stateStore: stateStore,
	}

	service := NewTokenRefreshService(manager)

	tests := []struct {
		name     string
		userIDs  []string
		tokens   map[string]*oauth2.Token
		wantErr  bool
		setupErr error
	}{
		{
			name:    "no users",
			userIDs: []string{},
			tokens:  map[string]*oauth2.Token{},
			wantErr: false,
		},
		{
			name:    "single user with valid token",
			userIDs: []string{"user1"},
			tokens: map[string]*oauth2.Token{
				"user1": {
					AccessToken:  "valid-token",
					TokenType:    "Bearer",
					Expiry:      time.Now().Add(time.Hour),
					RefreshToken: "refresh-token",
				},
			},
			wantErr: false,
		},
		{
			name:    "single user with expired token",
			userIDs: []string{"user2"},
			tokens: map[string]*oauth2.Token{
				"user2": {
					AccessToken:  "expired-token",
					TokenType:    "Bearer",
					Expiry:      time.Now().Add(-time.Hour),
					RefreshToken: "refresh-token",
				},
			},
			wantErr: false,
		},
		{
			name:    "multiple users with mixed token states",
			userIDs: []string{"user3", "user4"},
			tokens: map[string]*oauth2.Token{
				"user3": {
					AccessToken:  "valid-token",
					TokenType:    "Bearer",
					Expiry:      time.Now().Add(time.Hour),
					RefreshToken: "refresh-token",
				},
				"user4": {
					AccessToken:  "expired-token",
					TokenType:    "Bearer",
					Expiry:      time.Now().Add(-time.Hour),
					RefreshToken: "refresh-token",
				},
			},
			wantErr: false,
		},
		{
			name:     "user with storage error",
			userIDs:  []string{"error-user"},
			tokens:   map[string]*oauth2.Token{},
			wantErr:  false, // Should not return error as we continue with other users
			setupErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup tokens in storage
			for userID, token := range tt.tokens {
				tokenBytes, err := json.Marshal(token)
				require.NoError(t, err)
				err = storage.StoreToken(ctx, userID, tokenBytes, []byte("test-nonce"))
				require.NoError(t, err)
			}

			// Set up mock token source
			manager.SetTokenSource(&mockTokenSource{
				token: &oauth2.Token{
					AccessToken:  "refreshed-token",
					TokenType:    "Bearer",
					Expiry:      time.Now().Add(time.Hour),
					RefreshToken: "refresh-token",
				},
			})

			err := service.RefreshTokens(ctx, tt.userIDs)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Verify token states
			for userID, originalToken := range tt.tokens {
				tokenBytes, _, err := storage.GetToken(ctx, userID)
				require.NoError(t, err)

				var token oauth2.Token
				err = json.Unmarshal(tokenBytes, &token)
				require.NoError(t, err)

				if originalToken.Expiry.Before(time.Now().Add(5 * time.Minute)) {
					// Token should have been refreshed
					assert.Equal(t, "refreshed-token", token.AccessToken)
					assert.True(t, token.Expiry.After(time.Now()))
				} else {
					// Token should not have been refreshed
					assert.Equal(t, originalToken.AccessToken, token.AccessToken)
				}
			}
		})
	}
}

func TestTokenRefreshService_HandleRefreshJob(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	pkceStore := &mockPKCEStore{}
	stateStore := newMockStateStore()

	manager := &OAuthManager{
		storage:    storage,
		pkceStore:  pkceStore,
		stateStore: stateStore,
	}

	service := NewTokenRefreshService(manager)

	tests := []struct {
		name    string
		userID  string
		token   *oauth2.Token
		wantErr bool
	}{
		{
			name:   "valid job with expired token",
			userID: "test-user",
			token: &oauth2.Token{
				AccessToken:  "expired-token",
				TokenType:    "Bearer",
				Expiry:      time.Now().Add(-time.Hour),
				RefreshToken: "refresh-token",
			},
			wantErr: false,
		},
		{
			name:   "valid job with valid token",
			userID: "valid-user",
			token: &oauth2.Token{
				AccessToken:  "valid-token",
				TokenType:    "Bearer",
				Expiry:      time.Now().Add(time.Hour),
				RefreshToken: "refresh-token",
			},
			wantErr: false,
		},
		{
			name:    "invalid job payload",
			userID:  "",
			token:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.token != nil {
				tokenBytes, err := json.Marshal(tt.token)
				require.NoError(t, err)
				err = storage.StoreToken(ctx, tt.userID, tokenBytes, []byte("test-nonce"))
				require.NoError(t, err)
			}

			// Set up mock token source
			manager.SetTokenSource(&mockTokenSource{
				token: &oauth2.Token{
					AccessToken:  "refreshed-token",
					TokenType:    "Bearer",
					Expiry:      time.Now().Add(time.Hour),
					RefreshToken: "refresh-token",
				},
			})

			// Create job payload
			job := TokenRefreshJob{
				UserID: tt.userID,
			}
			payload, err := json.Marshal(job)
			require.NoError(t, err)

			err = service.HandleRefreshJob(ctx, payload)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			if tt.token != nil {
				tokenBytes, _, err := storage.GetToken(ctx, tt.userID)
				require.NoError(t, err)

				var token oauth2.Token
				err = json.Unmarshal(tokenBytes, &token)
				require.NoError(t, err)

				if tt.token.Expiry.Before(time.Now().Add(5 * time.Minute)) {
					// Token should have been refreshed
					assert.Equal(t, "refreshed-token", token.AccessToken)
					assert.True(t, token.Expiry.After(time.Now()))
				} else {
					// Token should not have been refreshed
					assert.Equal(t, tt.token.AccessToken, token.AccessToken)
				}
			}
		})
	}
}

func TestTokenRefreshService_CreateRefreshJob(t *testing.T) {
	manager := &OAuthManager{}
	service := NewTokenRefreshService(manager)

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
			wantErr: false, // CreateRefreshJob doesn't validate userID
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := service.CreateRefreshJob(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			var job TokenRefreshJob
			err = json.Unmarshal(payload, &job)
			require.NoError(t, err)
			assert.Equal(t, tt.userID, job.UserID)
		})
	}
}

func TestTokenRefreshService_GetRefreshSchedule(t *testing.T) {
	manager := &OAuthManager{}
	service := NewTokenRefreshService(manager)

	schedule := service.GetRefreshSchedule()
	assert.Equal(t, "0 * * * *", schedule)
} 