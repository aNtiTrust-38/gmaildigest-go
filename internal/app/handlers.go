package app

import (
	"net/http"
)

//
// Authentication Handlers
//

// handleLogin initiates the OAuth2 flow by redirecting the user to the Google consent page.
func (a *Application) handleLogin(w http.ResponseWriter, r *http.Request) {
	// For now, we'll use a hardcoded user ID.
	// This will be replaced with session management later.
	userID := "user-123"

	authURL, _, err := a.Auth.GetAuthURL(userID)
	if err != nil {
		http.Error(w, "Failed to generate auth URL", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL, http.StatusSeeOther)
}

// handleAuthCallback handles the redirect from Google after user consent.
// It exchanges the authorization code for a token and stores it.
func (a *Application) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: Replace hardcoded userID with real session management
	userID := "user-123"

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "Invalid request: missing code or state", http.StatusBadRequest)
		return
	}

	err := a.Auth.HandleCallback(r.Context(), code, state, userID)
	if err != nil {
		a.Logger.Printf("Auth callback error: %v", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// On success, redirect to the home page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleLogout clears the user's session and token data.
func (a *Application) handleLogout(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logic to clear session/token.
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Logout handler not implemented yet."))
} 