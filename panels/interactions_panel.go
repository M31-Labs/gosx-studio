package panels

import (
	"strconv"
	"strings"

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
//
// Wave 3A (inspector-presentation): this file used to render one full
// "Interactions" section per canvas target, always fully expanded — a host
// composing N targets (editorInteractionsPanels in muddy-noni-commerce loops
// over every canvas target and Fragments one RenderInteractionsPanel call per
// target) produced N repeated headers and N*len(kinds) always-visible
// Effect/Duration/Delay selects with no way to collapse or hide the ones
// nobody attached ("HOME with staging content" measured 162 visible selects
// across 15 targets). Two changes fix this without requiring a host rewrite:
//
//  1. RenderInteractionsPanel now renders ONE TARGET as a compact card: a
//     summary row (target label + attached-interaction summary) plus a
//     closed-by-default <details> disclosure holding the existing per-kind
//     edit rows. A host that keeps calling this per-target (unchanged call
//     signature) already gets a dramatically smaller render.
//  2. RenderInteractionsInspector is the new, correct entry point: it takes
//     every target's options together, renders the "Interactions" header
//     exactly once, lists targets that already have an attached interaction
//     as compact cards, and tucks every target with ZERO attached
//     interactions behind one shared "+ Add interaction" picker instead of
//     pre-expanding 15 control sets. Hosts should switch their per-target loop
//     to build a []InteractionsPanelOptions and call this once; see the
//     handoff report for the exact muddy-noni-commerce wiring change.

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

// InteractionsPanelOptions describes one canvas target the interactions
// inspector edits.
type InteractionsPanelOptions struct {
	ID, Action, CSRFToken         string
	PageKey, PageLabel, PageRoute string
	ComponentKey, ComponentLabel  string
	Rows                          []InteractionRow
}

// InteractionsInspectorOptions describes every canvas target the aggregate
// inspector composes into one panel. See RenderInteractionsInspector.
type InteractionsInspectorOptions struct {
	ID      string
	Targets []InteractionsPanelOptions
}

// RenderInteractionsPanel renders ONE target's compact interactions card: a
// summary row plus a closed-by-default disclosure with the reveal-on-scroll
// and hover-focus-state edit rows. Kept as a standalone entry point for hosts
// that still call it once per target (unchanged signature/back-compat); for
// the correct multi-target "+ Add interaction" picker UX see
// RenderInteractionsInspector.
func RenderInteractionsPanel(options InteractionsPanelOptions) gosx.Node {
	options = interactionsPanelDefaults(options)
	return renderInteractionsTargetCard(options)
}

// RenderInteractionsInspector renders the interactions inspector for every
// target at once: one "Interactions" header, a compact card per target that
// already has an attached interaction, and every target with zero attached
// interactions grouped behind a single "+ Add interaction" disclosure
// (punch #24 — add-interaction discoverability) instead of pre-expanded per
// target.
func RenderInteractionsInspector(options InteractionsInspectorOptions) gosx.Node {
	id := core.FirstNonEmpty(options.ID, "studio-interactions-inspector")
	attachedCards := make([]gosx.Node, 0, len(options.Targets))
	unattachedCards := make([]gosx.Node, 0, len(options.Targets))
	for _, target := range options.Targets {
		target = interactionsPanelDefaults(target)
		if interactionsTargetAttached(target.Rows) {
			attachedCards = append(attachedCards, renderInteractionsTargetCard(target))
			continue
		}
		unattachedCards = append(unattachedCards, renderInteractionsTargetCard(target))
	}
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Interactions")),
			gosx.El("p", nil, gosx.Text("Attach safe, typed behavior to canvas elements. Reduced motion always shows content immediately; keyboard focus always gets the same treatment as hover.")),
		),
	}
	if len(attachedCards) > 0 {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-interactions__list"),
			gosx.Attr("data-studio-interactions-list", "true"),
		), gosx.Fragment(attachedCards...)))
	} else {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text("No elements have interactions attached yet.")))
	}
	if len(unattachedCards) > 0 {
		children = append(children, gosx.El("details", gosx.Attrs(
			gosx.Attr("class", "studio-interactions__add"),
			gosx.Attr("data-studio-interactions-add", "true"),
		),
			gosx.El("summary", nil, gosx.Text(interactionsAddSummaryLabel(len(unattachedCards)))),
			gosx.Fragment(unattachedCards...),
		))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("id", id),
		gosx.Attr("data-studio-interactions-inspector-group", "true"),
	), gosx.Fragment(children...))
}

func interactionsPanelDefaults(options InteractionsPanelOptions) InteractionsPanelOptions {
	if options.ID == "" {
		options.ID = "studio-interactions-" + interactionsTargetIDSuffix(options.ComponentKey)
	}
	if options.Action == "" {
		options.Action = "/admin/editor/__actions/authoring"
	}
	return options
}

func interactionsAddSummaryLabel(count int) string {
	if count == 1 {
		return "+ Add interaction (1 element available)"
	}
	return "+ Add interaction (" + strconv.Itoa(count) + " elements available)"
}

// renderInteractionsTargetCard renders one target's compact card: a header
// (target label + a one-line summary of whatever is attached) plus a
// closed-by-default <details> disclosure with the full per-kind edit rows —
// the same rows renderInteractionRow always rendered, just no longer
// pre-expanded for every target on the page.
func renderInteractionsTargetCard(options InteractionsPanelOptions) gosx.Node {
	rows := make([]gosx.Node, 0, len(options.Rows))
	for _, row := range options.Rows {
		rows = append(rows, renderInteractionRow(options, row))
	}
	label := core.FirstNonEmpty(options.ComponentLabel, options.ComponentKey, "Untitled element")
	attached := interactionsTargetAttached(options.Rows)
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("id", options.ID),
		gosx.Attr("class", "studio-interactions-target"),
		gosx.Attr("data-studio-interactions-inspector", "true"),
		gosx.Attr("data-studio-interactions-target", "true"),
		gosx.Attr("data-studio-interaction-target-attached", core.BoolAttr(attached)),
		gosx.Attr("data-studio-selected-route", options.PageRoute),
		gosx.Attr("data-studio-selected-component", options.ComponentKey),
	),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-interactions-target__head")),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-interaction-target-label", "true")), gosx.Text(label)),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-interaction-target-summary", "true")), gosx.Text(interactionsTargetSummary(options.Rows))),
		),
		gosx.El("details", gosx.Attrs(gosx.Attr("class", "studio-interactions-target__editor")),
			gosx.El("summary", nil, gosx.Text(interactionsDisclosureLabel(attached))),
			gosx.Fragment(rows...),
		),
	)
}

func interactionsTargetAttached(rows []InteractionRow) bool {
	for _, row := range rows {
		if row.Attached {
			return true
		}
	}
	return false
}

func interactionsTargetSummary(rows []InteractionRow) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		if !row.Attached {
			continue
		}
		effectLabel := core.InteractionEffectLabel(core.InteractionEffect(row.Effect))
		parts = append(parts, row.KindLabel+": "+effectLabel)
	}
	if len(parts) == 0 {
		return "Not attached"
	}
	return strings.Join(parts, " · ")
}

func interactionsDisclosureLabel(attached bool) string {
	if attached {
		return "Edit interactions"
	}
	return "+ Add interaction"
}

func interactionsTargetIDSuffix(componentKey string) string {
	key := strings.TrimSpace(componentKey)
	if key == "" {
		return "target"
	}
	replacer := strings.NewReplacer(":", "-", " ", "-", ".", "-")
	return replacer.Replace(key)
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
