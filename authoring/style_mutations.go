package authoring

import (
	"sort"
	"strings"

	"m31labs.dev/gosx-studio/core"
)

// Responsive breakpoints for per-element styling, aligned with the editor's
// device toggles in the GoSX Studio design: desktop is the unprefixed base,
// tablet and mobile are max-width overrides (a desktop-first cascade).
const (
	StyleBreakpointBase   = "base"
	StyleBreakpointTablet = "tablet"
	StyleBreakpointMobile = "mobile"
)

// Element states a style override can target.
const (
	StyleStateDefault  = "default"
	StyleStateHover    = "hover"
	StyleStateFocus    = "focus"
	StyleStateActive   = "active"
	StyleStateDisabled = "disabled"
)

const maxStyleValueLength = 512

// supportedStyleProperties is the curated, structured-first allow-list of CSS
// properties the visual editor may set per element. It deliberately excludes
// free-form/raw CSS: every property here is one the inspector or canvas overlay
// exposes through a typed control. Keeping the set curated keeps authored pages
// out of div-soup and lets the server — not the browser — own what a legal edit
// is. The set covers the GoSX Studio inspector's Style section (alignment ->
// text-align, section background -> background-color, text color -> color,
// vertical padding -> padding-block, heading size -> font-size) plus the broader
// box/flex/typography/fill/effect vocabulary needed for Webflow-class parity.
var supportedStyleProperties = map[string]struct{}{
	// Box model
	"display": {}, "width": {}, "min-width": {}, "max-width": {},
	"height": {}, "min-height": {}, "max-height": {},
	"margin": {}, "margin-top": {}, "margin-right": {}, "margin-bottom": {}, "margin-left": {},
	"margin-block": {}, "margin-inline": {},
	"padding": {}, "padding-top": {}, "padding-right": {}, "padding-bottom": {}, "padding-left": {},
	"padding-block": {}, "padding-inline": {},
	"gap": {},
	// Flex / layout
	"flex-direction": {}, "justify-content": {}, "align-items": {}, "flex-wrap": {},
	"grid-template-columns": {},
	// Typography
	"color": {}, "font-size": {}, "font-weight": {}, "font-family": {},
	"line-height": {}, "letter-spacing": {}, "text-align": {}, "text-transform": {}, "text-decoration": {},
	// Fills / borders
	"background-color": {}, "background": {},
	"border-color": {}, "border-width": {}, "border-style": {}, "border-radius": {},
	// Effects / position
	"box-shadow": {}, "opacity": {},
	"position": {}, "top": {}, "right": {}, "bottom": {}, "left": {}, "z-index": {},
}

// SupportedStyleProperties returns the curated style properties the editor may
// set, sorted for stable enumeration (e.g. an inspector property menu).
func SupportedStyleProperties() []string {
	out := make([]string, 0, len(supportedStyleProperties))
	for property := range supportedStyleProperties {
		out = append(out, property)
	}
	sort.Strings(out)
	return out
}

// IsSupportedStyleProperty reports whether property is in the curated allow-list.
func IsSupportedStyleProperty(property string) bool {
	_, ok := supportedStyleProperties[strings.ToLower(strings.TrimSpace(property))]
	return ok
}

// SupportedStyleBreakpoints returns the legal breakpoint values in cascade order.
func SupportedStyleBreakpoints() []string {
	return []string{StyleBreakpointBase, StyleBreakpointTablet, StyleBreakpointMobile}
}

// SupportedStyleStates returns the legal element-state values.
func SupportedStyleStates() []string {
	return []string{StyleStateDefault, StyleStateHover, StyleStateFocus, StyleStateActive, StyleStateDisabled}
}

// normalizeStyleBreakpoint defaults an empty breakpoint to base and lowercases.
// Unknown values are left intact so Validate can reject them with a clear error
// rather than silently coercing (which would hide author mistakes).
func normalizeStyleBreakpoint(breakpoint string) string {
	breakpoint = strings.ToLower(strings.TrimSpace(breakpoint))
	if breakpoint == "" {
		return StyleBreakpointBase
	}
	return breakpoint
}

// normalizeStyleState defaults an empty state to default and lowercases.
func normalizeStyleState(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" {
		return StyleStateDefault
	}
	return state
}

func isSupportedStyleBreakpoint(breakpoint string) bool {
	switch breakpoint {
	case StyleBreakpointBase, StyleBreakpointTablet, StyleBreakpointMobile:
		return true
	default:
		return false
	}
}

func isSupportedStyleState(state string) bool {
	switch state {
	case StyleStateDefault, StyleStateHover, StyleStateFocus, StyleStateActive, StyleStateDisabled:
		return true
	default:
		return false
	}
}

// AuthoringMutationForStyle builds a set-style mutation that sets one CSS
// property on a component for the given breakpoint and element state. An empty
// breakpoint or state defaults to base / default.
func AuthoringMutationForStyle(page core.Page, component core.Component, property, value, breakpoint, state string) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationSetStyle,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
		Binding:        component.Binding,
		StyleProperty:  property,
		StyleValue:     value,
		Breakpoint:     breakpoint,
		State:          state,
	}.Normalize()
}

// styleValueRejection returns a non-empty reason when value is unsafe to persist
// as a single CSS declaration value. It rejects declaration/selector breakouts
// (`;`, `{`, `}`) and known CSS/markup injection vectors. The browser is never
// trusted to pre-validate — this is the server-owned guard.
func styleValueRejection(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Enter a value."
	}
	if len(value) > maxStyleValueLength {
		return "Value is too long."
	}
	if strings.ContainsAny(value, ";{}<>\\\n\r") {
		return "Value contains characters that are not allowed."
	}
	lower := strings.ToLower(value)
	for _, bad := range []string{"javascript:", "expression(", "url(javascript", "</", "<!--"} {
		if strings.Contains(lower, bad) {
			return "Value contains a disallowed pattern."
		}
	}
	return ""
}

// validateStyleMutation enforces the set-style contract: a real page+component
// target, a supported property, a safe value, and supported breakpoint/state.
func validateStyleMutation(validation *AuthoringValidation, mutation AuthoringMutation) {
	validateStyleTarget(validation, mutation)
	if mutation.StyleProperty == "" {
		validation.AddFieldError(AuthoringFieldStyleProperty, "Choose a style property.")
	} else if !IsSupportedStyleProperty(mutation.StyleProperty) {
		validation.AddFieldError(AuthoringFieldStyleProperty, "This style property is not supported.")
	}
	if reason := styleValueRejection(mutation.StyleValue); reason != "" {
		validation.AddFieldError(AuthoringFieldStyleValue, reason)
	} else if IsResponsiveLayoutProperty(mutation.StyleProperty) {
		if reason, ok := ValidateLayoutValue(mutation.StyleProperty, mutation.StyleValue); !ok {
			validation.AddFieldError(AuthoringFieldStyleValue, reason)
		}
	}
	if !isSupportedStyleBreakpoint(normalizeStyleBreakpoint(mutation.Breakpoint)) {
		validation.AddFieldError(AuthoringFieldBreakpoint, "Choose a supported breakpoint.")
	}
	if !isSupportedStyleState(normalizeStyleState(mutation.State)) {
		validation.AddFieldError(AuthoringFieldState, "Choose a supported state.")
	}
}

// validateStyleTarget validates the address shared by set/reset style. Reset
// intentionally does not validate a value because its meaning is to remove an
// explicit override and reveal the inherited cascade value.
func validateStyleTarget(validation *AuthoringValidation, mutation AuthoringMutation) {
	requirePageComponent(validation, mutation)
	if mutation.StyleProperty == "" {
		validation.AddFieldError(AuthoringFieldStyleProperty, "Choose a style property.")
	} else if !IsSupportedStyleProperty(mutation.StyleProperty) {
		validation.AddFieldError(AuthoringFieldStyleProperty, "This style property is not supported.")
	}
	if !isSupportedStyleBreakpoint(normalizeStyleBreakpoint(mutation.Breakpoint)) {
		validation.AddFieldError(AuthoringFieldBreakpoint, "Choose a supported breakpoint.")
	}
	if !isSupportedStyleState(normalizeStyleState(mutation.State)) {
		validation.AddFieldError(AuthoringFieldState, "Choose a supported state.")
	}
}
