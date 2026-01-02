package components

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/tessro/riff/internal/core"
	"github.com/tessro/riff/internal/tui/styles"
)

// Devices displays available playback devices
type Devices struct {
	selected int
}

// NewDevices creates a new Devices component
func NewDevices() *Devices {
	return &Devices{selected: 0}
}

// SelectNext selects the next device
func (d *Devices) SelectNext() {
	d.selected++
}

// SelectPrev selects the previous device
func (d *Devices) SelectPrev() {
	if d.selected > 0 {
		d.selected--
	}
}

// Selected returns the selected device index
func (d *Devices) Selected() int {
	return d.selected
}

// Render renders the devices panel
func (d *Devices) Render(devices []core.Device, width, height int, focused bool) string {
	title := styles.PanelTitle("Devices", focused)

	var content string
	if len(devices) == 0 {
		content = styles.Muted.Render("No devices found")
	} else {
		content = d.renderDevices(devices, width-4, height-4, focused)
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

func (d *Devices) renderDevices(devices []core.Device, width, maxLines int, focused bool) string {
	// Adjust selected if out of bounds
	if d.selected >= len(devices) {
		d.selected = len(devices) - 1
	}
	if d.selected < 0 {
		d.selected = 0
	}

	lines := make([]string, 0, len(devices))

	for i, device := range devices {
		icon := styles.DeviceIcon(string(device.Type))

		// Selection indicator
		selector := "  "
		if focused && i == d.selected {
			selector = "▸ "
		}

		// Active indicator
		active := ""
		if device.IsActive {
			active = styles.Playing.Render(" ●")
		}

		// Device name
		name := device.Name
		if i == d.selected && focused {
			name = styles.Highlight.Render(name)
		}

		line := fmt.Sprintf("%s%s %s%s", selector, icon, name, active)
		lines = append(lines, line)

		// Limit lines
		if len(lines) >= maxLines {
			break
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
