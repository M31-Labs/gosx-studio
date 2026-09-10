import { defineConfig, devices } from "@playwright/test";
import path from "node:path";
import baseConfig from "./playwright.config";

const artifactRoot = process.env.GOSX_STUDIO_NO_RELOAD_ARTIFACT_ROOT
  ? path.resolve(process.env.GOSX_STUDIO_NO_RELOAD_ARTIFACT_ROOT)
  : path.resolve(__dirname, "../.tiller/scratch/codex/studio-browser-02-no-reload");

const projects = ["chromium", "firefox", "webkit"] as const;

function deviceFor(project: (typeof projects)[number]) {
  if (project === "chromium") return devices["Desktop Chrome"];
  if (project === "firefox") return devices["Desktop Firefox"];
  return devices["Desktop Safari"];
}

export default defineConfig({
  ...baseConfig,
  testDir: __dirname,
  testMatch: ["authoringruntime_test.ts", "operationruntime_test.ts"],
  fullyParallel: false,
  workers: 1,
  retries: 0,
  outputDir: artifactRoot,
  projects: projects.map((name) => ({
    name,
    outputDir: path.join(artifactRoot, name),
    use: { ...deviceFor(name) },
  })),
});
