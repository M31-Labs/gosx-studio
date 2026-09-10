import { defineConfig, devices } from "@playwright/test";
import path from "node:path";
import baseConfig from "./playwright.config";
import {
  REFERENCE_APP_FIXTURES,
  REFERENCE_APP_PROJECTS,
} from "./reference_apps_browser_contract";

const artifactRoot = process.env.GOSX_STUDIO_REFERENCE_APP_ARTIFACT_ROOT
  ? path.resolve(process.env.GOSX_STUDIO_REFERENCE_APP_ARTIFACT_ROOT)
  : path.resolve(__dirname, "../.tiller/scratch/codex/reference-app-browser-matrix");

function projectOutputDir(projectName: string): string {
  return path.join(artifactRoot, projectName);
}

export default defineConfig({
  ...baseConfig,
  testDir: __dirname,
  testMatch: [...REFERENCE_APP_FIXTURES],
  fullyParallel: false,
  workers: 1,
  retries: 0,
  outputDir: artifactRoot,
  projects: REFERENCE_APP_PROJECTS.map((name) => ({
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
