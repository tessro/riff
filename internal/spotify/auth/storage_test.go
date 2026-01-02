package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenStorage(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "token.json")

	storage, err := NewTokenStorage(tokenPath)
	if err != nil {
		t.Fatalf("NewTokenStorage() error = %v", err)
	}

	// Initially should not exist
	if storage.Exists() {
		t.Error("Exists() = true, want false for new storage")
	}

	// Load should return nil for non-existent token
	token, err := storage.Load()
	if err != nil {
		t.Errorf("Load() error = %v", err)
	}
	if token != nil {
		t.Error("Load() should return nil for non-existent token")
	}

	// Save a token
	testToken := &Token{
		AccessToken:  "access_123",
		TokenType:    "Bearer",
		Scope:        "user-read-private",
		ExpiresIn:    3600,
		RefreshToken: "refresh_456",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	if err := storage.Save(testToken); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Now should exist
	if !storage.Exists() {
		t.Error("Exists() = false after save, want true")
	}

	// Load should return the token
	loaded, err := storage.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.AccessToken != testToken.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, testToken.AccessToken)
	}
	if loaded.RefreshToken != testToken.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, testToken.RefreshToken)
	}

	// Verify file permissions
	info, err := os.Stat(tokenPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("File permissions = %o, want 0600", mode)
	}

	// Delete should remove the token
	if err := storage.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if storage.Exists() {
		t.Error("Exists() = true after delete, want false")
	}
}

func TestTokenStorageNestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "nested", "dir", "token.json")

	storage, err := NewTokenStorage(tokenPath)
	if err != nil {
		t.Fatalf("NewTokenStorage() error = %v", err)
	}

	testToken := &Token{
		AccessToken: "test",
	}

	// Should create nested directories
	if err := storage.Save(testToken); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !storage.Exists() {
		t.Error("Token file not created in nested directory")
	}
}

func TestTokenStorageDeleteNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	tokenPath := filepath.Join(tmpDir, "nonexistent.json")

	storage, err := NewTokenStorage(tokenPath)
	if err != nil {
		t.Fatalf("NewTokenStorage() error = %v", err)
	}

	// Delete on non-existent file should not error
	if err := storage.Delete(); err != nil {
		t.Errorf("Delete() on non-existent file error = %v", err)
	}
}

func TestTokenStoragePath(t *testing.T) {
	path := "/custom/path/token.json"
	storage, err := NewTokenStorage(path)
	if err != nil {
		t.Fatalf("NewTokenStorage() error = %v", err)
	}

	if storage.Path() != path {
		t.Errorf("Path() = %q, want %q", storage.Path(), path)
	}
}
