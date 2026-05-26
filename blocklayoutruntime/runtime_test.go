package blocklayoutruntime

import (
	"strings"
	"testing"
)

// The blocklayoutruntime package is the Go-side surface for the Phase 3
// slice-4 burn-down of GoSXStudioBlockLayoutRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip from the legacy JS implementation in assets/studio-engines.js
//     to the .gsx-authored islands in this package.
//  2. The JS shim that studio-engines.js appends so the
//     window.GoSXStudioBlockLayoutRuntime methods delegate to the islands
//     when the flag is on.
//  3. The island runtime JS that publishes window.__gosx_blocklayout_runtime_*
//     globals the BridgeShim delegates to.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-4-blocklayoutruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the cross-frame signal channel updateVisibilityState writes through
// (slice 6 will swap the transitional legacy-delegation pattern below for a
// $preview.block.<key>.visible subscriber).

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if FeatureFlagKey != "block-layout-runtime-islands" {
		t.Fatalf("FeatureFlagKey must be %q per Phase 3 slice-4 contract; got %q", "block-layout-runtime-islands", FeatureFlagKey)
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
	// The shim must reference each method-specific island global, the public
	// global it shims (window.GoSXStudioBlockLayoutRuntime), and the feature
	// flag so the legacy path stays reachable when the flag is off. The
	// IslandGlobals struct holds the canonical names; any drift between the
	// struct and the shim is caught here.
	for _, fragment := range []string{
		"window.GoSXStudioBlockLayoutRuntime",
		IslandGlobals.Rows,
		IslandGlobals.RowKey,
		IslandGlobals.RowForKey,
		IslandGlobals.MoveRow,
		IslandGlobals.Renumber,
		IslandGlobals.SelectRow,
		IslandGlobals.CommitReorder,
		IslandGlobals.UpdateBlockLibraryState,
		IslandGlobals.UpdateVisibilityState,
		FeatureFlagKey,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
}

func TestIslandRuntimeJSPublishesRowsGlobal(t *testing.T) {
	// Method 1/9: rows(list) — legacy blockRows at studio-engines.js:1960.
	// Pure helper: Array.from(list.querySelectorAll("[data-block-studio-block]")).
	// Read-only DOM query; no side effects. The island global must be
	// published as a function value so BridgeShim.delegate sees it.
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	want := "window." + IslandGlobals.Rows + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// DOM contract — the block-list query selector. If muddy-noni renames
	// data-block-studio-block, this test fails and the slice author surfaces
	// the drift before parity tests start lying.
	if !strings.Contains(body, "data-block-studio-block") {
		t.Fatalf("IslandRuntimeJS() rows must query data-block-studio-block")
	}
}

func TestBridgeShimPreservesLegacyPathWhenFlagOff(t *testing.T) {
	shim := string(BridgeShim())
	// The shim is additive — when the island global is missing, the legacy
	// JS path must still run. Sanity check: look for the legacy function
	// names so a future refactor that drops the fallback is caught here.
	// The nine legacy function names are the catalog of GoSXStudioBlockLayoutRuntime
	// methods as wired in assets/studio-engines.js:2353.
	for _, legacy := range []string{
		"blockRows",
		"blockRowKey",
		"blockRowForKey",
		"moveBlockLayoutRow",
		"renumberBlockLayoutList",
		"selectBlockLayoutRow",
		"commitBlockLayoutReorder",
		"updateBlockLayoutLibraryState",
		"updateBlockLayoutVisibilityState",
	} {
		if !strings.Contains(shim, legacy) {
			t.Fatalf("BridgeShim() must retain legacy fallback for %q:\n%s", legacy, shim)
		}
	}
}
