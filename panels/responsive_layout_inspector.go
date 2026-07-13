package panels

import (
	"strconv"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

// ResponsiveLayoutValue carries one explicit persisted override and its
// target-local durability cursors. Presence is explicit so reset means delete,
// never "write the inherited value again".
type ResponsiveLayoutValue struct {
	Present         bool
	Value           string
	TargetHead      string
	UndoOperationID string
	RedoOperationID string
}

// ResponsiveLayoutInspectorOptions describes the selected canvas component.
// Values are keyed by "breakpoint/property".
type ResponsiveLayoutInspectorOptions struct {
	ID, Action, CSRFToken string
	Target                authoring.OperationTarget
	DocumentRevision      uint64
	Values                map[string]ResponsiveLayoutValue
}

// RenderResponsiveLayoutInspector renders real property controls for the
// selected component. Each property/scope owns a separate durable form so its
// head and Undo/Redo chain cannot corrupt the selected content target.
//
// Wave 3A (inspector-presentation): the breakpoint controls now live behind
// one closed-by-default <details> disclosure (fix (c) — collapsed-by-default
// sections). The "Responsive layout" heading and a one-line summary (target +
// override count) stay outside the disclosure so the panel is still
// scannable collapsed. A host that composes more than one target's inspector
// on the same page (muddy-noni-commerce's editorDirectEditPanels always
// renders both home:hero's and about:content's panel together, regardless of
// which one is actually selected — the respLayoutCount=2 defect) is expected
// to hide every instance except the one matching the live canvas selection;
// hostruntime/assets/workbench_runtime.js's scopeResponsiveLayoutInspectors
// is the studio-side idempotence guard that does this at runtime by toggling
// [hidden] (see that file), so double-inclusion still renders once even if
// the host keeps composing every target unconditionally.
func RenderResponsiveLayoutInspector(options ResponsiveLayoutInspectorOptions) gosx.Node {
	if options.ID == "" {
		options.ID = "studio-responsive-layout"
	}
	if options.Action == "" {
		options.Action = "/admin/editor/__actions/authoring"
	}
	target := options.Target.Normalize()
	breakpointNodes := make([]gosx.Node, 0, len(authoring.SupportedStyleBreakpoints()))
	for _, breakpoint := range authoring.SupportedStyleBreakpoints() {
		controls := make([]gosx.Node, 0, len(authoring.ResponsiveLayoutControls()))
		for _, control := range authoring.ResponsiveLayoutControls() {
			controls = append(controls, renderResponsiveLayoutControl(options, target, breakpoint, control))
		}
		attrs := []any{gosx.Attr("data-studio-layout-breakpoint", breakpoint)}
		if breakpoint == authoring.StyleBreakpointBase {
			attrs = append(attrs, gosx.Attr("open", "open"))
		}
		breakpointNodes = append(breakpointNodes, gosx.El("details", gosx.Attrs(attrs...),
			gosx.El("summary", nil, gosx.Text(layoutBreakpointLabel(breakpoint))),
			gosx.Fragment(controls...),
		))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("id", options.ID),
		gosx.Attr("data-studio-responsive-layout-inspector", "true"),
		gosx.Attr("data-studio-selected-route", target.Route),
		gosx.Attr("data-studio-selected-component", target.ComponentKey),
	),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-responsive-layout__head")),
			gosx.El("h3", nil, gosx.Text("Responsive layout")),
			gosx.El("span", gosx.Attrs(
				gosx.Attr("class", "studio-responsive-layout__summary"),
				gosx.Attr("data-studio-layout-summary", "true"),
			), gosx.Text(responsiveLayoutSummaryLabel(target, responsiveLayoutOverrideCount(options.Values)))),
		),
		gosx.El("details", gosx.Attrs(gosx.Attr("class", "studio-responsive-layout__disclosure")),
			gosx.El("summary", nil, gosx.Text("Edit layout")),
			gosx.El("p", nil, gosx.Text("Edit the selected canvas object with safe layout tokens. Smaller devices inherit until you add an override.")),
			gosx.Fragment(breakpointNodes...),
		),
	)
}

func responsiveLayoutOverrideCount(values map[string]ResponsiveLayoutValue) int {
	count := 0
	for _, value := range values {
		if value.Present {
			count++
		}
	}
	return count
}

func responsiveLayoutSummaryLabel(target authoring.OperationTarget, overrideCount int) string {
	label := core.FirstNonEmpty(target.ComponentKey, "Selected object")
	switch overrideCount {
	case 0:
		return label + " · No overrides"
	case 1:
		return label + " · 1 override"
	default:
		return label + " · " + strconv.Itoa(overrideCount) + " overrides"
	}
}

func renderResponsiveLayoutControl(options ResponsiveLayoutInspectorOptions, target authoring.OperationTarget, breakpoint string, control authoring.LayoutControl) gosx.Node {
	key := breakpoint + "/" + control.Property
	state := options.Values[key]
	effective, inheritedFrom := responsiveLayoutEffective(options.Values, breakpoint, control)
	selectID := options.ID + "-" + authoring.StyleFormIDPart(target.ComponentKey+"-"+breakpoint+"-"+control.Property)
	propertyTarget := target
	propertyTarget.Field = ""
	propertyTarget.Property = control.Property
	propertyTarget.Breakpoint = breakpoint
	propertyTarget.State = authoring.StyleStateDefault
	optionNodes := make([]gosx.Node, 0, len(control.Options))
	for _, option := range control.Options {
		attrs := []any{gosx.Attr("value", option.Value)}
		if option.Value == effective {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(option.Label)))
	}
	formAttrs := []any{
		gosx.Attr("method", "post"), gosx.Attr("action", options.Action),
		gosx.Attr("data-gosx-studio-durable-history", "true"),
		gosx.Attr("data-gosx-studio-operation-action", options.Action),
		gosx.Attr("data-studio-layout-control", control.Property),
		gosx.Attr("data-studio-document-revision", options.DocumentRevision),
		gosx.Attr("data-studio-target-head", state.TargetHead),
		gosx.Attr("data-studio-target-route", propertyTarget.Route),
		gosx.Attr("data-studio-target-page-id", propertyTarget.PageID),
		gosx.Attr("data-studio-target-field", ""),
		gosx.Attr("data-studio-target-component", propertyTarget.ComponentKey),
	}
	children := []gosx.Node{
		hidden(authoring.AuthoringFieldOperation, string(authoring.AuthoringOperationSetStyle)),
		hidden(authoring.AuthoringFieldPageRoute, propertyTarget.Route), hidden(authoring.AuthoringFieldPageKey, propertyTarget.PageID),
		hidden(authoring.AuthoringFieldComponentKey, propertyTarget.ComponentKey), hidden(authoring.AuthoringFieldBinding, ""),
		hidden(authoring.AuthoringFieldStyleProperty, control.Property), hidden(authoring.AuthoringFieldBreakpoint, breakpoint),
		hidden(authoring.AuthoringFieldState, authoring.StyleStateDefault), hidden(authoring.AuthoringFieldExpectedRevision, formatUint(options.DocumentRevision)),
		hidden(authoring.AuthoringFieldExpectedTargetHead, state.TargetHead),
	}
	if options.CSRFToken != "" {
		children = append(children, hidden("csrf_token", options.CSRFToken))
	}
	readout := "Explicit override"
	if !state.Present {
		if inheritedFrom == "" {
			readout = "Default token"
		} else {
			readout = "Inherited from " + layoutBreakpointLabel(inheritedFrom)
		}
	}
	children = append(children,
		gosx.El("label", gosx.Attrs(gosx.Attr("for", selectID)), gosx.Text(control.Label)),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", selectID), gosx.Attr("name", authoring.AuthoringFieldStyleValue), gosx.Attr("data-studio-layout-value", "true")), gosx.Fragment(optionNodes...)),
		gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-layout-presence", boolAttr(state.Present))), gosx.Text(readout)),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationSetStyle)), gosx.Attr("data-gosx-studio-operation-value-source", "#"+selectID), gosx.Attr("data-gosx-studio-style-property", control.Property), gosx.Attr("data-gosx-studio-component", propertyTarget.ComponentKey), gosx.Attr("data-gosx-studio-breakpoint", breakpoint), gosx.Attr("data-gosx-studio-style-state", authoring.StyleStateDefault)), gosx.Text("Apply")),
		layoutResetButton(propertyTarget.ComponentKey, control.Property, breakpoint, !state.Present),
		layoutHistoryControls(state, propertyTarget),
	)
	return gosx.El("form", gosx.Attrs(formAttrs...), gosx.Fragment(children...))
}

func responsiveLayoutEffective(values map[string]ResponsiveLayoutValue, breakpoint string, control authoring.LayoutControl) (value, inheritedFrom string) {
	if current := values[breakpoint+"/"+control.Property]; current.Present {
		return current.Value, ""
	}
	if breakpoint == authoring.StyleBreakpointMobile {
		if tablet := values[authoring.StyleBreakpointTablet+"/"+control.Property]; tablet.Present {
			return tablet.Value, authoring.StyleBreakpointTablet
		}
	}
	if breakpoint != authoring.StyleBreakpointBase {
		if base := values[authoring.StyleBreakpointBase+"/"+control.Property]; base.Present {
			return base.Value, authoring.StyleBreakpointBase
		}
	}
	if len(control.Options) > 0 {
		return control.Options[0].Value, ""
	}
	return "", ""
}

func layoutResetButton(component, property, breakpoint string, disabled bool) gosx.Node {
	attrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationResetStyle)), gosx.Attr("data-gosx-studio-style-property", property), gosx.Attr("data-gosx-studio-component", component), gosx.Attr("data-gosx-studio-breakpoint", breakpoint), gosx.Attr("data-gosx-studio-style-state", authoring.StyleStateDefault)}
	if disabled {
		attrs = append(attrs, gosx.Attr("disabled", "disabled"))
	}
	return gosx.El("button", gosx.Attrs(attrs...), gosx.Text("Reset to inherited"))
}

func layoutHistoryControls(state ResponsiveLayoutValue, target authoring.OperationTarget) gosx.Node {
	targetAttrs := []any{gosx.Attr("data-gosx-studio-style-property", target.Property), gosx.Attr("data-gosx-studio-component", target.ComponentKey), gosx.Attr("data-gosx-studio-breakpoint", target.Breakpoint), gosx.Attr("data-gosx-studio-style-state", target.State)}
	undo := append([]any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationUndo)), gosx.Attr("data-gosx-studio-history-operation-id", state.UndoOperationID)}, targetAttrs...)
	redo := append([]any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationRedo)), gosx.Attr("data-gosx-studio-history-operation-id", state.RedoOperationID)}, targetAttrs...)
	if state.UndoOperationID == "" {
		undo = append(undo, gosx.Attr("disabled", "disabled"))
	}
	if state.RedoOperationID == "" {
		redo = append(redo, gosx.Attr("disabled", "disabled"))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-studio-layout-history", "true")), gosx.El("button", gosx.Attrs(undo...), gosx.Text("Undo layout")), gosx.El("button", gosx.Attrs(redo...), gosx.Text("Redo layout")))
}

func layoutBreakpointLabel(breakpoint string) string {
	if breakpoint == "" {
		return "Base"
	}
	return strings.ToUpper(breakpoint[:1]) + breakpoint[1:]
}
