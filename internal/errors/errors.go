package errors

import (
	"errors"
	"fmt"
	"strings"
)

// Error types for common failure scenarios.
var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrNoActiveDevice   = errors.New("no active device")
	ErrDeviceNotFound   = errors.New("device not found")
	ErrTrackNotFound    = errors.New("track not found")
	ErrPremiumRequired  = errors.New("spotify premium required")
	ErrRateLimited      = errors.New("rate limited")
	ErrNetworkError     = errors.New("network error")
	ErrTimeout          = errors.New("request timeout")
	ErrConfigNotFound   = errors.New("config file not found")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// RiffError wraps an error with a user-friendly suggestion.
type RiffError struct {
	Err        error
	Suggestion string
}

func (e *RiffError) Error() string {
	return e.Err.Error()
}

func (e *RiffError) Unwrap() error {
	return e.Err
}

// WithSuggestion wraps an error with a helpful suggestion.
func WithSuggestion(err error, suggestion string) error {
	return &RiffError{
		Err:        err,
		Suggestion: suggestion,
	}
}

// GetSuggestion returns a suggestion for the given error.
func GetSuggestion(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's already a RiffError with suggestion
	var riffErr *RiffError
	if errors.As(err, &riffErr) && riffErr.Suggestion != "" {
		return riffErr.Suggestion
	}

	errStr := strings.ToLower(err.Error())

	// Authentication errors
	if errors.Is(err, ErrNotAuthenticated) || strings.Contains(errStr, "not authenticated") ||
		strings.Contains(errStr, "invalid access token") || strings.Contains(errStr, "token expired") {
		return "Run 'riff auth login' to authenticate with Spotify"
	}

	// Device errors
	if errors.Is(err, ErrNoActiveDevice) || strings.Contains(errStr, "no active device") ||
		strings.Contains(errStr, "player command failed: no active device") {
		return "Open Spotify on a device and start playing, or use --device to specify one"
	}

	if errors.Is(err, ErrDeviceNotFound) || strings.Contains(errStr, "device not found") {
		return "Run 'riff devices' to see available devices"
	}

	// Premium errors
	if errors.Is(err, ErrPremiumRequired) || strings.Contains(errStr, "premium required") ||
		strings.Contains(errStr, "restricted device") {
		return "This feature requires Spotify Premium"
	}

	// Rate limiting
	if errors.Is(err, ErrRateLimited) || strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "429") {
		return "Too many requests. Wait a moment and try again"
	}

	// Network errors
	if errors.Is(err, ErrNetworkError) || errors.Is(err, ErrTimeout) ||
		strings.Contains(errStr, "network") || strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") {
		return "Check your internet connection and try again"
	}

	// Config errors
	if errors.Is(err, ErrConfigNotFound) || strings.Contains(errStr, "config") {
		return "Run 'riff auth login' to set up your configuration"
	}

	// Server errors
	if strings.Contains(errStr, "500") || strings.Contains(errStr, "server error") {
		return "Spotify is having issues. Try again in a moment"
	}

	return ""
}

// Format returns a formatted error message with suggestion if available.
func Format(err error) string {
	if err == nil {
		return ""
	}

	suggestion := GetSuggestion(err)
	if suggestion != "" {
		return fmt.Sprintf("Error: %s\n\nSuggestion: %s", err.Error(), suggestion)
	}

	return fmt.Sprintf("Error: %s", err.Error())
}

// PartialResult represents a result that may have partial failures.
type PartialResult[T any] struct {
	Data   T
	Errors []error
}

// HasErrors returns true if there were any errors.
func (p *PartialResult[T]) HasErrors() bool {
	return len(p.Errors) > 0
}

// AddError adds an error to the partial result.
func (p *PartialResult[T]) AddError(err error) {
	if err != nil {
		p.Errors = append(p.Errors, err)
	}
}

// ErrorSummary returns a summary of all errors.
func (p *PartialResult[T]) ErrorSummary() string {
	if len(p.Errors) == 0 {
		return ""
	}
	if len(p.Errors) == 1 {
		return p.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d errors occurred:\n", len(p.Errors)))
	for i, err := range p.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return sb.String()
}
