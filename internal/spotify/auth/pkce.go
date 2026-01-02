package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const (
	// CodeVerifierLength is the length of the PKCE code verifier.
	// Spotify requires 43-128 characters; we use 64 for good entropy.
	CodeVerifierLength = 64

	// StateLength is the length of the state parameter for CSRF protection.
	StateLength = 32
)

// PKCE holds the code verifier and challenge for OAuth PKCE flow.
type PKCE struct {
	Verifier  string
	Challenge string
	State     string
}

// NewPKCE generates a new PKCE code verifier, challenge, and state.
func NewPKCE() (*PKCE, error) {
	verifier, err := generateRandomString(CodeVerifierLength)
	if err != nil {
		return nil, err
	}

	state, err := generateRandomString(StateLength)
	if err != nil {
		return nil, err
	}

	challenge := generateChallenge(verifier)

	return &PKCE{
		Verifier:  verifier,
		Challenge: challenge,
		State:     state,
	}, nil
}

// generateRandomString creates a cryptographically secure random string
// using URL-safe base64 characters (A-Z, a-z, 0-9, -, _).
func generateRandomString(length int) (string, error) {
	// We need more bytes than the target length because base64 encoding
	// expands the data. Request enough to ensure we have plenty.
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode to base64url (no padding) and trim to exact length
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded, nil
}

// generateChallenge creates the S256 code challenge from a verifier.
// challenge = base64url(sha256(verifier))
func generateChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
