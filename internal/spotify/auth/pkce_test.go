package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestNewPKCE(t *testing.T) {
	pkce, err := NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE() error = %v", err)
	}

	// Verify verifier length
	if len(pkce.Verifier) != CodeVerifierLength {
		t.Errorf("Verifier length = %d, want %d", len(pkce.Verifier), CodeVerifierLength)
	}

	// Verify state length
	if len(pkce.State) != StateLength {
		t.Errorf("State length = %d, want %d", len(pkce.State), StateLength)
	}

	// Verify challenge is correct SHA256 of verifier
	expectedHash := sha256.Sum256([]byte(pkce.Verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(expectedHash[:])
	if pkce.Challenge != expectedChallenge {
		t.Errorf("Challenge = %q, want %q", pkce.Challenge, expectedChallenge)
	}

	// Verify uniqueness (two calls should produce different values)
	pkce2, err := NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE() second call error = %v", err)
	}
	if pkce.Verifier == pkce2.Verifier {
		t.Error("Two PKCE instances have same verifier, expected unique")
	}
	if pkce.State == pkce2.State {
		t.Error("Two PKCE instances have same state, expected unique")
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short", 16},
		{"medium", 64},
		{"long", 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := generateRandomString(tt.length)
			if err != nil {
				t.Fatalf("generateRandomString(%d) error = %v", tt.length, err)
			}
			if len(s) != tt.length {
				t.Errorf("length = %d, want %d", len(s), tt.length)
			}

			// Verify only URL-safe base64 characters
			for _, c := range s {
				if !isURLSafeBase64Char(c) {
					t.Errorf("invalid character %q in random string", c)
				}
			}
		})
	}
}

func isURLSafeBase64Char(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

func TestGenerateChallenge(t *testing.T) {
	// Test with a known verifier
	verifier := "test_verifier_string"
	challenge := generateChallenge(verifier)

	// Verify it's valid base64url
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil {
		t.Fatalf("Challenge is not valid base64url: %v", err)
	}

	// SHA256 produces 32 bytes
	if len(decoded) != 32 {
		t.Errorf("Decoded challenge length = %d, want 32", len(decoded))
	}
}
