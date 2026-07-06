package studio

import "strings"

// SectionFieldWriter persists a per-instance home-section field value to the
// host draft store. The host implementation writes (sectionKey, field, value)
// through its own draft -> publish pipeline.
type SectionFieldWriter func(sectionKey, field, value string) error

// ApplySaveSectionField is the host-agnostic body for the section-field
// sub-path of the save-control authoring operation. It parses the section key
// and field name from m's Binding via ParseSectionFieldBinding, then delegates
// to write. The caller is responsible for routing to this function only when
// the binding matches a section-field pattern.
//
// labelForField produces the human-facing label for the parsed field (used in
// the success Message and the AuthoringChange). The host passes its own
// title-caser; a nil func or empty result falls back to the raw field key.
//
// Returns an AuthoringMutationResult with FieldErrors set on write failure, or
// a success result with RefreshPreview=true on success.
func ApplySaveSectionField(m AuthoringMutation, write SectionFieldWriter, labelForField func(field string) string) (AuthoringMutationResult, error) {
	sectionKey, field, ok := ParseSectionFieldBinding(m.Binding)
	if !ok {
		return AuthoringMutationResult{
			Message: "That section field could not be saved.",
			FieldErrors: map[string]string{
				AuthoringFieldBinding: "Binding does not match a section field pattern.",
			},
			Values: m.FormValues(),
		}, nil
	}
	if err := write(sectionKey, field, m.Value); err != nil {
		return AuthoringMutationResult{
			Message:     "That section field could not be saved.",
			FieldErrors: map[string]string{AuthoringFieldValue: "Could not save this section field."},
			Values:      m.FormValues(),
		}, nil
	}
	label := field
	if labelForField != nil {
		if titled := strings.TrimSpace(labelForField(field)); titled != "" {
			label = titled
		}
	}
	return AuthoringMutationResult{
		Message:        label + " saved.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "section-field-" + sectionKey + "-" + field,
			Label:     label,
			Summary:   "Updated the " + sectionKey + " section's " + field + ".",
			Kind:      "control",
			Component: sectionKey,
			Binding:   m.Binding,
		}},
	}, nil
}
