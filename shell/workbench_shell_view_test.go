package shell

import "testing"

func TestWorkbenchShellViewProjectsReusableEditorChrome(t *testing.T) {
	shell := New(Options{
		Title:      "Website editor",
		PreviewURL: "/",
		SaveAction: "/admin/editor/__actions/save",
		Canvas: CanvasSurface{
			RouteLabel:     "Home",
			SelectionLabel: "Hero selected",
			Zoom:           "fit",
		},
		Modes: []Mode{NewMode("home", "Home", true)},
		Viewports: []Viewport{
			NewViewport("desktop", "Desktop", "100%", false),
			NewViewport("tablet", "Tablet", "48rem", true),
		},
	})

	view := WorkbenchShellViewForShell(shell, WorkbenchShellViewOptions{
		FormID:            "editor-workbench-form",
		CSRFToken:         "csrf-token",
		StyleSystemID:     "theme-clay",
		StyleSystemValid:  true,
		ToolbarTitle:      "Client website",
		PreviewURL:        "/?gosx-preview=1",
		PreviewHref:       "/admin/storefront",
		PreviewLabel:      "Preview storefront",
		PreviewTitle:      "Storefront preview",
		PreviewFrameTitle: "Storefront preview",
		Commands: []map[string]any{
			{"key": "save", "label": "Save changes"},
		},
		FeatureFlagAttrs: map[string]any{"data-gosx-studio-feature-flag-field-runtime-islands": "true"},
	})

	checks := map[string]any{
		"class":             "gosx-studio-workbench",
		"formID":            "editor-workbench-form",
		"method":            "post",
		"action":            "/admin/editor/__actions/save",
		"hasAction":         true,
		"csrfToken":         "csrf-token",
		"hasCSRF":           true,
		"shellKey":          "website-editor",
		"autosaveURL":       "/admin/editor/__actions/save",
		"autosaveDelay":     "1400",
		"mode":              "home",
		"styleSystemID":     "theme-clay",
		"styleSystemValid":  "true",
		"toolbarTitle":      "Client website",
		"previewHref":       "/admin/storefront",
		"previewLabel":      "Preview storefront",
		"routeLabel":        "Home",
		"selectionLabel":    "Hero selected",
		"zoom":              "fit",
		"viewportKey":       "tablet",
		"viewportLabel":     "Tablet",
		"previewURL":        "/?gosx-preview=1",
		"previewTitle":      "Storefront preview",
		"previewFrameTitle": "Storefront preview",
	}
	for key, want := range checks {
		if view[key] != want {
			t.Fatalf("%s = %#v, want %#v", key, view[key], want)
		}
	}
	commands := view["commands"].([]map[string]any)
	if len(commands) != 1 || commands[0]["key"] != "save" || !view["hasCommands"].(bool) {
		t.Fatalf("expected command projection: %#v", view)
	}
	previewShell := view["previewShell"].(map[string]any)
	frameAttrs := previewShell["frameAttrs"].(map[string]any)
	if previewShell["inlineEditClass"] != "is-studio-inline-editing" || frameAttrs["data-studio-preview-src"] != "/?gosx-preview=1" {
		t.Fatalf("expected preview shell attrs: %#v", previewShell)
	}
	zoomLevels := view["zoomLevels"].([]map[string]any)
	if len(zoomLevels) != 4 || zoomLevels[0]["key"] != "fit" || zoomLevels[0]["pressed"] != true {
		t.Fatalf("expected default zoom levels: %#v", zoomLevels)
	}
	viewports := view["viewports"].([]map[string]any)
	if len(viewports) != 2 || viewports[1]["key"] != "tablet" || viewports[1]["width"] != "48rem" || viewports[1]["pressed"] != "true" {
		t.Fatalf("expected shell viewport projection: %#v", viewports)
	}
	left := view["leftResizer"].(map[string]any)
	right := view["rightResizer"].(map[string]any)
	if left["default"] != "320" || right["default"] != "416" {
		t.Fatalf("expected default resizers: %#v %#v", left, right)
	}
	flags := view["featureFlagAttrs"].(map[string]any)
	if flags["data-gosx-studio-feature-flag-field-runtime-islands"] != "true" {
		t.Fatalf("expected feature flags to project: %#v", flags)
	}
}

func TestWorkbenchShellViewAcceptsHostOverrides(t *testing.T) {
	view := WorkbenchShellView(WorkbenchShellSource{
		Title:         "Custom site",
		PreviewURL:    "/preview",
		BlockCount:    2,
		MediaCount:    3,
		RevisionCount: 4,
	}, WorkbenchShellViewOptions{
		Action:            "/save",
		Method:            "POST",
		AutosaveDelay:     2200,
		Mode:              "publish",
		CanvasPanelKey:    "custom-canvas",
		CommandEmptyTitle: "Nothing found",
		ZoomLevels: []WorkbenchZoomLevel{
			{Key: "fit", Label: "Fit"},
			{Key: "200", Label: "200%", Active: true},
		},
		LeftResizer: WorkbenchRailResizer{Min: 280, Max: 500, Default: 360},
		Extras:      map[string]any{"source": "host", "mode": "ignored"},
	})

	if view["blockCount"] != "2" || view["mediaCount"] != "3" || view["revisionCount"] != "4" {
		t.Fatalf("expected shell counts as attribute-safe strings: %#v", view)
	}
	if view["action"] != "/save" || view["method"] != "POST" || view["autosaveDelay"] != "2200" || view["mode"] != "publish" {
		t.Fatalf("expected host overrides: %#v", view)
	}
	if view["previewURL"] != "/preview" || view["previewHref"] != "/preview" || view["canvasPanelKey"] != "custom-canvas" || view["commandEmptyTitle"] != "Nothing found" {
		t.Fatalf("expected host labels: %#v", view)
	}
	zoomLevels := view["zoomLevels"].([]map[string]any)
	if len(zoomLevels) != 2 || zoomLevels[1]["key"] != "200" || zoomLevels[1]["pressed"] != true {
		t.Fatalf("expected custom zoom levels: %#v", zoomLevels)
	}
	left := view["leftResizer"].(map[string]any)
	if left["min"] != "280" || left["max"] != "500" || left["default"] != "360" {
		t.Fatalf("expected custom left resizer: %#v", left)
	}
	if view["source"] != "host" || view["mode"] != "publish" {
		t.Fatalf("expected extras without replacing core keys: %#v", view)
	}
}
