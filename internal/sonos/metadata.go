package sonos

import (
	"encoding/xml"
	"html"
	"strings"

	"github.com/tessro/riff/internal/core"
)

// DIDLLite represents DIDL-Lite metadata format used by UPnP.
type DIDLLite struct {
	XMLName xml.Name   `xml:"DIDL-Lite"`
	Items   []DIDLItem `xml:"item"`
}

// DIDLItem represents a single item in DIDL-Lite metadata.
type DIDLItem struct {
	Title       string `xml:"title"`
	Creator     string `xml:"creator"`
	Album       string `xml:"album"`
	AlbumArtURI string `xml:"albumArtURI"`
	Class       string `xml:"class"`
	Res         string `xml:"res"`
}

// parseTrackMetadata parses Sonos track metadata into a core.Track.
func parseTrackMetadata(metadata, uri string) *core.Track {
	if metadata == "" {
		return nil
	}

	// Unescape HTML entities
	metadata = html.UnescapeString(metadata)

	var didl DIDLLite
	if err := xml.Unmarshal([]byte(metadata), &didl); err != nil {
		return nil
	}

	if len(didl.Items) == 0 {
		return nil
	}

	item := didl.Items[0]
	source := detectSource(uri)

	return &core.Track{
		URI:     uri,
		Title:   item.Title,
		Artist:  item.Creator,
		Artists: splitArtists(item.Creator),
		Album:   item.Album,
		Source:  source,
	}
}

// detectSource determines the source platform from a track URI.
func detectSource(uri string) core.Source {
	uri = strings.ToLower(uri)

	if strings.Contains(uri, "spotify") ||
		strings.Contains(uri, "x-sonos-spotify:") {
		return core.SourceSpotify
	}

	return core.SourceSonos
}

// IsSpotifySource returns true if the URI indicates Spotify content.
func IsSpotifySource(uri string) bool {
	return detectSource(uri) == core.SourceSpotify
}

// splitArtists splits a creator string into individual artists.
func splitArtists(creator string) []string {
	if creator == "" {
		return nil
	}

	// Handle common separators
	for _, sep := range []string{" & ", ", ", " feat. ", " ft. ", " featuring "} {
		if strings.Contains(creator, sep) {
			parts := strings.Split(creator, sep)
			var artists []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					artists = append(artists, p)
				}
			}
			return artists
		}
	}

	return []string{creator}
}

// ExtractSpotifyTrackID extracts a Spotify track ID from a Sonos URI.
func ExtractSpotifyTrackID(uri string) string {
	// Format: x-sonos-spotify:spotify:track:TRACKID?...
	uri = strings.TrimPrefix(uri, "x-sonos-spotify:")

	if strings.HasPrefix(uri, "spotify:track:") {
		parts := strings.SplitN(uri, "?", 2)
		return strings.TrimPrefix(parts[0], "spotify:track:")
	}

	return ""
}
