package authoring

import "strings"

// StyleDraftWriter persists a single per-element style override to the host
// draft store. The host implementation writes (componentKey, property, value,
// breakpoint, state) through its own draft -> publish pipeline.
type StyleDraftWriter func(componentKey, property, value, breakpoint, state string) error

// StyleDraftResetter removes one explicit override. The host should preserve
// inherited values and only delete the breakpoint/state/property entry.
type StyleDraftResetter func(componentKey, property, breakpoint, state string) error

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
	if IsResponsiveLayoutProperty(m.StyleProperty) {
		if reason, ok := ValidateLayoutValue(m.StyleProperty, m.StyleValue); !ok {
			return AuthoringMutationResult{
				Message:     "That layout value is not allowed.",
				FieldErrors: map[string]string{AuthoringFieldStyleValue: reason},
				Values:      m.FormValues(),
			}, nil
		}
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

// ApplyResetStyle is the host-agnostic reset path. It never serializes the
// inherited value as a new override.
func ApplyResetStyle(m AuthoringMutation, reset StyleDraftResetter) (AuthoringMutationResult, error) {
	if strings.TrimSpace(m.ComponentKey) == "" || !IsSupportedStyleProperty(m.StyleProperty) {
		return AuthoringMutationResult{Message: "Choose a supported element style.", FieldErrors: map[string]string{AuthoringFieldStyleProperty: "This style property is not supported."}, Values: m.FormValues()}, nil
	}
	if err := reset(m.ComponentKey, m.StyleProperty, normalizeStyleBreakpoint(m.Breakpoint), normalizeStyleState(m.State)); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{Message: "Style reset.", PreviewURL: "/", RefreshPreview: true, Values: m.FormValues(), Changes: []AuthoringChange{{Key: "style:" + m.ComponentKey + ":" + m.StyleProperty, Label: m.StyleProperty, Summary: "Reset " + m.StyleProperty, Kind: "style", Component: m.ComponentKey}}}, nil
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
