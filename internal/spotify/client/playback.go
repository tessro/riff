package client

import (
	"context"
	"strconv"
)

// PlayOptions configures a play request.
type PlayOptions struct {
	ContextURI string      `json:"context_uri,omitempty"`
	URIs       []string    `json:"uris,omitempty"`
	Offset     *PlayOffset `json:"offset,omitempty"`
	PositionMS int         `json:"position_ms,omitempty"`
}

// PlayOffset specifies where to start playback in a context.
type PlayOffset struct {
	Position int    `json:"position,omitempty"` // Track index
	URI      string `json:"uri,omitempty"`      // Track URI
}

// Play starts or resumes playback.
// If opts is nil, resumes current playback.
// If deviceID is empty, uses the currently active device.
func (c *Client) Play(ctx context.Context, deviceID string, opts *PlayOptions) error {
	path := "/me/player/play"
	if deviceID != "" {
		path = BuildURL(path, map[string]string{"device_id": deviceID})
	}
	// Spotify requires a JSON body even for resume - send empty object if no options
	body := opts
	if body == nil {
		body = &PlayOptions{}
	}
	return c.Put(ctx, path, body, nil)
}

// Pause pauses playback.
func (c *Client) Pause(ctx context.Context, deviceID string) error {
	path := "/me/player/pause"
	if deviceID != "" {
		path = BuildURL(path, map[string]string{"device_id": deviceID})
	}
	return c.Put(ctx, path, nil, nil)
}

// Next skips to the next track.
func (c *Client) Next(ctx context.Context, deviceID string) error {
	path := "/me/player/next"
	if deviceID != "" {
		path = BuildURL(path, map[string]string{"device_id": deviceID})
	}
	return c.Post(ctx, path, nil, nil)
}

// Previous skips to the previous track.
func (c *Client) Previous(ctx context.Context, deviceID string) error {
	path := "/me/player/previous"
	if deviceID != "" {
		path = BuildURL(path, map[string]string{"device_id": deviceID})
	}
	return c.Post(ctx, path, nil, nil)
}

// Seek seeks to a position in the current track.
func (c *Client) Seek(ctx context.Context, positionMs int, deviceID string) error {
	params := map[string]string{
		"position_ms": strconv.Itoa(positionMs),
	}
	if deviceID != "" {
		params["device_id"] = deviceID
	}
	return c.Put(ctx, BuildURL("/me/player/seek", params), nil, nil)
}

// SetVolume sets the playback volume (0-100).
func (c *Client) SetVolume(ctx context.Context, percent int, deviceID string) error {
	params := map[string]string{
		"volume_percent": strconv.Itoa(percent),
	}
	if deviceID != "" {
		params["device_id"] = deviceID
	}
	return c.Put(ctx, BuildURL("/me/player/volume", params), nil, nil)
}

// SetRepeat sets the repeat mode (off, track, context).
func (c *Client) SetRepeat(ctx context.Context, state string, deviceID string) error {
	params := map[string]string{
		"state": state,
	}
	if deviceID != "" {
		params["device_id"] = deviceID
	}
	return c.Put(ctx, BuildURL("/me/player/repeat", params), nil, nil)
}

// SetShuffle sets the shuffle mode.
func (c *Client) SetShuffle(ctx context.Context, state bool, deviceID string) error {
	params := map[string]string{
		"state": strconv.FormatBool(state),
	}
	if deviceID != "" {
		params["device_id"] = deviceID
	}
	return c.Put(ctx, BuildURL("/me/player/shuffle", params), nil, nil)
}

// GetQueue returns the user's playback queue.
func (c *Client) GetQueue(ctx context.Context) (*Queue, error) {
	var queue Queue
	if err := c.Get(ctx, "/me/player/queue", &queue); err != nil {
		return nil, err
	}
	return &queue, nil
}

// AddToQueue adds a track to the playback queue.
func (c *Client) AddToQueue(ctx context.Context, uri string, deviceID string) error {
	params := map[string]string{
		"uri": uri,
	}
	if deviceID != "" {
		params["device_id"] = deviceID
	}
	return c.Post(ctx, BuildURL("/me/player/queue", params), nil, nil)
}

// TransferPlayback transfers playback to a different device.
func (c *Client) TransferPlayback(ctx context.Context, deviceID string, play bool) error {
	body := map[string]interface{}{
		"device_ids": []string{deviceID},
		"play":       play,
	}
	return c.Put(ctx, "/me/player", body, nil)
}
