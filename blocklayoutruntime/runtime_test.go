package blocklayoutruntime

import (
	"strings"
	"testing"
)

// The blocklayoutruntime package is the Go-side surface for the Phase 3
// slice-4 burn-down of GoSXStudioBlockLayoutRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip the island path (legacy bundle deleted 2026-05-27)
//     to the .gsx-authored islands in this package.
//  2. The JS shim that the legacy bundle appends so the
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
		// v0.5.1 listener-binding methods (Open questions 1 + 2 in
		// ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md).
		IslandGlobals.BindLibrary,
		IslandGlobals.BindVisibility,
		FeatureFlagKey,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
}

func TestIslandRuntimeJSPublishesRowsGlobal(t *testing.T) {
	// Method 1/9: rows(list) — legacy blockRows at the legacy bundle:1960.
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
	// Method 2/9: rowKey(row) — legacy blockRowKey at the legacy bundle:1964.
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
	// the legacy bundle:1968. Pure helper:
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
	// the legacy bundle:2072. Mutates the block-list by insertBefore'ing the
	// row before its previousElementSibling (direction="up") or its
	// nextElementSibling's nextSibling (direction="down"), then calls
	// renumber(list, "engine-buttons") + selectRow(list, rowKey(row)).
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.MoveRow + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// The renumber source string is part of the contract — engine listeners
	// branch on it (see the legacy bundle (removed 2026-05-27)). Catching drift here
	// before parity tests run.
	if !strings.Contains(body, "engine-buttons") {
		t.Fatalf("IslandRuntimeJS() moveRow must pass renumber source %q", "engine-buttons")
	}
}

func TestIslandRuntimeJSPublishesRenumberGlobal(t *testing.T) {
	// Method 5/9: renumber(list, source) — legacy renumberBlockLayoutList at
	// the legacy bundle:2047. Re-indexes [data-block-studio-order] inputs to
	// 1-based positions, writes data-block-studio-index attrs, refreshes
	// up/down move-button disabled states (updateBlockMoveButtons at
	// the legacy bundle:2062), and dispatches blockstudio:reorder + input +
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
	// the legacy bundle:2099. Toggles the .is-selected class on every row to
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
	// commitBlockLayoutReorder at the legacy bundle:2084. position="before"|
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

func TestIslandRuntimeJSPublishesUpdateBlockLibraryStateGlobal(t *testing.T) {
	// Method 8/9: updateBlockLibraryState(root) — legacy
	// updateBlockLayoutLibraryState at the legacy bundle:1972. For every
	// [data-editor-add-block] button in root, resolves the corresponding row
	// in document via rowForKey, reads the row's [data-editor-block-visible]
	// checkbox state, and updates the button's className / aria-pressed /
	// <small> label text.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.UpdateBlockLibraryState + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// DOM contract these helpers query / mutate. Drift in any of these
	// silently breaks the library-button sync after a visibility toggle.
	for _, contract := range []string{
		// Source-of-truth attribute for the library button -> block-key map.
		"data-editor-add-block",
		// Visibility checkbox the helper reads to decide button state.
		"data-editor-block-visible",
		// Base-class attribute the legacy reads to compose the final
		// className (button + base + button--ghost is-active / button--secondary).
		"data-editor-button-base",
		// Two button modifier classes the helper toggles.
		"button--ghost is-active",
		"button--secondary",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() updateBlockLibraryState must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesUpdateVisibilityStateGlobal(t *testing.T) {
	// Method 9/9: updateVisibilityState(check) — legacy
	// updateBlockLayoutVisibilityState at the legacy bundle:1994. Mutates the
	// enclosing [data-block-studio-block] row's classes / status text / pill
	// className based on the supplied [data-editor-block-visible] checkbox,
	// then publishes $preview.block.<key>.visible (slice 6 transitional
	// cleanup, Section G.3 — previously delegated cross-frame mutation to
	// window.GoSXStudioPreviewRuntime.setBlockVisibility). The cross-frame
	// relay (per ADR 0009) delivers the signal to the iframe's
	// preview_subscriber, where the [data-studio-block-key] node's display
	// gets toggled.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.UpdateVisibilityState + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contract — the row class toggled on hidden state, the status
		// text + pill labels users see in the editor.
		"editor-block--hidden",
		"data-editor-block-status",
		"data-editor-block-pill",
		"status--ready",
		// Slice 6 transitional cleanup (Section G.3) — the cross-frame
		// mutation now rides on $preview.block.<key>.visible. The signal
		// name prefix is asserted here so drift in either the writer or
		// the subscriber (gosx-studio/previewruntime/subscriber_runtime.js)
		// surfaces immediately.
		"$preview.block.",
		".visible",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() updateVisibilityState must preserve %q contract", contract)
		}
	}
}

func TestBridgeShimAutoMountsOnDocumentReady(t *testing.T) {
	// Pre-2026-05-27 the deleted studio-engines.js bundle's engine-mount
	// factory invoked bindBlockLayoutList / bindBlockLayoutLibrary /
	// bindBlockLayoutVisibility for the block-layout engine at boot.
	//
	// v0.5.0 restored the observable that DID exist as an island global:
	// updateBlockLibraryState — refreshes the library button states
	// (className / aria-pressed / label) against the current row visibility.
	// The two listener-binding helpers were deferred because their island
	// migration hadn't shipped yet.
	//
	// v0.5.1 ships those island migrations (bindLibrary, bindVisibility)
	// and the auto-mount now invokes all three so add-block button clicks
	// and visibility checkbox changes actually work in production. See
	// ~/.hyphae/spaces/m31labs-gosx/inbox/agents/2026-05-27-elm-bridgemount-restoration-v0-5-0.md
	// (Open questions 1 + 2) for the bug this closes.
	shim := string(BridgeShim())
	for _, fragment := range []string{
		"DOMContentLoaded",
		"window.GoSXStudioBlockLayoutRuntime.bindLibrary",
		"window.GoSXStudioBlockLayoutRuntime.bindVisibility",
		"window.GoSXStudioBlockLayoutRuntime.updateBlockLibraryState",
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing auto-mount fragment %q:\n%s", fragment, shim)
		}
	}
}

func TestIslandRuntimeJSPublishesBindLibraryGlobal(t *testing.T) {
	// v0.5.1 follow-on: bindLibrary(root) restores the deleted
	// bindBlockLayoutLibrary helper (studio-engines.js:2022 in the legacy
	// bundle deleted 2026-05-27). The island scopes a click delegate to
	// the editor workbench form for [data-editor-add-block] buttons; on
	// click, the row's visibility checkbox is flipped to checked (with
	// input + change events so the visibility island fires), the row is
	// selected, scrolled into view, focused, and the library buttons are
	// refreshed.
	//
	// Idempotency: per-form marker
	// data-gosx-studio-block-library-island-bound="true" (renamed from the
	// legacy gosxStudioBlockLibraryBound to make the post-deletion
	// ownership obvious). Compare to the deleted bindBlockLayoutLibrary
	// pattern documented in the slice-4 deletion log "Idempotency" table.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.BindLibrary + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contracts the click handler queries against.
		"data-editor-workbench",
		"data-editor-add-block",
		"data-editor-block-visible",
		// Idempotency marker (camelCase dataset key form).
		"gosxStudioBlockLibraryIslandBound",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindLibrary must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesBindVisibilityGlobal(t *testing.T) {
	// v0.5.1 follow-on: bindVisibility(root) restores the deleted
	// bindBlockLayoutVisibility helper (studio-engines.js:2010 in the
	// legacy bundle deleted 2026-05-27). For every
	// [data-editor-block-visible] checkbox in root, attaches a change
	// listener that calls updateVisibilityState (which toggles the row's
	// hidden class, status text, pill class, and publishes the
	// $preview.block.<key>.visible shared signal). Also fires
	// updateVisibilityState(check) once at bind time to sync the
	// storefront preview's initial state to the checkbox state.
	//
	// Idempotency: per-checkbox marker
	// data-gosx-studio-block-visibility-island-bound="true" (renamed from
	// the legacy gosxStudioBlockVisibilityBound to make the post-deletion
	// ownership obvious). Compare to the deleted bindBlockLayoutVisibility
	// pattern documented in the slice-4 deletion log "Idempotency" table.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.BindVisibility + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contract the change handler queries against.
		"data-editor-block-visible",
		// Idempotency marker (camelCase dataset key form).
		"gosxStudioBlockVisibilityIslandBound",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindVisibility must preserve %q contract", contract)
		}
	}
}

func TestBundleConcatenatesIslandRuntimeAndShim(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	// Island runtime must come before the BridgeShim — the shim consults
	// the global the island runtime publishes. Order matters: if the shim
	// ran first it would close over an undefined global and every dispatch
	// would fall through to the legacy path, silently disabling the slice.
	islandIdx := strings.Index(bundle, IslandGlobals.Rows+" ")
	shimIdx := strings.Index(bundle, "window.GoSXStudioBlockLayoutRuntime =")
	if islandIdx < 0 || shimIdx < 0 {
		t.Fatalf("Bundle() missing island runtime or shim:\n%s", bundle)
	}
	if islandIdx > shimIdx {
		t.Fatalf("Bundle() must place island runtime before shim (island=%d shim=%d)", islandIdx, shimIdx)
	}
}
