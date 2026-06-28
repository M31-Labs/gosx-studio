package workbenchruntime

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// The workbenchruntime package is the Go-side surface for the Phase 3 slice-7
// burn-down of GoSXStudioWorkbenchRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip the island path (legacy bundle deleted 2026-05-27)
//     to the .gsx-authored islands in this package.
//  2. The JS shim that the legacy bundle appends so the
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

func TestIslandRuntimeJSPublishesBindRailResizersGlobal(t *testing.T) {
	// Method 1/15: bindRailResizers(root) — legacy bindWorkbenchRailResizers
	// at the legacy bundle:194. Binds the [data-studio-resizer] pointer drag
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
	// the legacy binds against (see the legacy bundle:198-237). Drift here
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
	// the legacy bundle:516. Binds the workbench form's delegated click
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
		// the legacy bundle:520-561).
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
		// (mirrors the legacy bundle:577-580).
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
	// the legacy bundle:343. Sets data-studio-mode on the form, toggles
	// aria-pressed on [data-studio-mode-control] buttons, toggles
	// is-mode-active / hidden / aria-hidden on [data-studio-mode-panel]
	// siblings, scrolls active panel into view when requested, updates
	// [data-studio-mode-label] readouts, emits
	// gosxstudio:workbench-mode-change. The mode alias mapping keeps
	// legacy structure/content → home, style → look, and manage/flows →
	// advanced aliases while allowing preview to be its own visible editor
	// mode.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SetMode + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	if strings.Contains(body, `mode === "preview") return "publish"`) {
		t.Fatalf("IslandRuntimeJS() setMode must keep preview as its own mode")
	}
	for _, contract := range []string{
		// Form attribute written by setMode.
		"data-studio-mode",
		// DOM contracts the helper queries / mutates.
		"data-studio-mode-control",
		"data-studio-mode-panel",
		"data-studio-mode-label",
		// Class toggled on active mode panels (the legacy bundle:352).
		"is-mode-active",
		// Custom event name dispatched on mode change.
		"workbench-mode-change",
		// Mode aliases (mirror the legacy bundle:316-322 normalizer).
		`"structure"`,
		`"content"`,
		`"style"`,
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
	// at the legacy bundle:369. Sets data-studio-breakpoint on the form,
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
	// activateWorkbenchViewport at the legacy bundle:382. Clicks the
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
	// currentWorkbenchBreakpoint at the legacy bundle:393. Pure read: derives
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
		// (the legacy bundle:395-398).
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
	// at the legacy bundle:405. Sets data-studio-style-state on the form,
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
		// the legacy bundle:401).
		`"default"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() setStyleState must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSyncZoomGlobal(t *testing.T) {
	// Method 8/15: syncZoom(form, zoom) — legacy syncWorkbenchZoom at
	// the legacy bundle:421. Sets data-studio-canvas-zoom on the
	// [data-studio-canvas] element, syncs zoom toolbar state, emits
	// gosxstudio:workbench-zoom-change, refreshes the canvas via a resize event.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SyncZoom + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contracts the helper queries / writes.
		"data-studio-canvas",
		"data-studio-canvas-zoom",
		"data-studio-zoom-island",
		"data-studio-zoom-current",
		"button[data-studio-zoom], [role='button'][data-studio-zoom]",
		"aria-pressed",
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

func TestIslandRuntimeJSSyncZoomMutatesDOMAndEmitsEvent(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node executable is required for executable IslandRuntimeJS DOM coverage")
	}

	runtimeJS, err := json.Marshal(string(IslandRuntimeJS()))
	if err != nil {
		t.Fatalf("marshal IslandRuntimeJS(): %v", err)
	}

	script := `
const runtimeSource = ` + string(runtimeJS) + `;

class TestEventTarget {
  constructor() {
    this.listeners = new Map();
  }

  addEventListener(type, listener) {
    const listeners = this.listeners.get(type) || [];
    listeners.push(listener);
    this.listeners.set(type, listeners);
  }

  dispatchEvent(event) {
    event.target = event.target || this;
    const listeners = this.listeners.get(event.type) || [];
    for (const listener of listeners) listener.call(this, event);
    return true;
  }
}

class TestEvent {
  constructor(type, options = {}) {
    this.type = type;
    this.bubbles = Boolean(options.bubbles);
  }
}

class TestCustomEvent extends TestEvent {
  constructor(type, options = {}) {
    super(type, options);
    this.detail = options.detail || null;
  }
}

class TestElement extends TestEventTarget {
  constructor(tagName) {
    super();
    this.tagName = tagName.toUpperCase();
    this.attributes = new Map();
    this.children = [];
    this.parentElement = null;
  }

  appendChild(child) {
    child.parentElement = this;
    this.children.push(child);
    return child;
  }

  setAttribute(name, value) {
    this.attributes.set(name, String(value));
  }

  getAttribute(name) {
    return this.attributes.has(name) ? this.attributes.get(name) : null;
  }

  hasAttribute(name) {
    return this.attributes.has(name);
  }

  matches(selector) {
    return selector.split(",").some((part) => matchesSingleSelector(this, part.trim()));
  }

  querySelector(selector) {
    return this.querySelectorAll(selector)[0] || null;
  }

  querySelectorAll(selector) {
    const found = [];
    const visit = (node) => {
      for (const child of node.children) {
        if (child.matches(selector)) found.push(child);
        visit(child);
      }
    };
    visit(this);
    return found;
  }

  closest(selector) {
    let node = this;
    while (node) {
      if (node.matches(selector)) return node;
      node = node.parentElement;
    }
    return null;
  }
}

class TestDocument extends TestEventTarget {
  constructor() {
    super();
    this.body = new TestElement("body");
  }

  createElement(tagName) {
    return new TestElement(tagName);
  }

  querySelector(selector) {
    return this.body.querySelector(selector);
  }

  querySelectorAll(selector) {
    return this.body.querySelectorAll(selector);
  }
}

function matchesSingleSelector(element, selector) {
  if (!selector) return false;
  let rest = selector;
  const tag = rest.match(/^[a-zA-Z][a-zA-Z0-9-]*/);
  if (tag) {
    if (element.tagName.toLowerCase() !== tag[0].toLowerCase()) return false;
    rest = rest.slice(tag[0].length);
  }
  const attrPattern = /\[([^\]=]+)(?:=(['"]?)(.*?)\2)?\]/g;
  let match;
  let sawAttribute = false;
  while ((match = attrPattern.exec(rest)) !== null) {
    sawAttribute = true;
    const actual = element.getAttribute(match[1]);
    if (actual === null) return false;
    if (match[3] !== undefined && actual !== match[3]) return false;
  }
  const consumedAttrs = rest.replace(attrPattern, "");
  return consumedAttrs === "" && (Boolean(tag) || sawAttribute);
}

function assertEqual(actual, expected, message) {
  if (actual !== expected) {
    throw new Error(message + ": got " + JSON.stringify(actual) + ", want " + JSON.stringify(expected));
  }
}

const document = new TestDocument();
const rafCallbacks = [];
const window = new TestEventTarget();
window.document = document;
window.requestAnimationFrame = (callback) => {
  rafCallbacks.push(callback);
  return rafCallbacks.length;
};
window.setTimeout = (callback) => {
  rafCallbacks.push(callback);
  return rafCallbacks.length;
};
globalThis.window = window;
globalThis.document = document;
globalThis.Event = TestEvent;
globalThis.CustomEvent = TestCustomEvent;

eval(runtimeSource);

const form = document.createElement("form");
const canvas = document.createElement("section");
canvas.setAttribute("data-studio-canvas", "true");
canvas.setAttribute("data-studio-canvas-zoom", "100");
const zoomRoot = document.createElement("div");
zoomRoot.setAttribute("data-studio-zoom-island", "true");
zoomRoot.setAttribute("data-studio-zoom-current", "100");
const previousButton = document.createElement("button");
previousButton.setAttribute("data-studio-zoom", "100");
previousButton.setAttribute("aria-pressed", "true");
const matchingButton = document.createElement("button");
matchingButton.setAttribute("data-studio-zoom", "125");
matchingButton.setAttribute("aria-pressed", "false");

form.appendChild(canvas);
form.appendChild(zoomRoot);
zoomRoot.appendChild(previousButton);
zoomRoot.appendChild(matchingButton);
document.body.appendChild(form);

let zoomEvent = null;
let resizeEvents = 0;
document.addEventListener("gosxstudio:workbench-zoom-change", (event) => {
  zoomEvent = event;
});
window.addEventListener("resize", () => {
  resizeEvents++;
});

window.__gosx_workbench_runtime_island_syncZoom(form, "125");

assertEqual(canvas.getAttribute("data-studio-canvas-zoom"), "125", "canvas zoom attribute");
assertEqual(zoomRoot.getAttribute("data-studio-zoom-current"), "125", "zoom island current attribute");
assertEqual(matchingButton.getAttribute("aria-pressed"), "true", "matching zoom button aria-pressed");
assertEqual(previousButton.getAttribute("aria-pressed"), "false", "previous zoom button aria-pressed");
if (!zoomEvent) throw new Error("expected gosxstudio:workbench-zoom-change event");
assertEqual(zoomEvent.detail.zoom, "125", "zoom event detail.zoom");
assertEqual(zoomEvent.detail.form, form, "zoom event detail.form");
assertEqual(rafCallbacks.length, 1, "scheduled resize animation frame");
assertEqual(resizeEvents, 0, "resize should be deferred until the frame callback runs");
for (const callback of rafCallbacks.splice(0)) callback();
assertEqual(resizeEvents, 1, "resize event count after flushing the frame callback");
`

	cmd := exec.Command(node)
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("node DOM syncZoom check failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}

func TestIslandRuntimeJSPublishesActivateZoomGlobal(t *testing.T) {
	// Method 9/15: activateZoom(form, zoom) — legacy activateWorkbenchZoom
	// at the legacy bundle:430. Clicks the matching
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
		// the legacy bundle:433 — both button[data-studio-zoom] and
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
	// the legacy bundle:480. Flips data-studio-{left,right} between "open"
	// and "collapsed", sets data-studio-focus to "false", re-syncs the rail
	// toggle aria-pressed states, emits gosxstudio:workbench-rail-change,
	// refreshes the canvas.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ToggleRail + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Form attributes toggleRail writes (mirrors the legacy bundle:482-483).
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
	// the legacy bundle:489. Flips data-studio-focus between "true" and
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
	// the legacy bundle:476 (via setWorkbenchActivity at 467). Flips
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
	// the legacy bundle:294. Persistence wrapper: serializes the form's
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
		// localStorage key the legacy writes under (the legacy bundle:241).
		`"gosx-studio-editor-layout"`,
		// The three persisted properties (the legacy bundle:297-300).
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

func TestIslandRuntimeJSPublishesCurrentRailWidthGlobal(t *testing.T) {
	// Method 14/15: currentRailWidth(form, side, handle) — legacy
	// currentWorkbenchRailWidth at the legacy bundle:165. Pure read:
	// returns the form's --studio-{left,right}-width custom property as an
	// integer (or the sidebar's bounding-rect width fallback, or the
	// handle's bounds fallback). Per the slice plan, this method ships as
	// a signal-derivation rather than a mutator — it reads the same DOM
	// state setRailWidth writes (--studio-{left,right}-width CSS custom
	// properties) and returns it unmodified. No event dispatch, no canvas
	// refresh. The BridgeShim routes through it uniformly so future
	// $workbench.railWidth signal consumers can substitute a signal-read
	// for the property-read without changing the contract.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.CurrentRailWidth + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// CSS custom properties the method reads (mirrors the legacy bundle:166).
		"--studio-left-width",
		"--studio-right-width",
		// Sidebar fallback selector (railSidebar helper).
		"data-studio-sidebar",
		// Integer parse used to normalize the CSS-prop string.
		"parseInt",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() currentRailWidth must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSetRailWidthGlobal(t *testing.T) {
	// Method 15/15: setRailWidth(form, side, width, handle, committed) —
	// legacy setWorkbenchRailWidth at the legacy bundle:185. Clamps width
	// to the handle's min/max bounds, writes --studio-{left,right}-width
	// on the form, updates the handle's aria-valuenow, emits
	// gosxstudio:rail-width-change or gosxstudio:rail-width-commit
	// depending on the committed flag.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SetRailWidth + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// CSS custom properties the method writes.
		"--studio-left-width",
		"--studio-right-width",
		// ARIA attribute the helper updates.
		"aria-valuenow",
		// The two rail-width events emitWorkbenchRailWidth dispatches.
		"gosxstudio:rail-width-change",
		"gosxstudio:rail-width-commit",
		// Side discriminator strings.
		`"left"`,
		`"right"`,
		// Clamp helper used to enforce min/max bounds.
		"clampNumber",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() setRailWidth must preserve %q contract", contract)
		}
	}
}

func TestBridgeShimAutoMountsOnDocumentReady(t *testing.T) {
	// Pre-2026-05-27 the deleted studio-engines.js bundle auto-mounted every
	// runtime contract at DOMContentLoaded — for WorkbenchRuntime that meant
	// two binds at boot: bindWorkbenchRailResizers and bindWorkbenchChrome.
	// When the bundle was deleted the per-slice BridgeShim correctly re-
	// published window.GoSXStudioWorkbenchRuntime but the two auto-mounts
	// were lost — rail-resizer drag handles and toolbar chrome were never
	// wired on initial load. The v0.4.1 fix landed this same contract for
	// fieldruntime; v0.5.0 restores it across the remaining six islands. The
	// other thirteen methods on the contract are host-driven writes (mode
	// switches, viewport changes, zoom updates, etc.), not boot bindings.
	shim := string(BridgeShim())
	for _, fragment := range []string{
		"DOMContentLoaded",
		"window.GoSXStudioWorkbenchRuntime.bindRailResizers",
		"window.GoSXStudioWorkbenchRuntime.bindChrome",
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
