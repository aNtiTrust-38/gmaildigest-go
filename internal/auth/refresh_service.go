package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TokenRefreshJob represents a job to refresh a user's OAuth token
type TokenRefreshJob struct {
	UserID string `json:"user_id"`
}

// TokenRefreshService handles automatic token refresh for users
type TokenRefreshService struct {
	manager *OAuthManager
}

// NewTokenRefreshService creates a new TokenRefreshService
func NewTokenRefreshService(manager *OAuthManager) *TokenRefreshService {
	return &TokenRefreshService{
		manager: manager,
	}
}

// RefreshTokens refreshes tokens for all users that need refreshing
func (s *TokenRefreshService) RefreshTokens(ctx context.Context, userIDs []string) error {
	for _, userID := range userIDs {
		if err := s.refreshUserToken(ctx, userID); err != nil {
			// Log error but continue with other users
			fmt.Printf("Error refreshing token for user %s: %v\n", userID, err)
		}
	}
	return nil
}

// refreshUserToken refreshes the token for a single user
func (s *TokenRefreshService) refreshUserToken(ctx context.Context, userID string) error {
	token, err := s.manager.getToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Check if token needs refresh (refresh 5 minutes before expiry)
	if token.Expiry.After(time.Now().Add(5 * time.Minute)) {
		return nil // Token is still valid
	}

	if err := s.manager.RefreshToken(ctx, userID); err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	return nil
}

// CreateRefreshJob creates a job payload for token refresh
func (s *TokenRefreshService) CreateRefreshJob(userID string) ([]byte, error) {
	job := TokenRefreshJob{
		UserID: userID,
	}
	return json.Marshal(job)
}

// HandleRefreshJob handles a token refresh job
func (s *TokenRefreshService) HandleRefreshJob(ctx context.Context, payload []byte) error {
	var job TokenRefreshJob
	if err := json.Unmarshal(payload, &job); err != nil {
		return fmt.Errorf("failed to unmarshal job payload: %w", err)
	}

	return s.refreshUserToken(ctx, job.UserID)
}

// GetRefreshSchedule returns the cron schedule for token refresh
func (s *TokenRefreshService) GetRefreshSchedule() string {
	// Run every hour at minute 0
	return "0 * * * *"
} 