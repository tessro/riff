package wizard

import (
	"os"

	"github.com/tess/riff/internal/core"
	"golang.org/x/term"
)

// Interactive provides interactive fallback functionality.
type Interactive struct {
	enabled    bool
	searchFunc SearchFunc
	devices    []core.Device
}

// NewInteractive creates a new interactive handler.
func NewInteractive() *Interactive {
	return &Interactive{
		enabled: true,
	}
}

// SetEnabled enables or disables interactive mode.
func (i *Interactive) SetEnabled(enabled bool) {
	i.enabled = enabled
}

// SetSearchFunc sets the search function for the search wizard.
func (i *Interactive) SetSearchFunc(fn SearchFunc) {
	i.searchFunc = fn
}

// SetDevices sets the available devices for the device picker.
func (i *Interactive) SetDevices(devices []core.Device) {
	i.devices = devices
}

// IsTerminal returns true if stdout is a terminal.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// CanInteract returns true if interactive mode is available.
func (i *Interactive) CanInteract() bool {
	return i.enabled && IsTerminal()
}

// PromptSearch launches the search wizard if interactive mode is available.
// Returns the selected result, or nil if cancelled or not interactive.
func (i *Interactive) PromptSearch() (*SearchResult, error) {
	if !i.CanInteract() || i.searchFunc == nil {
		return nil, nil
	}
	return RunSearch(i.searchFunc)
}

// PromptDevice launches the device picker if interactive mode is available.
// Returns the selected device, or nil if cancelled or not interactive.
func (i *Interactive) PromptDevice() (*core.Device, error) {
	if !i.CanInteract() || len(i.devices) == 0 {
		return nil, nil
	}
	return RunDevicePicker(i.devices)
}

// NeedsTrack returns true if a track argument is required but missing.
func NeedsTrack(args []string) bool {
	return len(args) == 0
}

// NeedsDevice returns true if a device argument is required but missing.
func NeedsDevice(deviceFlag string, devices []core.Device) bool {
	if deviceFlag != "" {
		return false
	}
	// Check if there's exactly one active device
	activeCount := 0
	for _, d := range devices {
		if d.IsActive {
			activeCount++
		}
	}
	// Need to prompt if no active device or multiple active devices
	return activeCount != 1
}

// GetActiveDevice returns the single active device if there is exactly one.
func GetActiveDevice(devices []core.Device) *core.Device {
	var active *core.Device
	count := 0
	for i := range devices {
		if devices[i].IsActive {
			active = &devices[i]
			count++
		}
	}
	if count == 1 {
		return active
	}
	return nil
}
