package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
)

// DirectEditPanelOptions describes one selected canvas target. Hosts pass
// server-canonical identity and values; browser state is presentation only.
type DirectEditPanelOptions struct {
	FormID, Action, CSRFToken string
	Target                    authoring.OperationTarget
	Value                     string
	DocumentRevision          uint64
	TargetHead                string
	DurableHistory            bool
	Breakpoint                string
	StyleProperty             string
	StyleValue                string
	InheritedFrom             string
	UndoOperationID           string
	RedoOperationID           string
}

// RenderDirectEditPanel renders the compact selected-target editor. It keeps
// Save/Cancel explicit and emits operation metadata as hidden values so the
// operation runtime can submit exactly one accepted edit.
func RenderDirectEditPanel(options DirectEditPanelOptions) gosx.Node {
	if options.FormID == "" {
		options.FormID = "studio-direct-edit"
	}
	if options.Action == "" {
		options.Action = "/admin/editor/__actions/authoring"
	}
	if options.Breakpoint == "" {
		options.Breakpoint = authoring.StyleBreakpointBase
	}
	target := options.Target.Normalize()
	attrs := []any{gosx.Attr("id", options.FormID), gosx.Attr("method", "post"), gosx.Attr("action", options.Action), gosx.Attr("data-gosx-studio-durable-history", boolAttr(options.DurableHistory)), gosx.Attr("data-gosx-studio-operation-action", options.Action), gosx.Attr("data-studio-document-revision", options.DocumentRevision), gosx.Attr("data-studio-target-head", options.TargetHead)}
	attrs = append(attrs, gosx.Attr("data-studio-target-route", target.Route), gosx.Attr("data-studio-target-page-id", target.PageID), gosx.Attr("data-studio-target-field", target.Field), gosx.Attr("data-studio-target-component", target.ComponentKey))
	children := []gosx.Node{
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", authoring.AuthoringFieldOperation), gosx.Attr("value", string(authoring.AuthoringOperationSetField)))),
		hidden(authoring.AuthoringFieldPageRoute, target.Route), hidden(authoring.AuthoringFieldPageKey, target.PageID), hidden(authoring.AuthoringFieldComponentKey, target.ComponentKey), hidden(authoring.AuthoringFieldBinding, target.Field), hidden(authoring.AuthoringFieldStyleProperty, options.StyleProperty), hidden(authoring.AuthoringFieldBreakpoint, options.Breakpoint), hidden(authoring.AuthoringFieldState, target.State), hidden(authoring.AuthoringFieldExpectedRevision, formatUint(options.DocumentRevision)), hidden(authoring.AuthoringFieldExpectedTargetHead, options.TargetHead),
	}
	if options.CSRFToken != "" {
		children = append(children, hidden("csrf_token", options.CSRFToken))
	}
	children = append(children,
		gosx.El("label", gosx.Attrs(gosx.Attr("data-studio-direct-edit-field", "true")), gosx.El("span", nil, gosx.Text("Selected content")), gosx.El("textarea", gosx.Attrs(gosx.Attr("name", authoring.AuthoringFieldValue), gosx.Attr("data-studio-operation-value", "true")), gosx.Text(options.Value))),
		gosx.El("div", gosx.Attrs(gosx.Attr("data-studio-style-readout", "true")), gosx.Text(styleReadout(options))),
		styleOperationButtons(target),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationSetField))), gosx.Text("Save")),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "reset"), gosx.Attr("data-gosx-studio-operation-cancel", "true")), gosx.Text("Cancel")),
		historyControls(options),
	)
	return gosx.El("form", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func historyControls(options DirectEditPanelOptions) gosx.Node {
	undoAttrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationUndo)), gosx.Attr("data-gosx-studio-history-operation-id", options.UndoOperationID)}
	redoAttrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationRedo)), gosx.Attr("data-gosx-studio-history-operation-id", options.RedoOperationID)}
	if options.UndoOperationID == "" {
		undoAttrs = append(undoAttrs, gosx.Attr("disabled", "disabled"))
	}
	if options.RedoOperationID == "" {
		redoAttrs = append(redoAttrs, gosx.Attr("disabled", "disabled"))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-studio-durable-history", "true")), gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-durable-history-status", "true")), gosx.Text("Durable history")), gosx.El("button", gosx.Attrs(undoAttrs...), gosx.Text("Undo")), gosx.El("button", gosx.Attrs(redoAttrs...), gosx.Text("Redo")))
}

func styleOperationButtons(target authoring.OperationTarget) gosx.Node {
	component := target.ComponentKey
	if component == "" {
		if target.Field == "pages.about.title" {
			component = "about:content"
		} else {
			component = "home:hero"
		}
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("data-studio-direct-style-controls", "true")),
		gosx.El("legend", nil, gosx.Text("Responsive style")),
		styleButton(component, "background-color", "var(--color-accent)", "base", "Accent · Base"),
		styleButton(component, "padding-block", "var(--space-lg)", "mobile", "Roomy · Mobile"),
		styleButton(component, "flex-direction", "column", "mobile", "Stack · Mobile"),
		styleResetButton(component, "padding-block", "mobile", "Reset mobile padding"),
	)
}
func styleButton(component, property, value, breakpoint, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationSetStyle)), gosx.Attr("data-gosx-studio-operation-value", value), gosx.Attr("data-gosx-studio-style-property", property), gosx.Attr("data-gosx-studio-component", component), gosx.Attr("data-gosx-studio-breakpoint", breakpoint)), gosx.Text(label))
}
func styleResetButton(component, property, breakpoint, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-operation-kind", string(authoring.AuthoringOperationResetStyle)), gosx.Attr("data-gosx-studio-operation-value", ""), gosx.Attr("data-gosx-studio-style-property", property), gosx.Attr("data-gosx-studio-component", component), gosx.Attr("data-gosx-studio-breakpoint", breakpoint)), gosx.Text(label))
}

func hidden(name, value string) gosx.Node {
	return gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", name), gosx.Attr("value", value)))
}
func boolAttr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
func formatUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	const digits = "0123456789"
	out := make([]byte, 0, 20)
	for v > 0 {
		out = append([]byte{digits[v%10]}, out...)
		v /= 10
	}
	return string(out)
}
func styleReadout(options DirectEditPanelOptions) string {
	if options.InheritedFrom != "" {
		return "Inherited from " + options.InheritedFrom
	}
	if options.StyleValue != "" {
		return "Override · " + options.StyleValue
	}
	return "No explicit override"
}
