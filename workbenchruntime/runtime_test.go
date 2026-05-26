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

func TestIslandRuntimeJSPublishesBindRailResizersGlobal(t *testing.T) {
	// Method 1/15: bindRailResizers(root) — legacy bindWorkbenchRailResizers
	// at studio-engines.js:194. Binds the [data-studio-resizer] pointer drag
	// + ArrowLeft/ArrowRight keyboard nudges on rail handles. Uses
	// gosxStudioResizerIslandBound as the per-handle idempotency guard
	// (distinct from the legacy gosxStudioResizerBound to allow both paths
	// to coexist additively).
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	want := "window." + IslandGlobals.BindRailResizers + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// DOM contract — the resizer handle selectors and side discriminator
	// the legacy binds against (see studio-engines.js:198-237). Drift here
	// silently breaks the rail-resizer pointer / keyboard fan-out.
	for _, contract := range []string{
		"data-studio-resizer",
		// Idempotency guard key — distinct from legacy
		// gosxStudioResizerBound so both paths can coexist additively.
		"gosxStudioResizerIslandBound",
		// Pointer events the drag handler attaches.
		`"pointerdown"`,
		`"pointermove"`,
		`"pointerup"`,
		`"pointercancel"`,
		// Keyboard events the arrow-key handler attaches.
		`"keydown"`,
		`"ArrowLeft"`,
		`"ArrowRight"`,
		// Visual state contract toggled during a drag.
		"is-resizing",
		// Step sizes for ArrowLeft/Right + Shift modifier.
		"24",
		"48",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindRailResizers must preserve %q contract:\n%s", contract, body)
		}
	}
}

func TestIslandRuntimeJSPublishesBindChromeGlobal(t *testing.T) {
	// Method 2/15: bindChrome(root) — legacy bindWorkbenchChrome at
	// studio-engines.js:516. Binds the workbench form's delegated click
	// handler (fans out to setMode / syncViewport / syncZoom / setStyleState
	// / toggleRail / toggleFocus / toggleActivity), wires saveLayoutSoon to
	// rail-width-change/commit events, applies the persisted layout, and
	// seeds the workbench mode / viewport / zoom on initial bind.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.BindChrome + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Per-form idempotency guard — distinct from legacy
		// gosxStudioWorkbenchChromeBound so both paths coexist additively.
		"gosxStudioWorkbenchChromeIslandBound",
		// DOM selectors the delegated click handler walks (mirrors
		// studio-engines.js:520-561).
		"data-studio-mode-control",
		"data-studio-viewport",
		"data-studio-zoom",
		"data-studio-style-state",
		"data-studio-rail-toggle",
		"data-studio-focus-toggle",
		"data-studio-activity-toggle",
		// Custom event names the rail-width change/commit handlers
		// listen for.
		"gosxstudio:rail-width-change",
		"gosxstudio:rail-width-commit",
		// Initial seeded form attributes the bind sets when absent
		// (mirrors studio-engines.js:577-580).
		"data-studio-left",
		"data-studio-right",
		"data-studio-focus",
		"data-studio-activity-state",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindChrome must preserve %q contract:\n%s", contract, body)
		}
	}
}

func TestIslandRuntimeJSPublishesSetModeGlobal(t *testing.T) {
	// Method 3/15: setMode(form, mode, scroll) — legacy setWorkbenchMode at
	// studio-engines.js:343. Sets data-studio-mode on the form, toggles
	// aria-pressed on [data-studio-mode-control] buttons, toggles
	// is-mode-active / hidden / aria-hidden on [data-studio-mode-panel]
	// siblings, scrolls active panel into view when requested, updates
	// [data-studio-mode-label] readouts, emits
	// gosxstudio:workbench-mode-change. The mode alias mapping
	// (structure/content → home, style → look, preview → publish,
	// manage/flows → advanced) is preserved verbatim.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SetMode + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Form attribute written by setMode.
		"data-studio-mode",
		// DOM contracts the helper queries / mutates.
		"data-studio-mode-control",
		"data-studio-mode-panel",
		"data-studio-mode-label",
		// Class toggled on active mode panels (studio-engines.js:352).
		"is-mode-active",
		// Custom event name dispatched on mode change.
		"workbench-mode-change",
		// Mode aliases (mirror studio-engines.js:316-322 normalizer).
		`"structure"`,
		`"content"`,
		`"style"`,
		`"preview"`,
		`"manage"`,
		`"flows"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() setMode must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSyncViewportGlobal(t *testing.T) {
	// Method 4/15: syncViewport(form, viewport) — legacy syncWorkbenchViewport
	// at studio-engines.js:369. Sets data-studio-breakpoint on the form,
	// data-studio-preview-viewport on the .editor-preview-shell (only when
	// the viewport island is not already managing it), updates
	// [data-studio-viewport-label] readouts, emits
	// gosxstudio:workbench-viewport-change, refreshes the canvas.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SyncViewport + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Form attribute syncViewport writes.
		"data-studio-breakpoint",
		// Preview-shell selector + attribute syncViewport defers to when
		// the viewport island is not managing it.
		"editor-preview-shell",
		"data-studio-preview-viewport",
		// Viewport-island opt-out marker so the inline island can prevent
		// double-writes during the additive shipping window.
		"data-studio-viewport-island",
		// Viewport label readout selector.
		"data-studio-viewport-label",
		// Event name dispatched on viewport change.
		"workbench-viewport-change",
		// Default viewport when none provided.
		`"desktop"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() syncViewport must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesActivateViewportGlobal(t *testing.T) {
	// Method 5/15: activateViewport(form, viewport) — legacy
	// activateWorkbenchViewport at studio-engines.js:382. Clicks the
	// matching [data-studio-viewport="<x>"] button if present; otherwise
	// falls through to syncViewport. Routes user-initiated viewport
	// activation through the same dispatch as the toolbar / command palette.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ActivateViewport + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Button selector activateViewport prefers when present.
		"data-studio-viewport",
		// Fallback delegation when no button is matched.
		"syncViewportIsland",
		// Default viewport when none provided.
		`"desktop"`,
		// attrValue escape helper used to build the button selector.
		"attrValue",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() activateViewport must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesCurrentBreakpointGlobal(t *testing.T) {
	// Method 6/15: currentBreakpoint(form) — legacy
	// currentWorkbenchBreakpoint at studio-engines.js:393. Pure read: derives
	// the form's data-studio-breakpoint attribute (or the preview-shell's
	// data-studio-preview-viewport fallback, or "desktop"). Per the slice
	// plan, this method is implemented as a pure derivation from existing
	// form state — no mutator side effects, no signal writes; the BridgeShim
	// routes through it uniformly for API consistency with the rest of the
	// contract.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.CurrentBreakpoint + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// The two read sources currentBreakpoint derives from
		// (studio-engines.js:395-398).
		"data-studio-breakpoint",
		"data-studio-preview-viewport",
		"editor-preview-shell",
		// Default returned when nothing matches.
		`"desktop"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() currentBreakpoint must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSetStyleStateGlobal(t *testing.T) {
	// Method 7/15: setStyleState(form, state) — legacy setWorkbenchStyleState
	// at studio-engines.js:405. Sets data-studio-style-state on the form,
	// toggles aria-pressed on [data-studio-style-state] buttons, writes
	// data-style-state / data-style-breakpoint / data-style-valid on
	// [data-studio-style-scope] wrappers, emits
	// gosxstudio:workbench-style-state-change, refreshes the canvas.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SetStyleState + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Form attribute setStyleState writes.
		"data-studio-style-state",
		// Style-scope wrapper attributes setStyleState writes.
		"data-studio-style-scope",
		"data-style-state",
		"data-style-breakpoint",
		"data-style-valid",
		// Event name dispatched on style-state change.
		"workbench-style-state-change",
		// Default state when none provided (mirrors normalizer at
		// studio-engines.js:401).
		`"default"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() setStyleState must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSyncZoomGlobal(t *testing.T) {
	// Method 8/15: syncZoom(form, zoom) — legacy syncWorkbenchZoom at
	// studio-engines.js:421. Sets data-studio-canvas-zoom on the
	// [data-studio-canvas] element, emits gosxstudio:workbench-zoom-change,
	// refreshes the canvas via a resize event.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SyncZoom + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contracts the helper queries / writes.
		"data-studio-canvas",
		"data-studio-canvas-zoom",
		// Event name dispatched on zoom change.
		"workbench-zoom-change",
		// Default zoom when none provided.
		`"fit"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() syncZoom must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesActivateZoomGlobal(t *testing.T) {
	// Method 9/15: activateZoom(form, zoom) — legacy activateWorkbenchZoom
	// at studio-engines.js:430. Clicks the matching
	// [data-studio-zoom="<x>"] button if present; otherwise falls through
	// to syncZoom. Routes user-initiated zoom activation through the same
	// dispatch as the toolbar / command palette.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ActivateZoom + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Button selectors activateZoom prefers when present (mirrors
		// studio-engines.js:433 — both button[data-studio-zoom] and
		// [role='button'][data-studio-zoom]).
		"data-studio-zoom",
		// Fallback delegation when no button is matched.
		"syncZoomIsland",
		// Default zoom when none provided.
		`"fit"`,
		// attrValue escape helper used to build the button selector.
		"attrValue",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() activateZoom must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesToggleRailGlobal(t *testing.T) {
	// Method 10/15: toggleRail(form, side) — legacy toggleWorkbenchRail at
	// studio-engines.js:480. Flips data-studio-{left,right} between "open"
	// and "collapsed", sets data-studio-focus to "false", re-syncs the rail
	// toggle aria-pressed states, emits gosxstudio:workbench-rail-change,
	// refreshes the canvas.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ToggleRail + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Form attributes toggleRail writes (mirrors studio-engines.js:482-483).
		"data-studio-focus",
		// Side discriminator strings.
		`"left"`,
		`"right"`,
		// Rail state values toggled between.
		`"open"`,
		`"collapsed"`,
		// Event name dispatched on rail change.
		"workbench-rail-change",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() toggleRail must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesToggleFocusGlobal(t *testing.T) {
	// Method 11/15: toggleFocus(form) — legacy toggleWorkbenchFocus at
	// studio-engines.js:489. Flips data-studio-focus between "true" and
	// "false", re-syncs the rail toggle aria-pressed states, emits
	// gosxstudio:workbench-focus-change, refreshes the canvas.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ToggleFocus + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Form attribute toggleFocus flips.
		"data-studio-focus",
		// Event name dispatched on focus change.
		"workbench-focus-change",
		// Boolean state strings the toggle flips between.
		`"true"`,
		`"false"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() toggleFocus must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesToggleActivityGlobal(t *testing.T) {
	// Method 12/15: toggleActivity(form) — legacy toggleWorkbenchActivity at
	// studio-engines.js:476 (via setWorkbenchActivity at 467). Flips
	// data-studio-activity-state between "open" and "collapsed", re-syncs
	// activity-toggle buttons (Show/Hide textContent + aria-pressed),
	// persists via saveLayout, emits gosxstudio:workbench-activity-change,
	// refreshes the canvas.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ToggleActivity + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Form attribute toggleActivity flips.
		"data-studio-activity-state",
		// Activity-toggle button selector + drawer wrapper selector the
		// helper queries for the Show/Hide text update.
		"data-studio-activity-toggle",
		"data-studio-activity-drawer",
		// Show/Hide labels written to drawer-housed buttons.
		`"Show"`,
		`"Hide"`,
		// Activity state values toggled between.
		`"open"`,
		`"collapsed"`,
		// Event name dispatched on activity change. emitWorkbenchChange
		// prefixes with "workbench-" so the literal in source is the
		// suffix; the runtime CustomEvent name is
		// "gosxstudio:workbench-activity-change".
		`"activity-change"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() toggleActivity must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSaveLayoutGlobal(t *testing.T) {
	// Method 13/15: saveLayout(form) — legacy saveWorkbenchLayout at
	// studio-engines.js:294. Persistence wrapper: serializes the form's
	// --studio-left-width / --studio-right-width custom properties and
	// data-studio-activity-state attribute to localStorage under the
	// gosx-studio-editor-layout key. Used as both the public island and
	// the saveLayoutSoon (frameTask-wrapped) inside bindChrome.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SaveLayout + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// localStorage key the legacy writes under (studio-engines.js:241).
		`"gosx-studio-editor-layout"`,
		// The three persisted properties (studio-engines.js:297-300).
		"--studio-left-width",
		"--studio-right-width",
		"data-studio-activity-state",
		// localStorage write API.
		"localStorage",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() saveLayout must preserve %q contract", contract)
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
