package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tessro/riff/internal/spotify/auth"
)

const (
	// BaseURL is the Spotify Web API base URL.
	BaseURL = "https://api.spotify.com/v1"

	// Retry configuration for transient errors
	maxRetries    = 3
	baseRetryWait = 500 * time.Millisecond
)

// Client is a Spotify API client.
type Client struct {
	httpClient *http.Client
	clientID   string
	storage    *auth.TokenStorage
	token      *auth.Token
	mu         sync.RWMutex
	verbose    bool
	logFunc    func(format string, args ...interface{})
}

// New creates a new Spotify client.
func New(clientID string, storage *auth.TokenStorage) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		clientID:   clientID,
		storage:    storage,
	}
}

// SetVerbose enables verbose logging.
func (c *Client) SetVerbose(verbose bool, logFunc func(format string, args ...interface{})) {
	c.verbose = verbose
	c.logFunc = logFunc
}

func (c *Client) log(format string, args ...interface{}) {
	if c.verbose && c.logFunc != nil {
		c.logFunc(format, args...)
	}
}

// LoadToken loads the token from storage.
func (c *Client) LoadToken() error {
	token, err := c.storage.Load()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
	return nil
}

// SetToken sets the current token.
func (c *Client) SetToken(token *auth.Token) error {
	c.mu.Lock()
	c.token = token
	c.mu.Unlock()
	return c.storage.Save(token)
}

// IsAuthenticated returns true if there's a valid (non-expired) token.
func (c *Client) IsAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token != nil && !c.token.IsExpired()
}

// HasToken returns true if there's any token (even if expired).
func (c *Client) HasToken() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token != nil
}

// RefreshToken refreshes the access token if needed.
func (c *Client) RefreshToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token == nil {
		return fmt.Errorf("no token to refresh")
	}

	if !c.token.IsExpired() {
		return nil // Token is still valid
	}

	newToken, err := auth.RefreshAccessToken(ctx, c.clientID, c.token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Preserve refresh token if not returned
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = c.token.RefreshToken
	}

	c.token = newToken
	return c.storage.Save(newToken)
}

// getToken returns the current access token, refreshing if needed.
func (c *Client) getToken(ctx context.Context) (string, error) {
	if err := c.RefreshToken(ctx); err != nil {
		return "", err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.token == nil {
		return "", fmt.Errorf("not authenticated")
	}

	return c.token.AccessToken, nil
}

// Get performs a GET request to the Spotify API.
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.request(ctx, "GET", path, nil, result)
}

// Post performs a POST request to the Spotify API.
func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.request(ctx, "POST", path, body, result)
}

// Put performs a PUT request to the Spotify API.
func (c *Client) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.request(ctx, "PUT", path, body, result)
}

// Delete performs a DELETE request to the Spotify API.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.request(ctx, "DELETE", path, nil, nil)
}

func (c *Client) request(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	fullURL := BaseURL + path

	if jsonBody != nil {
		c.log("[spotify] %s %s\n  body: %s", method, fullURL, string(jsonBody))
	} else {
		c.log("[spotify] %s %s", method, fullURL)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait before retry (skip on first attempt)
		if attempt > 0 {
			wait := baseRetryWait * time.Duration(1<<(attempt-1)) // exponential backoff
			c.log("[spotify] retry %d/%d after %v (last error: %v)", attempt, maxRetries, wait, lastErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}

		var bodyReader io.Reader
		if jsonBody != nil {
			bodyReader = strings.NewReader(string(jsonBody))
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			c.log("[spotify] network error: %v", err)
			continue // Retry on network error
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			c.log("[spotify] read error: %v", err)
			continue
		}

		c.log("[spotify] response: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		if resp.StatusCode >= 400 {
			c.log("[spotify] response body: %s", string(respBody))
		}

		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		// Retry on 5xx server errors
		if resp.StatusCode >= 500 {
			var apiErr APIError
			if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.ErrorInfo.Message != "" {
				lastErr = &apiErr
			} else {
				lastErr = fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
			}
			c.log("[spotify] server error, will retry: %v", lastErr)
			continue // Retry
		}

		// Don't retry 4xx errors
		if resp.StatusCode >= 400 {
			var apiErr APIError
			if err := json.Unmarshal(respBody, &apiErr); err == nil && apiErr.ErrorInfo.Message != "" {
				return &apiErr
			}
			return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
		}

		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// APIError represents a Spotify API error response.
type APIError struct {
	ErrorInfo struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Spotify API error %d: %s", e.ErrorInfo.Status, e.ErrorInfo.Message)
}

// IsNoActiveDevice returns true if the error indicates no active device.
func (e *APIError) IsNoActiveDevice() bool {
	return e.ErrorInfo.Status == 404
}

// IsNoActiveDeviceError checks if an error is a "no active device" error.
func IsNoActiveDeviceError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.IsNoActiveDevice()
	}
	return false
}

// IsAlreadyPlayingError checks if an error is a 403 "restriction violated" error,
// which occurs when trying to resume playback that is already active.
func IsAlreadyPlayingError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.ErrorInfo.Status == 403
	}
	return false
}

// BuildURL builds a URL with query parameters.
func BuildURL(path string, params map[string]string) string {
	if len(params) == 0 {
		return path
	}

	u, _ := url.Parse(path)
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
