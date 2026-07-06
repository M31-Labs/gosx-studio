package studio

import (
	"strings"
)

// FieldNamePrefix turns route, resource, or field keys into the stable
// PascalCase fragments used by generated Studio form ids and names.
func FieldNamePrefix(parts ...string) string {
	out := strings.Builder{}
	for _, part := range parts {
		for _, segment := range strings.FieldsFunc(part, idSeparator) {
			segment = strings.TrimSpace(segment)
			if segment == "" {
				continue
			}
			out.WriteString(strings.ToUpper(segment[:1]))
			if len(segment) > 1 {
				out.WriteString(segment[1:])
			}
		}
	}
	return out.String()
}

// DOMID joins arbitrary Studio keys into a deterministic, lower-case DOM id.
func DOMID(prefix string, parts ...string) string {
	out := strings.Builder{}
	out.WriteString(strings.TrimSpace(prefix))
	for _, part := range parts {
		safe := DOMIDPart(part)
		if safe == "" {
			continue
		}
		if out.Len() > 0 {
			out.WriteByte('-')
		}
		out.WriteString(safe)
	}
	return out.String()
}

func DOMIDPart(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	out := strings.Builder{}
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			lastDash = false
			continue
		}
		if out.Len() > 0 && !lastDash {
			out.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(out.String(), "-")
}

func idSeparator(r rune) bool {
	return r == '-' || r == '_' || r == '.' || r == ' '
}
