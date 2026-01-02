package auth

import (
	"net/url"
)

const (
	// SpotifyAuthURL is the Spotify authorization endpoint.
	SpotifyAuthURL = "https://accounts.spotify.com/authorize"

	// SpotifyTokenURL is the Spotify token endpoint.
	SpotifyTokenURL = "https://accounts.spotify.com/api/token"

	// DefaultRedirectURI is the default callback URI for the local server.
	DefaultRedirectURI = "http://127.0.0.1:8888/callback"
)

// DefaultScopes are the Spotify scopes required for riff functionality.
var DefaultScopes = []string{
	"user-read-playback-state",
	"user-modify-playback-state",
	"user-read-currently-playing",
	"user-read-private",
	"user-read-email",
	"streaming",
}

// Config holds the OAuth configuration.
type Config struct {
	ClientID    string
	RedirectURI string
	Scopes      []string
}

// AuthURLParams contains the parameters for building an authorization URL.
type AuthURLParams struct {
	ClientID     string
	RedirectURI  string
	Scopes       []string
	CodeVerifier string
	State        string
}

// BuildAuthURL constructs the Spotify authorization URL with PKCE parameters.
func BuildAuthURL(params AuthURLParams, pkce *PKCE) string {
	u, _ := url.Parse(SpotifyAuthURL)

	q := u.Query()
	q.Set("client_id", params.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", params.RedirectURI)
	q.Set("code_challenge_method", "S256")
	q.Set("code_challenge", pkce.Challenge)
	q.Set("state", pkce.State)

	if len(params.Scopes) > 0 {
		scope := ""
		for i, s := range params.Scopes {
			if i > 0 {
				scope += " "
			}
			scope += s
		}
		q.Set("scope", scope)
	}

	u.RawQuery = q.Encode()
	return u.String()
}

// NewConfig creates a new OAuth configuration with defaults.
func NewConfig(clientID string) *Config {
	return &Config{
		ClientID:    clientID,
		RedirectURI: DefaultRedirectURI,
		Scopes:      DefaultScopes,
	}
}

// BuildAuthURLFromConfig is a convenience method to build an auth URL from config.
func (c *Config) BuildAuthURL(pkce *PKCE) string {
	return BuildAuthURL(AuthURLParams{
		ClientID:    c.ClientID,
		RedirectURI: c.RedirectURI,
		Scopes:      c.Scopes,
	}, pkce)
}
