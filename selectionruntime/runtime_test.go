package selectionruntime

import (
	"strings"
	"testing"
)

// The selectionruntime package is the Go-side surface for the Phase 3 slice-2
// burn-down of GoSXStudioSelectionRuntime. It owns:
//   1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//      to flip the island path (legacy bundle deleted 2026-05-27)
//      to the .gsx-authored island in this package.
//   2. The JS shim that the legacy bundle appends so the
//      window.GoSXStudioSelectionRuntime methods delegate to the island when
//      the flag is on.
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
	// must reference window.GoSXStudioSelectionRuntime (the public global
	// it shims), and must reference the feature flag so the legacy path
	// stays reachable when the flag is off.
	for _, fragment := range []string{
		"window.GoSXStudioSelectionRuntime",
		"__gosx_selection_runtime_island_bind",
		FeatureFlagKey,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
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
	// The island runtime must preserve the legacy bindSelectionSurface DOM
	// attribute contract — selection-action commandbar, workspace targets,
	// field-source focus, style-scope readouts, and the idempotency-guard
	// dataset key. Each entry below maps to one of the five sub-behaviors
	// enumerated in selection_bind.gsx; if a fragment is missing the
	// corresponding sub-behavior is silently broken.
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
