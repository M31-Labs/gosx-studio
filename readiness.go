package studio

import (
	"fmt"
	"strings"
)

type ShellReadinessStatus string

const (
	ShellReadinessReady ShellReadinessStatus = "ready"
	ShellReadinessWatch ShellReadinessStatus = "watch"
	ShellReadinessNext  ShellReadinessStatus = "next"
)

type ShellReadiness struct {
	Items []ShellReadinessItem
}

type ShellReadinessItem struct {
	Key         string
	Label       string
	Status      ShellReadinessStatus
	Summary     string
	Detail      string
	Href        string
	ActionLabel string
}

func NewShellReadiness(items ...ShellReadinessItem) ShellReadiness {
	return NormalizeShellReadiness(ShellReadiness{Items: items})
}

func NewShellReadinessItem(key, label string, status ShellReadinessStatus, summary, detail string) ShellReadinessItem {
	return ShellReadinessItem{
		Key:     key,
		Label:   label,
		Status:  status,
		Summary: summary,
		Detail:  detail,
	}
}

func (item ShellReadinessItem) WithHref(href string) ShellReadinessItem {
	item.Href = href
	return item
}

func (item ShellReadinessItem) WithActionLabel(label string) ShellReadinessItem {
	item.ActionLabel = label
	return item
}

func NormalizeShellReadiness(readiness ShellReadiness) ShellReadiness {
	out := make([]ShellReadinessItem, 0, len(readiness.Items))
	for _, item := range readiness.Items {
		item.Key = NormalizeKey(item.Key)
		item.Label = strings.TrimSpace(item.Label)
		item.Status = NormalizeShellReadinessStatus(item.Status)
		item.Summary = strings.TrimSpace(item.Summary)
		item.Detail = strings.TrimSpace(item.Detail)
		item.Href = strings.TrimSpace(item.Href)
		item.ActionLabel = strings.TrimSpace(item.ActionLabel)
		if item.ActionLabel == "" {
			item.ActionLabel = ShellReadinessActionLabel(item.Status)
		}
		if item.Key == "" || item.Label == "" {
			continue
		}
		out = append(out, item)
	}
	return ShellReadiness{Items: out}
}

func (readiness ShellReadiness) Counts() (ready, watch, next, total int) {
	normalized := NormalizeShellReadiness(readiness)
	for _, item := range normalized.Items {
		switch item.Status {
		case ShellReadinessReady:
			ready++
		case ShellReadinessNext:
			next++
		default:
			watch++
		}
	}
	return ready, watch, next, len(normalized.Items)
}

func (readiness ShellReadiness) Summary() string {
	ready, _, _, total := readiness.Counts()
	return fmt.Sprintf("%d/%d ready", ready, total)
}

func ShellReadinessView(readiness ShellReadiness) map[string]any {
	readiness = NormalizeShellReadiness(readiness)
	ready, watch, next, total := readiness.Counts()
	return map[string]any{
		"summary":    fmt.Sprintf("%d/%d ready", ready, total),
		"readyCount": ready,
		"watchCount": watch,
		"nextCount":  next,
		"total":      total,
		"items":      shellReadinessItemViews(readiness.Items),
	}
}

func shellReadinessItemViews(items []ShellReadinessItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		status := NormalizeShellReadinessStatus(item.Status)
		out = append(out, map[string]any{
			"key":         item.Key,
			"label":       item.Label,
			"status":      string(status),
			"statusLabel": ShellReadinessStatusLabel(status),
			"class":       "studio-readiness-card studio-readiness-card--" + string(status),
			"summary":     item.Summary,
			"detail":      item.Detail,
			"href":        item.Href,
			"hasHref":     item.Href != "",
			"actionLabel": FirstNonEmpty(item.ActionLabel, ShellReadinessActionLabel(status)),
		})
	}
	return out
}

func NormalizeShellReadinessStatus(status ShellReadinessStatus) ShellReadinessStatus {
	switch status {
	case ShellReadinessReady, ShellReadinessWatch, ShellReadinessNext:
		return status
	default:
		return ShellReadinessWatch
	}
}

func ShellReadinessStatusLabel(status ShellReadinessStatus) string {
	switch NormalizeShellReadinessStatus(status) {
	case ShellReadinessReady:
		return "Ready"
	case ShellReadinessNext:
		return "Next"
	default:
		return "Watch"
	}
}

func ShellReadinessActionLabel(status ShellReadinessStatus) string {
	if NormalizeShellReadinessStatus(status) == ShellReadinessReady {
		return "Review"
	}
	return "Open"
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
