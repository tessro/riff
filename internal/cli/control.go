package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/spotify/player"
)

var controlDevice string

var pauseCmd = &cobra.Command{
	Use:   "pause",
	Short: "Pause playback",
	Long:  `Pause the current playback.`,
	RunE:  runPause,
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume playback",
	Long:  `Resume paused playback.`,
	RunE:  runResume,
}

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Skip to next track",
	Long:  `Skip to the next track in the queue.`,
	RunE:  runNext,
}

var prevCmd = &cobra.Command{
	Use:   "prev",
	Short: "Go to previous track",
	Long:  `Go back to the previous track.`,
	RunE:  runPrev,
}

var restartCmd = &cobra.Command{
	Use:     "restart",
	Aliases: []string{"replay"},
	Short:   "Restart current track",
	Long:    `Restart the current track from the beginning.`,
	RunE:    runRestart,
}

var (
	volumeUp   bool
	volumeDown bool
)

var volumeCmd = &cobra.Command{
	Use:   "volume [level]",
	Short: "Set or adjust volume",
	Long: `Set the playback volume (0-100) or adjust it up/down.

Examples:
  riff volume 50      # Set volume to 50%
  riff volume --up    # Increase volume by 10%
  riff volume --down  # Decrease volume by 10%`,
	RunE: runVolume,
}

func init() {
	// Add device flag to all control commands
	pauseCmd.Flags().StringVarP(&controlDevice, "device", "d", "", "Target device")
	resumeCmd.Flags().StringVarP(&controlDevice, "device", "d", "", "Target device")
	nextCmd.Flags().StringVarP(&controlDevice, "device", "d", "", "Target device")
	prevCmd.Flags().StringVarP(&controlDevice, "device", "d", "", "Target device")
	restartCmd.Flags().StringVarP(&controlDevice, "device", "d", "", "Target device")
	volumeCmd.Flags().StringVarP(&controlDevice, "device", "d", "", "Target device")
	volumeCmd.Flags().BoolVar(&volumeUp, "up", false, "Increase volume by 10%")
	volumeCmd.Flags().BoolVar(&volumeDown, "down", false, "Decrease volume by 10%")

	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(prevCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(volumeCmd)
}

func runPause(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	p, err := getSpotifyPlayer(ctx)
	if err != nil {
		return err
	}

	if err := p.Pause(ctx); err != nil {
		return fmt.Errorf("failed to pause: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "paused"})
	} else {
		fmt.Println("⏸ Paused")
	}

	return nil
}

func runResume(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	p, err := getSpotifyPlayer(ctx)
	if err != nil {
		return err
	}

	if err := p.Play(ctx); err != nil {
		return fmt.Errorf("failed to resume: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "playing"})
	} else {
		fmt.Println("▶ Resumed")
	}

	return nil
}

func runNext(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	p, err := getSpotifyPlayer(ctx)
	if err != nil {
		return err
	}

	if err := p.Next(ctx); err != nil {
		return fmt.Errorf("failed to skip: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "skipped"})
	} else {
		fmt.Println("⏭ Skipped to next track")
	}

	return nil
}

func runPrev(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	p, err := getSpotifyPlayer(ctx)
	if err != nil {
		return err
	}

	if err := p.Prev(ctx); err != nil {
		return fmt.Errorf("failed to go back: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "previous"})
	} else {
		fmt.Println("⏮ Previous track")
	}

	return nil
}

func runRestart(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	p, err := getSpotifyPlayer(ctx)
	if err != nil {
		return err
	}

	if err := p.Seek(ctx, 0); err != nil {
		return fmt.Errorf("failed to restart: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "restarted"})
	} else {
		fmt.Println("⏪ Restarted track")
	}

	return nil
}

func runVolume(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	spotifyClient, err := getSpotifyClient()
	if err != nil {
		return err
	}

	p := player.New(spotifyClient)

	if controlDevice != "" {
		resolved, err := resolveDevice(ctx, spotifyClient, controlDevice)
		if err != nil {
			return err
		}
		if resolved.Platform != core.PlatformSpotify {
			return fmt.Errorf("volume control for Sonos devices not yet supported via --device flag")
		}
		p.SetDevice(resolved.SpotifyID)
	}

	// Get current state for relative adjustments or display
	state, err := p.GetState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get playback state: %w", err)
	}

	currentVolume := state.Volume

	var targetVolume int

	if volumeUp {
		targetVolume = currentVolume + 10
		if targetVolume > 100 {
			targetVolume = 100
		}
	} else if volumeDown {
		targetVolume = currentVolume - 10
		if targetVolume < 0 {
			targetVolume = 0
		}
	} else if len(args) > 0 {
		v, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid volume level: %s", args[0])
		}
		if v < 0 || v > 100 {
			return fmt.Errorf("volume must be between 0 and 100")
		}
		targetVolume = v
	} else {
		// Just show current volume
		if JSONOutput() {
			json.NewEncoder(os.Stdout).Encode(map[string]int{"volume": currentVolume})
		} else {
			fmt.Printf("🔊 Volume: %d%%\n", currentVolume)
		}
		return nil
	}

	if err := p.Volume(ctx, targetVolume); err != nil {
		return fmt.Errorf("failed to set volume: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"volume":   targetVolume,
			"previous": currentVolume,
		})
	} else {
		fmt.Printf("🔊 Volume: %d%% (was %d%%)\n", targetVolume, currentVolume)
	}

	return nil
}

func getSpotifyPlayer(ctx context.Context) (*player.Player, error) {
	spotifyClient, err := getSpotifyClient()
	if err != nil {
		return nil, err
	}

	p := player.New(spotifyClient)

	if controlDevice != "" {
		resolved, err := resolveDevice(ctx, spotifyClient, controlDevice)
		if err != nil {
			return nil, err
		}
		if resolved.Platform != core.PlatformSpotify {
			return nil, fmt.Errorf("control commands for Sonos devices not yet supported via --device flag")
		}
		p.SetDevice(resolved.SpotifyID)
	}

	return p, nil
}
