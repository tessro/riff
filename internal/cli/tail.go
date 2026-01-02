package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/core"
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

	// Create watcher
	watcher := tail.NewWatcher(player, tailInterval)

	// Handle Ctrl+C gracefully
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

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

// getPlayer returns a player based on config.
// This is a placeholder that will be implemented when integrating with Spotify/Sonos.
func getPlayer() (core.Player, error) {
	// TODO: Get player from config/discovery
	// For now, return an error indicating no player is configured
	return nil, fmt.Errorf("no player configured - run 'riff auth' first")
}
