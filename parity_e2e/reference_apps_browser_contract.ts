/**
 * Canonical reference-app suite and browser-project contract.
 *
 * The package scripts and the supplemental Actions matrix both consume these
 * names. The contract test compares the script/configuration with this list
 * so adding a suite cannot silently update only one lane.
 */
export const REFERENCE_APP_FIXTURES = [
  "reference_apps_authoring_test.ts",
  "reference_apps_visual_a11y_test.ts",
  "reference_apps_performance_test.ts",
  "reference_apps_canvas_test.ts",
  "reference_apps_canvas_interaction_test.ts",
  "reference_apps_canvas_selection_test.ts",
  "reference_apps_canvas_authoring_test.ts",
  "reference_apps_canvas_default_test.ts",
  "reference_apps_canvas_marquee_nav_test.ts",
  "reference_apps_canvas_wasm_free_test.ts",
  "reference_apps_canvas_html_surface_test.ts",
  "reference_apps_canvas_inline_edit_test.ts",
  "reference_apps_canvas_draft_publish_test.ts",
  "reference_apps_restore_revision_test.ts",
  "reference_apps_canvas_cards_lod_test.ts",
  "reference_apps_canvas_thumbnails_test.ts",
  "reference_apps_derived_nav_test.ts",
  "reference_apps_navigation_draft_preview_test.ts",
  "reference_apps_preview_mobile_width_test.ts",
  "reference_apps_stripe_plugin_test.ts",
  "reference_apps_canvas_first_test.ts",
  "reference_apps_collaboration_test.ts",
  "reference_apps_interactions_test.ts",
  "reference_apps_media_asset_test.ts",
  "reference_apps_collaboration_direct_edit_form_test.ts",
  "reference_apps_collaboration_sequential_edit_test.ts",
  "reference_apps_interactions_sequential_test.ts",
  "reference_apps_editor_declutter_test.ts",
  "reference_apps_publish_changeset_test.ts",
  "reference_apps_editor_inspector_presentation_test.ts",
  "reference_apps_page_canvas_inline_edit_preview_refresh_test.ts",
  "reference_apps_enterprise_polish_test.ts",
  "reference_apps_modern_editor_test.ts",
  "reference_apps_page_cms_lifecycle_test.ts",
] as const;

export const REFERENCE_APP_PROJECTS = ["chromium", "firefox", "webkit"] as const;

export const RELEASED_CONSUMER_FIXTURE_CASES = [
  { fixture: "reference_apps_modern_editor_test.ts", cases: 4 },
  { fixture: "reference_apps_page_cms_lifecycle_test.ts", cases: 1 },
  { fixture: "reference_apps_page_canvas_inline_edit_preview_refresh_test.ts", cases: 1 },
  { fixture: "reference_apps_navigation_draft_preview_test.ts", cases: 1 },
  { fixture: "reference_apps_canvas_default_test.ts", cases: 1 },
  { fixture: "reference_apps_enterprise_polish_test.ts", cases: 6 },
] as const;

export const RELEASED_CONSUMER_CASE_COUNT = RELEASED_CONSUMER_FIXTURE_CASES.reduce(
  (total, entry) => total + entry.cases,
  0,
);
