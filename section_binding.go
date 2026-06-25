package studio

import "strings"

// ParseSectionFieldBinding extracts (sectionKey, field) from a control binding
// that targets a per-instance section field, e.g.
// "home.section.pagehead.title" -> ("pagehead", "title"). Any page prefix before
// "section." is tolerated, so it is page-model agnostic. Returns ok=false otherwise.
//
// This is the host-agnostic router both editor adapters use to send per-instance
// block-field edits (the inline-edit / save-control path) to a section's value
// store instead of a global settings field.
func ParseSectionFieldBinding(binding string) (sectionKey, field string, ok bool) {
	const marker = "section."
	idx := strings.Index(binding, marker)
	if idx < 0 {
		return "", "", false
	}
	parts := strings.SplitN(binding[idx+len(marker):], ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	sectionKey = strings.TrimSpace(parts[0])
	field = strings.TrimSpace(parts[1])
	if sectionKey == "" || field == "" {
		return "", "", false
	}
	return sectionKey, field, true
}

// SectionFieldBinding builds the binding string for a per-instance section field.
// It is the inverse of ParseSectionFieldBinding.
func SectionFieldBinding(sectionKey, field string) string {
	return "home.section." + sectionKey + "." + field
}

// StyleFormIDPart reduces a string to a safe HTML id fragment (used to build
// stable per-control form ids in the inspector).
func StyleFormIDPart(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '-'
		}
	}, s)
}
