package studio

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func legacyMuddyGlobal(name string) string {
	return "__" + "muddy" + name
}

func jsFunctionBody(t *testing.T, script, name string) string {
	t.Helper()
	start := strings.Index(script, "function "+name+"(")
	if start < 0 {
		t.Fatalf("script missing function %s", name)
	}
	next := strings.Index(script[start+len("function "+name+"("):], "\n    function ")
	if next < 0 {
		return script[start:]
	}
	return script[start : start+len("function "+name+"(")+next]
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

func TestWorkbenchRuntimeDoesNotOwnShellRenderedPreviewDockSelectionSync(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function syncPreviewDock(frame, target, detail)`,
		`gosxstudio:editor-preview-dock-selection-sync`,
		`function previewShellForFrame`,
		`function previewDockForFrame`,
		`function normalizePreviewDock`,
		`shell.querySelector("[data-studio-preview-dock], [data-gosx-studio-preview-dock]")`,
		`gosxstudio:editor-preview-dock-bind`,
		`function createDockButton`,
		`data-gosx-studio-preview-dock-fallback`,
		`dock = document.createElement("div")`,
		`actions.appendChild(createDockButton`,
	} {
		if strings.Contains(script, check) {
			t.Fatalf("workbench runtime should not own preview dock selection-sync/lookup/binding/fallback fragment %q", check)
		}
	}
}

func TestWorkbenchRuntimeLetsFieldRuntimeOwnMirroredPreviewPatches(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function shouldTransportPreviewPatch(field, reason)`,
		`target.dispatchEvent(new CustomEvent("gosxstudio:field-preview-patch-resolve", {`,
		`var runtime = window.GoSXStudioFieldRuntime;`,
		`runtime.bindMirroring(root);`,
		`runtime.bind(root);`,
		`return !(result && result.transport === false);`,
		`emitFieldOperation("input", event.target);`,
		`if (shouldTransportPreviewPatch(event.target, "input")) postPreviewPatch("input", {}, event.target);`,
		`emitFieldOperation("change", event.target);`,
		`if (shouldTransportPreviewPatch(event.target, "change")) postPreviewPatch("change", {}, event.target);`,
		`shouldTransportPreviewPatch: shouldTransportPreviewPatch`,
		`postPreviewPatch("transaction", event.detail || {}, null);`,
		`postPreviewPatch("history-restore", event.detail || {}, null);`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing FieldRuntime preview patch ownership fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		`function fieldRuntimeMirrors(field)`,
		`field.matches("[data-editor-source], [data-studio-field-source], [data-editor-frame-attr-target]")`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should not own mirrored FieldRuntime preview patch policy fragment %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDelegatesFieldOperationEnvelopeToFieldRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function fieldOperationEnvelope(reason, field)`,
		`var payload = {`,
		`form: form,`,
		`reason: reason || "field",`,
		`field: field || null,`,
		`result: null`,
		`target.dispatchEvent(new CustomEvent("gosxstudio:field-operation-build", {`,
		`if (!envelope && bindFieldRuntimeForPreviewPatchResolution()) {`,
		`return emitEditorOperation("set_field", envelope);`,
		`emitFieldOperation("input", event.target);`,
		`emitFieldOperation("change", event.target);`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing delegated FieldRuntime operation builder fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		`function fieldPatch(field)`,
		`field.name === "csrf_token"`,
		`type === "button" || type === "submit" || type === "reset" || type === "file"`,
		`source: field.getAttribute("data-studio-field-source") || field.getAttribute("data-editor-source") || field.name`,
		`value: type === "checkbox" || type === "radio"`,
		`checked: !!field.checked`,
		`tag: String(field.tagName || "").toLowerCase()`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should leave concrete field operation patch construction to FieldRuntime; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDelegatesEditorPreviewPatchTransport(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function postPreviewPatch(reason, detail, field)`,
		`var payload = { form: form, reason: reason || "patch", detail: detail || {}, field: field || null };`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-patch-build-post", { bubbles: true, detail: payload }));`,
		`return payload.result || null;`,
		`postPreviewPatch("transaction", event.detail || {}, null);`,
		`postPreviewPatch("history-restore", event.detail || {}, null);`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing delegated editor preview patch transport fragment %q", check)
		}
	}
	body := jsFunctionBody(t, script, "postPreviewPatch")
	for _, forbidden := range []string{
		`type: "gosxstudio:preview-patch"`,
		`source: "gosx-studio"`,
		`field: fieldPatch(field)`,
		`var payload = { form: form, frames: frames, patch: patch`,
		`function applyPreviewPatch(frame, patch)`,
		`function updatePreviewTarget`,
		`frame.contentWindow.postMessage`,
		`new URL(frame.getAttribute("src"), window.location.href).origin`,
		`target.setAttribute("data-gosx-studio-preview-patched"`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("workbench runtime should delegate concrete preview patch transport; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDoesNotOwnEditorPreviewTargetLookup(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, forbidden := range []string{
		`function previewTargets(frame, patch)`,
		`gosxstudio:editor-preview-targets-resolve`,
		`var selectedTargets = detail.field ? previewTargets(frame, { field: { source: detail.field, name: detail.field } }) : [];`,
		`function previewPatchSelector`,
		`return queryAll(doc, previewPatchSelector(source));`,
		`'[data-studio-field="' + source + '"]'`,
		`'[data-editor-preview="' + source + '"]'`,
		`'[data-studio-field-source="' + source + '"]'`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should leave editor preview target lookup to PreviewRuntime; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewDocumentBindingToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "previewDocumentHost")
	for _, forbidden := range []string{
		`function bindPreviewDocument(frame)`,
		`gosxstudio:editor-preview-document-bind`,
		`function frameDocument(frame)`,
		`function previewSelectableNode(target)`,
		`function previewPointerEventEligible(event)`,
		`function previewKeyIntent(event)`,
		`frame.__gosxStudioPreviewDocument`,
		`data-gosx-studio-preview-selectable`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should not keep old preview document binding wrapper %q", forbidden)
		}
	}
	for _, check := range []string{
		`function previewDocumentHost()`,
		`previewSelectionDetail: previewSelectionDetail,`,
		`applyPreviewSelection: applyPreviewSelection,`,
		`finishInlineTextEdit: finishInlineTextEdit,`,
		`dispatchPreviewFieldNavigation: dispatchPreviewFieldNavigation,`,
		`startInlineTextFromDetail: startInlineTextFromDetail,`,
		`startInlineTextFromSelection: startInlineTextFromSelection,`,
		`handleInlineTextInput: handleInlineTextInput,`,
		`handleInlineTextKeyEvent: handleInlineTextKeyEvent,`,
		`handleInlineTextPaste: handleInlineTextPaste,`,
		`handleInlineTextBlur: handleInlineTextBlur,`,
		`updatePreviewDockPosition: updatePreviewDockPosition`,
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("previewDocumentHost missing PreviewRuntime document-bind fragment %q in:\n%s", check, body)
		}
	}

	for _, forbidden := range []string{
		`doc.addEventListener(`,
		`.addEventListener("click"`,
		`.addEventListener("dblclick"`,
		`.addEventListener("focusin"`,
		`.addEventListener("input"`,
		`.addEventListener("keydown"`,
		`.addEventListener("paste"`,
		`.addEventListener("blur"`,
		`.addEventListener("scroll"`,
		`contentWindow.addEventListener("resize"`,
		`target.closest("[data-studio-field], [data-editor-preview], [data-studio-field-source], [data-studio-block-key], [data-studio-node-id]")`,
		`[data-studio-block-key], [data-studio-node-id]`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("previewDocumentHost should not own concrete preview document binding; found %q in:\n%s", forbidden, body)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewFrameLifecycleToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "bindPreviewFrames")
	for _, check := range []string{
		`function bindPreviewFrames()`,
		`var detail = {`,
		`form: form,`,
		`host: Object.assign({`,
		`shouldTransportPreviewPatch: shouldTransportPreviewPatch`,
		`}, previewDocumentHost())`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-frames-bind", { bubbles: true, detail: detail }));`,
		`return detail.result || null;`,
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("bindPreviewFrames missing PreviewRuntime frame-bind fragment %q in:\n%s", check, body)
		}
	}

	for _, forbidden := range []string{
		`previewFrames().forEach`,
		`frame.dataset.gosxStudioPreviewBound`,
		`data-studio-preview-src`,
		`frame.addEventListener("load"`,
		`frame.addEventListener("error"`,
		`previewURL(frame)`,
		`syncPreviewFrame(frame, "load")`,
		`bindPreviewDocument(frame)`,
		`postPreviewPatch("load-sync", { route: previewURL(frame)`,
		`postPreviewLoadSyncPatch`,
		`setPreviewStatus: setPreviewStatus`,
		`syncPreviewFrame: syncPreviewFrame`,
		`bindPreviewDocument: bindPreviewDocument`,
		`setPreviewStatus("error", "Preview failed", "error")`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("bindPreviewFrames should not own concrete preview frame lifecycle; found %q in:\n%s", forbidden, body)
		}
	}
	if strings.Contains(script, `function previewURL(frame)`) {
		t.Fatalf("workbench runtime should not keep previewURL after delegating frame lifecycle")
	}
}

func TestWorkbenchRuntimeDelegatesPreviewEventPolicyToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "previewDocumentHost")
	for _, forbidden := range []string{
		`function previewPointerEventEligible(event)`,
		`function previewKeyIntent(event)`,
		`gosxstudio:editor-preview-pointer-event-eligible`,
		`gosxstudio:editor-preview-key-intent`,
		`previewPointerEventEligible(event)`,
		`previewKeyIntent(event)`,
		`var intent =`,
		`event.preventDefault()`,
		`event.stopPropagation()`,
		`event.defaultPrevented || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || event.button > 0`,
		`event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey`,
		`var focusedControl = event.target && event.target.closest ? event.target.closest("input, textarea, select, button, a[href], [contenteditable='true'], [contenteditable='plaintext-only']") : null;`,
		`if (focusedControl) return;`,
		`if (event.key === "[" || event.key === "]")`,
		`event.key === "]" ? "keyboard-next-field" : "keyboard-prev-field"`,
		`event.key === "F2" ? "keyboard-f2" : "keyboard-enter"`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("previewDocumentHost should delegate preview event policy; found %q in:\n%s", forbidden, body)
		}
	}
}

func TestWorkbenchRuntimeDelegatesEditorPreviewChromeToPreviewRuntimeEvents(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function setPreviewStatus(state, label, reason)`,
		`{ form: form, state: state, label: label, reason: reason || "" }`,
		`gosxstudio:editor-preview-status-set`,
		`function syncPreviewRoute(route, reason)`,
		`{ form: form, route: route || "", reason: reason || "" }`,
		`gosxstudio:editor-preview-route-sync`,
		`function refreshPreviewNow(reason, route)`,
		`{ form: form, route: route || "", reason: reason || "refresh" }`,
		`gosxstudio:editor-preview-refresh-now`,
		`function schedulePreviewRefresh(reason, route)`,
		`gosxstudio:editor-preview-refresh-schedule`,
		`return detail.result || null;`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing editor preview chrome delegation fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		`shell.setAttribute("data-gosx-studio-preview-url"`,
		`link.setAttribute("href", route)`,
		`next.searchParams.set("_gosx_preview"`,
		`frame.setAttribute("src", cacheBustURL`,
		`function cacheBustURL`,
		`function openLinks`,
		`queryAll(form, "[data-studio-selected-flow-route]")`,
		`queryAll(shell, "[data-studio-preview-status]")`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should delegate editor preview chrome ownership; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeSkipsMirroredFieldsDuringPreviewLoadSync(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "bindPreviewFrames")
	for _, check := range []string{
		`host: Object.assign({`,
		`shouldTransportPreviewPatch: shouldTransportPreviewPatch`,
		`}, previewDocumentHost())`,
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("workbench runtime missing delegated load-sync fragment %q", check)
		}
	}

	for _, forbidden := range []string{
		`function syncPreviewFrame(frame, reason)`,
		`gosxstudio:editor-preview-frame-sync`,
		`queryAll(form, "[data-studio-field-source], [data-editor-source], input[name], textarea[name], select[name]")`,
		`type: "gosxstudio:preview-patch"`,
		`field: fieldPatch(field)`,
		`count += applyPreviewPatch(frame, patch)`,
		`if (!frameDocument(frame)) return 0`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("bindPreviewFrames should only expose load-sync transport policy; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewInlineTextToInlineEditRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		"window.GoSXStudioInlineEditRuntime",
		"startPreviewTextSession",
		"syncPreviewTextSession",
		"finishPreviewTextSession",
		"handlePreviewTextInputEvent",
		"handlePreviewTextKeyEvent",
		"handlePreviewTextPasteEvent",
		"handlePreviewTextBlurEvent",
		"previewInlineTextHost",
		"controlForField: textControlForField",
		"setStatus: setPreviewStatus",
		"emitEditorOperation: emitEditorOperation",
		"emitEvent: emitPreviewInlineTextEvent",
		"updateDock: updatePreviewDockPosition",
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
		"function previewInlineTextOptions",
		"selection: function (edit)",
		"setDirty: function",
		"emitOperation: function",
		"emitEvent: function",
		"onFinish: function",
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should delegate preview inline-text lifecycle, found low-level fragment %q", forbidden)
		}
	}

	body := jsFunctionBody(t, script, "previewDocumentHost")
	for _, check := range []string{
		"handleInlineTextInput: handleInlineTextInput",
		"handleInlineTextKeyEvent: handleInlineTextKeyEvent",
		"handleInlineTextPaste: handleInlineTextPaste",
		"handleInlineTextBlur: handleInlineTextBlur",
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("previewDocumentHost should pass inline text host callbacks, missing %q in:\n%s", check, body)
		}
	}
	for _, forbidden := range []string{
		`doc.addEventListener("input"`,
		`doc.addEventListener("keydown"`,
		`doc.addEventListener("paste"`,
		`doc.addEventListener("blur"`,
		`syncInlineTextEdit(frame, "input")`,
		`finishInlineTextEdit(frame, false, "escape")`,
		`finishInlineTextEdit(frame, true, "enter")`,
		`finishInlineTextEdit(frame, true, "blur")`,
		`event.key === "Escape"`,
		`event.key === "Enter" && !event.shiftKey`,
		`event.clipboardData && event.clipboardData.getData`,
		`selection.deleteFromDocument()`,
		`selection.getRangeAt(0).insertNode(doc.createTextNode(text))`,
		`selection.collapseToEnd()`,
		`syncInlineTextEdit(frame, "paste")`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("previewDocumentHost should delegate preview inline-text event policy, found %q in:\n%s", forbidden, body)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewSelectionStateToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function previewSelectionDetail(node)`,
		`var payload = { target: node };`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-detail-resolve", { bubbles: true, detail: payload }))`,
		`if (payload.result) return payload.result;`,
		`runtime.bind(document.body || document);`,
		`return {};`,
		`function dispatchPreviewSelectionApply(detail, options)`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-apply", { bubbles: true, detail: payload }))`,
		`if (payload.result) return payload.result;`,
		`var runtime = window.GoSXStudioSelectionRuntime;`,
		`runtime.bind(document.body || document);`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-selection-apply-sync", { bubbles: true, detail: payload }))`,
		`host: Object.assign({`,
		`dispatchPreviewSelectionApply: dispatchPreviewSelectionApply`,
		`}, previewDockActionHost())`,
		`finishInlineTextEdit: finishInlineTextEdit,`,
		`if (!payload.result || !payload.result.handled || !payload.result.applied) return false;`,
		`function revealPreviewField(detail, reason)`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-reveal", { bubbles: true, detail: payload }))`,
		`if (payload.result) return !!payload.result.revealed;`,
		`if (options.reveal) revealPreviewField(detail, options.reason || "preview-select");`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview selection delegation fragment %q", check)
		}
	}

	for _, forbidden := range []string{
		`function readableFieldName`,
		`function inferInspectorEditable`,
		`function previewBlockLabel`,
		`fieldNode.getAttribute("data-studio-field")`,
		`node.getAttribute("data-studio-block-label")`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should delegate preview selection detail resolution, found %q", forbidden)
		}
	}

	applyBody := jsFunctionBody(t, script, "applyPreviewSelection")
	for _, forbidden := range []string{
		`clearInspectorSelection();`,
		`markInspectorSelection(source)`,
		`form.setAttribute("data-studio-field-selection"`,
		`form.removeAttribute("data-studio-field-selection"`,
		`form.setAttribute("data-studio-field-editable"`,
		`form.removeAttribute("data-studio-field-editable"`,
		`form.setAttribute("data-studio-field-action-label"`,
		`form.removeAttribute("data-studio-field-action-label"`,
		`form.setAttribute("data-studio-field-action-href"`,
		`form.removeAttribute("data-studio-field-action-href"`,
		`form.setAttribute("data-studio-field-action-formaction"`,
		`form.removeAttribute("data-studio-field-action-formaction"`,
		`form.setAttribute("data-studio-selection"`,
		`form.setAttribute("data-studio-selection-kind"`,
		`setReadout("[data-studio-selection-label]"`,
		`setReadout("[data-studio-selection-status]"`,
		`setReadout("[data-studio-field-selection-label]"`,
		`setAttribute("data-gosx-studio-preview-selected"`,
		`removeAttribute("data-gosx-studio-preview-selected"`,
		`setAttribute("data-gosx-studio-preview-selection"`,
		`setAttribute("data-studio-preview-selection"`,
		`removeAttribute("data-gosx-studio-preview-selection"`,
		`removeAttribute("data-studio-preview-selection"`,
		`emit(form, "gosxstudio:preview-select"`,
		`emitEditorOperation("select_preview"`,
		`var selectedEditable = result.editable || detail.editable || "";`,
		`field: result.field || detail.field || ""`,
		`editable: selectedEditable`,
		`label: result.label || detail.label || ""`,
		`blockLabel: result.blockLabel || detail.blockLabel || ""`,
		`action: result.action || ""`,
		`actionHref: result.actionHref || ""`,
		`actionFormAction: result.actionFormAction || ""`,
		`blockKey: result.blockKey || detail.blockKey || ""`,
		`nodeID: result.nodeID || detail.nodeID || ""`,
		`fieldSourceNode`,
		`inspectorSource`,
		`inspectorControl`,
		`revealInspectorSelection`,
		`gosxstudio:editor-preview-targets-resolve`,
		`gosxstudio:editor-preview-selection-clear-sync`,
		`gosxstudio:editor-preview-selection-marker-apply`,
		`gosxstudio:editor-preview-selection-chrome-apply`,
		`gosxstudio:editor-preview-dock-selection-resolve`,
		`gosxstudio:editor-preview-dock-selection-sync`,
		`previewTargets(`,
		`applyPreviewSelectionMarker(`,
		`applyPreviewSelectionChrome(`,
		`previewDockSelection(`,
		`syncPreviewDock(`,
	} {
		if strings.Contains(applyBody, forbidden) {
			t.Fatalf("applyPreviewSelection should delegate editor preview selection state; found %q in:\n%s", forbidden, applyBody)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewFieldTargetRevealToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function previewFieldTarget(detailOrField)`,
		`var detail = typeof detailOrField === "string" ? { field: detailOrField } : (detailOrField || {});`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-target-resolve", { bubbles: true, detail: payload }))`,
		`if (payload.result) return payload.result;`,
		`bindSelectionRuntime()`,
		`return { field: detail.field || "", source: null, control: null, found: false };`,
		`function revealPreviewField(detail, reason)`,
		`setMode("content", { reason: reason || "preview-select" });`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-reveal", { bubbles: true, detail: payload }))`,
		`if (payload.result) return !!payload.result.revealed;`,
		`var target = previewFieldTarget(field);`,
		`if (target.control && "value" in target.control) return target.control;`,
		`if (target.source && "value" in target.source) return target.source;`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview field target/reveal delegation fragment %q", check)
		}
	}

	for _, forbidden := range []string{
		`function inspectorSource`,
		`function inspectorControl`,
		`function revealInspectorSelection`,
		`form.querySelector('[data-studio-field-source="'`,
		`[data-editor-source="`,
		`.field-row, [data-studio-field-row]`,
		`focus({ preventScroll: true })`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should delegate concrete preview field lookup/reveal policy; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewFieldNavigationTelemetryToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function dispatchPreviewFieldNavigation(detail, navigation)`,
		`detail: detail || {}`,
		`navigation: navigation || {}`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-navigation-commit", { bubbles: true, detail: payload }))`,
		`if (payload.result) return payload.result;`,
		`var runtime = window.GoSXStudioSelectionRuntime;`,
		`runtime.bind(document.body || document);`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview field navigation delegation fragment %q", check)
		}
	}

	navigateBody := jsFunctionBody(t, script, "navigatePreviewField")
	for _, check := range []string{
		`form: form,`,
		`frame: frame,`,
		`direction: direction,`,
		`reason: reason || "field-navigation",`,
		`finishInlineTextEdit: finishInlineTextEdit,`,
		`previewSelectionDetail: previewSelectionDetail,`,
		`applyPreviewSelection: applyPreviewSelection,`,
		`dispatchPreviewFieldNavigation: dispatchPreviewFieldNavigation`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-field-navigation-run", { bubbles: true, detail: payload }))`,
		`return !!(payload.result && payload.result.navigated);`,
	} {
		if !strings.Contains(navigateBody, check) {
			t.Fatalf("navigatePreviewField missing field navigation boundary fragment %q in:\n%s", check, navigateBody)
		}
	}
	for _, forbidden := range []string{
		`emitEditorOperation("preview_field_navigate"`,
		`emit(form, "gosxstudio:preview-field-navigate"`,
		`previewFieldNavigationState(`,
		`% state.count`,
		`applyPreviewSelection(frame, nextTarget`,
		`dispatchPreviewFieldNavigation(nextDetail`,
	} {
		if strings.Contains(navigateBody, forbidden) {
			t.Fatalf("navigatePreviewField should delegate preview field navigation telemetry; found %q in:\n%s", forbidden, navigateBody)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewFieldMapMarkersToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	if strings.Contains(script, `function clearPreviewFieldMap`) {
		t.Fatal("workbench runtime should not own preview field-map clear orchestration after PreviewRuntime selection-clear migration")
	}
	if strings.Contains(script, `function syncPreviewFieldMap`) {
		t.Fatal("workbench runtime should not own preview field-map selection sync after PreviewRuntime dock selection-sync migration")
	}
}

func TestWorkbenchRuntimeDelegatesPreviewFieldNavigationStateToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, forbidden := range []string{
		`function previewFieldNavigationState`,
		`gosxstudio:editor-preview-field-navigation-state`,
		`function fieldKeyForTarget`,
		`function fieldNavigationScope`,
		`function previewFieldNodesForSelection`,
		`target.closest("[data-studio-block-key], [data-studio-node-id]")`,
		`doc.querySelector(selector)`,
		`var seen = {}`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should delegate preview field-navigation state calculation; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDoesNotOwnPreviewDockFieldNavigationUI(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	if strings.Contains(script, `function updatePreviewFieldNavigation`) {
		t.Fatal("workbench runtime should not own preview dock field-navigation UI sync after PreviewRuntime dock selection-sync migration")
	}
	for _, forbidden := range []string{
		`gosxstudio:editor-preview-dock-field-navigation-sync`,
		`setAttribute("data-gosx-studio-preview-field-count"`,
		`setAttribute("data-gosx-studio-preview-field-index"`,
		`meter.hidden`,
		`meter.textContent`,
		`button.hidden`,
		`button.disabled`,
		`button.setAttribute("aria-label"`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should not own preview dock field-navigation UI mutation/delegation; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewDockPositionToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "updatePreviewDockPosition")
	for _, check := range []string{
		`function updatePreviewDockPosition(frame)`,
		`var dock = frame && frame.__gosxStudioPreviewDock;`,
		`var target = frame && frame.__gosxStudioPreviewDockTarget;`,
		`var detail = { form: form, frame: frame, dock: dock, target: target };`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-position-sync", { bubbles: true, detail: detail }))`,
		`return detail.result || null;`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview dock position delegation fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		`previewShellForFrame(frame)`,
		`shell: shell`,
		`getBoundingClientRect`,
		`data-gosx-studio-preview-dock-placement`,
		`dock.style.top`,
		`dock.style.left`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("updatePreviewDockPosition should delegate concrete preview dock positioning mutation; found %q in:\n%s", forbidden, body)
		}
	}
}

func TestWorkbenchRuntimeDoesNotOwnPreviewDockShowContent(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, forbidden := range []string{
		`function syncPreviewDock(frame, target, detail)`,
		`gosxstudio:editor-preview-dock-selection-sync`,
		`previewDockForFrame(frame)`,
		`frame.__gosxStudioPreviewDock = dock`,
		`frame.__gosxStudioPreviewDockTarget = target`,
		`gosxstudio:editor-preview-dock-sync`,
		`updatePreviewFieldNavigation(frame, dock, target, detail)`,
		`syncPreviewFieldMap(frame, target, detail)`,
		`dock.hidden = false`,
		`dock.setAttribute("data-gosx-studio-preview-field"`,
		`dock.setAttribute("data-gosx-studio-preview-block"`,
		`dock.setAttribute("data-gosx-studio-preview-block-label"`,
		`dock.setAttribute("data-gosx-studio-preview-action-label"`,
		`dock.setAttribute("data-gosx-studio-preview-action-href"`,
		`dock.setAttribute("data-gosx-studio-preview-action-formaction"`,
		`dock.querySelector("[data-gosx-studio-preview-dock-label]").textContent`,
		`breadcrumb.hidden`,
		`breadcrumb.textContent`,
		`dockKindLabel(detail)`,
		`action.textContent`,
		`action.disabled`,
		`selection: { field: detail.field || ""`,
		`editable: detail.editable || ""`,
		`label: detail.label || ""`,
		`blockLabel: detail.blockLabel || ""`,
		`action: detail.action || ""`,
		`actionHref: detail.actionHref || ""`,
		`actionFormAction: detail.actionFormAction || ""`,
		`blockKey: detail.blockKey || ""`,
		`nodeID: detail.nodeID || ""`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should leave preview dock show/content mutation to PreviewRuntime; found %q", forbidden)
		}
	}
	if strings.Contains(script, `function dockKindLabel`) {
		t.Fatal("workbench runtime should not keep dockKindLabel after delegating preview dock content rendering")
	}
}

func TestWorkbenchRuntimeDelegatesPreviewDockDetailExtractionToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "previewDockDetail")
	for _, check := range []string{
		`function previewDockDetail(dock)`,
		`var detail = { form: form, dock: dock };`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-detail-resolve", { bubbles: true, detail: detail }))`,
		`return detail.result || {};`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview dock detail delegation fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		`dock.getAttribute("data-gosx-studio-preview-field")`,
		`dock.querySelector("[data-gosx-studio-preview-dock-label]")`,
		`data-gosx-studio-preview-action-formaction`,
		`form.getAttribute("data-studio-field-editable")`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("previewDockDetail should delegate concrete dock detail extraction; found %q in:\n%s", forbidden, body)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewFieldActionIntentToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function dispatchPreviewFieldActionResolve(detail)`,
		`detail: detail || {}`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-action-resolve", { bubbles: true, detail: payload }))`,
		`if (payload.result) return payload.result;`,
		`var runtime = window.GoSXStudioSelectionRuntime;`,
		`runtime.bind(document.body || document);`,
		`dispatchPreviewFieldActionResolve: dispatchPreviewFieldActionResolve`,
		`startInlineTextFromDetail: startInlineTextFromDetail`,
		`submitPreviewFieldAction: submitPreviewFieldAction`,
		`navigateToHref: navigatePreviewDockHref`,
		`function submitPreviewFieldAction(detail)`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-action-submit", { bubbles: true, detail: payload }))`,
		`if (payload.result) return !!payload.result;`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview field-action intent delegation fragment %q", check)
		}
	}

	for _, forbidden := range []string{
		`function isFormSubmitControl`,
		`function fieldActionSubmitter`,
		`form.requestSubmit`,
		`document.createElement("button")`,
		`form.dataset.gosxStudioPendingAction`,
		`form.submit()`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should delegate concrete field-action submit mechanics; found %q", forbidden)
		}
	}

	actionBody := jsFunctionBody(t, script, "runPreviewDockAction")
	for _, forbidden := range []string{
		`if (action === "field-action")`,
		`dispatchPreviewFieldActionResolve(detail)`,
		`startInlineTextFromDetail(frame, detail, "preview-dock")`,
		`submitPreviewFieldAction(detail)`,
		`window.location.href`,
		`revealPreviewField(detail, "preview-dock")`,
		`emitPreviewDockAction(action, detail)`,
		`detail.editable === "text"`,
		`detail.actionFormAction && submitPreviewFieldAction`,
		`if (detail.actionHref)`,
	} {
		if strings.Contains(actionBody, forbidden) {
			t.Fatalf("preview field-action branch should use resolved intent, found %q in:\n%s", forbidden, actionBody)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewDockContentStyleIntentToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function dispatchPreviewDockActionResolve(action, detail)`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-dock-action-resolve", { bubbles: true, detail: payload }))`,
		`if (payload.result) return payload.result;`,
		`var runtime = window.GoSXStudioSelectionRuntime;`,
		`runtime.bind(document.body || document);`,
		`action: action || ""`,
		`detail: detail || {}`,
		`dispatchPreviewDockActionResolve: dispatchPreviewDockActionResolve`,
		`setMode: setMode`,
		`revealPreviewField: revealPreviewField`,
		`emitPreviewDockAction: emitPreviewDockAction`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview dock content/style intent delegation fragment %q", check)
		}
	}

	actionBody := jsFunctionBody(t, script, "runPreviewDockAction")
	for _, forbidden := range []string{
		`if (action === "content" || action === "style")`,
		`dispatchPreviewDockActionResolve(action, detail)`,
		`setMode(dockIntent.mode`,
		`revealPreviewField(detail, "preview-dock")`,
		`emitPreviewDockAction(action, detail)`,
		`if (action === "content") {
        setMode("content"`,
		`if (action === "style") {
        setMode("style"`,
	} {
		if strings.Contains(actionBody, forbidden) {
			t.Fatalf("preview dock content/style branch should use resolved intent, found %q in:\n%s", forbidden, actionBody)
		}
	}
}

func TestWorkbenchRuntimeDispatchesPreviewDockActionRunToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "runPreviewDockAction")
	for _, check := range []string{
		`function previewDockActionHost()`,
		`clearPreviewSelections: clearPreviewSelections`,
		`dispatchPreviewSelectionClear: dispatchPreviewSelectionClear`,
		`emitPreviewDockAction: emitPreviewDockAction`,
		`dispatchPreviewDockActionResolve: dispatchPreviewDockActionResolve`,
		`setMode: setMode`,
		`revealPreviewField: revealPreviewField`,
		`finishInlineTextEdit: finishInlineTextEdit`,
		`previewSelectionDetail: previewSelectionDetail`,
		`applyPreviewSelection: applyPreviewSelection`,
		`dispatchPreviewFieldNavigation: dispatchPreviewFieldNavigation`,
		`dispatchPreviewFieldActionResolve: dispatchPreviewFieldActionResolve`,
		`startInlineTextFromDetail: startInlineTextFromDetail`,
		`submitPreviewFieldAction: submitPreviewFieldAction`,
		`navigateToHref: navigatePreviewDockHref`,
		`function runPreviewDockAction(frame, action)`,
		`form: form`,
		`frame: frame`,
		`action: action || ""`,
		`host: previewDockActionHost()`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-preview-dock-action-run", { bubbles: true, detail: payload }))`,
		`return !!(payload.result && payload.result.result);`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview dock action dispatcher fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		`if (action === "clear")`,
		`if (action === "content" || action === "style")`,
		`if (action === "field-action")`,
		`window.location.href`,
		`navigatePreviewField(frame, action === "next-field" ? 1 : -1`,
		`previewDockDetail(dock)`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("runPreviewDockAction should be a thin preview-runtime dispatcher; found %q in:\n%s", forbidden, body)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewSelectionClearToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, check := range []string{
		`function dispatchPreviewSelectionClear(reason)`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-selection-clear", { bubbles: true, detail: payload }))`,
		`if (payload.result) return true;`,
		`var runtime = window.GoSXStudioSelectionRuntime;`,
		`runtime.bind(document.body || document);`,
		`clearPreviewSelections: clearPreviewSelections`,
		`dispatchPreviewSelectionClear: dispatchPreviewSelectionClear`,
		`dispatchPreviewSelectionClear("keyboard-escape");`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing preview selection clear boundary fragment %q", check)
		}
	}
	for _, forbidden := range []string{
		`function clearInspectorSelection`,
		`function markInspectorSelection`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should not own preview inspector selection helper %q", forbidden)
		}
	}

	clearBody := jsFunctionBody(t, script, "clearPreviewSelections")
	for _, forbidden := range []string{
		`previewFrames().forEach`,
		`finishInlineTextEdit(frame, true, "clear-selection");`,
		`clearPreviewFieldMap`,
		`clearPreviewSelectionMarker`,
		`clearPreviewSelectionChrome`,
		`hidePreviewDocks`,
		`gosxstudio:editor-preview-field-map-clear`,
		`gosxstudio:editor-preview-selection-marker-clear`,
		`gosxstudio:editor-preview-selection-chrome-clear`,
		`gosxstudio:editor-preview-dock-hide`,
		`setAttribute("data-gosx-studio-preview-selected"`,
		`removeAttribute("data-gosx-studio-preview-selected"`,
		`setAttribute("data-gosx-studio-preview-selection"`,
		`setAttribute("data-studio-preview-selection"`,
		`removeAttribute("data-gosx-studio-preview-selection"`,
		`removeAttribute("data-studio-preview-selection"`,
	} {
		if strings.Contains(clearBody, forbidden) {
			t.Fatalf("clearPreviewSelections should delegate concrete preview selected-marker mutation; found %q in:\n%s", forbidden, clearBody)
		}
	}

	dockBody := jsFunctionBody(t, script, "runPreviewDockAction")
	for _, forbidden := range []string{
		`if (action === "clear")`,
		`clearPreviewSelections();`,
		`dispatchPreviewSelectionClear("preview-dock");`,
		`emitPreviewDockAction(action, detail);`,
		`clearInspectorSelection();`,
		`form.removeAttribute("data-studio-selection");`,
		`form.removeAttribute("data-studio-selection-kind");`,
		`form.removeAttribute("data-studio-field-selection");`,
		`form.removeAttribute("data-studio-field-editable");`,
		`form.removeAttribute("data-studio-field-action-label");`,
		`form.removeAttribute("data-studio-field-action-href");`,
		`form.removeAttribute("data-studio-field-action-formaction");`,
		`setReadout("[data-studio-selection-label]", "No selection");`,
		`setReadout("[data-studio-selection-status]", "No selection");`,
		`setReadout("[data-studio-field-selection-label]", "Block");`,
	} {
		if strings.Contains(dockBody, forbidden) {
			t.Fatalf("preview dock clear should delegate editor preview selection state; found %q in:\n%s", forbidden, dockBody)
		}
	}
}

func TestWorkbenchRuntimeDelegatesPreviewDockHideToPreviewRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, forbidden := range []string{
		`function hidePreviewDocks()`,
		`gosxstudio:editor-preview-dock-hide`,
		`dock.hidden = true`,
		`removeAttribute("data-gosx-studio-preview-field"`,
		`removeAttribute("data-gosx-studio-preview-block"`,
		`removeAttribute("data-gosx-studio-preview-action-label"`,
		`removeAttribute("data-gosx-studio-preview-action-href"`,
		`removeAttribute("data-gosx-studio-preview-action-formaction"`,
		`removeAttribute("data-gosx-studio-preview-block-label"`,
		`removeAttribute("data-gosx-studio-preview-field-count"`,
		`removeAttribute("data-gosx-studio-preview-field-index"`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should leave preview dock hide/reset to PreviewRuntime; found %q", forbidden)
		}
	}
}

func TestWorkbenchRuntimeLeavesPreviewDockSelectionActionTelemetryToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	body := jsFunctionBody(t, script, "emitPreviewDockAction")
	for _, check := range []string{
		`emit(form, "gosxstudio:preview-action", {`,
		`action: action || ""`,
		`detail: detail || {}`,
		`reason: "preview-dock"`,
	} {
		if !strings.Contains(body, check) {
			t.Fatalf("emitPreviewDockAction missing preview action contract %q in:\n%s", check, body)
		}
	}
	for _, forbidden := range []string{
		`field: detail.field || ""`,
		`editable: detail.editable || ""`,
		`label: detail.label || ""`,
		`blockLabel: detail.blockLabel || ""`,
		`blockKey: detail.blockKey || ""`,
		`actionLabel: detail.action || ""`,
		`actionHref: detail.actionHref || ""`,
		`actionFormAction: detail.actionFormAction || ""`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("emitPreviewDockAction should leave preview action envelope normalization to selection runtime; found %q in:\n%s", forbidden, body)
		}
	}
	if strings.Contains(body, `emitEditorOperation("preview_action"`) {
		t.Fatalf("emitPreviewDockAction should leave preview_action editor-operation telemetry to selection runtime:\n%s", body)
	}
	if strings.Contains(body, `"gosxstudio:selection-action"`) {
		t.Fatalf("emitPreviewDockAction should not dispatch selection-action telemetry:\n%s", body)
	}
}

func TestWorkbenchRuntimeLeavesSelectionActionsToSelectionRuntime(t *testing.T) {
	script := string(WorkbenchRuntimeScript())
	for _, forbidden := range []string{
		`function runSelectionAction(button)`,
		`function runSelectionTarget(target, label)`,
		`detail.kind === "selection-action"`,
		`function runInsert(button)`,
		`function runInsertTarget`,
		`detail.kind === "insert"`,
		`event.target.closest("[data-studio-selection-action]")`,
		`[data-studio-selection-action="`,
		`[data-studio-insert-block]`,
	} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("workbench runtime should not own selection/insert bridge fragment %q", forbidden)
		}
	}
	for _, check := range []string{
		`var mode = event.target.closest("[data-studio-mode-control]")`,
		`var viewport = event.target.closest("[data-studio-viewport]")`,
		`var zoom = event.target.closest("button[data-studio-zoom], [role='button'][data-studio-zoom]")`,
		`var rail = event.target.closest("[data-studio-rail-toggle]")`,
		`if (detail.kind === "mode")`,
		`if (detail.kind === "viewport")`,
		`if (detail.kind === "zoom")`,
		`if (detail.kind === "toggle")`,
	} {
		if !strings.Contains(script, check) {
			t.Fatalf("workbench runtime missing non-selection chrome fragment %q", check)
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
