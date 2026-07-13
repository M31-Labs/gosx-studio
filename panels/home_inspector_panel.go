package panels

import (
	"strings"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

type HomeInspectorPanelOptions struct {
	RootAttrs map[string]any
}

func RenderHomeInspectorPanel(view map[string]any, contentFields map[string]any, options HomeInspectorPanelOptions) gosx.Node {
	// #12/#13 — one save-state vocabulary: default to "Saved"/"Unsaved
	// changes" (matching shell/workbench_toolbar.go's top-bar save status)
	// rather than leaving this caller-supplied label blank when the host
	// omits it, so the right rail never invents its own word (e.g. "Ready")
	// for the same concept the top bar calls "Saved". Wave 3A (vocabulary
	// pruning, fix (e)): the FirstNonEmpty guard above only ever caught an
	// EMPTY host label — muddy-noni-commerce's editorHomeInspectorView
	// supplies a non-empty "Ready" unconditionally, so the "right rail never
	// invents 'Ready'" intent never actually fired. homeInspectorCleanLabel
	// now also folds that specific known synonym into "Saved" while still
	// passing through any other host-supplied label untouched.
	cleanLabel := homeInspectorCleanLabel(core.WorkbenchViewString(view, "cleanLabel"))
	return gosx.El("section", gosx.Attrs(homeInspectorPanelRootAttrs(view, options.RootAttrs)...),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-home-inspector__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
				gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
			),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("class", "studio-home-inspector__state"),
				gosx.Attr("data-studio-inspector-state", cleanLabel),
			), gosx.Text(cleanLabel)),
			gosx.El("label", gosx.Attrs(
				gosx.Attr("class", "studio-home-inspector__close"),
				gosx.Attr("for", "studioSiteMapInspectorFocus"),
				gosx.Attr("role", "button"),
				gosx.Attr("tabindex", "0"),
			), gosx.Text("Close")),
		),
		renderHomeInspectorTarget(view),
		renderHomeInspectorGroups(view),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-home-inspector__fields"),
			gosx.Attr("data-studio-home-inspector-fields", "true"),
		), renderHomeInspectorFieldPanel(contentFields)),
	)
}

// homeInspectorCleanLabel folds the one known discouraged synonym ("Ready")
// into the canonical "Saved" word the rest of the workbench (toolbar save
// status, dirtyLabel's "Unsaved changes") already uses, in addition to
// covering the empty-input case. See the call sites' comments.
func homeInspectorCleanLabel(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.EqualFold(trimmed, "ready") {
		return "Saved"
	}
	return trimmed
}

func homeInspectorPanelRootAttrs(view map[string]any, rootAttrs map[string]any) []any {
	attrs := []any{
		gosx.Attr("id", "studioHomeInspector"),
		gosx.Attr("class", "editor-panel editor-panel--home-inspector studio-home-inspector"),
		gosx.Attr("data-studio-home-inspector-panel", "true"),
		gosx.Attr("data-studio-mode-panel", "home"),
		gosx.Attr("data-studio-panel", "inspector"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-inspector-group-active", core.WorkbenchViewString(view, "defaultGroupKey")),
		gosx.Attr("data-studio-inspector-dirty", "false"),
		gosx.Attr("data-studio-inspector-clean-label", homeInspectorCleanLabel(core.WorkbenchViewString(view, "cleanLabel"))),
		gosx.Attr("data-studio-inspector-dirty-label", core.FirstNonEmpty(core.WorkbenchViewString(view, "dirtyLabel"), "Unsaved changes")),
		gosx.Attr("data-gosx-studio-home-inspector-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, rootAttrs)
	return attrs
}

// renderHomeInspectorTarget no longer renders the separate "Live preview" /
// "Draft safe" status pills (wave 3A, vocabulary pruning fix (e)): the panel
// header already shows the ONE Saved/Unsaved-changes indicator
// (data-studio-inspector-state, see RenderHomeInspectorPanel above) — a
// second pair of pills for roughly the same "your edits are safe until
// published" concept, using different words, was pure vocabulary noise.
func renderHomeInspectorTarget(view map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-home-inspector__target"),
		gosx.Attr("aria-label", core.WorkbenchViewString(view, "targetLabel")),
	),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "targetKicker"))),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(core.WorkbenchViewString(view, "selectionLabel"))),
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(view, "targetSummary"))),
		),
	)
}

func renderHomeInspectorGroups(view map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-home-inspector__groups"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", core.WorkbenchViewString(view, "groupLabel")),
	),
		renderHomeInspectorGroupButton(view, "copy", "copyLabel"),
		renderHomeInspectorGroupButton(view, "media", "mediaLabel"),
		renderHomeInspectorGroupButton(view, "sources", "sourcesLabel"),
		renderHomeInspectorGroupButton(view, "all", "allLabel"),
	)
}

func renderHomeInspectorGroupButton(view map[string]any, key, labelKey string) gosx.Node {
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("type", "button"),
		gosx.Attr("class", "studio-home-inspector__group"),
		gosx.Attr("data-studio-inspector-group", key),
		gosx.Attr("aria-pressed", core.BoolAttr(core.WorkbenchViewString(view, "defaultGroupKey") == key)),
	), gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(view, labelKey))))
}

func renderHomeInspectorFieldPanel(contentFields map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "empty"),
			gosx.Attr("hidden", core.WorkbenchViewBool(contentFields, "hasFields")),
		), gosx.Text(core.WorkbenchViewString(contentFields, "empty"))),
	}
	if fieldList, ok := contentFields["fieldList"].(gosx.Node); ok && core.WorkbenchViewBool(contentFields, "hasFields") {
		children = append(children, fieldList)
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(contentFields, "class")),
		gosx.Attr("data-studio-home-fields-panel", "true"),
		gosx.Attr("data-panel-key", core.WorkbenchViewString(contentFields, "key")),
		gosx.Attr("data-studio-panel", core.WorkbenchViewString(contentFields, "key")),
		gosx.Attr("data-studio-mode-panel", core.WorkbenchViewString(contentFields, "mode")),
		gosx.Attr("data-studio-engine-source", "gosx"),
	), gosx.Fragment(children...))
}
