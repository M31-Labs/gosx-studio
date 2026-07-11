package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

// This file extends the flow designer (flow_designer.go/flow_designer_panel.go)
// with durable editing of one action's fields/validation/binding, plus the
// isolated test-execution path (authoring.ApplyTestFlowAction). It is a
// companion panel rather than a change to flow_designer_panel.go's existing
// render functions: that panel's own tests assert it never renders a <form>
// element (every field there binds to one page-level shared form via
// form="..."), and multiple distinct submittable field/action rows sharing
// one external form would collide on hidden input names the moment more than
// one row can actually submit. This panel owns its own self-contained forms
// instead — the same choice panels/interactions_panel.go makes, and for the
// same reason (see its header comment).

// FlowFieldRow is one editable field on a flow action.
type FlowFieldRow struct {
	Name     string
	Label    string
	Kind     string
	Required bool
}

// FlowTestFieldValue is one test-value input rendered for the isolated
// test-execution form, one per field currently on the action.
type FlowTestFieldValue struct {
	Name  string
	Label string
	Value string
	Error string
}

// FlowFieldEditorOptions describes one flow action's durable editing surface.
type FlowFieldEditorOptions struct {
	ID, Action, CSRFToken  string
	FlowKey, FlowLabel     string
	ActionKey, ActionLabel string
	Binding                string
	BindingStatusLabel     string
	BindingReason          string
	Fields                 []FlowFieldRow
	TestValues             []FlowTestFieldValue
	TestMessage            string
}

// RenderFlowFieldEditorPanel renders the durable field/validation/action/binding
// editor plus the isolated test-execution form for one flow action.
func RenderFlowFieldEditorPanel(options FlowFieldEditorOptions) gosx.Node {
	if options.ID == "" {
		options.ID = "studio-flow-field-editor"
	}
	if options.Action == "" {
		options.Action = "/admin/editor/__actions/authoring"
	}
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Fields, validation, and binding")),
			gosx.El("p", nil, gosx.Text("Edit "+firstNonEmptyPanelValue(options.ActionLabel, options.ActionKey)+" on "+firstNonEmptyPanelValue(options.FlowLabel, options.FlowKey)+".")),
		),
		renderFlowActionBindingForm(options),
		renderFlowFieldRows(options),
		renderFlowAddFieldForm(options),
		renderFlowTestForm(options),
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("id", options.ID),
		gosx.Attr("data-studio-flow-field-editor", "true"),
		gosx.Attr("data-studio-flow-field-editor-flow", options.FlowKey),
		gosx.Attr("data-studio-flow-field-editor-action", options.ActionKey),
	), gosx.Fragment(children...))
}

func renderFlowActionBindingForm(options FlowFieldEditorOptions) gosx.Node {
	children := flowHiddenIdentity(options, authoring.AuthoringOperationSetFlowAction, options.CSRFToken)
	children = append(children,
		gosx.El("label", gosx.Attrs(gosx.Attr("for", options.ID+"-action-label")), gosx.Text("Action label")),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("id", options.ID+"-action-label"),
			gosx.Attr("name", authoring.AuthoringFieldFlowActionLabel),
			gosx.Attr("value", options.ActionLabel),
		)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", options.ID+"-binding")), gosx.Text("Submit binding (handler)")),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("id", options.ID+"-binding"),
			gosx.Attr("name", authoring.AuthoringFieldBinding),
			gosx.Attr("value", options.Binding),
			gosx.Attr("data-studio-flow-binding-input", "true"),
		)),
		gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-flow-binding-status", "true")), gosx.Text(options.BindingStatusLabel)),
	)
	if options.BindingReason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-flow-binding-reason", "true")), gosx.Text(options.BindingReason)))
	}
	children = append(children, gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Save action")))
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("method", "post"),
		gosx.Attr("action", options.Action),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
		gosx.Attr("data-studio-flow-action-form", options.ActionKey),
	), gosx.Fragment(children...))
}

func renderFlowFieldRows(options FlowFieldEditorOptions) gosx.Node {
	rows := make([]gosx.Node, 0, len(options.Fields))
	for _, field := range options.Fields {
		rows = append(rows, renderFlowFieldRow(options, field))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-studio-flow-field-rows", "true")), gosx.Fragment(rows...))
}

func renderFlowFieldRow(options FlowFieldEditorOptions, field FlowFieldRow) gosx.Node {
	rowID := options.ID + "-field-" + field.Name
	saveChildren := flowHiddenIdentity(options, authoring.AuthoringOperationSetFlowField, options.CSRFToken)
	saveChildren = append(saveChildren, hidden(authoring.AuthoringFieldFlowFieldName, field.Name))
	saveChildren = append(saveChildren,
		gosx.El("label", gosx.Attrs(gosx.Attr("for", rowID+"-label")), gosx.Text("Label")),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", rowID+"-label"), gosx.Attr("name", authoring.AuthoringFieldFlowFieldLabel), gosx.Attr("value", field.Label))),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", rowID+"-kind")), gosx.Text("Kind")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", rowID+"-kind"), gosx.Attr("name", authoring.AuthoringFieldControlKind)), gosx.Fragment(flowFieldKindOptions(field.Kind)...)),
	)
	requiredAttrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("id", rowID+"-required"), gosx.Attr("name", authoring.AuthoringFieldFlowFieldRequired), gosx.Attr("value", "true")}
	if field.Required {
		requiredAttrs = append(requiredAttrs, gosx.Attr("checked", "checked"))
	}
	saveChildren = append(saveChildren,
		gosx.El("label", gosx.Attrs(gosx.Attr("for", rowID+"-required")), gosx.Text("Required (validation rule)")),
		gosx.El("input", gosx.Attrs(requiredAttrs...)),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Save field")),
	)
	saveForm := gosx.El("form", gosx.Attrs(
		gosx.Attr("method", "post"),
		gosx.Attr("action", options.Action),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
		gosx.Attr("data-studio-flow-field-form", field.Name),
	), gosx.Fragment(saveChildren...))

	removeChildren := flowHiddenIdentity(options, authoring.AuthoringOperationRemoveFlowField, options.CSRFToken)
	removeChildren = append(removeChildren,
		hidden(authoring.AuthoringFieldFlowFieldName, field.Name),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Remove field")),
	)
	removeForm := gosx.El("form", gosx.Attrs(
		gosx.Attr("method", "post"),
		gosx.Attr("action", options.Action),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
		gosx.Attr("data-studio-flow-field-remove-form", field.Name),
	), gosx.Fragment(removeChildren...))

	return gosx.El("article", gosx.Attrs(gosx.Attr("data-studio-flow-field-row", field.Name)),
		gosx.El("strong", nil, gosx.Text(field.Name)),
		saveForm,
		removeForm,
	)
}

func renderFlowAddFieldForm(options FlowFieldEditorOptions) gosx.Node {
	children := flowHiddenIdentity(options, authoring.AuthoringOperationSetFlowField, options.CSRFToken)
	children = append(children,
		gosx.El("label", gosx.Attrs(gosx.Attr("for", options.ID+"-new-name")), gosx.Text("New field name")),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", options.ID+"-new-name"), gosx.Attr("name", authoring.AuthoringFieldFlowFieldName))),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", options.ID+"-new-label")), gosx.Text("Label")),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", options.ID+"-new-label"), gosx.Attr("name", authoring.AuthoringFieldFlowFieldLabel))),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", options.ID+"-new-kind")), gosx.Text("Kind")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", options.ID+"-new-kind"), gosx.Attr("name", authoring.AuthoringFieldControlKind)), gosx.Fragment(flowFieldKindOptions(string(core.ControlText))...)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", options.ID+"-new-required")), gosx.Text("Required (validation rule)")),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("id", options.ID+"-new-required"), gosx.Attr("name", authoring.AuthoringFieldFlowFieldRequired), gosx.Attr("value", "true"))),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Add field")),
	)
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("method", "post"),
		gosx.Attr("action", options.Action),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
		gosx.Attr("data-studio-flow-add-field-form", "true"),
	), gosx.Fragment(children...))
}

func renderFlowTestForm(options FlowFieldEditorOptions) gosx.Node {
	children := flowHiddenIdentity(options, authoring.AuthoringOperationTestFlowAction, options.CSRFToken)
	for _, value := range options.TestValues {
		fieldID := options.ID + "-test-" + value.Name
		children = append(children,
			gosx.El("label", gosx.Attrs(gosx.Attr("for", fieldID)), gosx.Text(value.Label)),
			gosx.El("input", gosx.Attrs(gosx.Attr("id", fieldID), gosx.Attr("name", value.Name), gosx.Attr("value", value.Value))),
		)
		if value.Error != "" {
			children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-flow-test-field-error", value.Name)), gosx.Text(value.Error)))
		}
	}
	children = append(children, gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Run test submission")))
	if options.TestMessage != "" {
		children = append(children, gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-flow-test-result", "true")), gosx.Text(options.TestMessage)))
	}
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("method", "post"),
		gosx.Attr("action", options.Action),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
		gosx.Attr("data-studio-flow-test-form", options.ActionKey),
	), gosx.Fragment(children...))
}

func flowHiddenIdentity(options FlowFieldEditorOptions, kind authoring.AuthoringOperationKind, csrfToken string) []gosx.Node {
	nodes := []gosx.Node{
		hidden(authoring.AuthoringFieldOperation, string(kind)),
		hidden(authoring.AuthoringFieldFlowKey, options.FlowKey),
		hidden(authoring.AuthoringFieldFlowActionKey, options.ActionKey),
	}
	if csrfToken != "" {
		nodes = append(nodes, hidden("csrf_token", csrfToken))
	}
	return nodes
}

func flowFieldKindOptions(selected string) []gosx.Node {
	options := make([]gosx.Node, 0, len(authoring.FlowFieldControlKinds()))
	for _, kind := range authoring.FlowFieldControlKinds() {
		attrs := []any{gosx.Attr("value", string(kind))}
		if string(kind) == selected {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(core.ControlKindLabel(kind))))
	}
	return options
}

func firstNonEmptyPanelValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
