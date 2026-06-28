package studio

import "m31labs.dev/gosx"

type PublishPanelOptions struct {
	RootAttrs           map[string]any
	PreviewShareNode    gosx.Node
	ActivityPanelNode   gosx.Node
	RevisionHistoryNode gosx.Node
}

func RenderPublishPanel(view map[string]any, options PublishPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", "editor-panel editor-panel--publish-studio studio-publish-panel"),
		gosx.Attr("data-studio-publish-panel", "true"),
		gosx.Attr("data-studio-mode-panel", "publish"),
		gosx.Attr("data-studio-panel", "publish"),
		gosx.Attr("data-studio-engine-source", "gosx"),
		gosx.Attr("data-studio-publish-status", workbenchMapString(view, "status")),
		gosx.Attr("data-gosx-studio-publish-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		renderPublishPanelHead(view),
		renderPublishPanelDraftStatus(view),
		renderPublishPanelActions(view),
	}
	if workbenchMapBool(view, "hasPendingChanges") {
		children = append(children, renderPublishPanelPendingChanges(view))
	}
	children = append(children,
		renderPublishPanelPreviewShare(options.PreviewShareNode),
		renderPublishPanelSchedule(view),
		renderPublishPanelDecision(view),
		renderPublishPanelSummary(view),
	)
	if workbenchMapBool(view, "hasChecks") {
		children = append(children, renderPublishPanelChecks(workbenchViewMapList(view, "checks")))
	}
	if workbenchMapBool(view, "hasImpacts") {
		children = append(children, renderPublishPanelImpacts(workbenchViewMapList(view, "impacts")))
	}
	children = append(children,
		renderPublishPanelActivity(options.ActivityPanelNode),
		renderPublishPanelHistory(options.RevisionHistoryNode),
	)

	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func renderPublishPanelHead(view map[string]any) gosx.Node {
	return gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "panelTitle"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(view, "summary"))),
		),
		gosx.El("output", nil, gosx.Text(workbenchMapString(view, "countLabel"))),
	)
}

func renderPublishPanelDraftStatus(view map[string]any) gosx.Node {
	hasDraft := workbenchMapBool(view, "hasDraft")
	return gosx.El("p", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__draft-status"),
		gosx.Attr("data-studio-publish-draft-status", "true"),
		gosx.Attr("data-studio-has-draft", BoolAttr(hasDraft)),
		gosx.Attr("hidden", !hasDraft),
	), gosx.Text(workbenchMapString(view, "draftStatusLabel")))
}

func renderPublishPanelActions(view map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("a", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("href", workbenchMapString(view, "previewHref")),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text("Open preview")),
	}
	if workbenchMapBool(view, "hasPublishAction") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--primary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("form", workbenchMapString(view, "formID")),
			gosx.Attr("formaction", workbenchMapString(view, "publishAction")),
			gosx.Attr("formmethod", "post"),
			gosx.Attr("data-admin-confirm", "Publish this draft?"),
			gosx.Attr("data-studio-submit-action", "publish"),
			gosx.Attr("data-studio-field-action-formaction", workbenchMapString(view, "publishAction")),
		), gosx.Text("Publish")))
	}
	if workbenchMapBool(view, "hasDiscardAction") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--ghost"),
			gosx.Attr("type", "submit"),
			gosx.Attr("form", workbenchMapString(view, "formID")),
			gosx.Attr("formaction", workbenchMapString(view, "discardAction")),
			gosx.Attr("formmethod", "post"),
			gosx.Attr("data-admin-confirm", "Discard unpublished changes?"),
			gosx.Attr("data-studio-submit-action", "discard"),
			gosx.Attr("data-studio-field-action-formaction", workbenchMapString(view, "discardAction")),
		), gosx.Text("Discard changes")))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__actions"),
		gosx.Attr("aria-label", "Publish actions"),
	), gosx.Fragment(children...))
}

func renderPublishPanelPendingChanges(view map[string]any) gosx.Node {
	items := make([]gosx.Node, 0, len(workbenchViewMapList(view, "pendingChanges")))
	for _, change := range workbenchViewMapList(view, "pendingChanges") {
		items = append(items, gosx.El("li", nil,
			gosx.El("span", nil, gosx.Text(workbenchMapString(change, "kindLabel"))),
			gosx.El("code", nil, gosx.Text(workbenchMapString(change, "path"))),
		))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__pending"),
		gosx.Attr("aria-label", "Unpublished changes"),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("What will change")),
			gosx.El("p", nil, gosx.Text(workbenchMapString(view, "draftSummary"))),
		),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "revision-diff-list")), gosx.Fragment(items...)),
	)
}

func renderPublishPanelPreviewShare(node gosx.Node) gosx.Node {
	children := []gosx.Node{}
	if !workbenchNodeEmpty(node) {
		children = append(children, node)
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__preview-share"),
		gosx.Attr("data-studio-publish-preview-share", "true"),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Share preview")),
			gosx.El("p", nil, gosx.Text("Send a draft preview for review without giving admin access.")),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-publish-panel__preview-share-slot"),
			gosx.Attr("data-studio-publish-preview-share-slot", "true"),
		), gosx.Fragment(children...)),
	)
}

func renderPublishPanelSchedule(view map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "field"),
			gosx.Attr("for", workbenchMapString(view, "scheduleInputID")),
		),
			gosx.El("span", nil, gosx.Text("Schedule for")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", workbenchMapString(view, "scheduleInputID")),
				gosx.Attr("name", workbenchMapString(view, "scheduleInputName")),
				gosx.Attr("type", "datetime-local"),
				gosx.Attr("value", workbenchMapString(view, "scheduleInputValue")),
				gosx.Attr("form", workbenchMapString(view, "formID")),
				gosx.Attr("data-studio-field-source", "lifecycle.schedule.publishAt"),
				gosx.Attr("data-studio-field-editable", "lifecycle"),
			)),
		),
		gosx.El("p", nil, gosx.Text(workbenchMapString(view, "scheduleHelp"))),
	}
	if workbenchMapBool(view, "hasScheduleAction") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("form", workbenchMapString(view, "formID")),
			gosx.Attr("formaction", workbenchMapString(view, "scheduleAction")),
			gosx.Attr("formmethod", "post"),
			gosx.Attr("data-admin-confirm", "Schedule this draft?"),
			gosx.Attr("data-studio-submit-action", "schedule"),
			gosx.Attr("data-studio-field-action-formaction", workbenchMapString(view, "scheduleAction")),
		), gosx.Text("Schedule")))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__schedule")), gosx.Fragment(children...))
}

func renderPublishPanelDecision(view map[string]any) gosx.Node {
	children := []gosx.Node{}
	if workbenchMapBool(view, "hasApproval") {
		children = append(children, renderPublishPanelApproval(workbenchViewMap(view, "approval")))
	}
	if workbenchMapBool(view, "hasSchedule") {
		children = append(children, renderPublishPanelScheduleDecision(workbenchViewMap(view, "schedule")))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__decision"),
		gosx.Attr("aria-label", "Release decision"),
	), gosx.Fragment(children...))
}

func renderPublishPanelApproval(approval map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-head")),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(approval, "label"))),
			gosx.El("output", nil, gosx.Text(workbenchMapString(approval, "statusLabel"))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-summary")), gosx.Text(workbenchMapString(approval, "summary"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(workbenchMapString(approval, "detail"))),
	}
	if workbenchMapBool(approval, "hasHref") {
		children = append(children, gosx.El("a", gosx.Attrs(
			gosx.Attr("href", workbenchMapString(approval, "href")),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(workbenchMapString(approval, "actionLabel"))))
	}
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(approval, "class"))), gosx.Fragment(children...))
}

func renderPublishPanelScheduleDecision(schedule map[string]any) gosx.Node {
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(schedule, "class"))),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-head")),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(schedule, "label"))),
			gosx.El("output", nil, gosx.Text(workbenchMapString(schedule, "statusLabel"))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-summary")), gosx.Text(workbenchMapString(schedule, "summary"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(workbenchMapString(schedule, "detail"))),
	)
}

func renderPublishPanelSummary(view map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-publish-review__summary"),
		gosx.Attr("aria-label", "Publish review summary"),
	),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(view, "readyCountLabel"))), gosx.Text("clear")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(view, "watchCountLabel"))), gosx.Text("watch")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(workbenchMapString(view, "nextCountLabel"))), gosx.Text("next")),
	)
}

func renderPublishPanelChecks(checks []map[string]any) gosx.Node {
	nodes := make([]gosx.Node, 0, len(checks))
	for _, check := range checks {
		children := []gosx.Node{
			gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__card-head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(workbenchMapString(check, "label"))),
					gosx.El("span", nil, gosx.Text(workbenchMapString(check, "scope"))),
				),
				gosx.El("output", nil, gosx.Text(workbenchMapString(check, "statusLabel"))),
			),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__check-summary")), gosx.Text(workbenchMapString(check, "summary"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(workbenchMapString(check, "detail"))),
		}
		if workbenchMapBool(check, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", workbenchMapString(check, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text(workbenchMapString(check, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(check, "class")),
			gosx.Attr("data-studio-publish-check", workbenchMapString(check, "key")),
		), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__checks"),
		gosx.Attr("aria-label", "Publish checks"),
	), gosx.Fragment(nodes...))
}

func renderPublishPanelImpacts(impacts []map[string]any) gosx.Node {
	nodes := make([]gosx.Node, 0, len(impacts))
	for _, impact := range impacts {
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(impact, "class")),
			gosx.Attr("data-studio-publish-impact", workbenchMapString(impact, "key")),
		),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(impact, "label"))),
				gosx.El("span", nil, gosx.Text(workbenchMapString(impact, "scope"))),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(impact, "value"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(impact, "detail"))),
		))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__impacts"),
		gosx.Attr("aria-label", "Release impact"),
	),
		gosx.El("h3", nil, gosx.Text("Release impact")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-review__impact-list")), gosx.Fragment(nodes...)),
	)
}

func renderPublishPanelActivity(node gosx.Node) gosx.Node {
	children := []gosx.Node{}
	if !workbenchNodeEmpty(node) {
		children = append(children, node)
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__collaboration"),
		gosx.Attr("data-studio-publish-activity", "true"),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Review activity")),
			gosx.El("p", nil, gosx.Text("Readiness, review notes, and suggested changes for this draft.")),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-publish-panel__activity-slot"),
			gosx.Attr("data-studio-publish-activity-slot", "true"),
		), gosx.Fragment(children...)),
	)
}

func renderPublishPanelHistory(node gosx.Node) gosx.Node {
	children := []gosx.Node{}
	if !workbenchNodeEmpty(node) {
		children = append(children, node)
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__history"),
		gosx.Attr("data-studio-publish-history", "true"),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Restore points")),
			gosx.El("p", nil, gosx.Text("Recent saved versions that can be restored before or after a release.")),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-publish-panel__history-slot"),
			gosx.Attr("data-studio-publish-history-slot", "true"),
		), gosx.Fragment(children...)),
	)
}
