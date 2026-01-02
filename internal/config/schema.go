package config

// Config is the root configuration structure.
type Config struct {
	Spotify  SpotifyConfig  `toml:"spotify"`
	Sonos    SonosConfig    `toml:"sonos"`
	Defaults DefaultsConfig `toml:"defaults"`
	Tail     TailConfig     `toml:"tail"`
	TUI      TUIConfig      `toml:"tui"`
	Log      LogConfig      `toml:"log"`
}

// SpotifyConfig holds Spotify API settings.
type SpotifyConfig struct {
	ClientID    string `toml:"client_id"`
	RedirectURI string `toml:"redirect_uri"`
}

// SonosConfig holds Sonos connection settings.
type SonosConfig struct {
	DefaultRoom      string `toml:"default_room"`
	DiscoveryTimeout int    `toml:"discovery_timeout"`
}

// DefaultsConfig holds default playback settings.
type DefaultsConfig struct {
	Volume   int    `toml:"volume"`
	Shuffle  bool   `toml:"shuffle"`
	Repeat   string `toml:"repeat"`
	Device   string `toml:"device"`
}

// TailConfig holds settings for tail/follow mode.
type TailConfig struct {
	Enabled  bool `toml:"enabled"`
	Interval int  `toml:"interval"`
}

// TUIConfig holds terminal UI settings.
type TUIConfig struct {
	Theme           string `toml:"theme"`
	RefreshInterval int    `toml:"refresh_interval"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `toml:"level"`
	File  string `toml:"file"`
}
