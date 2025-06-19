package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"gmaildigest-go/internal/worker"
)

type mockStorage struct {
	tokens map[string]*oauth2.Token
	mu     sync.Mutex
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		tokens: make(map[string]*oauth2.Token),
	}
}

func (m *mockStorage) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	token, ok := m.tokens[userID]
	if !ok {
		return nil, fmt.Errorf("token not found")
	}
	// Return a copy of the token to avoid concurrent modification
	return &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:      token.Expiry,
	}, nil
}

func (m *mockStorage) StoreToken(ctx context.Context, userID string, token *oauth2.Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Store a copy of the token to avoid concurrent modification
	m.tokens[userID] = &oauth2.Token{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		RefreshToken: token.RefreshToken,
		Expiry:      token.Expiry,
	}
	return nil
}

// mockTokenSource implements oauth2.TokenSource for testing
type mockTokenSource struct {
	token *oauth2.Token
}

func (m *mockTokenSource) Token() (*oauth2.Token, error) {
	if m.token == nil {
		return nil, fmt.Errorf("mock token source not set")
	}
	return m.token, nil
}

func TestTokenRefreshService_ScheduleTokenRefresh(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create scheduler
	pool := worker.NewWorkerPool(1)
	pool.Start()
	defer pool.Stop()

	scheduler, err := NewScheduler(ctx, db, pool)
	require.NoError(t, err)

	// Create OAuth config
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
		RedirectURL: "http://localhost:8080/callback",
		Scopes:     []string{"email", "profile"},
	}

	// Create token refresh service
	service := NewTokenRefreshService(scheduler, storage, config)

	// Test scheduling a token refresh job
	err = service.ScheduleTokenRefresh(ctx, "user1", "*/5 * * * *") // Every 5 minutes
	require.NoError(t, err)

	// Start the scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Wait for the job to be scheduled
	time.Sleep(500 * time.Millisecond)

	// Verify job was scheduled
	jobs, err := scheduler.ListJobs(ctx, &ListJobsOptions{
		Type: "token_refresh",
	})
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	// Verify job details
	job := jobs[0]
	assert.Equal(t, "token_refresh", job.Type)
	assert.Equal(t, "user1", job.UserID)
	assert.Equal(t, "*/5 * * * *", job.Schedule)

	// Verify payload
	var payload TokenRefreshPayload
	err = json.Unmarshal(job.Payload, &payload)
	require.NoError(t, err)
	assert.Equal(t, "user1", payload.UserID)
}

func TestTokenRefreshService_HandleTokenRefresh(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create scheduler
	pool := worker.NewWorkerPool(1)
	pool.Start()
	defer pool.Stop()

	scheduler, err := NewScheduler(ctx, db, pool)
	require.NoError(t, err)

	// Create OAuth config with mock HTTP client
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
		RedirectURL: "http://localhost:8080/callback",
		Scopes:     []string{"email", "profile"},
	}

	// Create token refresh service
	service := NewTokenRefreshService(scheduler, storage, config)

	// Store an expired token
	expiredToken := &oauth2.Token{
		AccessToken:  "old_token",
		TokenType:    "Bearer",
		RefreshToken: "refresh_token",
		Expiry:      time.Now().Add(-1 * time.Hour),
	}
	err = storage.StoreToken(ctx, "user1", expiredToken)
	require.NoError(t, err)

	// Create a mock HTTP client
	mockClient := &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: &mockBody{
					data: []byte(`{
						"access_token": "new_token",
						"token_type": "Bearer",
						"refresh_token": "new_refresh_token",
						"expires_in": 3600
					}`),
				},
			},
		},
	}

	// Set the mock client
	service.SetClient(mockClient)

	// Create a job for testing
	payload := TokenRefreshPayload{
		UserID: "user1",
	}
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)

	job := &Job{
		ID:         "1",
		UserID:     "user1",
		Type:       "token_refresh",
		Schedule:   "*/5 * * * *",
		Status:     JobStatusPending,
		Payload:    json.RawMessage(payloadBytes),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		NextRun:    time.Now(),
		RetryCount: 0,
	}

	// Handle the token refresh
	err = service.HandleTokenRefresh(ctx, job)
	require.NoError(t, err)

	// Verify token was refreshed
	newToken, err := storage.GetToken(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, "new_token", newToken.AccessToken)
	assert.Equal(t, "new_refresh_token", newToken.RefreshToken)
	assert.True(t, newToken.Valid())
}

func TestTokenRefreshService_TokenNotExpired(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create scheduler
	pool := worker.NewWorkerPool(1)
	pool.Start()
	defer pool.Stop()

	scheduler, err := NewScheduler(ctx, db, pool)
	require.NoError(t, err)

	// Create OAuth config
	config := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://example.com/auth",
			TokenURL: "https://example.com/token",
		},
		RedirectURL: "http://localhost:8080/callback",
		Scopes:     []string{"email", "profile"},
	}

	// Create token refresh service
	service := NewTokenRefreshService(scheduler, storage, config)

	// Store a valid token
	validToken := &oauth2.Token{
		AccessToken:  "valid_token",
		TokenType:    "Bearer",
		RefreshToken: "refresh_token",
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	err = storage.StoreToken(ctx, "user1", validToken)
	require.NoError(t, err)

	// Schedule a token refresh job
	err = service.ScheduleTokenRefresh(ctx, "user1", "*/5 * * * *")
	require.NoError(t, err)

	// Start the scheduler
	scheduler.Start()
	defer scheduler.Stop()

	// Wait for the job to be executed
	time.Sleep(500 * time.Millisecond)

	// Verify token was not refreshed
	currentToken, err := storage.GetToken(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, validToken.AccessToken, currentToken.AccessToken)
}

// Mock HTTP transport and body for testing
type mockTransport struct {
	response *http.Response
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := t.response
	if resp == nil {
		return nil, fmt.Errorf("no response configured")
	}
	// Clone the response to avoid concurrent access
	clone := *resp
	if resp.Body != nil {
		clone.Body = &mockBody{data: resp.Body.(*mockBody).data}
	}
	return &clone, nil
}

type mockBody struct {
	data []byte
	pos  int
}

func (b *mockBody) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}

func (b *mockBody) Close() error {
	return nil
} 