package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
	"github.com/tessro/riff/internal/spotify/player"
)

var (
	statusSpotify bool
	statusSonos   bool
	statusDevice  string
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current playback status",
	Long:  `Shows the current playback status across Spotify and Sonos devices.`,
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusSpotify, "spotify", false, "Show only Spotify status")
	statusCmd.Flags().BoolVar(&statusSonos, "sonos", false, "Show only Sonos status")
	statusCmd.Flags().StringVarP(&statusDevice, "device", "d", "", "Show status for specific device")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine which platforms to query
	showSpotify := !statusSonos || statusSpotify
	showSonos := !statusSpotify || statusSonos

	var states []*statusResult

	if showSpotify {
		state, err := getSpotifyStatus(ctx)
		if err != nil {
			if Verbose() {
				fmt.Fprintf(os.Stderr, "Spotify error: %v\n", err)
			}
		} else if state != nil {
			states = append(states, state)
		}
	}

	if showSonos {
		// TODO: Implement Sonos status when Phase 3 is complete
		if Verbose() {
			fmt.Fprintln(os.Stderr, "Sonos status not yet implemented")
		}
	}

	if len(states) == 0 {
		if JSONOutput() {
			json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"playing": false,
				"message": "No active playback",
			})
		} else {
			fmt.Println("No active playback")
		}
		return nil
	}

	// Filter by device if specified
	if statusDevice != "" {
		filtered := make([]*statusResult, 0)
		for _, s := range states {
			if s.Device != nil && (strings.EqualFold(s.Device.Name, statusDevice) || s.Device.ID == statusDevice) {
				filtered = append(filtered, s)
			}
		}
		states = filtered
	}

	if JSONOutput() {
		return outputStatusJSON(states)
	}
	return outputStatusTable(states)
}

type statusResult struct {
	Platform string
	State    *core.PlaybackState
	Device   *core.Device
}

func getSpotifyStatus(ctx context.Context) (*statusResult, error) {
	if cfg.Spotify.ClientID == "" {
		return nil, fmt.Errorf("spotify not configured")
	}

	storage, err := auth.NewTokenStorage("")
	if err != nil {
		return nil, err
	}

	spotifyClient := client.New(cfg.Spotify.ClientID, storage)
	if Verbose() {
		spotifyClient.SetVerbose(true, func(format string, args ...interface{}) {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		})
	}
	if err := spotifyClient.LoadToken(); err != nil {
		return nil, err
	}

	if !spotifyClient.HasToken() {
		return nil, fmt.Errorf("not authenticated")
	}

	p := player.New(spotifyClient)
	state, err := p.GetState(ctx)
	if err != nil {
		return nil, err
	}

	return &statusResult{
		Platform: "spotify",
		State:    state,
		Device:   state.Device,
	}, nil
}

func outputStatusJSON(states []*statusResult) error {
	output := make([]map[string]interface{}, 0, len(states))

	for _, s := range states {
		item := map[string]interface{}{
			"platform":   s.Platform,
			"is_playing": s.State.IsPlaying,
			"volume":     s.State.Volume,
		}

		if s.State.Track != nil {
			item["track"] = map[string]interface{}{
				"title":    s.State.Track.Title,
				"artist":   s.State.Track.Artist,
				"album":    s.State.Track.Album,
				"duration": s.State.Track.Duration.String(),
				"uri":      s.State.Track.URI,
			}
			item["progress"] = s.State.Progress.String()
			item["progress_percent"] = s.State.ProgressPercent()
		}

		if s.Device != nil {
			item["device"] = map[string]interface{}{
				"id":        s.Device.ID,
				"name":      s.Device.Name,
				"type":      s.Device.Type,
				"is_active": s.Device.IsActive,
			}
		}

		output = append(output, item)
	}

	return json.NewEncoder(os.Stdout).Encode(output)
}

func outputStatusTable(states []*statusResult) error {
	for i, s := range states {
		if i > 0 {
			fmt.Println()
		}

		// Platform header
		fmt.Printf("[%s]\n", strings.ToUpper(s.Platform))

		if s.State.Track == nil {
			fmt.Println("  No track playing")
			continue
		}

		// Track info
		playIcon := "▶"
		if !s.State.IsPlaying {
			playIcon = "⏸"
		}

		fmt.Printf("  %s %s\n", playIcon, s.State.Track.Title)
		fmt.Printf("    %s — %s\n", s.State.Track.Artist, s.State.Track.Album)

		// Progress bar
		progressBar := formatProgressBar(s.State.ProgressPercent(), 30)
		fmt.Printf("    %s %s / %s\n",
			progressBar,
			formatDuration(s.State.Progress),
			formatDuration(s.State.Track.Duration))

		// Device info
		if s.Device != nil {
			fmt.Printf("    📱 %s", s.Device.Name)
			if s.State.Volume > 0 {
				fmt.Printf(" (🔊 %d%%)", s.State.Volume)
			}
			fmt.Println()
		}
	}

	return nil
}

func formatProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("━", filled) + strings.Repeat("─", width-filled)
	return bar
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	return fmt.Sprintf("%d:%02d", m, s)
}
