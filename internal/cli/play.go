package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tess/riff/internal/spotify/auth"
	"github.com/tess/riff/internal/spotify/client"
	"github.com/tess/riff/internal/spotify/player"
)

var (
	playTo       string
	playAlbum    bool
	playPlaylist bool
	playArtist   bool
	playURI      string
	playShuffle  bool
)

var playCmd = &cobra.Command{
	Use:   "play [query]",
	Short: "Start or resume playback",
	Long: `Start playback of a track, album, playlist, or artist.
Without arguments, resumes current playback.

Examples:
  riff play                    # Resume playback
  riff play "bohemian rhapsody" # Search and play a track
  riff play --album "abbey road" # Search and play an album
  riff play --uri spotify:track:xxx # Play specific URI
  riff play --to "Kitchen"     # Resume on specific device`,
	RunE: runPlay,
}

func init() {
	playCmd.Flags().StringVar(&playTo, "to", "", "Target device name or ID")
	playCmd.Flags().BoolVar(&playAlbum, "album", false, "Search for albums")
	playCmd.Flags().BoolVar(&playPlaylist, "playlist", false, "Search for playlists")
	playCmd.Flags().BoolVar(&playArtist, "artist", false, "Search for artists")
	playCmd.Flags().StringVar(&playURI, "uri", "", "Play specific Spotify URI")
	playCmd.Flags().BoolVar(&playShuffle, "shuffle", false, "Enable shuffle mode")
	rootCmd.AddCommand(playCmd)
}

func runPlay(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if cfg.Spotify.ClientID == "" {
		return fmt.Errorf("spotify not configured")
	}

	storage, err := auth.NewTokenStorage("")
	if err != nil {
		return fmt.Errorf("failed to initialize token storage: %w", err)
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

	if !spotifyClient.HasToken() {
		return fmt.Errorf("not authenticated. Run 'riff auth login' first")
	}

	p := player.New(spotifyClient)

	// Set target device if specified
	if playTo != "" {
		deviceID, err := resolveDevice(ctx, spotifyClient, playTo)
		if err != nil {
			return err
		}
		p.SetDevice(deviceID)
	}

	// Set shuffle if requested
	if playShuffle {
		if err := spotifyClient.SetShuffle(ctx, true, ""); err != nil {
			if Verbose() {
				fmt.Fprintf(os.Stderr, "Warning: could not enable shuffle: %v\n", err)
			}
		}
	}

	// Handle different play modes
	if playURI != "" {
		return playWithFallback(ctx, spotifyClient, p, func() error {
			return playByURIInternal(ctx, p, playURI)
		}, playURI, "uri")
	}

	query := strings.Join(args, " ")
	if query == "" {
		// Just resume playback
		return playWithFallback(ctx, spotifyClient, p, func() error {
			return p.Play(ctx)
		}, "", "resume")
	}

	// Search and play
	return searchAndPlay(ctx, spotifyClient, p, query)
}

// playWithFallback attempts to play and falls back to default device on 404
func playWithFallback(ctx context.Context, c *client.Client, p *player.Player, playFunc func() error, uri, playType string) error {
	err := playFunc()
	if err == nil {
		if playType == "resume" && !JSONOutput() {
			fmt.Println("▶ Resumed playback")
		} else if playType == "uri" {
			if JSONOutput() {
				json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
					"status": "playing",
					"uri":    uri,
				})
			} else {
				fmt.Printf("▶ Playing %s\n", uri)
			}
		}
		return nil
	}

	// Check if error is "no active device"
	if !client.IsNoActiveDeviceError(err) {
		if playType == "resume" {
			return fmt.Errorf("failed to resume playback: %w", err)
		}
		return fmt.Errorf("failed to play: %w", err)
	}

	// Try to use default device
	defaultDevice := cfg.Defaults.Device
	if defaultDevice == "" {
		return fmt.Errorf("no active device and no default device configured. Set defaults.device in config or use --to flag")
	}

	if Verbose() {
		fmt.Fprintf(os.Stderr, "No active device, transferring to default: %s\n", defaultDevice)
	}

	// Resolve and transfer to default device
	deviceID, err := resolveDevice(ctx, c, defaultDevice)
	if err != nil {
		return fmt.Errorf("failed to resolve default device '%s': %w", defaultDevice, err)
	}

	// Transfer playback to the device (this wakes it up)
	if err := c.TransferPlayback(ctx, deviceID, false); err != nil {
		return fmt.Errorf("failed to transfer to default device: %w", err)
	}

	// Set the device on the player and retry
	p.SetDevice(deviceID)

	// Retry the play command
	if err := playFunc(); err != nil {
		return fmt.Errorf("failed to play on default device: %w", err)
	}

	deviceName := defaultDevice
	if !JSONOutput() {
		if playType == "resume" {
			fmt.Printf("▶ Resumed playback on %s\n", deviceName)
		} else if playType == "uri" {
			fmt.Printf("▶ Playing %s on %s\n", uri, deviceName)
		}
	} else if playType == "uri" {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "playing",
			"uri":    uri,
			"device": deviceName,
		})
	}

	return nil
}

func playByURIInternal(ctx context.Context, p *player.Player, uri string) error {
	return p.PlayURI(ctx, uri)
}

func playByURI(ctx context.Context, p *player.Player, uri string) error {
	if err := p.PlayURI(ctx, uri); err != nil {
		return fmt.Errorf("failed to play URI: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "playing",
			"uri":    uri,
		})
	} else {
		fmt.Printf("▶ Playing %s\n", uri)
	}

	return nil
}

func searchAndPlay(ctx context.Context, c *client.Client, p *player.Player, query string) error {
	var searchTypes []client.SearchType

	if playAlbum {
		searchTypes = []client.SearchType{client.SearchTypeAlbum}
	} else if playPlaylist {
		searchTypes = []client.SearchType{client.SearchTypePlaylist}
	} else if playArtist {
		searchTypes = []client.SearchType{client.SearchTypeArtist}
	} else {
		searchTypes = []client.SearchType{client.SearchTypeTrack}
	}

	results, err := c.Search(ctx, client.SearchOptions{
		Query: query,
		Types: searchTypes,
		Limit: 1,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// Play the first result with fallback to default device
	if playAlbum && results.Albums != nil && len(results.Albums.Items) > 0 {
		album := results.Albums.Items[0]
		return playSearchResultWithFallback(ctx, c, p, func() error {
			return p.PlayContext(ctx, album.URI, 0)
		}, "album", album.Name, album.Artists[0].Name, album.URI)
	}

	if playPlaylist && results.Playlists != nil && len(results.Playlists.Items) > 0 {
		playlist := results.Playlists.Items[0]
		return playSearchResultWithFallback(ctx, c, p, func() error {
			return p.PlayContext(ctx, playlist.URI, 0)
		}, "playlist", playlist.Name, playlist.Owner.DisplayName, playlist.URI)
	}

	if playArtist && results.Artists != nil && len(results.Artists.Items) > 0 {
		artist := results.Artists.Items[0]
		return playSearchResultWithFallback(ctx, c, p, func() error {
			return p.PlayContext(ctx, artist.URI, 0)
		}, "artist", artist.Name, "", artist.URI)
	}

	if results.Tracks != nil && len(results.Tracks.Items) > 0 {
		track := results.Tracks.Items[0]
		return playSearchResultWithFallback(ctx, c, p, func() error {
			return p.PlayURI(ctx, track.URI)
		}, "track", track.Name, track.Artists[0].Name, track.URI)
	}

	return fmt.Errorf("no results found for '%s'", query)
}

// playSearchResultWithFallback plays a search result with fallback to default device on 404
func playSearchResultWithFallback(ctx context.Context, c *client.Client, p *player.Player, playFunc func() error, itemType, name, artist, uri string) error {
	err := playFunc()
	if err == nil {
		outputPlayResult(itemType, name, artist, uri)
		return nil
	}

	// Check if error is "no active device"
	if !client.IsNoActiveDeviceError(err) {
		return fmt.Errorf("failed to play %s: %w", itemType, err)
	}

	// Try to use default device
	defaultDevice := cfg.Defaults.Device
	if defaultDevice == "" {
		return fmt.Errorf("no active device and no default device configured. Set defaults.device in config or use --to flag")
	}

	if Verbose() {
		fmt.Fprintf(os.Stderr, "No active device, transferring to default: %s\n", defaultDevice)
	}

	// Resolve and transfer to default device
	deviceID, err := resolveDevice(ctx, c, defaultDevice)
	if err != nil {
		return fmt.Errorf("failed to resolve default device '%s': %w", defaultDevice, err)
	}

	// Transfer playback to the device
	if err := c.TransferPlayback(ctx, deviceID, false); err != nil {
		return fmt.Errorf("failed to transfer to default device: %w", err)
	}

	// Set the device on the player and retry
	p.SetDevice(deviceID)

	// Retry the play command
	if err := playFunc(); err != nil {
		return fmt.Errorf("failed to play %s on default device: %w", itemType, err)
	}

	outputPlayResultWithDevice(itemType, name, artist, uri, defaultDevice)
	return nil
}

func outputPlayResult(itemType, name, artist, uri string) {
	if JSONOutput() {
		output := map[string]interface{}{
			"status": "playing",
			"type":   itemType,
			"name":   name,
			"uri":    uri,
		}
		if artist != "" {
			output["artist"] = artist
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if artist != "" {
			fmt.Printf("▶ Playing %s: %s by %s\n", itemType, name, artist)
		} else {
			fmt.Printf("▶ Playing %s: %s\n", itemType, name)
		}
	}
}

func outputPlayResultWithDevice(itemType, name, artist, uri, device string) {
	if JSONOutput() {
		output := map[string]interface{}{
			"status": "playing",
			"type":   itemType,
			"name":   name,
			"uri":    uri,
			"device": device,
		}
		if artist != "" {
			output["artist"] = artist
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if artist != "" {
			fmt.Printf("▶ Playing %s: %s by %s on %s\n", itemType, name, artist, device)
		} else {
			fmt.Printf("▶ Playing %s: %s on %s\n", itemType, name, device)
		}
	}
}

func resolveDevice(ctx context.Context, c *client.Client, nameOrID string) (string, error) {
	devices, err := c.GetDevices(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get devices: %w", err)
	}

	// First try exact ID match
	for _, d := range devices {
		if d.ID == nameOrID {
			return d.ID, nil
		}
	}

	// Then try case-insensitive name match
	nameLower := strings.ToLower(nameOrID)
	for _, d := range devices {
		if strings.ToLower(d.Name) == nameLower {
			return d.ID, nil
		}
	}

	// Finally try partial name match
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), nameLower) {
			return d.ID, nil
		}
	}

	return "", fmt.Errorf("device '%s' not found", nameOrID)
}
