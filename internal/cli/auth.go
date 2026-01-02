package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/browser"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Spotify authentication",
	Long:  `Commands for managing Spotify OAuth authentication.`,
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Spotify",
	Long:  `Opens a browser to authenticate with Spotify using OAuth PKCE flow.`,
	RunE:  runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored Spotify credentials",
	Long:  `Removes the stored Spotify OAuth tokens from the local machine.`,
	RunE:  runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Shows the current Spotify authentication status.`,
	RunE:  runAuthStatus,
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	if cfg.Spotify.ClientID == "" {
		return fmt.Errorf("spotify.client_id not configured. Set it in ~/.riffrc or via RIFF_SPOTIFY_CLIENT_ID")
	}

	// Generate PKCE parameters
	pkce, err := auth.NewPKCE()
	if err != nil {
		return fmt.Errorf("failed to generate PKCE: %w", err)
	}

	// Start callback server
	callbackServer, err := auth.NewCallbackServer(8888)
	if err != nil {
		return fmt.Errorf("failed to start callback server: %w", err)
	}
	callbackServer.Start()
	defer func() { _ = callbackServer.Shutdown(context.Background()) }()

	// Build auth URL
	config := auth.NewConfig(cfg.Spotify.ClientID)
	if cfg.Spotify.RedirectURI != "" {
		config.RedirectURI = cfg.Spotify.RedirectURI
	}
	authURL := config.BuildAuthURL(pkce)

	// Open browser
	fmt.Println("Opening browser for Spotify authentication...")
	if err := browser.Open(authURL); err != nil {
		fmt.Printf("Could not open browser automatically.\n")
		fmt.Printf("Please open this URL in your browser:\n\n%s\n\n", authURL)
	}

	// Wait for callback
	fmt.Println("Waiting for authentication...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := callbackServer.Wait(ctx)
	if err != nil {
		return fmt.Errorf("authentication timed out: %w", err)
	}

	if result.Error != "" {
		return fmt.Errorf("authentication failed: %s", result.Error)
	}

	// Verify state
	if result.State != pkce.State {
		return fmt.Errorf("state mismatch: possible CSRF attack")
	}

	// Exchange code for tokens
	fmt.Println("Exchanging code for tokens...")
	token, err := auth.ExchangeCode(ctx, cfg.Spotify.ClientID, result.Code, config.RedirectURI, pkce.Verifier)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Store token
	storage, err := auth.NewTokenStorage("")
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	if err := storage.Save(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	// Get user info to confirm success
	spotifyClient := client.New(cfg.Spotify.ClientID, storage)
	if Verbose() {
		spotifyClient.SetVerbose(true, func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		})
	}
	if err := spotifyClient.LoadToken(); err != nil {
		return fmt.Errorf("failed to load token: %w", err)
	}

	user, err := spotifyClient.GetCurrentUser(ctx)
	if err != nil {
		fmt.Println("Authentication successful! Token stored.")
		return nil
	}

	if JSONOutput() {
		output := map[string]interface{}{
			"status":       "authenticated",
			"user_id":      user.ID,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"product":      user.Product,
		}
		_ = json.NewEncoder(os.Stdout).Encode(output)
	} else {
		fmt.Printf("Successfully authenticated as %s (%s)\n", user.DisplayName, user.Email)
	}

	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	storage, err := auth.NewTokenStorage("")
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	if !storage.Exists() {
		if JSONOutput() {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "not_authenticated"})
		} else {
			fmt.Println("Not authenticated with Spotify.")
		}
		return nil
	}

	if err := storage.Delete(); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	if JSONOutput() {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "logged_out"})
	} else {
		fmt.Println("Logged out of Spotify.")
	}

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	storage, err := auth.NewTokenStorage("")
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
	}

	token, err := storage.Load()
	if err != nil {
		return fmt.Errorf("failed to load token: %w", err)
	}

	if token == nil {
		if JSONOutput() {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"authenticated": false,
			})
		} else {
			fmt.Println("Not authenticated with Spotify.")
			fmt.Println("Run 'riff auth login' to authenticate.")
		}
		return nil
	}

	// Try to get user info
	if cfg.Spotify.ClientID == "" {
		if JSONOutput() {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"authenticated": true,
				"expired":       token.IsExpired(),
				"expires_at":    token.ExpiresAt,
			})
		} else {
			if token.IsExpired() {
				fmt.Println("Authenticated but token expired.")
			} else {
				fmt.Println("Authenticated with Spotify.")
			}
		}
		return nil
	}

	spotifyClient := client.New(cfg.Spotify.ClientID, storage)
	if Verbose() {
		spotifyClient.SetVerbose(true, func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		})
	}
	if err := spotifyClient.LoadToken(); err != nil {
		return fmt.Errorf("failed to load token: %w", err)
	}

	ctx := context.Background()
	user, err := spotifyClient.GetCurrentUser(ctx)
	if err != nil {
		if JSONOutput() {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"authenticated": true,
				"expired":       true,
				"error":         err.Error(),
			})
		} else {
			fmt.Printf("Token may be expired or invalid: %v\n", err)
			fmt.Println("Run 'riff auth login' to re-authenticate.")
		}
		return nil
	}

	if JSONOutput() {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"authenticated": true,
			"expired":       false,
			"user_id":       user.ID,
			"display_name":  user.DisplayName,
			"email":         user.Email,
			"product":       user.Product,
			"expires_at":    token.ExpiresAt,
		})
	} else {
		fmt.Printf("Authenticated as: %s (%s)\n", user.DisplayName, user.Email)
		fmt.Printf("Account type: %s\n", user.Product)
		fmt.Printf("Token expires: %s\n", token.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}
