package previewruntime

import (
	"strings"
	"testing"
)

func islandJSFunctionBody(t *testing.T, script, name string) string {
	t.Helper()
	start := strings.Index(script, "function "+name+"(")
	if start < 0 {
		t.Fatalf("script missing function %s", name)
	}
	next := strings.Index(script[start+len("function "+name+"("):], "\n  function ")
	if next < 0 {
		return script[start:]
	}
	return script[start : start+len("function "+name+"(")+next]
}

func assertRetiredEditorPreviewRuntimeEventAbsent(t *testing.T, script, eventName string) {
	t.Helper()
	for _, forbidden := range []string{
		`doc.addEventListener("` + eventName + `"`,
		`dispatchEvent(new CustomEvent("` + eventName,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("IslandRuntimeJS() must not use retired event boundary %q; found %q", eventName, forbidden)
		}
	}
}

// The previewruntime package is the Go-side surface for the Phase 3 slice-6
// architectural burn-down of GoSXStudioPreviewRuntime. It owns:
//  1. The feature-flag key consumers can keep in
//     studio.ShellConfig.FeatureFlags as a stable host probe/string contract.
//  2. The JS shim that gosx-studio's runtime bundle appends so the
//     window.GoSXStudioPreviewRuntime methods delegate directly to island
//     globals that publish to $preview.* shared signals.
//  3. The editor-side island_runtime.js that publishes
//     window.__gosx_preview_runtime_island_* globals the BridgeShim
//     delegates to.
//  4. The preview-side subscriber_runtime.js — emitted by
//     PreviewSubscriberScript() — that the storefront mounts when loaded
//     inside the editor preview iframe (per ADR 0009's cross-frame
//     postMessage relay).
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-6-previewruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the canonical iframe-stays-via-signals decision (status-corrected
// 2026-05-26); and ~/.hyphae/spaces/m31labs-gosx/decisions/0009-iframe-transport-postmessage-relay.md
// for the cross-frame transport that makes Path A deliverable.

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if FeatureFlagKey != "preview-runtime-islands" {
		t.Fatalf("FeatureFlagKey must be %q per Phase 3 slice-6 contract; got %q", "preview-runtime-islands", FeatureFlagKey)
	}
	if !strings.HasSuffix(FeatureFlagKey, "-runtime-islands") {
		t.Fatalf("FeatureFlagKey %q should end with -runtime-islands per Phase 3 convention", FeatureFlagKey)
	}
}

func TestBridgeShimDelegatesToIslandGlobals(t *testing.T) {
	shim := string(BridgeShim())
	if shim == "" {
		t.Fatal("BridgeShim() must return a non-empty JS snippet")
	}
	// The shim must reference each method-specific island global and the
	// public global it exposes (window.GoSXStudioPreviewRuntime). The
	// IslandGlobals struct holds the canonical names; any drift between the
	// struct and the shim is caught here.
	for _, fragment := range []string{
		"window.GoSXStudioPreviewRuntime",
		"return island.apply(null, arguments)",
		"return undefined",
		IslandGlobals.Mount,
		IslandGlobals.SetBlockVisibility,
		IslandGlobals.ApplyTextUpdate,
		IslandGlobals.ApplyTheme,
		IslandGlobals.ApplyStyleImpact,
		IslandGlobals.ApplyCSS,
		IslandGlobals.ApplyFonts,
		IslandGlobals.UpdateHeaderLogo,
		IslandGlobals.RequestInlineEdit,
		IslandGlobals.CycleField,
		FeatureFlagKey,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
	for _, stale := range []string{
		"FLAG" + "_ATTR",
		"flag" + "Enabled",
		"data-gosx-studio-feature-flag-" + "preview-runtime-islands",
		"if (!" + "flag" + "Enabled()) return undefined",
		"legacy " + "path",
		"legacy " + "fallback",
		"fallback " + "branch",
		"fallback " + "path",
	} {
		if strings.Contains(shim, stale) {
			t.Fatalf("BridgeShim() contains stale gate/fallback fragment %q:\n%s", stale, shim)
		}
	}
}

func TestIslandRuntimeJSPublishesMountGlobal(t *testing.T) {
	// Method 1/10: mount(root) — legacy
	// window.GoSXStudioPreviewRuntime.mount at the legacy bundle:1119
	// (bound to init at line 1113). The legacy walks every workbench form
	// in root and calls bindPreview(form). The island writer publishes
	// $preview.mount.epoch — the subscriber re-runs its mount-acknowledged
	// routine when the epoch changes.
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	want := "window." + IslandGlobals.Mount + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Signal target — drift here breaks the cross-frame contract.
		"$preview.mount.epoch",
		`function editorPreviewWorkbenchForms(root)`,
		`scope.matches("form[data-studio-workbench], form[data-editor-workbench]")`,
		`forms.push(scope)`,
		`scope.querySelectorAll("form[data-studio-workbench], form[data-editor-workbench]")`,
		`editorPreviewWorkbenchForms(root).forEach(function (form)`,
		`syncEditorPreviewDockPositions(form)`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() mount must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSetBlockVisibilityGlobal(t *testing.T) {
	// Method 2/10: setBlockVisibility(key, visible) — legacy at preview-
	// runtime.js:873 (calls setPreviewBlockVisibility per .editor-preview-
	// frame at line 866). The island writer publishes
	// $preview.block.<key>.visible — the subscriber toggles
	// display: none on the matching [data-studio-block-key="<key>"] node.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SetBlockVisibility + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Signal target — uses dynamic <key> infix, so the static prefix
		// is what we assert.
		"$preview.block.",
		".visible",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() setBlockVisibility must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesApplyTextUpdateGlobal(t *testing.T) {
	// Method 3/10: applyTextUpdate(detail) — legacy at preview-
	// runtime.js:912. The legacy applies the value to three potential
	// targets: [data-editor-preview="<sourceKey>"] textContent,
	// detail.frameTarget selector textContent, and detail.attrTarget
	// attribute (with prefix/suffix). The island writer publishes the
	// whole detail object to $preview.text.<key> where key falls back
	// through sourceKey / frameTarget / attrTarget / "_". The subscriber
	// re-emits all three branches.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ApplyTextUpdate + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		"$preview.text.",
		// Payload fields the legacy and signal-based subscriber both
		// consume — drift here silently breaks the text-update fan-out.
		"sourceKey",
		"frameTarget",
		"attrTarget",
		"attrName",
		"attrPrefix",
		"attrSuffix",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() applyTextUpdate must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesApplyThemeGlobal(t *testing.T) {
	// Method 4/10: applyTheme(detail) — legacy at the legacy bundle:938.
	// The legacy rebuilds .site-shell classes (clearing theme--template-
	// / -kit- / -custom- / -nav- / -buttons- / -cards- / -spacing- /
	// -images- / -motion- / -palette- / -image- prefixes) and reapplies
	// new classes; sets each color-token CSS variable. The island writer
	// publishes a family of $preview.theme.* signals; the subscriber
	// reassembles them in a module-scope buffer and applies on every
	// write.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ApplyTheme + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		"$preview.theme.kit",
		"$preview.theme.template",
		"$preview.theme.palette",
		"$preview.theme.imageRatio",
		"$preview.theme.customClasses",
		"$preview.theme.styleClasses",
		"$preview.theme.colors",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() applyTheme must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesApplyStyleImpactGlobal(t *testing.T) {
	// Method 5/10: applyStyleImpact(selector) — legacy at preview-
	// runtime.js:986. The legacy marks every matching node with
	// [data-studio-style-impact-node] and returns the count. The signal-
	// based path publishes the selector and returns a best-effort
	// editor-side count (queried via the editor's view of the iframe);
	// the subscriber performs the authoritative attribute mutation.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ApplyStyleImpact + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		"$preview.style.impact.selector",
		// Editor-side advisory count: the editor's view of the iframe
		// via .editor-preview-frame contentDocument.
		".editor-preview-frame",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() applyStyleImpact must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesApplyCSSGlobal(t *testing.T) {
	// Method 6/10: applyCSS(cssText) — legacy at the legacy bundle:1019.
	// The legacy ensures a <style data-editor-live-css> element in the
	// iframe head and sets its textContent. The island writer publishes
	// to $preview.theme.cssCustom.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ApplyCSS + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	if !strings.Contains(body, "$preview.theme.cssCustom") {
		t.Fatalf("IslandRuntimeJS() applyCSS must publish $preview.theme.cssCustom")
	}
}

func TestIslandRuntimeJSPublishesApplyFontsGlobal(t *testing.T) {
	// Method 7/10: applyFonts(cssText) — legacy at the legacy bundle:1023.
	// Same shape as applyCSS but with [data-editor-live-fonts] marker.
	// The island writer publishes to $preview.theme.fontsCustom.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ApplyFonts + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	if !strings.Contains(body, "$preview.theme.fontsCustom") {
		t.Fatalf("IslandRuntimeJS() applyFonts must publish $preview.theme.fontsCustom")
	}
}

func TestIslandRuntimeJSPublishesUpdateHeaderLogoGlobal(t *testing.T) {
	// Method 8/10: updateHeaderLogo(detail) — legacy at preview-
	// runtime.js:1027. The legacy applies the payload to the iframe's
	// .brand element (img src/alt, --brand-logo-width / -offset-x /
	// -offset-y CSS variables, .brand--with-logo class toggle). The
	// island writer publishes a single $preview.brand.headerLogo signal
	// carrying src / alt / width / x / y.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.UpdateHeaderLogo + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		"$preview.brand.headerLogo",
		// Payload fields the subscriber decomposes onto the .brand DOM.
		// The slice plan lists src / alt / width as the sub-keys;
		// offsets x / y are carried in the payload for legacy parity.
		"src:",
		"alt:",
		"width:",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() updateHeaderLogo must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesRequestInlineEditGlobal(t *testing.T) {
	// Method 9/10: requestInlineEdit(detail) — legacy at preview-
	// runtime.js:1049. The legacy walks previewControllers (the
	// bindPreview state) and dispatches requestInlineEdit on the first
	// matching one. The signal-based path publishes a monotonic
	// requestId to $preview.editor.inlineEdit.requestId so repeated
	// identical detail payloads still trigger the subscriber.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.RequestInlineEdit + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	if !strings.Contains(body, "$preview.editor.inlineEdit.requestId") {
		t.Fatalf("IslandRuntimeJS() requestInlineEdit must publish $preview.editor.inlineEdit.requestId")
	}
}

func TestIslandRuntimeJSPublishesCycleFieldGlobal(t *testing.T) {
	// Method 10/10: cycleField(detail) — legacy at preview-
	// runtime.js:1058. Same shape as requestInlineEdit. The signal-based
	// path publishes a monotonic requestId to
	// $preview.editor.fieldCycle.requestId.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.CycleField + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	if !strings.Contains(body, "$preview.editor.fieldCycle.requestId") {
		t.Fatalf("IslandRuntimeJS() cycleField must publish $preview.editor.fieldCycle.requestId")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewChromeHelpers(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`window.__gosx_preview_runtime_island_setStatus = function (form, state, label, reason)`,
		`return setEditorPreviewStatus(form, state, label, reason);`,
		`window.__gosx_preview_runtime_island_syncRoute = function (form, route, reason)`,
		`return syncEditorPreviewRoute(form, route, reason);`,
		`window.__gosx_preview_runtime_island_refreshNow = function (form, reason, route)`,
		`return refreshEditorPreviewNow(form, reason, route);`,
		`window.__gosx_preview_runtime_island_scheduleRefresh = function (form, reason, route)`,
		`return scheduleEditorPreviewRefresh(form, reason, route);`,
		`detail.form && detail.form.querySelectorAll`,
		`closest("form[data-studio-workbench], form[data-editor-workbench]")`,
		`"[data-gosx-studio-preview], [data-studio-preview-shell], [data-studio-canvas]"`,
		`".editor-preview-frame, [data-studio-preview-frame]"`,
		`"[data-studio-preview-status]"`,
		`"[data-studio-open-preview]"`,
		`"[data-studio-selected-flow-route]"`,
		`shell.setAttribute("data-gosx-studio-preview-state", state || "")`,
		`shell.setAttribute("data-gosx-studio-preview-reason", reason || "")`,
		`frame.setAttribute("data-studio-preview-state", state || "")`,
		`shell.setAttribute("data-gosx-studio-preview-url", route)`,
		`frame.setAttribute("data-studio-preview-src", route)`,
		`link.setAttribute("href", route)`,
		`node.textContent = route`,
		`emitEditorPreview(form, "gosxstudio:preview-route"`,
		`emitEditorPreview(form, "gosxstudio:preview-refresh"`,
		`next.searchParams.set("_gosx_preview", String(Date.now()))`,
		`next.searchParams.set("_gosx_preview_reason", reason)`,
		`frame.setAttribute("src", cacheBustPreviewURL(base, reason))`,
		`setEditorPreviewStatus(form, "loading", "Refreshing preview", reason)`,
		`window.setTimeout(function () {`,
		`}, 180)`,
		`typeof WeakMap !== "undefined" ? new WeakMap() : null`,
		`form.__gosxStudioEditorPreviewRefreshTimer`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview chrome fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-status-set"`,
		`doc.addEventListener("gosxstudio:editor-preview-route-sync"`,
		`doc.addEventListener("gosxstudio:editor-preview-refresh-now"`,
		`doc.addEventListener("gosxstudio:editor-preview-refresh-schedule"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-status-set"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-route-sync"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-refresh-now"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-refresh-schedule"`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() should use editor preview chrome helpers, found %q", forbidden)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewPatchTransport(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`window.__gosx_preview_runtime_island_postPatch = function (form, reason, detail, field)`,
		`return postEditorPreviewPatch(form, reason, editorPreviewPatchEnvelope(reason, detail, field));`,
		`function editorPreviewPatchEnvelope(reason, detail, field)`,
		`reason: reason || "patch"`,
		`detail: detail || {}`,
		`field: editorPreviewFieldPatch(field)`,
		`function editorPreviewFrameDocument(frame)`,
		`frame.contentDocument || (frame.contentWindow && frame.contentWindow.document) || null`,
		`function editorPreviewPatchSelector(source)`,
		`'[data-studio-field="' + source + '"]'`,
		`'[data-editor-preview="' + source + '"]'`,
		`'[data-studio-field-source="' + source + '"]'`,
		`function updateEditorPreviewPatchTarget(target, field)`,
		`target.value = value`,
		`if (field.type === "checkbox" || field.type === "radio") target.checked = !!field.checked`,
		`target.setAttribute("src", value)`,
		`target.setAttribute("href", value || "#")`,
		`target.textContent = value`,
		`target.setAttribute("data-gosx-studio-preview-patched", "fresh")`,
		`target.setAttribute("data-gosx-studio-preview-patched", "true")`,
		`}, 220)`,
		`frame.setAttribute("data-studio-preview-patched-count", String(targets.length))`,
		`if (!patch.field) {`,
		`if (reason !== "load-sync") setEditorPreviewStatus(form, "dirty", "Live preview pending", reason || "patch")`,
		`emitEditorPreview(form, "gosxstudio:preview-patch", patch)`,
		`if (!frame.contentWindow || !frame.getAttribute("src")) return`,
		`frame.contentWindow.postMessage(patch, new URL(frame.getAttribute("src"), window.location.href).origin)`,
		`frame.contentWindow.postMessage(patch, window.location.origin)`,
		`return { handled: true, count: count }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview patch transport fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-patch-apply")
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-patch-post")
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview patch control events")
	}
	if strings.Contains(body, `doc.addEventListener("gosxstudio:editor-preview-patch-build-post"`) {
		t.Fatalf("IslandRuntimeJS() should expose postPatch helper instead of patch-build-post listener")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewFrameSync(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewFieldPatch(field)`,
		`if (!field || !field.name || field.disabled) return null`,
		`if (field.name === "csrf_token" || type === "button" || type === "submit" || type === "reset" || type === "file") return null`,
		`source: field.getAttribute("data-studio-field-source") || field.getAttribute("data-editor-source") || field.name`,
		`editable: field.getAttribute("data-studio-field-editable") || ""`,
		`value: type === "checkbox" || type === "radio" ? (field.checked ? (field.value || "on") : "") : (field.value || "")`,
		`checked: !!field.checked`,
		`tag: String(field.tagName || "").toLowerCase()`,
		`function syncEditorPreviewFrame(form, frame, reason, shouldTransport)`,
		`if (!form || !editorPreviewFrameDocument(frame)) return { handled: false, count: 0 }`,
		`reason = reason || "sync"`,
		`queryAll(form, "[data-studio-field-source], [data-editor-source], input[name], textarea[name], select[name]").forEach(function (field)`,
		`if (typeof shouldTransport === "function" && shouldTransport(field, reason) === false) return`,
		`type: "gosxstudio:preview-patch"`,
		`source: "gosx-studio"`,
		`detail: {}`,
		`field: editorPreviewFieldPatch(field)`,
		`count += applyEditorPreviewPatch(frame, patch).count || 0`,
		`if (count) emitEditorPreview(form, "gosxstudio:preview-sync", { count: count, reason: reason })`,
		`return { handled: true, count: count }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview frame sync fragment %q", fragment)
		}
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-frame-sync`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview frame-sync events")
	}
	if strings.Contains(body, `doc.addEventListener("gosxstudio:editor-preview-frame-sync"`) {
		t.Fatalf("IslandRuntimeJS() should expose local frame sync through bindFrames instead of frame-sync listener")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewTargetLookup(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewPatchTargets(frame, patch)`,
		`var frameDoc = editorPreviewFrameDocument(frame)`,
		`return queryAll(frameDoc, editorPreviewPatchSelector(source));`,
		`function editorPreviewPatchSelector(source)`,
		`'[data-studio-field="' + source + '"]'`,
		`'[data-editor-preview="' + source + '"]'`,
		`'[data-studio-field-source="' + source + '"]'`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview target lookup fragment %q", fragment)
		}
	}
	if strings.Contains(body, `doc.addEventListener("gosxstudio:editor-preview-targets-resolve"`) {
		t.Fatalf("IslandRuntimeJS() should use local editor preview target lookup helpers instead of targets-resolve listener")
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-targets-resolve`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview target lookup events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewSelectableNodeResolution(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewSelectableNode(target)`,
		`if (!target || !target.closest) return null`,
		`var selectable = target.closest("[data-studio-field], [data-editor-preview], [data-studio-field-source], [data-studio-block-key], [data-studio-node-id]")`,
		`var pageOnlyAncestor = selectable !== target && selectable.hasAttribute("data-studio-page-id")`,
		`return target.matches && target.matches("[data-studio-page-id]") ? target : null`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview selectable-node resolver fragment %q", fragment)
		}
	}
	if strings.Contains(body, `doc.addEventListener("gosxstudio:editor-preview-selectable-node-resolve"`) {
		t.Fatalf("IslandRuntimeJS() should use local selectable-node resolution instead of selectable-node-resolve listener")
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selectable-node-resolve`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview selectable-node resolver events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewEventPolicy(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewPointerEventEligible(event)`,
		`if (!event || event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return false`,
		`if (event.button > 0) return false`,
		`return true`,
		`function editorPreviewKeyboardIntent(event)`,
		`if (!event || event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return null`,
		`event.target.closest ? event.target.closest("input, textarea, select, button, a[href], [contenteditable='true'], [contenteditable='plaintext-only']") : null`,
		`if (focusedControl) return null`,
		`if (event.key === "[") return { action: "prev-field", reason: "keyboard-prev-field" }`,
		`if (event.key === "]") return { action: "next-field", reason: "keyboard-next-field" }`,
		`if (event.key === "F2") return { action: "inline-text", reason: "keyboard-f2" }`,
		`if (event.key === "Enter") return { action: "inline-text", reason: "keyboard-enter" }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview event-policy fragment %q", fragment)
		}
	}
	for _, eventName := range []string{
		`gosxstudio:editor-preview-pointer-event-eligible`,
		`gosxstudio:editor-preview-key-intent`,
	} {
		if strings.Contains(body, `doc.addEventListener("`+eventName+`"`) {
			t.Fatalf("IslandRuntimeJS() should use local event policy helpers instead of %s listener", eventName)
		}
		if strings.Contains(body, `dispatchEvent(new CustomEvent("`+eventName) {
			t.Fatalf("IslandRuntimeJS() must not recursively dispatch %s events", eventName)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDocumentBinding(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function bindEditorPreviewDocument(form, frame, host)`,
		`var frameDoc = editorPreviewFrameDocument(frame)`,
		`if (!frame || !frameDoc)`,
		`setEditorPreviewDiagnostic(form, "same-origin-unavailable"`,
		`if (editorPreviewBoundDocument(frame) === frameDoc) return { handled: false, bound: false }`,
		`setEditorPreviewBoundDocument(frame, frameDoc)`,
		`frameDoc.documentElement.setAttribute("data-gosx-studio-preview-selectable", "true")`,
		`function editorPreviewBoundDocument(frame)`,
		`return editorPreviewDocumentBindings ? editorPreviewDocumentBindings.get(frame) : frame.__gosxStudioPreviewRuntimeDocument`,
		`function setEditorPreviewBoundDocument(frame, frameDoc)`,
		`editorPreviewDocumentBindings.set(frame, frameDoc)`,
		`frame.__gosxStudioPreviewRuntimeDocument = frameDoc`,
		`var editorPreviewDocumentBindings = typeof WeakMap !== "undefined" ? new WeakMap() : null`,
		`function editorPreviewFrameTask(fn)`,
		`window.requestAnimationFrame(function ()`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview document binding fragment %q", fragment)
		}
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-document-bind`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview document-bind events")
	}
	if strings.Contains(body, `doc.addEventListener("gosxstudio:editor-preview-document-bind"`) {
		t.Fatalf("IslandRuntimeJS() should bind preview documents through local frame lifecycle helpers instead of document-bind listener")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewFrameLifecycleBinding(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`window.__gosx_preview_runtime_island_bindFrames = function (form, host)`,
		`return bindEditorPreviewFrames(form, host);`,
		`function bindEditorPreviewFrames(form, host)`,
		`var frames = editorPreviewFrames(form)`,
		`if (frame.dataset && frame.dataset.gosxStudioPreviewBound === "true") return`,
		`if (frame.dataset) frame.dataset.gosxStudioPreviewBound = "true"`,
		`if (!frame.getAttribute("data-studio-preview-src"))`,
		`frame.setAttribute("data-studio-preview-src", frame.getAttribute("src") || "")`,
		`frame.addEventListener("load", function ()`,
		`frame.addEventListener("error", function ()`,
		`bindEditorPreviewFrameDocument(form, frame, host)`,
		`bound += 1`,
		`return { handled: true, frames: frames.length, bound: bound }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview frame lifecycle fragment %q", fragment)
		}
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-frames-bind`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview frames-bind events")
	}
	if strings.Contains(body, `doc.addEventListener("gosxstudio:editor-preview-frames-bind"`) {
		t.Fatalf("IslandRuntimeJS() should expose bindFrames helper instead of frames-bind listener")
	}
}

func TestPreviewRuntimeIslandJSPreviewFrameLifecycleHandlersRunLocally(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function bindEditorPreviewFrameDocument(form, frame, host)`,
		`return bindEditorPreviewDocument(form, frame, host)`,
		`setEditorPreviewStatus(form, "ready", "", "load")`,
		`syncEditorPreviewFrame(form, frame, "load", typeof host.shouldTransportPreviewPatch === "function" ? host.shouldTransportPreviewPatch : null)`,
		`var route = editorPreviewURL(frame) || frame.getAttribute("src") || ""`,
		`postEditorPreviewPatch(form, "load-sync", editorPreviewPatchEnvelope("load-sync", { route: route }, null))`,
		`setEditorPreviewStatus(form, "error", "Preview failed", "error")`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview frame lifecycle local fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		`editorPreviewHostCall(host, "bindPreviewDocument", null, frame)`,
		`editorPreviewHostCall(host, "setPreviewStatus", null, "ready", "Ready", "load")`,
		`editorPreviewHostCall(host, "syncPreviewFrame", null, frame, "load")`,
		`editorPreviewHostCall(host, "postPreviewLoadSyncPatch", null, route)`,
		`editorPreviewHostCall(host, "setPreviewStatus", null, "error", "Preview failed", "error")`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() should not hand editor preview frame lifecycle to host first; found %q", forbidden)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDocumentListeners(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`frameDoc.addEventListener("click", function (event)`,
		`frameDoc.addEventListener("dblclick", function (event)`,
		`frameDoc.addEventListener("focusin", function (event)`,
		`frameDoc.addEventListener("input", function (event)`,
		`frameDoc.addEventListener("keydown", function (event)`,
		`frameDoc.addEventListener("paste", function (event)`,
		`frameDoc.addEventListener("blur", function (event)`,
		`frameDoc.addEventListener("scroll", repositionDock, true)`,
		`frame.contentWindow.addEventListener("resize", repositionDock)`,
		`if (editorPreviewInlineEditContains(frame, event.target)) return`,
		`if (!editorPreviewPointerEventEligible(event)) return`,
		`var target = editorPreviewSelectableNode(event.target)`,
		`var intent = editorPreviewKeyboardIntent(event)`,
		`if (frame.__gosxStudioInlineEdit) return`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview document listener fragment %q", fragment)
		}
	}
	if strings.Count(body, `frameDoc.addEventListener("keydown", function (event)`) < 2 {
		t.Fatalf("IslandRuntimeJS() should bind both inline-text and preview-key-intent keydown paths")
	}
}

func TestPreviewRuntimeIslandJSHandsPreviewDocumentEventsToHost(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewHostCall(host, name, fallback)`,
		`return host[name].apply(host, Array.prototype.slice.call(arguments, 3))`,
		`editorPreviewHostCall(host, "previewSelectionDetail", {}, target)`,
		`editorPreviewHostCall(host, "applyPreviewSelection", false, frame, target, detail, { reveal: true, reason: "click" })`,
		`editorPreviewHostCall(host, "applyPreviewSelection", false, frame, target, detail, { reveal: true, reason: "double-click" }) && editorPreviewHostCall(host, "startInlineTextFromDetail", false, frame, detail, "double-click")`,
		`editorPreviewHostCall(host, "applyPreviewSelection", false, frame, target, detail, { reveal: false, reason: "focus" })`,
		`editorPreviewHostCall(host, "handleInlineTextInput", false, frame, event)`,
		`editorPreviewHostCall(host, "handleInlineTextKeyEvent", false, frame, event)`,
		`var result = runEditorPreviewFieldNavigation({`,
		`direction: intent.action === "next-field" ? 1 : -1,`,
		`reason: intent.reason,`,
		`host: host`,
		`if (result && result.navigated)`,
		`startEditorPreviewInlineTextFromSelection(form, frame, host, intent.reason)`,
		`editorPreviewHostCall(host, "handleInlineTextPaste", false, frame, event)`,
		`editorPreviewHostCall(host, "handleInlineTextBlur", false, frame, event)`,
		`syncEditorPreviewDockPositionForFrame(frame)`,
		`event.preventDefault()`,
		`event.stopPropagation()`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview document host handoff fragment %q", fragment)
		}
	}
	if strings.Contains(body, `editorPreviewHostCall(host, "navigatePreviewField"`) {
		t.Fatalf("IslandRuntimeJS() should run preview field navigation locally, found old host callback")
	}
	if strings.Contains(body, `editorPreviewHostCall(host, "startInlineTextFromSelection"`) {
		t.Fatalf("IslandRuntimeJS() should extract selected inline-text detail locally, found old host callback")
	}
	if strings.Contains(body, `editorPreviewHostCall(host, "updatePreviewDockPosition"`) {
		t.Fatalf("IslandRuntimeJS() should sync preview dock positions locally, found old host callback")
	}
}

func TestPreviewRuntimeIslandJSStartsKeyboardInlineTextFromSelectedDockDetail(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	helper := islandJSFunctionBody(t, body, "startEditorPreviewInlineTextFromSelection")
	for _, fragment := range []string{
		`function startEditorPreviewInlineTextFromSelection(form, frame, host, reason)`,
		`var dock = frame && frame.__gosxStudioPreviewDock`,
		`if (!dock || dock.hidden) return false`,
		`var detail = editorPreviewDockDetail(form, dock)`,
		`return editorPreviewHostCall(host, "startInlineTextFromDetail", false, frame, detail, reason || "keyboard")`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing selected dock inline-text fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		`startInlineTextFromSelection`,
		`event.preventDefault()`,
		`event.stopPropagation()`,
	} {
		if strings.Contains(helper, forbidden) {
			t.Fatalf("selected dock inline-text helper should only extract detail and call host; found %q in:\n%s", forbidden, helper)
		}
	}
	keyboardBody := islandJSFunctionBody(t, body, "bindEditorPreviewDocument")
	for _, fragment := range []string{
		`if (intent.action !== "inline-text") return`,
		`if (startEditorPreviewInlineTextFromSelection(form, frame, host, intent.reason))`,
		`event.preventDefault()`,
		`event.stopPropagation()`,
	} {
		if !strings.Contains(keyboardBody, fragment) {
			t.Fatalf("keyboard inline-text handler should gate event cancellation on helper success, missing %q in:\n%s", fragment, keyboardBody)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewFieldMapMarkers(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewFieldKey(target)`,
		`target.getAttribute("data-studio-field") || target.getAttribute("data-editor-preview") || target.getAttribute("data-studio-field-source")`,
		`function clearEditorPreviewFieldMap(frame)`,
		`var frameDoc = editorPreviewFrameDocument(frame)`,
		`queryAll(frameDoc, "[data-gosx-studio-preview-field-scope], [data-gosx-studio-preview-field-current]").forEach(function (target)`,
		`target.removeAttribute("data-gosx-studio-preview-field-scope")`,
		`target.removeAttribute("data-gosx-studio-preview-field-current")`,
		`target.removeAttribute("data-gosx-studio-preview-field-position")`,
		`target.removeAttribute("data-gosx-studio-preview-field-total")`,
		`function syncEditorPreviewFieldMap(frame, fields, current, count)`,
		`var result = clearEditorPreviewFieldMap(frame)`,
		`fields = Array.isArray(fields) ? fields : []`,
		`candidate.setAttribute("data-gosx-studio-preview-field-scope", "true")`,
		`candidate.setAttribute("data-gosx-studio-preview-field-position", String(index + 1))`,
		`candidate.setAttribute("data-gosx-studio-preview-field-total", String(count))`,
		`candidate.setAttribute("data-gosx-studio-preview-field-current", "true")`,
		`return { handled: true, count: fields.length, cleared: result.count || 0 }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview field-map marker fragment %q", fragment)
		}
	}
	for _, eventName := range []string{
		`gosxstudio:editor-preview-field-map-clear`,
		`gosxstudio:editor-preview-field-map-sync`,
	} {
		if strings.Contains(body, `doc.addEventListener("`+eventName+`"`) {
			t.Fatalf("IslandRuntimeJS() should use local field-map helpers instead of %s listener", eventName)
		}
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-field-map`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview field-map control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewFieldNavigationState(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewFieldKeyForTarget(target)`,
		`return target.getAttribute("data-studio-field") || target.getAttribute("data-editor-preview") || target.getAttribute("data-studio-field-source") || ""`,
		`function editorPreviewFieldNavigationScope(frame, target, detail)`,
		`var frameDoc = editorPreviewFrameDocument(frame)`,
		`var targetBlock = target.closest("[data-studio-block-key], [data-studio-node-id]")`,
		`var selector = '[data-studio-block-key="' + attrValue(blockKey) + '"], [data-studio-node-id="' + attrValue(blockKey) + '"]'`,
		`return frameDoc.querySelector(selector) || frameDoc.body || frameDoc.documentElement`,
		`function editorPreviewFieldNodesForSelection(frame, target, detail)`,
		`var seen = {}`,
		`queryAll(scope, "[data-studio-field], [data-editor-preview], [data-studio-field-source]").filter(function (candidate)`,
		`var key = editorPreviewFieldKeyForTarget(candidate)`,
		`function editorPreviewFieldNavigationState(frame, target, detail)`,
		`if (!frame || !scope) return { handled: false, fields: [], count: 0, index: -1, current: "" }`,
		`var current = detail && detail.field ? detail.field : editorPreviewFieldKeyForTarget(target)`,
		`if (index < 0 && editorPreviewFieldKeyForTarget(candidate) === current) index = candidateIndex`,
		`handled: true`,
		`count: fields.length`,
		`current: current`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview field-navigation state fragment %q", fragment)
		}
	}
	if strings.Contains(body, `doc.addEventListener("gosxstudio:editor-preview-field-navigation-state"`) {
		t.Fatalf("IslandRuntimeJS() should use local field-navigation state helpers instead of field-navigation-state listener")
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-field-navigation-state`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview field-navigation state control events")
	}
}

func TestPreviewRuntimeIslandJSRunsEditorPreviewFieldNavigationLocally(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function runEditorPreviewFieldNavigation(detail, event)`,
		`var frame = detail.frame`,
		`var direction = Number(detail.direction) || 0`,
		`var reason = detail.reason || "field-navigation"`,
		`var host = detail.host || {}`,
		`var dock = frame && frame.__gosxStudioPreviewDock`,
		`var target = frame && frame.__gosxStudioPreviewDockTarget`,
		`if (!dock || dock.hidden || !target) return { handled: false, navigated: false }`,
		`editorPreviewHostCall(host, "finishInlineTextEdit", false, frame, true, "field-navigation")`,
		`var selection = editorPreviewHostCall(host, "previewSelectionDetail", {}, target) || {}`,
		`var state = editorPreviewFieldNavigationState(frame, target, selection)`,
		`if (!state.count) return { handled: false, navigated: false, state: state }`,
		`var currentIndex = state.index >= 0 ? state.index : (direction > 0 ? -1 : 0)`,
		`var nextIndex = (currentIndex + direction + state.count) % state.count`,
		`var nextTarget = state.fields[nextIndex]`,
		`var nextDetail = editorPreviewHostCall(host, "previewSelectionDetail", {}, nextTarget) || {}`,
		`if (!nextTarget || !nextDetail.field) return { handled: false, navigated: false, state: state }`,
		`editorPreviewHostCall(host, "applyPreviewSelection", false, frame, nextTarget, nextDetail, { reveal: true, reason: reason })`,
		`direction: direction > 0 ? "next" : "previous"`,
		`fieldIndex: nextIndex + 1`,
		`fieldCount: state.count`,
		`reason: reason`,
		`editorPreviewHostCall(host, "dispatchPreviewFieldNavigation", null, nextDetail, navigation)`,
		`return { handled: true, navigated: true, detail: nextDetail, navigation: navigation, state: state }`,
		`var result = runEditorPreviewFieldNavigation({`,
		`direction: intent.action === "next-field" ? 1 : -1,`,
		`reason: intent.reason,`,
		`var navigation = runEditorPreviewFieldNavigation({`,
		`direction: action === "next-field" ? 1 : -1`,
		`reason: "preview-dock"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing local editor preview field-navigation fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-field-navigation-run"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-field-navigation-run`,
		`setEditorPreviewResult(detail, runEditorPreviewFieldNavigation(detail, event))`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() must run editor preview field navigation locally without an external event boundary; found %q", forbidden)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewSelectionClearHelper(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`window.__gosx_preview_runtime_island_clearSelections = function (form, host, options)`,
		`return clearEditorPreviewSelections(form, host, options);`,
		`function clearEditorPreviewSelections(form, host, options)`,
		// #5 — Escape must cancel an in-progress inline edit, not commit it.
		// options.commit defaults to true (the "move to a new selection"
		// caller below still legitimately commits the prior edit); the
		// keyboard-Escape caller (hostruntime/assets/workbench_runtime.js)
		// passes { commit: false } so cancelling reverts instead.
		`var commit = options.commit !== false`,
		`var reason = commit ? "clear-selection" : "escape"`,
		`if (!form) return { handled: false, frames: 0, fieldMaps: 0, markers: 0, chrome: null, docks: null }`,
		`var frames = editorPreviewFrames(form)`,
		`editorPreviewHostCall(host, "finishInlineTextEdit", null, frame, commit, reason)`,
		`var fieldMap = clearEditorPreviewFieldMap(frame)`,
		`var marker = clearEditorPreviewSelectionMarker(frame)`,
		`fieldMaps += fieldMap && fieldMap.count ? fieldMap.count : 0`,
		`markers += marker && marker.count ? marker.count : 0`,
		`var chrome = clearEditorPreviewSelectionChrome(form)`,
		`var docks = hideEditorPreviewDocks(form)`,
		`return { handled: true, frames: frames.length, fieldMaps: fieldMaps, markers: markers, chrome: chrome, docks: docks }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview selection clear-sync fragment %q", fragment)
		}
	}

	clearBody := islandJSFunctionBody(t, body, "clearEditorPreviewSelections")
	for _, ordered := range [][]string{
		{
			`var commit = options.commit !== false`,
			`var frames = editorPreviewFrames(form)`,
			`editorPreviewHostCall(host, "finishInlineTextEdit", null, frame, commit, reason)`,
			`var fieldMap = clearEditorPreviewFieldMap(frame)`,
			`var marker = clearEditorPreviewSelectionMarker(frame)`,
			`var chrome = clearEditorPreviewSelectionChrome(form)`,
			`var docks = hideEditorPreviewDocks(form)`,
		},
	} {
		last := -1
		for _, fragment := range ordered {
			index := strings.Index(clearBody, fragment)
			if index < 0 {
				t.Fatalf("clearEditorPreviewSelections missing order fragment %q in:\n%s", fragment, clearBody)
			}
			if index < last {
				t.Fatalf("clearEditorPreviewSelections has out-of-order fragment %q in:\n%s", fragment, clearBody)
			}
			last = index
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-selection-clear-sync"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selection-clear-sync`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() must use clear selection helper without retired event boundary; found %q", forbidden)
		}
	}
}

// TestPreviewRuntimeIslandJSClearSelectionsCommitDefaultsTrue verifies the #5
// fix: clearEditorPreviewSelections defaults to committing an in-progress
// inline edit (the "moving to a new selection" caller,
// applyEditorPreviewSelectionSync, must keep committing the prior edit
// unchanged) and only cancels/reverts when a caller explicitly passes
// { commit: false } — the keyboard-Escape path.
func TestPreviewRuntimeIslandJSClearSelectionsCommitDefaultsTrue(t *testing.T) {
	body := string(IslandRuntimeJS())
	clearBody := islandJSFunctionBody(t, body, "clearEditorPreviewSelections")
	for _, fragment := range []string{
		`options = options || {}`,
		`var commit = options.commit !== false`,
	} {
		if !strings.Contains(clearBody, fragment) {
			t.Fatalf("clearEditorPreviewSelections missing commit-default fragment %q in:\n%s", fragment, clearBody)
		}
	}
	// The "apply a new selection" caller must NOT pass a third argument, so it
	// keeps the default (commit) behavior unchanged.
	if !strings.Contains(body, `var cleared = clearEditorPreviewSelections(form, host)`) {
		t.Fatalf("applyEditorPreviewSelectionSync must call clearEditorPreviewSelections(form, host) with no options override:\n%s", body)
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewSelectionApplyHelper(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`window.__gosx_preview_runtime_island_applySelection = function (detail)`,
		`return applyEditorPreviewSelectionSync(detail, null);`,
		`function applyEditorPreviewSelectionSync(detail, event)`,
		`var form = editorPreviewForm(detail, event)`,
		`var selection = detail.selection || detail.detail || {}`,
		`var options = detail.options || {}`,
		`var host = detail.host || {}`,
		`if (!selection.field && !selection.blockKey && !selection.nodeID && !selection.pageID)`,
		`var cleared = clearEditorPreviewSelections(form, host)`,
		`var selectedTargets = selection.field ? editorPreviewPatchTargets(frame, { field: { source: selection.field, name: selection.field } }) : []`,
		`if (!selectedTargets.length && target) selectedTargets = [target]`,
		`var marker = applyEditorPreviewSelectionMarker(frame, selectedTargets, true)`,
		`var applied = editorPreviewHostCall(host, "dispatchPreviewSelectionApply", null, selection, options)`,
		`if (!applied)`,
		`var chrome = applyEditorPreviewSelectionChrome(form, frame, selection.field || selection.blockKey || selection.nodeID || selection.pageID || "")`,
		`var dockSelection = editorPreviewDockSelection(selection, applied)`,
		`var dock = syncEditorPreviewDockSelection(form, frame, selectedTargets[0] || target, dockSelection, host)`,
		`return { handled: true, applied: applied, targets: selectedTargets, marker: marker, chrome: chrome, dock: dock, cleared: cleared }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview selection apply-sync fragment %q", fragment)
		}
	}

	applyBody := islandJSFunctionBody(t, body, "applyEditorPreviewSelectionSync")
	for _, ordered := range [][]string{
		{
			`var form = editorPreviewForm(detail, event)`,
			`if (!selection.field && !selection.blockKey && !selection.nodeID && !selection.pageID)`,
			`var cleared = clearEditorPreviewSelections(form, host)`,
			`var selectedTargets = selection.field ? editorPreviewPatchTargets(frame, { field: { source: selection.field, name: selection.field } }) : []`,
			`if (!selectedTargets.length && target) selectedTargets = [target]`,
			`var marker = applyEditorPreviewSelectionMarker(frame, selectedTargets, true)`,
			`var applied = editorPreviewHostCall(host, "dispatchPreviewSelectionApply", null, selection, options)`,
			`if (!applied)`,
			`var chrome = applyEditorPreviewSelectionChrome(form, frame, selection.field || selection.blockKey || selection.nodeID || selection.pageID || "")`,
			`var dockSelection = editorPreviewDockSelection(selection, applied)`,
			`var dock = syncEditorPreviewDockSelection(form, frame, selectedTargets[0] || target, dockSelection, host)`,
		},
	} {
		last := -1
		for _, fragment := range ordered {
			index := strings.Index(applyBody, fragment)
			if index < 0 {
				t.Fatalf("applyEditorPreviewSelectionSync missing order fragment %q in:\n%s", fragment, applyBody)
			}
			if index < last {
				t.Fatalf("applyEditorPreviewSelectionSync has out-of-order fragment %q in:\n%s", fragment, applyBody)
			}
			last = index
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-selection-apply-sync"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selection-apply-sync`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() must use apply selection helper without retired event boundary; found %q", forbidden)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewSelectionMarkers(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function clearEditorPreviewSelectionMarker(frame)`,
		`var frameDoc = editorPreviewFrameDocument(frame)`,
		`queryAll(frameDoc, "[data-gosx-studio-preview-selected]").forEach(function (target)`,
		`target.removeAttribute("data-gosx-studio-preview-selected")`,
		`return { handled: true, count: count }`,
		`function applyEditorPreviewSelectionMarker(frame, targets, clear)`,
		`var result = clear === false ? { count: 0 } : clearEditorPreviewSelectionMarker(frame)`,
		`targets = Array.isArray(targets) ? targets : []`,
		`target.setAttribute("data-gosx-studio-preview-selected", "true")`,
		`return { handled: true, count: targets.length, cleared: result.count || 0 }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview selected-marker fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-selection-marker-clear")
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-selection-marker-apply")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewSelectionChrome(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function clearEditorPreviewSelectionChrome(form)`,
		`var shells = editorPreviewShells(form)`,
		`var frames = editorPreviewFrames(form)`,
		`shell.removeAttribute("data-gosx-studio-preview-selection")`,
		`frame.removeAttribute("data-studio-preview-selection")`,
		`return { handled: true, shells: shells.length, frames: frames.length }`,
		`function applyEditorPreviewSelectionChrome(form, frame, selection)`,
		`shell.setAttribute("data-gosx-studio-preview-selection", selection)`,
		`frame.setAttribute("data-studio-preview-selection", selection)`,
		`return { handled: true, shells: shells.length, frames: 1 }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview selection chrome fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-selection-chrome-clear")
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-selection-chrome-apply")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockHide(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function hideEditorPreviewDocks(form)`,
		`var docks = queryAll(form, "[data-gosx-studio-preview-dock]")`,
		`dock.hidden = true`,
		`dock.removeAttribute("data-gosx-studio-preview-field")`,
		`dock.removeAttribute("data-gosx-studio-preview-block")`,
		`dock.removeAttribute("data-gosx-studio-preview-action-label")`,
		`dock.removeAttribute("data-gosx-studio-preview-action-href")`,
		`dock.removeAttribute("data-gosx-studio-preview-action-formaction")`,
		`dock.removeAttribute("data-gosx-studio-preview-block-label")`,
		`dock.removeAttribute("data-gosx-studio-preview-field-count")`,
		`dock.removeAttribute("data-gosx-studio-preview-field-index")`,
		`return { handled: true, count: docks.length }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock hide/reset fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-hide")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockNormalization(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function normalizeEditorPreviewDock(dock)`,
		`if (!dock || !dock.setAttribute || !dock.querySelector) return { handled: false, commands: 0 }`,
		`dock.setAttribute("data-gosx-studio-preview-dock", "true")`,
		`var title = dock.querySelector("[data-studio-preview-dock-title]")`,
		`title.setAttribute("data-gosx-studio-preview-dock-label", "true")`,
		`var fieldCount = dock.querySelector("[data-studio-preview-field-count]")`,
		`fieldCount.setAttribute("data-gosx-studio-preview-field-meter", "true")`,
		`var actions = queryAll(dock, "[data-studio-preview-action]")`,
		`button.setAttribute("data-gosx-studio-preview-command", button.getAttribute("data-studio-preview-action") || "")`,
		`return { handled: true, commands: actions.length }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock normalization fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-normalize")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockBinding(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function bindEditorPreviewDock(form, frame, dock, host)`,
		`var normalize = normalizeEditorPreviewDock(dock)`,
		`if (!dock || !dock.addEventListener)`,
		`return { handled: false, bound: false, commands: normalize.commands || 0 }`,
		`dock.__gosxStudioPreviewDockFrame = frame`,
		`dock.__gosxStudioPreviewDockHost = host`,
		`if (!dock.__gosxStudioPreviewDockHandler)`,
		`dock.__gosxStudioPreviewDockHandler = true`,
		`dock.addEventListener("click", function (event)`,
		`event.target.closest ? event.target.closest("[data-gosx-studio-preview-command], [data-studio-preview-action]")`,
		`if (!button || !dock.contains(button)) return`,
		`event.preventDefault()`,
		`var command = button.getAttribute("data-gosx-studio-preview-command") || button.getAttribute("data-studio-preview-action") || ""`,
		`runEditorPreviewDockAction({ form: form, frame: dock.__gosxStudioPreviewDockFrame || frame, action: command, host: actionHost }, event)`,
		`return { handled: true, bound: true, commands: normalize.commands || 0 }`,
		`return { handled: true, bound: false, commands: normalize.commands || 0 }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock binding fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-bind")
	if strings.Contains(body, `runPreviewDockAction`) {
		t.Fatalf("IslandRuntimeJS() dock binding should not call Workbench runPreviewDockAction")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockActionRunHelper(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`window.__gosx_preview_runtime_island_runDockAction = function (detail)`,
		`return runEditorPreviewDockAction(detail, null);`,
		`function runEditorPreviewDockAction(detail, event)`,
		`var form = editorPreviewForm(detail, event)`,
		`var frame = detail.frame`,
		`var action = detail.action || ""`,
		`var host = detail.host || {}`,
		`var dock = frame && frame.__gosxStudioPreviewDock`,
		`if (!dock || !action) return { handled: false, action: action, detail: {}, result: false }`,
		`var dockDetail = editorPreviewDockDetail(form, dock)`,
		`if (action === "clear")`,
		`editorPreviewHostCall(host, "clearPreviewSelections", null)`,
		`editorPreviewHostCall(host, "dispatchPreviewSelectionClear", null, "preview-dock")`,
		`editorPreviewHostCall(host, "emitPreviewDockAction", null, action, dockDetail)`,
		`if (action === "content" || action === "style")`,
		`var dockIntent = editorPreviewHostCall(host, "dispatchPreviewDockActionResolve", null, action, dockDetail) || { mode: action, reveal: action === "content" && !!dockDetail.field }`,
		`editorPreviewHostCall(host, "setMode", null, dockIntent.mode, { scroll: true, reason: "preview-dock" })`,
		`editorPreviewHostCall(host, "revealPreviewField", null, dockDetail, "preview-dock")`,
		`if (action === "prev-field" || action === "next-field")`,
		`var navigation = runEditorPreviewFieldNavigation({`,
		`direction: action === "next-field" ? 1 : -1`,
		`reason: "preview-dock"`,
		`dockDetail = editorPreviewDockDetail(form, dock)`,
		`return { handled: true, action: action, detail: dockDetail, result: false, navigation: navigation }`,
		`if (action === "field-action")`,
		`var intent = editorPreviewHostCall(host, "dispatchPreviewFieldActionResolve", null, dockDetail) || { reveal: !!dockDetail.field }`,
		`editorPreviewHostCall(host, "startInlineTextFromDetail", false, frame, dockDetail, "preview-dock")`,
		`editorPreviewHostCall(host, "submitPreviewFieldAction", false, dockDetail)`,
		`editorPreviewHostCall(host, "navigateToHref", null, intent.href || "")`,
		`window.location.href = intent.href || ""`,
		`return { handled: true, action: action, detail: dockDetail, result: true }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock action-run fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-action-run"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-action-run`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() must use dock action helper without retired event boundary; found %q", forbidden)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockShowContent(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewDockKindLabel(selection)`,
		`if (!selection || !selection.field) return "Block"`,
		`if (editable === "media" || editable === "image") return "Media field"`,
		`if (editable === "source") return "Source field"`,
		`if (editable === "flow") return "Flow field"`,
		`if (editable === "url" || editable === "link") return "Link field"`,
		`if (editable === "text") return "Text field"`,
		`function editorPreviewDockSelection(selection, applied)`,
		`selection = selection || {};`,
		`applied = applied || {};`,
		`field: applied.field || selection.field || ""`,
		`editable: applied.editable || selection.editable || ""`,
		`label: applied.label || selection.label || ""`,
		`blockLabel: applied.blockLabel || selection.blockLabel || ""`,
		`action: applied.action || selection.action || ""`,
		`actionHref: applied.actionHref || selection.actionHref || ""`,
		`actionFormAction: applied.actionFormAction || selection.actionFormAction || ""`,
		`blockKey: applied.blockKey || selection.blockKey || ""`,
		`nodeID: applied.nodeID || selection.nodeID || ""`,
		`function syncEditorPreviewDock(dock, selection)`,
		`selection = editorPreviewDockSelection(selection);`,
		`dock.hidden = false`,
		`dock.setAttribute("data-gosx-studio-preview-field", selection.field || "")`,
		`dock.setAttribute("data-gosx-studio-preview-block", selection.blockKey || selection.nodeID || "")`,
		`dock.setAttribute("data-gosx-studio-preview-block-label", selection.blockLabel || "")`,
		`dock.setAttribute("data-gosx-studio-preview-action-label", selection.action || "")`,
		`dock.setAttribute("data-gosx-studio-preview-action-href", selection.actionHref || "")`,
		`dock.setAttribute("data-gosx-studio-preview-action-formaction", selection.actionFormAction || "")`,
		`var label = dock.querySelector("[data-gosx-studio-preview-dock-label]")`,
		`if (label) label.textContent = selection.label || selection.field || selection.blockKey || "Preview selection"`,
		`breadcrumb.hidden = !selection.blockLabel || !selection.field`,
		`breadcrumb.textContent = selection.blockLabel && selection.field ? selection.blockLabel + " / " + (selection.label || selection.field) : ""`,
		`if (kind) kind.textContent = editorPreviewDockKindLabel(selection)`,
		`action.textContent = selection.action || (selection.editable === "text" ? "Edit text" : selection.editable === "media" || selection.editable === "image" ? "Media" : selection.editable === "flow" ? "Flow" : selection.editable === "source" ? "Source" : "Open")`,
		`action.disabled = !selection.field && !selection.action && !selection.actionHref && !selection.actionFormAction`,
		`return { handled: true }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock show/content fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-selection-resolve")
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-sync")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockSelectionSync(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewDockForFrame(frame)`,
		`var shell = editorPreviewShellForFrame(frame)`,
		`var dock = shell.querySelector("[data-studio-preview-dock], [data-gosx-studio-preview-dock]")`,
		`dock = form && form.querySelector("[data-studio-preview-dock], [data-gosx-studio-preview-dock]")`,
		`dock: dock`,
		`shell: shell`,
		`function syncEditorPreviewDockSelection(form, frame, target, selection, host)`,
		`var dockLookup = editorPreviewDockForFrame(frame)`,
		`var dock = dockLookup.dock`,
		`var shell = dockLookup.shell`,
		`if (!dock || !target) return { handled: false, dock: dock || null, state: null, bound: false, positioned: false }`,
		`var normalizedSelection = editorPreviewDockSelection(selection)`,
		`var bindResult = bindEditorPreviewDock(form, frame, dock, host)`,
		`frame.__gosxStudioPreviewDock = dock`,
		`frame.__gosxStudioPreviewDockTarget = target`,
		`syncEditorPreviewDock(dock, normalizedSelection)`,
		`var state = editorPreviewFieldNavigationState(frame, target, normalizedSelection)`,
		`syncEditorPreviewDockFieldNavigation(dock, state.count, state.index)`,
		`syncEditorPreviewFieldMap(frame, state.fields, state.current, state.count)`,
		`var position = syncEditorPreviewDockPosition(frame, dock, target, shell)`,
		`return { handled: true, dock: dock, state: state, bound: !!(bindResult && bindResult.bound), positioned: !!(position && position.handled) }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock selection sync fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-selection-sync")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockDetailExtraction(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function compactText(value)`,
		`return String(value || "").replace(/\s+/g, " ").trim()`,
		`function editorPreviewDockDetail(form, dock)`,
		`if (!dock || !dock.getAttribute || !dock.querySelector)`,
		`return { field: "", blockKey: "", blockLabel: "", label: "", action: "", actionHref: "", actionFormAction: "", editable: "" }`,
		`var label = dock.querySelector("[data-gosx-studio-preview-dock-label]")`,
		`field: dock.getAttribute("data-gosx-studio-preview-field") || ""`,
		`blockKey: dock.getAttribute("data-gosx-studio-preview-block") || ""`,
		`blockLabel: dock.getAttribute("data-gosx-studio-preview-block-label") || ""`,
		`label: compactText(label && label.textContent)`,
		`action: dock.getAttribute("data-gosx-studio-preview-action-label") || ""`,
		`actionHref: dock.getAttribute("data-gosx-studio-preview-action-href") || ""`,
		`actionFormAction: dock.getAttribute("data-gosx-studio-preview-action-formaction") || ""`,
		`editable: form ? form.getAttribute("data-studio-field-editable") || "" : ""`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock detail extraction fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-detail-resolve")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockFieldNavigationUI(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function syncEditorPreviewDockFieldNavigation(dock, count, index)`,
		`if (!dock || !dock.setAttribute) return { handled: false, count: 0, index: -1 }`,
		`dock.setAttribute("data-gosx-studio-preview-field-count", String(count))`,
		`dock.setAttribute("data-gosx-studio-preview-field-index", index >= 0 ? String(index + 1) : "")`,
		`var meter = dock.querySelector("[data-gosx-studio-preview-field-meter]")`,
		`meter.hidden = count < 2 || index < 0`,
		`meter.textContent = count > 1 && index >= 0 ? "Field " + (index + 1) + " of " + count : ""`,
		`["prev-field", "next-field"].forEach(function (action)`,
		`var button = dock.querySelector('[data-gosx-studio-preview-command="' + action + '"]')`,
		`button.hidden = count === 0`,
		`button.disabled = count < 2`,
		`button.setAttribute("aria-label", (action === "prev-field" ? "Previous" : "Next") + " editable field")`,
		`return { handled: true, count: count, index: index }`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock field-navigation UI fragment %q", fragment)
		}
	}
	assertRetiredEditorPreviewRuntimeEventAbsent(t, body, "gosxstudio:editor-preview-dock-field-navigation-sync")
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockPosition(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function editorPreviewShellForFrame(frame)`,
		`return frame && frame.closest ? frame.closest("[data-gosx-studio-preview], [data-studio-preview-shell], [data-studio-canvas]") : null`,
		`function editorPreviewClamp(value, min, max)`,
		`return Math.min(max, Math.max(min, value))`,
		`function syncEditorPreviewDockPosition(frame, dock, target, shell)`,
		`shell = shell || editorPreviewShellForFrame(frame)`,
		`if (!frame || !dock || !target || !shell || dock.hidden) return { handled: false, placement: "" }`,
		`var frameRect = frame.getBoundingClientRect()`,
		`var shellRect = shell.getBoundingClientRect()`,
		`var targetRect = target.getBoundingClientRect()`,
		`var left = frameRect.left - shellRect.left + targetRect.left + (targetRect.width / 2)`,
		`var top = frameRect.top - shellRect.top + targetRect.top`,
		`var maxLeft = Math.max(8, shellRect.width - 8)`,
		`left = editorPreviewClamp(left, 8, maxLeft)`,
		`var placement = top < 52 ? "bottom" : "top"`,
		`dock.setAttribute("data-gosx-studio-preview-dock-placement", placement)`,
		`dock.style.top = Math.round(placement === "bottom" ? frameRect.top - shellRect.top + targetRect.bottom + 10 : top - 10) + "px"`,
		`dock.style.left = Math.round(left) + "px"`,
		`return { handled: true, placement: placement }`,
		`function syncEditorPreviewDockPositionForFrame(frame)`,
		`return syncEditorPreviewDockPosition(frame, dock, target)`,
		`window.__gosx_preview_runtime_island_syncDockPosition = syncEditorPreviewDockPositionForFrame`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock position fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-position-sync"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-position-sync`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() must not keep editor preview dock position control event boundary %q", forbidden)
		}
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockPositionsSync(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`function syncEditorPreviewDockPositions(form)`,
		`if (!form) return { handled: false, count: 0 }`,
		`var count = 0`,
		`editorPreviewFrames(form).forEach(function (frame)`,
		`var dock = frame && frame.__gosxStudioPreviewDock`,
		`var target = frame && frame.__gosxStudioPreviewDockTarget`,
		`if (!dock || !target || dock.hidden) return`,
		`var result = syncEditorPreviewDockPosition(frame, dock, target)`,
		`if (result && result.handled) count += 1`,
		`return { handled: true, count: count }`,
		`window.addEventListener("resize", function ()`,
		`editorPreviewWorkbenchForms(doc).forEach(function (form)`,
		`syncEditorPreviewDockPositions(form)`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock positions sync fragment %q", fragment)
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-positions-sync"`,
		`dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-positions-sync`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("IslandRuntimeJS() must not keep editor preview dock positions control event boundary %q", forbidden)
		}
	}
}

func TestSubscriberRuntimeScriptPublishesObservers(t *testing.T) {
	// The preview-side subscriber registers observers for each
	// $preview.* signal family. This test guards against drift between
	// the writer's signal names (asserted above) and the subscriber's
	// observers — if the writer publishes a signal the subscriber
	// doesn't observe, the cross-frame contract silently breaks.
	body := string(PreviewSubscriberScript())
	if body == "" {
		t.Fatal("PreviewSubscriberScript() must return a non-empty JS snippet")
	}
	for _, signalName := range []string{
		"$preview.mount.epoch",
		"$preview.block.",
		"$preview.text.",
		"$preview.theme.kit",
		"$preview.theme.template",
		"$preview.theme.palette",
		"$preview.theme.imageRatio",
		"$preview.theme.customClasses",
		"$preview.theme.styleClasses",
		"$preview.theme.colors",
		"$preview.style.impact.selector",
		"$preview.theme.cssCustom",
		"$preview.theme.fontsCustom",
		"$preview.brand.headerLogo",
		"$preview.editor.inlineEdit.requestId",
		"$preview.editor.fieldCycle.requestId",
	} {
		if !strings.Contains(body, signalName) {
			t.Fatalf("PreviewSubscriberScript() missing observer for %q", signalName)
		}
	}
	// The subscriber resolves observers through the in-frame fan-out
	// hook. Drift in the hook name would silently disable the entire
	// subscriber side of the cross-frame contract.
	if !strings.Contains(body, "__gosx_observe_shared_signal") {
		t.Fatalf("PreviewSubscriberScript() must call window.__gosx_observe_shared_signal")
	}
}

func TestSubscriberRuntimeScriptPreservesLegacyDOMMutations(t *testing.T) {
	// The subscriber applies the same DOM mutations the legacy
	// the legacy bundle performed cross-frame. This test guards
	// against drift between the legacy mutation shape (asserted by
	// existing parity tests in slices 3/4/5 against their iframe-
	// crossing methods) and the subscriber's local-frame implementation.
	body := string(PreviewSubscriberScript())
	for _, contract := range []string{
		// setBlockVisibility — display style toggle on matching key.
		"data-studio-block-key",
		// applyTextUpdate — [data-editor-preview="<sourceKey>"] target.
		"data-editor-preview",
		// applyTheme — .site-shell class manipulation + color CSS vars.
		".site-shell",
		"theme--template-",
		"theme--kit-",
		"theme--palette-",
		"theme--image-",
		// applyStyleImpact — [data-studio-style-impact-node] attribute.
		"data-studio-style-impact-node",
		// applyCSS / applyFonts — <style data-editor-live-*> markers.
		"data-editor-live-css",
		"data-editor-live-fonts",
		// updateHeaderLogo — .brand / .brand__logo + CSS variables.
		".brand",
		".brand__logo",
		"--brand-logo-width",
		"--brand-logo-offset-x",
		"--brand-logo-offset-y",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("PreviewSubscriberScript() missing legacy DOM contract %q", contract)
		}
	}
}

func TestBridgeShimAutoMountsOnDocumentReady(t *testing.T) {
	// Pre-2026-05-27 the deleted preview-runtime.js bundle was wired via the
	// CanvasEngine factory in studio-engines.js, which called
	// GoSXStudioPreviewRuntime.mount(document) on engine mount. With both
	// bundles deleted the per-slice BridgeShim correctly re-published
	// window.GoSXStudioPreviewRuntime but the auto-mount call was lost —
	// nothing invoked .mount(document) on page load anymore, so the preview
	// epoch signal ($preview.mount.epoch) was never bumped at boot and the
	// preview iframe's subscriber never wired its bindPreview pass on
	// initial load. The v0.4.1 fix landed this same contract for
	// fieldruntime; v0.5.0 restores it across the remaining six islands.
	// The other nine methods on the contract are host-driven writes, not
	// boot bindings.
	shim := string(BridgeShim())
	for _, fragment := range []string{
		"DOMContentLoaded",
		"window.GoSXStudioPreviewRuntime.mount",
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing auto-mount fragment %q:\n%s", fragment, shim)
		}
	}
}

func TestBundleConcatenatesIslandRuntimeAndShim(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	// Island runtime must come before the BridgeShim — the shim consults
	// the globals the island runtime publishes. Order matters so direct
	// calls can observe any globals published during bundle initialization.
	islandIdx := strings.Index(bundle, IslandGlobals.Mount+" ")
	shimIdx := strings.Index(bundle, "window.GoSXStudioPreviewRuntime =")
	if islandIdx < 0 || shimIdx < 0 {
		t.Fatalf("Bundle() missing island runtime or shim:\n%s", bundle)
	}
	if islandIdx > shimIdx {
		t.Fatalf("Bundle() must place island runtime before shim (island=%d shim=%d)", islandIdx, shimIdx)
	}
}

func TestPreviewSubscriberScriptIsNonEmpty(t *testing.T) {
	// PreviewSubscriberScript() is the bundle the storefront mounts when
	// loaded in the editor preview iframe. It must be non-empty AND
	// separate from the editor-side bundle so storefront pages don't
	// ship the editor-side island writer code.
	subscriber := PreviewSubscriberScript()
	if len(subscriber) == 0 {
		t.Fatal("PreviewSubscriberScript() must return a non-empty bundle")
	}
	bundle := Bundle()
	if strings.Contains(string(bundle), string(subscriber[:min(len(subscriber), 200)])) {
		t.Fatal("PreviewSubscriberScript() must not be a substring of Bundle() — they ship on different pages")
	}
}

func TestIslandRuntimeOwnsPageCanvasRoutingDiagnosticsKeyboardAndRestore(t *testing.T) {
	script := string(IslandRuntimeJS())
	for _, fragment := range []string{
		`button[data-studio-preview-route]`,
		`refreshEditorPreviewNow(form, "route", route)`,
		`function diagnoseUnsupportedEditorPreviewNode(form, frame, target)`,
		`"unsupported-node"`,
		`"same-origin-unavailable"`,
		`"stale-selection"`,
		`data-gosx-studio-preview-tab-stop`,
		`event.key === "Enter" || event.key === " "`,
		`function restoreEditorPreviewSelection(form, frame, host)`,
		`function bindEditorPreviewSelectionRestoreHandshake(form, host)`,
		`gosxstudio:preview-selection-restore-request`,
		`gosxstudio:preview-selection-locator-restore`,
		`gosxstudio:preview-selection-locator-stale`,
		`detail.route = editorPreviewRouteForFrame(frame)`,
		`function stopEditorPreviewActivation(event)`,
		`function editorPreviewActionControl(target)`,
		`function blockEditorPreviewActionControl(form, frame, event, actionControl)`,
		`Form submission and navigation are disabled inside the authoring canvas.`,
		`if (!editorPreviewPointerEventEligible(event))`,
		`if (editorPreviewInlineEditContains(frame, event.target)) {`,
		`if (actionControl) event.preventDefault()`,
		`if (selectionApplied || actionControl)`,
		`frameDoc.addEventListener("auxclick"`,
		`frameDoc.addEventListener("submit"`,
		`"preview-action-blocked"`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-suspend"`,
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("page-canvas runtime missing %q", fragment)
		}
	}
}
