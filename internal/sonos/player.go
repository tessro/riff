package sonos

import (
	"context"
	"fmt"
	"time"

	"github.com/tessro/riff/internal/core"
)

// Player implements core.Player for Sonos devices.
type Player struct {
	client *Client
	device *Device
}

// NewPlayer creates a new Sonos player for the given device.
func NewPlayer(client *Client, device *Device) *Player {
	return &Player{
		client: client,
		device: device,
	}
}

// Play starts playback.
func (p *Player) Play(ctx context.Context) error {
	return p.client.Play(ctx, p.device)
}

// Pause pauses playback.
func (p *Player) Pause(ctx context.Context) error {
	return p.client.Pause(ctx, p.device)
}

// Next skips to the next track.
func (p *Player) Next(ctx context.Context) error {
	return p.client.Next(ctx, p.device)
}

// Prev skips to the previous track.
func (p *Player) Prev(ctx context.Context) error {
	return p.client.Previous(ctx, p.device)
}

// Seek seeks to the specified position in milliseconds.
func (p *Player) Seek(ctx context.Context, positionMs int) error {
	target := formatDuration(time.Duration(positionMs) * time.Millisecond)
	return p.client.Seek(ctx, p.device, target)
}

// Volume sets the volume level (0-100).
func (p *Player) Volume(ctx context.Context, percent int) error {
	return p.client.SetVolume(ctx, p.device, percent)
}

// GetState returns the current playback state.
func (p *Player) GetState(ctx context.Context) (*core.PlaybackState, error) {
	transport, err := p.client.GetTransportInfo(ctx, p.device)
	if err != nil {
		return nil, fmt.Errorf("get transport info: %w", err)
	}

	position, err := p.client.GetPositionInfo(ctx, p.device)
	if err != nil {
		return nil, fmt.Errorf("get position info: %w", err)
	}

	volume, err := p.client.GetVolume(ctx, p.device)
	if err != nil {
		return nil, fmt.Errorf("get volume: %w", err)
	}

	track := parseTrackMetadata(position.TrackMetaData, position.TrackURI)
	if track != nil {
		track.Duration = parseDuration(position.TrackDuration)
	}

	return &core.PlaybackState{
		Track:     track,
		Device:    p.coreDevice(),
		IsPlaying: transport.CurrentTransportState == "PLAYING",
		Progress:  parseDuration(position.RelTime),
		Volume:    volume,
	}, nil
}

// GetQueue returns the current queue.
func (p *Player) GetQueue(ctx context.Context) (*core.Queue, error) {
	// Sonos queue retrieval is more complex, returning empty for now
	return &core.Queue{}, nil
}

// AddToQueue adds a track to the queue.
func (p *Player) AddToQueue(ctx context.Context, trackURI string) error {
	// Would need AddURIToQueue SOAP call
	return fmt.Errorf("not implemented")
}

// coreDevice converts the Sonos device to a core.Device.
func (p *Player) coreDevice() *core.Device {
	return &core.Device{
		ID:       p.device.UUID,
		Name:     p.device.Name,
		Type:     core.DeviceTypeSpeaker,
		Platform: core.PlatformSonos,
		IsActive: true,
	}
}

// formatDuration formats a duration as HH:MM:SS.
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// parseDuration parses a duration string (HH:MM:SS or H:MM:SS).
func parseDuration(s string) time.Duration {
	var h, m, sec int
	fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec)
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
}

// Ensure Player implements core.Player
var _ core.Player = (*Player)(nil)
