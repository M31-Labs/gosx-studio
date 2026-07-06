package inspectorruntime

import (
	"strings"
	"testing"
)

func TestInspectorRuntimeFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	if !strings.HasSuffix(FeatureFlagKey, "-runtime-islands") {
		t.Fatalf("FeatureFlagKey %q must end with -runtime-islands per Phase 3 convention", FeatureFlagKey)
	}
}

func TestInspectorRuntimeJSIsNonEmpty(t *testing.T) {
	body := string(InspectorRuntimeScript())
	if body == "" {
		t.Fatal("InspectorRuntimeScript() must return a non-empty JS snippet")
	}
}

func TestInspectorRuntimeJSPublishesGlobal(t *testing.T) {
	body := string(InspectorRuntimeScript())
	if !strings.Contains(body, "window.GoSXStudioInspectorRuntime") {
		t.Fatalf("InspectorRuntimeScript() must publish window.GoSXStudioInspectorRuntime:\n%s", body)
	}
	// Must expose bind and bindAll
	if !strings.Contains(body, "bind") {
		t.Fatalf("InspectorRuntimeScript() must expose a bind method:\n%s", body)
	}
	if !strings.Contains(body, "bindAll") {
		t.Fatalf("InspectorRuntimeScript() must expose a bindAll method:\n%s", body)
	}
}

func TestInspectorRuntimeJSDirtyTracking(t *testing.T) {
	body := string(InspectorRuntimeScript())
	// The island must reference the inspector-control wrapper attribute
	if !strings.Contains(body, "data-gosx-studio-inspector-control") {
		t.Fatalf("InspectorRuntimeScript() must reference data-gosx-studio-inspector-control:\n%s", body)
	}
	// Must track dirty state
	if !strings.Contains(body, "dirty") {
		t.Fatalf("InspectorRuntimeScript() must implement dirty tracking:\n%s", body)
	}
}

func TestInspectorRuntimeJSNoSubmitLogic(t *testing.T) {
	body := string(InspectorRuntimeScript())
	// The island must NOT implement its own fetch/submit — authoring runtime owns that
	if strings.Contains(body, "fetch(") {
		t.Fatalf("InspectorRuntimeScript() must not implement fetch; submit is owned by authoringruntime:\n%s", body)
	}
	for _, forbidden := range []string{
		`addEventListener("submit"`,
		`addEventListener('submit'`,
		"requestSubmit",
		"form.submit",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("InspectorRuntimeScript() must not implement submit handling %q; submit is owned by authoringruntime:\n%s", forbidden, body)
		}
	}
}

func TestInspectorRuntimeJSHomeInspectorDelegation(t *testing.T) {
	body := string(InspectorRuntimeScript())
	for _, want := range []string{
		"bindHomeInspector",
		"data-studio-home-inspector-panel",
		"data-studio-inspector-group",
		"data-studio-inspector-group-active",
		"data-studio-home-inspector-fields",
		"data-studio-inspector-dirty",
		"data-studio-inspector-dirty-label",
		"data-studio-inspector-state",
		`addEventListener("click"`,
		`addEventListener("input"`,
		`addEventListener("change"`,
		"aria-pressed",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("InspectorRuntimeScript() missing Home inspector delegated contract %q:\n%s", want, body)
		}
	}
}

func TestBundleIsNonEmpty(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	if !strings.Contains(bundle, "window.GoSXStudioInspectorRuntime") {
		t.Fatalf("Bundle() must contain the inspector runtime global:\n%s", bundle)
	}
	if !strings.Contains(bundle, "data-studio-home-inspector-panel") {
		t.Fatalf("Bundle() must contain the Home inspector delegated runtime contract:\n%s", bundle)
	}
}
