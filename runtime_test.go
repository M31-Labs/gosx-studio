package studio

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEngineRuntimeOwnsStudioEngineFactories(t *testing.T) {
	engines := string(EngineRuntimeScript())
	for _, check := range []string{
		"studioEngineFactories",
		"GoSXStudioCanvas",
		"GoSXStudioSiteMap",
		"GoSXStudioFlowDesigner",
		"GoSXStudioBlockLayout",
		"GoSXStudioShowcase3D",
		"mountCanvasEngine",
		"mountBlockLayoutEngine",
		"GoSXStudioWorkbenchRuntime",
		"bindWorkbenchRailResizers",
		"bindWorkbenchChrome",
		"gosxStudioWorkbenchChromeBound",
		"button[data-studio-zoom]",
		"button[data-studio-style-state]",
		"emitWorkbenchChange(\"mode-change\"",
		"emitWorkbenchChange(\"viewport-change\"",
		"emitWorkbenchChange(\"style-state-change\"",
		"gosxstudio:rail-width-change",
		"gosxstudio:rail-width-commit",
		"GoSXStudioSelectionRuntime",
		"bindSelectionSurface",
		"gosxStudioSelectionBound",
		"gosxStudioSelectionCommandsBound",
		"data-studio-selection-action",
		"studio:field-select",
		"GoSXStudioFieldRuntime",
		"bindFieldRuntime",
		"bindFieldMirroring",
		"bindFieldClipboard",
		"data-editor-source",
		"data-studio-copy-target",
		"gosxStudioFieldMirrorBound",
		"gosxStudioFieldClipboardBound",
		"GoSXStudioBrandRuntime",
		"bindBrandLogo",
		"data-editor-logo-url",
		"gosxStudioBrandLogoBound",
		"GoSXStudioStyleRuntime",
		"bindTheme",
		"bindStyleWorkbench",
		"bindCSS",
		"bindFonts",
		"applyTheme: updateTheme",
		"syncStyleControlButtons",
		"setStyleControlValue",
		"data-studio-style-impact-panel",
		"gosxStudioThemeBound",
		"gosxStudioFontBound",
		"bindBlockLayoutLibrary",
		"bindBlockLayoutVisibility",
		"GoSXStudioBlockLayoutRuntime",
		"updateBlockLibraryState",
		"updateVisibilityState",
		"setPreviewBlockVisibility",
		"commitBlockLayoutReorder",
		"engine-preview",
		"GoSXStudioPreviewRuntime",
		"gosxstudio:engine-mounted",
	} {
		if !strings.Contains(engines, check) {
			t.Fatalf("engine runtime missing %q: %s", check, engines)
		}
	}

	legacyRuntimeConstName := "Block" + "StudioRuntimePath"
	legacyBlockRuntimeName := "block-" + "studio.js"
	if strings.Contains(engines, legacyRuntimeConstName) || strings.Contains(engines, legacyBlockRuntimeName) {
		t.Fatalf("engine runtime should not expose legacy block-studio script path")
	}
}

func TestPreviewRuntimeIsEngineMounted(t *testing.T) {
	preview := string(PreviewRuntimeScript())

	for _, check := range []string{"bindHandleDrag", "closestRowByY", "dataset.blockStudioBound", "data-block-studio-order", "updateMoveButtons"} {
		if strings.Contains(preview, check) {
			t.Fatalf("preview runtime should not own legacy list drag binder fragment %q", check)
		}
	}
	for _, check := range []string{"list.querySelectorAll(\"[data-block-studio-block]\")", "root.querySelector('[data-block-studio-block=\"", "row ? row.getAttribute(\"data-block-studio-block\")"} {
		if strings.Contains(preview, check) {
			t.Fatalf("preview runtime should depend on GoSXStudioBlockLayoutRuntime instead of fallback fragment %q", check)
		}
	}
	for _, check := range []string{"GoSXStudioPreviewRuntime", "mount: init", "setBlockVisibility", "applyTextUpdate", "applyTheme", "applyStyleImpact", "applyCSS", "applyFonts", "updateHeaderLogo", "requestInlineEdit", "cycleField"} {
		if !strings.Contains(preview, check) {
			t.Fatalf("preview runtime missing export %q: %s", check, preview)
		}
	}
	for _, check := range []string{"studio:inline-edit-request", "studio:field-cycle"} {
		if strings.Contains(preview, check) {
			t.Fatalf("preview runtime should expose API methods instead of listening for bridge event %q", check)
		}
	}
	for _, check := range []string{"DOMContentLoaded", "gosx:navigate", "gosx:render"} {
		if strings.Contains(preview, check) {
			t.Fatalf("preview runtime should be engine-mounted, found self-bootstrap fragment %q", check)
		}
	}
}

func TestStylesheetHandlerServesStudioChromeTokens(t *testing.T) {
	rec := httptest.NewRecorder()
	StylesheetHandler().ServeHTTP(rec, httptest.NewRequest("GET", StylesheetPath, nil))

	if rec.Code != 200 {
		t.Fatalf("stylesheet status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("stylesheet content type = %q", ct)
	}
	for _, check := range []string{"--size-studio-max", "--studio-stage-height", ".gosx-studio"} {
		if !strings.Contains(rec.Body.String(), check) {
			t.Fatalf("stylesheet missing %q: %s", check, rec.Body.String())
		}
	}
}

func TestEngineRuntimeHandlerServesCombinedRuntime(t *testing.T) {
	rec := httptest.NewRecorder()
	EngineRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", EngineRuntimePath, nil))

	if rec.Code != 200 {
		t.Fatalf("engine runtime status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Fatalf("engine runtime content type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "GoSXStudioPreviewRuntime") || !strings.Contains(body, "GoSXStudioEngineRuntime") {
		t.Fatalf("engine runtime should include preview and engine globals: %s", body)
	}
}

func TestAssetHrefAddsEscapedVersion(t *testing.T) {
	if got := AssetHref(EngineRuntimePath, "build 1"); got != EngineRuntimePath+"?v=build+1" {
		t.Fatalf("asset href = %q", got)
	}
	if got := AssetHref(EngineRuntimePath+"?debug=true", "build 1"); got != EngineRuntimePath+"?debug=true&v=build+1" {
		t.Fatalf("asset href with query = %q", got)
	}
	if got := AssetHref(EngineRuntimePath, " "); got != EngineRuntimePath {
		t.Fatalf("empty asset href = %q", got)
	}
}
