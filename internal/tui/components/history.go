package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/tui/styles"
)

// HistoryEntry represents a track in play history
type HistoryEntry struct {
	Track    *core.Track
	PlayedAt time.Time
	Skipped  bool
}

// History displays recently played tracks
type History struct {
	offset int
}

// NewHistory creates a new History component
func NewHistory() *History {
	return &History{offset: 0}
}

// Render renders the history panel
func (h *History) Render(entries []HistoryEntry, width, height int, focused bool) string {
	title := styles.PanelTitle("History", focused)

	var content string
	if len(entries) == 0 {
		content = styles.Muted.Render("No history yet")
	} else {
		content = h.renderHistory(entries, width-4, height-4)
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

func (h *History) renderHistory(entries []HistoryEntry, width, maxLines int) string {
	lines := make([]string, 0, maxLines)

	// Fixed overhead: icon (2) + " " (1) + " — " (3) + padding for time (8)
	const overhead = 14

	for i, entry := range entries {
		if i >= maxLines {
			break
		}

		track := entry.Track
		if track == nil {
			continue
		}

		// Time ago (right-aligned)
		timeAgo := formatTimeAgo(entry.PlayedAt)
		timeWidth := len(timeAgo)

		// Status icon
		icon := "✓"
		if entry.Skipped {
			icon = "⏭"
		}

		// Calculate available space for title + artist
		available := width - overhead - timeWidth
		titleLen := len(track.Title)
		artistLen := len(track.Artist)
		totalNeeded := titleLen + artistLen

		var title, artist string
		if totalNeeded <= available {
			// Everything fits
			title = track.Title
			artist = track.Artist
		} else {
			// Need to truncate - give artist at least 1/3 of space (min 8 chars)
			minArtist := available / 3
			if minArtist < 8 {
				minArtist = 8
			}
			if minArtist > available-8 {
				minArtist = available - 8
			}

			artistSpace := minArtist
			if artistLen < artistSpace {
				artistSpace = artistLen
			}
			titleSpace := available - artistSpace

			title = truncate(track.Title, titleSpace)
			artist = truncate(track.Artist, artistSpace)
		}

		// Build track info
		trackInfo := fmt.Sprintf("%s — %s", title, artist)
		trackInfoLen := len(title) + 3 + len(artist) // " — " is 3 chars

		// Calculate padding for right-alignment
		padding := width - 2 - trackInfoLen - timeWidth // 2 for icon + space
		if padding < 1 {
			padding = 1
		}

		line := fmt.Sprintf("%s %s%s%s",
			styles.Dim.Render(icon),
			trackInfo,
			lipgloss.NewStyle().Width(padding).Render(""),
			styles.Dim.Render(timeAgo))

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return t.Format("Jan 2")
}

// SearchResult represents a search result for the overlay
type SearchResult struct {
	Type   string // track, album, artist, playlist
	Name   string
	Artist string
	URI    string
}
