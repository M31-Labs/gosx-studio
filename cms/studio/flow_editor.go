package studio

import (
	"strings"

	"m31labs.dev/gosx"
	cmsflows "m31labs.dev/gosx-studio/cms/flows"
)

type FlowEditorOptions struct {
	LibraryClass       string
	LibraryTitle       string
	LibraryLabel       string
	TablistClass       string
	SummaryCardClass   string
	FieldsClass        string
	FieldsTitle        string
	FieldsLabel        string
	EditorCardClass    string
	HideHandlerRef     bool
	HandlerRefLabel    string
	PreviewButtonClass string
	PreviewButtonLabel string
	PublishAction      string
	PublishButtonClass string
	PublishButtonLabel string
	EmptyText          string
}

func RenderFlowEditor(flows []cmsflows.StudioFlow, options FlowEditorOptions) gosx.Node {
	return gosx.Fragment(
		RenderFlowEditorLibrary(flows, options),
		RenderFlowEditorFields(flows, options),
	)
}

func RenderFlowEditorLibrary(flows []cmsflows.StudioFlow, options FlowEditorOptions) gosx.Node {
	flowViews := normalizeFlowEditorViews(flows)
	cards := make([]gosx.Node, 0, len(flowViews))
	for index, flow := range flowViews {
		selected := index == 0
		className := firstNonEmpty(options.SummaryCardClass, "list-card studio-flow-summary")
		if selected && !strings.Contains(className, "is-selected") {
			className += " is-selected"
		}
		cards = append(cards, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", className),
			gosx.Attr("id", DOMID("studio-flow-tab", flow.Key)),
			gosx.Attr("type", "button"),
			gosx.Attr("role", "tab"),
			gosx.Attr("data-studio-flow-card", flow.Key),
			gosx.Attr("data-studio-flow-route", flow.Route),
			gosx.Attr("data-studio-flow-label", flow.Label),
			gosx.Attr("data-studio-flow-status", flow.StatusLabel),
			gosx.Attr("aria-controls", DOMID("studio-flow-editor", flow.Key)),
			gosx.Attr("aria-selected", studioBoolString(selected)),
			gosx.Attr("tabindex", studioTabIndex(selected)),
		),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-summary__head")),
				gosx.El("strong", nil, gosx.Text(flow.Label)),
				gosx.El("output", nil, gosx.Text(flow.StatusLabel)),
			),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-summary__meta")), gosx.Text(flow.Summary)),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("class", "studio-flow-summary__dirty"),
				gosx.Attr("data-studio-flow-dirty-badge", "true"),
				gosx.BoolAttr("hidden"),
			), gosx.Text("Saved")),
		))
	}
	if len(cards) == 0 {
		cards = append(cards, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(firstNonEmpty(options.EmptyText, "No flows registered yet."))))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.LibraryClass, "studio-panel studio-flow-library")),
		gosx.Attr("aria-label", firstNonEmpty(options.LibraryLabel, "Flows")),
	),
		gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.LibraryTitle, "Flows"))),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.TablistClass, "studio-flow-tablist")),
			gosx.Attr("data-studio-flow-library", "true"),
			gosx.Attr("role", "tablist"),
			gosx.Attr("aria-label", "Flow editor selection"),
		), gosx.Fragment(cards...)),
	)
}

func RenderFlowEditorFields(flows []cmsflows.StudioFlow, options FlowEditorOptions) gosx.Node {
	flowViews := normalizeFlowEditorViews(flows)
	editors := make([]gosx.Node, 0, len(flowViews))
	for index, flow := range flowViews {
		editors = append(editors, renderFlowEditorFieldset(flow, index == 0, options))
	}
	if len(editors) == 0 {
		editors = append(editors, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(firstNonEmpty(options.EmptyText, "No flows registered yet."))))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.FieldsClass, "studio-panel studio-flow-fields")),
		gosx.Attr("data-studio-flow-fields", "true"),
		gosx.Attr("aria-label", firstNonEmpty(options.FieldsLabel, "Flow fields")),
	),
		gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.FieldsTitle, "Flow fields"))),
		gosx.Fragment(editors...),
	)
}

func FlowHandlerRefInputName(flowKey string) string {
	return "flow" + FieldNamePrefix(flowKey) + "HandlerRef"
}

func FlowStepLabelInputName(flowKey, stepKey string) string {
	return "flow" + FieldNamePrefix(flowKey) + "Step" + FieldNamePrefix(stepKey) + "Label"
}

func renderFlowEditorFieldset(flow cmsflows.StudioFlow, selected bool, options FlowEditorOptions) gosx.Node {
	className := firstNonEmpty(options.EditorCardClass, "list-card studio-flow-editor")
	if selected && !strings.Contains(className, "is-selected") {
		className += " is-selected"
	}
	attrs := []any{
		gosx.Attr("class", className),
		gosx.Attr("id", DOMID("studio-flow-editor", flow.Key)),
		gosx.Attr("role", "tabpanel"),
		gosx.Attr("aria-labelledby", DOMID("studio-flow-tab", flow.Key)),
		gosx.Attr("aria-hidden", studioBoolString(!selected)),
		gosx.Attr("tabindex", "0"),
		gosx.Attr("data-studio-flow-editor", flow.Key),
		gosx.Attr("data-studio-flow-route", flow.Route),
	}
	if !selected {
		attrs = append(attrs, gosx.BoolAttr("hidden"))
	}
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(flow.Label)),
				gosx.El("span", nil, gosx.Text(flow.Summary+" / "+flow.StatusLabel)),
			),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("class", "studio-flow-editor__dirty"),
				gosx.Attr("data-studio-flow-editor-dirty", "true"),
				gosx.BoolAttr("hidden"),
			), gosx.Text("Saved")),
		),
	}
	handlerAttrs := []any{
		gosx.Attr("id", DOMID("studio-flow-handler", flow.Key)),
		gosx.Attr("name", FlowHandlerRefInputName(flow.Key)),
		gosx.Attr("value", flow.PrimaryAction.HandlerRef),
		gosx.Attr("data-studio-initial-value", flow.PrimaryAction.HandlerRef),
	}
	if options.HideHandlerRef {
		children = append(children, gosx.El("input", gosx.Attrs(append(handlerAttrs, gosx.Attr("type", "hidden"))...)))
	} else {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "field"),
			gosx.Attr("for", DOMID("studio-flow-handler", flow.Key)),
		),
			gosx.El("span", nil, gosx.Text(firstNonEmpty(options.HandlerRefLabel, "Handler ref"))),
			gosx.El("input", gosx.Attrs(handlerAttrs...)),
		))
	}
	for _, step := range flow.Steps {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "field"),
			gosx.Attr("for", DOMID("studio-flow-step", flow.Key, step.Key, "label")),
		),
			gosx.El("span", nil, gosx.Text(step.Label+" step label")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", DOMID("studio-flow-step", flow.Key, step.Key, "label")),
				gosx.Attr("name", FlowStepLabelInputName(flow.Key, step.Key)),
				gosx.Attr("value", step.Label),
				gosx.Attr("data-studio-initial-value", step.Label),
			)),
		))
	}
	buttons := []gosx.Node{}
	if flow.HasRoute || strings.TrimSpace(flow.Route) != "" {
		buttons = append(buttons, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.PreviewButtonClass, "button button--secondary")),
			gosx.Attr("id", DOMID("studio-flow-preview", flow.Key)),
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-preview-flow", flow.Route),
		), gosx.Text(firstNonEmpty(options.PreviewButtonLabel, "Preview route"))))
	}
	if strings.TrimSpace(options.PublishAction) != "" {
		buttons = append(buttons, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.PublishButtonClass, "button button--secondary")),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", options.PublishAction),
			gosx.Attr("name", "flowKey"),
			gosx.Attr("value", flow.Key),
		), gosx.Text(firstNonEmpty(options.PublishButtonLabel, "Publish flow"))))
	}
	children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(buttons...)))
	return gosx.El("article", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func normalizeFlowEditorViews(flows []cmsflows.StudioFlow) []cmsflows.StudioFlow {
	out := make([]cmsflows.StudioFlow, 0, len(flows))
	for _, flow := range flows {
		flow.Key = normalizeKey(firstNonEmpty(flow.Key, flow.Label))
		flow.Label = strings.TrimSpace(flow.Label)
		if flow.Key == "" || flow.Label == "" {
			continue
		}
		flow.Summary = strings.TrimSpace(flow.Summary)
		flow.StatusLabel = firstNonEmpty(flow.StatusLabel, "Draft")
		flow.Route = strings.TrimSpace(flow.Route)
		out = append(out, flow)
	}
	return out
}

func studioBoolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func studioTabIndex(selected bool) string {
	if selected {
		return "0"
	}
	return "-1"
}
