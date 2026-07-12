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
//  1. The feature-flag key retained as a studio.ShellConfig host probe/string
//     contract for the .gsx-authored islands in this package.
//  2. The JS shim that the runtime bundle appends so the
//     window.GoSXStudioWorkbenchRuntime methods delegate directly to the
//     islands.
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
	// Naming convention: "<contract>-runtime-islands". The key remains a
	// host-visible probe/string contract even though BridgeShim no longer
	// gates calls on it.
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
	// The shim must reference each method-specific island global and the
	// public global it shims (window.GoSXStudioWorkbenchRuntime). The
	// IslandGlobals struct holds the canonical names; any drift between
	// the struct and the shim is caught here.
	for _, fragment := range []string{
		"window.GoSXStudioWorkbenchRuntime",
		"function delegate(islandGlobal)",
		"var island = window[islandGlobal]",
		"return island.apply(null, arguments)",
		"return undefined",
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
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
	for _, stale := range []string{
		"FLAG" + "_ATTR",
		"flag" + "Enabled",
		"data-gosx-studio-feature-flag-" + "workbench-runtime-islands",
		"if (!" + "flag" + "Enabled()) return undefined",
		"legacy " + "path",
		"fallback " + "branch",
		"fallback " + "path",
	} {
		if strings.Contains(shim, stale) {
			t.Fatalf("BridgeShim() contains stale gate/fallback fragment %q:\n%s", stale, shim)
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
		// Viewport-island opt-out marker so the mounted viewport island can
		// own preview-shell writes without duplicate syncs.
		"data-studio-viewport-island",
		// Viewport label readout selector.
		"data-studio-viewport-label",
		// Viewport controls current-state and pressed-state selectors.
		"data-studio-viewport-current",
		"button[data-studio-viewport], [role='button'][data-studio-viewport]",
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

func TestIslandRuntimeJSSyncViewportUpdatesDOMState(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	runtimeSource, err := json.Marshal(string(IslandRuntimeJS()))
	if err != nil {
		t.Fatalf("marshal runtime source: %v", err)
	}
	script := `
const runtimeSource = ` + string(runtimeSource) + `;

class TestEvent {
  constructor(type) { this.type = type; }
}

class TestCustomEvent extends TestEvent {
  constructor(type, options) {
    super(type);
    this.detail = options && options.detail ? options.detail : {};
  }
}

class TestEventTarget {
  constructor() { this.listeners = {}; }
  addEventListener(type, listener) {
    (this.listeners[type] || (this.listeners[type] = [])).push(listener);
  }
  dispatchEvent(event) {
    const listeners = this.listeners[event.type] || [];
    for (const listener of listeners) listener(event);
    return true;
  }
}

class TestElement extends TestEventTarget {
  constructor(tagName) {
    super();
    this.tagName = tagName.toUpperCase();
    this.attributes = {};
    this.children = [];
    this.textContent = "";
  }
  setAttribute(name, value) { this.attributes[name] = String(value); }
  getAttribute(name) {
    return Object.prototype.hasOwnProperty.call(this.attributes, name) ? this.attributes[name] : null;
  }
  appendChild(child) {
    child.parentNode = this;
    this.children.push(child);
    return child;
  }
  querySelector(selector) {
    return this.querySelectorAll(selector)[0] || null;
  }
  querySelectorAll(selector) {
    const selectors = selector.split(",").map((item) => item.trim()).filter(Boolean);
    const results = [];
    const visit = (node) => {
      for (const child of node.children) {
        if (selectors.some((single) => matchesSingleSelector(child, single))) results.push(child);
        visit(child);
      }
    };
    visit(this);
    return results;
  }
}

class TestDocument extends TestEventTarget {
  constructor() {
    super();
    this.body = new TestElement("body");
  }
  createElement(tagName) { return new TestElement(tagName); }
}

function matchesSingleSelector(element, selector) {
  if (!selector) return false;
  let rest = selector;
  const tag = rest.match(/^[a-zA-Z][a-zA-Z0-9-]*/);
  if (tag) {
    if (element.tagName.toLowerCase() !== tag[0].toLowerCase()) return false;
    rest = rest.slice(tag[0].length);
  }
  const classPattern = /\.([a-zA-Z0-9_-]+)/g;
  let classMatch;
  while ((classMatch = classPattern.exec(rest)) !== null) {
    const classes = (element.getAttribute("class") || "").split(/\s+/);
    if (classes.indexOf(classMatch[1]) === -1) return false;
  }
  rest = rest.replace(classPattern, "");
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
  return consumedAttrs === "" && (Boolean(tag) || sawAttribute || selector.indexOf(".") === 0);
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
const shell = document.createElement("section");
shell.setAttribute("class", "editor-preview-shell");
shell.setAttribute("data-studio-preview-viewport", "desktop");
const viewportRoot = document.createElement("div");
viewportRoot.setAttribute("data-studio-viewport-current", "desktop");
const previousButton = document.createElement("button");
previousButton.setAttribute("data-studio-viewport", "desktop");
previousButton.setAttribute("aria-pressed", "true");
const matchingButton = document.createElement("button");
matchingButton.setAttribute("data-studio-viewport", "tablet");
matchingButton.setAttribute("aria-pressed", "false");
const label = document.createElement("span");
label.setAttribute("data-studio-viewport-label", "true");
label.textContent = "Desktop";

form.appendChild(shell);
form.appendChild(viewportRoot);
viewportRoot.appendChild(previousButton);
viewportRoot.appendChild(matchingButton);
form.appendChild(label);
document.body.appendChild(form);

let viewportEvent = null;
let resizeEvents = 0;
document.addEventListener("gosxstudio:workbench-viewport-change", (event) => {
  viewportEvent = event;
});
window.addEventListener("resize", () => {
  resizeEvents++;
});

window.__gosx_workbench_runtime_island_syncViewport(form, "tablet");

assertEqual(form.getAttribute("data-studio-breakpoint"), "tablet", "form breakpoint");
assertEqual(shell.getAttribute("data-studio-preview-viewport"), "tablet", "preview shell viewport");
assertEqual(viewportRoot.getAttribute("data-studio-viewport-current"), "tablet", "viewport controls current attribute");
assertEqual(matchingButton.getAttribute("aria-pressed"), "true", "matching viewport button aria-pressed");
assertEqual(previousButton.getAttribute("aria-pressed"), "false", "previous viewport button aria-pressed");
assertEqual(label.textContent, "Tablet", "viewport label text");
if (!viewportEvent) throw new Error("expected gosxstudio:workbench-viewport-change event");
assertEqual(viewportEvent.detail.viewport, "tablet", "viewport event detail.viewport");
assertEqual(viewportEvent.detail.form, form, "viewport event detail.form");
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
		t.Fatalf("node DOM syncViewport check failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
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
	// the globals the island runtime publishes. Order matters so the public
	// runtime can delegate immediately when the island globals are present.
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

// TestBindChromeIslandRestoresWorkingStateAcrossSimulatedSaveReload proves
// the "save discards mode/selection/scroll" fix end to end in a real JS
// engine: bindChrome on the "pre-save" form sets a non-default mode and a
// selection, then a FRESH form (simulating the DOM a host form POST's page
// navigation produces — new elements, but the same sessionStorage, exactly
// as a real browser tab preserves sessionStorage across same-tab
// navigation) is bound, and must come up already in the persisted mode with
// the persisted selection restored, not the server-rendered "home" default.
func TestBindChromeIslandRestoresWorkingStateAcrossSimulatedSaveReload(t *testing.T) {
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
  constructor() { this.listeners = new Map(); }
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
  constructor(type, options = {}) { this.type = type; this.bubbles = Boolean(options.bubbles); }
}

class TestCustomEvent extends TestEvent {
  constructor(type, options = {}) { super(type, options); this.detail = options.detail || null; }
}

class TestElement extends TestEventTarget {
  constructor(tagName) {
    super();
    this.tagName = tagName.toUpperCase();
    this.attributes = new Map();
    this.children = [];
    this.parentElement = null;
    this.style = { properties: new Map(), getPropertyValue(name) { return this.properties.get(name) || ""; }, setProperty(name, value) { this.properties.set(name, value); } };
    this.scrollTop = 0;
    this.scrollLeft = 0;
    this.classSet = new Set();
    this.classList = {
      toggle: (name, force) => {
        const on = force === undefined ? !this.classSet.has(name) : Boolean(force);
        if (on) this.classSet.add(name); else this.classSet.delete(name);
        return on;
      },
      contains: (name) => this.classSet.has(name),
    };
    const el = this;
    this.dataset = new Proxy({}, {
      get(_, prop) {
        const attr = "data-" + String(prop).replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
        const value = el.getAttribute(attr);
        return value === null ? undefined : value;
      },
      set(_, prop, value) {
        const attr = "data-" + String(prop).replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
        el.setAttribute(attr, value);
        return true;
      }
    });
  }
  appendChild(child) { child.parentElement = this; this.children.push(child); return child; }
  setAttribute(name, value) { this.attributes.set(name, String(value)); }
  getAttribute(name) { return this.attributes.has(name) ? this.attributes.get(name) : null; }
  hasAttribute(name) { return this.attributes.has(name); }
  removeAttribute(name) { this.attributes.delete(name); }
  matches(selector) { return selector.split(",").some((part) => matchesSingleSelector(this, part.trim())); }
  querySelector(selector) { return this.querySelectorAll(selector)[0] || null; }
  querySelectorAll(selector) {
    const found = [];
    const visit = (node) => { for (const child of node.children) { if (child.matches(selector)) found.push(child); visit(child); } };
    visit(this);
    return found;
  }
  closest(selector) {
    let node = this;
    while (node) { if (node.matches(selector)) return node; node = node.parentElement; }
    return null;
  }
}

class TestDocument extends TestEventTarget {
  constructor() { super(); this.body = new TestElement("body"); }
  createElement(tagName) { return new TestElement(tagName); }
  querySelector(selector) { return this.body.querySelector(selector); }
  querySelectorAll(selector) { return this.body.querySelectorAll(selector); }
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

// A real browser tab's sessionStorage survives a same-tab form POST
// navigation; this in-memory stub models exactly that (module-scope object,
// shared across the two "page loads" simulated below, never reset).
const sessionStorageBacking = new Map();
const sessionStorage = {
  getItem(key) { return sessionStorageBacking.has(key) ? sessionStorageBacking.get(key) : null; },
  setItem(key, value) { sessionStorageBacking.set(key, value); },
};

function freshWindow() {
  const document = new TestDocument();
  const rafCallbacks = [];
  const window = new TestEventTarget();
  window.document = document;
  window.sessionStorage = sessionStorage;
  window.localStorage = { getItem() { return null; }, setItem() {} };
  window.requestAnimationFrame = (callback) => { rafCallbacks.push(callback); return rafCallbacks.length; };
  window.setTimeout = (callback) => { rafCallbacks.push(callback); return rafCallbacks.length; };
  window.matchMedia = () => ({ matches: false });
  globalThis.window = window;
  globalThis.document = document;
  globalThis.Event = TestEvent;
  globalThis.CustomEvent = TestCustomEvent;
  return { document, window, rafCallbacks };
}

function buildForm(document) {
  const form = document.createElement("form");
  form.setAttribute("data-editor-workbench", "true");
  form.setAttribute("data-studio-workbench", "true");
  // Server always renders the default mode — proving the restore overrides
  // this, not merely reads it, is the whole point of this test.
  form.setAttribute("data-studio-mode", "home");
  const stage = document.createElement("div");
  stage.setAttribute("data-studio-stage", "true");
  form.appendChild(stage);
  const homePanel = document.createElement("section");
  homePanel.setAttribute("data-studio-mode-panel", "home");
  homePanel.classList.toggle("editor-panel", true);
  const advancedPanel = document.createElement("section");
  advancedPanel.setAttribute("data-studio-mode-panel", "advanced");
  advancedPanel.classList.toggle("editor-panel", true);
  form.appendChild(homePanel);
  form.appendChild(advancedPanel);
  document.body.appendChild(form);
  return { form, stage, homePanel, advancedPanel };
}

// ---- "Page load 1" (pre-save): operator switches to Advanced and selects an object.
{
  const { document } = freshWindow();
  eval(runtimeSource);
  const { form, stage, homePanel, advancedPanel } = buildForm(document);
  stage.scrollTop = 240;
  stage.scrollLeft = 12;

  window.__gosx_workbench_runtime_island_bindChrome(form);
  assertEqual(form.getAttribute("data-studio-mode"), "home", "initial bind honors server-rendered default before any mode switch");

  window.__gosx_workbench_runtime_island_setMode(form, "advanced", false);
  assertEqual(form.getAttribute("data-studio-mode"), "advanced", "setMode switches the form to advanced");
  assertEqual(Boolean(advancedPanel.hidden), false, "advanced panel visible in advanced mode");
  assertEqual(Boolean(homePanel.hidden), true, "home panel hidden outside home mode");

  form.setAttribute("data-studio-selection", "home:hero");
  // The MutationObserver persists selection changes; this Node stub has no
  // MutationObserver, so persist directly the same way the observer
  // callback would (proving the read/restore half of the contract without
  // requiring a MutationObserver polyfill in this harness).
  sessionStorage.setItem("gosx-studio-editor-working-state", JSON.stringify(Object.assign(
    JSON.parse(sessionStorage.getItem("gosx-studio-editor-working-state") || "{}"),
    { selection: "home:hero", scrollTop: stage.scrollTop, scrollLeft: stage.scrollLeft }
  )));
}

// ---- "Page load 2" (post-save): a save POST reloads the page. The server
// still renders the DEFAULT mode (it has no knowledge of client working
// state) — the fresh form below models exactly that regression.
{
  const { document, rafCallbacks } = freshWindow();
  eval(runtimeSource);
  const { form, stage, homePanel, advancedPanel } = buildForm(document);
  assertEqual(form.getAttribute("data-studio-mode"), "home", "fresh post-save DOM is server-rendered back to the default mode");

  window.__gosx_workbench_runtime_island_bindChrome(form);

  assertEqual(form.getAttribute("data-studio-mode"), "advanced", "bindChrome must restore the persisted mode across the simulated save reload, not the server default");
  assertEqual(Boolean(advancedPanel.hidden), false, "advanced panel visible again after restore");
  assertEqual(Boolean(homePanel.hidden), true, "home panel stays hidden after restore (still in advanced mode)");
  assertEqual(form.getAttribute("data-studio-selection"), "home:hero", "bindChrome must restore the persisted selection across the simulated save reload");

  for (const callback of rafCallbacks.splice(0)) callback();
  assertEqual(stage.scrollTop, 240, "bindChrome must restore the persisted scroll position (top) across the simulated save reload");
  assertEqual(stage.scrollLeft, 12, "bindChrome must restore the persisted scroll position (left) across the simulated save reload");
}
`

	cmd := exec.Command(node)
	cmd.Stdin = strings.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("node working-state restore check failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}
