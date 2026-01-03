package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// GetCurrentUser returns the current user's profile.
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	var user User
	if err := c.Get(ctx, "/me", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetDevices returns the user's available playback devices.
func (c *Client) GetDevices(ctx context.Context) ([]Device, error) {
	var resp DevicesResponse
	if err := c.Get(ctx, "/me/player/devices", &resp); err != nil {
		return nil, err
	}
	return resp.Devices, nil
}

// GetPlaybackState returns the current playback state.
func (c *Client) GetPlaybackState(ctx context.Context) (*PlaybackState, error) {
	var state PlaybackState
	if err := c.Get(ctx, "/me/player", &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SearchType represents a type of Spotify content to search.
type SearchType string

const (
	SearchTypeTrack    SearchType = "track"
	SearchTypeArtist   SearchType = "artist"
	SearchTypeAlbum    SearchType = "album"
	SearchTypePlaylist SearchType = "playlist"
)

// SearchOptions configures a search query.
type SearchOptions struct {
	Query  string
	Types  []SearchType
	Limit  int
	Offset int
	Market string
}

// Search performs a search query.
func (c *Client) Search(ctx context.Context, opts SearchOptions) (*SearchResponse, error) {
	if opts.Query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	types := make([]string, len(opts.Types))
	for i, t := range opts.Types {
		types[i] = string(t)
	}
	if len(types) == 0 {
		types = []string{"track"} // Default to track search
	}

	params := map[string]string{
		"q":    opts.Query,
		"type": strings.Join(types, ","),
	}

	if opts.Limit > 0 {
		params["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Offset > 0 {
		params["offset"] = strconv.Itoa(opts.Offset)
	}
	if opts.Market != "" {
		params["market"] = opts.Market
	}

	var resp SearchResponse
	if err := c.Get(ctx, BuildURL("/search", params), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetRecentlyPlayed returns the user's recently played tracks.
func (c *Client) GetRecentlyPlayed(ctx context.Context, limit int) (*RecentlyPlayedResponse, error) {
	params := make(map[string]string)
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}

	var resp RecentlyPlayedResponse
	if err := c.Get(ctx, BuildURL("/me/player/recently-played", params), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
