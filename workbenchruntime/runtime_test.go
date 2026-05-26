package workbenchruntime

import (
	"strings"
	"testing"
)

// The workbenchruntime package is the Go-side surface for the Phase 3 slice-7
// burn-down of GoSXStudioWorkbenchRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip from the legacy JS implementation in assets/studio-engines.js
//     to the .gsx-authored islands in this package.
//  2. The JS shim that studio-engines.js appends so the
//     window.GoSXStudioWorkbenchRuntime methods delegate to the islands when
//     the flag is on.
//  3. The island runtime JS that publishes window.__gosx_workbench_runtime_*
//     globals the BridgeShim delegates to.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-7-workbenchruntime.md
// for the slice plan. All 15 methods in this slice are editor-chrome — no
// iframe crossing required (workbench mode / viewport / zoom / rail state /
// activity drawer / focus mode / persistence are all editor-side).

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if FeatureFlagKey != "workbench-runtime-islands" {
		t.Fatalf("FeatureFlagKey must be %q per Phase 3 slice-7 contract; got %q", "workbench-runtime-islands", FeatureFlagKey)
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
	// public global it shims (window.GoSXStudioWorkbenchRuntime), and the
	// feature flag so the legacy path stays reachable when the flag is off.
	// The IslandGlobals struct holds the canonical names; any drift between
	// the struct and the shim is caught here.
	for _, fragment := range []string{
		"window.GoSXStudioWorkbenchRuntime",
		IslandGlobals.BindRailResizers,
		IslandGlobals.BindChrome,
		IslandGlobals.SetMode,
		IslandGlobals.SyncViewport,
		IslandGlobals.ActivateViewport,
		IslandGlobals.CurrentBreakpoint,
		IslandGlobals.SetStyleState,
		IslandGlobals.SyncZoom,
		IslandGlobals.ActivateZoom,
		IslandGlobals.ToggleRail,
		IslandGlobals.ToggleFocus,
		IslandGlobals.ToggleActivity,
		IslandGlobals.SaveLayout,
		IslandGlobals.CurrentRailWidth,
		IslandGlobals.SetRailWidth,
		FeatureFlagKey,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
}

func TestBridgeShimPreservesLegacyPathWhenFlagOff(t *testing.T) {
	shim := string(BridgeShim())
	// The shim is additive — when the island global is missing, the legacy
	// JS path must still run. Sanity check: look for the legacy function
	// names so a future refactor that drops the fallback is caught here.
	// The fifteen legacy function names are the catalog of
	// GoSXStudioWorkbenchRuntime methods as wired in
	// assets/studio-engines.js:2312 — every method name in the public
	// contract maps to a legacy function with a "Workbench" prefix
	// (bindRailResizers → bindWorkbenchRailResizers, etc).
	for _, legacy := range []string{
		"bindWorkbenchRailResizers",
		"bindWorkbenchChrome",
		"setWorkbenchMode",
		"syncWorkbenchViewport",
		"activateWorkbenchViewport",
		"currentWorkbenchBreakpoint",
		"setWorkbenchStyleState",
		"syncWorkbenchZoom",
		"activateWorkbenchZoom",
		"toggleWorkbenchRail",
		"toggleWorkbenchFocus",
		"toggleWorkbenchActivity",
		"saveWorkbenchLayout",
		"currentWorkbenchRailWidth",
		"setWorkbenchRailWidth",
	} {
		if !strings.Contains(shim, legacy) {
			t.Fatalf("BridgeShim() must retain legacy fallback for %q:\n%s", legacy, shim)
		}
	}
}

func TestBundleConcatenatesIslandRuntimeAndShim(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	// Island runtime must come before the BridgeShim — the shim consults
	// the globals the island runtime publishes. Order matters: if the shim
	// ran first it would close over an undefined global and every dispatch
	// would fall through to the legacy path, silently disabling the slice.
	// (The island file ends with the closing IIFE wrapper, which always
	// precedes the shim's leading ";(function () {" marker.)
	islandHeader := "island_runtime.js"
	shimHeader := "Phase 3 slice-7 WorkbenchRuntime island bridge"
	islandIdx := strings.Index(bundle, islandHeader)
	shimIdx := strings.Index(bundle, shimHeader)
	if islandIdx < 0 || shimIdx < 0 {
		t.Fatalf("Bundle() missing island runtime or shim markers:\nislandIdx=%d shimIdx=%d", islandIdx, shimIdx)
	}
	if islandIdx > shimIdx {
		t.Fatalf("Bundle() must place island runtime before shim (island=%d shim=%d)", islandIdx, shimIdx)
	}
}
