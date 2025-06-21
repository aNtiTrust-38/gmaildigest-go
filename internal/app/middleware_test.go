package app

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gmaildigest-go/internal/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nextHandler is a dummy handler that checks for a user ID in the context.
func nextHandler(t *testing.T, expectedUserID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := getUserIDFromContext(r)
		require.True(t, ok, "user ID not found in context")
		assert.Equal(t, expectedUserID, userID)
		fmt.Fprintln(w, "next handler called")
	}
}

func TestRequireAuthMiddleware(t *testing.T) {
	// Setup
	store := session.NewInMemoryStore()
	app := &Application{
		SessionStore: store,
		Logger:       log.New(io.Discard, "", 0),
	}
	userID := "user-123"

	t.Run("with valid session", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		sessionID, err := store.Create(req.Context(), userID, time.Hour)
		require.NoError(t, err)

		req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID})
		rr := httptest.NewRecorder()

		handler := app.requireAuth(nextHandler(t, userID))
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		body, _ := io.ReadAll(rr.Body)
		assert.Contains(t, string(body), "next handler called")
	})

	t.Run("with no session cookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		rr := httptest.NewRecorder()

		// The dummy nextHandler should not be called.
		dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		handler := app.requireAuth(dummyHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
	})

	t.Run("with invalid session ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{Name: "session_id", Value: "invalid-session-id"})
		rr := httptest.NewRecorder()

		dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("next handler should not be called")
		})

		handler := app.requireAuth(dummyHandler)
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusSeeOther, rr.Code)
		assert.Equal(t, "/login", rr.Header().Get("Location"))
		// Check that the invalid cookie was cleared
		cookies := rr.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "session_id", cookies[0].Name)
		assert.Equal(t, "", cookies[0].Value)
	})
} 