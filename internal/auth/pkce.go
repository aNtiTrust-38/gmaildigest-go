package auth

// PKCEStore defines the interface for storing and retrieving PKCE code verifiers.
type PKCEStore interface {
	GenerateCodeVerifier(length int) (string, error)
	GenerateCodeChallenge(verifier string) (string, error)
	StoreVerifier(state, verifier string) error
	GetVerifier(state string) (string, error)
	ValidateChallenge(challenge, verifier string) bool
} 