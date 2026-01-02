package sonos

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// Client provides high-level access to Sonos devices.
type Client struct {
	discovery *Discovery
	soap      *SOAPClient
}

// NewClient creates a new Sonos client.
func NewClient() *Client {
	return &Client{
		discovery: NewDiscovery(0),
		soap:      NewSOAPClient(),
	}
}

// Discover finds all Sonos devices on the network.
func (c *Client) Discover(ctx context.Context) ([]*Device, error) {
	return c.discovery.Discover(ctx)
}

// GetDevice returns a device by identifier (UUID, name, IP, or alias).
func (c *Client) GetDevice(identifier string) *Device {
	return c.discovery.GetDevice(identifier)
}

// SetAlias maps an alias to a device.
func (c *Client) SetAlias(alias, target string) {
	c.discovery.SetAlias(alias, target)
}

// DeviceInfo contains detailed device information.
type DeviceInfo struct {
	RoomName     string `xml:"RoomName"`
	ModelName    string `xml:"ModelName"`
	ModelNumber  string `xml:"ModelNumber"`
	SerialNumber string `xml:"SerialNumber"`
	SoftwareVersion string `xml:"SoftwareVersion"`
}

// GetDeviceInfo retrieves device information.
func (c *Client) GetDeviceInfo(ctx context.Context, device *Device) (*DeviceInfo, error) {
	resp, err := c.soap.Call(ctx, device.IP, device.Port, DevicePropertiesEndpoint, DevicePropertiesService, "GetZoneAttributes", nil)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Body struct {
			Response struct {
				CurrentZoneName string `xml:"CurrentZoneName"`
			} `xml:"GetZoneAttributesResponse"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &DeviceInfo{
		RoomName: envelope.Body.Response.CurrentZoneName,
	}, nil
}

// TransportInfo contains playback transport state.
type TransportInfo struct {
	CurrentTransportState  string
	CurrentTransportStatus string
	CurrentSpeed           string
}

// GetTransportInfo retrieves the current transport state.
func (c *Client) GetTransportInfo(ctx context.Context, device *Device) (*TransportInfo, error) {
	args := map[string]string{"InstanceID": "0"}
	resp, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "GetTransportInfo", args)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Body struct {
			Response TransportInfo `xml:"GetTransportInfoResponse"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &envelope.Body.Response, nil
}

// PositionInfo contains track position information.
type PositionInfo struct {
	Track         int    `xml:"Track"`
	TrackDuration string `xml:"TrackDuration"`
	TrackMetaData string `xml:"TrackMetaData"`
	TrackURI      string `xml:"TrackURI"`
	RelTime       string `xml:"RelTime"`
	AbsTime       string `xml:"AbsTime"`
}

// GetPositionInfo retrieves the current track position.
func (c *Client) GetPositionInfo(ctx context.Context, device *Device) (*PositionInfo, error) {
	args := map[string]string{"InstanceID": "0"}
	resp, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "GetPositionInfo", args)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Body struct {
			Response PositionInfo `xml:"GetPositionInfoResponse"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &envelope.Body.Response, nil
}

// MediaInfo contains current media information.
type MediaInfo struct {
	NrTracks           int    `xml:"NrTracks"`
	MediaDuration      string `xml:"MediaDuration"`
	CurrentURI         string `xml:"CurrentURI"`
	CurrentURIMetaData string `xml:"CurrentURIMetaData"`
}

// GetMediaInfo retrieves current media information.
func (c *Client) GetMediaInfo(ctx context.Context, device *Device) (*MediaInfo, error) {
	args := map[string]string{"InstanceID": "0"}
	resp, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "GetMediaInfo", args)
	if err != nil {
		return nil, err
	}

	var envelope struct {
		Body struct {
			Response MediaInfo `xml:"GetMediaInfoResponse"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(resp, &envelope); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &envelope.Body.Response, nil
}

// GetVolume retrieves the current volume level (0-100).
func (c *Client) GetVolume(ctx context.Context, device *Device) (int, error) {
	args := map[string]string{
		"InstanceID": "0",
		"Channel":    "Master",
	}
	resp, err := c.soap.Call(ctx, device.IP, device.Port, RenderingControlEndpoint, RenderingControlService, "GetVolume", args)
	if err != nil {
		return 0, err
	}

	var envelope struct {
		Body struct {
			Response struct {
				CurrentVolume string `xml:"CurrentVolume"`
			} `xml:"GetVolumeResponse"`
		} `xml:"Body"`
	}
	if err := xml.Unmarshal(resp, &envelope); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}

	vol, _ := strconv.Atoi(envelope.Body.Response.CurrentVolume)
	return vol, nil
}

// SetVolume sets the volume level (0-100).
func (c *Client) SetVolume(ctx context.Context, device *Device, volume int) error {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}

	args := map[string]string{
		"InstanceID":    "0",
		"Channel":       "Master",
		"DesiredVolume": strconv.Itoa(volume),
	}
	_, err := c.soap.Call(ctx, device.IP, device.Port, RenderingControlEndpoint, RenderingControlService, "SetVolume", args)
	return err
}

// Play starts playback.
func (c *Client) Play(ctx context.Context, device *Device) error {
	args := map[string]string{
		"InstanceID": "0",
		"Speed":      "1",
	}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "Play", args)
	return err
}

// Pause pauses playback.
func (c *Client) Pause(ctx context.Context, device *Device) error {
	args := map[string]string{"InstanceID": "0"}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "Pause", args)
	return err
}

// Next skips to the next track.
func (c *Client) Next(ctx context.Context, device *Device) error {
	args := map[string]string{"InstanceID": "0"}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "Next", args)
	return err
}

// Previous skips to the previous track.
func (c *Client) Previous(ctx context.Context, device *Device) error {
	args := map[string]string{"InstanceID": "0"}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "Previous", args)
	return err
}

// Seek seeks to a position in the current track.
func (c *Client) Seek(ctx context.Context, device *Device, target string) error {
	args := map[string]string{
		"InstanceID": "0",
		"Unit":       "REL_TIME",
		"Target":     target,
	}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "Seek", args)
	return err
}

// IsPlaying returns true if the device is currently playing.
func (c *Client) IsPlaying(ctx context.Context, device *Device) (bool, error) {
	info, err := c.GetTransportInfo(ctx, device)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(info.CurrentTransportState, "PLAYING"), nil
}

// AddURIToQueue adds a URI to the playback queue.
func (c *Client) AddURIToQueue(ctx context.Context, device *Device, uri, metadata string) error {
	args := map[string]string{
		"InstanceID":                      "0",
		"EnqueuedURI":                     uri,
		"EnqueuedURIMetaData":             metadata,
		"DesiredFirstTrackNumberEnqueued": "0",
		"EnqueueAsNext":                   "0",
	}
	_, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "AddURIToQueue", args)
	return err
}

// PlayURI sets the transport URI and starts playback.
func (c *Client) PlayURI(ctx context.Context, device *Device, uri, metadata string) error {
	args := map[string]string{
		"InstanceID":         "0",
		"CurrentURI":         uri,
		"CurrentURIMetaData": metadata,
	}
	if _, err := c.soap.Call(ctx, device.IP, device.Port, AVTransportEndpoint, AVTransportService, "SetAVTransportURI", args); err != nil {
		return fmt.Errorf("set transport URI: %w", err)
	}
	return c.Play(ctx, device)
}
