package authoringruntime

import (
	"strings"
	"testing"
)

func TestIslandRuntimeJSOwnsAuthoringResultFeedback(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	for _, fragment := range []string{
		"GoSXStudioAuthoringRuntime",
		"gosx:form:result",
		"gosxstudio:authoring-result",
		"data-gosx-studio-authoring-managed",
		"data-gosx-studio-authoring-selected",
		"data-gosx-studio-authoring-state",
		"data-gosx-studio-preview-url",
		"data-gosx-studio-canvas-node",
		"gosxstudio:canvas-select",
		"data-studio-site-map-component",
		"GoSXStudioPreviewRuntime",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing authoring feedback fragment %q", fragment)
		}
	}
}

func TestBundleReturnsAuthoringRuntime(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	if !strings.Contains(bundle, "window.GoSXStudioAuthoringRuntime") {
		t.Fatalf("Bundle() missing runtime global:\n%s", bundle)
	}
}

func TestManagedAuthoringFormsNeverFallThroughToNativeSubmit(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, want := range []string{
		`return true`,
		`submitAuthoringManagedForm(form, event.submitter || null)`,
		`setFormError(form)`,
		`credentials: "same-origin"`,
		`new FormData(form, submitter)`,
		`!capturedSubmitter`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("IslandRuntimeJS() missing no-reload managed submit fragment %q", want)
		}
	}
	for _, forbidden := range []string{
		`function formSubmitTarget`,
		`formSubmitTarget(form, submitter)`,
		`method !== "GET" && method !== "POST") return false`,
		`return isSameOrigin(formSubmissionAction(form, submitter) || window.location.href)`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("managed authoring form can still fall through to native submit via %q", forbidden)
		}
	}
}

func TestManagedAuthoringRefreshPreservesFocusAndLatestResponse(t *testing.T) {
	body := string(IslandRuntimeJS())
	for _, want := range []string{
		`function captureFocus()`,
		`function restoreFocus(snapshot)`,
		`target.focus({ preventScroll: true })`,
		`function captureMutableControls(root)`,
		`function changedMutableControls(root, baseline)`,
		`function restoreMutableControls(snapshot)`,
		`markPreservedEditsDirty()`,
		`state === "dirty" || state === "pending" || state === "error"`,
		`function nextSubmitSequence(form)`,
		`function currentSubmitSequence(form, sequence)`,
		`function submitBaselineState(form)`,
		`function clearSubmitBaselineState(form)`,
		`if (!currentSubmitSequence(form, sequence)) return`,
		`isCurrent: function () { return currentSubmitSequence(form, sequence); }`,
		`if (!hasKeys(data)) return null`,
		`Studio action failed; no structured authoring response.`,
		`function applySubmitError(form, result, meta)`,
		`gosxstudio:authoring-error`,
		`data-gosx-studio-authoring-field-errors`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("IslandRuntimeJS() missing focus/order safety fragment %q", want)
		}
	}
}
