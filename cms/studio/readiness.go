package studio

import (
	"fmt"
	"strings"
)

type ReadinessStatus string

const (
	ReadinessReady ReadinessStatus = "ready"
	ReadinessWatch ReadinessStatus = "watch"
	ReadinessNext  ReadinessStatus = "next"
)

type Readiness struct {
	Items []ReadinessItem
}

type ReadinessItem struct {
	Key         string
	Label       string
	Status      ReadinessStatus
	Summary     string
	Detail      string
	Href        string
	ActionLabel string
}

func NewReadiness(items ...ReadinessItem) Readiness {
	return NormalizeReadiness(Readiness{Items: items})
}

func NewReadinessItem(key, label string, status ReadinessStatus, summary, detail string) ReadinessItem {
	return ReadinessItem{
		Key:     key,
		Label:   label,
		Status:  status,
		Summary: summary,
		Detail:  detail,
	}
}

func (item ReadinessItem) WithHref(href string) ReadinessItem {
	item.Href = href
	return item
}

func (item ReadinessItem) WithActionLabel(label string) ReadinessItem {
	item.ActionLabel = label
	return item
}

func NormalizeReadiness(readiness Readiness) Readiness {
	out := make([]ReadinessItem, 0, len(readiness.Items))
	for _, item := range readiness.Items {
		item.Key = normalizeKey(item.Key)
		item.Label = strings.TrimSpace(item.Label)
		item.Status = normalizeReadinessStatus(item.Status)
		item.Summary = strings.TrimSpace(item.Summary)
		item.Detail = strings.TrimSpace(item.Detail)
		item.Href = strings.TrimSpace(item.Href)
		item.ActionLabel = strings.TrimSpace(item.ActionLabel)
		if item.ActionLabel == "" {
			item.ActionLabel = readinessActionLabel(item.Status)
		}
		if item.Key == "" || item.Label == "" {
			continue
		}
		out = append(out, item)
	}
	return Readiness{Items: out}
}

func (readiness Readiness) Counts() (ready, watch, next, total int) {
	normalized := NormalizeReadiness(readiness)
	for _, item := range normalized.Items {
		switch item.Status {
		case ReadinessReady:
			ready++
		case ReadinessNext:
			next++
		default:
			watch++
		}
	}
	return ready, watch, next, len(normalized.Items)
}

func (readiness Readiness) Summary() string {
	ready, _, _, total := readiness.Counts()
	return fmt.Sprintf("%d/%d ready", ready, total)
}

func ReadinessView(readiness Readiness) map[string]any {
	readiness = NormalizeReadiness(readiness)
	ready, watch, next, total := readiness.Counts()
	return map[string]any{
		"summary":    fmt.Sprintf("%d/%d ready", ready, total),
		"readyCount": ready,
		"watchCount": watch,
		"nextCount":  next,
		"total":      total,
		"items":      readinessItemViews(readiness.Items),
	}
}

func readinessItemViews(items []ReadinessItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		status := normalizeReadinessStatus(item.Status)
		out = append(out, map[string]any{
			"key":         item.Key,
			"label":       item.Label,
			"status":      string(status),
			"statusLabel": readinessStatusLabel(status),
			"class":       "studio-readiness-card studio-readiness-card--" + string(status),
			"summary":     item.Summary,
			"detail":      item.Detail,
			"href":        item.Href,
			"hasHref":     item.Href != "",
			"actionLabel": firstNonEmpty(item.ActionLabel, readinessActionLabel(status)),
		})
	}
	return out
}

func normalizeReadinessStatus(status ReadinessStatus) ReadinessStatus {
	switch status {
	case ReadinessReady, ReadinessWatch, ReadinessNext:
		return status
	default:
		return ReadinessWatch
	}
}

func readinessStatusLabel(status ReadinessStatus) string {
	switch normalizeReadinessStatus(status) {
	case ReadinessReady:
		return "Ready"
	case ReadinessNext:
		return "Next"
	default:
		return "Watch"
	}
}

func readinessActionLabel(status ReadinessStatus) string {
	if normalizeReadinessStatus(status) == ReadinessReady {
		return "Review"
	}
	return "Open"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
