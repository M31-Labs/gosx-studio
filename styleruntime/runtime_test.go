package styleruntime

import (
	"strings"
	"testing"
)

// The styleruntime package is the Go-side surface for the Phase 3 slice-5
// burn-down of GoSXStudioStyleRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip from the legacy JS implementation in assets/studio-engines.js
//     to the .gsx-authored islands in this package.
//  2. The JS shim that studio-engines.js appends so the
//     window.GoSXStudioStyleRuntime methods delegate to the islands when the
//     flag is on.
//  3. The island runtime JS that publishes window.__gosx_style_runtime_*
//     globals the BridgeShim delegates to.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-5-styleruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the cross-frame signal channel applyTheme / showImpact / restoreImpact
// write through (slice 6 will swap the transitional legacy-delegation pattern
// below for $preview.theme.* / $preview.style.impact.* subscribers).

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if FeatureFlagKey != "style-runtime-islands" {
		t.Fatalf("FeatureFlagKey must be %q per Phase 3 slice-5 contract; got %q", "style-runtime-islands", FeatureFlagKey)
	}
	if !strings.HasSuffix(FeatureFlagKey, "-runtime-islands") {
		t.Fatalf("FeatureFlagKey %q should end with -runtime-islands per Phase 3 convention", FeatureFlagKey)
	}
}

func TestBridgeShimDelegatesToIslandGlobals(t *testing.T) {
	shim := string(BridgeShim())
	if shim == "" {
		t.Fatal("BridgeShim() must return a non-empty JS snippet")
	}
	// The shim must reference each method-specific island global, the public
	// global it shims (window.GoSXStudioStyleRuntime), and the feature flag
	// so the legacy path stays reachable when the flag is off. The
	// IslandGlobals struct holds the canonical names; any drift between the
	// struct and the shim is caught here.
	for _, fragment := range []string{
		"window.GoSXStudioStyleRuntime",
		IslandGlobals.BindTheme,
		IslandGlobals.BindWorkbench,
		IslandGlobals.BindCSS,
		IslandGlobals.BindFonts,
		IslandGlobals.ApplyTheme,
		IslandGlobals.SyncControlButtons,
		IslandGlobals.ShowImpact,
		IslandGlobals.RestoreImpact,
		IslandGlobals.SetControlValue,
		IslandGlobals.ResetControlValue,
		FeatureFlagKey,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
}

func TestIslandRuntimeJSPublishesBindThemeGlobal(t *testing.T) {
	// Method 1/10: bindTheme(root) — legacy bindTheme at studio-engines.js:1421.
	// Binds theme kit / template / palette / image-ratio / custom-template /
	// style-class / color-token controls; on input/change dispatches the
	// internal updateTheme + applies kit/preset side effects via applyKit /
	// applyPresetDefaults. Uses [data-gosx-studio-theme-bound] dataset key as
	// the per-control idempotency guard (distinct from the legacy key to allow
	// both paths to coexist additively).
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	want := "window." + IslandGlobals.BindTheme + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// DOM contract — the seven theme input selectors the legacy binds (see
	// studio-engines.js:1423). Drift in any of these silently breaks the
	// theme fan-out.
	for _, contract := range []string{
		// Top-level theme controls.
		`name="themeKit"`,
		`name="themeTemplate"`,
		`name="themePalette"`,
		`name="customTemplateName"`,
		"data-editor-custom-class",
		"data-editor-style-class",
		`name="themeImageRatio"`,
		"data-editor-color-token",
		// Idempotency guard key — distinct from legacy gosxStudioThemeBound
		// so both paths can coexist during the additive window.
		"gosxStudioThemeIslandBound",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindTheme must preserve %q contract", contract)
		}
	}
}

func TestBridgeShimPreservesLegacyPathWhenFlagOff(t *testing.T) {
	shim := string(BridgeShim())
	// The shim is additive — when the island global is missing, the legacy
	// JS path must still run. Sanity check: look for the legacy function
	// names so a future refactor that drops the fallback is caught here.
	// The ten legacy function names are the catalog of GoSXStudioStyleRuntime
	// methods as wired in assets/studio-engines.js:2341 — note four method
	// names differ from the public method name (bindWorkbench →
	// bindStyleWorkbench, applyTheme → updateTheme, showImpact →
	// showStyleImpact, restoreImpact → restoreStyleImpact, syncControlButtons
	// → syncStyleControlButtons, setControlValue → setStyleControlValue,
	// resetControlValue → resetStyleControlValue). bindTheme / bindCSS /
	// bindFonts have matching legacy names.
	for _, legacy := range []string{
		"bindTheme",
		"bindStyleWorkbench",
		"bindCSS",
		"bindFonts",
		"updateTheme",
		"syncStyleControlButtons",
		"showStyleImpact",
		"restoreStyleImpact",
		"setStyleControlValue",
		"resetStyleControlValue",
	} {
		if !strings.Contains(shim, legacy) {
			t.Fatalf("BridgeShim() must retain legacy fallback for %q:\n%s", legacy, shim)
		}
	}
}
