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

func TestEngineRuntimeIncludesFieldRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-1 burn-down of GoSXStudioFieldRuntime. The engine
	// runtime asset must include both the island-runtime JS (publishes the
	// __gosx_field_runtime_island_* globals) and the BridgeShim that
	// delegates window.GoSXStudioFieldRuntime calls when the
	// "field-runtime-islands" feature flag is on. See
	// gosx-studio/fieldruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_field_runtime_island_bind",
		"__gosx_field_runtime_island_bindMirroring",
		"__gosx_field_runtime_island_bindClipboard",
		"field-runtime-islands",
		// The shim's flag check must be in the bundle so the runtime is
		// actually gated, not just loaded.
		"data-gosx-studio-feature-flag-field-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing fieldruntime fragment %q", fragment)
		}
	}
	// The shim must override window.GoSXStudioFieldRuntime; without that
	// assignment the legacy methods are never wrapped.
	shimAssign := "window.GoSXStudioFieldRuntime = {"
	if !strings.Contains(engines, shimAssign) {
		// Allow for whitespace tolerance — the shim uses a multi-line
		// object literal. Check for the LHS at least.
		if !strings.Contains(engines, "window.GoSXStudioFieldRuntime =") {
			t.Fatalf("engine runtime missing fieldruntime shim assignment")
		}
	}
}

func TestEngineRuntimeIncludesSelectionRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-2 burn-down of GoSXStudioSelectionRuntime. The engine
	// runtime asset must include both the island-runtime JS (publishes the
	// __gosx_selection_runtime_island_bind global) and the BridgeShim that
	// delegates window.GoSXStudioSelectionRuntime calls when the
	// "selection-runtime-islands" feature flag is on. See
	// gosx-studio/selectionruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_selection_runtime_island_bind",
		"selection-runtime-islands",
		// The shim's flag check must be in the bundle so the runtime is
		// actually gated, not just loaded.
		"data-gosx-studio-feature-flag-selection-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing selectionruntime fragment %q", fragment)
		}
	}
	// The shim must override window.GoSXStudioSelectionRuntime; without
	// that assignment the legacy method is never wrapped.
	if !strings.Contains(engines, "window.GoSXStudioSelectionRuntime =") {
		t.Fatalf("engine runtime missing selectionruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesBrandRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-3 burn-down of GoSXStudioBrandRuntime. The engine
	// runtime asset must include both the island-runtime JS (publishes the
	// __gosx_brand_runtime_island_bindLogo and __gosx_brand_runtime_island_updateHeaderLogo
	// globals) and the BridgeShim that delegates window.GoSXStudioBrandRuntime
	// calls when the "brand-runtime-islands" feature flag is on. See
	// gosx-studio/brandruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_brand_runtime_island_bindLogo",
		"__gosx_brand_runtime_island_updateHeaderLogo",
		"brand-runtime-islands",
		// The shim's flag check must be in the bundle so the runtime is
		// actually gated, not just loaded.
		"data-gosx-studio-feature-flag-brand-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing brandruntime fragment %q", fragment)
		}
	}
	// The shim must override window.GoSXStudioBrandRuntime; without that
	// assignment the legacy methods are never wrapped.
	if !strings.Contains(engines, "window.GoSXStudioBrandRuntime =") {
		t.Fatalf("engine runtime missing brandruntime shim assignment")
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
