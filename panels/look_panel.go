package panels

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

type LookPanelOptions struct {
	RootAttrs map[string]any
}

func RenderLookPanel(view map[string]any, options LookPanelOptions) gosx.Node {
	settings := core.WorkbenchViewMap(view, "settings")
	children := []gosx.Node{renderLookPanelHead(view)}
	for _, group := range core.WorkbenchViewMapList(view, "groups") {
		children = append(children, renderLookPanelGroupInput(group))
	}
	children = append(children,
		renderLookPanelGroups(view),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-look-panel__scope"),
			gosx.Attr("data-studio-look-scope", "true"),
		),
			renderLookStyleScope(core.WorkbenchViewMap(settings, "scope")),
			renderLookStyleWorkbench(core.WorkbenchViewMap(settings, "workbench")),
		),
		renderLookPanelThemeSlot(view, settings),
		renderLookPanelLayoutSlot(view, settings),
		renderLookPanelColorSlot(view, settings),
		renderLookPanelDetailsSlot(view, settings),
	)
	return gosx.El("section", gosx.Attrs(lookPanelRootAttrs(view, options.RootAttrs)...), gosx.Fragment(children...))
}

func lookPanelRootAttrs(view map[string]any, rootAttrs map[string]any) []any {
	attrs := []any{
		gosx.Attr("class", "editor-panel editor-panel--look-studio studio-look-panel"),
		gosx.Attr("data-studio-look-panel", "true"),
		gosx.Attr("data-studio-mode-panel", "look"),
		gosx.Attr("data-studio-panel", "style"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-look-group-active", core.WorkbenchViewString(view, "defaultGroupKey")),
		gosx.Attr("data-gosx-studio-look-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

func renderLookPanelHead(view map[string]any) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-look-panel__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(view, "summary"))),
		),
	)
}

func renderLookPanelGroupInput(group map[string]any) gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("class", "studio-look-panel__group-input"),
		gosx.Attr("id", core.WorkbenchViewString(group, "inputID")),
		gosx.Attr("type", "radio"),
		gosx.Attr("name", "studioLookGroup"),
		gosx.Attr("value", core.WorkbenchViewString(group, "key")),
		gosx.Attr("checked", core.WorkbenchViewBool(group, "selected")),
		gosx.Attr("aria-label", core.WorkbenchViewString(group, "label")),
	))
}

func renderLookPanelGroups(view map[string]any) gosx.Node {
	groups := core.WorkbenchViewMapList(view, "groups")
	children := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "studio-look-panel__group"),
			gosx.Attr("for", core.WorkbenchViewString(group, "inputID")),
			gosx.Attr("data-studio-look-group-label", core.WorkbenchViewString(group, "key")),
		),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(group, "label"))),
			gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(group, "summary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-look-panel__groups"),
		gosx.Attr("role", "radiogroup"),
		gosx.Attr("aria-label", core.WorkbenchViewString(view, "groupLabel")),
	), gosx.Fragment(children...))
}

func renderLookPanelThemeSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "theme", "Theme", "Starter kit and template.",
		renderLookChoiceGroup(core.WorkbenchViewMap(settings, "kitGroup")),
		renderLookChoiceGroup(core.WorkbenchViewMap(settings, "templateGroup")),
	)
}

func renderLookPanelLayoutSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "layout", "Layout", "Hero shape, width, density, and rhythm.",
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(settings, "customBuilderClass")),
			gosx.Attr("data-custom-template-builder", "true"),
		),
			renderLookField(core.WorkbenchViewMap(settings, "customNameField")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-control-grid")),
				gosx.Fragment(renderLookSelectControls(core.WorkbenchViewMapList(settings, "customControls"))...),
			),
		),
	)
}

func renderLookPanelColorSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "color", "Color", "Palette and editable color tokens.",
		renderLookSelectControl(core.WorkbenchViewMap(settings, "palette")),
		renderLookSwatches(core.WorkbenchViewMapList(settings, "swatches")),
	)
}

func renderLookPanelDetailsSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "details", "Details", "Navigation, buttons, cards, images, spacing, and motion.",
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-control-grid")),
			gosx.Fragment(renderLookSelectControls(core.WorkbenchViewMapList(settings, "styleControls"))...),
		),
		renderLookRadioGroup(core.WorkbenchViewMap(settings, "imageCrop")),
	)
}

func renderLookPanelGroupSlot(view map[string]any, key, title, summary string, children ...gosx.Node) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-look-panel__group-slot"),
		gosx.Attr("data-studio-look-group-slot", key),
		gosx.Attr("data-studio-look-group-selected", core.BoolAttr(core.WorkbenchViewString(view, "defaultGroupKey") == key)),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text(title)),
			gosx.El("p", nil, gosx.Text(summary)),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-look-panel__group-body"),
			gosx.Attr("data-studio-look-group-body", key),
		), gosx.Fragment(children...)),
	)
}

func renderLookStyleScope(scope map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(scope, "class")),
		gosx.Attr("data-studio-style-scope", "true"),
		gosx.Attr("data-style-valid", core.BoolAttr(core.WorkbenchViewBool(scope, "valid"))),
		gosx.Attr("aria-label", "Active style scope"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-scope__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Scope")),
				gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-style-scope-block", "true")), gosx.Text(core.WorkbenchViewString(scope, "blockLabel"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-validity", "true")), gosx.Text(core.WorkbenchViewString(scope, "validityLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-scope__grid")),
			renderLookScopeField("System", core.WorkbenchViewString(scope, "systemID"), "data-studio-style-system-label"),
			renderLookScopeField("Page", core.WorkbenchViewString(scope, "pageLabel"), ""),
			renderLookScopeField("Field", core.WorkbenchViewString(scope, "fieldLabel"), "data-studio-style-scope-field"),
			renderLookScopeField("Breakpoint", core.WorkbenchViewString(scope, "breakpointLabel"), "data-studio-style-breakpoint-label"),
		),
		renderLookStyleStatebar(scope),
	)
}

func renderLookScopeField(label, value, attr string) gosx.Node {
	attrs := []any{}
	if attr != "" {
		attrs = append(attrs, gosx.Attr(attr, "true"))
	}
	// #15 — long scope values ("SYSTEM muddy-no…", "DEFAU…") clip under the
	// grid's min-width + ellipsis truncation (see studio.css
	// .studio-style-scope__grid output); a title tooltip keeps the full value
	// discoverable on hover/focus without widening the card.
	if value != "" {
		attrs = append(attrs, gosx.Attr("title", value))
	}
	return gosx.El("span", nil,
		gosx.El("small", nil, gosx.Text(label)),
		gosx.El("output", gosx.Attrs(attrs...), gosx.Text(value)),
	)
}

func renderLookStyleStatebar(scope map[string]any) gosx.Node {
	children := []gosx.Node{}
	for _, state := range core.WorkbenchViewMapList(scope, "states") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-style-state", core.WorkbenchViewString(state, "key")),
			gosx.Attr("aria-pressed", core.WorkbenchViewBool(state, "active")),
		), gosx.Text(core.WorkbenchViewString(state, "label"))))
	}
	children = append(children, gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-state-label", "true")), gosx.Text(core.WorkbenchViewString(scope, "activeState"))))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-style-statebar"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Style state"),
	), gosx.Fragment(children...))
}

func renderLookStyleWorkbench(workbench map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(workbench, "kicker"))),
			gosx.El("h3", nil, gosx.Text(core.WorkbenchViewString(workbench, "title"))),
		),
		renderLookStyleImpact(core.WorkbenchViewMap(workbench, "impact")),
	}
	recipeNodes := []gosx.Node{}
	for _, recipe := range core.WorkbenchViewMapList(workbench, "groups") {
		recipeNodes = append(recipeNodes, renderLookStyleRecipe(recipe))
	}
	children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-recipe-grid")), gosx.Fragment(recipeNodes...)))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(workbench, "class")),
		gosx.Attr("data-studio-style-workbench", "true"),
	), gosx.Fragment(children...))
}

// renderLookStyleImpact renders the Recipes grid's "Preview impact" card.
//
// Wave 3B (tamarind, residual #6 — LOOK Recipes card crowding): live against
// the ~20rem right rail (--size-studio-right), the impact label/summary
// column and the count/scope/state chip column were crowding each other —
// confirmed live: "No…" (an already-truncated label) wrapped against
// "Awaiting / scoped / change." across three cramped lines beside
// full-width "0 AFFECTED" / "SITE > HOME > HERO" / "DEFAULT / DESKTOP"
// chips. Mirrors the #15 fix already applied to renderLookScopeField: every
// value that can truncate under the grid's min-width/ellipsis rules
// (studio.css/site.css) gets a title tooltip so the full text stays
// discoverable on hover/focus without widening the card; the width-budget
// half of the fix (min-width for the text column, a tighter chip max-width)
// lives in host CSS (muddy-noni-commerce's public/site.css
// .studio-style-impact / .studio-style-impact__meta rules).
func renderLookStyleImpact(impact map[string]any) gosx.Node {
	label := core.WorkbenchViewString(impact, "label")
	summary := core.WorkbenchViewString(impact, "summary")
	countLabel := core.WorkbenchViewString(impact, "countLabel")
	scopeLabel := core.WorkbenchViewString(impact, "scopeLabel")
	stateLabel := core.WorkbenchViewString(impact, "stateLabel")
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-style-impact"),
		gosx.Attr("data-studio-style-impact-panel", "true"),
		gosx.Attr("aria-live", "polite"),
	),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(impact, "kicker"))),
			gosx.El("strong", gosx.Attrs(lookTooltipAttrs("data-studio-style-impact-label", label)...), gosx.Text(label)),
			gosx.El("p", gosx.Attrs(lookTooltipAttrs("data-studio-style-impact-summary", summary)...), gosx.Text(summary)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-impact__meta")),
			gosx.El("output", gosx.Attrs(lookTooltipAttrs("data-studio-style-impact-count", countLabel)...), gosx.Text(countLabel)),
			gosx.El("span", gosx.Attrs(lookTooltipAttrs("data-studio-style-impact-scope", scopeLabel)...), gosx.Text(scopeLabel)),
			gosx.El("span", gosx.Attrs(lookTooltipAttrs("data-studio-style-impact-state", stateLabel)...), gosx.Text(stateLabel)),
		),
	)
}

// lookTooltipAttrs builds the marker attribute plus a title tooltip carrying
// the full value when non-empty (see renderLookStyleImpact/
// renderLookScopeField's identical #15 pattern).
func lookTooltipAttrs(markerAttr, value string) []any {
	attrs := []any{gosx.Attr(markerAttr, "true")}
	if value != "" {
		attrs = append(attrs, gosx.Attr("title", value))
	}
	return attrs
}

// lookChoiceButtonAttrs builds a segmented-choice option button's attrs
// (Hero shape's EDITORIAL/OVERLAY/SPLIT, Width's NARROW/STANDARD/WIDE,
// etc.) plus a title tooltip mirroring its own label, unless the host
// already supplied one. handoff-4 (text truncation sweep): these buttons
// sit in a fixed 3-column grid (muddy-noni-commerce's public/site.css
// .studio-style-choice-row) narrow enough that "IMMERSIVE"/"STANDARD"
// clip to "IMMER…"/"STAND…" -- the accompanying CSS fix lets them wrap
// onto two lines, and this tooltip covers hosts/viewports where a label
// still doesn't fit.
func lookChoiceButtonAttrs(values map[string]any, label string) []any {
	attrs := BlockLibraryPanelMapAttrs(values)
	if label == "" {
		return attrs
	}
	if _, hasTitle := values["title"]; hasTitle {
		return attrs
	}
	return append(attrs, gosx.Attr("title", label))
}

func renderLookStyleRecipe(recipe map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-recipe-card__head")),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(recipe, "label"))),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-readout", core.WorkbenchViewString(core.WorkbenchViewMap(recipe, "readout"), "field"))), gosx.Text(core.WorkbenchViewString(core.WorkbenchViewMap(recipe, "readout"), "label"))),
		),
		renderLookStyleVisual(recipe),
	}
	for _, control := range core.WorkbenchViewMapList(recipe, "controls") {
		children = append(children, renderLookStyleControlGroup(control))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-style-recipe-card"),
		gosx.Attr("data-studio-style-recipe", core.WorkbenchViewString(recipe, "key")),
	), gosx.Fragment(children...))
}

func renderLookStyleVisual(recipe map[string]any) gosx.Node {
	marks := core.WorkbenchViewMapList(recipe, "marks")
	children := make([]gosx.Node, 0, len(marks))
	for _, mark := range marks {
		children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-style-mark", core.WorkbenchViewString(mark, "key")))))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(recipe, "visualFullClass")),
		gosx.Attr("aria-hidden", "true"),
	), gosx.Fragment(children...))
}

func renderLookStyleControlGroup(control map[string]any) gosx.Node {
	options := core.WorkbenchViewMapList(control, "options")
	children := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		label := core.WorkbenchViewString(option, "label")
		children = append(children, gosx.El("button", gosx.Attrs(lookChoiceButtonAttrs(core.WorkbenchViewMap(option, "attrs"), label)...), gosx.Text(label)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-control-group")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-control-group__head")),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(control, "label"))),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("data-studio-style-reset", core.WorkbenchViewString(control, "fieldName")),
			), gosx.Text("Reset")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-choice-row")), gosx.Fragment(children...)),
	)
}

func renderLookChoiceGroup(group map[string]any) gosx.Node {
	cards := core.WorkbenchViewMapList(group, "cards")
	children := make([]gosx.Node, 0, len(cards))
	for _, card := range cards {
		children = append(children, gosx.El("label", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(card, "cardAttrs"))...),
			gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(card, "inputAttrs"))...)),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(card, "label"))),
				gosx.El("small", nil, gosx.Text(core.WorkbenchViewString(card, "summary"))),
			),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(group, "class"))),
		gosx.El("legend", nil, gosx.Text(core.WorkbenchViewString(group, "legend"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(group, "gridClass"))), gosx.Fragment(children...)),
	)
}

func renderLookField(field map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", core.WorkbenchViewString(field, "id"))), gosx.Text(core.WorkbenchViewString(field, "label"))),
	}
	if core.WorkbenchViewBool(field, "isArea") {
		children = append(children, gosx.El("textarea", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(field, "controlAttrs"))...), gosx.Text(core.WorkbenchViewString(field, "value"))))
	}
	if core.WorkbenchViewBool(field, "isSelect") {
		children = append(children, renderLookSelectElement(field))
	}
	if core.WorkbenchViewBool(field, "isCheckbox") || core.WorkbenchViewBool(field, "isInput") {
		children = append(children, gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(field, "controlAttrs"))...)))
	}
	if core.WorkbenchViewBool(field, "hasHelp") {
		children = append(children, gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(core.WorkbenchViewString(field, "help"))))
	}
	return gosx.El("div", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(field, "rowAttrs"))...), gosx.Fragment(children...))
}

func renderLookSelectControls(controls []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(controls))
	for _, control := range controls {
		nodes = append(nodes, renderLookSelectControl(control))
	}
	return nodes
}

func renderLookSelectControl(control map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(control, "rowAttrs"))...),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", core.WorkbenchViewString(control, "id"))), gosx.Text(core.WorkbenchViewString(control, "label"))),
		renderLookSelectElement(control),
	)
}

func renderLookSelectElement(control map[string]any) gosx.Node {
	options := core.WorkbenchViewMapList(control, "options")
	children := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		children = append(children, gosx.El("option", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(option, "attrs"))...), gosx.Text(core.WorkbenchViewString(option, "label"))))
	}
	return gosx.El("select", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(control, "controlAttrs"))...), gosx.Fragment(children...))
}

func renderLookRadioGroup(group map[string]any) gosx.Node {
	options := core.WorkbenchViewMapList(group, "options")
	children := []gosx.Node{gosx.El("legend", nil, gosx.Text(core.WorkbenchViewString(group, "legend")))}
	for _, option := range options {
		children = append(children, gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(option, "attrs"))...)),
			gosx.Text(" "),
			gosx.Text(core.WorkbenchViewString(option, "label")),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(BlockLibraryPanelMapAttrs(core.WorkbenchViewMap(group, "attrs"))...), gosx.Fragment(children...))
}

func renderLookSwatches(tokens []map[string]any) gosx.Node {
	children := make([]gosx.Node, 0, len(tokens))
	for _, token := range tokens {
		children = append(children, gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "theme-swatch"),
			gosx.Attr("style", core.WorkbenchViewString(token, "style")),
			gosx.Attr("title", core.WorkbenchViewString(token, "label")),
		)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "theme-swatch-row"),
		gosx.Attr("data-editor-theme-swatches", "true"),
	), gosx.Fragment(children...))
}
