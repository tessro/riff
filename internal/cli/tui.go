package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/tui"
)

var tuiRefresh int

var tuiCmd = &cobra.Command{
	Use:     "ui",
	Aliases: []string{"tui"},
	Short:   "Launch interactive dashboard",
	Long: `Launch the interactive terminal dashboard.

The dashboard provides a live view with:
  • Now Playing - current track, progress, device
  • Queue - upcoming tracks
  • Devices - available playback devices
  • History - recently played tracks

Keyboard shortcuts:
  q, Ctrl+C    Quit
  ?            Help
  /            Search
  Space        Play/Pause
  n            Next track
  p            Previous track
  +/-          Volume up/down
  Tab          Switch panel`,
	RunE: runTUI,
}

func init() {
	tuiCmd.Flags().IntVar(&tuiRefresh, "refresh", 1000, "Refresh interval in milliseconds")
	rootCmd.AddCommand(tuiCmd)
}

func runTUI(cmd *cobra.Command, args []string) error {
	if cfg.Spotify.ClientID == "" {
		return fmt.Errorf("spotify not configured. Set spotify.client_id in ~/.riffrc")
	}

	refreshRate := time.Duration(tuiRefresh) * time.Millisecond
	return tui.Run(cfg.Spotify.ClientID, refreshRate)
}
