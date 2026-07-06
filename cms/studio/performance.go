package studio

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx"
)

type PerformanceSignal struct {
	Key     string
	Label   string
	Value   string
	Budget  string
	Status  ReadinessStatus
	Summary string
}

type PerformancePanelOptions struct {
	Class       string
	Kicker      string
	Title       string
	EmptyTitle  string
	EmptyDetail string
}

func NewPerformanceSignal(key, label, value, budget string, status ReadinessStatus, summary string) PerformanceSignal {
	return PerformanceSignal{
		Key:     normalizeKey(key),
		Label:   strings.TrimSpace(label),
		Value:   strings.TrimSpace(value),
		Budget:  strings.TrimSpace(budget),
		Status:  normalizeReadinessStatus(status),
		Summary: strings.TrimSpace(summary),
	}
}

func RenderPerformancePanel(signals []PerformanceSignal, options PerformancePanelOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio-performance")
	signals = normalizePerformanceSignals(signals)
	ready, watch, next := performanceCounts(signals)
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__kicker")), gosx.Text(firstNonEmpty(options.Kicker, "Performance"))),
				gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, "Smoothness budgets"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", className+"__count")), gosx.Text(fmt.Sprintf("%d/%d ready", ready, len(signals)))),
		),
	}
	if len(signals) == 0 {
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("class", className+"__empty")),
			gosx.El("strong", nil, gosx.Text(firstNonEmpty(options.EmptyTitle, "No budgets"))),
			gosx.El("p", nil, gosx.Text(firstNonEmpty(options.EmptyDetail, "Register editor performance signals to keep authoring smooth."))),
		))
	} else {
		items := make([]gosx.Node, 0, len(signals))
		for _, signal := range signals {
			items = append(items, renderPerformanceSignal(className, signal))
		}
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", className+"__summary"),
			gosx.Attr("aria-label", "Performance summary"),
		),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(ready))), gosx.Text(" ready")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(watch))), gosx.Text(" watch")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(next))), gosx.Text(" next")),
		))
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(items...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-performance", "true"),
	), gosx.Fragment(children...))
}

func renderPerformanceSignal(className string, signal PerformanceSignal) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__card-head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(signal.Label)),
				gosx.El("span", nil, gosx.Text(signal.Value)),
			),
			gosx.El("output", nil, gosx.Text(readinessStatusLabel(signal.Status))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__budget")), gosx.Text(firstNonEmpty(signal.Budget, "No budget"))),
	}
	if signal.Summary != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__summary-text")), gosx.Text(signal.Summary)))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", className+"__card "+className+"__card--"+string(signal.Status)),
		gosx.Attr("data-studio-performance-signal", signal.Key),
	), gosx.Fragment(children...))
}

func normalizePerformanceSignals(signals []PerformanceSignal) []PerformanceSignal {
	out := make([]PerformanceSignal, 0, len(signals))
	for _, signal := range signals {
		signal.Key = normalizeKey(signal.Key)
		signal.Label = strings.TrimSpace(signal.Label)
		signal.Value = strings.TrimSpace(signal.Value)
		signal.Budget = strings.TrimSpace(signal.Budget)
		signal.Status = normalizeReadinessStatus(signal.Status)
		signal.Summary = strings.TrimSpace(signal.Summary)
		if signal.Key == "" {
			signal.Key = normalizeKey(signal.Label)
		}
		if signal.Label == "" {
			continue
		}
		if signal.Value == "" {
			signal.Value = "Tracked"
		}
		out = append(out, signal)
	}
	return out
}

func performanceCounts(signals []PerformanceSignal) (ready, watch, next int) {
	for _, signal := range signals {
		switch normalizeReadinessStatus(signal.Status) {
		case ReadinessReady:
			ready++
		case ReadinessNext:
			next++
		default:
			watch++
		}
	}
	return ready, watch, next
}
