import { expect, test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";
import enterpriseConfig from "./playwright.enterprise.config";
import {
  SHARED_RUNTIME_FIXTURES,
  SHARED_RUNTIME_PROJECTS,
} from "./reference_apps_fixture_contract";

test.describe("shared-runtime fixture contract", () => {
  test("keeps every frozen fixture selected by all three shared-runtime projects", () => {
    test.setTimeout(120_000);
    expect(new Set(SHARED_RUNTIME_FIXTURES).size).toBe(SHARED_RUNTIME_FIXTURES.length);
    expect(SHARED_RUNTIME_FIXTURES).toHaveLength(15);
    expect(enterpriseConfig.testMatch).toEqual([...SHARED_RUNTIME_FIXTURES]);

    const projects = enterpriseConfig.projects ?? [];
    expect(projects.map((project) => project.name)).toEqual([...SHARED_RUNTIME_PROJECTS]);
    expect(SHARED_RUNTIME_PROJECTS).toEqual(["chromium", "firefox", "webkit"]);

    for (const fixture of SHARED_RUNTIME_FIXTURES) {
      expect(existsSync(path.join(__dirname, fixture)), `${fixture} must exist before the matrix can pass`).toBe(true);
    }

    const cli = path.join(__dirname, "node_modules", "@playwright", "test", "cli.js");
    const config = path.join(__dirname, "playwright.enterprise.config.ts");
    const fixtureNames = SHARED_RUNTIME_FIXTURES.map((fixture) => fixture.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"));
    const fixturePattern = new RegExp(
      `^\\s+\\[([^\\]]+)\\]\\s+›\\s+(${fixtureNames.join("|")}):`,
      "gm",
    );
    for (const project of SHARED_RUNTIME_PROJECTS) {
      const output = execFileSync(
        process.execPath,
        [cli, "test", `--config=${config}`, ...SHARED_RUNTIME_FIXTURES, `--project=${project}`, "--list"],
        {
          cwd: __dirname,
          encoding: "utf8",
          stdio: ["ignore", "pipe", "pipe"],
          maxBuffer: 20 * 1024 * 1024,
        },
      );
      const counts = new Map<string, number>();
      for (const match of output.matchAll(fixturePattern)) {
        expect(match[1], `${project} --list selected another project`).toBe(project);
        counts.set(match[2], (counts.get(match[2]) ?? 0) + 1);
      }
      for (const fixture of SHARED_RUNTIME_FIXTURES) {
        expect(counts.get(fixture) ?? 0, `${project} --list must discover ${fixture}`).toBeGreaterThan(0);
      }
      const total = /Total:\s+(\d+)\s+tests?\s+in\s+(\d+)\s+files?/.exec(output);
      expect(total, `${project} --list must report a machine-readable total`).not.toBeNull();
      expect(Number(total?.[2])).toBe(SHARED_RUNTIME_FIXTURES.length);
    }
  });
});
