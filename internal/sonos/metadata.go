package sonos

import (
	"encoding/xml"
	"html"
	"regexp"
	"strings"

	"github.com/tessro/riff/internal/core"
)

// DIDLLite represents DIDL-Lite metadata format used by UPnP.
type DIDLLite struct {
	XMLName xml.Name   `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ DIDL-Lite"`
	Items   []DIDLItem `xml:"urn:schemas-upnp-org:metadata-1-0/DIDL-Lite/ item"`
}

// DIDLItem represents a single item in DIDL-Lite metadata.
type DIDLItem struct {
	// Dublin Core namespace elements
	Title   string `xml:"http://purl.org/dc/elements/1.1/ title"`
	Creator string `xml:"http://purl.org/dc/elements/1.1/ creator"`
	// UPnP namespace elements
	Album       string `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ album"`
	AlbumArtURI string `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ albumArtURI"`
	Class       string `xml:"urn:schemas-upnp-org:metadata-1-0/upnp/ class"`
	// Default namespace
	Res string `xml:"res"`
}

// parseTrackMetadata parses Sonos track metadata into a core.Track.
func parseTrackMetadata(metadata, uri string) *core.Track {
	if metadata == "" {
		return nil
	}

	// Unescape HTML entities
	metadata = html.UnescapeString(metadata)

	// Try namespace-aware parsing first
	var didl DIDLLite
	if err := xml.Unmarshal([]byte(metadata), &didl); err == nil && len(didl.Items) > 0 {
		item := didl.Items[0]
		if item.Title != "" {
			return &core.Track{
				URI:     uri,
				Title:   item.Title,
				Artist:  item.Creator,
				Artists: splitArtists(item.Creator),
				Album:   item.Album,
				Source:  detectSource(uri),
			}
		}
	}

	// Fallback: extract elements using regex (handles any namespace prefix)
	title := extractXMLElement(metadata, "title")
	creator := extractXMLElement(metadata, "creator")
	album := extractXMLElement(metadata, "album")

	if title == "" {
		return nil
	}

	return &core.Track{
		URI:     uri,
		Title:   title,
		Artist:  creator,
		Artists: splitArtists(creator),
		Album:   album,
		Source:  detectSource(uri),
	}
}

// extractXMLElement extracts content from an XML element, ignoring namespace prefixes.
func extractXMLElement(xml, localName string) string {
	// Match <prefix:localName>content</prefix:localName> or <localName>content</localName>
	re := regexp.MustCompile(`<(?:\w+:)?` + localName + `[^>]*>([^<]*)</(?:\w+:)?` + localName + `>`)
	matches := re.FindStringSubmatch(xml)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
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
