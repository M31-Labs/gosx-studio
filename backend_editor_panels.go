package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

// RenderPublishPanel renders the backend publish/review surface from host-owned
// view data. The host remains responsible for lifecycle actions and data loading.
func RenderPublishPanel(view map[string]any) gosx.Node {
	panel := backendSurfaceMap(view, "publishPanel")
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(panel, "kicker"))),
				gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "panelTitle"))),
				gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "summary"))),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(panel, "countLabel"))),
		),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("class", "studio-publish-panel__draft-status"),
			gosx.Attr("data-studio-publish-draft-status", "true"),
			gosx.Attr("data-studio-has-draft", BoolAttr(backendMapBool(panel, "hasDraft"))),
			gosx.Attr("hidden", !backendMapBool(panel, "hasDraft")),
		), gosx.Text(workbenchMapString(panel, "draftStatusLabel"))),
		renderPublishActions(panel),
	}
	if backendMapBool(panel, "hasPendingChanges") {
		children = append(children, renderPublishPending(panel))
	}
	children = append(children,
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__preview-share"), gosx.Attr("data-studio-publish-preview-share", "true")),
			gosx.El("header", nil, gosx.El("h3", nil, gosx.Text("Share preview")), gosx.El("p", nil, gosx.Text("Send a draft preview for review without giving admin access."))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__preview-share-slot"), gosx.Attr("data-studio-publish-preview-share-slot", "true")), RenderPreviewSharePanel(view)),
		),
		renderPublishSchedule(panel),
		renderPublishDecision(panel),
		renderPublishSummary(panel),
	)
	if backendMapBool(panel, "hasChecks") {
		children = append(children, renderPublishChecks(panel))
	}
	if backendMapBool(panel, "hasImpacts") {
		children = append(children, renderPublishImpacts(panel))
	}
	children = append(children,
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__collaboration"), gosx.Attr("data-studio-publish-activity", "true")),
			gosx.El("header", nil, gosx.El("h3", nil, gosx.Text("Review activity")), gosx.El("p", nil, gosx.Text("Readiness, review notes, and suggested changes for this draft."))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__activity-slot"), gosx.Attr("data-studio-publish-activity-slot", "true")), RenderActivityPanel(view)),
		),
		gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__history"), gosx.Attr("data-studio-publish-history", "true")),
			gosx.El("header", nil, gosx.El("h3", nil, gosx.Text("Restore points")), gosx.El("p", nil, gosx.Text("Recent saved versions that can be restored before or after a release."))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__history-slot"), gosx.Attr("data-studio-publish-history-slot", "true")), RenderRevisionHistoryPanel(view)),
		),
	)
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "editor-panel editor-panel--publish-studio studio-publish-panel"),
		gosx.Attr("data-studio-publish-panel", "true"),
		gosx.Attr("data-studio-mode-panel", "publish"),
		gosx.Attr("data-studio-panel", "publish"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-publish-status", workbenchMapString(panel, "status")),
	), gosx.Fragment(children...))
}

func RenderRevisionHistoryPanel(view map[string]any) gosx.Node {
	panel := backendSurfaceMap(view, "revisionHistory")
	items := backendMapList(panel, "items")
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "headerClass"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title")))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty"), gosx.Attr("hidden", backendMapBool(panel, "hasItems"))), gosx.Text(workbenchMapString(panel, "empty"))),
	}
	rows := make([]gosx.Node, 0, len(items))
	for _, revision := range items {
		rows = append(rows, renderRevisionHistoryItem(revision))
	}
	children = append(children, gosx.El("ul", gosx.Attrs(
		gosx.Attr("class", "field-list field-list--stacked"),
		gosx.Attr("hidden", !backendMapBool(panel, "hasItems")),
	), gosx.Fragment(rows...)))
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(panel, "class")),
		gosx.Attr("data-panel-key", workbenchMapString(panel, "panelKey")),
		gosx.Attr("data-studio-revision-history", "true"),
	), gosx.Fragment(children...))
}

func RenderPreviewSharePanel(view map[string]any) gosx.Node {
	panel := backendSurfaceMap(view, "previewShare")
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-preview-share__head")),
			gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-preview-share__kicker")), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title")))),
			gosx.El("output", nil, gosx.Text(workbenchMapString(panel, "status"))),
		),
	}
	if !backendMapBool(panel, "hasHref") {
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("class", "studio-preview-share__empty")), gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "emptyTitle"))), gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "emptyDetail")))))
	} else {
		children = append(children,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-preview-share__detail")), gosx.Text(workbenchMapString(panel, "detail"))),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-preview-share__field")),
				gosx.El("span", nil, gosx.Text(workbenchMapString(panel, "inputLabel"))),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "url"), gosx.Attr("readonly", "readonly"), gosx.Attr("value", workbenchMapString(panel, "href")), gosx.Attr("data-studio-preview-url", "true"), gosx.Attr("aria-label", workbenchMapString(panel, "inputLabel")))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-preview-share__actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-studio-copy-target", "[data-studio-preview-url]")), gosx.Text(workbenchMapString(panel, "copyLabel"))),
				gosx.El("a", gosx.Attrs(gosx.Attr("href", workbenchMapString(panel, "href")), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noreferrer")), gosx.Text(workbenchMapString(panel, "openLabel"))),
			),
			gosx.El("dl", gosx.Attrs(gosx.Attr("class", "studio-preview-share__meta")),
				gosx.El("dt", nil, gosx.Text("Resource")), gosx.El("dd", nil, gosx.Text(workbenchMapString(panel, "resource"))),
				gosx.El("dt", nil, gosx.Text("Audience")), gosx.El("dd", nil, gosx.Text(workbenchMapString(panel, "audience"))),
				gosx.El("dt", nil, gosx.Text("Expires")), gosx.El("dd", nil, gosx.Text(workbenchMapString(panel, "expiresLabel"))),
			),
		)
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class")), gosx.Attr("data-studio-preview-share", "true"), gosx.Attr("data-studio-preview-share-state", workbenchMapString(panel, "state"))), gosx.Fragment(children...))
}

func RenderActivityPanel(view map[string]any) gosx.Node {
	panel := backendSurfaceMap(view, "activityPanel")
	readiness := workbenchViewMap(panel, "readiness")
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-activity-head")),
			gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title")))),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-readiness-score")), gosx.Text(workbenchMapString(panel, "score"))),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-studio-activity-toggle", "true"), gosx.Attr("aria-pressed", workbenchMapString(panel, "togglePressed"))), gosx.Text(workbenchMapString(panel, "toggleLabel"))),
		),
		renderReadinessSummary(readiness),
		renderReadinessList(readiness),
	}
	if backendMapBool(panel, "hasHealth") {
		children = append(children, renderActivityHealthPanel(workbenchViewMap(panel, "health")))
	}
	if backendMapBool(panel, "hasPerformance") {
		children = append(children, renderActivityPerformancePanel(workbenchViewMap(panel, "performance")))
	}
	children = append(children, renderActivityEmptyPanel(workbenchViewMap(panel, "comments")), renderActivityEmptyPanel(workbenchViewMap(panel, "proposals")))
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class")), gosx.Attr("data-studio-activity-drawer", "true"), gosx.Attr("aria-label", "Activity and readiness")), gosx.Fragment(children...))
}

func RenderBrandPanel(view map[string]any) gosx.Node {
	panel := backendSurfaceMap(view, "brandPanel")
	fields := backendSurfaceMap(view, "brandFields")
	groups := backendMapList(panel, "groups")
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__head")),
			gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title"))), gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "summary")))),
		),
		renderBrandPreview(panel, fields),
	}
	children = append(children, renderRadioInputs(groups, "studio-brand-panel__group-input", "studioBrandGroup")...)
	children = append(children, renderBrandGroups(panel, groups))
	for _, group := range groups {
		children = append(children, renderBrandGroupSlot(group, fields))
	}
	children = append(children, gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__group-slot studio-brand-media-picker-slot"), gosx.Attr("data-studio-brand-group-slot", "files"), gosx.Attr("data-studio-brand-group-selected", BoolAttr(false))), RenderBrandMediaPicker(view)))
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "editor-panel editor-panel--brand-studio studio-brand-panel"),
		gosx.Attr("data-studio-brand-panel", "true"),
		gosx.Attr("data-studio-mode-panel", "brand"),
		gosx.Attr("data-studio-panel", "brand"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-brand-group-active", workbenchMapString(panel, "defaultGroupKey")),
	), gosx.Fragment(children...))
}

func RenderBrandMediaPicker(view map[string]any) gosx.Node {
	props := backendSurfaceMap(view, "mediaPicker")
	assets := backendMapList(props, "assets")
	cards := make([]gosx.Node, 0, len(assets))
	for _, asset := range assets {
		cards = append(cards, gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(asset, "cardClass")), gosx.Attr("data-studio-media-asset", workbenchMapString(asset, "id")), gosx.Attr("data-studio-media-filter-group", workbenchMapString(asset, "filterGroup"))),
			gosx.El("img", gosx.Attrs(gosx.Attr("src", workbenchMapString(asset, "url")), gosx.Attr("alt", workbenchMapString(asset, "alt")))),
			gosx.El("div", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(asset, "filename"))), gosx.El("span", nil, gosx.Text(workbenchMapString(asset, "statusLabel")))),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__actions"), gosx.Attr("aria-label", workbenchMapString(asset, "actionLabel"))),
				gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit"), gosx.Attr("form", workbenchMapString(props, "formID")), gosx.Attr("formaction", workbenchMapString(props, "saveAction")), gosx.Attr("name", workbenchMapString(props, "logoPickName")), gosx.Attr("value", workbenchMapString(asset, "url"))), gosx.Text("Use logo")),
				gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit"), gosx.Attr("form", workbenchMapString(props, "formID")), gosx.Attr("formaction", workbenchMapString(props, "saveAction")), gosx.Attr("name", workbenchMapString(props, "faviconPickName")), gosx.Attr("value", workbenchMapString(asset, "url"))), gosx.Text("Use icon")),
			),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker"), gosx.Attr("data-studio-media-picker-island", "brand"), gosx.Attr("data-studio-engine-source", "gosx"), gosx.Attr("data-studio-media-filter", "all")),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__head")), gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(props, "kicker"))), gosx.El("h3", nil, gosx.Text(workbenchMapString(props, "title"))), gosx.El("p", nil, gosx.Text(workbenchMapString(props, "summary")))), gosx.El("a", gosx.Attrs(gosx.Attr("href", workbenchMapString(props, "uploadHref")), gosx.Attr("data-gosx-link", "true")), gosx.Text("Upload"))),
		gosx.El("input", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__filter-input"), gosx.Attr("id", "studioBrandMediaFilterAll"), gosx.Attr("type", "radio"), gosx.Attr("name", "studioBrandMediaFilter"), gosx.Attr("value", "all"), gosx.Attr("checked", true), gosx.Attr("aria-label", "All"))),
		gosx.El("input", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__filter-input"), gosx.Attr("id", "studioBrandMediaFilterReady"), gosx.Attr("type", "radio"), gosx.Attr("name", "studioBrandMediaFilter"), gosx.Attr("value", "ready"), gosx.Attr("aria-label", "Ready"))),
		gosx.El("input", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__filter-input"), gosx.Attr("id", "studioBrandMediaFilterMissingAlt"), gosx.Attr("type", "radio"), gosx.Attr("name", "studioBrandMediaFilter"), gosx.Attr("value", "missing-alt"), gosx.Attr("aria-label", "Needs alt"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__filters"), gosx.Attr("role", "toolbar"), gosx.Attr("aria-label", workbenchMapString(props, "filterLabel"))),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "studioBrandMediaFilterAll"), gosx.Attr("role", "button"), gosx.Attr("data-studio-media-filter-option", "all"), gosx.Attr("aria-pressed", true)), gosx.Text("All")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "studioBrandMediaFilterReady"), gosx.Attr("role", "button"), gosx.Attr("data-studio-media-filter-option", "ready"), gosx.Attr("aria-pressed", false)), gosx.Text("Ready")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "studioBrandMediaFilterMissingAlt"), gosx.Attr("role", "button"), gosx.Attr("data-studio-media-filter-option", "missing-alt"), gosx.Attr("aria-pressed", false)), gosx.Text("Needs alt")),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__empty"), gosx.Attr("hidden", backendMapBool(props, "hasAssets"))), gosx.Text(workbenchMapString(props, "emptyLabel"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-media-picker__grid"), gosx.Attr("aria-label", workbenchMapString(props, "assetLabel"))), gosx.Fragment(cards...)),
	)
}

func RenderFlowDesignerPanel(view map[string]any) gosx.Node {
	props := backendSurfaceMap(view, "flowDesigner")
	flows := backendMapList(props, "flows")
	selected := workbenchMapString(props, "defaultFlowKey")
	items := make([]gosx.Node, 0, len(flows))
	for _, flow := range flows {
		key := workbenchMapString(flow, "key")
		isSelected := key == selected
		inputID := "studioFlowSelect-" + NormalizeKey(key)
		items = append(items, gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__item"), gosx.Attr("role", "listitem"), gosx.Attr("data-studio-flow-item", key)),
			gosx.El("input", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__input"), gosx.Attr("id", inputID), gosx.Attr("type", "radio"), gosx.Attr("name", "studioFlowDesignerFlow"), gosx.Attr("value", key), gosx.Attr("checked", isSelected), gosx.Attr("aria-label", workbenchMapString(flow, "label")))),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", inputID), gosx.Attr("class", workbenchMapString(flow, "cardClass")), gosx.Attr("data-editor-flow", key), gosx.Attr("data-studio-flow-card", key), gosx.Attr("data-studio-flow-card-selected", BoolAttr(isSelected)), gosx.Attr("data-studio-flow-route", workbenchMapString(flow, "route")), gosx.Attr("data-studio-flow-label", workbenchMapString(flow, "label")), gosx.Attr("data-studio-flow-status", workbenchMapString(flow, "statusLabel")), gosx.Attr("aria-pressed", isSelected)),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__head")), gosx.El("strong", nil, gosx.Text(workbenchMapString(flow, "label"))), gosx.El("output", nil, gosx.Text(workbenchMapString(flow, "statusLabel")))),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__body")), gosx.Text(workbenchMapString(flow, "description"))),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-card__meta")), gosx.El("span", nil, gosx.Text(workbenchMapString(flow, "summary"))), gosx.El("span", nil, gosx.Text(workbenchMapString(flow, "handlerStatusLabel")))),
				gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-flow-summary__dirty"), gosx.Attr("data-studio-flow-dirty-badge", key), gosx.Attr("data-studio-flow-dirty-visible", BoolAttr(false))), gosx.Text("Unsaved")),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__editors")), renderFlowEditor(props, flow, isSelected)),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "editor-panel editor-panel--flow-designer studio-flow-designer"), gosx.Attr("data-studio-mode-panel", "advanced"), gosx.Attr("data-studio-panel", "flow-designer"), gosx.Attr("data-studio-flow-designer-panel", "true"), gosx.Attr("data-studio-engine-source", "gosx"), gosx.Attr("data-studio-flow-selected", selected)),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")), gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Advanced")), gosx.El("h2", nil, gosx.Text("Form flows")), gosx.El("p", nil, gosx.Text("Build and publish the site forms people use to contact, request, subscribe, and continue checkout."))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__grid")),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-designer__list"), gosx.Attr("role", "list"), gosx.Attr("aria-label", "Form flows")), gosx.Fragment(items...)),
		),
	)
}

func RenderAdvancedPanel(view map[string]any) gosx.Node {
	panel := backendSurfaceMap(view, "advancedPanel")
	groups := backendMapList(panel, "groups")
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-advanced-panel__head")), gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title"))), gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "summary"))))),
	}
	children = append(children, renderRadioInputs(groups, "studio-advanced-panel__group-input", "studioAdvancedGroup")...)
	children = append(children, renderAdvancedGroups(panel, groups))
	for _, group := range groups {
		children = append(children, renderAdvancedGroupSlot(view, group))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "editor-panel editor-panel--advanced-studio studio-advanced-panel"), gosx.Attr("data-studio-advanced-panel", "true"), gosx.Attr("data-studio-mode-panel", "advanced"), gosx.Attr("data-studio-panel", "advanced"), gosx.Attr("data-studio-engine-source", "gosx"), gosx.Attr("data-studio-advanced-group-active", workbenchMapString(panel, "defaultGroupKey"))), gosx.Fragment(children...))
}

func renderPublishActions(panel map[string]any) gosx.Node {
	children := []gosx.Node{gosx.El("a", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("href", workbenchMapString(panel, "previewHref")), gosx.Attr("data-gosx-link", "true")), gosx.Text("Open preview"))}
	if backendMapBool(panel, "hasPublishAction") {
		children = append(children, submitActionButton("button button--primary", workbenchMapString(panel, "formID"), workbenchMapString(panel, "publishAction"), "publish", "Publish this draft?", "Publish"))
	}
	if backendMapBool(panel, "hasDiscardAction") {
		children = append(children, submitActionButton("button button--ghost", workbenchMapString(panel, "formID"), workbenchMapString(panel, "discardAction"), "discard", "Discard unpublished changes?", "Discard changes"))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__actions"), gosx.Attr("aria-label", "Publish actions")), gosx.Fragment(children...))
}

func renderPublishPending(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__pending"), gosx.Attr("aria-label", "Unpublished changes")),
		gosx.El("header", nil, gosx.El("h3", nil, gosx.Text("What will change")), gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "draftSummary")))),
		renderDiffList(backendMapList(panel, "pendingChanges")),
	)
}

func renderPublishSchedule(panel map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "field"), gosx.Attr("for", workbenchMapString(panel, "scheduleInputID"))), gosx.El("span", nil, gosx.Text("Schedule for")), gosx.El("input", gosx.Attrs(gosx.Attr("id", workbenchMapString(panel, "scheduleInputID")), gosx.Attr("name", workbenchMapString(panel, "scheduleInputName")), gosx.Attr("type", "datetime-local"), gosx.Attr("value", workbenchMapString(panel, "scheduleInputValue")), gosx.Attr("form", workbenchMapString(panel, "formID")), gosx.Attr("data-studio-field-source", "lifecycle.schedule.publishAt"), gosx.Attr("data-studio-field-editable", "lifecycle")))),
		gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "scheduleHelp"))),
	}
	if backendMapBool(panel, "hasScheduleAction") {
		children = append(children, submitActionButton("button button--secondary", workbenchMapString(panel, "formID"), workbenchMapString(panel, "scheduleAction"), "schedule", "Schedule this draft?", "Schedule"))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__schedule")), gosx.Fragment(children...))
}

func renderPublishDecision(panel map[string]any) gosx.Node {
	children := []gosx.Node{}
	if backendMapBool(panel, "hasApproval") {
		children = append(children, renderPublishDecisionCard(workbenchViewMap(panel, "approval"), true))
	}
	if backendMapBool(panel, "hasSchedule") {
		children = append(children, renderPublishDecisionCard(workbenchViewMap(panel, "schedule"), false))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__decision"), gosx.Attr("aria-label", "Release decision")), gosx.Fragment(children...))
}

func renderPublishDecisionCard(card map[string]any, withLink bool) gosx.Node {
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-head")), gosx.El("strong", nil, gosx.Text(workbenchMapString(card, "label"))), gosx.El("output", nil, gosx.Text(workbenchMapString(card, "statusLabel")))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-summary")), gosx.Text(workbenchMapString(card, "summary"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(workbenchMapString(card, "detail"))),
	}
	if withLink && backendMapBool(card, "hasHref") {
		children = append(children, gosx.El("a", gosx.Attrs(gosx.Attr("href", workbenchMapString(card, "href")), gosx.Attr("data-gosx-link", "true")), gosx.Text(workbenchMapString(card, "actionLabel"))))
	}
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(card, "class"))), gosx.Fragment(children...))
}

func renderPublishSummary(panel map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-review__summary"), gosx.Attr("aria-label", "Publish review summary")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "readyCountLabel"))), gosx.Text("clear")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "watchCountLabel"))), gosx.Text("watch")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "nextCountLabel"))), gosx.Text("next")),
	)
}

func renderPublishChecks(panel map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, check := range backendMapList(panel, "checks") {
		children := []gosx.Node{
			gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__card-head")), gosx.El("div", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(check, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(check, "scope")))), gosx.El("output", nil, gosx.Text(workbenchMapString(check, "statusLabel")))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__check-summary")), gosx.Text(workbenchMapString(check, "summary"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(workbenchMapString(check, "detail"))),
		}
		if backendMapBool(check, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(gosx.Attr("href", workbenchMapString(check, "href")), gosx.Attr("data-gosx-link", "true")), gosx.Text(workbenchMapString(check, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(check, "class")), gosx.Attr("data-studio-publish-check", workbenchMapString(check, "key"))), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__checks"), gosx.Attr("aria-label", "Publish checks")), gosx.Fragment(nodes...))
}

func renderPublishImpacts(panel map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, impact := range backendMapList(panel, "impacts") {
		nodes = append(nodes, gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(impact, "class")), gosx.Attr("data-studio-publish-impact", workbenchMapString(impact, "key"))),
			gosx.El("div", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(impact, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(impact, "scope")))),
			gosx.El("output", nil, gosx.Text(workbenchMapString(impact, "value"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(impact, "detail"))),
		))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__impacts"), gosx.Attr("aria-label", "Release impact")), gosx.El("h3", nil, gosx.Text("Release impact")), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-review__impact-list")), gosx.Fragment(nodes...)))
}

func renderRevisionHistoryItem(revision map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("strong", nil, gosx.Text(workbenchMapString(revision, "title"))),
		gosx.El("span", nil, gosx.Text(workbenchMapString(revision, "actionLabel"))),
		gosx.El("time", gosx.Attrs(gosx.Attr("datetime", workbenchMapString(revision, "createdMachine")), gosx.Attr("data-viewer-time", "datetime")), gosx.Text(workbenchMapString(revision, "createdLabel"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("hidden", !backendMapBool(revision, "hasSummary"))), gosx.Text(workbenchMapString(revision, "summary"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "revision-diff-summary"), gosx.Attr("hidden", !backendMapBool(revision, "hasDiff"))), gosx.Text(workbenchMapString(revision, "changeSummary"))),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "revision-diff-list"), gosx.Attr("hidden", !backendMapBool(revision, "hasDiff"))), gosx.Fragment(renderDiffItems(backendMapList(revision, "changeItems"))...)),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("type", "submit"), gosx.Attr("form", workbenchMapString(revision, "formID")), gosx.Attr("formaction", workbenchMapString(revision, "restoreAction")), gosx.Attr("formmethod", "post"), gosx.Attr("name", workbenchMapString(revision, "revisionInputName")), gosx.Attr("value", workbenchMapString(revision, "revisionInputValue")), gosx.Attr("data-admin-confirm", workbenchMapString(revision, "confirm")), gosx.Attr("data-studio-submit-action", "restoreRevision"), gosx.Attr("data-studio-field-action-formaction", workbenchMapString(revision, "restoreAction"))), gosx.Text(workbenchMapString(revision, "buttonLabel"))),
	}
	return gosx.El("li", gosx.Attrs(gosx.Attr("data-studio-revision", workbenchMapString(revision, "key"))), gosx.Fragment(children...))
}

func renderDiffList(changes []map[string]any) gosx.Node {
	return gosx.El("ul", gosx.Attrs(gosx.Attr("class", "revision-diff-list")), gosx.Fragment(renderDiffItems(changes)...))
}

func renderDiffItems(changes []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(changes))
	for _, change := range changes {
		nodes = append(nodes, gosx.El("li", nil, gosx.El("span", nil, gosx.Text(workbenchMapString(change, "kindLabel"))), gosx.El("code", nil, gosx.Text(workbenchMapString(change, "path")))))
	}
	return nodes
}

func renderReadinessSummary(readiness map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-summary"), gosx.Attr("aria-label", "Readiness summary")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "readyCount"))), gosx.Text("ready")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "watchCount"))), gosx.Text("watch")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "nextCount"))), gosx.Text("next")),
	)
}

func renderReadinessList(readiness map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, item := range backendMapList(readiness, "items") {
		children := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-card__head")), gosx.El("div", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(item, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(item, "summary")))), gosx.El("output", nil, gosx.Text(workbenchMapString(item, "statusLabel")))),
		}
		if workbenchMapString(item, "detail") != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(workbenchMapString(item, "detail"))))
		}
		if backendMapBool(item, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(gosx.Attr("href", workbenchMapString(item, "href")), gosx.Attr("data-gosx-link", "true")), gosx.Text(workbenchMapString(item, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(item, "class")), gosx.Attr("data-readiness-key", workbenchMapString(item, "key"))), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-list")), gosx.Fragment(nodes...))
}

func renderActivityHealthPanel(panel map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, check := range backendMapList(panel, "checks") {
		children := []gosx.Node{gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__card-head")), gosx.El("div", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(check, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(check, "scope")))), gosx.El("output", nil, gosx.Text(workbenchMapString(check, "statusLabel")))), gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__value")), gosx.Text(workbenchMapString(check, "value")))}
		if workbenchMapString(check, "detail") != "" {
			children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__detail")), gosx.Text(workbenchMapString(check, "detail"))))
		}
		if backendMapBool(check, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(gosx.Attr("href", workbenchMapString(check, "href")), gosx.Attr("data-gosx-link", "true")), gosx.Text(workbenchMapString(check, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(check, "class")), gosx.Attr("data-studio-health-check", workbenchMapString(check, "key"))), gosx.Fragment(children...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class")), gosx.Attr("data-studio-health", "true")), renderStatusHead("studio-health", panel), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__summary"), gosx.Attr("aria-label", "Site health summary")), gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "readyCount"))), gosx.Text("healthy")), gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "watchCount"))), gosx.Text("watch")), gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "nextCount"))), gosx.Text("next"))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__list")), gosx.Fragment(nodes...)))
}

func renderActivityPerformancePanel(panel map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, signal := range backendMapList(panel, "signals") {
		children := []gosx.Node{gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__card-head")), gosx.El("div", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(signal, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(signal, "value")))), gosx.El("output", nil, gosx.Text(workbenchMapString(signal, "statusLabel")))), gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__budget")), gosx.Text(workbenchMapString(signal, "budget")))}
		if workbenchMapString(signal, "summary") != "" {
			children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__summary-text")), gosx.Text(workbenchMapString(signal, "summary"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(signal, "class")), gosx.Attr("data-studio-performance-signal", workbenchMapString(signal, "key"))), gosx.Fragment(children...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class")), gosx.Attr("data-studio-performance", "true")), renderStatusHead("studio-performance", panel), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__summary"), gosx.Attr("aria-label", "Performance summary")), gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "readyCount"))), gosx.Text("ready")), gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "watchCount"))), gosx.Text("watch")), gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "nextCount"))), gosx.Text("next"))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__list")), gosx.Fragment(nodes...)))
}

func renderStatusHead(prefix string, panel map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", prefix+"__head")), gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", prefix+"__kicker")), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title")))), gosx.El("output", gosx.Attrs(gosx.Attr("class", prefix+"__count")), gosx.Text(workbenchMapString(panel, "summary"))))
}

func renderActivityEmptyPanel(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class"))), gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "headClass"))), gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "kickerClass"))), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title")))), gosx.El("output", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "countClass"))), gosx.Text(workbenchMapString(panel, "countLabel")))), gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "emptyClass"))), gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "emptyTitle"))), gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "emptyDetail")))))
}

func renderBrandPreview(panel, fields map[string]any) gosx.Node {
	preview := workbenchViewMap(fields, "preview")
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__preview-shell")), gosx.El("header", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "previewLabel")))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__preview-slot"), gosx.Attr("data-studio-brand-preview-slot", "true")), gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-brand-preview"), gosx.Attr("data-editor-brand-preview", "true")), gosx.El("span", gosx.Attrs(gosx.Attr("class", "editor-brand-preview__corner")), gosx.Text(workbenchMapString(preview, "cornerLabel"))), gosx.El("button", gosx.Attrs(gosx.Attr("class", "editor-brand-handle"), gosx.Attr("type", "button"), gosx.Attr("aria-label", workbenchMapString(preview, "buttonLabel")), gosx.Attr("data-editor-brand-handle", "true")), gosx.El("img", gosx.Attrs(gosx.Attr("src", workbenchMapString(preview, "logoURL")), gosx.Attr("alt", workbenchMapString(preview, "logoAlt")), gosx.Attr("data-editor-brand-logo", "true")))), gosx.El("output", gosx.Attrs(gosx.Attr("class", "editor-brand-readout"), gosx.Attr("data-editor-logo-readout", "true"), gosx.Attr("aria-live", "polite"))))))
}

func renderRadioInputs(groups []map[string]any, className, name string) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		nodes = append(nodes, gosx.El("input", gosx.Attrs(gosx.Attr("class", className), gosx.Attr("id", workbenchMapString(group, "inputID")), gosx.Attr("type", "radio"), gosx.Attr("name", name), gosx.Attr("value", workbenchMapString(group, "key")), gosx.Attr("checked", backendMapBool(group, "selected")), gosx.Attr("aria-label", workbenchMapString(group, "label")))))
	}
	return nodes
}

func renderBrandGroups(panel map[string]any, groups []map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, group := range groups {
		nodes = append(nodes, gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__group"), gosx.Attr("for", workbenchMapString(group, "inputID")), gosx.Attr("data-studio-brand-group-label", workbenchMapString(group, "key"))), gosx.El("strong", nil, gosx.Text(workbenchMapString(group, "label"))), gosx.El("small", nil, gosx.Text(workbenchMapString(group, "summary")))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__groups"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", workbenchMapString(panel, "groupLabel"))), gosx.Fragment(nodes...))
}

func renderBrandGroupSlot(group, fields map[string]any) gosx.Node {
	key := workbenchMapString(group, "key")
	body := []gosx.Node{}
	switch key {
	case "logo":
		body = append(body, renderFieldList(backendMapList(fields, "logoFields"), "studio-brand-panel__field-list", false))
	case "placement":
		body = append(body, renderFieldList(backendMapList(fields, "layoutFields"), "studio-brand-panel__field-list", false), renderBrandTools(workbenchViewMap(fields, "tools")))
	case "files":
		media := workbenchViewMap(fields, "media")
		body = append(body, renderFieldList(backendMapList(fields, "fileFields"), "studio-brand-panel__field-list", false), gosx.El("a", gosx.Attrs(gosx.Attr("class", workbenchMapString(media, "class")), gosx.Attr("href", workbenchMapString(media, "href")), gosx.Attr("data-gosx-link", "true")), gosx.Text(workbenchMapString(media, "label"))))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__group-slot"), gosx.Attr("data-studio-brand-group-slot", key), gosx.Attr("data-studio-brand-group-selected", BoolAttr(backendMapBool(group, "selected")))), gosx.El("header", nil, gosx.El("h3", nil, gosx.Text(workbenchMapString(group, "label"))), gosx.El("p", nil, gosx.Text(workbenchMapString(group, "summary")))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-brand-panel__group-body"), gosx.Attr("data-studio-brand-group-body", key)), gosx.Fragment(body...)))
}

func renderBrandTools(tools map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-brand-tools")), gosx.El("label", gosx.Attrs(gosx.Attr("class", "editor-brand-snap")), gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("data-editor-logo-snap", "true"), gosx.Attr("checked", backendMapBool(tools, "snapChecked")))), gosx.El("span", nil, gosx.Text("Snap"))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--compact")), gosx.El("label", gosx.Attrs(gosx.Attr("for", "logoSnapSize")), gosx.Text(workbenchMapString(tools, "gridLabel"))), gosx.El("input", gosx.Attrs(gosx.Attr("id", "logoSnapSize"), gosx.Attr("type", "number"), gosx.Attr("min", "1"), gosx.Attr("max", "32"), gosx.Attr("value", workbenchMapString(tools, "snapSize")), gosx.Attr("data-editor-logo-snap-size", "true")))), gosx.El("button", gosx.Attrs(gosx.Attr("class", "home-section-move"), gosx.Attr("type", "button"), gosx.Attr("data-editor-logo-reset", "true")), gosx.Text(workbenchMapString(tools, "resetLabel"))))
}

func renderFlowEditor(props, flow map[string]any, selected bool) gosx.Node {
	key := workbenchMapString(flow, "key")
	readiness := workbenchViewMap(flow, "readiness")
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", "studio-flow-editor"), gosx.Attr("data-editor-flow", key), gosx.Attr("data-studio-flow-editor", key), gosx.Attr("data-studio-flow-valid", BoolAttr(backendMapBool(readiness, "canPublish"))), gosx.Attr("data-studio-flow-readiness-status", workbenchMapString(readiness, "status")), gosx.Attr("data-studio-flow-dirty", BoolAttr(false)), gosx.Attr("data-studio-flow-editor-visible", BoolAttr(selected))),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__head")), gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(flow, "statusLabel"))), gosx.El("h3", nil, gosx.Text(workbenchMapString(flow, "label"))), gosx.El("p", nil, gosx.Text(workbenchMapString(flow, "summary")))), gosx.El("output", gosx.Attrs(gosx.Attr("class", workbenchMapString(readiness, "badgeClass")), gosx.Attr("data-studio-flow-editor-state", key), gosx.Attr("data-studio-flow-editor-state-visible", BoolAttr(true))), gosx.Text(workbenchMapString(readiness, "label"))), gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__state studio-flow-editor__state--watch"), gosx.Attr("data-studio-flow-editor-state", key), gosx.Attr("data-studio-flow-editor-state-visible", BoolAttr(false))), gosx.Text("Unsaved changes")), gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--secondary"), gosx.Attr("type", "submit"), gosx.Attr("form", workbenchMapString(props, "formID")), gosx.Attr("formaction", workbenchMapString(props, "publishAction")), gosx.Attr("formmethod", "post"), gosx.Attr("name", "flowKey"), gosx.Attr("value", key), gosx.Attr("data-studio-submit-action", "publish-flow")), gosx.Text("Publish flow"))),
		renderFlowReadiness(key, readiness, backendMapList(flow, "readinessChecks")),
		renderFlowGraph(backendMapList(flow, "nodes")),
		renderFlowRoute(props, flow),
		renderFlowSteps(props, flow),
		renderFlowActions(flow),
	)
}

func renderFlowReadiness(key string, readiness map[string]any, checks []map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, check := range checks {
		nodes = append(nodes, gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(check, "class")), gosx.Attr("role", "listitem"), gosx.Attr("tabindex", workbenchMapString(check, "tabIndex")), gosx.Attr("data-studio-flow-check", workbenchMapString(check, "key")), gosx.Attr("data-studio-flow-check-status", workbenchMapString(check, "status"))), gosx.El("span", nil, gosx.Text(workbenchMapString(check, "statusLabel"))), gosx.El("strong", nil, gosx.Text(workbenchMapString(check, "label"))), gosx.El("small", nil, gosx.Text(workbenchMapString(check, "summary")))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(readiness, "class")), gosx.Attr("data-studio-flow-readiness", key)), gosx.El("header", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(readiness, "summary")))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-checks"), gosx.Attr("role", "list"), gosx.Attr("aria-label", "Flow readiness checks")), gosx.Fragment(nodes...)))
}

func renderFlowGraph(nodes []map[string]any) gosx.Node {
	out := []gosx.Node{}
	for _, node := range nodes {
		out = append(out, gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(node, "class")), gosx.Attr("role", "listitem"), gosx.Attr("tabindex", workbenchMapString(node, "tabIndex")), gosx.Attr("data-studio-flow-node", workbenchMapString(node, "key")), gosx.Attr("data-studio-flow-node-kind", workbenchMapString(node, "kind")), gosx.Attr("data-studio-flow-node-status", workbenchMapString(node, "status"))), gosx.El("span", nil, gosx.Text(workbenchMapString(node, "kindLabel"))), gosx.El("strong", nil, gosx.Text(workbenchMapString(node, "label"))), gosx.El("small", nil, gosx.Text(workbenchMapString(node, "summary")))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-graph"), gosx.Attr("role", "list"), gosx.Attr("aria-label", "Flow map")), gosx.Fragment(out...))
}

func renderFlowRoute(props, flow map[string]any) gosx.Node {
	key := workbenchMapString(flow, "key")
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__route")),
		readonlyField(workbenchMapString(flow, "routeInputID"), workbenchMapString(flow, "routeInputName"), workbenchMapString(flow, "route"), workbenchMapString(props, "formID"), "Public page", "flow."+key+".route", "routing"),
		readonlyField(workbenchMapString(flow, "embedInputID"), workbenchMapString(flow, "embedInputName"), workbenchMapString(flow, "embedTarget"), workbenchMapString(props, "formID"), "Appears in", "flow."+key+".embedTarget", "routing"),
		gosx.El("input", gosx.Attrs(gosx.Attr("id", workbenchMapString(flow, "handlerInputID")), gosx.Attr("name", workbenchMapString(flow, "handlerInputName")), gosx.Attr("value", workbenchMapString(flow, "handlerRef")), gosx.Attr("form", workbenchMapString(props, "formID")), gosx.Attr("data-studio-field-source", workbenchMapString(flow, "handlerSource")), gosx.Attr("data-studio-field-editable", "flow"), gosx.Attr("data-studio-flow-handler-ref", "true"), gosx.Attr("type", "hidden"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(flow, "handlerClass")), gosx.Attr("data-studio-flow-handler-summary", key)), gosx.El("strong", nil, gosx.Text(workbenchMapString(flow, "handlerStatusLabel"))), gosx.El("span", nil, gosx.Text(workbenchMapString(flow, "handlerDetail")))),
	)
}

func renderFlowSteps(props, flow map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, step := range backendMapList(flow, "steps") {
		nodes = append(nodes, gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-flow-step"), gosx.Attr("data-studio-flow-step", workbenchMapString(step, "key")), gosx.Attr("tabindex", "0")), gosx.El("label", gosx.Attrs(gosx.Attr("class", "field"), gosx.Attr("for", workbenchMapString(step, "labelInputID"))), gosx.El("span", nil, gosx.Text(workbenchMapString(step, "label"))), gosx.El("input", gosx.Attrs(gosx.Attr("id", workbenchMapString(step, "labelInputID")), gosx.Attr("name", workbenchMapString(step, "labelInputName")), gosx.Attr("value", workbenchMapString(step, "label")), gosx.Attr("form", workbenchMapString(props, "formID")), gosx.Attr("data-studio-field-source", workbenchMapString(step, "labelSource")), gosx.Attr("data-studio-field-editable", "flow")))), gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-flow-step__body"), gosx.Attr("data-studio-field-source", workbenchMapString(step, "bodySource")), gosx.Attr("data-studio-field-editable", "flow")), gosx.Text(workbenchMapString(step, "bodySummary")))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__steps"), gosx.Attr("aria-label", "Flow steps")), gosx.Fragment(nodes...))
}

func renderFlowActions(flow map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, action := range backendMapList(flow, "actions") {
		fields := []gosx.Node{}
		for _, field := range backendMapList(action, "fields") {
			fields = append(fields, gosx.El("span", gosx.Attrs(gosx.Attr("class", "studio-flow-field"), gosx.Attr("data-studio-flow-field", workbenchMapString(field, "name")), gosx.Attr("tabindex", "0")), gosx.El("strong", nil, gosx.Text(workbenchMapString(field, "label"))), gosx.El("small", nil, gosx.Text(workbenchMapString(field, "requiredLabel")))))
		}
		nodes = append(nodes, gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-flow-action"), gosx.Attr("data-studio-flow-action", workbenchMapString(action, "key")), gosx.Attr("tabindex", "0")), gosx.El("header", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(action, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(action, "fieldCount")), gosx.Text("fields"))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-field-list")), gosx.Fragment(fields...))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-editor__actions"), gosx.Attr("aria-label", "Flow actions")), gosx.Fragment(nodes...))
}

func readonlyField(id, name, value, formID, label, source, editable string) gosx.Node {
	return gosx.El("label", gosx.Attrs(gosx.Attr("class", "field"), gosx.Attr("for", id)), gosx.El("span", nil, gosx.Text(label)), gosx.El("input", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", name), gosx.Attr("value", value), gosx.Attr("form", formID), gosx.Attr("data-studio-field-source", source), gosx.Attr("data-studio-field-editable", editable), gosx.Attr("readonly", "readonly"))))
}

func renderAdvancedGroups(panel map[string]any, groups []map[string]any) gosx.Node {
	nodes := []gosx.Node{}
	for _, group := range groups {
		nodes = append(nodes, gosx.El("label", gosx.Attrs(gosx.Attr("class", "studio-advanced-panel__group"), gosx.Attr("for", workbenchMapString(group, "inputID")), gosx.Attr("data-studio-advanced-group-label", workbenchMapString(group, "key"))), gosx.El("strong", nil, gosx.Text(workbenchMapString(group, "label"))), gosx.El("small", nil, gosx.Text(workbenchMapString(group, "summary")))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-advanced-panel__groups"), gosx.Attr("role", "radiogroup"), gosx.Attr("aria-label", workbenchMapString(panel, "groupLabel"))), gosx.Fragment(nodes...))
}

func renderAdvancedGroupSlot(view, group map[string]any) gosx.Node {
	key := workbenchMapString(group, "key")
	body := []gosx.Node{}
	switch key {
	case "flows":
		body = append(body, RenderFlowDesignerPanel(view))
	case "tools":
		body = append(body, renderAdvancedToolsPanel(backendSurfaceMap(view, "toolsPanel")))
	case "schema":
		body = append(body, renderAdvancedFieldPanel(workbenchViewMap(backendSurfaceMap(view, "advancedFields"), "workspace")))
	case "schedule":
		body = append(body, renderAdvancedFieldPanel(workbenchViewMap(backendSurfaceMap(view, "advancedFields"), "calendar")))
	case "typography":
		body = append(body, renderAdvancedFieldPanel(workbenchViewMap(backendSurfaceMap(view, "advancedFields"), "type")))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-advanced-panel__group-slot"), gosx.Attr("data-studio-advanced-group-slot", key), gosx.Attr("data-studio-advanced-group-selected", BoolAttr(backendMapBool(group, "selected")))), gosx.El("header", nil, gosx.El("h3", nil, gosx.Text(workbenchMapString(group, "label"))), gosx.El("p", nil, gosx.Text(workbenchMapString(group, "summary")))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-advanced-panel__group-body"), gosx.Attr("data-studio-advanced-group-body", key)), gosx.Fragment(body...)))
}

func renderAdvancedToolsPanel(panel map[string]any) gosx.Node {
	children := []gosx.Node{gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")), gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title")))), renderResourceAdapters(panel)}
	if !backendMapBool(panel, "hasItems") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(panel, "empty"))))
	} else {
		items := []gosx.Node{}
		for _, item := range backendMapList(panel, "items") {
			items = append(items, gosx.El("a", backendAttrs(workbenchViewMap(item, "attrs")), gosx.El("strong", nil, gosx.Text(workbenchMapString(item, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(item, "summary")))))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "gridClass"))), gosx.Fragment(items...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class")), gosx.Attr("data-panel-key", workbenchMapString(panel, "key")), gosx.Attr("data-studio-panel", workbenchMapString(panel, "key")), gosx.Attr("data-studio-mode-panel", workbenchMapString(panel, "mode")), gosx.Attr("data-studio-engine-source", "gosx")), gosx.Fragment(children...))
}

func renderResourceAdapters(panel map[string]any) gosx.Node {
	children := []gosx.Node{gosx.El("header", nil, gosx.El("div", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "resourcesTitle"))), gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "resourcesSummary")))))}
	if !backendMapBool(panel, "hasAdapters") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(panel, "resourceEmpty"))))
	} else {
		cards := []gosx.Node{}
		for _, adapter := range backendMapList(panel, "adapters") {
			cards = append(cards, gosx.El("article", gosx.Attrs(gosx.Attr("class", "studio-resource-adapter-card"), gosx.Attr("data-studio-resource-adapter", workbenchMapString(adapter, "kind")), gosx.Attr("data-studio-resource-surface", workbenchMapString(adapter, "surface"))), gosx.El("strong", nil, gosx.Text(workbenchMapString(adapter, "label"))), gosx.El("span", nil, gosx.Text(workbenchMapString(adapter, "summary"))), gosx.El("div", nil, gosx.El("small", nil, gosx.Text(workbenchMapString(adapter, "surface"))), gosx.El("small", nil, gosx.Text(workbenchMapString(adapter, "capabilityLabel"))), gosx.El("small", nil, gosx.Text(workbenchMapString(adapter, "bindingLabel"))))))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "resourceGridClass"))), gosx.Fragment(cards...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "studio-resource-adapters"), gosx.Attr("data-studio-resource-adapters", "true")), gosx.Fragment(children...))
}

func renderAdvancedFieldPanel(panel map[string]any) gosx.Node {
	children := []gosx.Node{gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")), gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(panel, "kicker"))), gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title"))))}
	if !backendMapBool(panel, "hasFields") {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(workbenchMapString(panel, "empty"))))
	} else {
		children = append(children, renderFieldList(backendMapList(panel, "fields"), "studio-advanced-fields-panel__fields", true))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class")), gosx.Attr("data-panel-key", workbenchMapString(panel, "key")), gosx.Attr("data-studio-panel", workbenchMapString(panel, "key")), gosx.Attr("data-studio-mode-panel", workbenchMapString(panel, "mode")), gosx.Attr("data-studio-engine-source", "gosx")), gosx.Fragment(children...))
}

func renderFieldList(fields []map[string]any, className string, wrapAdvanced bool) gosx.Node {
	nodes := make([]gosx.Node, 0, len(fields))
	for _, field := range fields {
		node := renderFieldRow(field)
		if wrapAdvanced {
			node = gosx.El("div", gosx.Attrs(gosx.Attr("data-studio-advanced-field-row", "true")), node)
		}
		nodes = append(nodes, node)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(nodes...))
}

func renderFieldRow(field map[string]any) gosx.Node {
	if backendMapBool(field, "isCard") {
		actions := []gosx.Node{}
		for _, action := range backendMapList(field, "actions") {
			if backendMapBool(action, "hasFormAction") {
				actions = append(actions, gosx.El("button", backendAttrs(workbenchViewMap(action, "attrs")), gosx.Text(workbenchMapString(action, "label"))))
			} else {
				actions = append(actions, gosx.El("a", backendAttrs(workbenchViewMap(action, "attrs")), gosx.Text(workbenchMapString(action, "label"))))
			}
		}
		card := gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-source-card")), gosx.El("strong", nil, gosx.Text(workbenchMapString(field, "cardTitle"))), gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help"), gosx.Attr("hidden", !backendMapBool(field, "hasHelp"))), gosx.Text(workbenchMapString(field, "help"))), gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(actions...)))
		return gosx.El("div", backendAttrs(workbenchViewMap(field, "rowAttrs")), gosx.El("label", nil, gosx.Text(workbenchMapString(field, "label"))), card)
	}
	children := []gosx.Node{gosx.El("label", gosx.Attrs(gosx.Attr("for", workbenchMapString(field, "id"))), gosx.Text(workbenchMapString(field, "label")))}
	switch {
	case backendMapBool(field, "isArea"):
		children = append(children, gosx.El("textarea", backendAttrs(workbenchViewMap(field, "controlAttrs")), gosx.Text(workbenchMapString(field, "value"))))
	case backendMapBool(field, "isSelect"):
		options := []gosx.Node{}
		for _, option := range backendMapList(field, "options") {
			options = append(options, gosx.El("option", backendAttrs(workbenchViewMap(option, "attrs")), gosx.Text(workbenchMapString(option, "label"))))
		}
		children = append(children, gosx.El("select", backendAttrs(workbenchViewMap(field, "controlAttrs")), gosx.Fragment(options...)))
	case backendMapBool(field, "isCheckbox"), backendMapBool(field, "isInput"):
		children = append(children, gosx.El("input", backendAttrs(workbenchViewMap(field, "controlAttrs"))))
	}
	if backendMapBool(field, "hasHelp") {
		children = append(children, gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(workbenchMapString(field, "help"))))
	}
	return gosx.El("div", backendAttrs(workbenchViewMap(field, "rowAttrs")), gosx.Fragment(children...))
}

func submitActionButton(className, formID, action, submitAction, confirm, label string) gosx.Node {
	return gosx.El("button", gosx.Attrs(gosx.Attr("class", className), gosx.Attr("type", "submit"), gosx.Attr("form", formID), gosx.Attr("formaction", action), gosx.Attr("formmethod", "post"), gosx.Attr("data-admin-confirm", confirm), gosx.Attr("data-studio-submit-action", submitAction), gosx.Attr("data-studio-field-action-formaction", action)), gosx.Text(label))
}

func backendSurfaceMap(view map[string]any, key string) map[string]any {
	if nested := workbenchViewMap(view, key); nested != nil {
		return nested
	}
	return view
}

func backendMapList(view map[string]any, key string) []map[string]any {
	return workbenchViewMapList(view, key)
}

func backendMapBool(view map[string]any, key string) bool {
	return workbenchMapBool(view, key)
}

func backendAttrs(values map[string]any) any {
	if values == nil {
		return gosx.Attrs()
	}
	attrs := make([]any, 0, len(values))
	for key, value := range values {
		name := strings.TrimSpace(key)
		if name == "" || value == nil {
			continue
		}
		attrs = append(attrs, gosx.Attr(name, value))
	}
	return gosx.Attrs(attrs...)
}
