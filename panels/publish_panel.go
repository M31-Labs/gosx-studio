package panels

import (
	"strings"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

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
		gosx.Attr("data-studio-publish-status", core.WorkbenchViewString(view, "status")),
		gosx.Attr("data-gosx-studio-publish-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		renderPublishPanelHead(view),
		renderPublishPanelDraftStatus(view),
		renderPublishPanelActions(view),
	}
	if core.WorkbenchViewBool(view, "hasPendingChanges") {
		children = append(children, renderPublishPanelPendingChanges(view))
	}
	children = append(children,
		renderPublishPanelPreviewShare(options.PreviewShareNode),
	)
	if core.WorkbenchViewBool(view, "hasEnvironments") {
		children = append(children, renderPublishPanelEnvironments(core.WorkbenchViewMapList(view, "environments")))
	}
	children = append(children,
		renderPublishPanelSchedule(view),
		renderPublishPanelDecision(view),
		renderPublishPanelSummary(view),
	)
	if core.WorkbenchViewBool(view, "hasChecks") {
		children = append(children, renderPublishPanelChecks(core.WorkbenchViewMapList(view, "checks")))
	}
	if core.WorkbenchViewBool(view, "hasImpacts") {
		children = append(children, renderPublishPanelImpacts(core.WorkbenchViewMapList(view, "impacts")))
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
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "panelTitle"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(view, "summary"))),
		),
		gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(view, "countLabel"))),
	)
}

func renderPublishPanelDraftStatus(view map[string]any) gosx.Node {
	hasDraft := core.WorkbenchViewBool(view, "hasDraft")
	return gosx.El("p", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__draft-status"),
		gosx.Attr("data-studio-publish-draft-status", "true"),
		gosx.Attr("data-studio-has-draft", core.BoolAttr(hasDraft)),
		gosx.Attr("hidden", !hasDraft),
	), gosx.Text(core.WorkbenchViewString(view, "draftStatusLabel")))
}

func renderPublishPanelActions(view map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("a", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("href", core.WorkbenchViewString(view, "previewHref")),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text("Open preview")),
	}
	if core.WorkbenchViewBool(view, "hasPublishAction") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--primary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("form", core.WorkbenchViewString(view, "formID")),
			gosx.Attr("formaction", core.WorkbenchViewString(view, "publishAction")),
			gosx.Attr("formmethod", "post"),
			gosx.Attr("data-admin-confirm", "Publish this draft?"),
			gosx.Attr("data-studio-submit-action", "publish"),
			gosx.Attr("data-studio-field-action-formaction", core.WorkbenchViewString(view, "publishAction")),
		), gosx.Text("Publish")))
	}
	if core.WorkbenchViewBool(view, "hasDiscardAction") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--ghost"),
			gosx.Attr("type", "submit"),
			gosx.Attr("form", core.WorkbenchViewString(view, "formID")),
			gosx.Attr("formaction", core.WorkbenchViewString(view, "discardAction")),
			gosx.Attr("formmethod", "post"),
			gosx.Attr("data-admin-confirm", "Discard unpublished changes?"),
			gosx.Attr("data-studio-submit-action", "discard"),
			gosx.Attr("data-studio-field-action-formaction", core.WorkbenchViewString(view, "discardAction")),
		), gosx.Text("Discard changes")))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__actions"),
		gosx.Attr("aria-label", "Publish actions"),
	), gosx.Fragment(children...))
}

func renderPublishPanelPendingChanges(view map[string]any) gosx.Node {
	items := make([]gosx.Node, 0, len(core.WorkbenchViewMapList(view, "pendingChanges")))
	for _, change := range core.WorkbenchViewMapList(view, "pendingChanges") {
		items = append(items, gosx.El("li", nil,
			gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(change, "kindLabel"))),
			gosx.El("code", nil, gosx.Text(core.WorkbenchViewString(change, "path"))),
		))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__pending"),
		gosx.Attr("aria-label", "Unpublished changes"),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("What will change")),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(view, "draftSummary"))),
		),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "revision-diff-list")), gosx.Fragment(items...)),
	)
}

func renderPublishPanelPreviewShare(node gosx.Node) gosx.Node {
	children := []gosx.Node{}
	if !core.WorkbenchNodeEmpty(node) {
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

func renderPublishPanelEnvironments(environments []map[string]any) gosx.Node {
	nodes := make([]gosx.Node, 0, len(environments))
	for _, environment := range environments {
		state := publishEnvironmentState(environment)
		value := publishEnvironmentValue(environment)
		if state == "tbd" && value == "" {
			value = "TBD"
		}
		valueNode := gosx.El("span", gosx.Attrs(
			gosx.Attr("class", "studio-publish-environment__value"),
			gosx.Attr("data-studio-publish-environment-value", "true"),
		), gosx.Text(value))
		if publishEnvironmentHasHref(environment, state) {
			valueNode = gosx.El("a", gosx.Attrs(
				gosx.Attr("class", "studio-publish-environment__value"),
				gosx.Attr("href", publishEnvironmentHref(environment)),
				gosx.Attr("target", "_blank"),
				gosx.Attr("rel", "noreferrer"),
				gosx.Attr("data-gosx-link", "true"),
				gosx.Attr("data-studio-publish-environment-value", "true"),
			), gosx.Text(value))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", publishEnvironmentClass(environment, state)),
			gosx.Attr("data-studio-publish-environment", core.WorkbenchViewString(environment, "key")),
			gosx.Attr("data-studio-publish-environment-state", state),
		),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(environment, "label"))),
				gosx.El("output", nil, gosx.Text(publishEnvironmentStateLabel(environment, state))),
			),
			valueNode,
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(environment, "detail"))),
		))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__environments"),
		gosx.Attr("aria-label", "Publishing environments"),
		gosx.Attr("data-studio-publish-environments", "true"),
	),
		gosx.El("header", nil,
			gosx.El("h3", nil, gosx.Text("Publishing environments")),
			gosx.El("p", nil, gosx.Text("Open the deployed staging or production storefront for release review.")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-environment-list")), gosx.Fragment(nodes...)),
	)
}

func publishEnvironmentClass(environment map[string]any, state string) string {
	className := core.WorkbenchViewString(environment, "class")
	if className != "" {
		return className
	}
	return "studio-publish-environment studio-publish-environment--" + state
}

func publishEnvironmentState(environment map[string]any) string {
	value := strings.ToLower(publishEnvironmentValue(environment))
	if value == "" || value == "tbd" {
		return "tbd"
	}
	state := strings.ToLower(core.WorkbenchViewString(environment, "state"))
	switch state {
	case "ready", "tbd":
		return state
	}
	return "ready"
}

func publishEnvironmentStateLabel(environment map[string]any, state string) string {
	if label := core.WorkbenchViewString(environment, "stateLabel"); label != "" {
		return label
	}
	if state == "tbd" {
		return "TBD"
	}
	return "Ready"
}

func publishEnvironmentValue(environment map[string]any) string {
	if value := core.WorkbenchViewString(environment, "value"); value != "" {
		return value
	}
	return core.WorkbenchViewString(environment, "url")
}

func publishEnvironmentHref(environment map[string]any) string {
	if href := core.WorkbenchViewString(environment, "href"); href != "" {
		return href
	}
	return core.WorkbenchViewString(environment, "url")
}

func publishEnvironmentHasHref(environment map[string]any, state string) bool {
	if state == "tbd" {
		return false
	}
	if publishEnvironmentHref(environment) == "" {
		return false
	}
	if _, ok := environment["hasHref"]; !ok {
		return true
	}
	return core.WorkbenchViewBool(environment, "hasHref")
}

func renderPublishPanelSchedule(view map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "field"),
			gosx.Attr("for", core.WorkbenchViewString(view, "scheduleInputID")),
		),
			gosx.El("span", nil, gosx.Text("Schedule for")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", core.WorkbenchViewString(view, "scheduleInputID")),
				gosx.Attr("name", core.WorkbenchViewString(view, "scheduleInputName")),
				gosx.Attr("type", "datetime-local"),
				gosx.Attr("value", core.WorkbenchViewString(view, "scheduleInputValue")),
				gosx.Attr("form", core.WorkbenchViewString(view, "formID")),
				gosx.Attr("data-studio-field-source", "lifecycle.schedule.publishAt"),
				gosx.Attr("data-studio-field-editable", "lifecycle"),
			)),
		),
		gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(view, "scheduleHelp"))),
	}
	if core.WorkbenchViewBool(view, "hasScheduleAction") {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("form", core.WorkbenchViewString(view, "formID")),
			gosx.Attr("formaction", core.WorkbenchViewString(view, "scheduleAction")),
			gosx.Attr("formmethod", "post"),
			gosx.Attr("data-admin-confirm", "Schedule this draft?"),
			gosx.Attr("data-studio-submit-action", "schedule"),
			gosx.Attr("data-studio-field-action-formaction", core.WorkbenchViewString(view, "scheduleAction")),
		), gosx.Text("Schedule")))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-publish-panel__schedule")), gosx.Fragment(children...))
}

func renderPublishPanelDecision(view map[string]any) gosx.Node {
	children := []gosx.Node{}
	if core.WorkbenchViewBool(view, "hasApproval") {
		children = append(children, renderPublishPanelApproval(core.WorkbenchViewMap(view, "approval")))
	}
	if core.WorkbenchViewBool(view, "hasSchedule") {
		children = append(children, renderPublishPanelScheduleDecision(core.WorkbenchViewMap(view, "schedule")))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-publish-panel__decision"),
		gosx.Attr("aria-label", "Release decision"),
	), gosx.Fragment(children...))
}

func renderPublishPanelApproval(approval map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-head")),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(approval, "label"))),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(approval, "statusLabel"))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-summary")), gosx.Text(core.WorkbenchViewString(approval, "summary"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(core.WorkbenchViewString(approval, "detail"))),
	}
	if core.WorkbenchViewBool(approval, "hasHref") {
		children = append(children, gosx.El("a", gosx.Attrs(
			gosx.Attr("href", core.WorkbenchViewString(approval, "href")),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(core.WorkbenchViewString(approval, "actionLabel"))))
	}
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(approval, "class"))), gosx.Fragment(children...))
}

func renderPublishPanelScheduleDecision(schedule map[string]any) gosx.Node {
	return gosx.El("article", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(schedule, "class"))),
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-head")),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(schedule, "label"))),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(schedule, "statusLabel"))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__decision-summary")), gosx.Text(core.WorkbenchViewString(schedule, "summary"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(core.WorkbenchViewString(schedule, "detail"))),
	)
}

func renderPublishPanelSummary(view map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-publish-review__summary"),
		gosx.Attr("aria-label", "Publish review summary"),
	),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(view, "readyCountLabel"))), gosx.Text("clear")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(view, "watchCountLabel"))), gosx.Text("watch")),
		gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(view, "nextCountLabel"))), gosx.Text("next")),
	)
}

func renderPublishPanelChecks(checks []map[string]any) gosx.Node {
	nodes := make([]gosx.Node, 0, len(checks))
	for _, check := range checks {
		children := []gosx.Node{
			gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-publish-review__card-head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(check, "label"))),
					gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(check, "scope"))),
				),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(check, "statusLabel"))),
			),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__check-summary")), gosx.Text(core.WorkbenchViewString(check, "summary"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-publish-review__detail")), gosx.Text(core.WorkbenchViewString(check, "detail"))),
		}
		if core.WorkbenchViewBool(check, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", core.WorkbenchViewString(check, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text(core.WorkbenchViewString(check, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(check, "class")),
			gosx.Attr("data-studio-publish-check", core.WorkbenchViewString(check, "key")),
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
			gosx.Attr("class", core.WorkbenchViewString(impact, "class")),
			gosx.Attr("data-studio-publish-impact", core.WorkbenchViewString(impact, "key")),
		),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(impact, "label"))),
				gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(impact, "scope"))),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(impact, "value"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(impact, "detail"))),
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
	if !core.WorkbenchNodeEmpty(node) {
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
	if !core.WorkbenchNodeEmpty(node) {
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
