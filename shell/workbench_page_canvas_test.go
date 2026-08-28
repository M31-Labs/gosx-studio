package shell

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderWorkbenchPageCanvasOwnsCanonicalFrameRouteViewportAndDiagnostics(t *testing.T) {
	html := gosx.RenderHTML(RenderWorkbenchPageCanvas(WorkbenchPageCanvasOptions{
		View: map[string]any{
			"selectionLabel": "Hero",
			"viewportKey":    "desktop",
			"viewports": []map[string]any{
				{"key": "desktop", "label": "Desktop", "width": "100%", "pressed": true},
				{"key": "mobile", "label": "Mobile", "width": "24rem"},
			},
		},
		URL:         "/?gosx-preview=1&amp;unsafe=escaped",
		Title:       "Noni <Home>",
		IFrameTitle: `Noni "Home" preview`,
		Routes: []WorkbenchPageCanvasRoute{
			{Key: "home", Label: "Home", URL: "/?gosx-preview=1", Active: true},
			{Key: "about", Label: "About", URL: "/about?gosx-preview=1"},
		},
	}))
	for _, fragment := range []string{
		`data-gosx-studio-page-canvas="true"`,
		`data-studio-preview-route="/?gosx-preview=1"`,
		`data-studio-preview-route="/about?gosx-preview=1"`,
		`data-studio-viewport="mobile"`,
		`data-studio-preview-diagnostic="true"`,
		`aria-live="polite"`,
		`data-studio-preview-frame="true"`,
		`title="Noni &#34;Home&#34; preview"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("page canvas missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "<form") {
		t.Fatalf("page canvas must never nest an authoring form:\n%s", html)
	}
}

func TestRenderWorkbenchPageCanvasSectionOrderLaneUsesDurableFieldContract(t *testing.T) {
	html := gosx.RenderHTML(RenderWorkbenchPageCanvas(WorkbenchPageCanvasOptions{
		URL:          "/?gosx-preview=1",
		AssetVersion: "release 1",
		SectionOrder: &WorkbenchPageCanvasSectionOrderOptions{
			Enabled:          true,
			Action:           "/admin/editor/__actions/operation",
			Route:            "/",
			SourceRoute:      "/?gosx-preview=1",
			PageID:           "home",
			Field:            "home.sections.order",
			ComponentKey:     "",
			DocumentRevision: "12",
			TargetHead:       "head-order",
			UndoOperationID:  "op-undo",
			Items: []WorkbenchPageCanvasSectionOrderItem{
				{Key: "hero", Label: "Hero"},
				{Key: "gallery", Label: "Gallery"},
				{Key: "contact", Label: "Contact"},
			},
		},
	}))
	for _, fragment := range []string{
		`data-gosx-studio-section-order="true"`,
		`data-gosx-studio-section-order-enabled="true"`,
		`data-gosx-studio-section-order-route="/"`,
		`data-gosx-studio-section-order-source-route="/"`,
		`data-gosx-studio-section-order-item="hero"`,
		`data-gosx-studio-section-order-handle="true"`,
		`data-gosx-studio-section-order-move="up"`,
		`data-gosx-studio-section-order-history="undo"`,
		`data-gosx-studio-section-order-form="true"`,
		`data-studio-target-field="home.sections.order"`,
		`data-studio-target-component=""`,
		`data-studio-target-head="head-order"`,
		`id="studio-section-order-guidance"`,
		`aria-describedby="studio-section-order-guidance"`,
		`<script src="/_gosx/studio/section-order-runtime.js?v=release+1" defer="defer" data-gosx-studio-section-order-runtime="true"></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("page canvas section order missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<form`) {
		t.Fatalf("section order lane must not render a nested form; runtime creates a detached durable form at commit time:\n%s", html)
	}
	if !strings.Contains(html, `data-gosx-studio-history-operation-id="op-undo"`) {
		t.Fatalf("page canvas section order must expose undo id:\n%s", html)
	}
}

func TestRenderWorkbenchPageCanvasSectionOrderAssetVersionPreservesReleaseURLContract(t *testing.T) {
	render := func(version string) string {
		return gosx.RenderHTML(RenderWorkbenchPageCanvas(WorkbenchPageCanvasOptions{
			URL:          "/?gosx-preview=1",
			AssetVersion: version,
			SectionOrder: &WorkbenchPageCanvasSectionOrderOptions{
				Enabled: true,
				Items:   []WorkbenchPageCanvasSectionOrderItem{{Key: "hero"}},
			},
		}))
	}

	withoutVersion := render("")
	if want := `<script src="/_gosx/studio/section-order-runtime.js"`; !strings.Contains(withoutVersion, want) {
		t.Fatalf("empty asset version must preserve unversioned section-order URL %q:\n%s", want, withoutVersion)
	}
	if strings.Contains(withoutVersion, `section-order-runtime.js?v=`) {
		t.Fatalf("empty asset version must not add a query version:\n%s", withoutVersion)
	}

	escaped := render(`release 1&beta`)
	if want := `<script src="/_gosx/studio/section-order-runtime.js?v=release+1%26beta"`; !strings.Contains(escaped, want) {
		t.Fatalf("asset version must be URL-escaped in section-order URL %q:\n%s", want, escaped)
	}

	newRelease := render("release-2")
	if escaped == newRelease || !strings.Contains(newRelease, `section-order-runtime.js?v=release-2`) {
		t.Fatalf("changing asset version must change the rendered section-order URL:\nold=%s\nnew=%s", escaped, newRelease)
	}
}

func TestRenderWorkbenchPageCanvasSectionOrderDoesNotNestWorkbenchForm(t *testing.T) {
	pageCanvas := RenderWorkbenchPageCanvas(WorkbenchPageCanvasOptions{
		URL: "/?gosx-preview=1",
		SectionOrder: &WorkbenchPageCanvasSectionOrderOptions{
			Enabled: true,
			Action:  "/admin/editor/__actions/operation",
			Items:   []WorkbenchPageCanvasSectionOrderItem{{Key: "hero"}, {Key: "gallery"}},
		},
	})
	html := gosx.RenderHTML(RenderWorkbenchFrame(nil, WorkbenchFrameOptions{Board: []gosx.Node{pageCanvas}}))
	if got := strings.Count(html, "<form"); got != 1 {
		t.Fatalf("section order must use a detached runtime form, got %d forms:\n%s", got, html)
	}
	if strings.Contains(html, `<form data-gosx-studio-section-order-form`) {
		t.Fatalf("section order adapter must remain a non-form node:\n%s", html)
	}
}

func TestRenderWorkbenchPageCanvasSectionOrderDisabledWhenUnsupported(t *testing.T) {
	html := gosx.RenderHTML(RenderWorkbenchPageCanvas(WorkbenchPageCanvasOptions{
		URL: "/about?gosx-preview=1",
		SectionOrder: &WorkbenchPageCanvasSectionOrderOptions{
			Enabled:        true,
			Route:          "/",
			SourceRoute:    "/",
			DisabledReason: "Section ordering applies only to Home.",
			Items: []WorkbenchPageCanvasSectionOrderItem{
				{Key: "hero", Label: "Hero"},
			},
		},
	}))
	for _, fragment := range []string{
		`data-gosx-studio-section-order-enabled="false"`,
		`data-gosx-studio-section-order-disabled-reason="Section ordering applies only to Home."`,
		`disabled="disabled"`,
		`Section ordering applies only to Home.`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("disabled section order missing %q:\n%s", fragment, html)
		}
	}
}
