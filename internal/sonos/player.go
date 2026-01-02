package sonos

import (
	"context"
	"fmt"
	"strings"
	"sync"
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
	// Fetch transport, position, and volume in parallel
	type result struct {
		transport *TransportInfo
		position  *PositionInfo
		volume    int
		err       error
	}

	ch := make(chan result, 1)
	go func() {
		var r result
		var wg sync.WaitGroup
		var mu sync.Mutex

		wg.Add(3)

		go func() {
			defer wg.Done()
			t, err := p.client.GetTransportInfo(ctx, p.device)
			mu.Lock()
			if err != nil && r.err == nil {
				r.err = fmt.Errorf("get transport info: %w", err)
			}
			r.transport = t
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			pos, err := p.client.GetPositionInfo(ctx, p.device)
			mu.Lock()
			if err != nil && r.err == nil {
				r.err = fmt.Errorf("get position info: %w", err)
			}
			r.position = pos
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			v, err := p.client.GetVolume(ctx, p.device)
			mu.Lock()
			if err != nil && r.err == nil {
				r.err = fmt.Errorf("get volume: %w", err)
			}
			r.volume = v
			mu.Unlock()
		}()

		wg.Wait()
		ch <- r
	}()

	r := <-ch
	if r.err != nil {
		return nil, r.err
	}

	track := parseTrackMetadata(r.position.TrackMetaData, r.position.TrackURI)
	if track != nil {
		track.Duration = parseDuration(r.position.TrackDuration)
	}

	return &core.PlaybackState{
		Track:     track,
		Device:    p.coreDevice(),
		IsPlaying: r.transport.CurrentTransportState == "PLAYING",
		Progress:  parseDuration(r.position.RelTime),
		Volume:    r.volume,
	}, nil
}

// GetQueue returns the current queue.
func (p *Player) GetQueue(ctx context.Context) (*core.Queue, error) {
	// Sonos queue retrieval is more complex, returning empty for now
	return &core.Queue{}, nil
}

// AddToQueue adds a track to the queue.
func (p *Player) AddToQueue(ctx context.Context, trackURI string) error {
	return p.client.AddURIToQueue(ctx, p.device, trackURI, "")
}

// PlayURI plays a specific URI on the device.
func (p *Player) PlayURI(ctx context.Context, uri string) error {
	sonosURI, _ := ConvertSpotifyURIWithMetadata(uri)

	// For Spotify tracks, try direct SetAVTransportURI first
	if strings.HasPrefix(uri, "spotify:track:") {
		return p.client.PlayURI(ctx, p.device, sonosURI, "")
	}

	// For containers, use queue approach
	if strings.HasPrefix(uri, "spotify:") {
		// Clear queue errors are non-fatal
		_ = p.client.ClearQueue(ctx, p.device)
		if err := p.client.AddURIToQueue(ctx, p.device, sonosURI, ""); err != nil {
			return fmt.Errorf("add to queue: %w", err)
		}
		return p.client.PlayFromQueue(ctx, p.device)
	}

	// Non-Spotify URIs
	return p.client.PlayURI(ctx, p.device, sonosURI, "")
}

// ConvertSpotifyURIWithMetadata converts a Spotify URI to Sonos format with DIDL-Lite metadata.
func ConvertSpotifyURIWithMetadata(uri string) (sonosURI, metadata string) {
	if !strings.HasPrefix(uri, "spotify:") {
		return uri, ""
	}

	// Sonos uses the spotify URI directly (not URL-encoded) for most operations
	// sid=12 is Spotify's service ID on Sonos
	suffix := "?sid=12&flags=8224&sn=1"

	// Different URI schemes for different content types
	switch {
	case strings.HasPrefix(uri, "spotify:track:"):
		sonosURI = "x-sonos-spotify:" + uri + suffix
		metadata = ""
	case strings.HasPrefix(uri, "spotify:album:"):
		sonosURI = "x-rincon-cpcontainer:1004206c" + uri + suffix
		metadata = ""
	case strings.HasPrefix(uri, "spotify:playlist:"):
		sonosURI = "x-rincon-cpcontainer:1006206c" + uri + suffix
		metadata = ""
	case strings.HasPrefix(uri, "spotify:artist:"):
		sonosURI = "x-rincon-cpcontainer:1006206c" + uri + suffix
		metadata = ""
	default:
		sonosURI = "x-sonos-spotify:" + uri + suffix
		metadata = ""
	}
	return
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
	_, _ = fmt.Sscanf(s, "%d:%d:%d", &h, &m, &sec)
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
}

// Ensure Player implements core.Player
var _ core.Player = (*Player)(nil)
