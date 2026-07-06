import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

function legacyMuddyGlobal(name) {
  return "__" + "muddy" + name;
}

test("WASM-free client installs Studio primary global only", () => {
  const source = readFileSync(new URL("./canvas_wasm_free_client.js", import.meta.url), "utf8");
  const attrs = new Map();
  const window = {
    addEventListener() {},
    devicePixelRatio: 1,
  };
  const document = {
    documentElement: {
      setAttribute(name, value) {
        attrs.set(name, value);
      },
    },
    readyState: "complete",
    querySelectorAll() {
      return [];
    },
    addEventListener() {},
  };
  const context = vm.createContext({
    window,
    document,
    console,
    navigator: {},
    performance: { now: () => 0 },
    setTimeout() {},
    HTMLCanvasElement: class HTMLCanvasElement {},
  });

  vm.runInContext(source, context, { filename: "canvas_wasm_free_client.js" });

  const runtime = window.GoSXStudioCanvasWASMFreeClientRuntime;
  assert.ok(runtime, "primary Studio runtime global should be installed");
  assert.equal(window[legacyMuddyGlobal("CanvasWasmFreeClient")], undefined);
  assert.equal(typeof runtime.mountAll, "function");
  assert.equal(attrs.get("data-gosx-canvas-wasm-free-client"), "true");
});
