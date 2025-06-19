package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
)

// PKCEStore handles PKCE code verifier and challenge generation
type PKCEStore interface {
	GenerateCodeVerifier(length int) (string, error)
	GenerateCodeChallenge(verifier string) (string, error)
	ValidateChallenge(challenge, verifier string) bool
}

// PKCEGenerator implements PKCEStore interface
type PKCEGenerator struct{}

// NewPKCEGenerator creates a new PKCEGenerator instance
func NewPKCEGenerator() *PKCEGenerator {
	return &PKCEGenerator{}
}

// GenerateCodeVerifier generates a random code verifier string
func (p *PKCEGenerator) GenerateCodeVerifier(length int) (string, error) {
	if length < 43 || length > 128 {
		return "", fmt.Errorf("code verifier length must be between 43 and 128 characters")
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	charsetLen := big.NewInt(int64(len(charset)))
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = charset[num.Int64()]
	}

	return string(result), nil
}

// GenerateCodeChallenge generates a code challenge from the verifier
func (p *PKCEGenerator) GenerateCodeChallenge(verifier string) (string, error) {
	if verifier == "" {
		return "", fmt.Errorf("code verifier cannot be empty")
	}

	// Calculate SHA256 hash of the verifier
	hash := sha256.New()
	hash.Write([]byte(verifier))
	challenge := hash.Sum(nil)

	// Base64URL encode the hash
	encoded := base64.RawURLEncoding.EncodeToString(challenge)
	return encoded, nil
}

// ValidateChallenge validates a code challenge against a verifier
func (p *PKCEGenerator) ValidateChallenge(challenge, verifier string) bool {
	if challenge == "" || verifier == "" {
		return false
	}

	calculatedChallenge, err := p.GenerateCodeChallenge(verifier)
	if err != nil {
		return false
	}

	return challenge == calculatedChallenge
} 