package panels

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

type LookPanelOptions struct {
	RootAttrs map[string]any
}

func RenderLookPanel(view map[string]any, options LookPanelOptions) gosx.Node {
	settings := workbenchViewMap(view, "settings")
	children := []gosx.Node{renderLookPanelHead(view)}
	for _, group := range workbenchViewMapList(view, "groups") {
		children = append(children, renderLookPanelGroupInput(group))
	}
	children = append(children,
		renderLookPanelGroups(view),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-look-panel__scope"),
			gosx.Attr("data-studio-look-scope", "true"),
		),
			renderLookStyleScope(workbenchViewMap(settings, "scope")),
			renderLookStyleWorkbench(workbenchViewMap(settings, "workbench")),
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
		gosx.Attr("data-studio-look-group-active", workbenchMapString(view, "defaultGroupKey")),
		gosx.Attr("data-gosx-studio-look-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

func renderLookPanelHead(view map[string]any) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-look-panel__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(view, "summary"))),
		),
	)
}

func renderLookPanelGroupInput(group map[string]any) gosx.Node {
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("class", "studio-look-panel__group-input"),
		gosx.Attr("id", workbenchMapString(group, "inputID")),
		gosx.Attr("type", "radio"),
		gosx.Attr("name", "studioLookGroup"),
		gosx.Attr("value", workbenchMapString(group, "key")),
		gosx.Attr("checked", workbenchMapBool(group, "selected")),
		gosx.Attr("aria-label", workbenchMapString(group, "label")),
	))
}

func renderLookPanelGroups(view map[string]any) gosx.Node {
	groups := workbenchViewMapList(view, "groups")
	children := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		children = append(children, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "studio-look-panel__group"),
			gosx.Attr("for", workbenchMapString(group, "inputID")),
			gosx.Attr("data-studio-look-group-label", workbenchMapString(group, "key")),
		),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(group, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(group, "summary"))),
		))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-look-panel__groups"),
		gosx.Attr("role", "radiogroup"),
		gosx.Attr("aria-label", workbenchMapString(view, "groupLabel")),
	), gosx.Fragment(children...))
}

func renderLookPanelThemeSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "theme", "Theme", "Starter kit and template.",
		renderLookChoiceGroup(workbenchViewMap(settings, "kitGroup")),
		renderLookChoiceGroup(workbenchViewMap(settings, "templateGroup")),
	)
}

func renderLookPanelLayoutSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "layout", "Layout", "Hero shape, width, density, and rhythm.",
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(settings, "customBuilderClass")),
			gosx.Attr("data-custom-template-builder", "true"),
		),
			renderLookField(workbenchViewMap(settings, "customNameField")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-control-grid")),
				gosx.Fragment(renderLookSelectControls(workbenchViewMapList(settings, "customControls"))...),
			),
		),
	)
}

func renderLookPanelColorSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "color", "Color", "Palette and editable color tokens.",
		renderLookSelectControl(workbenchViewMap(settings, "palette")),
		renderLookSwatches(workbenchViewMapList(settings, "swatches")),
	)
}

func renderLookPanelDetailsSlot(view, settings map[string]any) gosx.Node {
	return renderLookPanelGroupSlot(view, "details", "Details", "Navigation, buttons, cards, images, spacing, and motion.",
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-control-grid")),
			gosx.Fragment(renderLookSelectControls(workbenchViewMapList(settings, "styleControls"))...),
		),
		renderLookRadioGroup(workbenchViewMap(settings, "imageCrop")),
	)
}

func renderLookPanelGroupSlot(view map[string]any, key, title, summary string, children ...gosx.Node) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-look-panel__group-slot"),
		gosx.Attr("data-studio-look-group-slot", key),
		gosx.Attr("data-studio-look-group-selected", core.BoolAttr(workbenchMapString(view, "defaultGroupKey") == key)),
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
		gosx.Attr("class", workbenchMapString(scope, "class")),
		gosx.Attr("data-studio-style-scope", "true"),
		gosx.Attr("data-style-valid", core.BoolAttr(workbenchMapBool(scope, "valid"))),
		gosx.Attr("aria-label", "Active style scope"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-scope__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Scope")),
				gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-style-scope-block", "true")), gosx.Text(workbenchMapString(scope, "blockLabel"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-validity", "true")), gosx.Text(workbenchMapString(scope, "validityLabel"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-scope__grid")),
			renderLookScopeField("System", workbenchMapString(scope, "systemID"), "data-studio-style-system-label"),
			renderLookScopeField("Page", workbenchMapString(scope, "pageLabel"), ""),
			renderLookScopeField("Field", workbenchMapString(scope, "fieldLabel"), "data-studio-style-scope-field"),
			renderLookScopeField("Breakpoint", workbenchMapString(scope, "breakpointLabel"), "data-studio-style-breakpoint-label"),
		),
		renderLookStyleStatebar(scope),
	)
}

func renderLookScopeField(label, value, attr string) gosx.Node {
	attrs := []any{}
	if attr != "" {
		attrs = append(attrs, gosx.Attr(attr, "true"))
	}
	return gosx.El("span", nil,
		gosx.El("small", nil, gosx.Text(label)),
		gosx.El("output", gosx.Attrs(attrs...), gosx.Text(value)),
	)
}

func renderLookStyleStatebar(scope map[string]any) gosx.Node {
	children := []gosx.Node{}
	for _, state := range workbenchViewMapList(scope, "states") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-style-state", workbenchMapString(state, "key")),
			gosx.Attr("aria-pressed", workbenchMapBool(state, "active")),
		), gosx.Text(workbenchMapString(state, "label"))))
	}
	children = append(children, gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-state-label", "true")), gosx.Text(workbenchMapString(scope, "activeState"))))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-style-statebar"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Style state"),
	), gosx.Fragment(children...))
}

func renderLookStyleWorkbench(workbench map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(workbench, "kicker"))),
			gosx.El("h3", nil, gosx.Text(workbenchMapString(workbench, "title"))),
		),
		renderLookStyleImpact(workbenchViewMap(workbench, "impact")),
	}
	recipeNodes := []gosx.Node{}
	for _, recipe := range workbenchViewMapList(workbench, "groups") {
		recipeNodes = append(recipeNodes, renderLookStyleRecipe(recipe))
	}
	children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-recipe-grid")), gosx.Fragment(recipeNodes...)))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(workbench, "class")),
		gosx.Attr("data-studio-style-workbench", "true"),
	), gosx.Fragment(children...))
}

func renderLookStyleImpact(impact map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-style-impact"),
		gosx.Attr("data-studio-style-impact-panel", "true"),
		gosx.Attr("aria-live", "polite"),
	),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(impact, "kicker"))),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-style-impact-label", "true")), gosx.Text(workbenchMapString(impact, "label"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-style-impact-summary", "true")), gosx.Text(workbenchMapString(impact, "summary"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-impact__meta")),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-impact-count", "true")), gosx.Text(workbenchMapString(impact, "countLabel"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-style-impact-scope", "true")), gosx.Text(workbenchMapString(impact, "scopeLabel"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-style-impact-state", "true")), gosx.Text(workbenchMapString(impact, "stateLabel"))),
		),
	)
}

func renderLookStyleRecipe(recipe map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-recipe-card__head")),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(recipe, "label"))),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-readout", workbenchMapString(workbenchViewMap(recipe, "readout"), "field"))), gosx.Text(workbenchMapString(workbenchViewMap(recipe, "readout"), "label"))),
		),
		renderLookStyleVisual(recipe),
	}
	for _, control := range workbenchViewMapList(recipe, "controls") {
		children = append(children, renderLookStyleControlGroup(control))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-style-recipe-card"),
		gosx.Attr("data-studio-style-recipe", workbenchMapString(recipe, "key")),
	), gosx.Fragment(children...))
}

func renderLookStyleVisual(recipe map[string]any) gosx.Node {
	marks := workbenchViewMapList(recipe, "marks")
	children := make([]gosx.Node, 0, len(marks))
	for _, mark := range marks {
		children = append(children, gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-style-mark", workbenchMapString(mark, "key")))))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(recipe, "visualFullClass")),
		gosx.Attr("aria-hidden", "true"),
	), gosx.Fragment(children...))
}

func renderLookStyleControlGroup(control map[string]any) gosx.Node {
	options := workbenchViewMapList(control, "options")
	children := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		children = append(children, gosx.El("button", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(option, "attrs"))...), gosx.Text(workbenchMapString(option, "label"))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-control-group")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-control-group__head")),
			gosx.El("span", nil, gosx.Text(workbenchMapString(control, "label"))),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("data-studio-style-reset", workbenchMapString(control, "fieldName")),
			), gosx.Text("Reset")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-choice-row")), gosx.Fragment(children...)),
	)
}

func renderLookChoiceGroup(group map[string]any) gosx.Node {
	cards := workbenchViewMapList(group, "cards")
	children := make([]gosx.Node, 0, len(cards))
	for _, card := range cards {
		children = append(children, gosx.El("label", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(card, "cardAttrs"))...),
			gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(card, "inputAttrs"))...)),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(card, "label"))),
				gosx.El("small", nil, gosx.Text(workbenchMapString(card, "summary"))),
			),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", workbenchMapString(group, "class"))),
		gosx.El("legend", nil, gosx.Text(workbenchMapString(group, "legend"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(group, "gridClass"))), gosx.Fragment(children...)),
	)
}

func renderLookField(field map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(field, "id"))), gosx.Text(workbenchMapString(field, "label"))),
	}
	if workbenchMapBool(field, "isArea") {
		children = append(children, gosx.El("textarea", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(field, "controlAttrs"))...), gosx.Text(workbenchMapString(field, "value"))))
	}
	if workbenchMapBool(field, "isSelect") {
		children = append(children, renderLookSelectElement(field))
	}
	if workbenchMapBool(field, "isCheckbox") || workbenchMapBool(field, "isInput") {
		children = append(children, gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(field, "controlAttrs"))...)))
	}
	if workbenchMapBool(field, "hasHelp") {
		children = append(children, gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(workbenchMapString(field, "help"))))
	}
	return gosx.El("div", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(field, "rowAttrs"))...), gosx.Fragment(children...))
}

func renderLookSelectControls(controls []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(controls))
	for _, control := range controls {
		nodes = append(nodes, renderLookSelectControl(control))
	}
	return nodes
}

func renderLookSelectControl(control map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(control, "rowAttrs"))...),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(control, "id"))), gosx.Text(workbenchMapString(control, "label"))),
		renderLookSelectElement(control),
	)
}

func renderLookSelectElement(control map[string]any) gosx.Node {
	options := workbenchViewMapList(control, "options")
	children := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		children = append(children, gosx.El("option", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(option, "attrs"))...), gosx.Text(workbenchMapString(option, "label"))))
	}
	return gosx.El("select", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(control, "controlAttrs"))...), gosx.Fragment(children...))
}

func renderLookRadioGroup(group map[string]any) gosx.Node {
	options := workbenchViewMapList(group, "options")
	children := []gosx.Node{gosx.El("legend", nil, gosx.Text(workbenchMapString(group, "legend")))}
	for _, option := range options {
		children = append(children, gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(option, "attrs"))...)),
			gosx.Text(" "),
			gosx.Text(workbenchMapString(option, "label")),
		))
	}
	return gosx.El("fieldset", gosx.Attrs(BlockLibraryPanelMapAttrs(workbenchViewMap(group, "attrs"))...), gosx.Fragment(children...))
}

func renderLookSwatches(tokens []map[string]any) gosx.Node {
	children := make([]gosx.Node, 0, len(tokens))
	for _, token := range tokens {
		children = append(children, gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "theme-swatch"),
			gosx.Attr("style", workbenchMapString(token, "style")),
			gosx.Attr("title", workbenchMapString(token, "label")),
		)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "theme-swatch-row"),
		gosx.Attr("data-editor-theme-swatches", "true"),
	), gosx.Fragment(children...))
}
