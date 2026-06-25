package studio

import "strings"

// StyleDraftWriter persists a single per-element style override to the host
// draft store. The host implementation writes (componentKey, property, value,
// breakpoint, state) through its own draft -> publish pipeline.
type StyleDraftWriter func(componentKey, property, value, breakpoint, state string) error

// SectionFieldWriter persists a per-instance home-section field value to the
// host draft store. The host implementation writes (sectionKey, field, value)
// through its own draft -> publish pipeline.
type SectionFieldWriter func(sectionKey, field, value string) error

// AppearanceWriter receives the filtered, allowedKeys-gated form map from
// ApplySaveAppearance and persists it to the host draft store. It returns
// host-level field errors (nil map == success) and any fatal error.
type AppearanceWriter func(rawForm map[string]string) (fieldErrors map[string]string, err error)

// ApplySetStyle is the host-agnostic body for the set-style authoring
// operation. It re-validates the style property against IsSupportedStyleProperty
// and re-sanitizes the value (defense in depth — the host writer must never
// trust the client), then delegates to write.
//
// Returns an AuthoringMutationResult with FieldErrors set on validation
// failure, or a success result with RefreshPreview=true on success.
func ApplySetStyle(m AuthoringMutation, write StyleDraftWriter) (AuthoringMutationResult, error) {
	if strings.TrimSpace(m.ComponentKey) == "" {
		return AuthoringMutationResult{
			Message:     "Choose an element to style.",
			FieldErrors: map[string]string{AuthoringFieldComponentKey: "Choose a component to style."},
			Values:      m.FormValues(),
		}, nil
	}
	if !IsSupportedStyleProperty(m.StyleProperty) {
		return AuthoringMutationResult{
			Message:     "That style property is not supported.",
			FieldErrors: map[string]string{AuthoringFieldStyleProperty: "This style property is not supported."},
			Values:      m.FormValues(),
		}, nil
	}
	if reason := styleValueRejection(m.StyleValue); reason != "" {
		return AuthoringMutationResult{
			Message:     "That style value is not allowed.",
			FieldErrors: map[string]string{AuthoringFieldStyleValue: reason},
			Values:      m.FormValues(),
		}, nil
	}
	if err := write(m.ComponentKey, m.StyleProperty, m.StyleValue, m.Breakpoint, m.State); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Style updated.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "style:" + m.ComponentKey + ":" + m.StyleProperty,
			Label:     m.StyleProperty,
			Summary:   "Set " + m.StyleProperty + " to " + m.StyleValue,
			Kind:      "style",
			Component: m.ComponentKey,
		}},
	}, nil
}

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

// ApplySaveAppearance is the host-agnostic body for the save-appearance
// authoring operation. It filters m.RawForm to the keys present in
// allowedKeys (the host's curated appearance form-key list), then delegates to
// write. Keys absent from RawForm are not forwarded, so the host's formValue
// fallback preserves existing draft values for all other appearance fields.
//
// Returns a validation-failure result (no write) when the filtered form is
// empty. Otherwise returns the write's field errors or a success result with
// RefreshPreview=true.
func ApplySaveAppearance(m AuthoringMutation, allowedKeys []string, write AppearanceWriter) (AuthoringMutationResult, error) {
	form := make(map[string]string, len(allowedKeys))
	for _, key := range allowedKeys {
		if value, ok := m.RawForm[key]; ok {
			form[key] = value
		}
	}
	if len(form) == 0 {
		return AuthoringMutationResult{
			Message: "No appearance values were submitted.",
			FieldErrors: map[string]string{
				AuthoringFieldOperation: "save-appearance requires at least one color or font field.",
			},
			Values: m.FormValues(),
		}, nil
	}
	fieldErrors, err := write(form)
	if err != nil {
		return AuthoringMutationResult{}, err
	}
	if len(fieldErrors) > 0 {
		return AuthoringMutationResult{
			Message:     "Please correct the highlighted fields.",
			FieldErrors: fieldErrors,
			Values:      m.FormValues(),
		}, nil
	}
	return AuthoringMutationResult{
		Message:        "Appearance saved.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:     "appearance",
			Label:   "Brand appearance",
			Summary: "Updated brand colors and fonts.",
			Kind:    "appearance",
		}},
	}, nil
}
