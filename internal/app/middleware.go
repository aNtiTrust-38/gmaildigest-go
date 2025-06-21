package app

import (
	"context"
	"net/http"
)

// contextKey is a custom type to use as a key for context values.
type contextKey string

// userContextKey is the key for storing the user ID in the request context.
const userContextKey = contextKey("userID")

// requireAuth is a middleware that ensures a user is authenticated.
// If the user is not authenticated, it redirects them to the login page.
func (a *Application) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		sessionID := cookie.Value
		userID, err := a.SessionStore.Get(r.Context(), sessionID)
		if err != nil {
			a.Logger.Printf("middleware: failed to get session %q: %v", sessionID, err)
			// Clear the invalid cookie
			http.SetCookie(w, &http.Cookie{
				Name:   "session_id",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add the user ID to the request context
		reqWithUser := withUserID(r, userID)

		// Call the next handler in the chain
		next.ServeHTTP(w, reqWithUser)
	})
}

// withUserID adds the user ID to the request's context.
func withUserID(r *http.Request, userID string) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, userID)
	return r.WithContext(ctx)
}

// getUserIDFromContext retrieves the user ID from the request's context.
func getUserIDFromContext(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(userContextKey).(string)
	return userID, ok
} 