package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/sonos"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
	"github.com/tessro/riff/internal/spotify/player"
)

// resolvedDevice represents a device that can be either Spotify or Sonos.
type resolvedDevice struct {
	Platform    core.Platform
	SpotifyID   string        // Populated if Platform == PlatformSpotify
	SonosDevice *sonos.Device // Populated if Platform == PlatformSonos
	Name        string
}

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

	// Resolve target device if specified
	var targetDevice *resolvedDevice
	if playTo != "" {
		targetDevice, err = resolveDevice(ctx, spotifyClient, playTo)
		if err != nil {
			return err
		}
	}

	// Handle Sonos device playback
	if targetDevice != nil && targetDevice.Platform == core.PlatformSonos {
		return runPlaySonos(ctx, spotifyClient, targetDevice, args)
	}

	// Spotify playback path
	p := player.New(spotifyClient)

	// Set target device if specified
	if targetDevice != nil {
		p.SetDevice(targetDevice.SpotifyID)
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

// runPlaySonos handles playback to a Sonos device directly.
func runPlaySonos(ctx context.Context, spotifyClient *client.Client, device *resolvedDevice, args []string) error {
	sonosClient := sonos.NewClient()
	sonosPlayer := sonos.NewPlayer(sonosClient, device.SonosDevice)

	// Handle URI playback
	if playURI != "" {
		if err := sonosPlayer.PlayURI(ctx, playURI); err != nil {
			return fmt.Errorf("failed to play on Sonos: %w", err)
		}
		if JSONOutput() {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"status": "playing",
				"uri":    playURI,
				"device": device.Name,
			})
		} else {
			fmt.Printf("▶ Playing %s on %s (Sonos)\n", playURI, device.Name)
		}
		return nil
	}

	query := strings.Join(args, " ")
	if query == "" {
		// Just resume playback
		if err := sonosPlayer.Play(ctx); err != nil {
			return fmt.Errorf("failed to resume on Sonos: %w", err)
		}
		if !JSONOutput() {
			fmt.Printf("▶ Resumed playback on %s (Sonos)\n", device.Name)
		}
		return nil
	}

	// Search using Spotify, then play on Sonos
	return searchAndPlaySonos(ctx, spotifyClient, sonosPlayer, device.Name, query)
}

// searchAndPlaySonos searches Spotify and plays the result on Sonos.
func searchAndPlaySonos(ctx context.Context, c *client.Client, sonosPlayer *sonos.Player, deviceName, query string) error {
	var searchType client.SearchType
	switch {
	case playAlbum:
		searchType = client.SearchTypeAlbum
	case playPlaylist:
		searchType = client.SearchTypePlaylist
	case playArtist:
		searchType = client.SearchTypeArtist
	default:
		searchType = client.SearchTypeTrack
	}

	results, err := c.Search(ctx, client.SearchOptions{
		Query: query,
		Types: []client.SearchType{searchType},
		Limit: 1,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	var uri, name, artist string
	switch searchType {
	case client.SearchTypeTrack:
		if len(results.Tracks.Items) == 0 {
			return fmt.Errorf("no tracks found for '%s'", query)
		}
		track := results.Tracks.Items[0]
		uri = track.URI
		name = track.Name
		if len(track.Artists) > 0 {
			artist = track.Artists[0].Name
		}
	case client.SearchTypeAlbum:
		if len(results.Albums.Items) == 0 {
			return fmt.Errorf("no albums found for '%s'", query)
		}
		album := results.Albums.Items[0]
		uri = album.URI
		name = album.Name
		if len(album.Artists) > 0 {
			artist = album.Artists[0].Name
		}
	case client.SearchTypePlaylist:
		if len(results.Playlists.Items) == 0 {
			return fmt.Errorf("no playlists found for '%s'", query)
		}
		playlist := results.Playlists.Items[0]
		uri = playlist.URI
		name = playlist.Name
	case client.SearchTypeArtist:
		if len(results.Artists.Items) == 0 {
			return fmt.Errorf("no artists found for '%s'", query)
		}
		a := results.Artists.Items[0]
		uri = a.URI
		name = a.Name
	}

	if err := sonosPlayer.PlayURI(ctx, uri); err != nil {
		return fmt.Errorf("failed to play on Sonos: %w", err)
	}

	if JSONOutput() {
		output := map[string]interface{}{
			"status": "playing",
			"type":   searchType,
			"name":   name,
			"uri":    uri,
			"device": deviceName,
		}
		if artist != "" {
			output["artist"] = artist
		}
		json.NewEncoder(os.Stdout).Encode(output)
	} else {
		if artist != "" {
			fmt.Printf("▶ Playing %s: %s by %s on %s (Sonos)\n", searchType, name, artist, deviceName)
		} else {
			fmt.Printf("▶ Playing %s: %s on %s (Sonos)\n", searchType, name, deviceName)
		}
	}

	return nil
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

	// Handle 403 for resume - Spotify returns 403 when already playing
	if playType == "resume" && client.IsAlreadyPlayingError(err) {
		if !JSONOutput() {
			fmt.Println("▶ Already playing")
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

	// Try to use default device or show picker
	defaultDeviceName := cfg.Defaults.Device
	var deviceID string
	var deviceName string
	if defaultDeviceName == "" {
		// No default configured, show interactive picker
		deviceID, deviceName, err = selectDevice(ctx, c)
		if err != nil {
			return err
		}
	} else {
		if Verbose() {
			fmt.Fprintf(os.Stderr, "No active device, transferring to default: %s\n", defaultDeviceName)
		}

		// Resolve default device
		resolved, err := resolveDevice(ctx, c, defaultDeviceName)
		if err != nil {
			return fmt.Errorf("failed to resolve default device '%s': %w", defaultDeviceName, err)
		}

		// For fallback, we only support Spotify devices (Sonos fallback would need different handling)
		if resolved.Platform != core.PlatformSpotify {
			return fmt.Errorf("default device '%s' is a Sonos device; use --to flag explicitly", defaultDeviceName)
		}
		deviceID = resolved.SpotifyID
		deviceName = resolved.Name
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

	// Try to use default device or show picker
	defaultDeviceName := cfg.Defaults.Device
	var deviceID string
	var deviceName string
	if defaultDeviceName == "" {
		// No default configured, show interactive picker
		deviceID, deviceName, err = selectDevice(ctx, c)
		if err != nil {
			return err
		}
	} else {
		if Verbose() {
			fmt.Fprintf(os.Stderr, "No active device, transferring to default: %s\n", defaultDeviceName)
		}

		// Resolve default device
		resolved, err := resolveDevice(ctx, c, defaultDeviceName)
		if err != nil {
			return fmt.Errorf("failed to resolve default device '%s': %w", defaultDeviceName, err)
		}

		// For fallback, we only support Spotify devices
		if resolved.Platform != core.PlatformSpotify {
			return fmt.Errorf("default device '%s' is a Sonos device; use --to flag explicitly", defaultDeviceName)
		}
		deviceID = resolved.SpotifyID
		deviceName = resolved.Name
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

	outputPlayResultWithDevice(itemType, name, artist, uri, deviceName)
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

func resolveDevice(ctx context.Context, c *client.Client, nameOrID string) (*resolvedDevice, error) {
	devices, err := c.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	// First try exact ID match in Spotify
	for _, d := range devices {
		if d.ID == nameOrID {
			return &resolvedDevice{
				Platform:  core.PlatformSpotify,
				SpotifyID: d.ID,
				Name:      d.Name,
			}, nil
		}
	}

	// Then try case-insensitive name match in Spotify
	nameLower := strings.ToLower(nameOrID)
	for _, d := range devices {
		if strings.ToLower(d.Name) == nameLower {
			return &resolvedDevice{
				Platform:  core.PlatformSpotify,
				SpotifyID: d.ID,
				Name:      d.Name,
			}, nil
		}
	}

	// Try partial name match in Spotify
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), nameLower) {
			return &resolvedDevice{
				Platform:  core.PlatformSpotify,
				SpotifyID: d.ID,
				Name:      d.Name,
			}, nil
		}
	}

	// Not found in Spotify - try Sonos
	if sonosDevice := findSonosDevice(ctx, nameOrID); sonosDevice != nil {
		return &resolvedDevice{
			Platform:    core.PlatformSonos,
			SonosDevice: sonosDevice,
			Name:        sonosDevice.Name,
		}, nil
	}

	return nil, fmt.Errorf("device '%s' not found", nameOrID)
}

// findSonosDevice finds a device on the local Sonos network.
func findSonosDevice(ctx context.Context, nameOrID string) *sonos.Device {
	sonosClient := sonos.NewClient()
	sonosDevices, err := sonosClient.Discover(ctx)
	if err != nil || len(sonosDevices) == 0 {
		return nil
	}

	// Get zone groups for device names
	groups, err := sonosClient.ListGroups(ctx, sonosDevices[0])
	if err != nil {
		// Fall back to basic device info
		nameLower := strings.ToLower(nameOrID)
		for _, d := range sonosDevices {
			if d.UUID == nameOrID ||
				strings.ToLower(d.Name) == nameLower ||
				strings.Contains(strings.ToLower(d.Name), nameLower) {
				return d
			}
		}
		return nil
	}

	// Check zone group members
	nameLower := strings.ToLower(nameOrID)
	for _, g := range groups {
		for _, m := range g.Members {
			if m.UUID == nameOrID ||
				strings.ToLower(m.Name) == nameLower ||
				strings.Contains(strings.ToLower(m.Name), nameLower) {
				return m
			}
		}
	}

	return nil
}

// selectDevice shows an interactive picker for device selection
func selectDevice(ctx context.Context, c *client.Client) (deviceID, deviceName string, err error) {
	devices, err := c.GetDevices(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get devices: %w", err)
	}

	if len(devices) == 0 {
		return "", "", fmt.Errorf("no devices found. Make sure Spotify is open on at least one device")
	}

	// Build options for picker
	var options []huh.Option[string]
	for _, d := range devices {
		label := d.Name
		if d.Type != "" {
			label = fmt.Sprintf("%s (%s)", d.Name, d.Type)
		}
		options = append(options, huh.NewOption(label, d.ID))
	}

	var selectedID string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("No active device found").
				Description("Select a device to play on").
				Options(options...).
				Value(&selectedID),
		),
	)

	if err := form.Run(); err != nil {
		return "", "", fmt.Errorf("selection cancelled")
	}

	// Find the device name
	for _, d := range devices {
		if d.ID == selectedID {
			return selectedID, d.Name, nil
		}
	}

	return selectedID, "", nil
}
