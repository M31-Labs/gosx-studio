package studio

import (
	"sort"
	"strings"
)

// StyleOverrides is the generic per-element style-override carrier:
//
//	componentKey -> "breakpoint/state" -> cssProperty -> value
//
// It matches a host's per-component override map exactly (e.g. muddy
// cms.SiteSettings.ComponentStyles), so a host converts with a single cast.
// Values are produced by the set-style grammar (style_mutations.go), which
// validates the property against the curated allow-list and sanitizes the value
// before it ever reaches here.
type StyleOverrides map[string]map[string]map[string]string

// RenderComponentStylesCSS renders per-element overrides as scoped
// [data-gosx-component="<key>"] CSS, intended to be layered on top of the host's
// brand CSS. tablet/mobile breakpoints wrap in desktop-first max-width media
// queries; non-default element states add a pseudo-class. Emission is direct
// (values are already sanitized at the set-style boundary).
func RenderComponentStylesCSS(overrides StyleOverrides) string {
	if len(overrides) == 0 {
		return ""
	}
	var b strings.Builder
	for _, componentKey := range sortedOverrideKeys(overrides) {
		scopes := overrides[componentKey]
		for _, scope := range sortedOverrideKeys(scopes) {
			props := scopes[scope]
			if len(props) == 0 {
				continue
			}
			breakpoint, state := splitOverrideScope(scope)
			selector := `[data-gosx-component="` + cssEscapeOverrideAttr(componentKey) + `"]` + overrideStatePseudo(state)

			var decls strings.Builder
			for _, property := range sortedOverrideKeys(props) {
				decls.WriteString(property)
				decls.WriteString(":")
				decls.WriteString(props[property])
				decls.WriteString(";")
			}
			rule := selector + "{" + decls.String() + "}"
			if media := overrideBreakpointMedia(breakpoint); media != "" {
				b.WriteString(media)
				b.WriteString("{")
				b.WriteString(rule)
				b.WriteString("}")
				continue
			}
			b.WriteString(rule)
		}
	}
	return b.String()
}

func sortedOverrideKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// splitOverrideScope splits a "breakpoint/state" scope key, defaulting a
// malformed scope to base/default. The vocabulary matches the set-style grammar
// constants (StyleBreakpoint*/StyleState*) so the renderer and writer cannot drift.
func splitOverrideScope(scope string) (breakpoint, state string) {
	if i := strings.IndexByte(scope, '/'); i >= 0 {
		breakpoint = scope[:i]
		state = scope[i+1:]
	}
	if breakpoint == "" {
		breakpoint = StyleBreakpointBase
	}
	if state == "" {
		state = StyleStateDefault
	}
	return breakpoint, state
}

func overrideBreakpointMedia(breakpoint string) string {
	switch breakpoint {
	case StyleBreakpointTablet:
		return "@media (max-width:63.99rem)"
	case StyleBreakpointMobile:
		return "@media (max-width:47.99rem)"
	default:
		return ""
	}
}

func overrideStatePseudo(state string) string {
	switch state {
	case StyleStateHover:
		return ":hover"
	case StyleStateFocus:
		return ":focus"
	case StyleStateActive:
		return ":active"
	case StyleStateDisabled:
		return ":disabled"
	default:
		return ""
	}
}

func cssEscapeOverrideAttr(value string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value)
}
