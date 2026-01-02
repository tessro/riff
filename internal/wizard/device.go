package wizard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tessro/riff/internal/core"
)

// DeviceModel is the bubbletea model for the device picker.
type DeviceModel struct {
	devices  []core.Device
	cursor   int
	selected *core.Device
	width    int
	height   int
}

// Styles for device picker
var (
	deviceTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	deviceItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	deviceSelectedStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Background(lipgloss.Color("237"))

	deviceActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	deviceInactiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	deviceTypeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)

// NewDeviceModel creates a new device picker model.
func NewDeviceModel(devices []core.Device) DeviceModel {
	return DeviceModel{
		devices: devices,
		width:   80,
		height:  20,
	}
}

// Init initializes the model.
func (m DeviceModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m DeviceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit

		case "enter", " ":
			if len(m.devices) > 0 && m.cursor < len(m.devices) {
				m.selected = &m.devices[m.cursor]
				return m, tea.Quit
			}

		case "up", "k", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j", "ctrl+n":
			if m.cursor < len(m.devices)-1 {
				m.cursor++
			}

		case "home", "g":
			m.cursor = 0

		case "end", "G":
			m.cursor = len(m.devices) - 1
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// View renders the model.
func (m DeviceModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(deviceTitleStyle.Render("📱 Select Device"))
	b.WriteString("\n\n")

	if len(m.devices) == 0 {
		b.WriteString(deviceInactiveStyle.Render("No devices found"))
		b.WriteString("\n\n")
		b.WriteString(deviceTypeStyle.Render("Make sure Spotify is open on a device or Sonos speakers are on the network."))
	} else {
		for i, device := range m.devices {
			// Build device line
			var line strings.Builder

			// Status indicator
			if device.IsActive {
				line.WriteString(deviceActiveStyle.Render("● "))
			} else {
				line.WriteString(deviceInactiveStyle.Render("○ "))
			}

			// Device name
			line.WriteString(device.Name)

			// Device type and platform
			typeInfo := " " + deviceTypeStyle.Render("("+string(device.Type)+", "+string(device.Platform)+")")
			line.WriteString(typeInfo)

			// Account info if available
			if device.Account != "" {
				line.WriteString(deviceTypeStyle.Render(" - " + device.Account))
			}

			// Render with selection style
			if i == m.cursor {
				b.WriteString(deviceSelectedStyle.Render("▸ " + line.String()))
			} else {
				b.WriteString(deviceItemStyle.Render("  " + line.String()))
			}
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString("\n")
	b.WriteString(deviceTypeStyle.Render("↑/↓ navigate • enter select • esc quit"))
	b.WriteString("\n")
	b.WriteString(deviceTypeStyle.Render("● active  ○ inactive"))

	return b.String()
}

// Selected returns the selected device, or nil if none.
func (m DeviceModel) Selected() *core.Device {
	return m.selected
}

// RunDevicePicker runs the device picker and returns the selected device.
func RunDevicePicker(devices []core.Device) (*core.Device, error) {
	model := NewDeviceModel(devices)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	return finalModel.(DeviceModel).Selected(), nil
}
