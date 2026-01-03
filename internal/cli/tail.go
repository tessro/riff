package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/sonos"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
	"github.com/tessro/riff/internal/spotify/player"
	"github.com/tessro/riff/internal/tail"
)

var (
	tailAll       bool
	tailDevice    string
	tailNoEmoji   bool
	tailTimestamp bool
	tailFormat    string
	tailInterval  time.Duration
)

var tailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Follow playback changes in real-time",
	Long: `Watch for playback state changes and print them as they happen.

Events tracked:
  - Track changes (new song started)
  - Track completions (song finished)
  - Track skips (song skipped before completion)
  - Pause/Resume
  - Volume changes
  - Device changes`,
	RunE: runTail,
}

func init() {
	tailCmd.Flags().BoolVarP(&tailAll, "all", "a", false, "watch all devices")
	tailCmd.Flags().StringVarP(&tailDevice, "device", "d", "", "device to watch")
	tailCmd.Flags().BoolVar(&tailNoEmoji, "no-emoji", false, "disable emoji output")
	tailCmd.Flags().BoolVarP(&tailTimestamp, "timestamp", "t", false, "show timestamps")
	tailCmd.Flags().StringVarP(&tailFormat, "format", "f", "", "custom format template")
	tailCmd.Flags().DurationVarP(&tailInterval, "interval", "i", time.Second, "poll interval")

	rootCmd.AddCommand(tailCmd)
}

func runTail(cmd *cobra.Command, args []string) error {
	// Get player (placeholder - would get from config/discovery)
	player, err := getPlayer()
	if err != nil {
		return fmt.Errorf("get player: %w", err)
	}

	// Create formatter
	formatter := tail.NewFormatter(
		tail.WithEmoji(!tailNoEmoji),
		tail.WithTimestamp(tailTimestamp),
		tail.WithTemplate(tailFormat),
	)

	// Handle Ctrl+C gracefully
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Show recently played tracks and current song on startup
	showInitialState(ctx, player, formatter)

	// Create watcher
	watcher := tail.NewWatcher(player, tailInterval)

	// Start watching in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.Start(ctx)
	}()

	// Print events as they arrive
	for {
		select {
		case event, ok := <-watcher.Events():
			if !ok {
				return nil
			}
			fmt.Println(formatter.Format(event))

		case err := <-errCh:
			if err == context.Canceled {
				return nil
			}
			return err
		}
	}
}

// showInitialState displays recently played tracks and current song on startup.
func showInitialState(ctx context.Context, p core.Player, formatter *tail.Formatter) {
	// Get recently played tracks (show last 5)
	history, err := p.GetRecentlyPlayed(ctx, 5)
	if err == nil && len(history) > 0 {
		// Print in reverse order (oldest first) so newest is at bottom
		for i := len(history) - 1; i >= 0; i-- {
			entry := history[i]
			if entry.Track != nil {
				timestamp := ""
				if tailTimestamp {
					timestamp = entry.PlayedAt.Local().Format("15:04:05") + " "
				}
				emoji := ""
				if !tailNoEmoji {
					emoji = "⏪ "
				}
				fmt.Printf("%s%s%s — %s\n", timestamp, emoji, entry.Track.Artist, entry.Track.Title)
			}
		}
	}

	// Get current state
	state, err := p.GetState(ctx)
	if err == nil && state != nil && state.Track != nil {
		event := tail.Event{
			Type:    tail.EventTrackChange,
			Current: state,
		}
		fmt.Println(formatter.Format(event))
	}
}

// getPlayer returns a player based on config and flags.
func getPlayer() (core.Player, error) {
	ctx := context.Background()

	// Try Spotify first if configured
	if cfg.Spotify.ClientID != "" {
		storage, err := auth.NewTokenStorage("")
		if err == nil {
			spotifyClient := client.New(cfg.Spotify.ClientID, storage)
			if err := spotifyClient.LoadToken(); err == nil && spotifyClient.HasToken() {
				return player.New(spotifyClient), nil
			}
		}
	}

	// Try Sonos discovery
	sonosClient := sonos.NewClient()
	devices, err := sonosClient.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("no player available: spotify not authenticated and sonos discovery failed: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no player available: spotify not authenticated and no sonos devices found")
	}

	// If specific device requested, find it
	if tailDevice != "" {
		groups, err := sonosClient.ListGroups(ctx, devices[0])
		if err == nil {
			for _, g := range groups {
				for _, m := range g.Members {
					if strings.EqualFold(m.Name, tailDevice) || m.UUID == tailDevice {
						return sonos.NewPlayer(sonosClient, m), nil
					}
				}
			}
		}
		return nil, fmt.Errorf("device '%s' not found", tailDevice)
	}

	// Use default room from config or first device
	if cfg.Sonos.DefaultRoom != "" {
		groups, err := sonosClient.ListGroups(ctx, devices[0])
		if err == nil {
			for _, g := range groups {
				if g.Coordinator != nil && strings.EqualFold(g.Coordinator.Name, cfg.Sonos.DefaultRoom) {
					return sonos.NewPlayer(sonosClient, g.Coordinator), nil
				}
			}
		}
	}

	// Fall back to first coordinator
	groups, err := sonosClient.ListGroups(ctx, devices[0])
	if err == nil && len(groups) > 0 && groups[0].Coordinator != nil {
		return sonos.NewPlayer(sonosClient, groups[0].Coordinator), nil
	}

	return sonos.NewPlayer(sonosClient, devices[0]), nil
}
