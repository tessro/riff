package config

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		Spotify: SpotifyConfig{
			RedirectURI: "http://127.0.0.1:8888/callback",
		},
		Sonos: SonosConfig{
			DiscoveryTimeout: 5,
		},
		Defaults: DefaultsConfig{
			Volume:  50,
			Shuffle: false,
			Repeat:  "off",
		},
		Tail: TailConfig{
			Enabled:  false,
			Interval: 1000,
		},
		TUI: TUIConfig{
			Theme:           "auto",
			RefreshInterval: 1000,
		},
		Log: LogConfig{
			Level: "info",
		},
	}
}

// ApplyDefaults fills in zero values with sensible defaults.
func (c *Config) ApplyDefaults() {
	d := Default()

	// Spotify
	if c.Spotify.RedirectURI == "" {
		c.Spotify.RedirectURI = d.Spotify.RedirectURI
	}

	// Sonos
	if c.Sonos.DiscoveryTimeout == 0 {
		c.Sonos.DiscoveryTimeout = d.Sonos.DiscoveryTimeout
	}

	// Defaults
	if c.Defaults.Volume == 0 {
		c.Defaults.Volume = d.Defaults.Volume
	}
	if c.Defaults.Repeat == "" {
		c.Defaults.Repeat = d.Defaults.Repeat
	}

	// Tail
	if c.Tail.Interval == 0 {
		c.Tail.Interval = d.Tail.Interval
	}

	// TUI
	if c.TUI.Theme == "" {
		c.TUI.Theme = d.TUI.Theme
	}
	if c.TUI.RefreshInterval == 0 {
		c.TUI.RefreshInterval = d.TUI.RefreshInterval
	}

	// Log
	if c.Log.Level == "" {
		c.Log.Level = d.Log.Level
	}
}
