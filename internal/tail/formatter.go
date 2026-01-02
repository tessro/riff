package tail

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// Formatter formats events for output.
type Formatter struct {
	showEmoji     bool
	showTimestamp bool
	template      *template.Template
}

// FormatterOption configures a Formatter.
type FormatterOption func(*Formatter)

// WithEmoji enables emoji output.
func WithEmoji(enabled bool) FormatterOption {
	return func(f *Formatter) {
		f.showEmoji = enabled
	}
}

// WithTimestamp enables timestamp output.
func WithTimestamp(enabled bool) FormatterOption {
	return func(f *Formatter) {
		f.showTimestamp = enabled
	}
}

// WithTemplate sets a custom format template.
func WithTemplate(tmpl string) FormatterOption {
	return func(f *Formatter) {
		if tmpl != "" {
			t, err := template.New("format").Parse(tmpl)
			if err == nil {
				f.template = t
			}
		}
	}
}

// NewFormatter creates a new formatter with the given options.
func NewFormatter(opts ...FormatterOption) *Formatter {
	f := &Formatter{
		showEmoji:     true,
		showTimestamp: false,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Format formats an event as a string.
func (f *Formatter) Format(e Event) string {
	if f.template != nil {
		return f.formatTemplate(e)
	}
	return f.formatLine(e)
}

// formatLine formats an event as a simple line.
func (f *Formatter) formatLine(e Event) string {
	var parts []string

	// Timestamp
	if f.showTimestamp {
		parts = append(parts, e.Timestamp.Format("15:04:05"))
	}

	// Emoji
	if f.showEmoji {
		parts = append(parts, eventEmoji(e.Type))
	}

	// Event description
	parts = append(parts, f.eventDescription(e))

	return strings.Join(parts, " ")
}

// formatTemplate formats an event using a custom template.
func (f *Formatter) formatTemplate(e Event) string {
	data := templateData{
		Type:      eventTypeName(e.Type),
		Emoji:     eventEmoji(e.Type),
		Timestamp: e.Timestamp,
		Time:      e.Timestamp.Format("15:04:05"),
	}

	if e.Current != nil && e.Current.Track != nil {
		data.Title = e.Current.Track.Title
		data.Artist = e.Current.Track.Artist
		data.Album = e.Current.Track.Album
	}

	if e.Current != nil && e.Current.Device != nil {
		data.Device = e.Current.Device.Name
	}

	if e.Current != nil {
		data.Volume = e.Current.Volume
	}

	var buf bytes.Buffer
	if err := f.template.Execute(&buf, data); err != nil {
		return f.formatLine(e)
	}
	return buf.String()
}

type templateData struct {
	Type      string
	Emoji     string
	Timestamp time.Time
	Time      string
	Title     string
	Artist    string
	Album     string
	Device    string
	Volume    int
}

// eventDescription returns a human-readable description of the event.
func (f *Formatter) eventDescription(e Event) string {
	switch e.Type {
	case EventTrackChange:
		if e.Current != nil && e.Current.Track != nil {
			return fmt.Sprintf("Now playing: %s - %s",
				e.Current.Track.Artist,
				e.Current.Track.Title)
		}
		return "Track changed"

	case EventTrackComplete:
		if e.Previous != nil && e.Previous.Track != nil {
			return fmt.Sprintf("Finished: %s - %s",
				e.Previous.Track.Artist,
				e.Previous.Track.Title)
		}
		return "Track completed"

	case EventTrackSkip:
		if e.Previous != nil && e.Previous.Track != nil {
			return fmt.Sprintf("Skipped: %s - %s",
				e.Previous.Track.Artist,
				e.Previous.Track.Title)
		}
		return "Track skipped"

	case EventPause:
		return "Paused"

	case EventResume:
		return "Resumed"

	case EventVolumeChange:
		if e.Current != nil {
			return fmt.Sprintf("Volume: %d%%", e.Current.Volume)
		}
		return "Volume changed"

	case EventDeviceChange:
		if e.Current != nil && e.Current.Device != nil {
			return fmt.Sprintf("Device: %s", e.Current.Device.Name)
		}
		return "Device changed"

	default:
		return "Unknown event"
	}
}

// eventEmoji returns an emoji for the event type.
func eventEmoji(t EventType) string {
	switch t {
	case EventTrackChange:
		return "🎵"
	case EventTrackComplete:
		return "✅"
	case EventTrackSkip:
		return "⏭️"
	case EventPause:
		return "⏸️"
	case EventResume:
		return "▶️"
	case EventVolumeChange:
		return "🔊"
	case EventDeviceChange:
		return "📱"
	default:
		return "❓"
	}
}

// eventTypeName returns the name of the event type.
func eventTypeName(t EventType) string {
	switch t {
	case EventTrackChange:
		return "track_change"
	case EventTrackComplete:
		return "track_complete"
	case EventTrackSkip:
		return "track_skip"
	case EventPause:
		return "pause"
	case EventResume:
		return "resume"
	case EventVolumeChange:
		return "volume_change"
	case EventDeviceChange:
		return "device_change"
	default:
		return "unknown"
	}
}
