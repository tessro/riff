package tail

import (
	"context"
	"time"

	"github.com/tessro/riff/internal/core"
)

// EventType represents the type of playback event.
type EventType int

const (
	EventTrackChange EventType = iota
	EventTrackComplete
	EventTrackSkip
	EventPause
	EventResume
	EventVolumeChange
	EventDeviceChange
)

// Event represents a playback state change.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Previous  *core.PlaybackState
	Current   *core.PlaybackState
}

// Watcher polls a player for state changes and emits events.
type Watcher struct {
	player   core.Player
	interval time.Duration
	events   chan Event
	done     chan struct{}
}

// NewWatcher creates a new state watcher.
func NewWatcher(player core.Player, interval time.Duration) *Watcher {
	if interval == 0 {
		interval = time.Second
	}
	return &Watcher{
		player:   player,
		interval: interval,
		events:   make(chan Event, 16),
		done:     make(chan struct{}),
	}
}

// Events returns the channel of playback events.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Start begins polling for state changes.
func (w *Watcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	defer close(w.events)

	var prev *core.PlaybackState

	// Get initial state
	state, err := w.player.GetState(ctx)
	if err == nil {
		prev = state
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.done:
			return nil
		case <-ticker.C:
			curr, err := w.player.GetState(ctx)
			if err != nil {
				continue
			}

			events := diffStates(prev, curr)
			for _, e := range events {
				select {
				case w.events <- e:
				default:
					// Drop event if channel is full
				}
			}

			prev = curr
		}
	}
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	close(w.done)
}

// diffStates compares two states and returns detected events.
func diffStates(prev, curr *core.PlaybackState) []Event {
	if curr == nil {
		return nil
	}

	now := time.Now()
	var events []Event

	// First poll - no previous state
	if prev == nil {
		if curr.HasTrack() {
			events = append(events, Event{
				Type:      EventTrackChange,
				Timestamp: now,
				Current:   curr,
			})
		}
		return events
	}

	// Track change detection
	if trackChanged(prev, curr) {
		eventType := EventTrackChange

		// Check if it was a completion vs skip
		if prev.HasTrack() && wasCompleted(prev) {
			eventType = EventTrackComplete
		} else if prev.HasTrack() && wasSkipped(prev) {
			eventType = EventTrackSkip
		}

		events = append(events, Event{
			Type:      eventType,
			Timestamp: now,
			Previous:  prev,
			Current:   curr,
		})
	}

	// Pause/Resume detection
	if prev.IsPlaying && !curr.IsPlaying {
		events = append(events, Event{
			Type:      EventPause,
			Timestamp: now,
			Previous:  prev,
			Current:   curr,
		})
	} else if !prev.IsPlaying && curr.IsPlaying {
		events = append(events, Event{
			Type:      EventResume,
			Timestamp: now,
			Previous:  prev,
			Current:   curr,
		})
	}

	// Volume change detection
	if prev.Volume != curr.Volume {
		events = append(events, Event{
			Type:      EventVolumeChange,
			Timestamp: now,
			Previous:  prev,
			Current:   curr,
		})
	}

	// Device change detection
	if deviceChanged(prev, curr) {
		events = append(events, Event{
			Type:      EventDeviceChange,
			Timestamp: now,
			Previous:  prev,
			Current:   curr,
		})
	}

	return events
}

// trackChanged returns true if the track changed.
func trackChanged(prev, curr *core.PlaybackState) bool {
	if prev.Track == nil && curr.Track == nil {
		return false
	}
	if prev.Track == nil || curr.Track == nil {
		return true
	}
	return prev.Track.URI != curr.Track.URI
}

// wasCompleted returns true if the track likely completed naturally.
func wasCompleted(state *core.PlaybackState) bool {
	if state.Track == nil || state.Track.Duration == 0 {
		return false
	}
	// Consider completed if progress is >= 95% of duration
	threshold := float64(state.Track.Duration) * 0.95
	return float64(state.Progress) >= threshold
}

// wasSkipped returns true if the track was likely skipped.
func wasSkipped(state *core.PlaybackState) bool {
	if state.Track == nil || state.Track.Duration == 0 {
		return true // Assume skip if we can't determine
	}
	// Consider skipped if progress is < 95% of duration
	threshold := float64(state.Track.Duration) * 0.95
	return float64(state.Progress) < threshold
}

// deviceChanged returns true if the device changed.
func deviceChanged(prev, curr *core.PlaybackState) bool {
	if prev.Device == nil && curr.Device == nil {
		return false
	}
	if prev.Device == nil || curr.Device == nil {
		return true
	}
	return prev.Device.ID != curr.Device.ID
}
