// Package mediaurl owns the narrow URL contract shared by generic CMS
// serializers. It validates destinations without fetching or probing them.
package mediaurl

import (
	"net/url"
	"strings"
	"unicode"
)

// Destination identifies the HTML attribute that will receive a URL.
//
// Image destinations accept only HTTP(S) and relative references. Link
// destinations additionally accept mailto: and tel:. The distinction keeps a
// contact action useful while ensuring an image never receives a non-image
// navigation scheme.
type Destination uint8

const (
	DestinationImage Destination = iota
	DestinationLink
)

// ForImage validates a URL intended for an image src attribute.
func ForImage(raw string) (string, bool) {
	return Sanitize(raw, DestinationImage)
}

// ForLink validates a URL intended for a link href attribute.
func ForLink(raw string) (string, bool) {
	return Sanitize(raw, DestinationLink)
}

// Sanitize returns the trimmed, original URL when it is safe for the given
// destination, or an empty string otherwise.
//
// The contract deliberately permits only http:, https:, and relative
// references. Relative references may be rooted (/media/a.jpg), dot-relative
// (./media/a.jpg or ../media/a.jpg), or ordinary path/query/fragment
// references. Protocol-relative //host references are not accepted because
// they bypass an explicit scheme decision. mailto: and tel: are link-only.
// Control characters, whitespace inside a URL, backslashes, malformed URLs,
// active schemes, and unknown schemes are rejected. This function performs no
// network request and does not claim to validate the bytes at a destination.
func Sanitize(raw string, destination Destination) (string, bool) {
	value, ok := normalize(raw)
	if !ok {
		return "", false
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", false
	}

	if parsed.Scheme != "" {
		scheme := strings.ToLower(parsed.Scheme)
		switch scheme {
		case "http", "https":
			// Require the conventional authority form. url.Parse accepts
			// opaque forms such as http:asset, which are ambiguous here.
			if !strings.HasPrefix(strings.ToLower(value), scheme+"://") || parsed.Host == "" || parsed.Hostname() == "" || parsed.Opaque != "" {
				return "", false
			}
			return value, true
		case "mailto", "tel":
			if destination != DestinationLink || parsed.Opaque == "" || parsed.Host != "" {
				return "", false
			}
			return value, true
		default:
			return "", false
		}
	}

	// Any leading // is authority-like (including ///host/path), even when
	// net/url leaves the latter's Host field empty. It is intentionally outside
	// the explicit contract.
	if strings.HasPrefix(value, "//") || parsed.Host != "" {
		return "", false
	}
	if parsed.Path == "" && parsed.RawQuery == "" && parsed.Fragment == "" {
		return "", false
	}
	return value, true
}

func normalize(raw string) (string, bool) {
	if raw == "" || hasControl(raw) {
		return "", false
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	for _, r := range value {
		if unicode.IsSpace(r) || r == '\\' {
			return "", false
		}
	}
	return value, true
}

func hasControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) || r == 0x7f {
			return true
		}
	}
	return false
}
