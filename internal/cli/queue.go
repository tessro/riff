package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
	"github.com/tessro/riff/internal/spotify/player"
)

var queueLimit int

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage playback queue",
	Long:  `View and manage the playback queue.`,
	RunE:  runQueueList,
}

var queueAddCmd = &cobra.Command{
	Use:   "add <query>",
	Short: "Add a track to the queue",
	Long: `Search for a track and add it to the queue.

Examples:
  riff queue add "bohemian rhapsody"
  riff queue add --uri spotify:track:xxx`,
	Args: cobra.MinimumNArgs(1),
	RunE: runQueueAdd,
}

var queueRemoveCmd = &cobra.Command{
	Use:   "remove <index>",
	Short: "Remove a track from the queue",
	Long: `Remove a track at the specified position from the queue.
Note: Spotify API does not support queue removal. This is a placeholder.`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueRemove,
}

var queueClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the queue",
	Long: `Clear all tracks from the queue.
Note: Spotify API does not support queue clearing. This is a placeholder.`,
	RunE: runQueueClear,
}

var queueMoveCmd = &cobra.Command{
	Use:   "move <from> <to>",
	Short: "Move a track in the queue",
	Long: `Move a track from one position to another.
Note: Spotify API does not support queue reordering. This is a placeholder.`,
	Args: cobra.ExactArgs(2),
	RunE: runQueueMove,
}

var queueAddURI string

func init() {
	queueCmd.Flags().IntVarP(&queueLimit, "limit", "l", 20, "Maximum number of tracks to show")
	queueAddCmd.Flags().StringVar(&queueAddURI, "uri", "", "Add specific Spotify URI to queue")

	queueCmd.AddCommand(queueAddCmd)
	queueCmd.AddCommand(queueRemoveCmd)
	queueCmd.AddCommand(queueClearCmd)
	queueCmd.AddCommand(queueMoveCmd)
	rootCmd.AddCommand(queueCmd)
}

func runQueueList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	spotifyClient, err := getSpotifyClient()
	if err != nil {
		return err
	}

	p := player.New(spotifyClient)
	queue, err := p.GetQueue(ctx)
	if err != nil {
		return fmt.Errorf("failed to get queue: %w", err)
	}

	if queue.IsEmpty() {
		if JSONOutput() {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"queue":   []interface{}{},
				"message": "Queue is empty",
			})
		} else {
			fmt.Println("Queue is empty")
		}
		return nil
	}

	// Apply limit
	tracks := queue.Tracks
	if queueLimit > 0 && len(tracks) > queueLimit {
		tracks = tracks[:queueLimit]
	}

	if JSONOutput() {
		output := make([]map[string]interface{}, len(tracks))
		for i, t := range tracks {
			output[i] = map[string]interface{}{
				"position": i,
				"title":    t.Title,
				"artist":   t.Artist,
				"album":    t.Album,
				"duration": t.Duration.String(),
				"uri":      t.URI,
			}
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"queue": output,
			"total": len(queue.Tracks),
		})
	}

	// Table output
	fmt.Println("Queue:")
	for i, t := range tracks {
		prefix := "  "
		if i == 0 {
			prefix = "▶ "
		}
		fmt.Printf("%s%d. %s — %s (%s)\n", prefix, i+1, t.Title, t.Artist, formatDuration(t.Duration))
	}

	if len(queue.Tracks) > queueLimit {
		fmt.Printf("\n... and %d more tracks\n", len(queue.Tracks)-queueLimit)
	}

	return nil
}

func runQueueAdd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	spotifyClient, err := getSpotifyClient()
	if err != nil {
		return err
	}

	p := player.New(spotifyClient)

	var uri string
	var trackName string

	if queueAddURI != "" {
		uri = queueAddURI
		trackName = uri
	} else {
		// Search for the track
		query := args[0]
		results, err := spotifyClient.Search(ctx, client.SearchOptions{
			Query: query,
			Types: []client.SearchType{client.SearchTypeTrack},
			Limit: 1,
		})
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if results.Tracks == nil || len(results.Tracks.Items) == 0 {
			return fmt.Errorf("no tracks found for '%s'", query)
		}

		track := results.Tracks.Items[0]
		uri = track.URI
		trackName = fmt.Sprintf("%s by %s", track.Name, track.Artists[0].Name)
	}

	if err := p.AddToQueue(ctx, uri); err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status": "added",
			"uri":    uri,
			"name":   trackName,
		})
	} else {
		fmt.Printf("Added to queue: %s\n", trackName)
	}

	return nil
}

func runQueueRemove(cmd *cobra.Command, args []string) error {
	index, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid index: %s", args[0])
	}

	// Spotify API doesn't support queue removal
	return fmt.Errorf("queue removal is not supported by Spotify API (requested index: %d)", index)
}

func runQueueClear(cmd *cobra.Command, args []string) error {
	// Spotify API doesn't support queue clearing
	return fmt.Errorf("queue clearing is not supported by Spotify API")
}

func runQueueMove(cmd *cobra.Command, args []string) error {
	from, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid from index: %s", args[0])
	}
	to, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid to index: %s", args[1])
	}

	// Spotify API doesn't support queue reordering
	return fmt.Errorf("queue reordering is not supported by Spotify API (requested move: %d -> %d)", from, to)
}

func getSpotifyClient() (*client.Client, error) {
	if cfg.Spotify.ClientID == "" {
		return nil, fmt.Errorf("spotify not configured")
	}

	storage, err := auth.NewTokenStorage("")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize token storage: %w", err)
	}

	spotifyClient := client.New(cfg.Spotify.ClientID, storage)
	if Verbose() {
		spotifyClient.SetVerbose(true, func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		})
	}
	if err := spotifyClient.LoadToken(); err != nil {
		return nil, fmt.Errorf("failed to load token: %w", err)
	}

	if !spotifyClient.HasToken() {
		return nil, fmt.Errorf("not authenticated. Run 'riff auth login' first")
	}

	return spotifyClient, nil
}
