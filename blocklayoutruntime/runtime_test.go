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

func TestIslandRuntimeJSPublishesRowKeyGlobal(t *testing.T) {
	// Method 2/9: rowKey(row) — legacy blockRowKey at studio-engines.js:1964.
	// Pure helper: row.getAttribute("data-block-studio-block") || "".
	// Read-only; returns string. The island global must publish the function.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.RowKey + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
}

func TestIslandRuntimeJSPublishesRowForKeyGlobal(t *testing.T) {
	// Method 3/9: rowForKey(root, key) — legacy blockRowForKey at
	// studio-engines.js:1968. Pure helper:
	// root.querySelector('[data-block-studio-block="<key>"]').
	// Uses the same attrValue escape as the legacy attrValue helper so keys
	// containing double quotes or backslashes don't break the selector.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.RowForKey + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
}

func TestIslandRuntimeJSPublishesMoveRowGlobal(t *testing.T) {
	// Method 4/9: moveRow(list, row, direction) — legacy moveBlockLayoutRow at
	// studio-engines.js:2072. Mutates the block-list by insertBefore'ing the
	// row before its previousElementSibling (direction="up") or its
	// nextElementSibling's nextSibling (direction="down"), then calls
	// renumber(list, "engine-buttons") + selectRow(list, rowKey(row)).
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.MoveRow + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// The renumber source string is part of the contract — engine listeners
	// branch on it (see assets/studio-engines.js:2054). Catching drift here
	// before parity tests run.
	if !strings.Contains(body, "engine-buttons") {
		t.Fatalf("IslandRuntimeJS() moveRow must pass renumber source %q", "engine-buttons")
	}
}

func TestIslandRuntimeJSPublishesRenumberGlobal(t *testing.T) {
	// Method 5/9: renumber(list, source) — legacy renumberBlockLayoutList at
	// studio-engines.js:2047. Re-indexes [data-block-studio-order] inputs to
	// 1-based positions, writes data-block-studio-index attrs, refreshes
	// up/down move-button disabled states (updateBlockMoveButtons at
	// studio-engines.js:2062), and dispatches blockstudio:reorder + input +
	// change events on the list.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.Renumber + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// DOM + event contracts the engine listeners depend on. Drift in any of
	// these silently breaks the reorder fan-out.
	for _, contract := range []string{
		// Order-input attribute the legacy reads + writes.
		"data-block-studio-order",
		// Move-button disabled state — the up/down buttons sit on every row
		// and disable at the list ends.
		"data-block-studio-move",
		// Index attribute that records the row's position after renumbering.
		"data-block-studio-index",
		// CustomEvent name the form's reorder listener watches.
		"blockstudio:reorder",
		// Default source string when caller passes no source argument.
		"block-layout-engine",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() renumber must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSelectRowGlobal(t *testing.T) {
	// Method 6/9: selectRow(root, key) — legacy selectBlockLayoutRow at
	// studio-engines.js:2099. Toggles the .is-selected class on every row to
	// match the supplied key, scrollIntoView's the newly-selected row, and
	// dispatches a blockstudio:select CustomEvent on document. No-op when key
	// is empty.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SelectRow + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// CustomEvent contract — the selection commandbar + inspector form watch
	// for blockstudio:select.
	if !strings.Contains(body, "blockstudio:select") {
		t.Fatalf("IslandRuntimeJS() selectRow must dispatch %q", "blockstudio:select")
	}
	// CSS class used to highlight the selected row in the editor.
	if !strings.Contains(body, "is-selected") {
		t.Fatalf("IslandRuntimeJS() selectRow must toggle is-selected class")
	}
}

func TestIslandRuntimeJSPublishesCommitReorderGlobal(t *testing.T) {
	// Method 7/9: commitReorder(list, key, targetKey, position) — legacy
	// commitBlockLayoutReorder at studio-engines.js:2084. position="before"|
	// "after". Guards no-op (returns false when key/targetKey missing or
	// equal); otherwise insertBefore + renumber(list, "engine-preview") +
	// selectRow(list, key); returns true.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.CommitReorder + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// The "engine-preview" source string is passed to renumber so the form
	// listeners can distinguish a canvas-preview reorder from an
	// engine-buttons reorder. Drift here would silently miscategorise the
	// source on subscribers.
	if !strings.Contains(body, "engine-preview") {
		t.Fatalf("IslandRuntimeJS() commitReorder must pass renumber source %q", "engine-preview")
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
