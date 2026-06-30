package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendEditorPagePreservesEditorShellContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendEditorPage(BackendEditorPageProps{
		Media: []BackendEditorMediaAsset{{
			URL:      "/media/hero.jpg",
			Filename: "hero.jpg",
			Alt:      "Hero alt",
		}},
		SaveStatus: BackendEditorActionStatus{
			OK:      true,
			Message: "Saved.",
		},
		AuthoringStatus: BackendEditorActionStatus{
			Submitted: true,
			Message:   "Authoring failed.",
		},
		PublishStatus: BackendEditorActionStatus{
			OK:      true,
			Message: "Published.",
		},
		PublishFlowStatus: BackendEditorActionStatus{
			Submitted: true,
			Message:   "Flow failed.",
		},
		RestoreStatus: BackendEditorActionStatus{
			Submitted: true,
			Message:   "Restore failed.",
		},
		RevisionRestored: true,
		WorkbenchShell: gosx.El("section", gosx.Attrs(gosx.Attr("data-workbench", "true")),
			gosx.Text("workbench"),
		),
		SupportNodes: []gosx.Node{
			gosx.El("form", gosx.Attrs(gosx.Attr("data-site-map-authoring", "true"))),
			gosx.El("form", gosx.Attrs(gosx.Attr("data-style-palette", "true"))),
			gosx.El("div", gosx.Attrs(gosx.Attr("data-engine-hosts", "true"))),
		},
		Scripts: BackendEditorScripts{
			WorkbenchRuntime: "/_gosx/studio/workbench-runtime.js",
			CommandRuntime:   "/_gosx/studio/command-palette.js",
			StateRuntime:     "/_gosx/studio/state-runtime.js",
			EngineRuntime:    "/_gosx/studio/studio-engines.js",
		},
	}))

	for _, fragment := range []string{
		`<div class="admin-page editor-page" data-gosx-studio-backend-editor-renderer="gosx-studio">`,
		`<datalist id="editor-media-urls"><option value="/media/hero.jpg" label="hero.jpg" data-media-alt="Hero alt">hero.jpg</option></datalist>`,
		`<p class="form-status form-status--ok">Saved.</p>`,
		`<p class="form-status form-status--error">Authoring failed.</p>`,
		`<p class="form-status form-status--ok">Published.</p>`,
		`<p class="form-status form-status--error">Flow failed.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<p class="form-status form-status--ok">Editor settings restored.</p>`,
		`<section data-workbench="true">workbench</section>`,
		`<form data-site-map-authoring="true"></form>`,
		`<form data-style-palette="true"></form>`,
		`<div data-engine-hosts="true"></div>`,
		`<script src="/_gosx/studio/workbench-runtime.js" defer data-gosx-studio-workbench-runtime="true"></script>`,
		`<script src="/_gosx/studio/command-palette.js" defer data-gosx-studio-command-runtime="true"></script>`,
		`<script src="/_gosx/studio/state-runtime.js" defer data-gosx-studio-state-runtime="true"></script>`,
		`<script src="/_gosx/studio/studio-engines.js" defer data-gosx-studio-engine-runtime="true"></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("editor page renderer missing %q:\n%s", fragment, html)
		}
	}

	assertOrder(t, html,
		`<datalist id="editor-media-urls">`,
		`<p class="form-status form-status--ok">Saved.</p>`,
		`<section data-workbench="true">workbench</section>`,
		`<form data-site-map-authoring="true"></form>`,
		`<form data-style-palette="true"></form>`,
		`<div data-engine-hosts="true"></div>`,
		`data-gosx-studio-workbench-runtime="true"`,
		`data-gosx-studio-command-runtime="true"`,
		`data-gosx-studio-state-runtime="true"`,
		`data-gosx-studio-engine-runtime="true"`,
	)
}

func TestRenderBackendEditorPageDefaultsAndSplitNodes(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendEditorPage(BackendEditorPageProps{}))
	if !strings.Contains(html, `<div class="admin-page editor-page" data-gosx-studio-backend-editor-renderer="gosx-studio">`) {
		t.Fatalf("editor page default wrapper missing:\n%s", html)
	}
	if !strings.Contains(html, `<datalist id="editor-media-urls"></datalist>`) {
		t.Fatalf("editor page default datalist missing:\n%s", html)
	}
	if strings.Contains(html, `form-status`) {
		t.Fatalf("editor page default statuses should be empty:\n%s", html)
	}

	statuses := gosx.RenderHTML(RenderBackendEditorStatuses(BackendEditorPageProps{}))
	if statuses != "" {
		t.Fatalf("empty editor statuses should render empty, got %q", statuses)
	}

	scripts := gosx.RenderHTML(RenderBackendEditorRuntimeScripts(BackendEditorScripts{}))
	for _, fragment := range []string{
		`data-gosx-studio-workbench-runtime="true"`,
		`data-gosx-studio-command-runtime="true"`,
		`data-gosx-studio-state-runtime="true"`,
		`data-gosx-studio-engine-runtime="true"`,
	} {
		if !strings.Contains(scripts, fragment) {
			t.Fatalf("empty script renderer missing %q:\n%s", fragment, scripts)
		}
	}
}

func TestRenderBackendEditorPageRendersEngineHostsFromProps(t *testing.T) {
	runtime := &recordingStudioEngineRuntime{}
	html := gosx.RenderHTML(RenderBackendEditorPage(BackendEditorPageProps{
		WorkbenchShell: gosx.El("section", gosx.Attrs(gosx.Attr("data-workbench", "true"))),
		SupportNodes: []gosx.Node{
			gosx.El("form", gosx.Attrs(gosx.Attr("data-site-map-authoring", "true"))),
			gosx.El("form", gosx.Attrs(gosx.Attr("data-style-palette", "true"))),
		},
		EngineHosts: []map[string]any{{
			"key":          "flow-designer",
			"name":         FlowDesignerName,
			"mountId":      "gosx-studio-flow-engine",
			"class":        "studio-flow-engine-host",
			"capabilities": []string{"canvas", "pointer"},
		}},
		EngineRuntime: runtime,
	}))

	if len(runtime.calls) != 1 {
		t.Fatalf("runtime call count = %d, want 1", len(runtime.calls))
	}
	for _, fragment := range []string{
		`<form data-site-map-authoring="true"></form>`,
		`<form data-style-palette="true"></form>`,
		`<div class="studio-engine-hosts" aria-hidden="true" data-gosx-studio-engines="true" data-gosx-studio-engine-hosts-renderer="gosx-studio">`,
		`id="gosx-studio-flow-engine"`,
		`data-gosx-engine="GoSXStudioFlowDesigner"`,
		`data-gosx-engine-kind="surface"`,
		`data-gosx-studio-engine="flow-designer"`,
		`data-gosx-engine-capabilities="canvas pointer"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("editor page engine host render missing %q:\n%s", fragment, html)
		}
	}
	assertOrder(t, html,
		`<section data-workbench="true"></section>`,
		`<form data-site-map-authoring="true"></form>`,
		`<form data-style-palette="true"></form>`,
		`<div class="studio-engine-hosts"`,
		`data-gosx-studio-workbench-runtime="true"`,
	)
}

func TestRenderBackendEditorPageRendersStylePanelFromProps(t *testing.T) {
	runtime := &recordingStudioEngineRuntime{}
	html := gosx.RenderHTML(RenderBackendEditorPage(BackendEditorPageProps{
		WorkbenchShell: gosx.El("section", gosx.Attrs(gosx.Attr("data-workbench", "true"))),
		SupportNodes: []gosx.Node{
			gosx.El("form", gosx.Attrs(gosx.Attr("data-site-map-authoring", "true"))),
		},
		StylePanelView: map[string]any{
			"palette": []map[string]any{{
				"key":      "accent",
				"name":     "colorAccent",
				"label":    "Accent",
				"cssVar":   "--color-accent",
				"value":    "#b5651d",
				"fallback": "#b5651d",
			}},
			"fonts": []map[string]any{{
				"role":      "display",
				"label":     "Display",
				"nameField": "displayFont",
				"urlField":  "displayFontUrl",
				"family":    "Fraunces",
				"url":       "https://fonts.example.com/fraunces.woff2",
			}},
		},
		StylePanelFormID:    "editorStylePaletteForm",
		StylePanelAction:    "/admin/editor/__actions/authoring",
		StylePanelCSRFToken: "csrf-token",
		EngineHosts: []map[string]any{{
			"key":     "flow-designer",
			"name":    FlowDesignerName,
			"mountId": "gosx-studio-flow-engine",
		}},
		EngineRuntime: runtime,
	}))

	if len(runtime.calls) != 1 {
		t.Fatalf("runtime call count = %d, want 1", len(runtime.calls))
	}
	for _, fragment := range []string{
		`<form data-site-map-authoring="true"></form>`,
		`data-gosx-studio-style-panel="true"`,
		`id="editorStylePaletteForm"`,
		`action="/admin/editor/__actions/authoring"`,
		`name="csrf_token" value="csrf-token"`,
		`data-editor-color-token="--color-accent"`,
		`data-editor-font-name="display"`,
		`data-gosx-studio-engine-hosts-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("editor page style panel render missing %q:\n%s", fragment, html)
		}
	}
	assertOrder(t, html,
		`<section data-workbench="true"></section>`,
		`<form data-site-map-authoring="true"></form>`,
		`data-gosx-studio-style-panel="true"`,
		`data-gosx-studio-engine-hosts-renderer="gosx-studio"`,
		`data-gosx-studio-workbench-runtime="true"`,
	)
}

func TestRenderBackendEditorPageSkipsEmptyStylePanelProps(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendEditorPage(BackendEditorPageProps{
		StylePanelView: map[string]any{
			"palette": []map[string]any{{
				"key":      "accent",
				"name":     "colorAccent",
				"label":    "Accent",
				"cssVar":   "--color-accent",
				"value":    "#b5651d",
				"fallback": "#b5651d",
			}},
		},
	}))

	if strings.Contains(html, `data-gosx-studio-style-panel="true"`) {
		t.Fatalf("style panel should not render without an action:\n%s", html)
	}
}

func TestRenderBackendEditorPageStylePanelNodeOverridesView(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendEditorPage(BackendEditorPageProps{
		SupportNodes: []gosx.Node{
			gosx.El("form", gosx.Attrs(gosx.Attr("data-site-map-authoring", "true"))),
		},
		StylePanelView: map[string]any{
			"palette": []map[string]any{{
				"key":      "accent",
				"name":     "colorAccent",
				"label":    "Accent",
				"cssVar":   "--color-accent",
				"value":    "#b5651d",
				"fallback": "#b5651d",
			}},
		},
		StylePanelAction: "/admin/editor/__actions/authoring",
		StylePanelNode:   gosx.El("form", gosx.Attrs(gosx.Attr("data-explicit-style-panel", "true"))),
	}))

	if !strings.Contains(html, `<form data-explicit-style-panel="true"></form>`) {
		t.Fatalf("explicit style panel node missing:\n%s", html)
	}
	if strings.Contains(html, `data-gosx-studio-style-panel="true"`) {
		t.Fatalf("style panel view should not render when explicit node is supplied:\n%s", html)
	}
	assertOrder(t, html,
		`<form data-site-map-authoring="true"></form>`,
		`<form data-explicit-style-panel="true"></form>`,
	)
}

func TestRenderBackendEditorPageEngineHostsNodeOverridesHostViews(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendEditorPage(BackendEditorPageProps{
		SupportNodes: []gosx.Node{
			gosx.El("form", gosx.Attrs(gosx.Attr("data-style-palette", "true"))),
		},
		EngineHosts: []map[string]any{{
			"key":     "flow-designer",
			"name":    FlowDesignerName,
			"mountId": "gosx-studio-flow-engine",
		}},
		EngineHostsNode: gosx.El("div", gosx.Attrs(gosx.Attr("data-explicit-engine-hosts", "true"))),
	}))

	if !strings.Contains(html, `<div data-explicit-engine-hosts="true"></div>`) {
		t.Fatalf("explicit engine hosts node missing:\n%s", html)
	}
	if strings.Contains(html, `data-gosx-studio-engine-hosts-renderer="gosx-studio"`) {
		t.Fatalf("engine host views should not render when explicit node is supplied:\n%s", html)
	}
	assertOrder(t, html,
		`<form data-style-palette="true"></form>`,
		`<div data-explicit-engine-hosts="true"></div>`,
	)
}

func assertOrder(t *testing.T, html string, fragments ...string) {
	t.Helper()
	last := -1
	for _, fragment := range fragments {
		next := strings.Index(html, fragment)
		if next < 0 {
			t.Fatalf("missing ordered fragment %q:\n%s", fragment, html)
		}
		if next <= last {
			t.Fatalf("fragment %q rendered out of order:\n%s", fragment, html)
		}
		last = next
	}
}
