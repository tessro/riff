package config

import (
	"errors"
	"fmt"
	"net/url"
)

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	var errs []error

	if err := c.Spotify.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("spotify: %w", err))
	}
	if err := c.Sonos.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("sonos: %w", err))
	}
	if err := c.Defaults.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("defaults: %w", err))
	}
	if err := c.Tail.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tail: %w", err))
	}
	if err := c.TUI.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("tui: %w", err))
	}
	if err := c.Log.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("log: %w", err))
	}

	return errors.Join(errs...)
}

// Validate checks SpotifyConfig for errors.
func (c *SpotifyConfig) Validate() error {
	if c.RedirectURI != "" {
		if _, err := url.Parse(c.RedirectURI); err != nil {
			return fmt.Errorf("invalid redirect_uri: %w", err)
		}
	}
	return nil
}

// Validate checks SonosConfig for errors.
func (c *SonosConfig) Validate() error {
	if c.DiscoveryTimeout < 0 {
		return errors.New("discovery_timeout must be non-negative")
	}
	return nil
}

// Validate checks DefaultsConfig for errors.
func (c *DefaultsConfig) Validate() error {
	if c.Volume < 0 || c.Volume > 100 {
		return errors.New("volume must be between 0 and 100")
	}
	switch c.Repeat {
	case "", "off", "track", "context":
		// valid
	default:
		return fmt.Errorf("invalid repeat mode: %s (must be off, track, or context)", c.Repeat)
	}
	return nil
}

// Validate checks TailConfig for errors.
func (c *TailConfig) Validate() error {
	if c.Interval < 0 {
		return errors.New("interval must be non-negative")
	}
	return nil
}

// Validate checks TUIConfig for errors.
func (c *TUIConfig) Validate() error {
	switch c.Theme {
	case "", "auto", "dark", "light":
		// valid
	default:
		return fmt.Errorf("invalid theme: %s (must be auto, dark, or light)", c.Theme)
	}
	if c.RefreshInterval < 0 {
		return errors.New("refresh_interval must be non-negative")
	}
	return nil
}

// Validate checks LogConfig for errors.
func (c *LogConfig) Validate() error {
	switch c.Level {
	case "", "debug", "info", "warn", "error":
		// valid
	default:
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Level)
	}
	return nil
}
