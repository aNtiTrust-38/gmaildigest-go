package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPKCE_GenerateCodeVerifier(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{
			name:    "valid length - 43",
			length:  43,
			wantErr: false,
		},
		{
			name:    "valid length - 128",
			length:  128,
			wantErr: false,
		},
		{
			name:    "invalid length - too short",
			length:  42,
			wantErr: true,
		},
		{
			name:    "invalid length - too long",
			length:  129,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkce := NewPKCEGenerator()
			verifier, err := pkce.GenerateCodeVerifier(tt.length)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, verifier)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, verifier, tt.length)
			assert.Regexp(t, "^[A-Za-z0-9._~-]+$", verifier)
		})
	}
}

func TestPKCE_GenerateCodeChallenge(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
		want     string
		wantErr  bool
	}{
		{
			name:     "valid verifier",
			verifier: "test-verifier-123",
			want:     base64URLEncode(sha256.New().Sum([]byte("test-verifier-123"))),
			wantErr:  false,
		},
		{
			name:     "empty verifier",
			verifier: "",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkce := NewPKCEGenerator()
			challenge, err := pkce.GenerateCodeChallenge(tt.verifier)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, challenge)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, challenge)
			assert.Regexp(t, "^[A-Za-z0-9._~-]+$", challenge)
		})
	}
}

func TestPKCE_ValidateChallenge(t *testing.T) {
	pkce := NewPKCEGenerator()
	verifier, err := pkce.GenerateCodeVerifier(43)
	require.NoError(t, err)

	challenge, err := pkce.GenerateCodeChallenge(verifier)
	require.NoError(t, err)

	tests := []struct {
		name      string
		challenge string
		verifier  string
		want      bool
	}{
		{
			name:      "valid pair",
			challenge: challenge,
			verifier:  verifier,
			want:      true,
		},
		{
			name:      "invalid verifier",
			challenge: challenge,
			verifier:  "wrong-verifier",
			want:      false,
		},
		{
			name:      "empty challenge",
			challenge: "",
			verifier:  verifier,
			want:      false,
		},
		{
			name:      "empty verifier",
			challenge: challenge,
			verifier:  "",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := pkce.ValidateChallenge(tt.challenge, tt.verifier)
			assert.Equal(t, tt.want, valid)
		})
	}
}

func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
} 