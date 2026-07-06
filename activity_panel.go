package studio

import (
	"fmt"

	"m31labs.dev/gosx"
)

type ActivityPanelOptions struct {
	RootAttrs map[string]any
}

func RenderActivityPanel(view map[string]any, options ActivityPanelOptions) gosx.Node {
	attrs := []any{
		gosx.Attr("class", workbenchMapString(view, "class")),
		gosx.Attr("data-studio-activity-drawer", "true"),
		gosx.Attr("aria-label", "Activity and readiness"),
		gosx.Attr("data-gosx-studio-activity-panel-renderer", "gosx-studio"),
	}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)

	children := []gosx.Node{
		renderActivityPanelHead(view),
		renderActivityReadinessSummary(workbenchViewMap(view, "readiness")),
		renderActivityReadinessList(workbenchViewMapList(workbenchViewMap(view, "readiness"), "items")),
	}
	if workbenchMapBool(view, "hasHealth") {
		children = append(children, renderActivityHealthPanel(workbenchViewMap(view, "health")))
	}
	if workbenchMapBool(view, "hasPerformance") {
		children = append(children, renderActivityPerformancePanel(workbenchViewMap(view, "performance")))
	}
	children = append(children,
		renderActivityCommentPanel(workbenchViewMap(view, "comments")),
		renderActivityProposalPanel(workbenchViewMap(view, "proposals")),
	)

	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func renderActivityPanelHead(view map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-activity-head")),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(workbenchMapString(view, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(view, "title"))),
		),
		gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-readiness-score")), gosx.Text(workbenchMapString(view, "score"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-activity-toggle", "true"),
			gosx.Attr("aria-pressed", BoolAttr(workbenchMapBool(view, "togglePressed"))),
		), gosx.Text(workbenchMapString(view, "toggleLabel"))),
	)
}

func renderActivityReadinessSummary(readiness map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-readiness-summary"),
		gosx.Attr("aria-label", "Readiness summary"),
	),
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "readyCount"))),
			gosx.Text("ready"),
		),
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "watchCount"))),
			gosx.Text("watch"),
		),
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(readiness, "nextCount"))),
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
					gosx.El("strong", nil, gosx.Text(workbenchMapString(item, "label"))),
					gosx.El("span", nil, gosx.Text(workbenchMapString(item, "summary"))),
				),
				gosx.El("output", nil, gosx.Text(workbenchMapString(item, "statusLabel"))),
			),
		}
		if detail := workbenchMapString(item, "detail"); detail != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(detail)))
		}
		if workbenchMapBool(item, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", workbenchMapString(item, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text(workbenchMapString(item, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(item, "class")),
			gosx.Attr("data-readiness-key", workbenchMapString(item, "key")),
		), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-list")), gosx.Fragment(nodes...))
}

func renderActivityHealthPanel(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(panel, "class")),
		gosx.Attr("data-studio-health", "true"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__kicker")), gosx.Text(workbenchMapString(panel, "kicker"))),
				gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-health__count")), gosx.Text(workbenchMapString(panel, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-health__summary"),
			gosx.Attr("aria-label", "Site health summary"),
		),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "readyCount"))),
				gosx.Text("healthy"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "watchCount"))),
				gosx.Text("watch"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "nextCount"))),
				gosx.Text("next"),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__list")),
			gosx.Fragment(renderActivityHealthChecks(workbenchViewMapList(panel, "checks"))...),
		),
	)
}

func renderActivityHealthChecks(checks []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(checks))
	for _, check := range checks {
		children := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-health__card-head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(workbenchMapString(check, "label"))),
					gosx.El("span", nil, gosx.Text(workbenchMapString(check, "scope"))),
				),
				gosx.El("output", nil, gosx.Text(workbenchMapString(check, "statusLabel"))),
			),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__value")), gosx.Text(workbenchMapString(check, "value"))),
		}
		if detail := workbenchMapString(check, "detail"); detail != "" {
			children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-health__detail")), gosx.Text(detail)))
		}
		if workbenchMapBool(check, "hasHref") {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", workbenchMapString(check, "href")),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text(workbenchMapString(check, "actionLabel"))))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(check, "class")),
			gosx.Attr("data-studio-health-check", workbenchMapString(check, "key")),
		), gosx.Fragment(children...)))
	}
	return nodes
}

func renderActivityPerformancePanel(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", workbenchMapString(panel, "class")),
		gosx.Attr("data-studio-performance", "true"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__kicker")), gosx.Text(workbenchMapString(panel, "kicker"))),
				gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-performance__count")), gosx.Text(workbenchMapString(panel, "summary"))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-performance__summary"),
			gosx.Attr("aria-label", "Performance summary"),
		),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "readyCount"))),
				gosx.Text("ready"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "watchCount"))),
				gosx.Text("watch"),
			),
			gosx.El("span", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "nextCount"))),
				gosx.Text("next"),
			),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__list")),
			gosx.Fragment(renderActivityPerformanceSignals(workbenchViewMapList(panel, "signals"))...),
		),
	)
}

func renderActivityPerformanceSignals(signals []map[string]any) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(signals))
	for _, signal := range signals {
		children := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-performance__card-head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(workbenchMapString(signal, "label"))),
					gosx.El("span", nil, gosx.Text(workbenchMapString(signal, "value"))),
				),
				gosx.El("output", nil, gosx.Text(workbenchMapString(signal, "statusLabel"))),
			),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__budget")), gosx.Text(workbenchMapString(signal, "budget"))),
		}
		if summary := workbenchMapString(signal, "summary"); summary != "" {
			children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "studio-performance__summary-text")), gosx.Text(summary)))
		}
		nodes = append(nodes, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", workbenchMapString(signal, "class")),
			gosx.Attr("data-studio-performance-signal", workbenchMapString(signal, "key")),
		), gosx.Fragment(children...)))
	}
	return nodes
}

func renderActivityEmptyPanel(panel map[string]any) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "class"))),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "headClass"))),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "kickerClass"))), gosx.Text(workbenchMapString(panel, "kicker"))),
				gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "countClass"))), gosx.Text(workbenchMapString(panel, "countLabel"))),
		),
		gosx.El("article", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "emptyClass"))),
			gosx.El("strong", nil, gosx.Text(workbenchMapString(panel, "emptyTitle"))),
			gosx.El("p", nil, gosx.Text(workbenchMapString(panel, "emptyDetail"))),
		),
	)
}

func renderActivitySubpanelHead(panel map[string]any) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "headClass"))),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "kickerClass"))), gosx.Text(workbenchMapString(panel, "kicker"))),
			gosx.El("h2", nil, gosx.Text(workbenchMapString(panel, "title"))),
		),
		gosx.El("output", gosx.Attrs(gosx.Attr("class", workbenchMapString(panel, "countClass"))), gosx.Text(workbenchMapString(panel, "countLabel"))),
	)
}

func renderActivityCommentPanel(panel map[string]any) gosx.Node {
	comments := activityPanelItems(panel, "comments")
	if len(comments) == 0 {
		return renderActivityEmptyPanel(panel)
	}
	className := workbenchMapString(panel, "class")
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
	className := workbenchMapString(panel, "class")
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
				gosx.El("strong", nil, gosx.Text(firstNonEmpty(workbenchMapString(comment, "targetLabel"), "Page"))),
				gosx.El("span", nil, gosx.Text(activityCommentMeta(comment))),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(comment, "statusLabel"))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__body")), gosx.Text(workbenchMapString(comment, "body"))),
	}
	if reason := workbenchMapString(comment, "decisionReason"); reason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__decision")), gosx.Text(reason)))
	}
	if workbenchMapBool(comment, "canResolve") || workbenchMapBool(comment, "canReopen") {
		children = append(children, renderActivityCommentActions(className, comment))
	}
	status := workbenchMapString(comment, "status")
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", activityPanelCardClass(className, status)),
		gosx.Attr("data-studio-comment", workbenchMapString(comment, "id")),
		gosx.Attr("data-studio-comment-status", status),
		gosx.Attr("data-studio-comment-block", workbenchMapString(comment, "blockID")),
		gosx.Attr("data-studio-comment-field", workbenchMapString(comment, "field")),
	), gosx.Fragment(children...))
}

func renderActivityCommentActions(className string, comment map[string]any) gosx.Node {
	children := []gosx.Node{}
	if workbenchMapBool(comment, "canResolve") {
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
	if workbenchMapBool(comment, "canReopen") {
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
		gosx.Attr("data-studio-"+options.DataPrefix+"-id", workbenchMapString(item, options.IDKey)),
		gosx.Attr("data-studio-"+options.DataPrefix+"-event", workbenchMapString(item, options.EventKey)),
	}
	formID := workbenchMapString(item, "formID")
	action := workbenchMapString(item, "decisionAction")
	name := workbenchMapString(item, "decisionInputName")
	value := workbenchMapString(item, options.ValueKey)
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
				gosx.El("strong", nil, gosx.Text(firstNonEmpty(workbenchMapString(proposal, "title"), "Untitled suggestion"))),
				gosx.El("span", nil, gosx.Text(activityProposalMeta(proposal))),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(proposal, "statusLabel"))),
		),
	}
	if summary := workbenchMapString(proposal, "summary"); summary != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__summary")), gosx.Text(summary)))
	}
	if review := workbenchMapString(proposal, "reviewSummary"); review != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__review")), gosx.Text(review)))
	}
	if items := activityPanelItems(proposal, "items"); len(items) > 0 {
		nodes := make([]gosx.Node, 0, len(items))
		for _, item := range items {
			nodes = append(nodes, gosx.El("li", gosx.Attrs(
				gosx.Attr("data-studio-proposal-operation", workbenchMapString(item, "operationID")),
				gosx.Attr("data-studio-proposal-operation-kind", workbenchMapString(item, "kind")),
			), gosx.Text(workbenchMapString(item, "summary"))))
		}
		children = append(children, gosx.El("ol", gosx.Attrs(gosx.Attr("class", className+"__ops")), gosx.Fragment(nodes...)))
	}
	if workbenchMapBool(proposal, "canAccept") || workbenchMapBool(proposal, "canReject") {
		children = append(children, renderActivityProposalActions(className, proposal))
	} else if reason := workbenchMapString(proposal, "decisionReason"); reason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__decision")), gosx.Text(reason)))
	}
	status := workbenchMapString(proposal, "status")
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", activityPanelCardClass(className, status)),
		gosx.Attr("data-studio-proposal", workbenchMapString(proposal, "id")),
		gosx.Attr("data-studio-proposal-status", status),
	), gosx.Fragment(children...))
}

func renderActivityProposalActions(className string, proposal map[string]any) gosx.Node {
	children := []gosx.Node{}
	if workbenchMapBool(proposal, "canAccept") {
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
	if workbenchMapBool(proposal, "canReject") {
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
	if actor := workbenchMapString(comment, "decisionActorID"); actor != "" {
		return fmt.Sprintf("%s by %s", workbenchMapString(comment, "statusLabel"), actor)
	}
	if actor := workbenchMapString(comment, "actorID"); actor != "" {
		return "From " + actor
	}
	return "Review note"
}

func activityProposalMeta(proposal map[string]any) string {
	if actor := workbenchMapString(proposal, "decisionActorID"); actor != "" {
		return fmt.Sprintf("%s by %s", workbenchMapString(proposal, "statusLabel"), actor)
	}
	count := activityPanelInt(proposal, "operationCount")
	if actor := workbenchMapString(proposal, "actorID"); actor != "" {
		return fmt.Sprintf("%d operations from %s", count, actor)
	}
	return fmt.Sprintf("%d operations", count)
}

func activityPanelItems(view map[string]any, key string) []map[string]any {
	if items := workbenchViewMapList(view, key); len(items) > 0 {
		return items
	}
	return workbenchViewMapList(view, "items")
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
