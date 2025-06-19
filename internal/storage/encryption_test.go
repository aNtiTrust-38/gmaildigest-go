package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenEncryption_RoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef") // 32 bytes for AES-256
	plaintext := []byte("sensitive-token-data")

	// Test encryption
	ciphertext, nonce, err := EncryptToken(key, plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEmpty(t, nonce)
	assert.NotEqual(t, plaintext, ciphertext)

	// Test decryption
	decrypted, err := DecryptToken(key, ciphertext, nonce)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestTokenEncryption_InvalidKey(t *testing.T) {
	shortKey := []byte("too-short") // Less than 32 bytes
	plaintext := []byte("sensitive-token-data")

	// Test encryption with invalid key
	_, _, err := EncryptToken(shortKey, plaintext)
	assert.Error(t, err)
}

func TestTokenEncryption_InvalidNonce(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	plaintext := []byte("sensitive-token-data")

	ciphertext, _, err := EncryptToken(key, plaintext)
	require.NoError(t, err)

	// Test decryption with invalid nonce
	invalidNonce := []byte("invalid-nonce")
	_, err = DecryptToken(key, ciphertext, invalidNonce)
	assert.Error(t, err)
}

func TestTokenEncryption_InvalidCiphertext(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	plaintext := []byte("sensitive-token-data")

	_, nonce, err := EncryptToken(key, plaintext)
	require.NoError(t, err)

	// Test decryption with invalid ciphertext
	invalidCiphertext := []byte("invalid-ciphertext")
	_, err = DecryptToken(key, invalidCiphertext, nonce)
	assert.Error(t, err)
}

func TestTokenEncryption_DifferentKey(t *testing.T) {
	key1 := []byte("0123456789abcdef0123456789abcdef")
	key2 := []byte("fedcba9876543210fedcba9876543210")
	plaintext := []byte("sensitive-token-data")

	// Encrypt with key1
	ciphertext, nonce, err := EncryptToken(key1, plaintext)
	require.NoError(t, err)

	// Try to decrypt with key2
	_, err = DecryptToken(key2, ciphertext, nonce)
	assert.Error(t, err)
} 