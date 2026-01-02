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
		return playByURI(ctx, p, playURI)
	}

	query := strings.Join(args, " ")
	if query == "" {
		// Just resume playback
		if err := p.Play(ctx); err != nil {
			return fmt.Errorf("failed to resume playback: %w", err)
		}
		if !JSONOutput() {
			fmt.Println("▶ Resumed playback")
		}
		return nil
	}

	// Search and play
	return searchAndPlay(ctx, spotifyClient, p, query)
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

	// Play the first result
	if playAlbum && results.Albums != nil && len(results.Albums.Items) > 0 {
		album := results.Albums.Items[0]
		if err := p.PlayContext(ctx, album.URI, 0); err != nil {
			return fmt.Errorf("failed to play album: %w", err)
		}
		outputPlayResult("album", album.Name, album.Artists[0].Name, album.URI)
		return nil
	}

	if playPlaylist && results.Playlists != nil && len(results.Playlists.Items) > 0 {
		playlist := results.Playlists.Items[0]
		if err := p.PlayContext(ctx, playlist.URI, 0); err != nil {
			return fmt.Errorf("failed to play playlist: %w", err)
		}
		outputPlayResult("playlist", playlist.Name, playlist.Owner.DisplayName, playlist.URI)
		return nil
	}

	if playArtist && results.Artists != nil && len(results.Artists.Items) > 0 {
		artist := results.Artists.Items[0]
		if err := p.PlayContext(ctx, artist.URI, 0); err != nil {
			return fmt.Errorf("failed to play artist: %w", err)
		}
		outputPlayResult("artist", artist.Name, "", artist.URI)
		return nil
	}

	if results.Tracks != nil && len(results.Tracks.Items) > 0 {
		track := results.Tracks.Items[0]
		if err := p.PlayURI(ctx, track.URI); err != nil {
			return fmt.Errorf("failed to play track: %w", err)
		}
		outputPlayResult("track", track.Name, track.Artists[0].Name, track.URI)
		return nil
	}

	return fmt.Errorf("no results found for '%s'", query)
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
