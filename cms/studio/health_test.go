package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestHealthReportView(t *testing.T) {
	report := NewHealthReport(
		NewHealthCheck("media-alt", "Media alt text", "Media", ReadinessWatch, "2 assets missing alt", "Add alt text before publishing.").WithHref("/admin/media"),
		NewHealthCheck("flows", "Flow handlers", "Forms", ReadinessReady, "4 executable", "All public forms have handlers."),
		HealthCheck{Key: "skip"},
	)
	view := HealthReportView(report)
	if view["summary"] != "1/2 healthy" || view["readyCount"] != 1 || view["watchCount"] != 1 || view["nextCount"] != 0 || view["total"] != 2 {
		t.Fatalf("unexpected view counts: %#v", view)
	}
	checks := view["checks"].([]map[string]any)
	if checks[0]["key"] != "media-alt" || checks[0]["statusLabel"] != "Watch" || checks[0]["actionLabel"] != "Open" || checks[0]["hasHref"] != true {
		t.Fatalf("unexpected first check view: %#v", checks[0])
	}
}

func TestRenderHealthPanel(t *testing.T) {
	html := gosx.RenderHTML(RenderHealthPanel(NewHealthReport(
		NewHealthCheck("copy", "Required copy", "Content", ReadinessReady, "Homepage copy present", "Title, tagline, and hero fields are filled."),
		NewHealthCheck("preview", "Preview secret", "Deployment", ReadinessNext, "Missing secret", "Set a preview signing secret."),
	), HealthPanelOptions{Class: "studio-health"}))
	for _, want := range []string{
		`data-studio-health="true"`,
		`class="studio-health"`,
		`1/2 healthy`,
		`data-studio-health-check="copy"`,
		`Required copy`,
		`Preview secret`,
		`Missing secret`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in health panel html: %s", want, html)
		}
	}
}

func TestRenderHealthPanelEmpty(t *testing.T) {
	html := gosx.RenderHTML(RenderHealthPanel(HealthReport{}, HealthPanelOptions{EmptyTitle: "No checks"}))
	for _, want := range []string{
		`data-studio-health="true"`,
		`No checks`,
		`Register content, media, flow, and deployment checks`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in empty health panel html: %s", want, html)
		}
	}
}

func TestNormalizeHealthReportDefaults(t *testing.T) {
	report := NormalizeHealthReport(HealthReport{Checks: []HealthCheck{
		{Label: "  Flow handlers  ", Scope: " ", Value: " ", Status: "unknown"},
		{Key: "skip"},
	}})
	if len(report.Checks) != 1 {
		t.Fatalf("unexpected checks: %#v", report.Checks)
	}
	check := report.Checks[0]
	if check.Key != "flow-handlers" || check.Label != "Flow handlers" || check.Scope != "Site" || check.Value != "Watch" || check.Status != ReadinessWatch {
		t.Fatalf("unexpected normalized check: %#v", check)
	}
}
