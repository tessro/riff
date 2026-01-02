package client

// User represents a Spotify user profile.
type User struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Country     string   `json:"country"`
	Product     string   `json:"product"`
	Type        string   `json:"type"`
	URI         string   `json:"uri"`
	Href        string   `json:"href"`
	Images      []Image  `json:"images"`
	Followers   Followers `json:"followers"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

// Image represents an image resource.
type Image struct {
	URL    string `json:"url"`
	Height int    `json:"height"`
	Width  int    `json:"width"`
}

// Followers represents follower information.
type Followers struct {
	Total int `json:"total"`
}

// ExternalURLs contains external URLs for a resource.
type ExternalURLs struct {
	Spotify string `json:"spotify"`
}

// Device represents a Spotify playback device.
type Device struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	IsActive       bool   `json:"is_active"`
	IsRestricted   bool   `json:"is_restricted"`
	IsPrivateSession bool `json:"is_private_session"`
	VolumePercent  *int   `json:"volume_percent"` // Nullable
	SupportsVolume bool   `json:"supports_volume"`
}

// DevicesResponse is the response from the devices endpoint.
type DevicesResponse struct {
	Devices []Device `json:"devices"`
}

// PlaybackState represents the current playback state.
type PlaybackState struct {
	Device               Device  `json:"device"`
	ShuffleState         bool    `json:"shuffle_state"`
	RepeatState          string  `json:"repeat_state"` // off, track, context
	Timestamp            int64   `json:"timestamp"`
	ProgressMS           int     `json:"progress_ms"`
	IsPlaying            bool    `json:"is_playing"`
	Item                 *Track  `json:"item"`
	CurrentlyPlayingType string  `json:"currently_playing_type"` // track, episode, ad, unknown
	Context              *Context `json:"context"`
}

// Track represents a Spotify track.
type Track struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	URI         string       `json:"uri"`
	Href        string       `json:"href"`
	DurationMS  int          `json:"duration_ms"`
	Explicit    bool         `json:"explicit"`
	IsPlayable  bool         `json:"is_playable"`
	TrackNumber int          `json:"track_number"`
	DiscNumber  int          `json:"disc_number"`
	Popularity  int          `json:"popularity"`
	Artists     []Artist     `json:"artists"`
	Album       Album        `json:"album"`
	ExternalURLs ExternalURLs `json:"external_urls"`
	ExternalIDs ExternalIDs  `json:"external_ids"`
}

// Artist represents a Spotify artist.
type Artist struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	URI          string       `json:"uri"`
	Href         string       `json:"href"`
	Type         string       `json:"type"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

// Album represents a Spotify album.
type Album struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	URI          string       `json:"uri"`
	Href         string       `json:"href"`
	AlbumType    string       `json:"album_type"`
	TotalTracks  int          `json:"total_tracks"`
	ReleaseDate  string       `json:"release_date"`
	Images       []Image      `json:"images"`
	Artists      []Artist     `json:"artists"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

// Context represents a playback context (album, artist, playlist).
type Context struct {
	Type         string       `json:"type"`
	URI          string       `json:"uri"`
	Href         string       `json:"href"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

// ExternalIDs contains external identifiers.
type ExternalIDs struct {
	ISRC string `json:"isrc"`
	EAN  string `json:"ean"`
	UPC  string `json:"upc"`
}

// SearchResponse represents the response from a search query.
type SearchResponse struct {
	Tracks    *SearchTracks    `json:"tracks"`
	Artists   *SearchArtists   `json:"artists"`
	Albums    *SearchAlbums    `json:"albums"`
	Playlists *SearchPlaylists `json:"playlists"`
}

// SearchTracks contains track search results.
type SearchTracks struct {
	Items  []Track `json:"items"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
	Href   string  `json:"href"`
	Next   string  `json:"next"`
}

// SearchArtists contains artist search results.
type SearchArtists struct {
	Items  []Artist `json:"items"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Href   string   `json:"href"`
	Next   string   `json:"next"`
}

// SearchAlbums contains album search results.
type SearchAlbums struct {
	Items  []Album `json:"items"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
	Href   string  `json:"href"`
	Next   string  `json:"next"`
}

// SearchPlaylists contains playlist search results.
type SearchPlaylists struct {
	Items  []Playlist `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
	Href   string     `json:"href"`
	Next   string     `json:"next"`
}

// Playlist represents a Spotify playlist.
type Playlist struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	URI          string       `json:"uri"`
	Href         string       `json:"href"`
	Description  string       `json:"description"`
	Public       bool         `json:"public"`
	Collaborative bool        `json:"collaborative"`
	Images       []Image      `json:"images"`
	Owner        User         `json:"owner"`
	ExternalURLs ExternalURLs `json:"external_urls"`
	Tracks       struct {
		Total int    `json:"total"`
		Href  string `json:"href"`
	} `json:"tracks"`
}

// Queue represents the user's playback queue.
type Queue struct {
	CurrentlyPlaying *Track  `json:"currently_playing"`
	Queue            []Track `json:"queue"`
}
