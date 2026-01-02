package auth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCallbackServer(t *testing.T) {
	// Create server on random port (0)
	server, err := NewCallbackServer(0)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}

	server.Start()
	defer func() { _ = server.Shutdown(context.Background()) }()

	port := server.Port()
	if port == 0 {
		t.Fatal("Server port should not be 0 after starting")
	}

	// Simulate callback from Spotify
	go func() {
		time.Sleep(50 * time.Millisecond)
		url := fmt.Sprintf("http://localhost:%d/callback?code=test_code&state=test_state", port)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("Failed to make callback request: %v", err)
			return
		}
		_ = resp.Body.Close()
	}()

	// Wait for result
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := server.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if result.Code != "test_code" {
		t.Errorf("Code = %q, want %q", result.Code, "test_code")
	}
	if result.State != "test_state" {
		t.Errorf("State = %q, want %q", result.State, "test_state")
	}
	if result.Error != "" {
		t.Errorf("Error = %q, want empty", result.Error)
	}
}

func TestCallbackServerError(t *testing.T) {
	server, err := NewCallbackServer(0)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}

	server.Start()
	defer func() { _ = server.Shutdown(context.Background()) }()

	port := server.Port()

	// Simulate error callback from Spotify
	go func() {
		time.Sleep(50 * time.Millisecond)
		url := fmt.Sprintf("http://localhost:%d/callback?error=access_denied&state=test_state", port)
		resp, err := http.Get(url)
		if err != nil {
			t.Errorf("Failed to make callback request: %v", err)
			return
		}
		_ = resp.Body.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := server.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if result.Error != "access_denied" {
		t.Errorf("Error = %q, want %q", result.Error, "access_denied")
	}
}

func TestCallbackServerTimeout(t *testing.T) {
	server, err := NewCallbackServer(0)
	if err != nil {
		t.Fatalf("NewCallbackServer() error = %v", err)
	}

	server.Start()
	defer func() { _ = server.Shutdown(context.Background()) }()

	// Create a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// No callback made, should timeout
	_, err = server.Wait(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Wait() error = %v, want %v", err, context.DeadlineExceeded)
	}
}
