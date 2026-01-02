package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultTokenFileName is the default name for the token file.
	DefaultTokenFileName = "spotify_token.json"
)

// TokenStorage handles persisting tokens to disk.
type TokenStorage struct {
	path string
}

// NewTokenStorage creates a new token storage at the specified path.
// If path is empty, uses the default location (~/.config/riff/spotify_token.json).
func NewTokenStorage(path string) (*TokenStorage, error) {
	if path == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get config directory: %w", err)
		}
		path = filepath.Join(configDir, "riff", DefaultTokenFileName)
	}

	return &TokenStorage{path: path}, nil
}

// Save persists a token to disk.
func (s *TokenStorage) Save(token *Token) error {
	// Ensure directory exists
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Write with restricted permissions (owner only)
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// Load reads a token from disk.
func (s *TokenStorage) Load() (*Token, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No token stored yet
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token file: %w", err)
	}

	return &token, nil
}

// Delete removes the stored token.
func (s *TokenStorage) Delete() error {
	err := os.Remove(s.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}
	return nil
}

// Exists returns true if a token file exists.
func (s *TokenStorage) Exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// Path returns the path to the token file.
func (s *TokenStorage) Path() string {
	return s.path
}
