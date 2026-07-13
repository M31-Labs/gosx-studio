package panels

import (
	"strconv"

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

// FlowFieldEditorInspectorOptions describes every flow action the aggregate
// inspector composes into one panel. See RenderFlowFieldEditorInspector.
type FlowFieldEditorInspectorOptions struct {
	ID      string
	Actions []FlowFieldEditorOptions
}

// RenderFlowFieldEditorPanel renders one flow action's compact card: a
// summary row (action + flow label, field count) plus a closed-by-default
// <details> disclosure holding the existing action-binding/field-rows/
// add-field/test forms.
//
// Wave 3B (tamarind, ADVANCED discipline): this used to render a full,
// always-expanded "Fields, validation, and binding" section per call. A host
// composing one call per flow action (muddy-noni-commerce's
// editorFlowFieldEditorPanels loops every flow's every action and Fragments
// one RenderFlowFieldEditorPanel per action) produced N repeated headers and
// N*(fields+1) always-visible Kind selects with no way to collapse the ones
// the operator isn't editing right now — the magnolia-3 re-score's "4
// duplicate 'Fields, validation, and binding' panels" / "18 visible selects"
// ADVANCED finding, on this seed's 4 flow actions (contact-submit,
// purchase-request-submit, checkout-handoff-continue, newsletter-submit).
// This mirrors the exact fix wave 3A already proved for
// panels/interactions_panel.go's identical "N-times-duplicated block"
// shape: RenderFlowFieldEditorPanel now renders ONE compact card (unchanged
// call signature, so a host that keeps calling it per-action already gets a
// dramatically smaller render); RenderFlowFieldEditorInspector is the new,
// correct aggregate entry point that renders the "Fields, validation, and
// binding" header exactly once and lists every action as a collapsed card.
func RenderFlowFieldEditorPanel(options FlowFieldEditorOptions) gosx.Node {
	options = flowFieldEditorDefaults(options)
	return renderFlowFieldEditorActionCard(options)
}

// RenderFlowFieldEditorInspector renders the flow field editor for every
// action at once: one "Fields, validation, and binding" header, then one
// collapsed-by-default compact card per action (see
// renderFlowFieldEditorActionCard) instead of N fully-expanded, individually
// headered sections.
func RenderFlowFieldEditorInspector(options FlowFieldEditorInspectorOptions) gosx.Node {
	id := core.FirstNonEmpty(options.ID, "studio-flow-field-editor-inspector")
	cards := make([]gosx.Node, 0, len(options.Actions))
	for _, action := range options.Actions {
		cards = append(cards, renderFlowFieldEditorActionCard(flowFieldEditorDefaults(action)))
	}
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Fields, validation, and binding")),
			gosx.El("p", nil, gosx.Text("Edit the fields, validation rules, and submit binding for each form flow action.")),
		),
	}
	if len(cards) > 0 {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-flow-field-editor__list"),
			gosx.Attr("data-studio-flow-field-editor-list", "true"),
		), gosx.Fragment(cards...)))
	} else {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text("No form flow actions yet.")))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("id", id),
		gosx.Attr("data-studio-flow-field-editor-inspector-group", "true"),
	), gosx.Fragment(children...))
}

func flowFieldEditorDefaults(options FlowFieldEditorOptions) FlowFieldEditorOptions {
	if options.ID == "" {
		options.ID = "studio-flow-field-editor"
	}
	if options.Action == "" {
		options.Action = "/admin/editor/__actions/authoring"
	}
	return options
}

// renderFlowFieldEditorActionCard renders one action's compact card: a
// header (action · flow label + a one-line field-count summary) plus a
// closed-by-default <details> disclosure with the full action-binding/
// field-rows/add-field/test forms — the same forms this file always
// rendered, just no longer pre-expanded for every action on the page.
func renderFlowFieldEditorActionCard(options FlowFieldEditorOptions) gosx.Node {
	label := firstNonEmptyPanelValue(options.ActionLabel, options.ActionKey) + " · " + firstNonEmptyPanelValue(options.FlowLabel, options.FlowKey)
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("id", options.ID),
		gosx.Attr("class", "studio-flow-field-editor"),
		gosx.Attr("data-studio-flow-field-editor", "true"),
		gosx.Attr("data-studio-flow-field-editor-flow", options.FlowKey),
		gosx.Attr("data-studio-flow-field-editor-action", options.ActionKey),
	),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-flow-field-editor__head")),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-flow-field-editor-label", "true")), gosx.Text(label)),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-flow-field-editor-summary", "true")), gosx.Text(flowFieldEditorSummary(options))),
		),
		gosx.El("details", gosx.Attrs(gosx.Attr("class", "studio-flow-field-editor__editor")),
			gosx.El("summary", nil, gosx.Text("Edit fields, validation, and binding")),
			renderFlowActionBindingForm(options),
			renderFlowFieldRows(options),
			renderFlowAddFieldForm(options),
			renderFlowTestForm(options),
		),
	)
}

func flowFieldEditorSummary(options FlowFieldEditorOptions) string {
	count := len(options.Fields)
	required := 0
	for _, field := range options.Fields {
		if field.Required {
			required++
		}
	}
	switch count {
	case 0:
		return "No fields yet"
	case 1:
		if required == 1 {
			return "1 field · 1 required"
		}
		return "1 field"
	default:
		if required > 0 {
			return strconv.Itoa(count) + " fields · " + strconv.Itoa(required) + " required"
		}
		return strconv.Itoa(count) + " fields"
	}
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
