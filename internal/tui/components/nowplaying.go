package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/tess/riff/internal/core"
	"github.com/tess/riff/internal/tui/styles"
)

// NowPlaying displays the currently playing track
type NowPlaying struct{}

// NewNowPlaying creates a new NowPlaying component
func NewNowPlaying() *NowPlaying {
	return &NowPlaying{}
}

// Render renders the now playing panel
func (n *NowPlaying) Render(state *core.PlaybackState, width, height int, focused bool) string {
	title := styles.PanelTitle("Now Playing", focused)

	var content string
	if state == nil || state.Track == nil {
		content = styles.Muted.Render("No track playing")
	} else {
		content = n.renderTrack(state, width-4)
	}

	panel := styles.Panel("", focused).
		Width(width).
		Height(height)

	return panel.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		content,
	))
}

func (n *NowPlaying) renderTrack(state *core.PlaybackState, width int) string {
	track := state.Track

	// Status icon and track title
	icon := styles.StatusIcon(state.IsPlaying)
	titleStyle := styles.Title.Copy().Width(width - 4)
	title := titleStyle.Render(track.Title)

	// Artist and album
	artist := styles.Subtitle.Render(track.Artist)
	album := styles.Dim.Render(track.Album)

	// Progress bar
	progressWidth := width - 14 // Account for times on either side
	if progressWidth < 10 {
		progressWidth = 10
	}
	progressBar := styles.ProgressBar(state.ProgressPercent(), progressWidth)
	currentTime := formatDuration(state.Progress)
	totalTime := formatDuration(track.Duration)
	progress := fmt.Sprintf("%s %s %s", currentTime, progressBar, totalTime)

	// Device info
	deviceInfo := ""
	if state.Device != nil {
		deviceIcon := styles.DeviceIcon(string(state.Device.Type))
		deviceInfo = fmt.Sprintf("%s %s", deviceIcon, state.Device.Name)
		if state.Volume > 0 {
			deviceInfo += fmt.Sprintf(" 🔊 %d%%", state.Volume)
		}
		deviceInfo = styles.Muted.Render(deviceInfo)
	}

	// Playback controls indicator
	controls := n.renderControls(state)

	return lipgloss.JoinVertical(lipgloss.Left,
		icon+" "+title,
		"  "+artist,
		"  "+album,
		"",
		progress,
		"",
		deviceInfo,
		controls,
	)
}

func (n *NowPlaying) renderControls(state *core.PlaybackState) string {
	var controls string

	// Shuffle indicator
	// Note: We don't have shuffle state in core.PlaybackState yet
	controls += styles.Dim.Render("⏮ ")

	if state.IsPlaying {
		controls += styles.Playing.Render("⏸")
	} else {
		controls += styles.Paused.Render("▶")
	}

	controls += styles.Dim.Render(" ⏭")

	return lipgloss.NewStyle().
		Align(lipgloss.Center).
		Render(controls)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d % time.Minute) / time.Second
	return fmt.Sprintf("%d:%02d", m, s)
}
