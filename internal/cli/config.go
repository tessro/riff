package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/config"
	"github.com/tessro/riff/internal/spotify/auth"
	"github.com/tessro/riff/internal/spotify/client"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Commands for viewing and editing riff configuration.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration values.`,
	RunE:  runConfigShow,
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration file",
	Long:  `Open the configuration file in your default editor.`,
	RunE:  runConfigEdit,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	Long:  `Create a new configuration file with default values.`,
	RunE:  runConfigInit,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Supported keys:
  defaults.device    Default playback device name or ID
  defaults.volume    Default volume (0-100)
  defaults.shuffle   Default shuffle state (true/false)
  defaults.repeat    Default repeat mode (off/track/context)
  spotify.client_id  Spotify client ID

Examples:
  riff config set defaults.device "MacBook Pro"
  riff config set defaults.volume 50`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configSetDeviceCmd = &cobra.Command{
	Use:   "set-device",
	Short: "Interactively select default device",
	Long:  `Shows a picker to select the default playback device.`,
	RunE:  runConfigSetDevice,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configSetDeviceCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	if JSONOutput() {
		return json.NewEncoder(os.Stdout).Encode(cfg)
	}

	// Pretty print as TOML
	encoder := toml.NewEncoder(os.Stdout)
	encoder.Indent = "  "
	return encoder.Encode(cfg)
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found at %s. Run 'riff config init' first", configPath)
	}

	// Find editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"nano", "vim", "vi", "notepad"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found. Set EDITOR environment variable")
	}

	// Open editor
	editorCmd := exec.Command(editor, configPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	return editorCmd.Run()
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config
	defaultCfg := config.Default()

	// Write to file
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Write header comment
	_, _ = fmt.Fprintln(f, "# Riff Configuration")
	_, _ = fmt.Fprintln(f, "# https://github.com/tessro/riff")
	_, _ = fmt.Fprintln(f, "")

	// Write config
	encoder := toml.NewEncoder(f)
	encoder.Indent = "  "
	if err := encoder.Encode(defaultCfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if JSONOutput() {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "created",
			"path":   configPath,
		})
	} else {
		fmt.Printf("Created config file: %s\n", configPath)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Set your Spotify client ID in the config file or via RIFF_SPOTIFY_CLIENT_ID")
		fmt.Println("  2. Run 'riff auth login' to authenticate with Spotify")
	}

	return nil
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ".riffrc"
	}

	return filepath.Join(home, ".riffrc")
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	configPath := getConfigPath()

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found at %s. Run 'riff config init' first", configPath)
	}

	// Read the current config file as raw TOML
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Parse and update based on key
	var rawConfig map[string]interface{}
	if _, err := toml.Decode(string(data), &rawConfig); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	// Parse the key (e.g., "defaults.device" -> ["defaults", "device"])
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid key format. Use 'section.key' (e.g., defaults.device)")
	}

	section, field := parts[0], parts[1]

	// Get or create the section
	sectionMap, ok := rawConfig[section].(map[string]interface{})
	if !ok {
		sectionMap = make(map[string]interface{})
		rawConfig[section] = sectionMap
	}

	// Convert value to appropriate type based on field
	var typedValue interface{}
	switch key {
	case "defaults.volume", "sonos.discovery_timeout", "tail.interval", "tui.refresh_interval":
		// Integer fields
		i, err := fmt.Sscanf(value, "%d", &typedValue)
		if err != nil || i != 1 {
			return fmt.Errorf("value must be an integer for %s", key)
		}
		var intVal int
		_, _ = fmt.Sscanf(value, "%d", &intVal)
		typedValue = intVal
	case "defaults.shuffle", "tail.enabled":
		// Boolean fields
		typedValue = value == "true" || value == "1" || value == "yes"
	default:
		// String fields
		typedValue = value
	}

	sectionMap[field] = typedValue

	// Write back to file
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Write header comment
	_, _ = fmt.Fprintln(f, "# Riff Configuration")
	_, _ = fmt.Fprintln(f, "# https://github.com/tessro/riff")
	_, _ = fmt.Fprintln(f, "")

	encoder := toml.NewEncoder(f)
	encoder.Indent = "  "
	if err := encoder.Encode(rawConfig); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if JSONOutput() {
		_ = json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status": "updated",
			"key":    key,
			"value":  value,
		})
	} else {
		fmt.Printf("Set %s = %s\n", key, value)
	}

	return nil
}

func runConfigSetDevice(cmd *cobra.Command, args []string) error {
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

	// Fetch available devices
	devices, err := spotifyClient.GetDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to get devices: %w", err)
	}

	if len(devices) == 0 {
		return fmt.Errorf("no devices found. Make sure Spotify is open on at least one device")
	}

	// Build options for picker
	var options []huh.Option[string]
	for _, d := range devices {
		label := d.Name
		if d.Type != "" {
			label = fmt.Sprintf("%s (%s)", d.Name, d.Type)
		}
		if d.IsActive {
			label = label + " [active]"
		}
		options = append(options, huh.NewOption(label, d.ID))
	}

	var selectedID string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select default device").
				Description("This device will be used when no active device is found").
				Options(options...).
				Value(&selectedID),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("selection cancelled: %w", err)
	}

	// Find the device name for display
	var deviceName string
	for _, d := range devices {
		if d.ID == selectedID {
			deviceName = d.Name
			break
		}
	}

	// Save to config using the set command logic
	return runConfigSet(cmd, []string{"defaults.device", deviceName})
}
