package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gmaildigest-go/internal/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// MockStorage is a mock implementation of the auth.Storage interface for testing.
type MockStorage struct{}

func (m *MockStorage) StoreToken(ctx context.Context, userID string, token *oauth2.Token) error {
	return nil // No-op for this test
}

func (m *MockStorage) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	return nil, nil // No-op for this test
}

func TestHandlers_Login(t *testing.T) {
	// Setup a dummy authenticator for the test
	mockStorage := &MockStorage{}
	oauthManager := auth.NewOAuthManager(mockStorage, auth.NewInMemoryPKCEStore(), auth.NewInMemoryStateStore())
	// We need to load credentials to have a valid config for URL generation
	err := oauthManager.LoadCredentials("../../test/fixtures/dummy_credentials.json")
	require.NoError(t, err)

	// Manually set the redirect URL for the test config
	oauthManager.SetRedirectURL("http://localhost/auth/callback")

	app := &Application{Auth: oauthManager}

	req, err := http.NewRequest("GET", "/login", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.handleLogin)
	handler.ServeHTTP(rr, req)

	// Assert: Check for redirect
	assert.Equal(t, http.StatusSeeOther, rr.Code, "handler returned wrong status code")

	// Assert: Check that the redirect location is not empty
	location, err := rr.Result().Location()
	require.NoError(t, err, "handler did not return a location header")
	assert.NotEmpty(t, location.String(), "redirect URL should not be empty")
} 