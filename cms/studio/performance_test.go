package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderPerformancePanel(t *testing.T) {
	html := gosx.RenderHTML(RenderPerformancePanel([]PerformanceSignal{
		NewPerformanceSignal("frame-batching", "Frame batching", "rAF", "preview writes <= 1 per frame", ReadinessReady, "Typing and dragging coalesce DOM work."),
		NewPerformanceSignal("media", "Media picker", "210 assets", "<= 250 assets before virtualized picker", ReadinessWatch, "Large libraries should graduate to async search."),
	}, PerformancePanelOptions{Class: "studio-performance"}))
	for _, want := range []string{
		`data-studio-performance="true"`,
		`class="studio-performance"`,
		`1/2 ready`,
		`data-studio-performance-signal="frame-batching"`,
		`preview writes &lt;= 1 per frame`,
		`Frame batching`,
		`Media picker`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in performance panel html: %s", want, html)
		}
	}
}

func TestRenderPerformancePanelEmpty(t *testing.T) {
	html := gosx.RenderHTML(RenderPerformancePanel(nil, PerformancePanelOptions{EmptyTitle: "No signals"}))
	for _, want := range []string{
		`data-studio-performance="true"`,
		`No signals`,
		`Register editor performance signals`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in empty performance panel html: %s", want, html)
		}
	}
}

func TestNormalizePerformanceSignalsSkipsMissingLabels(t *testing.T) {
	signals := normalizePerformanceSignals([]PerformanceSignal{
		{Key: " custom key ", Label: "  Frame  ", Value: " ", Status: "unknown"},
		{Key: "skip"},
	})
	if len(signals) != 1 {
		t.Fatalf("unexpected normalized signals: %#v", signals)
	}
	if signals[0].Key != "custom-key" || signals[0].Label != "Frame" || signals[0].Value != "Tracked" || signals[0].Status != ReadinessWatch {
		t.Fatalf("unexpected signal: %#v", signals[0])
	}
}
