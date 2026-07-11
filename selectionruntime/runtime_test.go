package selectionruntime

import (
	"strings"
	"testing"
)

// The selectionruntime package is the Go-side surface for the Phase 3 slice-2
// burn-down of GoSXStudioSelectionRuntime. It owns:
//   1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//      or emit as a host probe attribute. The binding path is always the
//      .gsx-authored island in this package.
//   2. The JS shim that installs window.GoSXStudioSelectionRuntime and
//      delegates its methods directly to the island.
//   3. The island runtime JS that publishes window.__gosx_selection_runtime_*
//      globals the BridgeShim delegates to.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-2-selectionruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the cross-frame signal channel any cross-iframe sub-behaviors write through.

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if FeatureFlagKey != "selection-runtime-islands" {
		t.Fatalf("FeatureFlagKey must be %q per Phase 3 slice-2 contract; got %q", "selection-runtime-islands", FeatureFlagKey)
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
	// The shim must reference the global the island publishes itself at,
	// must reference window.GoSXStudioSelectionRuntime (the public global it
	// shims), and must install bind as a direct delegate.
	for _, fragment := range []string{
		"window.GoSXStudioSelectionRuntime",
		"__gosx_selection_runtime_island_bind",
		"function delegate(islandGlobal)",
		"return island.apply(null, arguments);",
		`bind: delegate("__gosx_selection_runtime_island_bind")`,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
	for _, fragment := range []string{
		"flag" + "Enabled",
		"FLAG" + "_ATTR",
		"bind" + "SelectionSurface",
		"fallback" + " branch",
		"fallback" + " path",
		"legacy" + " bundle",
	} {
		if strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() must not contain stale selection gate/fallback fragment %q:\n%s", fragment, shim)
		}
	}
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

func TestIslandRuntimeJSPublishesIslandGlobal(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	// The global published here is the delegation target BridgeShim consults.
	if !strings.Contains(body, "window.__gosx_selection_runtime_island_bind ") {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", "window.__gosx_selection_runtime_island_bind ")
	}
	// The island runtime must preserve the selection DOM attribute contract:
	// selection-action commandbar, workspace targets, field-source focus,
	// style-scope readouts, and the idempotency-guard dataset key. Each entry
	// below maps to one of the five sub-behaviors enumerated in
	// selection_bind.gsx; if a fragment is missing the corresponding
	// sub-behavior is silently broken.
	for _, contract := range []string{
		// Sub-behavior 1: block selection
		"data-block-studio-block",
		"blockstudio:select",
		// Sub-behavior 2: workspace target selection
		"data-studio-panel-link",
		"data-studio-site-page",
		"data-studio-workspace-selection",
		// Sub-behavior 3: field focus
		"data-studio-field-source",
		"studio:field-select",
		"data-studio-field-selection",
		// Sub-behavior 4: selection commandbar
		"data-studio-selection-action",
		"data-studio-selection-status",
		// Sub-behavior 5: style-scope readouts
		"data-studio-style-scope",
		"data-studio-style-state",
		"data-studio-style-breakpoint",
		// Idempotency guard — mirrors legacy gosxStudioSelectionBound key
		"gosxStudioSelectionIslandBound",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() selection contract missing %q", contract)
		}
	}
}

func TestSelectionRuntimeIslandOwnsSelectionActionTelemetry(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`function selectionActionLabel(action, labelOverride)`,
		`|| labelOverride || action || ""`,
		`function emitSelectionAction(action, labelOverride)`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:selection-action", { bubbles: true, detail: detail }))`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:workbench-action", { bubbles: true, detail: detail }))`,
		`action: action || ""`,
		`label: selectionActionLabel(action, labelOverride)`,
		`selection: form.getAttribute("data-studio-selection") || form.getAttribute("data-gosx-studio-canvas-selected") || ""`,
		`kind: form.getAttribute("data-studio-selection-kind") || ""`,
		`button.getAttribute("aria-label") || button.getAttribute("title") || button.textContent`,
		`function runSelectionAction(action, labelOverride)`,
		`emitSelectionAction(action, labelOverride);`,
		`runSelectionAction(target, detail.label);`,
		`if (event.preventDefault) event.preventDefault();`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing selection-action telemetry contract %q", contract)
		}
	}
}

func TestSelectionRuntimeIslandOwnsInsertBlockDispatch(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`function insertActionLabel(button, fallback)`,
		`function runInsert(button)`,
		`function runInsertTarget(target, label)`,
		`button.getAttribute("data-studio-insert-block") || button.getAttribute("data-editor-add-block") || ""`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:insert-block", { bubbles: true, detail: detail }))`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:workbench-action", {`,
		`detail: { action: "insert-block", target: target, label: detail.label }`,
		`var add = target ? form.querySelector('[data-editor-add-block="' + attrValue(target) + '"]') : null;`,
		`if (add && add !== button && add.click) add.click();`,
		`form.querySelector('[data-studio-insert-block="' + attrValue(target) + '"], [data-editor-add-block="' + attrValue(target) + '"]')`,
		`detail: { target: target, label: label || target }`,
		`detail: { action: "insert-block", target: target, label: label || target }`,
		`if (kind === "insert" && runInsertTarget(detail.target, detail.label)) {`,
		`runInsertTarget(detail.target, detail.label)`,
		`var insert = event.target.closest && event.target.closest("[data-studio-insert-block]");`,
		`runInsert(insert);`,
		`if (event.preventDefault) event.preventDefault();`,
		`event.preventDefault();`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing insert-block dispatch contract %q", contract)
		}
	}
}

func TestSelectionRuntimeIslandMirrorsPreviewActionSelectionTelemetry(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-action", mirrorPreviewActionSelection);`,
		`function mirrorPreviewActionSelection(event)`,
		`function previewDockActionEnvelope(action, detail)`,
		`actionLabel: detail.actionLabel || detail.action || ""`,
		`var envelope = event.detail || {};`,
		`var detail = previewDockActionEnvelope(envelope.action || "", envelope.detail || envelope);`,
		`var reason = envelope.reason || "preview-dock";`,
		`selectionOperationCounter += 1;`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-operation", {`,
		`id: "studio-op-" + Date.now() + "-" + selectionOperationCounter`,
		`kind: "preview_action"`,
		`source: "gosx-studio"`,
		`mutation: false`,
		`reason: reason`,
		`field: detail.field || ""`,
		`editable: detail.editable || ""`,
		`blockKey: detail.blockKey || ""`,
		`selection: form.getAttribute("data-studio-selection") || detail.blockKey || detail.field || ""`,
		`kind: form.getAttribute("data-studio-selection-kind") || "preview"`,
		`payload: {`,
		`action: detail.action || ""`,
		`label: detail.label || ""`,
		`blockLabel: detail.blockLabel || ""`,
		`actionLabel: detail.actionLabel || ""`,
		`actionHref: detail.actionHref || ""`,
		`actionFormAction: detail.actionFormAction || ""`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:selection-action", { bubbles: true, detail: {`,
		`action: detail.action || ""`,
		`label: detail.actionLabel || detail.label || detail.action || ""`,
		`selection: form.getAttribute("data-studio-selection") || detail.blockKey || detail.field || ""`,
		`kind: form.getAttribute("data-studio-selection-kind") || "preview"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview-action selection mirror contract %q", contract)
		}
	}
	mirrorBody := jsFunctionBody(t, body, "mirrorPreviewActionSelection")
	for _, forbidden := range []string{
		`gosxstudio:workbench-action`,
		`gosxstudio:preview-action`,
		`gosxstudio:selection-action", mirrorPreviewActionSelection`,
	} {
		if strings.Contains(mirrorBody, forbidden) {
			t.Fatalf("preview-action mirror should only dispatch selection-action telemetry; found %q in:\n%s", forbidden, mirrorBody)
		}
	}
}

func TestSelectionRuntimeIslandOwnsPreviewFieldNavigationTelemetry(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-field-navigation-commit", emitPreviewFieldNavigation);`,
		`function emitPreviewFieldNavigation(event)`,
		`function previewFieldNavigationEnvelope(detail, navigation)`,
		`reason: navigation.reason || "field-navigation"`,
		`var normalized = previewFieldNavigationEnvelope(envelope.detail || envelope, envelope.navigation || {});`,
		`var detail = normalized.detail;`,
		`var navigation = normalized.navigation;`,
		`selectionOperationCounter += 1;`,
		`id: "studio-op-" + Date.now() + "-" + selectionOperationCounter`,
		`kind: "preview_field_navigate"`,
		`source: "gosx-studio"`,
		`mutation: false`,
		`selection: form.getAttribute("data-studio-selection") || field`,
		`kind: form.getAttribute("data-studio-selection-kind") || "preview-field"`,
		`payload: {`,
		`direction: direction`,
		`fieldIndex: fieldIndex`,
		`fieldCount: fieldCount`,
		`label: label`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-navigate", {`,
		`field: field`,
		`editable: editable`,
		`blockKey: blockKey`,
		`reason: reason`,
		`envelope.result = { emitted: true };`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview field navigation telemetry contract %q", contract)
		}
	}

	handlerBody := jsFunctionBody(t, body, "emitPreviewFieldNavigation")
	for _, contract := range []string{
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-operation", {`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-field-navigate", {`,
		`field: field`,
		`editable: editable`,
		`label: label`,
		`blockKey: blockKey`,
		`direction: direction`,
		`fieldIndex: fieldIndex`,
		`fieldCount: fieldCount`,
		`reason: reason`,
	} {
		if !strings.Contains(handlerBody, contract) {
			t.Fatalf("preview field navigation handler missing public telemetry fragment %q in:\n%s", contract, handlerBody)
		}
	}
}

func TestSelectionRuntimeIslandOwnsPreviewFieldActionResolve(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-field-action-resolve", resolvePreviewFieldAction);`,
		`function resolvePreviewFieldAction(event)`,
		`function previewFieldActionEnvelope(detail)`,
		`var envelope = event.detail || {};`,
		`var detail = previewFieldActionEnvelope(envelope.detail || envelope);`,
		`var field = envelope.field || "";`,
		`var editable = envelope.editable || "";`,
		`var label = envelope.label || "";`,
		`var blockKey = envelope.blockKey || "";`,
		`var action = envelope.action || "";`,
		`var actionHref = envelope.actionHref || "";`,
		`var actionFormAction = envelope.actionFormAction || "";`,
		`field = detail.field;`,
		`editable = detail.editable;`,
		`label = detail.label;`,
		`blockKey = detail.blockKey;`,
		`action = detail.action;`,
		`actionHref = detail.actionHref;`,
		`actionFormAction = detail.actionFormAction;`,
		`inlineText: editable === "text"`,
		`submit: !!actionFormAction`,
		`navigate: !!actionHref`,
		`href: actionHref || ""`,
		`reveal: !!field`,
		`field: field`,
		`editable: editable`,
		`label: label`,
		`blockKey: blockKey`,
		`action: action`,
		`actionHref: actionHref`,
		`actionFormAction: actionFormAction`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview field-action resolve contract %q", contract)
		}
	}

	handlerBody := jsFunctionBody(t, body, "resolvePreviewFieldAction")
	for _, forbidden := range []string{
		`form.dispatchEvent`,
		`gosxstudio:preview-action`,
		`gosxstudio:selection-action`,
		`gosxstudio:editor-operation`,
	} {
		if strings.Contains(handlerBody, forbidden) {
			t.Fatalf("preview field-action resolve should only set envelope.result; found %q in:\n%s", forbidden, handlerBody)
		}
	}
}

func TestSelectionRuntimeIslandOwnsPreviewFieldActionSubmit(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-field-action-submit", submitPreviewFieldAction);`,
		`function isFormSubmitControl(node)`,
		`function fieldActionSubmitter(source, formAction)`,
		`function submitPreviewFieldAction(event)`,
		`var envelope = event.detail || {};`,
		`var detail = previewFieldActionEnvelope(envelope.detail || envelope);`,
		`envelope.result = false;`,
		`var source = fieldSource(detail.field || "");`,
		`var submitter = fieldActionSubmitter(source, detail.actionFormAction || "");`,
		`var action = detail.actionFormAction || (submitter && (submitter.getAttribute("data-studio-field-action-formaction") || submitter.getAttribute("formaction"))) || "";`,
		`var confirmMessage = (submitter && submitter.getAttribute("data-admin-confirm")) || (source && source.getAttribute("data-admin-confirm")) || "";`,
		`if (confirmMessage && !window.confirm(confirmMessage)) return;`,
		`form.dataset.gosxStudioPendingAction = action;`,
		`form.dataset.gosxStudioPendingActionLabel = detail.action || detail.label || "Field action";`,
		`if (submitter && form.requestSubmit)`,
		`form.requestSubmit(submitter);`,
		`var button = document.createElement("button");`,
		`button.type = "submit";`,
		`button.hidden = true;`,
		`button.setAttribute("formaction", action);`,
		`form.requestSubmit(button);`,
		`form.setAttribute("action", action);`,
		`form.submit();`,
		`if (!submitter || !submitter.click) return;`,
		`submitter.click();`,
		`envelope.result = true;`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview field-action submit contract %q", contract)
		}
	}

	handlerBody := jsFunctionBody(t, body, "submitPreviewFieldAction")
	for _, forbidden := range []string{
		`form.dispatchEvent`,
		`gosxstudio:preview-action`,
		`gosxstudio:selection-action`,
		`gosxstudio:editor-operation`,
	} {
		if strings.Contains(handlerBody, forbidden) {
			t.Fatalf("preview field-action submit should only execute the submit envelope; found %q in:\n%s", forbidden, handlerBody)
		}
	}
}

func TestSelectionRuntimeIslandOwnsPreviewDockActionResolve(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-dock-action-resolve", resolvePreviewDockAction);`,
		`function resolvePreviewDockAction(event)`,
		`function previewDockActionEnvelope(action, detail)`,
		`var envelope = event.detail || {};`,
		`var detail = previewDockActionEnvelope(envelope.action || "", envelope.detail || envelope);`,
		`var action = detail.action || "";`,
		`var field = detail.field || "";`,
		`var editable = detail.editable || "";`,
		`var label = detail.label || "";`,
		`var blockKey = detail.blockKey || "";`,
		`var blockLabel = detail.blockLabel || "";`,
		`var actionLabel = detail.actionLabel || "";`,
		`var actionHref = detail.actionHref || "";`,
		`var actionFormAction = detail.actionFormAction || "";`,
		`mode: action === "content" ? "content" : action === "style" ? "style" : ""`,
		`reveal: action === "content" && !!field`,
		`action: action`,
		`field: field`,
		`editable: editable`,
		`label: label`,
		`blockKey: blockKey`,
		`blockLabel: blockLabel`,
		`actionLabel: actionLabel`,
		`actionHref: actionHref`,
		`actionFormAction: actionFormAction`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview dock action resolve contract %q", contract)
		}
	}

	handlerBody := jsFunctionBody(t, body, "resolvePreviewDockAction")
	for _, forbidden := range []string{
		`form.dispatchEvent`,
		`gosxstudio:preview-action`,
		`gosxstudio:selection-action`,
		`gosxstudio:editor-operation`,
	} {
		if strings.Contains(handlerBody, forbidden) {
			t.Fatalf("preview dock action resolve should only set envelope.result; found %q in:\n%s", forbidden, handlerBody)
		}
	}
}

func TestSelectionRuntimeIslandOwnsPreviewSelectionApply(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-selection-detail-resolve", resolvePreviewSelectionDetail);`,
		`function resolvePreviewSelectionDetail(event)`,
		`var node = envelope.target;`,
		`if (!node || !node.closest)`,
		`envelope.result = {};`,
		`var fieldNode = node.closest("[data-studio-field], [data-editor-preview], [data-studio-field-source]");`,
		`var blockNode = node.closest("[data-studio-block-key], [data-studio-node-id]");`,
		`fieldNode.getAttribute("data-studio-field") || fieldNode.getAttribute("data-editor-preview") || fieldNode.getAttribute("data-studio-field-source") || ""`,
		`fieldNode.getAttribute("data-studio-editable") || fieldNode.getAttribute("data-studio-field-editable") || ""`,
		`var source = fieldSource(field);`,
		`var control = fieldControl(source);`,
		`if (!editable) editable = inferPreviewEditableKind(source, control);`,
		`fieldNode.getAttribute("data-studio-field-label") || previewReadableFieldName(field, editable)`,
		`fieldNode.getAttribute("data-studio-field-action") || ""`,
		`fieldNode.getAttribute("data-studio-field-action-href") || ""`,
		`fieldNode.getAttribute("data-studio-field-action-formaction") || ""`,
		`blockNode.getAttribute("data-studio-block-key") || ""`,
		`blockNode.getAttribute("data-studio-node-id") || ""`,
		`var blockLabel = blockNode ? previewBlockLabel(blockNode, blockKey || nodeID || "") : "";`,
		`if (!label) label = blockLabel || "Preview selection";`,
		`source: field,`,
		`function previewReadableFieldName(field, editable)`,
		`if (editable === "media" || editable === "image") return value === "Media" ? value : value + " media";`,
		`if (editable === "source") return value + " source";`,
		`if (editable === "flow") return value + " flow";`,
		`if (editable === "url" || editable === "link") return value + " URL";`,
		`function previewBlockLabel(node, fallback)`,
		`node.getAttribute("data-studio-block-label") || node.getAttribute("data-studio-node-label") || node.getAttribute("aria-label") || ""`,
		`node.querySelector && node.querySelector("[data-studio-block-title], h1, h2, h3, strong")`,
		`form.addEventListener("gosxstudio:preview-selection-apply", applyPreviewSelectionState);`,
		`function applyPreviewSelectionState(event)`,
		`function inferPreviewEditableKind(source, control)`,
		`source.getAttribute("data-studio-field-editable") || source.getAttribute("data-studio-editable") || ""`,
		`if (tag === "input" || tag === "select") return "text";`,
		`return "";`,
		`var editable = detail.editable || inferPreviewEditableKind(source, control) || "";`,
		`clearPreviewInspectorSelection();`,
		`markPreviewInspectorSelection(source, control);`,
		`target.setAttribute("data-gosx-studio-inspector-selected", "true");`,
		`form.setAttribute("data-studio-field-selection", result.field);`,
		`form.setAttribute("data-studio-field-editable", result.editable);`,
		`form.setAttribute("data-studio-field-action-label", actionLabel);`,
		`form.setAttribute("data-studio-field-action-href", actionHref);`,
		`form.setAttribute("data-studio-field-action-formaction", actionFormAction);`,
		`form.setAttribute("data-studio-selection", result.selectionKey);`,
		`form.setAttribute("data-studio-selection-kind", result.kind);`,
		`setSelectionReadout(result.label || result.field || result.blockKey || "Preview selection");`,
		`setReadout("[data-studio-selection-status]", result.field ? "Preview field" : "Preview selection");`,
		`form.querySelectorAll("[data-studio-field-selection-label]")`,
		`envelope.result = result;`,
		`selectionOperationCounter += 1;`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:editor-operation", {`,
		`id: "studio-op-" + Date.now() + "-" + selectionOperationCounter`,
		`kind: "select_preview"`,
		`source: "gosx-studio"`,
		`mutation: false`,
		`reason: options.reason || "preview"`,
		`field: result.field || detail.field || ""`,
		`editable: result.editable || detail.editable || ""`,
		`blockKey: result.blockKey || detail.blockKey || ""`,
		`nodeID: result.nodeID || detail.nodeID || ""`,
		`selection: result.selectionKey || detail.blockKey || detail.nodeID || detail.field || ""`,
		`kind: result.kind || (detail.field ? "preview-field" : "preview")`,
		`label: result.label || detail.label || ""`,
		`blockLabel: result.blockLabel || detail.blockLabel || ""`,
		`action: result.action || ""`,
		`actionHref: result.actionHref || ""`,
		`actionFormAction: result.actionFormAction || ""`,
		`form.dispatchEvent(new CustomEvent("gosxstudio:preview-select"`,
		`actionFormAction: result.actionFormAction`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview-selection apply contract %q", contract)
		}
	}
}

func TestSelectionRuntimeIslandOwnsPreviewFieldTargetReveal(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-field-target-resolve", resolvePreviewFieldTarget);`,
		`form.addEventListener("gosxstudio:preview-field-reveal", revealPreviewField);`,
		`function previewFieldTarget(detail)`,
		`var field = typeof detail === "string" ? detail : (detail.field || (detail.detail && detail.detail.field) || "");`,
		`var source = fieldSource(field);`,
		`var control = fieldControl(source);`,
		`found: !!source`,
		`function revealPreviewFieldTarget(target)`,
		`var row = target.source.closest ? target.source.closest(".field-row, [data-studio-field-row]") : null;`,
		`var scrollTarget = row || target.source;`,
		`scrollTarget.scrollIntoView({ behavior: reduced ? "auto" : "smooth", block: "center" });`,
		`if (target.control && target.control.focus) target.control.focus({ preventScroll: true });`,
		`}, reduced ? 0 : 120);`,
		`function resolvePreviewFieldTarget(event)`,
		`envelope.result = previewFieldTarget(envelope.detail || envelope);`,
		`function revealPreviewField(event)`,
		`var target = previewFieldTarget(envelope.detail || envelope);`,
		`revealed: revealPreviewFieldTarget(target),`,
		`field: target.field`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview field target/reveal contract %q", contract)
		}
	}
}

func TestSelectionRuntimeIslandOwnsPreviewSelectionClear(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, contract := range []string{
		`form.addEventListener("gosxstudio:preview-selection-clear", clearPreviewSelectionState);`,
		`function clearPreviewSelectionState(event)`,
		`clearPreviewInspectorSelection();`,
		`target.removeAttribute("data-gosx-studio-inspector-selected");`,
		`target.classList.remove("is-studio-field-active", "is-preview-selected");`,
		`form.removeAttribute("data-studio-selection");`,
		`form.removeAttribute("data-studio-selection-kind");`,
		`form.removeAttribute("data-studio-field-selection");`,
		`form.removeAttribute("data-studio-field-editable");`,
		`form.removeAttribute("data-studio-field-action-label");`,
		`form.removeAttribute("data-studio-field-action-href");`,
		`form.removeAttribute("data-studio-field-action-formaction");`,
		`setSelectionReadout("No selection");`,
		`updateSelectionStatus(null);`,
		`readout.textContent = "Block";`,
		`updateFieldActionLabels();`,
		`updateStyleScope();`,
		`writeSharedSignal("$selection.fieldFocus", { field: "", editable: "", label: "" });`,
		`writeSharedSignal("$selection.block", { key: "", label: "" });`,
		`writeSharedSignal("$selection.workspaceTarget", { scope: "", key: "", label: "" });`,
		`envelope.result = { cleared: true };`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() missing preview-selection clear contract %q", contract)
		}
	}
}

func TestBridgeShimAutoMountsOnDocumentReady(t *testing.T) {
	// Pre-2026-05-27 the deleted studio-engines.js bundle auto-mounted every
	// runtime contract at DOMContentLoaded via its top-level init() function.
	// When the bundle was deleted the per-slice BridgeShim correctly re-
	// published window.GoSXStudioSelectionRuntime, but the auto-mount call
	// was lost — nothing invoked .bind(document) on page load anymore, so
	// none of the five selection sub-behaviors (block selection, workspace
	// target, field focus, commandbar, style-scope readouts) wired their
	// listeners on initial load.
	//
	// v0.5.0 restores the auto-mount inside the shim so the contract is
	// local to each slice and doesn't depend on the deleted global init()
	// coming back. Defer to DOMContentLoaded if the document is still
	// loading so we don't double-mount during SSR hydration.
	shim := string(BridgeShim())
	for _, fragment := range []string{
		"DOMContentLoaded",
		"window.GoSXStudioSelectionRuntime.bind",
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
	// the global the island runtime publishes. Order matters.
	islandIdx := strings.Index(bundle, "__gosx_selection_runtime_island_bind ")
	shimIdx := strings.Index(bundle, "window.GoSXStudioSelectionRuntime =")
	if islandIdx < 0 || shimIdx < 0 {
		t.Fatalf("Bundle() missing island runtime or shim:\n%s", bundle)
	}
	if islandIdx > shimIdx {
		t.Fatalf("Bundle() must place island runtime before shim (island=%d shim=%d)", islandIdx, shimIdx)
	}
}

func TestIslandRuntimePersistsOnlyRouteScopedSelectionLocator(t *testing.T) {
	script := string(IslandRuntimeJS())
	for _, fragment := range []string{
		`function selectionLocatorStorageKey(form, route)`,
		`window.location.origin + window.location.pathname`,
		`window.sessionStorage.setItem(selectionLocatorStorageKey(form, locator.route), JSON.stringify(locator))`,
		`route: normalizeSelectionLocatorRoute(detail.route || "/")`,
		`pageID: compactText(detail.pageID)`,
		`field: compactText(detail.field)`,
		`blockKey: compactText(detail.blockKey)`,
		`nodeID: compactText(detail.nodeID)`,
		`kind: compactText(detail.kind`,
		`function syncInspectorForExactSelection(key)`,
		`function pageNavigatorIdentityForRoute(route)`,
		`form.setAttribute("data-studio-preview-route-current", route)`,
		`syncPageNavigatorSelection(pageID, route)`,
		`syncInspectorForExactSelection(pageID)`,
		`gosxstudio:preview-selection-restore-request`,
		`selection-runtime-bound`,
		`gosxstudio:preview-selection-locator-restore`,
		`gosxstudio:preview-selection-locator-stale`,
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("selection locator runtime missing %q", fragment)
		}
	}
	for _, forbidden := range []string{`locator.content`, `locator.value`, `locator.html`, `locator.text`} {
		if strings.Contains(script, forbidden) {
			t.Fatalf("selection locator must not persist content via %q", forbidden)
		}
	}
}
