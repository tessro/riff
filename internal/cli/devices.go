package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
	"github.com/tessro/riff/internal/spotify/player"
)

var devicesRefresh bool

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List available playback devices",
	Long:  `Lists all available playback devices across Spotify and Sonos.`,
	RunE:  runDevices,
}

func init() {
	devicesCmd.Flags().BoolVarP(&devicesRefresh, "refresh", "r", false, "Force refresh device list")
	rootCmd.AddCommand(devicesCmd)
}

func runDevices(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	var allDevices []deviceInfo

	// Get Spotify devices
	spotifyDevices, err := getSpotifyDevices(ctx)
	if err != nil {
		if Verbose() {
			fmt.Fprintf(os.Stderr, "Spotify error: %v\n", err)
		}
	} else {
		allDevices = append(allDevices, spotifyDevices...)
	}

	// TODO: Get Sonos devices when Phase 3 is complete

	if len(allDevices) == 0 {
		if JSONOutput() {
			json.NewEncoder(os.Stdout).Encode([]interface{}{})
		} else {
			fmt.Println("No devices found")
		}
		return nil
	}

	if JSONOutput() {
		return outputDevicesJSON(allDevices)
	}
	return outputDevicesTable(allDevices)
}

type deviceInfo struct {
	Device   *core.Device
	Volume   *int
	Platform string
}

func getSpotifyDevices(ctx context.Context) ([]deviceInfo, error) {
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
	devices, err := p.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]deviceInfo, len(devices))
	for i, d := range devices {
		dev := d // Copy to avoid aliasing
		result[i] = deviceInfo{
			Device:   &dev,
			Platform: "spotify",
		}
	}

	return result, nil
}

func outputDevicesJSON(devices []deviceInfo) error {
	output := make([]map[string]interface{}, 0, len(devices))

	for _, d := range devices {
		item := map[string]interface{}{
			"id":        d.Device.ID,
			"name":      d.Device.Name,
			"type":      d.Device.Type,
			"platform":  d.Platform,
			"is_active": d.Device.IsActive,
		}
		if d.Volume != nil {
			item["volume"] = *d.Volume
		}
		output = append(output, item)
	}

	return json.NewEncoder(os.Stdout).Encode(output)
}

func outputDevicesTable(devices []deviceInfo) error {
	// Group by platform
	spotify := make([]deviceInfo, 0)
	sonos := make([]deviceInfo, 0)

	for _, d := range devices {
		switch d.Platform {
		case "spotify":
			spotify = append(spotify, d)
		case "sonos":
			sonos = append(sonos, d)
		}
	}

	if len(spotify) > 0 {
		fmt.Println("[SPOTIFY]")
		for _, d := range spotify {
			printDevice(d)
		}
	}

	if len(sonos) > 0 {
		if len(spotify) > 0 {
			fmt.Println()
		}
		fmt.Println("[SONOS]")
		for _, d := range sonos {
			printDevice(d)
		}
	}

	return nil
}

func printDevice(d deviceInfo) {
	icon := getDeviceIcon(d.Device.Type)
	active := ""
	if d.Device.IsActive {
		active = " ●"
	}

	fmt.Printf("  %s %s%s\n", icon, d.Device.Name, active)

	if Verbose() {
		fmt.Printf("      ID: %s\n", d.Device.ID)
		fmt.Printf("      Type: %s\n", d.Device.Type)
	}
}

func getDeviceIcon(deviceType core.DeviceType) string {
	switch deviceType {
	case core.DeviceTypeComputer:
		return "💻"
	case core.DeviceTypePhone:
		return "📱"
	case core.DeviceTypeSpeaker:
		return "🔊"
	case core.DeviceTypeTV:
		return "📺"
	case core.DeviceTypeSoundbar:
		return "🎵"
	default:
		return "🎧"
	}
}
