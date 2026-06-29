package studio

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func legacyMuddyGlobal(name string) string {
	return "__" + "muddy" + name
}

// The legacy monolithic JS bundles (studio-engines.js and
// preview-runtime.js) were deleted on 2026-05-27 — see Phase 3
// burn-down. The tests below cover the post-deletion contract: the
// engine runtime bundle is the concatenation of per-slice island
// bundles, and each slice publishes its shim globals so consumer code
// (window.GoSXStudio{Field,Selection,Brand,BlockLayout,Style,Workbench,Preview}Runtime)
// remains intact.

func TestStylesheetHandlerServesStudioChromeTokens(t *testing.T) {
	rec := httptest.NewRecorder()
	StylesheetHandler().ServeHTTP(rec, httptest.NewRequest("GET", StylesheetPath, nil))

	if rec.Code != 200 {
		t.Fatalf("stylesheet status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("stylesheet content type = %q", ct)
	}
	for _, check := range []string{
		"--size-studio-max",
		"--studio-stage-height",
		".gosx-studio",
		".admin-page",
		".admin-page .admin-heading",
		".admin-page .resource-grid",
		".admin-page .data-table",
		".admin-page .admin-form",
		"data-gosx-studio-authoring-selected",
		"data-gosx-studio-preview-patched",
		"data-gosx-studio-preview-dock",
		"data-gosx-studio-inline-editing",
		`[data-studio-site-filter="site"]`,
		`a[data-studio-site-group]:not([data-studio-site-group="site"])`,
	} {
		if !strings.Contains(rec.Body.String(), check) {
			t.Fatalf("stylesheet missing %q: %s", check, rec.Body.String())
		}
	}
}

func TestWorkbenchRuntimeDoesNotInjectPreviewStyles(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		"gosx-studio-preview-patch-style",
		"gosx-studio-preview-dock-style",
		`createElement("style")`,
	} {
		if strings.Contains(script, check) {
			t.Fatalf("workbench runtime should not contain preview style injector fragment %q", check)
		}
	}
}

func TestWorkbenchRuntimePrefersShellRenderedPreviewDock(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`"[data-gosx-studio-preview], [data-studio-preview-shell], [data-studio-canvas]"`,
		`frame.closest("[data-gosx-studio-preview], [data-studio-preview-shell], [data-studio-canvas]")`,
		`shell.querySelector("[data-studio-preview-dock], [data-gosx-studio-preview-dock]")`,
		`return normalizePreviewDock(frame, dock)`,
		"if (dock) return normalizePreviewDock(frame, dock);\n      return null;",
		`dock.querySelector("[data-studio-preview-dock-title]")`,
		`dock.querySelector("[data-studio-preview-field-count]")`,
		`event.target.closest ? event.target.closest("[data-gosx-studio-preview-command], [data-studio-preview-action]")`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing shell-rendered dock contract %q", check)
		}
	}
	for _, check := range []string{
		`function createDockButton`,
		`data-gosx-studio-preview-dock-fallback`,
		`dock = document.createElement("div")`,
		`actions.appendChild(createDockButton`,
	} {
		if strings.Contains(script, check) {
			t.Fatalf("workbench runtime should not contain preview dock fallback fragment %q", check)
		}
	}
}

func TestWorkbenchRuntimeLetsFieldRuntimeOwnMirroredPreviewPatches(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function fieldRuntimeMirrors(field)`,
		`field.matches("[data-editor-source], [data-studio-field-source], [data-editor-frame-attr-target]")`,
		`emitFieldOperation("input", event.target);`,
		`if (!fieldRuntimeMirrors(event.target)) postPreviewPatch("input", {}, event.target);`,
		`emitFieldOperation("change", event.target);`,
		`if (!fieldRuntimeMirrors(event.target)) postPreviewPatch("change", {}, event.target);`,
		`postPreviewPatch("load-sync", { route: previewURL(frame) || frame.getAttribute("src") || "" }, null);`,
		`postPreviewPatch("transaction", event.detail || {}, null);`,
		`postPreviewPatch("history-restore", event.detail || {}, null);`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing FieldRuntime preview patch ownership fragment %q", check)
		}
	}
}

func TestWorkbenchRuntimeSkipsMirroredFieldsDuringPreviewLoadSync(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`queryAll(form, "[data-studio-field-source], [data-editor-source], input[name], textarea[name], select[name]").forEach(function (field) {`,
		`if (fieldRuntimeMirrors(field)) return;`,
		`count += applyPreviewPatch(frame, patch);`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing load-sync mirrored-field skip/fallback fragment %q", check)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewInlineTextToInlineEditRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		"window.GoSXStudioInlineEditRuntime",
		"startPreviewTextEdit",
		"syncPreviewTextEdit",
		"finishPreviewTextEdit",
		"previewInlineTextOptions",
		"controlForField: textControlForField",
		"emitOperation: function (type, operation)",
		"emitEvent: function (name, detail)",
		"onFinish: function (finishedFrame)",
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing InlineEditRuntime preview inline-text delegate fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		"function placeCaretAtEnd",
		`target.setAttribute("contenteditable", "plaintext-only")`,
		`target.removeAttribute("contenteditable")`,
		"function inlineTextPayload",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should delegate preview inline-text lifecycle, found low-level fragment %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeLeavesSelectionActionsToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, forbidden := range []string{
		`function runSelectionAction(button)`,
		`function runSelectionTarget(target, label)`,
		`detail.kind === "selection-action"`,
		`event.target.closest("[data-studio-selection-action]")`,
		`[data-studio-selection-action="`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should not own selection action bridge fragment %q", forbidden)
		}
	}
	for _, check := range []string{
		`function runInsert(button)`,
		`emit(form, "gosxstudio:insert-block", detail)`,
		`if (detail.kind === "insert") return runInsertTarget(detail.target, detail.label);`,
		`var mode = event.target.closest("[data-studio-mode-control]")`,
		`var rail = event.target.closest("[data-studio-rail-toggle]")`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing non-selection chrome/insert fragment %q", check)
		}
	}
}

func TestLegacyRuntimeHandlersServeStudioOwnedAssets(t *testing.T) {
	for name, tt := range map[string]struct {
		path    string
		script  []byte
		handler func() *httptest.ResponseRecorder
	}{
		"workbench": {
			path:   WorkbenchRuntimePath,
			script: WorkbenchRuntimeScript(),
			handler: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				WorkbenchRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", WorkbenchRuntimePath, nil))
				return rec
			},
		},
		"command": {
			path:   CommandRuntimePath,
			script: CommandRuntimeScript(),
			handler: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				CommandRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", CommandRuntimePath, nil))
				return rec
			},
		},
		"state": {
			path:   StateRuntimePath,
			script: StateRuntimeScript(),
			handler: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				StateRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", StateRuntimePath, nil))
				return rec
			},
		},
	} {
		if len(tt.script) == 0 {
			t.Fatalf("%s runtime script is empty", name)
		}
		rec := tt.handler()
		if rec.Code != 200 {
			t.Fatalf("%s runtime status = %d", name, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
			t.Fatalf("%s runtime content type = %q", name, ct)
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("%s runtime handler body is empty for %s", name, tt.path)
		}
		if rec.Body.String() != string(tt.script) {
			t.Fatalf("%s runtime handler body does not match script", name)
		}
	}
}

func TestCanvasInlineRuntimeHandlersServeStudioOwnedAssets(t *testing.T) {
	for name, tt := range map[string]struct {
		path   string
		script []byte
		checks []string
		serve  func() *httptest.ResponseRecorder
	}{
		"canvas-inline-edit": {
			path:   CanvasInlineEditPath,
			script: CanvasInlineEditScript(),
			checks: []string{
				"window.GoSXStudioCanvasInlineEditRuntime",
				"persistRepaintSafe",
			},
			serve: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				CanvasInlineEditHandler().ServeHTTP(rec, httptest.NewRequest("GET", CanvasInlineEditPath, nil))
				return rec
			},
		},
		"canvas-selection-bridge": {
			path:   CanvasSelectionBridgePath,
			script: CanvasSelectionBridgeScript(),
			checks: []string{
				"window.GoSXStudioCanvasSelectionBridgeRuntime",
				"$surface.event.selectedID",
				"$surface.event.selectedIDs",
				"GoSXStudioSiteMapRuntime",
			},
			serve: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				CanvasSelectionBridgeHandler().ServeHTTP(rec, httptest.NewRequest("GET", CanvasSelectionBridgePath, nil))
				return rec
			},
		},
		"canvas2d-painter": {
			path:   Canvas2DPainterPath,
			script: Canvas2DPainterScript(),
			checks: []string{
				"window.GoSXStudioCanvas2DPainterRuntime",
				"paint: paintCanvasBundle",
			},
			serve: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				Canvas2DPainterHandler().ServeHTTP(rec, httptest.NewRequest("GET", Canvas2DPainterPath, nil))
				return rec
			},
		},
		"canvas-wasm-free-client": {
			path:   CanvasWASMFreeClientPath,
			script: CanvasWASMFreeClientScript(),
			checks: []string{
				"window.GoSXStudioCanvasWASMFreeClientRuntime",
				"canvas.__gosxStudioCanvasWASMFree",
				"canvas.GoSXStudioCanvasWasmFree",
				"data-gosx-canvas-wasm-free-client",
			},
			serve: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				CanvasWASMFreeClientHandler().ServeHTTP(rec, httptest.NewRequest("GET", CanvasWASMFreeClientPath, nil))
				return rec
			},
		},
		"canvas-default-inline-installer": {
			path:   CanvasDefaultInlineInstallerPath,
			script: CanvasDefaultInlineInstallerScript(),
			checks: []string{
				"window.GoSXStudioCanvasDefaultInlineInstallerRuntime",
				"data-studio-site-map-canvas-default",
			},
			serve: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				CanvasDefaultInlineInstallerHandler().ServeHTTP(rec, httptest.NewRequest("GET", CanvasDefaultInlineInstallerPath, nil))
				return rec
			},
		},
		"canvas-contextual-panel": {
			path:   CanvasContextualPanelPath,
			script: CanvasContextualPanelScript(),
			checks: []string{
				"window.GoSXStudioCanvasContextualPanelRuntime",
				"GoSXStudioCanvasInlineEditRuntime",
				"render: render",
			},
			serve: func() *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				CanvasContextualPanelHandler().ServeHTTP(rec, httptest.NewRequest("GET", CanvasContextualPanelPath, nil))
				return rec
			},
		},
	} {
		if len(tt.script) == 0 {
			t.Fatalf("%s runtime script is empty", name)
		}
		rec := tt.serve()
		if rec.Code != 200 {
			t.Fatalf("%s runtime status = %d", name, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
			t.Fatalf("%s runtime content type = %q", name, ct)
		}
		if rec.Body.String() != string(tt.script) {
			t.Fatalf("%s runtime handler body does not match script", name)
		}
		for _, check := range tt.checks {
			if !strings.Contains(rec.Body.String(), check) {
				t.Fatalf("%s runtime handler body missing %q", name, check)
			}
		}
		for _, forbidden := range []string{
			legacyMuddyGlobal("CanvasInlineEdit"),
			legacyMuddyGlobal("Canvas2DPainter"),
			legacyMuddyGlobal("CanvasWasmFreeClient"),
			legacyMuddyGlobal("CanvasDefaultInlineInstaller"),
			legacyMuddyGlobal("CanvasContextualPanel"),
			legacyMuddyGlobal("CanvasWasmFree"),
		} {
			if strings.Contains(rec.Body.String(), forbidden) {
				t.Fatalf("%s runtime handler body should not contain %q", name, forbidden)
			}
		}
	}
}

func TestEngineRuntimeIncludesFieldRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-1 GoSXStudioFieldRuntime island contract. The engine
	// runtime asset must include the island-runtime JS (publishes the
	// __gosx_field_runtime_island_* globals) and the BridgeShim that
	// installs window.GoSXStudioFieldRuntime. See
	// gosx-studio/fieldruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_field_runtime_island_bind",
		"__gosx_field_runtime_island_bindMirroring",
		"__gosx_field_runtime_island_bindClipboard",
		"field-runtime-islands",
		"data-gosx-studio-feature-flag-field-runtime-islands",
		"data-studio-field-source",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing fieldruntime fragment %q", fragment)
		}
	}
	if !strings.Contains(engines, "window.GoSXStudioFieldRuntime =") {
		t.Fatalf("engine runtime missing fieldruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesSelectionRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-2 GoSXStudioSelectionRuntime island contract. See
	// gosx-studio/selectionruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_selection_runtime_island_bind",
		"selection-runtime-islands",
		"data-gosx-studio-feature-flag-selection-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing selectionruntime fragment %q", fragment)
		}
	}
	if !strings.Contains(engines, "window.GoSXStudioSelectionRuntime =") {
		t.Fatalf("engine runtime missing selectionruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesStyleRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-5 GoSXStudioStyleRuntime island contract. See
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
		"data-gosx-studio-feature-flag-style-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing styleruntime fragment %q", fragment)
		}
	}
	if !strings.Contains(engines, "window.GoSXStudioStyleRuntime =") {
		t.Fatalf("engine runtime missing styleruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesBlockLayoutRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-4 GoSXStudioBlockLayoutRuntime island contract. See
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
		"data-gosx-studio-feature-flag-block-layout-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing blocklayoutruntime fragment %q", fragment)
		}
	}
	if !strings.Contains(engines, "window.GoSXStudioBlockLayoutRuntime =") {
		t.Fatalf("engine runtime missing blocklayoutruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesBrandRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-3 GoSXStudioBrandRuntime island contract. See
	// gosx-studio/brandruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_brand_runtime_island_bindLogo",
		"__gosx_brand_runtime_island_updateHeaderLogo",
		"brand-runtime-islands",
		"data-gosx-studio-feature-flag-brand-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing brandruntime fragment %q", fragment)
		}
	}
	if !strings.Contains(engines, "window.GoSXStudioBrandRuntime =") {
		t.Fatalf("engine runtime missing brandruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesWorkbenchRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-7 GoSXStudioWorkbenchRuntime island contract. See
	// gosx-studio/workbenchruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"__gosx_workbench_runtime_island_bindRailResizers",
		"__gosx_workbench_runtime_island_bindChrome",
		"__gosx_workbench_runtime_island_setMode",
		"__gosx_workbench_runtime_island_syncViewport",
		"__gosx_workbench_runtime_island_activateViewport",
		"__gosx_workbench_runtime_island_currentBreakpoint",
		"__gosx_workbench_runtime_island_setStyleState",
		"__gosx_workbench_runtime_island_syncZoom",
		"__gosx_workbench_runtime_island_activateZoom",
		"__gosx_workbench_runtime_island_toggleRail",
		"__gosx_workbench_runtime_island_toggleFocus",
		"__gosx_workbench_runtime_island_toggleActivity",
		"__gosx_workbench_runtime_island_saveLayout",
		"__gosx_workbench_runtime_island_currentRailWidth",
		"__gosx_workbench_runtime_island_setRailWidth",
		"workbench-runtime-islands",
		"data-gosx-studio-feature-flag-workbench-runtime-islands",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing workbenchruntime fragment %q", fragment)
		}
	}
	if !strings.Contains(engines, "window.GoSXStudioWorkbenchRuntime =") {
		t.Fatalf("engine runtime missing workbenchruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesPreviewRuntimeIslandBundle(t *testing.T) {
	// Phase 3 slice-6 architectural island contract for
	// GoSXStudioPreviewRuntime per ADR 0008 (canonical) + ADR 0009
	// (cross-frame transport).
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
		"data-gosx-studio-feature-flag-preview-runtime-islands",
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
	if !strings.Contains(engines, "window.GoSXStudioPreviewRuntime =") {
		t.Fatalf("engine runtime missing previewruntime shim assignment")
	}
}

func TestEngineRuntimeIncludesAuthoringRuntimeBundle(t *testing.T) {
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"window.GoSXStudioAuthoringRuntime",
		"gosx:form:result",
		"gosxstudio:authoring-result",
		"data-gosx-studio-authoring-selected",
		"data-gosx-studio-preview-url",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing authoringruntime fragment %q", fragment)
		}
	}
}

func TestEngineRuntimeIncludesSiteMapRuntimeBundle(t *testing.T) {
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"window.GoSXStudioSiteMapRuntime",
		"data-studio-site-map-filter-control",
		"data-studio-site-map-detail-target",
		"data-studio-site-map-palette-control",
		"data-studio-site-map-selected-node",
		"gosxstudio:site-map-change",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing sitemapruntime fragment %q", fragment)
		}
	}
}

func TestEngineRuntimeIncludesInlineEditRuntimeBundle(t *testing.T) {
	// InlineEditRuntime island contract — host-agnostic inline editing.
	// See gosx-studio/inlineeditruntime/runtime.go.
	engines := string(EngineRuntimeScript())
	for _, fragment := range []string{
		"window.GoSXStudioInlineEditRuntime",
		"deriveKeys",
		"data-studio-field",
		"data-gosx-studio-inline-edit-action",
		"data-gosx-studio-authoring-managed",
	} {
		if !strings.Contains(engines, fragment) {
			t.Fatalf("engine runtime missing inlineeditruntime fragment %q", fragment)
		}
	}
}

func TestAuthoringRuntimeScriptIsNonEmpty(t *testing.T) {
	runtime := string(AuthoringRuntimeScript())
	if runtime == "" {
		t.Fatal("AuthoringRuntimeScript() must return non-empty bundle")
	}
	if !strings.Contains(runtime, "GoSXStudioAuthoringRuntime") {
		t.Fatalf("AuthoringRuntimeScript() missing authoring runtime global")
	}
}

func TestSiteMapRuntimeScriptIsNonEmpty(t *testing.T) {
	runtime := string(SiteMapRuntimeScript())
	if runtime == "" {
		t.Fatal("SiteMapRuntimeScript() must return non-empty bundle")
	}
	if !strings.Contains(runtime, "GoSXStudioSiteMapRuntime") {
		t.Fatalf("SiteMapRuntimeScript() missing site-map runtime global")
	}
}

func TestPreviewSubscriberScriptIsNonEmpty(t *testing.T) {
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
	if strings.Contains(body, "__gosx_preview_runtime_island_mount") {
		t.Fatalf("preview subscriber must not contain editor-side island writer globals")
	}
}

func TestEngineRuntimeHandlerServesIslandBundles(t *testing.T) {
	rec := httptest.NewRecorder()
	EngineRuntimeHandler().ServeHTTP(rec, httptest.NewRequest("GET", EngineRuntimePath, nil))

	if rec.Code != 200 {
		t.Fatalf("engine runtime status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/javascript") {
		t.Fatalf("engine runtime content type = %q", ct)
	}
	body := rec.Body.String()
	// The handler serves the concatenated island bundles; each slice's
	// shim installs its window.GoSXStudio* global. We sample one shim
	// from each slice to verify the full chain made it into the body.
	for _, check := range []string{
		"window.GoSXStudioFieldRuntime =",
		"window.GoSXStudioSelectionRuntime =",
		"window.GoSXStudioBrandRuntime =",
		"window.GoSXStudioBlockLayoutRuntime =",
		"window.GoSXStudioStyleRuntime =",
		"window.GoSXStudioWorkbenchRuntime =",
		"window.GoSXStudioPreviewRuntime =",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("engine runtime body missing %q", check)
		}
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
