package auth

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildAuthURL(t *testing.T) {
	pkce := &PKCE{
		Verifier:  "test_verifier",
		Challenge: "test_challenge",
		State:     "test_state",
	}

	params := AuthURLParams{
		ClientID:    "test_client_id",
		RedirectURI: "http://localhost:8888/callback",
		Scopes:      []string{"user-read-private", "user-read-email"},
	}

	authURL := BuildAuthURL(params, pkce)

	// Parse the URL
	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("BuildAuthURL() produced invalid URL: %v", err)
	}

	// Verify base URL
	if u.Scheme != "https" || u.Host != "accounts.spotify.com" || u.Path != "/authorize" {
		t.Errorf("BuildAuthURL() base URL = %s://%s%s, want https://accounts.spotify.com/authorize",
			u.Scheme, u.Host, u.Path)
	}

	// Verify query parameters
	q := u.Query()

	tests := []struct {
		param string
		want  string
	}{
		{"client_id", "test_client_id"},
		{"response_type", "code"},
		{"redirect_uri", "http://localhost:8888/callback"},
		{"code_challenge_method", "S256"},
		{"code_challenge", "test_challenge"},
		{"state", "test_state"},
		{"scope", "user-read-private user-read-email"},
	}

	for _, tt := range tests {
		if got := q.Get(tt.param); got != tt.want {
			t.Errorf("BuildAuthURL() %s = %q, want %q", tt.param, got, tt.want)
		}
	}
}

func TestBuildAuthURLNoScopes(t *testing.T) {
	pkce := &PKCE{
		Verifier:  "test_verifier",
		Challenge: "test_challenge",
		State:     "test_state",
	}

	params := AuthURLParams{
		ClientID:    "test_client_id",
		RedirectURI: "http://localhost:8888/callback",
		Scopes:      nil,
	}

	authURL := BuildAuthURL(params, pkce)
	u, _ := url.Parse(authURL)
	q := u.Query()

	if scope := q.Get("scope"); scope != "" {
		t.Errorf("BuildAuthURL() with no scopes has scope = %q, want empty", scope)
	}
}

func TestNewConfig(t *testing.T) {
	config := NewConfig("my_client_id")

	if config.ClientID != "my_client_id" {
		t.Errorf("ClientID = %q, want %q", config.ClientID, "my_client_id")
	}

	if config.RedirectURI != DefaultRedirectURI {
		t.Errorf("RedirectURI = %q, want %q", config.RedirectURI, DefaultRedirectURI)
	}

	if len(config.Scopes) != len(DefaultScopes) {
		t.Errorf("Scopes length = %d, want %d", len(config.Scopes), len(DefaultScopes))
	}
}

func TestConfigBuildAuthURL(t *testing.T) {
	config := NewConfig("test_client")
	pkce, _ := NewPKCE()

	authURL := config.BuildAuthURL(pkce)

	// Verify it contains expected parameters
	if !strings.Contains(authURL, "client_id=test_client") {
		t.Error("BuildAuthURL() missing client_id")
	}
	if !strings.Contains(authURL, "code_challenge=") {
		t.Error("BuildAuthURL() missing code_challenge")
	}
	if !strings.Contains(authURL, "state=") {
		t.Error("BuildAuthURL() missing state")
	}
}
