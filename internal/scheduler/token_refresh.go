package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

// TokenRefreshPayload represents the data needed for a token refresh job
type TokenRefreshPayload struct {
	UserID string `json:"user_id"`
}

// TokenRefreshService handles automatic token refresh for users
type TokenRefreshService struct {
	scheduler *Scheduler
	storage   Storage
	config    *oauth2.Config
	client    *http.Client
}

// NewTokenRefreshService creates a new token refresh service
func NewTokenRefreshService(scheduler *Scheduler, storage Storage, config *oauth2.Config) *TokenRefreshService {
	if scheduler == nil {
		panic("scheduler cannot be nil")
	}
	if storage == nil {
		panic("storage cannot be nil")
	}
	if config == nil {
		panic("config cannot be nil")
	}
	
	service := &TokenRefreshService{
		scheduler: scheduler,
		storage:   storage,
		config:    config,
		client:    http.DefaultClient,
	}

	// Register the token refresh handler
	scheduler.RegisterTokenRefreshHandler(service.HandleTokenRefresh)

	return service
}

// SetClient sets the HTTP client for the token refresh service
func (s *TokenRefreshService) SetClient(client *http.Client) {
	if client == nil {
		client = http.DefaultClient
	}
	s.client = client
}

// ScheduleTokenRefresh schedules a token refresh job for a user
func (s *TokenRefreshService) ScheduleTokenRefresh(ctx context.Context, userID string, schedule string) error {
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}
	if schedule == "" {
		return fmt.Errorf("schedule cannot be empty")
	}

	payload := TokenRefreshPayload{
		UserID: userID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal token refresh payload: %w", err)
	}

	_, err = s.scheduler.ScheduleJob(userID, "token_refresh", schedule, json.RawMessage(payloadBytes))
	return err
}

// HandleTokenRefresh handles a token refresh job
func (s *TokenRefreshService) HandleTokenRefresh(ctx context.Context, job *Job) error {
	if job == nil {
		return fmt.Errorf("job cannot be nil")
	}

	var payload TokenRefreshPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal token refresh payload: %w", err)
	}

	if payload.UserID == "" {
		return fmt.Errorf("userID cannot be empty in payload")
	}

	// Get the current token
	token, err := s.storage.GetToken(ctx, payload.UserID)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Check if the token needs to be refreshed
	if token.Valid() {
		// Update job status and schedule next run
		job.Status = JobStatusCompleted
		job.LastError = ""
		job.RetryCount = 0
		job.NextRun = time.Now().Add(time.Hour) // Default: refresh every hour
		return nil
	}

	// Create a context with the HTTP client
	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.client)

	// Create a token source from the existing token
	tokenSource := s.config.TokenSource(ctx, token)

	// Get a new token
	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Store the new token
	if err := s.storage.StoreToken(ctx, payload.UserID, newToken); err != nil {
		return fmt.Errorf("failed to store refreshed token: %w", err)
	}

	// Update job status and schedule next run
	job.Status = JobStatusCompleted
	job.LastError = ""
	job.RetryCount = 0
	job.NextRun = time.Now().Add(time.Hour) // Default: refresh every hour

	return nil
} 