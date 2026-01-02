package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/sonos"
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
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "paused"})
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
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "playing"})
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
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "skipped"})
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
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "previous"})
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
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{"status": "restarted"})
	} else {
		fmt.Println("⏪ Restarted track")
	}

	return nil
}

func runVolume(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Determine target volume from args/flags
	var targetVolume *int
	if volumeUp || volumeDown || len(args) > 0 {
		v := 0
		targetVolume = &v
		if len(args) > 0 {
			val, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid volume level: %s", args[0])
			}
			if val < 0 || val > 100 {
				return fmt.Errorf("volume must be between 0 and 100")
			}
			*targetVolume = val
		}
	}

	// Try to find active playback - check Sonos first since it's local
	sonosPlayer, sonosState := getActiveSonosPlayer(ctx)

	// If Sonos is playing, control it
	if sonosPlayer != nil && sonosState != nil && sonosState.IsPlaying {
		return runVolumeOnPlayer(ctx, sonosPlayer, sonosState.Volume, targetVolume, "sonos")
	}

	// Otherwise try Spotify
	spotifyClient, err := getSpotifyClient()
	if err != nil {
		// If no Spotify and we found a Sonos (even if not playing), use that
		if sonosPlayer != nil && sonosState != nil {
			return runVolumeOnPlayer(ctx, sonosPlayer, sonosState.Volume, targetVolume, "sonos")
		}
		return err
	}

	p := player.New(spotifyClient)

	if controlDevice != "" {
		resolved, err := resolveDevice(ctx, spotifyClient, controlDevice)
		if err != nil {
			return err
		}
		if resolved.Platform == core.PlatformSonos && resolved.SonosDevice != nil {
			// Use Sonos for this device
			sonosClient := sonos.NewClient()
			sp := sonos.NewPlayer(sonosClient, resolved.SonosDevice)
			state, _ := sp.GetState(ctx)
			vol := 0
			if state != nil {
				vol = state.Volume
			}
			return runVolumeOnPlayer(ctx, sp, vol, targetVolume, "sonos")
		}
		p.SetDevice(resolved.SpotifyID)
	}

	state, err := p.GetState(ctx)
	if err != nil {
		// If Spotify fails but we have Sonos available, use that
		if sonosPlayer != nil && sonosState != nil {
			return runVolumeOnPlayer(ctx, sonosPlayer, sonosState.Volume, targetVolume, "sonos")
		}
		return fmt.Errorf("failed to get playback state: %w", err)
	}

	return runVolumeOnPlayer(ctx, p, state.Volume, targetVolume, "spotify")
}

// volumeController is an interface for volume control across platforms.
type volumeController interface {
	Volume(ctx context.Context, percent int) error
}

func runVolumeOnPlayer(ctx context.Context, p volumeController, currentVolume int, targetVolume *int, platform string) error {
	if targetVolume == nil {
		// Just show current volume
		if JSONOutput() {
			_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
				"volume":   currentVolume,
				"platform": platform,
			})
		} else {
			fmt.Printf("🔊 Volume: %d%% (%s)\n", currentVolume, platform)
		}
		return nil
	}

	// Calculate target if relative
	target := *targetVolume
	if volumeUp {
		target = currentVolume + 10
		if target > 100 {
			target = 100
		}
	} else if volumeDown {
		target = currentVolume - 10
		if target < 0 {
			target = 0
		}
	}

	if err := p.Volume(ctx, target); err != nil {
		return fmt.Errorf("failed to set volume: %w", err)
	}

	if JSONOutput() {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"volume":   target,
			"previous": currentVolume,
			"platform": platform,
		})
	} else {
		fmt.Printf("🔊 Volume: %d%% (was %d%%) [%s]\n", target, currentVolume, platform)
	}

	return nil
}

// getActiveSonosPlayer returns a Sonos player and its state if one is actively playing.
func getActiveSonosPlayer(ctx context.Context) (*sonos.Player, *core.PlaybackState) {
	client := sonos.NewClient()

	devices, err := client.Discover(ctx)
	if err != nil || len(devices) == 0 {
		return nil, nil
	}

	groups, err := client.ListGroups(ctx, devices[0])
	if err != nil {
		return nil, nil
	}

	// Find a playing group
	for _, g := range groups {
		if g.Coordinator == nil {
			continue
		}
		player := sonos.NewPlayer(client, g.Coordinator)
		state, err := player.GetState(ctx)
		if err != nil {
			continue
		}
		if state.IsPlaying || state.Track != nil {
			return player, state
		}
	}

	// No playing group found, return the first coordinator anyway
	for _, g := range groups {
		if g.Coordinator != nil {
			player := sonos.NewPlayer(client, g.Coordinator)
			state, _ := player.GetState(ctx)
			return player, state
		}
	}

	return nil, nil
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
