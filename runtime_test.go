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

func TestEngineRuntimeIncludesStyleRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-5 burn-down of GoSXStudioStyleRuntime. The engine
	// runtime asset must include both the island-runtime JS (publishes the
	// ten __gosx_style_runtime_island_* globals) and the BridgeShim that
	// delegates window.GoSXStudioStyleRuntime calls when the
	// "style-runtime-islands" feature flag is on. See
	// gosx-studio/styleruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_style_runtime_island_bindTheme",
		"__gosx_style_runtime_island_bindWorkbench",
		"__gosx_style_runtime_island_bindCSS",
		"__gosx_style_runtime_island_bindFonts",
		"__gosx_style_runtime_island_applyTheme",
		"__gosx_style_runtime_island_syncControlButtons",
		"__gosx_style_runtime_island_showImpact",
		"__gosx_style_runtime_island_restoreImpact",
		"__gosx_style_runtime_island_setControlValue",
		"__gosx_style_runtime_island_resetControlValue",
		"style-runtime-islands",
		// The shim's flag check must be in the bundle so the runtime is
		// actually gated, not just loaded.
		"data-gosx-studio-feature-flag-style-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing styleruntime fragment %q", fragment)
		}
	}
	// The shim must override window.GoSXStudioStyleRuntime; without that
	// assignment the legacy methods are never wrapped. The bundle already
	// declares the legacy version of the global via studio-engines.js at
	// line 2341; the slice-5 shim reassigns it.
	if !strings.Contains(engines, "window.GoSXStudioStyleRuntime =") {
		t.Fatalf("engine runtime missing styleruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesBlockLayoutRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-4 burn-down of GoSXStudioBlockLayoutRuntime. The engine
	// runtime asset must include both the island-runtime JS (publishes the
	// nine __gosx_blocklayout_runtime_island_* globals) and the BridgeShim
	// that delegates window.GoSXStudioBlockLayoutRuntime calls when the
	// "block-layout-runtime-islands" feature flag is on. See
	// gosx-studio/blocklayoutruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_blocklayout_runtime_island_rows",
		"__gosx_blocklayout_runtime_island_rowKey",
		"__gosx_blocklayout_runtime_island_rowForKey",
		"__gosx_blocklayout_runtime_island_moveRow",
		"__gosx_blocklayout_runtime_island_renumber",
		"__gosx_blocklayout_runtime_island_selectRow",
		"__gosx_blocklayout_runtime_island_commitReorder",
		"__gosx_blocklayout_runtime_island_updateBlockLibraryState",
		"__gosx_blocklayout_runtime_island_updateVisibilityState",
		"block-layout-runtime-islands",
		// The shim's flag check must be in the bundle so the runtime is
		// actually gated, not just loaded.
		"data-gosx-studio-feature-flag-block-layout-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing blocklayoutruntime fragment %q", fragment)
		}
	}
	// The shim must override window.GoSXStudioBlockLayoutRuntime; without
	// that assignment the legacy methods are never wrapped. The bundle
	// already declares the legacy version of the global via studio-engines.js
	// at line 2353; the slice-4 shim reassigns it.
	if !strings.Contains(engines, "window.GoSXStudioBlockLayoutRuntime =") {
		t.Fatalf("engine runtime missing blocklayoutruntime shim assignment")
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

func TestEngineRuntimeIncludesPreviewRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-6 architectural burn-down of GoSXStudioPreviewRuntime.
	// The engine runtime asset must include both the editor-side island-
	// runtime JS (publishes the ten __gosx_preview_runtime_island_* globals)
	// and the BridgeShim that delegates window.GoSXStudioPreviewRuntime
	// calls to $preview.* shared-signal writes when the
	// "preview-runtime-islands" feature flag is on. See
	// gosx-studio/previewruntime/runtime.go and
	// ~/.hyphae/spaces/m31labs-gosx/decisions/0009-iframe-transport-postmessage-relay.md
	// for the cross-frame transport that delivers the writes to the
	// iframe-side subscriber.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_preview_runtime_island_mount",
		"__gosx_preview_runtime_island_setBlockVisibility",
		"__gosx_preview_runtime_island_applyTextUpdate",
		"__gosx_preview_runtime_island_applyTheme",
		"__gosx_preview_runtime_island_applyStyleImpact",
		"__gosx_preview_runtime_island_applyCSS",
		"__gosx_preview_runtime_island_applyFonts",
		"__gosx_preview_runtime_island_updateHeaderLogo",
		"__gosx_preview_runtime_island_requestInlineEdit",
		"__gosx_preview_runtime_island_cycleField",
		"preview-runtime-islands",
		// The shim's flag check must be in the bundle so the runtime is
		// actually gated, not just loaded.
		"data-gosx-studio-feature-flag-preview-runtime-islands",
		// Signal-namespace contract per ADR 0008 (canonical) + ADR 0009
		// (transport). Drift in any of these silently breaks the cross-
		// frame contract — the editor would write to a signal the
		// preview-side subscriber doesn't observe.
		"$preview.mount.epoch",
		"$preview.block.",
		"$preview.text.",
		"$preview.theme.kit",
		"$preview.style.impact.selector",
		"$preview.theme.cssCustom",
		"$preview.theme.fontsCustom",
		"$preview.brand.headerLogo",
		"$preview.editor.inlineEdit.requestId",
		"$preview.editor.fieldCycle.requestId",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing previewruntime fragment %q", fragment)
		}
	}
	// The shim must override window.GoSXStudioPreviewRuntime; without that
	// assignment the legacy methods (declared by preview-runtime.js at
	// line 1118) are never wrapped and the feature flag has no effect.
	// The shim's reassignment is the slice's atomic flip mechanism.
	if !strings.Contains(engines, "window.GoSXStudioPreviewRuntime =") {
		t.Fatalf("engine runtime missing previewruntime shim assignment")
	}
}

func TestPreviewSubscriberScriptIsNonEmpty(t *testing.T) {
	// PreviewSubscriberScript() is the bundle the storefront mounts when
	// loaded in the editor preview iframe (per ADR 0009's preview-mode
	// bootstrap). It must be non-empty and contain the $preview.* observer
	// registrations the editor-side writes target.
	subscriber := string(PreviewSubscriberScript())
	if subscriber == "" {
		t.Fatal("PreviewSubscriberScript() must return non-empty bundle")
	}
	for _, signalName := range []string{
		"$preview.mount.epoch",
		"$preview.block.",
		"$preview.text.",
		"$preview.theme.kit",
		"$preview.brand.headerLogo",
	} {
		if !strings.Contains(subscriber, signalName) {
			t.Fatalf("PreviewSubscriberScript() missing observer for %q", signalName)
		}
	}
}

func TestPreviewSubscriberHandlerServesSubscriberScript(t *testing.T) {
	rec := httptest.NewRecorder()
	PreviewSubscriberHandler().ServeHTTP(rec, httptest.NewRequest("GET", PreviewSubscriberPath, nil))
	if rec.Code != 200 {
		t.Fatalf("preview subscriber status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Fatalf("preview subscriber content type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "$preview.") {
		t.Fatalf("preview subscriber should observe $preview.* signals: %s", body)
	}
	// The subscriber bundle must be SEPARATE from the editor bundle so
	// storefront pages don't ship the editor-side island writer code.
	// We assert this by checking the editor's writer globals are NOT in
	// the subscriber bundle (cf. EngineRuntimeScript which has them).
	if strings.Contains(body, "__gosx_preview_runtime_island_mount") {
		t.Fatalf("preview subscriber must not contain editor-side island writer globals")
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
