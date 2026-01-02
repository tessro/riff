package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tessro/riff/internal/config"
)

var (
	cfgFile string
	jsonOut bool
	verbose bool

	cfg *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "riff",
	Short: "Control Spotify and Sonos from the command line",
	Long:  `Riff is a unified CLI for controlling music playback across Spotify and Sonos devices.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig()
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ~/.riffrc)")
	rootCmd.PersistentFlags().BoolVarP(&jsonOut, "json", "j", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func initConfig() error {
	var err error
	if cfgFile != "" {
		cfg, err = config.LoadFrom(cfgFile)
	} else {
		cfg, err = config.Load()
	}
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	return nil
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// Config returns the loaded configuration.
func Config() *config.Config {
	return cfg
}

// JSONOutput returns true if JSON output is requested.
func JSONOutput() bool {
	return jsonOut
}

// Verbose returns true if verbose output is requested.
func Verbose() bool {
	return verbose
}
