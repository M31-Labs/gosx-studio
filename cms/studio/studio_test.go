package studio

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/lifecycle"
	"m31labs.dev/gosx-studio/cms/media"
)

func TestNewNormalizesShell(t *testing.T) {
	shell := New(Options{
		SaveAction:   "/admin/editor?action=save",
		BlockCatalog: []blockstudio.Definition{{Key: "hero", Label: "Hero"}},
		Media:        []media.Asset{{URL: "/media/forest.jpg"}},
		Revisions:    []lifecycle.Revision{{ID: "rev_1"}},
		Left:         []Panel{{Key: "Site Map", Label: "Site map"}},
		Right:        []Panel{{Key: "Inspector", Label: "Inspector"}},
	})
	if shell.Title != "GoSX Studio" || shell.PreviewURL != "/" || shell.BlockCount != 1 || shell.MediaCount != 1 || shell.RevisionCount != 1 {
		t.Fatalf("unexpected shell: %#v", shell)
	}
	if len(shell.Actions) != 1 || shell.Actions[0].Key != "save" || !shell.Actions[0].Primary {
		t.Fatalf("expected default save action, got %#v", shell.Actions)
	}
	if shell.Left[0].Key != "site-map" || shell.Right[0].Key != "inspector" {
		t.Fatalf("expected normalized panels, got %#v %#v", shell.Left, shell.Right)
	}
}

func TestNewDefaults(t *testing.T) {
	shell := New(Options{})
	if shell.Title != "GoSX Studio" || shell.PreviewURL != "/" {
		t.Fatalf("unexpected defaults: %#v", shell)
	}
	if shell.SaveAction != "" || shell.RestoreAction != "" {
		t.Fatalf("expected empty actions URLs by default: %#v", shell)
	}
	if shell.BlockCount != 0 || shell.MediaCount != 0 || shell.RevisionCount != 0 || shell.HasMedia || shell.HasRevisions {
		t.Fatalf("unexpected empty counts: %#v", shell)
	}
	if len(shell.Modes) != 4 || shell.Modes[0].Key != "structure" || !shell.Modes[0].Active {
		t.Fatalf("unexpected default modes: %#v", shell.Modes)
	}
	if len(shell.Viewports) != 3 || shell.Viewports[2].Key != "mobile" || shell.Viewports[2].Width != "24rem" {
		t.Fatalf("unexpected default viewports: %#v", shell.Viewports)
	}
	if shell.Canvas.RouteLabel != "GoSX Studio" || shell.Canvas.SelectionLabel != "No selection" || shell.Canvas.Zoom != "fit" {
		t.Fatalf("unexpected default canvas: %#v", shell.Canvas)
	}
}

func TestActionNormalization(t *testing.T) {
	shell := New(Options{
		SaveAction: "/save",
		Actions: []Action{
			{Key: "Preview Page", Label: " Preview ", Href: " /preview "},
			{Key: "save", Label: "Custom save", Href: "/custom-save"},
			{Key: "missing-label", Href: "/skip"},
		},
		Navigation: []Section{{
			Key:     "Main Nav",
			Label:   " Main ",
			Summary: " Primary links ",
			Actions: []Action{
				{Key: "Media Library", Label: " Media ", Href: " /media "},
				{Key: "invalid"},
			},
		}},
	})
	if len(shell.Actions) != 2 {
		t.Fatalf("expected custom save to prevent default save insertion: %#v", shell.Actions)
	}
	if shell.Actions[0].Key != "preview-page" || shell.Actions[0].Label != "Preview" || shell.Actions[0].Href != "/preview" {
		t.Fatalf("unexpected normalized preview action: %#v", shell.Actions[0])
	}
	if shell.Actions[1].Key != "save" || shell.Actions[1].Label != "Custom save" || shell.Actions[1].Href != "/custom-save" {
		t.Fatalf("unexpected normalized save action: %#v", shell.Actions[1])
	}
	if len(shell.Navigation) != 1 || shell.Navigation[0].Key != "main-nav" || shell.Navigation[0].Summary != "Primary links" {
		t.Fatalf("unexpected navigation section: %#v", shell.Navigation)
	}
	if len(shell.Navigation[0].Actions) != 1 || shell.Navigation[0].Actions[0].Key != "media-library" {
		t.Fatalf("unexpected navigation actions: %#v", shell.Navigation[0].Actions)
	}
}

func TestPanelAndCatalogSummaries(t *testing.T) {
	archivedAt := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	created := time.Date(2026, 5, 2, 10, 30, 0, 0, time.UTC)
	shell := New(Options{
		BlockCatalog: []blockstudio.Definition{{
			Key:        "hero",
			Label:      "Hero",
			Summary:    "Lead story",
			Kind:       "section",
			Preview:    "/blocks/hero",
			DefaultOn:  true,
			Locked:     true,
			Repeatable: true,
			Icon:       "image",
			Fields:     []blockstudio.FieldDefinition{{Name: "title"}},
		}},
		Media: []media.Asset{{
			ID:          "asset_1",
			URL:         "/media/forest.jpg",
			Alt:         "Forest",
			Filename:    "forest.jpg",
			ContentType: "image/jpeg",
			Size:        2048,
			ArchivedAt:  &archivedAt,
		}},
		Revisions: []lifecycle.Revision{{
			ID:            "rev_1",
			ResourceKind:  "page",
			ResourceID:    "home",
			ResourceTitle: "Home",
			Action:        "publish",
			Summary:       "Published home",
			Created:       created,
		}},
		Left:  []Panel{NewPanel("Content Tree", "Content tree", "3 pages")},
		Right: []Panel{{Key: " Inspector ", Label: " Inspector ", Summary: " Fields "}},
	})
	if len(shell.Blocks) != 1 || shell.Blocks[0].FieldCount != 1 || !shell.Blocks[0].DefaultOn || !shell.Blocks[0].Locked || !shell.Blocks[0].Repeatable {
		t.Fatalf("unexpected block summary: %#v", shell.Blocks)
	}
	if len(shell.MediaLibrary) != 1 || !shell.MediaLibrary[0].Archived || shell.MediaLibrary[0].Filename != "forest.jpg" {
		t.Fatalf("unexpected media summary: %#v", shell.MediaLibrary)
	}
	if len(shell.RevisionLog) != 1 || shell.RevisionLog[0].Created != "2026-05-02T10:30:00Z" {
		t.Fatalf("unexpected revision summary: %#v", shell.RevisionLog)
	}
	if shell.Left[0].Key != "content-tree" || shell.Left[0].Summary != "3 pages" || shell.Right[0].Key != "inspector" || shell.Right[0].Summary != "Fields" {
		t.Fatalf("unexpected panel summaries: %#v %#v", shell.Left, shell.Right)
	}
}

func TestRevisionSummariesIncludeDiffs(t *testing.T) {
	created := time.Date(2026, 5, 2, 10, 30, 0, 0, time.UTC)
	shell := New(Options{
		Revisions: []lifecycle.Revision{
			{
				ID:           "new",
				ResourceKind: "page",
				ResourceID:   "home",
				Action:       "publish",
				Snapshot:     json.RawMessage(`{"title":"Home","hero":{"headline":"Welcome families"}}`),
				Created:      created.Add(time.Hour),
			},
			{
				ID:           "old",
				ResourceKind: "page",
				ResourceID:   "home",
				Action:       "preview",
				Snapshot:     json.RawMessage(`{"title":"Home","hero":{"headline":"Welcome"}}`),
				Created:      created,
			},
		},
	})
	if len(shell.RevisionLog) != 2 || !shell.RevisionLog[0].HasDiff || shell.RevisionLog[0].ChangeCount != 1 {
		t.Fatalf("expected newest revision to include diff summary: %#v", shell.RevisionLog)
	}
	if shell.RevisionLog[0].ChangeSummary != "1 changed field." || shell.RevisionLog[1].HasDiff {
		t.Fatalf("unexpected revision diff summaries: %#v", shell.RevisionLog)
	}
	view := View(shell)
	revisions := view["revisions"].([]map[string]any)
	if revisions[0]["changeSummary"] != "1 changed field." || revisions[0]["changeCount"] != 1 || revisions[0]["hasDiff"] != true {
		t.Fatalf("unexpected revision diff view: %#v", revisions)
	}
}

func TestMetricsAndExtras(t *testing.T) {
	extras := map[string]any{
		"workflow": "draft",
		"title":    "should not replace core title",
	}
	shell := New(Options{
		Title:   "Studio",
		Metrics: []Metric{NewMetric("Open Drafts", "Open drafts", 4), {Key: "skip"}},
		Extras:  extras,
	})
	extras["workflow"] = "published"
	if len(shell.Metrics) != 1 || shell.Metrics[0].Key != "open-drafts" || shell.Metrics[0].Value != 4 {
		t.Fatalf("unexpected metrics: %#v", shell.Metrics)
	}
	if shell.Extras["workflow"] != "draft" {
		t.Fatalf("expected extras to be cloned: %#v", shell.Extras)
	}
	view := View(shell)
	if view["title"] != "Studio" || view["workflow"] != "draft" {
		t.Fatalf("unexpected extras passthrough: %#v", view)
	}
	metrics := view["metrics"].([]map[string]any)
	if len(metrics) != 1 || metrics[0]["key"] != "open-drafts" || metrics[0]["value"] != 4 {
		t.Fatalf("unexpected metric views: %#v", metrics)
	}
	viewExtras := view["extras"].(map[string]any)
	viewExtras["workflow"] = "changed"
	if shell.Extras["workflow"] != "draft" {
		t.Fatalf("expected view extras to be cloned: %#v", shell.Extras)
	}
}

func TestShellModesViewportsAndCanvasView(t *testing.T) {
	shell := New(Options{
		Title: "Studio",
		Modes: []Mode{
			NewMode("Structure", "Structure", false),
			NewMode("Style", "Style", true),
			NewMode("Preview", "Preview", true),
		},
		Viewports: []Viewport{
			NewViewport("Desktop", "Desktop", "100%", false),
			NewViewport("Mobile", "Mobile", "24rem", true),
		},
		Canvas: CanvasSurface{RouteLabel: "Home", SelectionLabel: "Hero", Zoom: "100"},
	})
	if len(shell.Modes) != 3 || shell.Modes[1].Key != "style" || !shell.Modes[1].Active || shell.Modes[2].Active {
		t.Fatalf("unexpected normalized modes: %#v", shell.Modes)
	}
	if len(shell.Viewports) != 2 || !shell.Viewports[1].Active || shell.Viewports[1].Width != "24rem" {
		t.Fatalf("unexpected normalized viewports: %#v", shell.Viewports)
	}
	view := View(shell)
	modes := view["modes"].([]map[string]any)
	viewports := view["viewports"].([]map[string]any)
	canvas := view["canvas"].(map[string]any)
	if modes[1]["key"] != "style" || modes[1]["active"] != true || viewports[1]["key"] != "mobile" || canvas["selectionLabel"] != "Hero" {
		t.Fatalf("unexpected shell surface view: %#v %#v %#v", modes, viewports, canvas)
	}
	if modes[1]["pressed"] != "true" || modes[2]["pressed"] != "false" || viewports[1]["pressed"] != "true" {
		t.Fatalf("expected aria pressed helpers in shell view: %#v %#v", modes, viewports)
	}
}

func TestReadinessView(t *testing.T) {
	readiness := NewReadiness(
		NewReadinessItem("Studio shell", "Studio shell", ReadinessReady, "Mounted", "Canvas and rails are ready."),
		NewReadinessItem("media", "Media", ReadinessWatch, "Alt text", "One asset needs alt text.").WithHref("/admin/media"),
		NewReadinessItem("calendar", "Calendar", ReadinessNext, "Needed for Pajaritos", "Register calendar recipes."),
		ReadinessItem{Key: "skip"},
	)
	if readiness.Summary() != "1/3 ready" {
		t.Fatalf("unexpected summary: %s", readiness.Summary())
	}
	view := ReadinessView(readiness)
	if view["summary"] != "1/3 ready" || view["readyCount"] != 1 || view["watchCount"] != 1 || view["nextCount"] != 1 || view["total"] != 3 {
		t.Fatalf("unexpected readiness view: %#v", view)
	}
	items := view["items"].([]map[string]any)
	if len(items) != 3 {
		t.Fatalf("expected 3 readiness items, got %#v", items)
	}
	if items[0]["key"] != "studio-shell" || items[0]["statusLabel"] != "Ready" || items[0]["actionLabel"] != "Review" {
		t.Fatalf("unexpected ready item: %#v", items[0])
	}
	if items[1]["hasHref"] != true || items[1]["actionLabel"] != "Open" {
		t.Fatalf("unexpected watch item: %#v", items[1])
	}
	if items[2]["class"] != "studio-readiness-card studio-readiness-card--next" {
		t.Fatalf("unexpected next item class: %#v", items[2])
	}
}

func TestRenderShell(t *testing.T) {
	shell := New(Options{
		Title:      "Nature School Studio",
		PreviewURL: "/programs",
		Actions:    []Action{{Key: "preview", Label: "Preview", Href: "/programs"}},
		Left:       []Panel{{Key: "calendar", Label: "Calendar", Summary: "Upcoming sessions"}},
		Right:      []Panel{{Key: "inspector", Label: "Inspector", Children: []gosx.Node{gosx.El("p", nil, gosx.Text("Fields"))}}},
	})
	html := gosx.RenderHTML(Render(shell))
	for _, want := range []string{
		`class="gosx-studio"`,
		"Nature School Studio",
		`src="/programs"`,
		`data-studio-mode-control="structure"`,
		`data-studio-viewport="desktop"`,
		`data-studio-selection-label="true"`,
		`data-panel-key="calendar"`,
		`data-panel-key="inspector"`,
		`data-action-key="preview"`,
		"Fields",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in shell html: %s", want, html)
		}
	}
}

func TestView(t *testing.T) {
	shell := New(Options{
		Title:         "Studio",
		PreviewURL:    "/",
		SaveAction:    "/admin/editor?action=save",
		RestoreAction: "/admin/editor?action=restoreRevision",
		BlockCatalog:  []blockstudio.Definition{{Key: "hero"}},
		Media:         []media.Asset{{URL: "/media/forest.jpg"}},
		Revisions:     []lifecycle.Revision{{ID: "rev_1"}},
		Left:          []Panel{{Key: "map", Label: "Map", Summary: "Pages"}},
		Actions:       []Action{{Key: "preview", Label: "Preview", Href: "/", Primary: false}},
		Readiness:     NewReadiness(NewReadinessItem("shell", "Shell", ReadinessReady, "Mounted", "Ready.")),
	})
	view := View(shell)
	if view["title"] != "Studio" || view["saveAction"] != "/admin/editor?action=save" || view["blockCount"] != 1 || view["hasMedia"] != true || view["hasRevisions"] != true {
		t.Fatalf("unexpected view: %#v", view)
	}
	for _, key := range []string{"previewURL", "restoreAction", "mediaCount", "revisionCount", "actions", "leftPanels", "rightPanels", "modes", "viewports", "canvas"} {
		if _, ok := view[key]; !ok {
			t.Fatalf("expected compatibility key %q in view: %#v", key, view)
		}
	}
	actions := view["actions"].([]map[string]any)
	if len(actions) != 2 || actions[0]["key"] != "save" || actions[0]["class"] != "button button--primary" {
		t.Fatalf("unexpected action views: %#v", actions)
	}
	left := view["leftPanels"].([]map[string]any)
	if len(left) != 1 || left[0]["key"] != "map" || left[0]["hasSummary"] != true {
		t.Fatalf("unexpected panel views: %#v", left)
	}
	blocks := view["blockCatalog"].([]map[string]any)
	media := view["media"].([]map[string]any)
	revisions := view["revisions"].([]map[string]any)
	if len(blocks) != 1 || blocks[0]["key"] != "hero" || len(media) != 1 || media[0]["url"] != "/media/forest.jpg" || len(revisions) != 1 || revisions[0]["id"] != "rev_1" {
		t.Fatalf("unexpected catalog views: %#v %#v %#v", blocks, media, revisions)
	}
	readiness := view["readiness"].(map[string]any)
	if readiness["summary"] != "1/1 ready" {
		t.Fatalf("unexpected readiness view: %#v", readiness)
	}
}
