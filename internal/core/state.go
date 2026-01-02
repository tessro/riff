package core

import "time"

// PlaybackState represents the current playback state.
type PlaybackState struct {
	Track     *Track        `json:"track"`
	Device    *Device       `json:"device"`
	Account   string        `json:"account"`
	IsPlaying bool          `json:"is_playing"`
	Progress  time.Duration `json:"progress"`
	Volume    int           `json:"volume"`
}

// HasTrack returns true if there is an active track.
func (s *PlaybackState) HasTrack() bool {
	return s != nil && s.Track != nil
}

// ProgressPercent returns playback progress as a percentage (0-100).
func (s *PlaybackState) ProgressPercent() float64 {
	if s == nil || s.Track == nil || s.Track.Duration == 0 {
		return 0
	}
	return float64(s.Progress) / float64(s.Track.Duration) * 100
}
