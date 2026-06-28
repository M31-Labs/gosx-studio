package studio

import "m31labs.dev/gosx"

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
		renderActivityEmptyPanel(workbenchViewMap(view, "comments")),
		renderActivityEmptyPanel(workbenchViewMap(view, "proposals")),
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
