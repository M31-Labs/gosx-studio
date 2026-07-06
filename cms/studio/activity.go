package studio

import (
	"fmt"

	"m31labs.dev/gosx"
	studiocollab "m31labs.dev/gosx-studio/cms/studio/collab"
)

type ActivityOptions struct {
	Class           string
	ReadinessLabel  string
	ExtraPanels     []gosx.Node
	ProposalOptions studiocollab.RenderProposalOptions
	CommentOptions  studiocollab.RenderCommentOptions
	Collapsed       bool
}

func RenderActivityPanel(readiness Readiness, proposals studiocollab.Snapshot, options ActivityOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "studio-activity-drawer")
	readiness = NormalizeReadiness(readiness)
	score := readiness.Summary()
	if len(readiness.Items) == 0 {
		score = "0/0 ready"
	}
	ready, watch, next, _ := readiness.Counts()
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-activity-head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Activity")),
				gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.ReadinessLabel, "Readiness"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", "studio-readiness-score")), gosx.Text(score)),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("data-studio-activity-toggle", "true"),
				gosx.Attr("aria-pressed", boolAttr(!options.Collapsed)),
			), gosx.Text(activityToggleLabel(options.Collapsed))),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-readiness-summary"),
			gosx.Attr("aria-label", "Readiness summary"),
		),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(ready))), gosx.Text(" ready")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(watch))), gosx.Text(" watch")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(next))), gosx.Text(" next")),
		),
		renderReadinessList(readiness),
	}
	children = append(children, options.ExtraPanels...)
	children = append(children,
		studiocollab.RenderCommentPanel(proposals, options.CommentOptions),
		studiocollab.RenderProposalPanel(proposals, options.ProposalOptions),
	)
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-activity-drawer", "true"),
		gosx.Attr("aria-label", "Activity and readiness"),
	), gosx.Fragment(children...))
}

func renderReadinessList(readiness Readiness) gosx.Node {
	readiness = NormalizeReadiness(readiness)
	items := make([]gosx.Node, 0, len(readiness.Items))
	for _, item := range readiness.Items {
		children := []gosx.Node{
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-card__head")),
				gosx.El("div", nil,
					gosx.El("strong", nil, gosx.Text(item.Label)),
					gosx.El("span", nil, gosx.Text(item.Summary)),
				),
				gosx.El("output", nil, gosx.Text(readinessStatusLabel(item.Status))),
			),
		}
		if item.Detail != "" {
			children = append(children, gosx.El("p", nil, gosx.Text(item.Detail)))
		}
		if item.Href != "" {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("href", item.Href),
				gosx.Attr("data-gosx-link", "true"),
			), gosx.Text(item.ActionLabel)))
		}
		items = append(items, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", "studio-readiness-card studio-readiness-card--"+string(item.Status)),
			gosx.Attr("data-readiness-key", item.Key),
		), gosx.Fragment(children...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-readiness-list")), gosx.Fragment(items...))
}

func activityToggleLabel(collapsed bool) string {
	if collapsed {
		return "Show"
	}
	return "Hide"
}
