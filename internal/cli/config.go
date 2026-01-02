package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"github.com/tess/riff/internal/config"
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

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configInitCmd)
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
	defer f.Close()

	// Write header comment
	fmt.Fprintln(f, "# Riff Configuration")
	fmt.Fprintln(f, "# https://github.com/tess/riff")
	fmt.Fprintln(f, "")

	// Write config
	encoder := toml.NewEncoder(f)
	encoder.Indent = "  "
	if err := encoder.Encode(defaultCfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if JSONOutput() {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
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
