package app

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
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

	sessionID, err := a.SessionStore.Create(r.Context(), userID, 24*time.Hour)
	if err != nil {
		a.Logger.Printf("Failed to create session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	// On success, redirect to the home page.
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// handleLogout clears the user's session and token data.
func (a *Application) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		// If there's no cookie, there's nothing to do. Redirect to login.
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	sessionID := cookie.Value

	// Delete the session from the store. We ignore errors here.
	_ = a.SessionStore.Delete(r.Context(), sessionID)

	// Clear the cookie by setting its max-age to -1.
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	// Redirect to the login page.
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

//
// Application Handlers
//

// handleDashboard is a protected handler that displays a welcome message
// to the authenticated user.
func (a *Application) handleDashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserIDFromContext(r)
	if !ok {
		// This should not happen if the middleware is applied correctly.
		http.Error(w, "Could not identify user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Welcome, %s!", userID)
}

func (a *Application) handleTelegramConnect(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	telegramUserID, err := strconv.ParseInt(tokenStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}

	// The Telegram Chat ID is the same as the User ID for private chats.
	telegramChatID := telegramUserID

	userID := a.getUserIDFromContext(r)

	err = a.storage.UpdateUserTelegramDetails(r.Context(), userID, telegramUserID, telegramChatID)
	if err != nil {
		a.logger.Printf("Failed to update telegram details for user %s: %v", userID, err)
		http.Error(w, "Failed to connect account. Please try again.", http.StatusInternalServerError)
		return
	}

	a.logger.Printf("User %s successfully connected telegram account with user ID %d", userID, telegramUserID)

	// Respond with a simple success message
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Telegram account successfully connected! You can now close this window."))
} 