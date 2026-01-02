package player

import (
	"testing"
	"time"

	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/spotify/client"
)

func TestConvertTrack(t *testing.T) {
	spotifyTrack := &client.Track{
		ID:         "track123",
		URI:        "spotify:track:track123",
		Name:       "Test Song",
		DurationMS: 180000,
		Artists: []client.Artist{
			{Name: "Artist One"},
			{Name: "Artist Two"},
		},
		Album: client.Album{
			Name: "Test Album",
		},
	}

	coreTrack := convertTrack(spotifyTrack)

	if coreTrack.ID != "track123" {
		t.Errorf("ID = %q, want %q", coreTrack.ID, "track123")
	}
	if coreTrack.Title != "Test Song" {
		t.Errorf("Title = %q, want %q", coreTrack.Title, "Test Song")
	}
	if coreTrack.Artist != "Artist One" {
		t.Errorf("Artist = %q, want %q", coreTrack.Artist, "Artist One")
	}
	if len(coreTrack.Artists) != 2 {
		t.Errorf("Artists count = %d, want 2", len(coreTrack.Artists))
	}
	if coreTrack.Album != "Test Album" {
		t.Errorf("Album = %q, want %q", coreTrack.Album, "Test Album")
	}
	if coreTrack.Duration != 180*time.Second {
		t.Errorf("Duration = %v, want %v", coreTrack.Duration, 180*time.Second)
	}
	if coreTrack.Source != core.SourceSpotify {
		t.Errorf("Source = %q, want %q", coreTrack.Source, core.SourceSpotify)
	}
}

func TestConvertDevice(t *testing.T) {
	spotifyDevice := &client.Device{
		ID:       "device123",
		Name:     "My Speaker",
		Type:     "Speaker",
		IsActive: true,
	}

	coreDevice := convertDevice(spotifyDevice)

	if coreDevice.ID != "device123" {
		t.Errorf("ID = %q, want %q", coreDevice.ID, "device123")
	}
	if coreDevice.Name != "My Speaker" {
		t.Errorf("Name = %q, want %q", coreDevice.Name, "My Speaker")
	}
	if coreDevice.Type != core.DeviceTypeSpeaker {
		t.Errorf("Type = %q, want %q", coreDevice.Type, core.DeviceTypeSpeaker)
	}
	if coreDevice.Platform != core.PlatformSpotify {
		t.Errorf("Platform = %q, want %q", coreDevice.Platform, core.PlatformSpotify)
	}
	if !coreDevice.IsActive {
		t.Error("IsActive = false, want true")
	}
}

func TestConvertNilTrack(t *testing.T) {
	result := convertTrack(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}
}

func TestConvertNilDevice(t *testing.T) {
	result := convertDevice(nil)
	if result != nil {
		t.Error("Expected nil for nil input")
	}
}
