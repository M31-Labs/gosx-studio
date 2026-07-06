package panels

import (
	"fmt"

	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/core"
)

type ActivityPanelOptions struct {
	RootAttrs map[string]any
}

func RenderActivityPanel(view map[string]any, options ActivityPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", core.WorkbenchViewString(view, "class")),
		gosx.Attr("data-studio-activity-drawer", "true"),
		gosx.Attr("aria-label", "Activity and readiness"),
		gosx.Attr("data-gosx-studio-activity-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		renderActivityPanelHead(view),
		renderActivityReadinessSummary(core.WorkbenchViewMap(view, "readiness")),
		renderActivityReadinessList(core.WorkbenchViewMapList(core.WorkbenchViewMap(view, "readiness"), "items")),
	}
	if core.WorkbenchViewBool(view, "hasHealth") {
		children = append(children, renderActivityHealthPanel(core.WorkbenchViewMap(view, "health")))
	}
	if core.WorkbenchViewBool(view, "hasPerformance") {
		children = append(children, renderActivityPerformancePanel(core.WorkbenchViewMap(view, "performance")))
	}
	children = append(children,
		renderActivityCommentPanel(core.WorkbenchViewMap(view, "comments")),
		renderActivityProposalPanel(core.WorkbenchViewMap(view, "proposals")),
	)

	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func renderActivityPanelHead(view map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-activity-head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(core.WorkbenchViewString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(view, "title"))),
		),
		gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-readiness-score")), gosx.Text(core.WorkbenchViewString(view, "score"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-activity-toggle", "true"),
			gosx.Attr("aria-pressed", core.BoolAttr(core.WorkbenchViewBool(view, "togglePressed"))),
		), gosx.Text(core.WorkbenchViewString(view, "toggleLabel"))),
	)
}

func renderActivityReadinessSummary(readiness map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-readiness-summary"),
		gosx.Attr("aria-label", "Readiness summary"),
	),
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(readiness, "readyCount"))),
			gosx.Text("ready"),
		),
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(readiness, "watchCount"))),
			gosx.Text("watch"),
		),
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(readiness, "nextCount"))),
			gosx.Text("next"),
		),
	)
}

func renderActivityReadinessList(items []map[string]any) gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		children := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-card__head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(item, "label"))),
					gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(item, "summary"))),
				),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(item, "statusLabel"))),
			),
		}
		if detail := core.WorkbenchViewString(item, "detail"); detail != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(detail)))
		}
		if core.WorkbenchViewBool(item, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", core.WorkbenchViewString(item, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text(core.WorkbenchViewString(item, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(item, "class")),
			gosx.Attr("data-readiness-key", core.WorkbenchViewString(item, "key")),
		), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-list")), gosx.Fragment(nodes...))
}

func renderActivityHealthPanel(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(panel, "class")),
		gosx.Attr("data-studio-health", "true"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__kicker")), gosx.Text(core.WorkbenchViewString(panel, "kicker"))),
				gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(panel, "title"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-health__count")), gosx.Text(core.WorkbenchViewString(panel, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-health__summary"),
			gosx.Attr("aria-label", "Site health summary"),
		),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(panel, "readyCount"))),
				gosx.Text("healthy"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(panel, "watchCount"))),
				gosx.Text("watch"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(panel, "nextCount"))),
				gosx.Text("next"),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__list")),
			gosx.Fragment(renderActivityHealthChecks(core.WorkbenchViewMapList(panel, "checks"))...),
		),
	)
}

func renderActivityHealthChecks(checks []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(checks))
	for _, check := range checks {
		children := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__card-head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(check, "label"))),
					gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(check, "scope"))),
				),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(check, "statusLabel"))),
			),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__value")), gosx.Text(core.WorkbenchViewString(check, "value"))),
		}
		if detail := core.WorkbenchViewString(check, "detail"); detail != "" {
			children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__detail")), gosx.Text(detail)))
		}
		if core.WorkbenchViewBool(check, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", core.WorkbenchViewString(check, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text(core.WorkbenchViewString(check, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(check, "class")),
			gosx.Attr("data-studio-health-check", core.WorkbenchViewString(check, "key")),
		), gosx.Fragment(children...)))
	}
	return nodes
}

func renderActivityPerformancePanel(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", core.WorkbenchViewString(panel, "class")),
		gosx.Attr("data-studio-performance", "true"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__kicker")), gosx.Text(core.WorkbenchViewString(panel, "kicker"))),
				gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(panel, "title"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-performance__count")), gosx.Text(core.WorkbenchViewString(panel, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-performance__summary"),
			gosx.Attr("aria-label", "Performance summary"),
		),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(panel, "readyCount"))),
				gosx.Text("ready"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(panel, "watchCount"))),
				gosx.Text("watch"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(panel, "nextCount"))),
				gosx.Text("next"),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__list")),
			gosx.Fragment(renderActivityPerformanceSignals(core.WorkbenchViewMapList(panel, "signals"))...),
		),
	)
}

func renderActivityPerformanceSignals(signals []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(signals))
	for _, signal := range signals {
		children := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__card-head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(signal, "label"))),
					gosx.El("span", nil, gosx.Text(core.WorkbenchViewString(signal, "value"))),
				),
				gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(signal, "statusLabel"))),
			),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__budget")), gosx.Text(core.WorkbenchViewString(signal, "budget"))),
		}
		if summary := core.WorkbenchViewString(signal, "summary"); summary != "" {
			children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__summary-text")), gosx.Text(summary)))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", core.WorkbenchViewString(signal, "class")),
			gosx.Attr("data-studio-performance-signal", core.WorkbenchViewString(signal, "key")),
		), gosx.Fragment(children...)))
	}
	return nodes
}

func renderActivityEmptyPanel(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "class"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "headClass"))),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "kickerClass"))), gosx.Text(core.WorkbenchViewString(panel, "kicker"))),
				gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(panel, "title"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "countClass"))), gosx.Text(core.WorkbenchViewString(panel, "countLabel"))),
		),
		gosx.El("article", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "emptyClass"))),
			gosx.El("strong", nil, gosx.Text(core.WorkbenchViewString(panel, "emptyTitle"))),
			gosx.El("p", nil, gosx.Text(core.WorkbenchViewString(panel, "emptyDetail"))),
		),
	)
}

func renderActivitySubpanelHead(panel map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "headClass"))),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "kickerClass"))), gosx.Text(core.WorkbenchViewString(panel, "kicker"))),
			gosx.El("h2", nil, gosx.Text(core.WorkbenchViewString(panel, "title"))),
		),
		gosx.El("output", gosx.Attrs(gosx.Attr("class", core.WorkbenchViewString(panel, "countClass"))), gosx.Text(core.WorkbenchViewString(panel, "countLabel"))),
	)
}

func renderActivityCommentPanel(panel map[string]any) gosx.Node {
	comments := activityPanelItems(panel, "comments")
	if len(comments) == 0 {
		return renderActivityEmptyPanel(panel)
	}
	className := core.WorkbenchViewString(panel, "class")
	cards := make([]gosx.Node, 0, len(comments))
	for _, comment := range comments {
		cards = append(cards, renderActivityCommentCard(className, comment))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", className)),
		renderActivitySubpanelHead(panel),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(cards...)),
	)
}

func renderActivityProposalPanel(panel map[string]any) gosx.Node {
	proposals := activityPanelItems(panel, "proposals")
	if len(proposals) == 0 {
		return renderActivityEmptyPanel(panel)
	}
	className := core.WorkbenchViewString(panel, "class")
	cards := make([]gosx.Node, 0, len(proposals))
	for _, proposal := range proposals {
		cards = append(cards, renderActivityProposalCard(className, proposal))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", className)),
		renderActivitySubpanelHead(panel),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(cards...)),
	)
}

func renderActivityCommentCard(className string, comment map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__card-head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(firstNonEmpty(core.WorkbenchViewString(comment, "targetLabel"), "Page"))),
				gosx.El("span", nil, gosx.Text(activityCommentMeta(comment))),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(comment, "statusLabel"))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__body")), gosx.Text(core.WorkbenchViewString(comment, "body"))),
	}
	if reason := core.WorkbenchViewString(comment, "decisionReason"); reason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__decision")), gosx.Text(reason)))
	}
	if core.WorkbenchViewBool(comment, "canResolve") || core.WorkbenchViewBool(comment, "canReopen") {
		children = append(children, renderActivityCommentActions(className, comment))
	}
	status := core.WorkbenchViewString(comment, "status")
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", activityPanelCardClass(className, status)),
		gosx.Attr("data-studio-comment", core.WorkbenchViewString(comment, "id")),
		gosx.Attr("data-studio-comment-status", status),
		gosx.Attr("data-studio-comment-block", core.WorkbenchViewString(comment, "blockID")),
		gosx.Attr("data-studio-comment-field", core.WorkbenchViewString(comment, "field")),
	), gosx.Fragment(children...))
}

func renderActivityCommentActions(className string, comment map[string]any) gosx.Node {
	children := []gosx.Node{}
	if core.WorkbenchViewBool(comment, "canResolve") {
		attrs := activityDecisionButtonAttrs(comment, activityDecisionButtonOptions{
			ClassName:  className + "__resolve",
			DataPrefix: "comment",
			DataAction: "resolve",
			IDKey:      "id",
			EventKey:   "resolveEvent",
			ValueKey:   "resolveDecisionValue",
		})
		children = append(children, gosx.El("button", gosx.Attrs(attrs...), gosx.Text("Resolve")))
	}
	if core.WorkbenchViewBool(comment, "canReopen") {
		attrs := activityDecisionButtonAttrs(comment, activityDecisionButtonOptions{
			ClassName:  className + "__reopen",
			DataPrefix: "comment",
			DataAction: "reopen",
			IDKey:      "id",
			EventKey:   "reopenEvent",
			ValueKey:   "reopenDecisionValue",
		})
		children = append(children, gosx.El("button", gosx.Attrs(attrs...), gosx.Text("Reopen")))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__actions")), gosx.Fragment(children...))
}

type activityDecisionButtonOptions struct {
	ClassName  string
	DataPrefix string
	DataAction string
	IDKey      string
	EventKey   string
	ValueKey   string
}

func activityDecisionButtonAttrs(item map[string]any, options activityDecisionButtonOptions) []any {
	attrs := []any{
		gosx.Attr("type", "button"),
		gosx.Attr("class", options.ClassName),
		gosx.Attr("data-studio-"+options.DataPrefix+"-action", options.DataAction),
		gosx.Attr("data-studio-"+options.DataPrefix+"-id", core.WorkbenchViewString(item, options.IDKey)),
		gosx.Attr("data-studio-"+options.DataPrefix+"-event", core.WorkbenchViewString(item, options.EventKey)),
	}
	formID := core.WorkbenchViewString(item, "formID")
	action := core.WorkbenchViewString(item, "decisionAction")
	name := core.WorkbenchViewString(item, "decisionInputName")
	value := core.WorkbenchViewString(item, options.ValueKey)
	if formID == "" || action == "" || name == "" || value == "" {
		return attrs
	}
	attrs[0] = gosx.Attr("type", "submit")
	return append(attrs,
		gosx.Attr("form", formID),
		gosx.Attr("formaction", action),
		gosx.Attr("formmethod", "post"),
		gosx.Attr("name", name),
		gosx.Attr("value", value),
	)
}

func renderActivityProposalCard(className string, proposal map[string]any) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__card-head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(firstNonEmpty(core.WorkbenchViewString(proposal, "title"), "Untitled suggestion"))),
				gosx.El("span", nil, gosx.Text(activityProposalMeta(proposal))),
			),
			gosx.El("output", nil, gosx.Text(core.WorkbenchViewString(proposal, "statusLabel"))),
		),
	}
	if summary := core.WorkbenchViewString(proposal, "summary"); summary != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__summary")), gosx.Text(summary)))
	}
	if review := core.WorkbenchViewString(proposal, "reviewSummary"); review != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__review")), gosx.Text(review)))
	}
	if items := activityPanelItems(proposal, "items"); len(items) > 0 {
		nodes := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			nodes = append(nodes, gosx.El("li", gosx.Attrs(
				gosx.Attr("data-studio-proposal-operation", core.WorkbenchViewString(item, "operationID")),
				gosx.Attr("data-studio-proposal-operation-kind", core.WorkbenchViewString(item, "kind")),
			), gosx.Text(core.WorkbenchViewString(item, "summary"))))
		}
		children = append(children, gosx.El("ol", gosx.Attrs(gosx.Attr("class", className+"__ops")), gosx.Fragment(nodes...)))
	}
	if core.WorkbenchViewBool(proposal, "canAccept") || core.WorkbenchViewBool(proposal, "canReject") {
		children = append(children, renderActivityProposalActions(className, proposal))
	} else if reason := core.WorkbenchViewString(proposal, "decisionReason"); reason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__decision")), gosx.Text(reason)))
	}
	status := core.WorkbenchViewString(proposal, "status")
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", activityPanelCardClass(className, status)),
		gosx.Attr("data-studio-proposal", core.WorkbenchViewString(proposal, "id")),
		gosx.Attr("data-studio-proposal-status", status),
	), gosx.Fragment(children...))
}

func renderActivityProposalActions(className string, proposal map[string]any) gosx.Node {
	children := []gosx.Node{}
	if core.WorkbenchViewBool(proposal, "canAccept") {
		attrs := activityDecisionButtonAttrs(proposal, activityDecisionButtonOptions{
			ClassName:  className + "__accept",
			DataPrefix: "proposal",
			DataAction: "accept",
			IDKey:      "id",
			EventKey:   "acceptEvent",
			ValueKey:   "acceptDecisionValue",
		})
		children = append(children, gosx.El("button", gosx.Attrs(attrs...), gosx.Text("Accept")))
	}
	if core.WorkbenchViewBool(proposal, "canReject") {
		attrs := activityDecisionButtonAttrs(proposal, activityDecisionButtonOptions{
			ClassName:  className + "__reject",
			DataPrefix: "proposal",
			DataAction: "reject",
			IDKey:      "id",
			EventKey:   "rejectEvent",
			ValueKey:   "rejectDecisionValue",
		})
		children = append(children, gosx.El("button", gosx.Attrs(attrs...), gosx.Text("Reject")))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__actions")), gosx.Fragment(children...))
}

func activityCommentMeta(comment map[string]any) string {
	if actor := core.WorkbenchViewString(comment, "decisionActorID"); actor != "" {
		return fmt.Sprintf("%s by %s", core.WorkbenchViewString(comment, "statusLabel"), actor)
	}
	if actor := core.WorkbenchViewString(comment, "actorID"); actor != "" {
		return "From " + actor
	}
	return "Review note"
}

func activityProposalMeta(proposal map[string]any) string {
	if actor := core.WorkbenchViewString(proposal, "decisionActorID"); actor != "" {
		return fmt.Sprintf("%s by %s", core.WorkbenchViewString(proposal, "statusLabel"), actor)
	}
	count := activityPanelInt(proposal, "operationCount")
	if actor := core.WorkbenchViewString(proposal, "actorID"); actor != "" {
		return fmt.Sprintf("%d operations from %s", count, actor)
	}
	return fmt.Sprintf("%d operations", count)
}

func activityPanelItems(view map[string]any, key string) []map[string]any {
	if items := core.WorkbenchViewMapList(view, key); len(items) > 0 {
		return items
	}
	return core.WorkbenchViewMapList(view, "items")
}

func activityPanelCardClass(className, status string) string {
	if status == "" {
		return className + "__card"
	}
	return className + "__card " + className + "__card--" + status
}

func activityPanelInt(values map[string]any, key string) int {
	switch typed := values[key].(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		var out int
		_, _ = fmt.Sscanf(typed, "%d", &out)
		return out
	default:
		return 0
	}
}
