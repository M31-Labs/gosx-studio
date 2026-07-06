package core

import (
	"fmt"
	"strconv"
	"strings"
)

// FirstNonEmpty returns the first value that is non-blank after trimming
// whitespace, or "" if every value is blank. It is the single canonical home
// for this helper: every layer (core, authoring, panels, shell, and
// gosx-cms/studio via the root facade) uses it instead of maintaining local
// duplicates.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// NormalizeKey lowercases value and replaces underscores/spaces with hyphens,
// producing a stable attribute/lookup key.
func NormalizeKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

// BoolAttr renders a bool as the string "true"/"false" for HTML attribute
// output.
func BoolAttr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

// FmtAny renders any value as a trimmed string via fmt.Sprint.
func FmtAny(value any) string {
	return strings.TrimSpace(fmt.Sprint(value))
}

// CountLabel renders a count with a singular/plural noun, e.g. "1 page" or
// "3 pages".
func CountLabel(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + plural
}

// WorkspaceCoord formats a float64 workspace coordinate for CSS positioning,
// trimming trailing zeros and a bare "-0" down to "0".
func WorkspaceCoord(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" || text == "-0" {
		return "0"
	}
	return text
}

// WorkspaceToken slugs value down to lowercase alphanumerics separated by
// single hyphens, for use in composition-workspace node/link keys and DOM
// input IDs. Returns "node" if value has no alphanumeric characters.
func WorkspaceToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "node"
	}
	return token
}
