package previewruntime

import (
	"strings"
	"testing"
)

// The previewruntime package is the Go-side surface for the Phase 3 slice-6
// architectural burn-down of GoSXStudioPreviewRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip the island path (legacy bundle deleted 2026-05-27)
//     to the .gsx-authored subscriber + island writer in this package.
//  2. The JS shim that gosx-studio's runtime bundle appends so the
//     window.GoSXStudioPreviewRuntime methods publish to $preview.* shared
//     signals when the flag is on (and fall back to the legacy 1130-line
//     bridge when off).
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
	// The shim must reference each method-specific island global, the
	// public global it shims (window.GoSXStudioPreviewRuntime), and the
	// feature flag so the legacy path stays reachable when the flag is
	// off. The IslandGlobals struct holds the canonical names; any drift
	// between the struct and the shim is caught here.
	for _, fragment := range []string{
		"window.GoSXStudioPreviewRuntime",
		"data-gosx-studio-feature-flag-preview-runtime-islands",
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
}

func TestIslandRuntimeJSPublishesMountGlobal(t *testing.T) {
	// Method 1/10: mount(root) — legacy
	// window.GoSXStudioPreviewRuntime.mount at the legacy bundle:1119
	// (bound to init at line 1113). The legacy walks every
	// [data-editor-workbench] in root and calls bindPreview(form). The
	// island writer publishes $preview.mount.epoch — the subscriber
	// re-runs its mount-acknowledged routine when the epoch changes.
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

func TestPreviewRuntimeIslandJSOwnsEditorPreviewChromeEvents(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-status-set"`,
		`doc.addEventListener("gosxstudio:editor-preview-route-sync"`,
		`doc.addEventListener("gosxstudio:editor-preview-refresh-now"`,
		`doc.addEventListener("gosxstudio:editor-preview-refresh-schedule"`,
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
		`detail.result = result`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview chrome fragment %q", fragment)
		}
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview chrome control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewPatchTransport(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-patch-apply"`,
		`doc.addEventListener("gosxstudio:editor-preview-patch-post"`,
		`setEditorPreviewResult(detail, applyEditorPreviewPatch(detail.frame, detail.patch))`,
		`setEditorPreviewResult(detail, postEditorPreviewPatch(form, detail.reason, detail.patch))`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview patch control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewTargetLookup(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-targets-resolve"`,
		`var field = detail.patch && detail.patch.field;`,
		`var source = field && (field.source || field.name);`,
		`detail.result = { handled: false, targets: [], count: 0 };`,
		`var targets = editorPreviewPatchTargets(detail.frame, detail.patch);`,
		`detail.result = { handled: true, targets: targets, count: targets.length };`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-targets-resolve`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview target lookup events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewFieldMapMarkers(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-field-map-clear"`,
		`doc.addEventListener("gosxstudio:editor-preview-field-map-sync"`,
		`setEditorPreviewResult(detail, clearEditorPreviewFieldMap(detail.frame))`,
		`setEditorPreviewResult(detail, syncEditorPreviewFieldMap(detail.frame, detail.fields, detail.current, detail.count))`,
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
		`doc.addEventListener("gosxstudio:editor-preview-field-navigation-state"`,
		`setEditorPreviewResult(detail, editorPreviewFieldNavigationState(detail.frame, detail.target, detail.selection || detail.detail || {}))`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-field-navigation-state`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview field-navigation state control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewSelectionMarkers(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-selection-marker-clear"`,
		`doc.addEventListener("gosxstudio:editor-preview-selection-marker-apply"`,
		`setEditorPreviewResult(detail, clearEditorPreviewSelectionMarker(detail.frame))`,
		`setEditorPreviewResult(detail, applyEditorPreviewSelectionMarker(detail.frame, detail.targets, detail.clear))`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selection-marker`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview selected-marker control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewSelectionChrome(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-selection-chrome-clear"`,
		`doc.addEventListener("gosxstudio:editor-preview-selection-chrome-apply"`,
		`var form = editorPreviewForm(detail, event)`,
		`setEditorPreviewResult(detail, clearEditorPreviewSelectionChrome(form))`,
		`setEditorPreviewResult(detail, applyEditorPreviewSelectionChrome(form, detail.frame, detail.selection))`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selection-chrome`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview selection chrome control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockHide(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-hide"`,
		`var form = editorPreviewForm(detail, event)`,
		`setEditorPreviewResult(detail, hideEditorPreviewDocks(form))`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-hide`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview dock hide control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockNormalization(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-normalize"`,
		`setEditorPreviewResult(detail, normalizeEditorPreviewDock(detail.dock))`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-normalize`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview dock normalize control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockShowContent(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-sync"`,
		`setEditorPreviewResult(detail, syncEditorPreviewDock(detail.dock, detail.selection))`,
		`function editorPreviewDockKindLabel(selection)`,
		`if (!selection || !selection.field) return "Block"`,
		`if (editable === "media" || editable === "image") return "Media field"`,
		`if (editable === "source") return "Source field"`,
		`if (editable === "flow") return "Flow field"`,
		`if (editable === "url" || editable === "link") return "Link field"`,
		`if (editable === "text") return "Text field"`,
		`function syncEditorPreviewDock(dock, selection)`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-sync`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview dock sync control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockFieldNavigationUI(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-field-navigation-sync"`,
		`setEditorPreviewResult(detail, syncEditorPreviewDockFieldNavigation(detail.dock, detail.count, detail.index))`,
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
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-field-navigation-sync`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview dock field-navigation control events")
	}
}

func TestPreviewRuntimeIslandJSOwnsEditorPreviewDockPosition(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		`doc.addEventListener("gosxstudio:editor-preview-dock-position-sync"`,
		`setEditorPreviewResult(detail, syncEditorPreviewDockPosition(detail.frame, detail.dock, detail.target, detail.shell))`,
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
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing editor preview dock position fragment %q", fragment)
		}
	}
	if strings.Contains(body, `dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-position-sync`) {
		t.Fatalf("IslandRuntimeJS() must not recursively dispatch editor preview dock position control events")
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
	// Island runtime must come before the BridgeShim — the shim
	// consults the globals the island runtime publishes. Order matters:
	// if the shim ran first it would close over an undefined global and
	// every dispatch would fall through to the legacy path, silently
	// disabling the slice.
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
