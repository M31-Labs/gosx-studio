import { defineConfig, devices } from "@playwright/test";
import path from "node:path";
import baseConfig from "./playwright.config";

const enterpriseStandaloneFixtures = [
  "contenteditorruntime_test.ts",
  "contenteditor_focus_test.ts",
  "contenteditor_pointer_test.ts",
  "contenteditor_revision_test.ts",
  "contenteditor_save_adversarial_test.ts",
  "enterprise_editor_quality_test.ts",
  "mediaruntime_test.ts",
  "sectionorderruntime_test.ts",
  "state_history_modern_test.ts",
  "gallery_responsive_polish_test.ts",
  "gesture_cancel_test.ts",
];

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
  testMatch: enterpriseStandaloneFixtures,
  fullyParallel: false,
  workers: 1,
  outputDir: path.join(taskArtifactRoot, "playwright"),
  projects: [
    {
      name: "chromium",
      outputDir: projectOutputDir("chromium"),
      use: { ...devices["Desktop Chrome"] },
    },
    {
      name: "firefox",
      outputDir: projectOutputDir("firefox"),
      use: { ...devices["Desktop Firefox"] },
    },
    {
      name: "webkit",
      outputDir: projectOutputDir("webkit"),
      use: { ...devices["Desktop Safari"] },
    },
  ],
});
