import { expect, test } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import referenceAppsConfig from "./playwright.reference-apps.config";
import {
  REFERENCE_APP_FIXTURES,
  REFERENCE_APP_PROJECTS,
  RELEASED_CONSUMER_CASE_COUNT,
  RELEASED_CONSUMER_FIXTURE_CASES,
} from "./reference_apps_browser_contract";

const releasedFixtures = RELEASED_CONSUMER_FIXTURE_CASES.map((entry) => entry.fixture);

function scriptFiles(script: string | undefined): string[] {
  return [...(script ?? "").matchAll(/reference_apps_[a-z0-9_]+_test\.ts/g)].map((match) => match[0]);
}

function listCases(project: string, fixtures: readonly string[]): string {
  const cli = path.join(__dirname, "node_modules", "@playwright", "test", "cli.js");
  const config = path.join(__dirname, "playwright.reference-apps.config.ts");
  return execFileSync(
    process.execPath,
    [cli, "test", `--config=${config}`, ...fixtures, `--project=${project}`, "--list"],
    {
      cwd: __dirname,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
      maxBuffer: 20 * 1024 * 1024,
    },
  );
}

type ListedCase = {
  project: string;
  fixture: string;
};

function listedCases(output: string): ListedCase[] {
  return [...output.matchAll(/^\s+\[([^\]]+)\]\s+›\s+(reference_apps_[a-z0-9_]+_test\.ts):/gm)].map((match) => ({
    project: match[1],
    fixture: match[2],
  }));
}

test.describe("reference-app browser matrix contract", () => {
  test("keeps package selection and the canonical three-engine config aligned", () => {
    const packageJSON = JSON.parse(readFileSync(path.join(__dirname, "package.json"), "utf8")) as {
      scripts?: Record<string, string>;
    };
    expect(scriptFiles(packageJSON.scripts?.["test:reference-apps"])).toEqual([...REFERENCE_APP_FIXTURES]);
    expect(scriptFiles(packageJSON.scripts?.["test:released-consumer"])).toEqual(releasedFixtures);
    for (const fixture of REFERENCE_APP_FIXTURES) {
      expect(existsSync(path.join(__dirname, fixture)), `${fixture} must exist before host matrix execution`).toBe(true);
    }
    expect(referenceAppsConfig.testMatch).toEqual([...REFERENCE_APP_FIXTURES]);
    expect(referenceAppsConfig.retries).toBe(0);
    expect((referenceAppsConfig.projects ?? []).map((project) => project.name)).toEqual([...REFERENCE_APP_PROJECTS]);
  });

  test("machine-checks the released 14-case manifest for every selected engine", () => {
    test.setTimeout(120_000);
    for (const project of REFERENCE_APP_PROJECTS) {
      const output = listCases(project, releasedFixtures);
      const entries = listedCases(output);
      expect(entries.every((entry) => entry.project === project), `${project} --list selected another project`).toBe(true);
      const total = /Total:\s+(\d+)\s+tests?\s+in\s+(\d+)\s+files?/.exec(output);
      expect(total, `${project} --list must report a machine-readable total`).not.toBeNull();
      expect(Number(total?.[1])).toBe(RELEASED_CONSUMER_CASE_COUNT);
      expect(Number(total?.[2])).toBe(releasedFixtures.length);
      const actualCounts = new Map<string, number>();
      for (const entry of entries) actualCounts.set(entry.fixture, (actualCounts.get(entry.fixture) ?? 0) + 1);
      for (const expected of RELEASED_CONSUMER_FIXTURE_CASES) {
        expect(actualCounts.get(expected.fixture), `${project} --list case count for ${expected.fixture}`).toBe(expected.cases);
      }
      expect([...actualCounts.keys()].sort()).toEqual([...releasedFixtures].sort());
    }
  });

  test("discovers every candidate fixture for every selected engine", () => {
    test.setTimeout(120_000);
    for (const project of REFERENCE_APP_PROJECTS) {
      const output = listCases(project, REFERENCE_APP_FIXTURES);
      const entries = listedCases(output);
      expect(entries.every((entry) => entry.project === project), `${project} --list selected another project`).toBe(true);
      expect([...new Set(entries.map((entry) => entry.fixture))].sort()).toEqual([...REFERENCE_APP_FIXTURES].sort());
    }
  });
});
