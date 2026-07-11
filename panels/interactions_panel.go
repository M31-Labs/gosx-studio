package panels

import (
	"strconv"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

// The interactions inspector attaches typed, allow-listed interactions
// (core/interactions.go) to the selected canvas component. Each row posts an
// ordinary form directly to the authoring action — it deliberately does not
// opt into the durable-history/collaboration operation bridge
// (operationruntime), whose OperationTarget/OperationKind shapes are frozen to
// set-field/set-style/reset-style/undo/redo and a fixed target address; a
// generic interaction payload (kind/effect/duration/delay/once) does not fit
// that shape without extending the durable operation protocol, which is out
// of scope here. A later slice can add collaboration-aware interaction
// editing once that protocol grows an extensible payload.

// InteractionOption is one allow-listed choice for an interaction's effect,
// duration, or delay control.
type InteractionOption struct {
	Value string
	Label string
}

// InteractionRow is one interaction kind's editable row. Studio supports at
// most one interaction per kind per target in this iteration, so the row IS
// the interaction rather than a list the operator manages by key.
type InteractionRow struct {
	Kind              string
	KindLabel         string
	Attached          bool
	InteractionKey    string
	Effect            string
	EffectOptions     []InteractionOption
	DurationMS        int
	DurationOptions   []int
	DelayMS           int
	DelayOptions      []int
	Once              bool
	ReducedMotionNote string
	Status            string
	StatusLabel       string
	Reason            string
}

// InteractionsPanelOptions describes the selected canvas component the
// interactions inspector edits.
type InteractionsPanelOptions struct {
	ID, Action, CSRFToken         string
	PageKey, PageLabel, PageRoute string
	ComponentKey, ComponentLabel  string
	Rows                          []InteractionRow
}

// RenderInteractionsPanel renders the reveal-on-scroll and hover-focus-state
// rows for the selected component.
func RenderInteractionsPanel(options InteractionsPanelOptions) gosx.Node {
	if options.ID == "" {
		options.ID = "studio-interactions"
	}
	if options.Action == "" {
		options.Action = "/admin/editor/__actions/authoring"
	}
	rows := make([]gosx.Node, 0, len(options.Rows))
	for _, row := range options.Rows {
		rows = append(rows, renderInteractionRow(options, row))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("id", options.ID),
		gosx.Attr("data-studio-interactions-inspector", "true"),
		gosx.Attr("data-studio-selected-route", options.PageRoute),
		gosx.Attr("data-studio-selected-component", options.ComponentKey),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Interactions")),
			gosx.El("p", nil, gosx.Text("Attach safe, typed behavior to the selected element. Reduced motion always shows content immediately; keyboard focus always gets the same treatment as hover.")),
		),
		gosx.Fragment(rows...),
	)
}

func renderInteractionRow(options InteractionsPanelOptions, row InteractionRow) gosx.Node {
	rowID := options.ID + "-" + row.Kind
	effectOptions := make([]gosx.Node, 0, len(row.EffectOptions))
	for _, option := range row.EffectOptions {
		attrs := []any{gosx.Attr("value", option.Value)}
		if option.Value == row.Effect {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		effectOptions = append(effectOptions, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(option.Label)))
	}
	durationOptions := make([]gosx.Node, 0, len(row.DurationOptions))
	for _, ms := range row.DurationOptions {
		value := strconv.Itoa(ms)
		attrs := []any{gosx.Attr("value", value)}
		if ms == row.DurationMS {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		durationOptions = append(durationOptions, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(value+" ms")))
	}
	delayOptions := make([]gosx.Node, 0, len(row.DelayOptions))
	for _, ms := range row.DelayOptions {
		value := strconv.Itoa(ms)
		attrs := []any{gosx.Attr("value", value)}
		if ms == row.DelayMS {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		delayOptions = append(delayOptions, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(value+" ms delay")))
	}
	saveChildren := []gosx.Node{
		hidden(authoring.AuthoringFieldOperation, string(authoring.AuthoringOperationSetInteraction)),
		hidden(authoring.AuthoringFieldPageKey, options.PageKey),
		hidden(authoring.AuthoringFieldPageRoute, options.PageRoute),
		hidden(authoring.AuthoringFieldComponentKey, options.ComponentKey),
		hidden(authoring.AuthoringFieldInteractionKey, row.InteractionKey),
		hidden(authoring.AuthoringFieldInteractionKind, row.Kind),
	}
	if options.CSRFToken != "" {
		saveChildren = append(saveChildren, hidden("csrf_token", options.CSRFToken))
	}
	saveChildren = append(saveChildren,
		gosx.El("label", gosx.Attrs(gosx.Attr("for", rowID+"-effect")), gosx.Text("Effect")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", rowID+"-effect"), gosx.Attr("name", authoring.AuthoringFieldInteractionEffect)), gosx.Fragment(effectOptions...)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", rowID+"-duration")), gosx.Text("Duration")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", rowID+"-duration"), gosx.Attr("name", authoring.AuthoringFieldInteractionDurationMS)), gosx.Fragment(durationOptions...)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", rowID+"-delay")), gosx.Text("Delay")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", rowID+"-delay"), gosx.Attr("name", authoring.AuthoringFieldInteractionDelayMS)), gosx.Fragment(delayOptions...)),
	)
	if row.Kind == string(core.InteractionRevealOnScroll) {
		onceAttrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("id", rowID+"-once"), gosx.Attr("name", authoring.AuthoringFieldInteractionOnce), gosx.Attr("value", "true")}
		if row.Once {
			onceAttrs = append(onceAttrs, gosx.Attr("checked", "checked"))
		}
		saveChildren = append(saveChildren,
			gosx.El("label", gosx.Attrs(gosx.Attr("for", rowID+"-once")), gosx.Text("Reveal once")),
			gosx.El("input", gosx.Attrs(onceAttrs...)),
		)
	}
	saveChildren = append(saveChildren,
		gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-interaction-reduced-motion-note", "true")), gosx.Text(row.ReducedMotionNote)),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text(attachedButtonLabel(row.Attached))),
	)
	saveForm := gosx.El("form", gosx.Attrs(
		gosx.Attr("method", "post"),
		gosx.Attr("action", options.Action),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
		gosx.Attr("data-studio-interaction-row", row.Kind),
		gosx.Attr("data-studio-interaction-attached", core.BoolAttr(row.Attached)),
		gosx.Attr("data-studio-interaction-status", row.Status),
	), gosx.Fragment(saveChildren...))

	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("strong", nil, gosx.Text(row.KindLabel)),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-interaction-status-label", "true")), gosx.Text(row.StatusLabel)),
		),
		saveForm,
	}
	if row.Reason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-interaction-reason", "true")), gosx.Text(row.Reason)))
	}
	if row.Attached {
		removeChildren := []gosx.Node{
			hidden(authoring.AuthoringFieldOperation, string(authoring.AuthoringOperationRemoveInteraction)),
			hidden(authoring.AuthoringFieldPageKey, options.PageKey),
			hidden(authoring.AuthoringFieldPageRoute, options.PageRoute),
			hidden(authoring.AuthoringFieldComponentKey, options.ComponentKey),
			hidden(authoring.AuthoringFieldInteractionKey, row.InteractionKey),
		}
		if options.CSRFToken != "" {
			removeChildren = append(removeChildren, hidden("csrf_token", options.CSRFToken))
		}
		removeChildren = append(removeChildren, gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Remove")))
		children = append(children, gosx.El("form", gosx.Attrs(
			gosx.Attr("method", "post"),
			gosx.Attr("action", options.Action),
			gosx.Attr("data-gosx-studio-authoring-managed", "true"),
			gosx.Attr("data-studio-interaction-remove", row.Kind),
		), gosx.Fragment(removeChildren...)))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("data-studio-interaction-card", row.Kind),
	), gosx.Fragment(children...))
}

func attachedButtonLabel(attached bool) string {
	if attached {
		return "Save"
	}
	return "Add"
}

// InteractionRowFromInteraction builds an InteractionRow from a
// (possibly zero-value) core.Interaction for kind, using core's own
// allow-lists so the panel and the server validate against exactly the same
// options.
func InteractionRowFromInteraction(kind core.InteractionKind, interaction core.Interaction, diagnostic core.InteractionDiagnostic) InteractionRow {
	effectOptions := make([]InteractionOption, 0, len(core.InteractionEffectOptions(kind)))
	for _, effect := range core.InteractionEffectOptions(kind) {
		effectOptions = append(effectOptions, InteractionOption{Value: string(effect), Label: core.InteractionEffectLabel(effect)})
	}
	attached := interaction.Kind == kind
	effect := interaction.Effect
	durationMS := interaction.DurationMS
	delayMS := interaction.DelayMS
	if !attached {
		normalized := core.Interaction{Kind: kind}.Normalize()
		effect = normalized.Effect
		durationMS = normalized.DurationMS
		delayMS = normalized.DelayMS
	}
	status := "ready"
	statusLabel := "Not attached"
	reason := ""
	if attached {
		status = string(diagnostic.Status)
		statusLabel = core.ReadinessStatusLabel(diagnostic.Status)
		reason = diagnostic.Reason
	}
	return InteractionRow{
		Kind:              string(kind),
		KindLabel:         core.InteractionKindLabel(kind),
		Attached:          attached,
		InteractionKey:    interaction.Key,
		Effect:            string(effect),
		EffectOptions:     effectOptions,
		DurationMS:        durationMS,
		DurationOptions:   core.InteractionDurationOptionsMS(),
		DelayMS:           delayMS,
		DelayOptions:      core.InteractionDelayOptionsMS(),
		Once:              interaction.Once,
		ReducedMotionNote: "Reduced motion: content shows immediately, no animation plays.",
		Status:            status,
		StatusLabel:       statusLabel,
		Reason:            reason,
	}
}
