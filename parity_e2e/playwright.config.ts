// Playwright config for the gosx-studio runtime parity test harness.
//
// Phase 3 of the gosx-studio runtime burn-down (see
// ~/.hyphae/spaces/m31labs-gosx/plans/gosx-vm-unification-and-editor-bridge-burn-down.md)
// replaces seven JavaScript runtime contracts in `the legacy bundle (removed 2026-05-27)`
// and `the legacy bundle (removed 2026-05-27)` with compiled `.gsx` islands and engines.
// Each slice ships additively (legacy JS + island both live, gated by a
// feature flag in `studio.ShellConfig.FeatureFlags`) and proves equivalence
// against the legacy bundle through these Playwright suites before the JS
// implementation is deleted.
//
// The shared harness in `harness.ts` boots the muddy-noni-commerce editor
// route in two modes: baseline (legacy JS active) and candidate (island
// path active). Each per-contract suite (`<contract>_test.ts`) drives the
// same user interaction in both modes and asserts observable equivalence.

import { defineConfig, devices } from "@playwright/test";

const baseURL = process.env.GOSX_STUDIO_PARITY_BASE_URL ?? "http://127.0.0.1:3010";

export default defineConfig({
  testDir: "./",
  testMatch: ["**/*_test.ts"],

  // Parity tests boot two long-lived consumer processes per test file
  // (baseline + candidate). Parallel workers would race for the same
  // candidate-mode shell port. Serial-by-file is the safe default; the
  // harness can grant its own per-worker ports later if throughput matters.
  fullyParallel: false,
  workers: 1,

  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["github"]] : "list",

  use: {
    baseURL,
    actionTimeout: 10_000,
    navigationTimeout: 30_000,
    trace: "retain-on-failure",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
