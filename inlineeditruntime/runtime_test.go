package inlineeditruntime

import (
	"strings"
	"testing"
)

// TestInlineEditRuntimeFeatureFlagKey verifies the Phase 3 naming convention.
func TestInlineEditRuntimeFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	if !strings.HasSuffix(FeatureFlagKey, "-runtime-islands") {
		t.Fatalf("FeatureFlagKey %q must end with -runtime-islands per Phase 3 convention", FeatureFlagKey)
	}
}

// TestInlineEditRuntimeScriptIsNonEmpty checks the embedded JS is non-empty.
func TestInlineEditRuntimeScriptIsNonEmpty(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	if body == "" {
		t.Fatal("InlineEditRuntimeScript() must return a non-empty JS snippet")
	}
}

// TestInlineEditRuntimePublishesGlobal checks the IIFE publishes the expected
// window global with the install method.
func TestInlineEditRuntimePublishesGlobal(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	if !strings.Contains(body, "window.GoSXStudioInlineEditRuntime") {
		t.Fatalf("InlineEditRuntimeScript() must publish window.GoSXStudioInlineEditRuntime:\n%s", body)
	}
	if !strings.Contains(body, "install") {
		t.Fatalf("InlineEditRuntimeScript() must expose an install method:\n%s", body)
	}
}

// TestInlineEditRuntimeInstallIsIdempotent verifies the install-guard flag is
// present in the source so a second call to install() on the same root is
// a no-op.
func TestInlineEditRuntimeInstallIsIdempotent(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	// The guard attribute must be referenced in the JS.
	if !strings.Contains(body, "INSTALLED_FLAG") && !strings.Contains(body, "__gosxInlineEditInstalled") {
		t.Fatalf("InlineEditRuntimeScript() must include an idempotency guard against double-install:\n%s", body)
	}
}

// TestInlineEditRuntimeEditableFieldContract verifies the data-studio-field
// + contenteditable delegate target contract.
func TestInlineEditRuntimeEditableFieldContract(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	if !strings.Contains(body, "data-studio-field") {
		t.Fatalf("InlineEditRuntimeScript() must reference data-studio-field attribute:\n%s", body)
	}
	if !strings.Contains(body, "contenteditable") {
		t.Fatalf("InlineEditRuntimeScript() must check contenteditable attribute:\n%s", body)
	}
}

// TestInlineEditRuntimeKeyDerivation verifies the deriveKeys function is
// present and covers the leading-"pages"-drop and 3-segment tail contract.
func TestInlineEditRuntimeKeyDerivation(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	if !strings.Contains(body, "deriveKeys") {
		t.Fatalf("InlineEditRuntimeScript() must implement deriveKeys:\n%s", body)
	}
	// The function must split on "." and handle the "pages" prefix.
	if !strings.Contains(body, `"pages"`) {
		t.Fatalf("InlineEditRuntimeScript() deriveKeys must drop a leading \"pages\" segment:\n%s", body)
	}
	// The three output keys.
	for _, key := range []string{"page", "component", "control"} {
		if !strings.Contains(body, key) {
			t.Fatalf("InlineEditRuntimeScript() deriveKeys must produce %q key:\n%s", key, body)
		}
	}
}

// TestInlineEditRuntimeCommitBehavior checks the commit-on-blur / Enter logic
// and the duplicate-POST guard (rebase baseline after commit).
func TestInlineEditRuntimeCommitBehavior(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	// Must handle focusin/focusout/keydown.
	for _, event := range []string{"focusin", "focusout", "keydown"} {
		if !strings.Contains(body, event) {
			t.Fatalf("InlineEditRuntimeScript() must handle %q event:\n%s", event, body)
		}
	}
	// Enter key guard.
	if !strings.Contains(body, `"Enter"`) {
		t.Fatalf("InlineEditRuntimeScript() must handle Enter key:\n%s", body)
	}
	// Must POST only when value changed.
	if !strings.Contains(body, "prev") && !strings.Contains(body, "changed") {
		t.Fatalf("InlineEditRuntimeScript() must guard against posting unchanged values:\n%s", body)
	}
}

// TestInlineEditRuntimeEscapeCancelsWithoutCommit verifies the #5 fix: Escape
// during a direct canvas-surface inline edit reverts the in-progress value to
// its pre-edit baseline and ends editing WITHOUT going through commit() (no
// POST), unlike Enter/blur which intentionally still commit.
func TestInlineEditRuntimeEscapeCancelsWithoutCommit(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"function cancelEdit(el)",
		`if (ev.key === "Escape") {`,
		"cancelEdit(el);",
		"var original = startOf(el);",
		"if (original === undefined) return;",
		"if (textOf(el) !== original) el.textContent = original;",
		"clearStart(el);",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() missing Escape-cancel fragment %q:\n%s", fragment, body)
		}
	}
	// Escape must be handled ahead of the Enter-commit branch in the same
	// keydown listener, and must return before falling into it.
	escapeIndex := strings.Index(body, `if (ev.key === "Escape") {`)
	enterIndex := strings.Index(body, `if (ev.key !== "Enter" || ev.shiftKey === true) return;`)
	if escapeIndex < 0 || enterIndex < 0 || escapeIndex >= enterIndex {
		t.Fatalf("InlineEditRuntimeScript() must check Escape before the Enter-commit guard in install()'s keydown listener:\n%s", body)
	}
}

// TestInlineEditRuntimePOSTBody verifies the expected form field names that
// mirror the server's AuthoringMutationFromForm contract.
func TestInlineEditRuntimePOSTBody(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, field := range []string{
		"gosx_studio_operation",
		"gosx_studio_binding",
		"gosx_studio_value",
		"gosx_studio_page_key",
		"gosx_studio_component_key",
		"gosx_studio_control_key",
	} {
		if !strings.Contains(body, field) {
			t.Fatalf("InlineEditRuntimeScript() POST body must include %q:\n%s", field, body)
		}
	}
	// update-page operation support.
	if !strings.Contains(body, "update-page") {
		t.Fatalf("InlineEditRuntimeScript() must support update-page operation:\n%s", body)
	}
	// save-control default operation.
	if !strings.Contains(body, "save-control") {
		t.Fatalf("InlineEditRuntimeScript() must support save-control operation:\n%s", body)
	}
}

// TestInlineEditRuntimeActionResolutionOrder verifies the four-step route
// resolution is documented and the markers are present so muddy can rely on
// it without reading internal source.
func TestInlineEditRuntimeActionResolutionOrder(t *testing.T) {
	body := string(InlineEditRuntimeScript())

	// Step 1: opts.action / opts.csrfToken
	if !strings.Contains(body, "opts.action") {
		t.Fatalf("InlineEditRuntimeScript() must accept opts.action (resolution step 1):\n%s", body)
	}
	if !strings.Contains(body, "opts.csrfToken") {
		t.Fatalf("InlineEditRuntimeScript() must accept opts.csrfToken (resolution step 1):\n%s", body)
	}

	// Step 2: data-gosx-studio-inline-edit-action / data-gosx-studio-inline-edit-csrf attributes on root.
	if !strings.Contains(body, "data-gosx-studio-inline-edit-action") {
		t.Fatalf("InlineEditRuntimeScript() must read data-gosx-studio-inline-edit-action from root (step 2):\n%s", body)
	}
	if !strings.Contains(body, "data-gosx-studio-inline-edit-csrf") {
		t.Fatalf("InlineEditRuntimeScript() must read data-gosx-studio-inline-edit-csrf from root (step 2):\n%s", body)
	}

	// Step 3: managed authoring form in the document.
	if !strings.Contains(body, "data-gosx-studio-authoring-managed") {
		t.Fatalf("InlineEditRuntimeScript() must check data-gosx-studio-authoring-managed form (step 3):\n%s", body)
	}

	// Step 4: window.location.href fallback.
	if !strings.Contains(body, "window.location.href") {
		t.Fatalf("InlineEditRuntimeScript() must fall back to window.location.href (step 4):\n%s", body)
	}
}

// TestInlineEditRuntimeNoHardcodedMuddyRoute asserts the muddy-specific
// /admin/editor/__actions/authoring path is NOT hardcoded without an override
// — i.e., it does not appear as an unoverridable constant.
func TestInlineEditRuntimeNoHardcodedMuddyRoute(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	// The muddy default route must not be baked in as a constant.
	// The file should not contain the muddy-specific literal at all.
	if strings.Contains(body, "/admin/editor/__actions/authoring") {
		t.Fatalf("InlineEditRuntimeScript() must NOT hardcode the muddy route /admin/editor/__actions/authoring; use configurable resolution:\n%s", body)
	}
}

// TestInlineEditRuntimeCSRFResolution verifies the CSRF token is resolved and
// attached as an X-CSRF-Token header on the POST.
func TestInlineEditRuntimeCSRFResolution(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	if !strings.Contains(body, "X-CSRF-Token") {
		t.Fatalf("InlineEditRuntimeScript() must set X-CSRF-Token header on POST:\n%s", body)
	}
}

// TestInlineEditRuntimeAuthoringRuntimeIntegration verifies that after a
// successful POST, the script hands off to window.GoSXStudioAuthoringRuntime
// .handleResult when present.
func TestInlineEditRuntimeAuthoringRuntimeIntegration(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	if !strings.Contains(body, "GoSXStudioAuthoringRuntime") {
		t.Fatalf("InlineEditRuntimeScript() must integrate with window.GoSXStudioAuthoringRuntime:\n%s", body)
	}
	if !strings.Contains(body, "handleResult") {
		t.Fatalf("InlineEditRuntimeScript() must call handleResult on GoSXStudioAuthoringRuntime:\n%s", body)
	}
}

// TestInlineEditRuntimeOnCommitHook verifies the optional opts.onCommit hook
// is invoked before the POST so muddy can supply repaint-safe behaviour.
func TestInlineEditRuntimeOnCommitHook(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	if !strings.Contains(body, "opts.onCommit") {
		t.Fatalf("InlineEditRuntimeScript() must expose opts.onCommit hook:\n%s", body)
	}
}

func TestInlineEditRuntimePublishesPreviewTextSessionMethods(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"startPreviewTextEdit",
		"syncPreviewTextEdit",
		"finishPreviewTextEdit",
		"startPreviewTextSession",
		"syncPreviewTextSession",
		"finishPreviewTextSession",
		"handlePreviewTextInputEvent",
		"handlePreviewTextKeyEvent",
		"handlePreviewTextPasteEvent",
		"handlePreviewTextBlurEvent",
		"previewTextEditState",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() must expose preview text session method %q:\n%s", fragment, body)
		}
	}
}

func TestInlineEditRuntimeOwnsPreviewTextDocumentEventPolicy(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"function previewTextEventEdit(frame, event)",
		"function handlePreviewTextInputEvent(frame, event, host)",
		`syncPreviewTextSession(frame, "input", host)`,
		"function handlePreviewTextKeyEvent(frame, event, host)",
		`event.key === "Escape"`,
		`finishPreviewTextSession(frame, false, "escape", host)`,
		`event.key === "Enter" && !event.shiftKey`,
		`finishPreviewTextSession(frame, true, "enter", host)`,
		"function handlePreviewTextPasteEvent(frame, event, host)",
		`event.clipboardData && event.clipboardData.getData`,
		`event.clipboardData.getData("text/plain")`,
		"var doc = frameDocument(frame)",
		"selection.deleteFromDocument()",
		"selection.getRangeAt(0).insertNode(doc.createTextNode(text))",
		"selection.collapseToEnd()",
		`syncPreviewTextSession(frame, "paste", host)`,
		"function handlePreviewTextBlurEvent(frame, event, host)",
		`finishPreviewTextSession(frame, true, "blur", host)`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() missing preview inline-text document event fragment %q:\n%s", fragment, body)
		}
	}
}

func TestInlineEditRuntimeOwnsPreviewTextSessionLifecycle(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"frame.__gosxStudioInlineEdit",
		`detail.editable !== "text"`,
		`frame.__gosxStudioPreviewDockTarget`,
		`target.setAttribute("contenteditable", "plaintext-only")`,
		`hadHref: target.hasAttribute && target.hasAttribute("href")`,
		`originalHref: target.hasAttribute && target.hasAttribute("href") ? target.getAttribute("href") || "" : ""`,
		`if (edit.hadHref) target.removeAttribute("href")`,
		`if (edit.hadHref) edit.target.setAttribute("href", edit.originalHref)`,
		`target.setAttribute("spellcheck", "true")`,
		`target.setAttribute("data-gosx-studio-inline-editing", "true")`,
		`target.removeAttribute("contenteditable")`,
		`target.removeAttribute("data-gosx-studio-inline-editing")`,
		`data-gosx-studio-inline-field`,
		"placeCaretAtEnd",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() must own preview inline-text lifecycle fragment %q:\n%s", fragment, body)
		}
	}
}

func TestInlineEditRuntimePreviewTextOperationsAndEvents(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"inline_text_start",
		"set_text",
		"inline_text_cancel",
		"gosxstudio:inline-text-start",
		"gosxstudio:inline-text",
		"gosxstudio:inline-text-commit",
		"gosxstudio:inline-text-cancel",
		"inlineTextPayload",
		"inlineTextEventDetail",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() missing preview inline-text operation/event fragment %q:\n%s", fragment, body)
		}
	}
}

func TestInlineEditRuntimePreviewTextCallbacks(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"opts.form",
		"opts.controlForField",
		"opts.selection",
		"opts.setDirty",
		"opts.emitOperation",
		"opts.emitEditorOperation",
		"opts.emitEvent",
		"opts.onFinish",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() missing preview inline-text callback hook %q:\n%s", fragment, body)
		}
	}
}

func TestInlineEditRuntimePreviewTextRequiresBackingControl(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"function valueBearingPreviewTextControl(control)",
		`if (!control || !("value" in control) || control.disabled) return null;`,
		`if (tag === "input" && type === "hidden") return null;`,
		`return valueBearingPreviewTextControl(control);`,
		`var control = previewTextControl(detail.field, opts);`,
		`if (!control) return false;`,
		`target.setAttribute("contenteditable", "plaintext-only");`,
		`target.setAttribute("data-gosx-studio-inline-editing", "true");`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() missing preview text backing-control fragment %q:\n%s", fragment, body)
		}
	}
	controlIdx := strings.Index(body, `var control = previewTextControl(detail.field, opts);`)
	gateIdx := strings.Index(body, `if (!control) return false;`)
	contentEditableIdx := strings.Index(body, `target.setAttribute("contenteditable", "plaintext-only");`)
	if controlIdx < 0 || gateIdx < 0 || contentEditableIdx < 0 || !(controlIdx < gateIdx && gateIdx < contentEditableIdx) {
		t.Fatalf("InlineEditRuntimeScript() must reject unbacked preview text before setting contenteditable:\n%s", body)
	}
}

func TestInlineEditRuntimeOwnsPreviewTextSessionAdapter(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"function previewTextSessionOptions(frame, host)",
		"form: host.form || null",
		"target: host.target || (frame && frame.__gosxStudioPreviewDockTarget)",
		"controlForField: host.controlForField",
		`form.getAttribute("data-studio-selection")`,
		`form.getAttribute("data-studio-selection-kind")`,
		"selection: selection || edit.blockKey || edit.field || \"\"",
		`host.setStatus("dirty", "Unsaved edit", reason || "inline-text")`,
		"host.setDirty(reason || \"inline-text\")",
		"typeof host.emitOperation === \"function\"",
		"host.emitOperation(type, operation)",
		"typeof host.emitEditorOperation === \"function\"",
		"host.emitEditorOperation(type, operation)",
		"host.emitEvent(name, detail)",
		"typeof host.onFinish === \"function\"",
		"host.onFinish(finishedFrame, edit, reason, commit)",
		"typeof host.updateDock === \"function\"",
		"host.updateDock(finishedFrame, edit, reason, commit)",
		"startPreviewTextEdit(frame, detail, reason, previewTextSessionOptions(frame, host))",
		"syncPreviewTextEdit(frame, reason, previewTextSessionOptions(frame, host))",
		"finishPreviewTextEdit(frame, commit, reason, previewTextSessionOptions(frame, host))",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() missing preview text session adapter fragment %q:\n%s", fragment, body)
		}
	}
}

// TestInlineEditRuntimeNoPersistRepaintSafeLogic verifies that muddy-specific
// bundle-rewriting is NOT ported into the generic island. The prohibition is
// documented in a header comment (it is fine for the word to appear there), but
// the JS must not contain function declarations or calls that implement it.
func TestInlineEditRuntimeNoPersistRepaintSafeLogic(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	// persistRepaintSafe must not be implemented as a function.
	if strings.Contains(body, "function persistRepaintSafe") {
		t.Fatalf("InlineEditRuntimeScript() must NOT implement persistRepaintSafe; use opts.onCommit instead:\n%s", body)
	}
	// Must not reference muddy canvas-board runtime attributes used only by persistRepaintSafe.
	for _, muddySpecific := range []string{
		"__muddyInlineEditInstalled",
		"data-gosx-canvas-bundle",
		"legacyCanvasInlineEditAlias",
		"PAINTER_CACHE",
		"__gosxHTMLMarkup",
	} {
		if strings.Contains(body, muddySpecific) {
			t.Fatalf("InlineEditRuntimeScript() must NOT reference muddy-specific symbol %q:\n%s", muddySpecific, body)
		}
	}
}

// TestInlineEditRuntimeDurableDispatch guards handoff-31 ("one product
// path"): a contenteditable field the host opts into
// (data-gosx-studio-durable-history="true") must submit through
// window.GoSXStudioCollaborationRuntime when it is available/synced, using
// the same OperationRequest wire shape operationruntime/island_runtime.js's
// direct-edit forms build for kind "set-field" — so the same server-side
// decode path, OperationRecord log, change-set, and undo/redo machinery
// apply regardless of which client island produced the request.
func TestInlineEditRuntimeDurableDispatch(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	for _, fragment := range []string{
		"data-gosx-studio-durable-history",
		"data-gosx-studio-durable-field",
		"data-gosx-studio-durable-page-id",
		"data-gosx-studio-durable-route",
		"data-gosx-studio-durable-component",
		"function durableMetaFromElement(el, binding)",
		"function durableOperationRequest(durable, value)",
		`kind: "set-field"`,
		"window.GoSXStudioCollaborationRuntime",
		"collaboration.available()",
		"collaboration.submit(request)",
		"collaboration.expectedHead",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("InlineEditRuntimeScript() missing durable dispatch fragment %q:\n%s", fragment, body)
		}
	}
}

// TestInlineEditRuntimeDurablePathRetiresLegacyPostForThatCommit is the
// absence assertion the durable unification requires: when the durable
// collaboration path is taken, commit() must `return` before ever reaching
// the legacy resolveAction/doFetch POST — i.e. exactly one write per commit,
// never both. This locks the control-flow shape (early return immediately
// after collaboration.submit(...)), not just the presence of both code
// paths somewhere in the file.
func TestInlineEditRuntimeDurablePathRetiresLegacyPostForThatCommit(t *testing.T) {
	body := string(InlineEditRuntimeScript())
	durableIdx := strings.Index(body, "var collaborationReady = durable.enabled")
	returnIdx := strings.Index(body, "return; // Durable path taken: the legacy POST below never fires for this commit.")
	actionIdx := strings.Index(body, "var action = resolveAction(root, opts.action);")
	if durableIdx < 0 || returnIdx < 0 || actionIdx < 0 {
		t.Fatalf("InlineEditRuntimeScript() must contain the durable-path early return before the legacy POST:\n%s", body)
	}
	if !(durableIdx < returnIdx && returnIdx < actionIdx) {
		t.Fatalf("InlineEditRuntimeScript() durable dispatch must return BEFORE resolving the legacy POST action, so a durable commit never also fires the legacy write:\n%s", body)
	}
}

// TestBundleIsNonEmpty verifies Bundle() produces the same non-empty result.
func TestBundleIsNonEmpty(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	if !strings.Contains(bundle, "window.GoSXStudioInlineEditRuntime") {
		t.Fatalf("Bundle() must contain the inline-edit runtime global:\n%s", bundle)
	}
}
