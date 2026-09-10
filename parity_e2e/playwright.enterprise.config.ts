import { defineConfig, devices } from "@playwright/test";
import path from "node:path";
import baseConfig from "./playwright.config";
import {
  SHARED_RUNTIME_FIXTURES,
  SHARED_RUNTIME_PROJECTS,
} from "./reference_apps_fixture_contract";

const taskArtifactRoot = process.env.GOSX_STUDIO_ENTERPRISE_ARTIFACT_ROOT
  ? path.resolve(process.env.GOSX_STUDIO_ENTERPRISE_ARTIFACT_ROOT)
  : path.resolve(
      __dirname,
      "../.tiller/scratch/codex/enterprise-polish-20260827/crossbrowser24",
    );

// The opt-in config owns an isolated default. A caller may provide a unique
// root for a run, while the normal playwright.config.ts keeps its canonical
// quality04 location when this config is not selected.
const qualityArtifactRoot = process.env.QUALITY_ARTIFACT_ROOT
  ? path.resolve(process.env.QUALITY_ARTIFACT_ROOT)
  : path.join(taskArtifactRoot, "quality");
process.env.QUALITY_ARTIFACT_ROOT = qualityArtifactRoot;

function projectOutputDir(projectName: string): string {
  return path.join(taskArtifactRoot, "playwright", projectName);
}

export default defineConfig({
  ...baseConfig,
  testDir: __dirname,
  testMatch: [...SHARED_RUNTIME_FIXTURES],
  fullyParallel: false,
  workers: 1,
  outputDir: path.join(taskArtifactRoot, "playwright"),
  projects: SHARED_RUNTIME_PROJECTS.map((name) => ({
    name,
    outputDir: projectOutputDir(name),
    use: {
      ...(name === "chromium"
        ? devices["Desktop Chrome"]
        : name === "firefox"
          ? devices["Desktop Firefox"]
          : devices["Desktop Safari"]),
    },
  })),
});
