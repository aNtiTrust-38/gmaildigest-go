package storage

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2"
	"io"
)

// This key should be 32 bytes and loaded securely from config.
// Using a placeholder for now.
var encryptionKey = []byte("12345678901234567890123456789012")

// TokenStore handles the logic for storing and retrieving OAuth2 tokens,
// including encryption and decryption.
type TokenStore struct {
	db Storage
}

// NewTokenStore creates a new TokenStore.
func NewTokenStore(db Storage) *TokenStore {
	return &TokenStore{db: db}
}

// GetToken retrieves a decrypted oauth2.Token for a user.
func (ts *TokenStore) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	encryptedToken, nonce, err := ts.db.GetToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get encrypted token from db: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}

	decryptedData, err := aesgcm.Open(nil, nonce, encryptedToken, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(decryptedData, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// StoreToken encrypts and stores an oauth2.Token for a user.
func (ts *TokenStore) StoreToken(ctx context.Context, userID string, token *oauth2.Token) error {
	if token == nil {
		return errors.New("token cannot be nil")
	}

	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create gcm: %w", err)
	}

	encryptedToken := aesgcm.Seal(nil, nonce, tokenBytes, nil)

	return ts.db.StoreToken(ctx, userID, encryptedToken, nonce)
} 