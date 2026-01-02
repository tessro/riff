package core

import "time"

// Source indicates the origin platform of a track.
type Source string

const (
	SourceSpotify Source = "spotify"
	SourceSonos   Source = "sonos"
)

// Track represents a playable audio track.
type Track struct {
	ID       string        `json:"id"`
	URI      string        `json:"uri"`
	Title    string        `json:"title"`
	Artist   string        `json:"artist"`
	Artists  []string      `json:"artists"`
	Album    string        `json:"album"`
	Duration time.Duration `json:"duration"`
	Source   Source        `json:"source"`
}
