import type { Page } from "@playwright/test";

export const DEFAULT_CANVAS_SELECTOR = "canvas[data-gosx-surface-kind='canvas2d']";
export const WASM_FREE_CANVAS_SELECTOR = "canvas[data-gosx-canvas-wasm-free='true']";

export type CanvasPaintSample = {
  painted: boolean;
  width: number;
  height: number;
  nonBackground: number;
  distinctColors: number;
  reason: string;
};

export type CanvasRenderEvidence = {
  webgpuRoute: boolean;
  fallback2D: boolean;
  backendAttr: string | null;
  surfaceKind: string | null;
  surfaceID: string | null;
  webgpuLoaded: boolean;
  webgpuAPI: boolean;
  webgpuProbe: unknown;
  webgpuDiagnostics: unknown;
  hostAttrs: Record<string, string | null>;
  fallbackWarnings: string[];
  sampled2DPaint: boolean;
  twoDPaint: CanvasPaintSample | null;
  reason: string;
};

type WaitOptions = {
  selector?: string;
  timeoutMS?: number;
};

// Matches the WebGPU->fallback signals emitted on BOTH editor paths:
//  - wasm-free JS client (canvas_wasm_free_client.js): "wasm-free static WebGPU canvas unavailable (...)"
//  - full-WASM gosx CanvasBoard / core probe (bootstrap-feature-scene3d): "WebGPU probe: requestAdapter returned null", "No available adapters"
// Previously only the first two were matched, so the full-WASM canvas tests never
// recognised a no-GPU fallback and timed out instead of asserting the 2D route.
const WEBGPU_FALLBACK_RE =
  /(?:canvas2d WebGPU backend unavailable|wasm-free static WebGPU canvas unavailable|WebGPU probe: requestAdapter returned null|No available adapters|WebGPU backend unavailable|falling back to (?:Canvas2D|WebGL))/i;

export function canvasWebGPUFallbackWarnings(consoleWarnings: string[]): string[] {
  return consoleWarnings.filter((line) => WEBGPU_FALLBACK_RE.test(line));
}

export async function waitForCanvasBoardRenderEvidence(
  page: Page,
  consoleWarnings: string[],
  options: WaitOptions = {},
): Promise<CanvasRenderEvidence> {
  const selector = options.selector ?? DEFAULT_CANVAS_SELECTOR;
  const deadline = Date.now() + (options.timeoutMS ?? 60_000);
  let last = await readCanvasBoardRouteEvidence(page, consoleWarnings, selector);
  while (Date.now() < deadline) {
    last = await readCanvasBoardRouteEvidence(page, consoleWarnings, selector);
    if (last.webgpuRoute) return last;
    if (last.fallbackWarnings.length > 0) {
      const paint = await sampleCanvas2DPaint(page, selector);
      last = {
        ...last,
        fallback2D: paint.painted,
        sampled2DPaint: true,
        twoDPaint: paint,
        reason: paint.painted ? "fallback-2d-painted" : `fallback-2d-${paint.reason}`,
      };
      if (last.fallback2D) return last;
    }
    await page.waitForTimeout(250);
  }
  return last;
}

export function formatCanvasRenderEvidence(evidence: CanvasRenderEvidence): string {
  return JSON.stringify({
    webgpuRoute: evidence.webgpuRoute,
    fallback2D: evidence.fallback2D,
    backendAttr: evidence.backendAttr,
    surfaceKind: evidence.surfaceKind,
    surfaceID: evidence.surfaceID,
    webgpuLoaded: evidence.webgpuLoaded,
    webgpuAPI: evidence.webgpuAPI,
    webgpuProbe: evidence.webgpuProbe,
    webgpuDiagnostics: evidence.webgpuDiagnostics,
    hostAttrs: evidence.hostAttrs,
    fallbackWarnings: evidence.fallbackWarnings,
    sampled2DPaint: evidence.sampled2DPaint,
    twoDPaint: evidence.twoDPaint,
    reason: evidence.reason,
  });
}

export async function canvasFingerprint2D(
  page: Page,
  selector = DEFAULT_CANVAS_SELECTOR,
): Promise<{ signature: number; nonBackground: number }> {
  return page.evaluate((canvasSelector) => {
    const sample = (window as unknown as {
      __gosx_canvas_e2e_fingerprint_2d?: (selector: string) => { signature: number; nonBackground: number };
    }).__gosx_canvas_e2e_fingerprint_2d;
    if (typeof sample === "function") return sample(canvasSelector);
    return { signature: -1, nonBackground: 0 };
  }, selector);
}

export async function waitForCanvasWebGPUFrameAdvance(
  page: Page,
  from: number,
  options: WaitOptions = {},
): Promise<number | null> {
  const selector = options.selector ?? DEFAULT_CANVAS_SELECTOR;
  const deadline = Date.now() + (options.timeoutMS ?? 8_000);
  let last = await readCanvasWebGPUFrameSeq(page, selector);
  while (Date.now() < deadline) {
    last = await readCanvasWebGPUFrameSeq(page, selector);
    if (last > from) return last;
    await page.waitForTimeout(100);
  }
  return last > from ? last : null;
}

export async function readCanvasWebGPUFrameSeq(page: Page, selector = DEFAULT_CANVAS_SELECTOR): Promise<number> {
  return page.evaluate((canvasSelector) => {
    const canvas = document.querySelector(canvasSelector);
    const host = canvas && canvas.parentElement;
    const raw = host && host.getAttribute("data-gosx-scene3d-webgpu-frame-seq");
    const n = Number(raw || 0);
    return Number.isFinite(n) ? n : 0;
  }, selector);
}

async function readCanvasBoardRouteEvidence(
  page: Page,
  consoleWarnings: string[],
  selector: string,
): Promise<CanvasRenderEvidence> {
  const snapshot = await page.evaluate((canvasSelector) => {
    const w = window as unknown as {
      __gosx_scene3d_webgpu_loaded?: boolean;
      __gosx_scene3d_webgpu_api?: { createRenderer?: unknown; diagnostics?: () => unknown };
      __gosx_scene3d_webgpu_probe?: () => unknown;
      __gosx_scene3d_webgpu_diagnostics?: () => unknown;
    };
    const canvas = document.querySelector(canvasSelector);
    const host = canvas && canvas.parentElement;
    const hostAttrs: Record<string, string | null> = {};
    for (const name of [
      "data-gosx-scene3d-webgpu-frame-seq",
      "data-gosx-scene3d-webgpu-frame-at",
      "data-gosx-scene3d-webgpu-mesh-objects",
      "data-gosx-scene3d-webgpu-line-entries",
      "data-gosx-scene3d-webgpu-last-error",
      "data-gosx-canvas-wasm-free-webgpu",
      "data-gosx-canvas-wasm-free-fallback",
      "data-gosx-canvas-wasm-free-fallback-reason",
    ]) {
      hostAttrs[name] = host instanceof Element ? host.getAttribute(name) : null;
    }
    let webgpuProbe: unknown = null;
    let webgpuDiagnostics: unknown = null;
    try {
      webgpuProbe = typeof w.__gosx_scene3d_webgpu_probe === "function" ? w.__gosx_scene3d_webgpu_probe() : null;
    } catch (error) {
      webgpuProbe = { error: String(error) };
    }
    try {
      if (typeof w.__gosx_scene3d_webgpu_diagnostics === "function") {
        webgpuDiagnostics = w.__gosx_scene3d_webgpu_diagnostics();
      } else if (w.__gosx_scene3d_webgpu_api && typeof w.__gosx_scene3d_webgpu_api.diagnostics === "function") {
        webgpuDiagnostics = w.__gosx_scene3d_webgpu_api.diagnostics();
      }
    } catch (error) {
      webgpuDiagnostics = { error: String(error) };
    }
    return {
      backendAttr: canvas instanceof Element ? canvas.getAttribute("data-gosx-canvas-backend") : null,
      surfaceKind: canvas instanceof Element ? canvas.getAttribute("data-gosx-surface-kind") : null,
      surfaceID: canvas instanceof Element ? canvas.getAttribute("data-gosx-surface-id") : null,
      webgpuLoaded: w.__gosx_scene3d_webgpu_loaded === true,
      webgpuAPI: !!(w.__gosx_scene3d_webgpu_api && typeof w.__gosx_scene3d_webgpu_api.createRenderer === "function"),
      webgpuProbe,
      webgpuDiagnostics,
      hostAttrs,
    };
  }, selector);
  const frameSeq = Number(snapshot.hostAttrs["data-gosx-scene3d-webgpu-frame-seq"] || 0);
  const webgpuRoute = snapshot.backendAttr === "webgpu" && Number.isFinite(frameSeq) && frameSeq > 0;
  const fallbackWarnings = canvasWebGPUFallbackWarnings(consoleWarnings);
  return {
    ...snapshot,
    webgpuRoute,
    fallback2D: false,
    fallbackWarnings,
    sampled2DPaint: false,
    twoDPaint: null,
    reason: webgpuRoute ? "webgpu-frame-published" : fallbackWarnings.length > 0 ? "fallback-warning-observed" : "pending-route-evidence",
  };
}

async function sampleCanvas2DPaint(page: Page, selector: string): Promise<CanvasPaintSample> {
  await install2DSampler(page);
  return page.evaluate((canvasSelector) => {
    const sample = (window as unknown as {
      __gosx_canvas_e2e_sample_2d?: (selector: string) => CanvasPaintSample;
    }).__gosx_canvas_e2e_sample_2d;
    if (typeof sample !== "function") {
      return { painted: false, width: 0, height: 0, nonBackground: 0, distinctColors: 0, reason: "sampler-missing" };
    }
    return sample(canvasSelector);
  }, selector);
}

async function install2DSampler(page: Page) {
  await page.evaluate(() => {
    const w = window as unknown as {
      __gosx_canvas_e2e_sample_2d?: (selector: string) => CanvasPaintSample;
      __gosx_canvas_e2e_fingerprint_2d?: (selector: string) => { signature: number; nonBackground: number };
    };
    if (typeof w.__gosx_canvas_e2e_sample_2d === "function" && typeof w.__gosx_canvas_e2e_fingerprint_2d === "function") return;
    const read = (selector: string) => {
      const canvas = document.querySelector(selector);
      if (!(canvas instanceof HTMLCanvasElement)) {
        return { ok: false, width: 0, height: 0, data: null as Uint8ClampedArray | null, reason: "no-canvas" };
      }
      const ctx = canvas.getContext("2d");
      if (!ctx) {
        return { ok: false, width: canvas.width, height: canvas.height, data: null as Uint8ClampedArray | null, reason: "no-2d-context" };
      }
      const width = canvas.width;
      const height = canvas.height;
      if (width === 0 || height === 0) {
        return { ok: false, width, height, data: null as Uint8ClampedArray | null, reason: "zero-backing-store" };
      }
      try {
        return { ok: true, width, height, data: ctx.getImageData(0, 0, width, height).data, reason: "sampled" };
      } catch (error) {
        return { ok: false, width, height, data: null as Uint8ClampedArray | null, reason: `getImageData-threw:${String(error)}` };
      }
    };
    w.__gosx_canvas_e2e_sample_2d = (selector: string) => {
      const sampled = read(selector);
      if (!sampled.ok || !sampled.data) {
        return { painted: false, width: sampled.width, height: sampled.height, nonBackground: 0, distinctColors: 0, reason: sampled.reason };
      }
      const bg = { r: 0x0f, g: 0x17, b: 0x20 };
      const tol = 10;
      let nonBackground = 0;
      const colors = new Set<number>();
      const stride = 4 * Math.max(1, Math.floor((sampled.width * sampled.height) / 200000));
      for (let i = 0; i < sampled.data.length; i += stride) {
        const r = sampled.data[i];
        const g = sampled.data[i + 1];
        const b = sampled.data[i + 2];
        const a = sampled.data[i + 3];
        if (a === 0) continue;
        colors.add((r << 16) | (g << 8) | b);
        if (Math.abs(r - bg.r) > tol || Math.abs(g - bg.g) > tol || Math.abs(b - bg.b) > tol) nonBackground++;
      }
      return {
        painted: nonBackground > 0 && colors.size > 1,
        width: sampled.width,
        height: sampled.height,
        nonBackground,
        distinctColors: colors.size,
        reason: nonBackground > 0 ? "painted" : "background-only",
      };
    };
    w.__gosx_canvas_e2e_fingerprint_2d = (selector: string) => {
      const sampled = read(selector);
      if (!sampled.ok || !sampled.data) return { signature: -1, nonBackground: 0 };
      const bg = { r: 0x0f, g: 0x17, b: 0x20 };
      const tol = 10;
      let sig = 2166136261 >>> 0;
      let nonBackground = 0;
      const stride = 4 * Math.max(1, Math.floor((sampled.width * sampled.height) / 50000));
      for (let i = 0; i < sampled.data.length; i += stride) {
        const r = sampled.data[i], g = sampled.data[i + 1], b = sampled.data[i + 2], a = sampled.data[i + 3];
        if (a === 0) continue;
        if (Math.abs(r - bg.r) > tol || Math.abs(g - bg.g) > tol || Math.abs(b - bg.b) > tol) {
          nonBackground++;
          sig ^= (i & 0xffff) ^ ((r << 16) | (g << 8) | b);
          sig = Math.imul(sig, 16777619) >>> 0;
        }
      }
      return { signature: sig, nonBackground };
    };
  });
}
