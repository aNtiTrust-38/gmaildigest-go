package scheduler

import (
	"context"
	"golang.org/x/oauth2"
)

// Storage defines the interface required by the TokenRefreshService
// for handling high-level OAuth2 token operations.
type Storage interface {
	// GetToken retrieves a token for a given user ID
	GetToken(ctx context.Context, userID string) (*oauth2.Token, error)

	// StoreToken stores a token for a given user ID
	StoreToken(ctx context.Context, userID string, token *oauth2.Token) error
} 