package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
)

// Load reads configuration from standard locations with environment overrides.
// Search order: ~/.riffrc, $XDG_CONFIG_HOME/riff/config.toml, ~/.config/riff/config.toml
func Load() (*Config, error) {
	cfg := &Config{}

	// Try loading from file
	path := findConfigFile()
	if path != "" {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, err
		}
	}

	// Apply defaults, then environment variable overrides
	cfg.ApplyDefaults()
	applyEnvOverrides(cfg)

	return cfg, nil
}

// LoadFrom reads configuration from a specific file path.
func LoadFrom(path string) (*Config, error) {
	cfg := &Config{}
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	cfg.ApplyDefaults()
	applyEnvOverrides(cfg)
	return cfg, nil
}

// findConfigFile returns the first existing config file path.
func findConfigFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	paths := []string{
		filepath.Join(home, ".riffrc"),
	}

	// XDG_CONFIG_HOME or default
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(home, ".config")
	}
	paths = append(paths, filepath.Join(xdgConfig, "riff", "config.toml"))

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}

// applyEnvOverrides applies environment variable overrides to the config.
func applyEnvOverrides(cfg *Config) {
	// Spotify
	if v := os.Getenv("RIFF_SPOTIFY_CLIENT_ID"); v != "" {
		cfg.Spotify.ClientID = v
	}
	if v := os.Getenv("RIFF_SPOTIFY_REDIRECT_URI"); v != "" {
		cfg.Spotify.RedirectURI = v
	}

	// Sonos
	if v := os.Getenv("RIFF_SONOS_DEFAULT_ROOM"); v != "" {
		cfg.Sonos.DefaultRoom = v
	}
	if v := os.Getenv("RIFF_SONOS_DISCOVERY_TIMEOUT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Sonos.DiscoveryTimeout = i
		}
	}

	// TUI
	if v := os.Getenv("RIFF_TUI_THEME"); v != "" {
		cfg.TUI.Theme = v
	}
	if v := os.Getenv("RIFF_TUI_REFRESH_INTERVAL"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.TUI.RefreshInterval = i
		}
	}

	// Log
	if v := os.Getenv("RIFF_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("RIFF_LOG_FILE"); v != "" {
		cfg.Log.File = v
	}
}
