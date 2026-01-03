package core

import (
	"context"
	"time"
)

// Player defines the interface for music playback control.
type Player interface {
	// Playback control
	Play(ctx context.Context) error
	Pause(ctx context.Context) error
	Next(ctx context.Context) error
	Prev(ctx context.Context) error
	Seek(ctx context.Context, positionMs int) error

	// Volume control
	Volume(ctx context.Context, percent int) error

	// State queries
	GetState(ctx context.Context) (*PlaybackState, error)
	GetQueue(ctx context.Context) (*Queue, error)
	GetRecentlyPlayed(ctx context.Context, limit int) ([]HistoryEntry, error)

	// Queue manipulation
	AddToQueue(ctx context.Context, trackURI string) error
}

// HistoryEntry represents a recently played track.
type HistoryEntry struct {
	Track    *Track
	PlayedAt time.Time
}
