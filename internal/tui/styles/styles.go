package styles

import "github.com/charmbracelet/lipgloss"

// Colors - a pleasant color palette
var (
	// Primary colors
	Primary     = lipgloss.Color("#7C3AED") // Purple
	Secondary   = lipgloss.Color("#10B981") // Green
	Accent      = lipgloss.Color("#F59E0B") // Amber

	// Status colors
	Success     = lipgloss.Color("#10B981") // Green
	Warning     = lipgloss.Color("#F59E0B") // Amber
	Error       = lipgloss.Color("#EF4444") // Red
	Info        = lipgloss.Color("#3B82F6") // Blue

	// Neutral colors
	Background  = lipgloss.Color("#1F2937") // Dark gray
	Surface     = lipgloss.Color("#374151") // Medium gray
	Border      = lipgloss.Color("#4B5563") // Light gray
	Text        = lipgloss.Color("#F9FAFB") // White
	TextMuted   = lipgloss.Color("#9CA3AF") // Gray
	TextDim     = lipgloss.Color("#6B7280") // Darker gray

	// Spotify green
	SpotifyGreen = lipgloss.Color("#1DB954")
)

// Text styles
var (
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Text)

	Subtitle = lipgloss.NewStyle().
		Foreground(TextMuted)

	Label = lipgloss.NewStyle().
		Foreground(TextDim)

	Highlight = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary)

	Muted = lipgloss.NewStyle().
		Foreground(TextMuted)

	Dim = lipgloss.NewStyle().
		Foreground(TextDim)

	Playing = lipgloss.NewStyle().
		Foreground(SpotifyGreen)

	Paused = lipgloss.NewStyle().
		Foreground(Warning)
)

// Border styles
var (
	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border)

	FocusedBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary)

	NoBorder = lipgloss.NewStyle().
		Border(lipgloss.HiddenBorder())
)

// Panel creates a styled panel with optional focus
func Panel(title string, focused bool) lipgloss.Style {
	style := BorderStyle.Padding(0, 1)

	if focused {
		style = FocusedBorder.Padding(0, 1)
	}

	return style
}

// PanelTitle creates a styled panel title
func PanelTitle(title string, focused bool) string {
	style := Label
	if focused {
		style = Highlight
	}
	return style.Render(" " + title + " ")
}

// ProgressBar creates a progress bar string
func ProgressBar(percent float64, width int) string {
	filled := int(percent / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	filledStyle := lipgloss.NewStyle().Foreground(Primary)
	emptyStyle := lipgloss.NewStyle().Foreground(Border)

	bar := filledStyle.Render(Repeat("━", filled)) +
		emptyStyle.Render(Repeat("─", width-filled))

	return bar
}

// StatusIcon returns an icon for playback status
func StatusIcon(playing bool) string {
	if playing {
		return Playing.Render("▶")
	}
	return Paused.Render("⏸")
}

// DeviceIcon returns an icon for device type
func DeviceIcon(deviceType string) string {
	switch deviceType {
	case "computer", "Computer":
		return "💻"
	case "smartphone", "Smartphone":
		return "📱"
	case "speaker", "Speaker":
		return "🔊"
	case "tv", "TV":
		return "📺"
	default:
		return "🎧"
	}
}

// Repeat repeats a string n times
func Repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
