package core

// DeviceType indicates the kind of playback device.
type DeviceType string

const (
	DeviceTypeSpeaker   DeviceType = "speaker"
	DeviceTypeComputer  DeviceType = "computer"
	DeviceTypePhone     DeviceType = "phone"
	DeviceTypeTV        DeviceType = "tv"
	DeviceTypeSoundbar  DeviceType = "soundbar"
)

// Platform indicates the device's platform.
type Platform string

const (
	PlatformSpotify Platform = "spotify"
	PlatformSonos   Platform = "sonos"
)

// Device represents a playback device.
type Device struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Type     DeviceType `json:"type"`
	Platform Platform   `json:"platform"`
	IsActive bool       `json:"is_active"`
	Account  string     `json:"account"`
}
