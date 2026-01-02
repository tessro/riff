package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/tess/riff/internal/core"
	"github.com/tess/riff/internal/tui/styles"
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

	for i, entry := range entries {
		if i >= maxLines {
			break
		}

		track := entry.Track
		if track == nil {
			continue
		}

		// Time ago
		timeAgo := formatTimeAgo(entry.PlayedAt)

		// Status icon
		icon := "✓"
		if entry.Skipped {
			icon = styles.Dim.Render("⏭")
		}

		// Track info
		trackInfo := fmt.Sprintf("%s — %s",
			truncate(track.Title, width-25),
			truncate(track.Artist, 12))

		line := fmt.Sprintf("%s %s  %s",
			styles.Dim.Render(icon),
			trackInfo,
			styles.Dim.Render(timeAgo))

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
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
