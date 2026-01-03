package player

import (
	"context"
	"time"

	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/spotify/client"
)

// Player implements core.Player for Spotify.
type Player struct {
	client   *client.Client
	deviceID string // Optional: target device ID
}

// New creates a new Spotify player.
func New(c *client.Client) *Player {
	return &Player{client: c}
}

// SetDevice sets the target device for playback commands.
func (p *Player) SetDevice(deviceID string) {
	p.deviceID = deviceID
}

// Play starts or resumes playback.
func (p *Player) Play(ctx context.Context) error {
	return p.client.Play(ctx, p.deviceID, nil)
}

// PlayURI starts playback of a specific URI (track, album, playlist).
func (p *Player) PlayURI(ctx context.Context, uri string) error {
	return p.client.Play(ctx, p.deviceID, &client.PlayOptions{
		URIs: []string{uri},
	})
}

// PlayContext starts playback of a context (album, playlist) at a specific position.
func (p *Player) PlayContext(ctx context.Context, contextURI string, offset int) error {
	return p.client.Play(ctx, p.deviceID, &client.PlayOptions{
		ContextURI: contextURI,
		Offset:     &client.PlayOffset{Position: offset},
	})
}

// Pause pauses playback.
func (p *Player) Pause(ctx context.Context) error {
	return p.client.Pause(ctx, p.deviceID)
}

// Next skips to the next track.
func (p *Player) Next(ctx context.Context) error {
	return p.client.Next(ctx, p.deviceID)
}

// Prev skips to the previous track.
func (p *Player) Prev(ctx context.Context) error {
	return p.client.Previous(ctx, p.deviceID)
}

// Seek seeks to a position in the current track.
func (p *Player) Seek(ctx context.Context, positionMs int) error {
	return p.client.Seek(ctx, positionMs, p.deviceID)
}

// Volume sets the playback volume (0-100).
func (p *Player) Volume(ctx context.Context, percent int) error {
	return p.client.SetVolume(ctx, percent, p.deviceID)
}

// GetState returns the current playback state.
func (p *Player) GetState(ctx context.Context) (*core.PlaybackState, error) {
	state, err := p.client.GetPlaybackState(ctx)
	if err != nil {
		return nil, err
	}

	if state == nil {
		return &core.PlaybackState{}, nil
	}

	coreState := &core.PlaybackState{
		IsPlaying: state.IsPlaying,
		Progress:  time.Duration(state.ProgressMS) * time.Millisecond,
	}

	if state.Device.VolumePercent != nil {
		coreState.Volume = *state.Device.VolumePercent
	}

	if state.Device.ID != "" {
		coreState.Device = convertDevice(&state.Device)
	}

	if state.Item != nil {
		coreState.Track = convertTrack(state.Item)
	}

	return coreState, nil
}

// GetQueue returns the current playback queue.
func (p *Player) GetQueue(ctx context.Context) (*core.Queue, error) {
	queue, err := p.client.GetQueue(ctx)
	if err != nil {
		return nil, err
	}

	coreQueue := &core.Queue{
		Tracks: make([]core.Track, 0, len(queue.Queue)+1),
	}

	// Add currently playing track as first in queue
	if queue.CurrentlyPlaying != nil {
		coreQueue.Tracks = append(coreQueue.Tracks, *convertTrack(queue.CurrentlyPlaying))
	}

	// Add queued tracks
	for _, t := range queue.Queue {
		coreQueue.Tracks = append(coreQueue.Tracks, *convertTrack(&t))
	}

	return coreQueue, nil
}

// GetRecentlyPlayed returns the user's recently played tracks.
func (p *Player) GetRecentlyPlayed(ctx context.Context, limit int) ([]core.HistoryEntry, error) {
	resp, err := p.client.GetRecentlyPlayed(ctx, limit)
	if err != nil {
		return nil, err
	}

	entries := make([]core.HistoryEntry, len(resp.Items))
	for i, item := range resp.Items {
		playedAt, _ := time.Parse(time.RFC3339, item.PlayedAt)
		entries[i] = core.HistoryEntry{
			Track:    convertTrack(&item.Track),
			PlayedAt: playedAt,
		}
	}
	return entries, nil
}

// AddToQueue adds a track to the playback queue.
func (p *Player) AddToQueue(ctx context.Context, trackURI string) error {
	return p.client.AddToQueue(ctx, trackURI, p.deviceID)
}

// TransferPlayback transfers playback to a different device.
func (p *Player) TransferPlayback(ctx context.Context, deviceID string, play bool) error {
	return p.client.TransferPlayback(ctx, deviceID, play)
}

// GetDevices returns the user's available playback devices.
func (p *Player) GetDevices(ctx context.Context) ([]core.Device, error) {
	devices, err := p.client.GetDevices(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]core.Device, len(devices))
	for i, d := range devices {
		result[i] = *convertDevice(&d)
	}
	return result, nil
}

// convertTrack converts a Spotify track to a core track.
func convertTrack(t *client.Track) *core.Track {
	if t == nil {
		return nil
	}

	artists := make([]string, len(t.Artists))
	for i, a := range t.Artists {
		artists[i] = a.Name
	}

	artist := ""
	if len(artists) > 0 {
		artist = artists[0]
	}

	return &core.Track{
		ID:       t.ID,
		URI:      t.URI,
		Title:    t.Name,
		Artist:   artist,
		Artists:  artists,
		Album:    t.Album.Name,
		Duration: time.Duration(t.DurationMS) * time.Millisecond,
		Source:   core.SourceSpotify,
	}
}

// convertDevice converts a Spotify device to a core device.
func convertDevice(d *client.Device) *core.Device {
	if d == nil {
		return nil
	}

	deviceType := core.DeviceType(d.Type)
	// Map Spotify device types to core types
	switch d.Type {
	case "Computer":
		deviceType = core.DeviceTypeComputer
	case "Smartphone":
		deviceType = core.DeviceTypePhone
	case "Speaker":
		deviceType = core.DeviceTypeSpeaker
	case "TV":
		deviceType = core.DeviceTypeTV
	}

	return &core.Device{
		ID:       d.ID,
		Name:     d.Name,
		Type:     deviceType,
		Platform: core.PlatformSpotify,
		IsActive: d.IsActive,
	}
}

// Ensure Player implements core.Player
var _ core.Player = (*Player)(nil)
