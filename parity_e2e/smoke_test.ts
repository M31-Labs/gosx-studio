// Smoke test for the parity harness itself.
//
// Verifies that `bootBaseline` / `bootCandidate` reach the editor route
// without error in both modes. Slice tests build on this; if the smoke
// test is red, every parity assertion below it is unreliable.
//
// Tagged @smoke so the harness can be exercised in isolation via
// `npm run smoke` without running the per-contract parity suites.

import { test, expect } from "@playwright/test";
import { bootBaseline, bootCandidate, disposeBoot } from "./harness";

test.describe("@smoke parity harness", () => {
  test("boots baseline mode and reaches the editor route", async ({ browser }) => {
    const handle = await bootBaseline(browser);
    try {
      expect(handle.mode).toBe("baseline");
      await expect(handle.page.locator("[data-studio-canvas]").first()).toBeAttached();
    } finally {
      await disposeBoot(handle);
    }
  });

  test("boots candidate mode and reaches the editor route", async ({ browser }) => {
    const handle = await bootCandidate(browser);
    try {
      expect(handle.mode).toBe("candidate");
      await expect(handle.page.locator("[data-studio-canvas]").first()).toBeAttached();
    } finally {
      await disposeBoot(handle);
    }
  });
});
