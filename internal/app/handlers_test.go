package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gmaildigest-go/internal/auth"
	"gmaildigest-go/internal/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// MockStorage is a mock implementation of the auth.Storage interface for testing.
type MockStorage struct {
	token     *oauth2.Token
	isDeleted bool
}

func (m *MockStorage) StoreToken(ctx context.Context, userID string, token *oauth2.Token) error {
	m.token = token
	return nil
}

func (m *MockStorage) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	if m.isDeleted {
		return nil, nil
	}
	return m.token, nil
}

func (m *MockStorage) TokenWasStored() bool {
	return m.token != nil && !m.isDeleted
}

func (m *MockStorage) DeleteToken(ctx context.Context, userID string) error {
	m.isDeleted = true
	m.token = nil
	return nil
}

// MockTokenSource is a mock implementation of oauth2.TokenSource for testing.
type MockTokenSource struct{}

func (m *MockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(1 * time.Hour),
	}, nil
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

func TestHandlers_AuthCallback(t *testing.T) {
	// Setup
	mockStorage := &MockStorage{}
	pkceStore := auth.NewInMemoryPKCEStore()
	stateStore := auth.NewInMemoryStateStore()
	oauthManager := auth.NewOAuthManager(mockStorage, pkceStore, stateStore)

	// We need to set a mock token source to bypass the real token exchange
	oauthManager.SetTokenSource(&MockTokenSource{})

	// We still need to load credentials for the config to be non-nil
	err := oauthManager.LoadCredentials("../../test/fixtures/dummy_credentials.json")
	require.NoError(t, err)

	userID := "user-123"
	state := "test-state"
	verifier := "test-verifier"
	// Store the necessary state and verifier for the callback to succeed
	stateStore.StoreState(userID, state)
	pkceStore.StoreVerifier(state, verifier)

	app := &Application{
		Auth:         oauthManager,
		SessionStore: session.NewInMemoryStore(),
		Logger:       log.New(io.Discard, "", 0),
	}

	// Create a request with the necessary query parameters
	reqURL := fmt.Sprintf("/auth/callback?code=test-code&state=%s", state)
	req, err := http.NewRequest("GET", reqURL, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.handleAuthCallback)
	handler.ServeHTTP(rr, req)

	// Assert: Check for redirect to success page
	assert.Equal(t, http.StatusSeeOther, rr.Code, "handler returned wrong status code")
	location, err := rr.Result().Location()
	require.NoError(t, err)
	assert.Equal(t, "/", location.Path, "handler redirected to wrong path")

	// Assert: Check that a token was stored
	assert.True(t, mockStorage.TokenWasStored(), "token was not stored")

	// Assert: Check that a session cookie was set
	cookies := rr.Result().Cookies()
	assert.Len(t, cookies, 1, "expected exactly one cookie to be set")
	sessionCookie := cookies[0]
	assert.Equal(t, "session_id", sessionCookie.Name)
	assert.NotEmpty(t, sessionCookie.Value)
	assert.True(t, sessionCookie.HttpOnly)
}

func TestHandlers_Logout(t *testing.T) {
	// Setup
	ctx := context.Background()
	store := session.NewInMemoryStore()
	userID := "user-to-logout"

	// Create a session for the user
	sessionID, err := store.Create(ctx, userID, time.Hour)
	require.NoError(t, err)

	app := &Application{
		Logger:       log.New(io.Discard, "", 0),
		SessionStore: store,
	}

	// Create a request with the session cookie
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID,
	})

	rr := httptest.NewRecorder()

	// Execute
	handler := http.HandlerFunc(app.handleLogout)
	handler.ServeHTTP(rr, req)

	// Assert: Check for a redirect
	assert.Equal(t, http.StatusSeeOther, rr.Code)
	location, err := rr.Result().Location()
	require.NoError(t, err)
	assert.Equal(t, "/login", location.Path)

	// Assert: Check that the session cookie was cleared
	cookies := rr.Result().Cookies()
	assert.Len(t, cookies, 1, "expected exactly one cookie")
	cookie := cookies[0]
	assert.Equal(t, "session_id", cookie.Name)
	assert.Equal(t, "", cookie.Value)
	assert.NotZero(t, cookie.MaxAge, "MaxAge should be set to clear the cookie")
	assert.True(t, cookie.MaxAge < 0, "MaxAge should be negative to clear the cookie")

	// Assert: Check that the session was deleted from the store
	_, err = store.Get(ctx, sessionID)
	assert.Error(t, err, "session should have been deleted from the store")
} 