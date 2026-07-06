package studio

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx"
)

type HealthReport struct {
	Checks []HealthCheck
}

type HealthCheck struct {
	Key         string
	Label       string
	Scope       string
	Status      ReadinessStatus
	Value       string
	Detail      string
	Href        string
	ActionLabel string
}

type HealthPanelOptions struct {
	Class       string
	Kicker      string
	Title       string
	EmptyTitle  string
	EmptyDetail string
}

func NewHealthReport(checks ...HealthCheck) HealthReport {
	return NormalizeHealthReport(HealthReport{Checks: checks})
}

func NewHealthCheck(key, label, scope string, status ReadinessStatus, value, detail string) HealthCheck {
	return HealthCheck{
		Key:    key,
		Label:  label,
		Scope:  scope,
		Status: status,
		Value:  value,
		Detail: detail,
	}
}

func (check HealthCheck) WithHref(href string) HealthCheck {
	check.Href = href
	return check
}

func (check HealthCheck) WithActionLabel(label string) HealthCheck {
	check.ActionLabel = label
	return check
}

func NormalizeHealthReport(report HealthReport) HealthReport {
	out := make([]HealthCheck, 0, len(report.Checks))
	for _, check := range report.Checks {
		check.Key = normalizeKey(check.Key)
		check.Label = strings.TrimSpace(check.Label)
		check.Scope = strings.TrimSpace(check.Scope)
		check.Status = normalizeReadinessStatus(check.Status)
		check.Value = strings.TrimSpace(check.Value)
		check.Detail = strings.TrimSpace(check.Detail)
		check.Href = strings.TrimSpace(check.Href)
		check.ActionLabel = strings.TrimSpace(check.ActionLabel)
		if check.Key == "" {
			check.Key = normalizeKey(check.Label)
		}
		if check.Scope == "" {
			check.Scope = "Site"
		}
		if check.Value == "" {
			check.Value = readinessStatusLabel(check.Status)
		}
		if check.ActionLabel == "" {
			check.ActionLabel = readinessActionLabel(check.Status)
		}
		if check.Key == "" || check.Label == "" {
			continue
		}
		out = append(out, check)
	}
	return HealthReport{Checks: out}
}

func (report HealthReport) Counts() (ready, watch, next, total int) {
	normalized := NormalizeHealthReport(report)
	for _, check := range normalized.Checks {
		switch check.Status {
		case ReadinessReady:
			ready++
		case ReadinessNext:
			next++
		default:
			watch++
		}
	}
	return ready, watch, next, len(normalized.Checks)
}

func (report HealthReport) Summary() string {
	ready, _, _, total := report.Counts()
	return fmt.Sprintf("%d/%d healthy", ready, total)
}

func HealthReportView(report HealthReport) map[string]any {
	report = NormalizeHealthReport(report)
	ready, watch, next, total := report.Counts()
	return map[string]any{
		"summary":    fmt.Sprintf("%d/%d healthy", ready, total),
		"readyCount": ready,
		"watchCount": watch,
		"nextCount":  next,
		"total":      total,
		"checks":     healthCheckViews(report.Checks),
	}
}

func RenderHealthPanel(report HealthReport, options HealthPanelOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "studio-health")
	report = NormalizeHealthReport(report)
	ready, watch, next, _ := report.Counts()
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__kicker")), gosx.Text(firstNonEmpty(options.Kicker, "Health"))),
				gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, "Site health"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", className+"__count")), gosx.Text(report.Summary())),
		),
	}
	if len(report.Checks) == 0 {
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("class", className+"__empty")),
			gosx.El("strong", nil, gosx.Text(firstNonEmpty(options.EmptyTitle, "No health checks"))),
			gosx.El("p", nil, gosx.Text(firstNonEmpty(options.EmptyDetail, "Register content, media, flow, and deployment checks to review site health before publish."))),
		))
	} else {
		children = append(children, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", className+"__summary"),
			gosx.Attr("aria-label", "Site health summary"),
		),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(ready))), gosx.Text(" healthy")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(watch))), gosx.Text(" watch")),
			gosx.El("span", nil, gosx.El("strong", nil, gosx.Text(fmt.Sprint(next))), gosx.Text(" next")),
		))
		items := make([]gosx.Node, 0, len(report.Checks))
		for _, check := range report.Checks {
			items = append(items, renderHealthCheck(className, check))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(items...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-health", "true"),
	), gosx.Fragment(children...))
}

func healthCheckViews(checks []HealthCheck) []map[string]any {
	out := make([]map[string]any, 0, len(checks))
	for _, check := range checks {
		status := normalizeReadinessStatus(check.Status)
		out = append(out, map[string]any{
			"key":         check.Key,
			"label":       check.Label,
			"scope":       check.Scope,
			"status":      string(status),
			"statusLabel": readinessStatusLabel(status),
			"class":       "studio-health__card studio-health__card--" + string(status),
			"value":       check.Value,
			"detail":      check.Detail,
			"href":        check.Href,
			"hasHref":     check.Href != "",
			"actionLabel": firstNonEmpty(check.ActionLabel, readinessActionLabel(status)),
		})
	}
	return out
}

func renderHealthCheck(className string, check HealthCheck) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__card-head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(check.Label)),
				gosx.El("span", nil, gosx.Text(check.Scope)),
			),
			gosx.El("output", nil, gosx.Text(readinessStatusLabel(check.Status))),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__value")), gosx.Text(check.Value)),
	}
	if check.Detail != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__detail")), gosx.Text(check.Detail)))
	}
	if check.Href != "" {
		children = append(children, gosx.El("a", gosx.Attrs(
			gosx.Attr("href", check.Href),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(check.ActionLabel)))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", className+"__card "+className+"__card--"+string(check.Status)),
		gosx.Attr("data-studio-health-check", check.Key),
	), gosx.Fragment(children...))
}
