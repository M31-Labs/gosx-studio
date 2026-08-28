package authoring

import (
	"sort"
	"strings"
)

// LayoutValueOption is one server-approved value presented by the responsive
// layout inspector. Value is persisted as CSS only after ValidateLayoutValue
// accepts the property/value pair.
type LayoutValueOption struct {
	Value string
	Label string
}

// LayoutControl describes a curated box-model or layout property. The browser
// receives this finite vocabulary; it never receives an arbitrary CSS editor.
type LayoutControl struct {
	Property string
	Label    string
	Group    string
	Options  []LayoutValueOption
}

var responsiveLayoutControls = []LayoutControl{
	{Property: "display", Label: "Layout mode", Group: "Layout", Options: layoutOptions("block", "Block", "flex", "Stack / flex", "grid", "Grid")},
	{Property: "flex-direction", Label: "Direction", Group: "Layout", Options: layoutOptions("row", "Horizontal", "column", "Vertical")},
	{Property: "grid-template-columns", Label: "Columns", Group: "Layout", Options: layoutOptions("minmax(0,1fr)", "One", "repeat(2,minmax(0,1fr))", "Two", "repeat(3,minmax(0,1fr))", "Three", "repeat(4,minmax(0,1fr))", "Four")},
	{Property: "align-items", Label: "Align", Group: "Alignment", Options: layoutOptions("stretch", "Stretch", "flex-start", "Start", "center", "Center", "flex-end", "End")},
	{Property: "justify-content", Label: "Distribute", Group: "Alignment", Options: layoutOptions("flex-start", "Start", "center", "Center", "flex-end", "End", "space-between", "Space between")},
	{Property: "flex-wrap", Label: "Wrap", Group: "Alignment", Options: layoutOptions("nowrap", "No wrap", "wrap", "Wrap")},
	{Property: "gap", Label: "Gap", Group: "Spacing", Options: spacingOptions(true)},
	{Property: "padding", Label: "Padding", Group: "Spacing", Options: spacingOptions(true)},
	{Property: "padding-block", Label: "Vertical padding", Group: "Spacing", Options: spacingOptions(true)},
	{Property: "padding-inline", Label: "Horizontal padding", Group: "Spacing", Options: spacingOptions(true)},
	{Property: "width", Label: "Width", Group: "Size", Options: layoutOptions("auto", "Auto", "100%", "Full", "min(100%,48rem)", "Content", "min(100%,72rem)", "Wide")},
	{Property: "max-width", Label: "Maximum width", Group: "Size", Options: layoutOptions("none", "None", "32rem", "Small", "48rem", "Content", "72rem", "Wide")},
}

func layoutOptions(values ...string) []LayoutValueOption {
	out := make([]LayoutValueOption, 0, len(values)/2)
	for i := 0; i+1 < len(values); i += 2 {
		out = append(out, LayoutValueOption{Value: values[i], Label: values[i+1]})
	}
	return out
}

func spacingOptions(includeZero bool) []LayoutValueOption {
	values := []string{"var(--space-xs)", "XS", "var(--space-sm)", "Small", "var(--space-md)", "Medium", "var(--space-lg)", "Large", "var(--space-xl)", "XL"}
	if includeZero {
		values = append([]string{"0", "None"}, values...)
	}
	return layoutOptions(values...)
}

// ResponsiveLayoutControls returns a copy of the ordered inspector schema.
func ResponsiveLayoutControls() []LayoutControl {
	out := make([]LayoutControl, len(responsiveLayoutControls))
	for i, control := range responsiveLayoutControls {
		out[i] = control
		out[i].Options = append([]LayoutValueOption(nil), control.Options...)
	}
	return out
}

// IsResponsiveLayoutProperty reports whether a style property belongs to the
// typed responsive layout subset.
func IsResponsiveLayoutProperty(property string) bool {
	_, ok := responsiveLayoutControl(property)
	return ok
}

// ValidateLayoutValue rejects arbitrary CSS, negative spacing and values that
// do not belong to the exact property vocabulary.
func ValidateLayoutValue(property, value string) (string, bool) {
	control, ok := responsiveLayoutControl(property)
	if !ok {
		return "This is not a responsive layout property.", false
	}
	value = strings.TrimSpace(value)
	for _, option := range control.Options {
		if value == option.Value {
			return "", true
		}
	}
	return "Choose an approved " + strings.ToLower(control.Label) + " value.", false
}

func responsiveLayoutControl(property string) (LayoutControl, bool) {
	property = strings.ToLower(strings.TrimSpace(property))
	for _, control := range responsiveLayoutControls {
		if control.Property == property {
			return control, true
		}
	}
	return LayoutControl{}, false
}

// ResponsiveLayoutProperties returns stable property names for adapters and
// conformance tests.
func ResponsiveLayoutProperties() []string {
	properties := make([]string, 0, len(responsiveLayoutControls))
	for _, control := range responsiveLayoutControls {
		properties = append(properties, control.Property)
	}
	sort.Strings(properties)
	return properties
}
