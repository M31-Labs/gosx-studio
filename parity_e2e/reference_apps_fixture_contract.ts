/**
 * Single source of truth for the opt-in shared-runtime browser matrix.
 *
 * All fifteen entries are the declared shared-runtime inventory. Every entry
 * must contribute at least one Playwright case to each canonical project;
 * this prevents a present-but-standalone Node file from silently disappearing
 * from the browser matrix.
 */
export const SHARED_RUNTIME_FIXTURES = [
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
  "runtime_teardown_test.ts",
  "contenteditor_touch_test.ts",
  "media_metadata_hardening_test.ts",
  "runtime_bundle_contract_test.ts",
] as const;

export const SHARED_RUNTIME_PROJECTS = ["chromium", "firefox", "webkit"] as const;

export const CURRENT_SHARED_RUNTIME_FIXTURES = SHARED_RUNTIME_FIXTURES.slice(0, 15);

export const H5_PENDING_SHARED_RUNTIME_FIXTURES = [] as const;
