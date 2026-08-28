import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

// Behavioral proof for handoff-31's inline-edit durable dispatch: this drives
// the REAL island_runtime.js source (not a reimplementation) through
// install()'s focusin/focusout commit flow against a minimal hand-rolled DOM
// (matching the precedent set by canvaswasmfreeruntime's own
// canvas_wasm_free_client.test.mjs — no jsdom dependency in this module).

function loadRuntime(context) {
  const source = readFileSync(new URL("./island_runtime.js", import.meta.url), "utf8");
  vm.createContext(context);
  vm.runInContext(source, context, { filename: "island_runtime.js" });
  return context.window.GoSXStudioInlineEditRuntime;
}

function makeRoot() {
  const listeners = {};
  return {
    __attrs: {},
    getAttribute(name) { return Object.prototype.hasOwnProperty.call(this.__attrs, name) ? this.__attrs[name] : null; },
    addEventListener(name, fn) { (listeners[name] = listeners[name] || []).push(fn); },
    trigger(name, event) { (listeners[name] || []).forEach((fn) => fn(event)); },
  };
}

function makeEditable(attrs, text) {
  const state = Object.assign({}, attrs);
  return {
    __isEditable: true,
    getAttribute(name) { return Object.prototype.hasOwnProperty.call(state, name) ? state[name] : null; },
    hasAttribute(name) { return Object.prototype.hasOwnProperty.call(state, name); },
    setAttribute(name, value) { state[name] = value; },
    removeAttribute(name) { delete state[name]; },
    get textContent() { return text; },
    set textContent(v) { text = v; },
    parentElement: null,
    blur() {},
  };
}

function baseContext() {
  const document = {
    readyState: "complete",
    querySelector() { return null; },
    addEventListener() {},
  };
  const window = { document, GoSXStudioAuthoringRuntime: undefined, GoSXStudioCollaborationRuntime: undefined };
  window.location = { href: "https://example.test/editor" };
  return { window, document, console };
}

test("durable-enabled field with available collaboration submits via GoSXStudioCollaborationRuntime and never fetches", () => {
  const context = baseContext();
  const submitted = [];
  context.window.GoSXStudioCollaborationRuntime = {
    available: () => true,
    expectedHead: (request) => "head-" + request.target.field,
    submit: (request) => { submitted.push(request); return Promise.resolve({ ok: true }); },
  };
  let fetchCalled = false;
  context.window.fetch = () => { fetchCalled = true; return Promise.resolve({ ok: true, json: () => Promise.resolve({}) }); };

  const runtime = loadRuntime(context);
  const root = makeRoot();
  const el = makeEditable({
    "data-studio-field": "home.hero.headline",
    "contenteditable": "true",
    "data-gosx-studio-durable-history": "true",
    "data-gosx-studio-durable-field": "hero.headline",
    "data-gosx-studio-durable-page-id": "page:home",
    "data-gosx-studio-durable-route": "/",
    "data-gosx-studio-durable-component": "home:hero",
  }, "Original headline");

  runtime.install(root, { action: "/legacy-action", csrfToken: "tok" });

  root.trigger("focusin", { target: el });
  el.textContent = "New durable headline";
  root.trigger("focusout", { target: el });

  assert.equal(submitted.length, 1, "expected exactly one durable operation submission");
  assert.equal(submitted[0].kind, "set-field");
  assert.equal(submitted[0].target.field, "hero.headline", "must use the durable-field override, not the raw canvas binding");
  assert.equal(submitted[0].target.pageId, "page:home");
  assert.equal(submitted[0].target.componentKey, "home:hero");
  assert.equal(submitted[0].value, "New durable headline");
  assert.equal(submitted[0].expectedTargetHead, "head-hero.headline", "must resolve expectedTargetHead via collaboration.expectedHead");
  assert.equal(fetchCalled, false, "the legacy save-control POST must NOT fire when the durable path is taken (absence assertion)");
});

test("durable-enabled field falls back to the legacy POST when collaboration is unavailable", () => {
  const context = baseContext();
  context.window.GoSXStudioCollaborationRuntime = { available: () => false, submit: () => Promise.reject(new Error("not mounted")) };
  let fetchCalled = false;
  let fetchAction = null;
  context.window.fetch = (action) => { fetchCalled = true; fetchAction = action; return Promise.resolve({ ok: true, json: () => Promise.resolve({}) }); };

  const runtime = loadRuntime(context);
  const root = makeRoot();
  const el = makeEditable({
    "data-studio-field": "home.hero.headline",
    "contenteditable": "true",
    "data-gosx-studio-durable-history": "true",
    "data-gosx-studio-durable-field": "hero.headline",
  }, "Original headline");

  runtime.install(root, { action: "/legacy-action", csrfToken: "tok" });
  root.trigger("focusin", { target: el });
  el.textContent = "Fallback headline";
  root.trigger("focusout", { target: el });

  assert.equal(fetchCalled, true, "an unavailable collaboration transport must fall back to the legacy POST");
  assert.equal(fetchAction, "/legacy-action");
});

test("a field without the durable opt-in keeps using the legacy POST even when collaboration IS available", () => {
  const context = baseContext();
  let submitCalled = false;
  context.window.GoSXStudioCollaborationRuntime = { available: () => true, expectedHead: () => "", submit: () => { submitCalled = true; return Promise.resolve({}); } };
  let fetchCalled = false;
  context.window.fetch = () => { fetchCalled = true; return Promise.resolve({ ok: true, json: () => Promise.resolve({}) }); };

  const runtime = loadRuntime(context);
  const root = makeRoot();
  const el = makeEditable({ "data-studio-field": "tagline", "contenteditable": "true" }, "Original tagline");

  runtime.install(root, { action: "/legacy-action", csrfToken: "tok" });
  root.trigger("focusin", { target: el });
  el.textContent = "New tagline";
  root.trigger("focusout", { target: el });

  assert.equal(submitCalled, false, "a non-opted-in field must never route through the durable path");
  assert.equal(fetchCalled, true, "a non-opted-in field must keep using the legacy POST");
});
