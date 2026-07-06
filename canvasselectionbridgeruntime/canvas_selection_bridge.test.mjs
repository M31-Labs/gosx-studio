import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

test("selection bridge installs Studio primary global", () => {
  const source = readFileSync(new URL("./canvas_selection_bridge.js", import.meta.url), "utf8");
  const attrs = new Map();
  const window = {
    GoSXStudioSiteMapRuntime: {
      setState() {},
    },
    setInterval() {
      return 1;
    },
    clearInterval() {},
    addEventListener() {},
  };
  const document = {
    documentElement: {
      setAttribute(name, value) {
        attrs.set(name, value);
      },
    },
    querySelector() {
      return null;
    },
    addEventListener() {},
  };
  const context = vm.createContext({
    window,
    document,
    console,
  });

  vm.runInContext(source, context, { filename: "canvas_selection_bridge.js" });

  const runtime = window.GoSXStudioCanvasSelectionBridgeRuntime;
  assert.ok(runtime, "primary Studio runtime global should be installed");
  assert.equal(typeof runtime.tick, "function");
  assert.equal(typeof runtime.selectedID, "function");
  assert.equal(typeof runtime.selectedIDs, "function");
  assert.equal(typeof runtime.applied, "function");
  assert.equal(typeof runtime.appliedIDs, "function");
  assert.equal(attrs.get("data-gosx-studio-canvas-selection-bridge-bound"), "true");
});
