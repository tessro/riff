package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/tui/styles"
)

// Queue displays the playback queue
type Queue struct {
	offset   int
	selected int
}

// NewQueue creates a new Queue component
func NewQueue() *Queue {
	return &Queue{
		offset:   0,
		selected: 0,
	}
}

// ScrollDown scrolls the queue down
func (q *Queue) ScrollDown() {
	q.offset++
}

// ScrollUp scrolls the queue up
func (q *Queue) ScrollUp() {
	if q.offset > 0 {
		q.offset--
	}
}

// Selected returns the selected index
func (q *Queue) Selected() int {
	return q.selected
}

// Render renders the queue panel
func (q *Queue) Render(queue *core.Queue, width, height int, focused bool) string {
	title := styles.PanelTitle("Queue", focused)

	var content string
	if queue == nil || queue.IsEmpty() {
		content = styles.Muted.Render("Queue is empty")
	} else {
		content = q.renderQueue(queue, width-4, height-4)
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

func (q *Queue) renderQueue(queue *core.Queue, width, maxLines int) string {
	tracks := queue.Tracks

	// Adjust offset if needed
	if q.offset >= len(tracks) {
		q.offset = 0
	}

	// Calculate visible range
	visibleCount := maxLines - 1 // Leave room for "more" indicator
	if visibleCount < 1 {
		visibleCount = 1
	}

	start := q.offset
	end := start + visibleCount
	if end > len(tracks) {
		end = len(tracks)
	}

	lines := make([]string, 0, end-start+1)

	// Fixed overhead: "XX. " (4) + "▶ " or "  " (2) + " — " (3) = 9 chars
	const overhead = 9

	for i := start; i < end; i++ {
		track := tracks[i]

		// Number
		num := fmt.Sprintf("%2d.", i+1)

		// Calculate available space for title + artist
		available := width - overhead
		titleLen := len(track.Title)
		artistLen := len(track.Artist)
		totalNeeded := titleLen + artistLen

		var title, artist string
		if totalNeeded <= available {
			// Everything fits, no truncation needed
			title = track.Title
			artist = track.Artist
		} else {
			// Need to truncate - give artist at least 1/3 of space (min 10 chars)
			minArtist := available / 3
			if minArtist < 10 {
				minArtist = 10
			}
			if minArtist > available-10 {
				minArtist = available - 10
			}

			artistSpace := minArtist
			if artistLen < artistSpace {
				artistSpace = artistLen
			}
			titleSpace := available - artistSpace

			title = truncate(track.Title, titleSpace)
			artist = truncate(track.Artist, artistSpace)
		}

		// Highlight current track (index 0)
		var line string
		if i == 0 {
			line = styles.Playing.Render(fmt.Sprintf("%s ▶ %s — %s", num, title, artist))
		} else {
			line = fmt.Sprintf("%s   %s — %s",
				styles.Dim.Render(num),
				title,
				styles.Muted.Render(artist))
		}

		lines = append(lines, line)
	}

	// Show "more" indicator
	if end < len(tracks) {
		more := styles.Dim.Render(fmt.Sprintf("    ... and %d more", len(tracks)-end))
		lines = append(lines, more)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
