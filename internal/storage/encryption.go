package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const (
	// KeySize is the required size for the encryption key (32 bytes for AES-256)
	KeySize = 32
	// NonceSize is the size of the nonce used in AES-GCM
	NonceSize = 12
)

var (
	ErrInvalidKeySize = errors.New("invalid key size: must be 32 bytes for AES-256")
	ErrInvalidNonce   = errors.New("invalid nonce size")
)

// EncryptToken encrypts a token using AES-256-GCM
func EncryptToken(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	if len(key) != KeySize {
		return nil, nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce = make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext = aesGCM.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// DecryptToken decrypts a token using AES-256-GCM
func DecryptToken(key, ciphertext, nonce []byte) (plaintext []byte, err error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(nonce) != aesGCM.NonceSize() {
		return nil, ErrInvalidNonce
	}

	plaintext, err = aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
} 