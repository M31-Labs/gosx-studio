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

func TestIslandRuntimeJSPublishesBindWorkbenchGlobal(t *testing.T) {
	// Method 2/10: bindWorkbench(root) — legacy bindStyleWorkbench at
	// studio-engines.js:1884. Resolves the [data-editor-workbench] form,
	// guards re-entry via the per-form idempotency dataset key, binds click /
	// pointerover / pointerout / focusin / focusout / input / change
	// listeners that dispatch setControlValue / resetControlValue /
	// showImpact / restoreImpact, registers bindStudioFrameLoad("data-editor-
	// style-impact-island-frame-bound") so the impact restores on iframe
	// reload, and runs syncControlButtons(form) + restoreImpact() on bind.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.BindWorkbench + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contracts the workbench listeners attach against.
		"data-editor-workbench",
		"data-studio-style-control",
		"data-studio-style-reset",
		"data-studio-style-value",
		// Per-form idempotency guard — distinct from legacy
		// gosxStudioStyleWorkbenchBound so both paths coexist additively.
		"gosxStudioStyleWorkbenchIslandBound",
		// Theme/template/custom-class controls the input/change handler
		// re-syncs on so the workbench buttons stay current with form state.
		`name="themeTemplate"`,
		`name="themeImageRatio"`,
		"data-editor-custom-class",
		"data-editor-style-class",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindWorkbench must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesBindCSSGlobal(t *testing.T) {
	// Method 3/10: bindCSS(root) — legacy bindCSS at studio-engines.js:1570.
	// Binds the [data-editor-css] textarea so input dispatches
	// PreviewRuntime.applyCSS(value) via a frameTask-coalesced handler.
	// Re-binds on preview-frame load via bindStudioFrameLoad. Calls applyCSS
	// once on bind to seed the preview with the current value.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.BindCSS + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contract — the custom CSS textarea selector.
		"data-editor-css",
		// Per-textarea idempotency guard — distinct from legacy
		// gosxStudioCSSBound so both paths coexist additively.
		"gosxStudioCSSIslandBound",
		// Cross-frame delegate the island calls into for the actual CSS
		// injection (slice 6 will replace with $preview.customCSS signal).
		"applyCSS",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindCSS must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesBindFontsGlobal(t *testing.T) {
	// Method 4/10: bindFonts(root) — legacy bindFonts at studio-engines.js:1602.
	// For each of the three slots (display / body / mono), reads
	// [data-editor-font-name="<slot>"] and [data-editor-font-url="<slot>"]
	// inputs and recomputes the combined @font-face + :root font-variable
	// CSS string. Dispatches PreviewRuntime.applyFonts(css) via
	// frameTask. Re-binds on preview-frame load.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.BindFonts + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contracts — font-slot inputs.
		"data-editor-font-name",
		"data-editor-font-url",
		// The three font slots the legacy iterates over (see line 1609).
		`"display"`,
		`"body"`,
		`"mono"`,
		// CSS family variable tokens written to :root (line 1618).
		"--font-display",
		"--font-body",
		"--font-mono",
		// Required for the @font-face rules emitted on populated slots.
		"@font-face",
		"font-display:swap",
		// Per-input idempotency guard — distinct from legacy
		// gosxStudioFontBound so both paths coexist additively.
		"gosxStudioFontIslandBound",
		// Cross-frame delegate the island calls into for the actual font CSS
		// injection (slice 6 will replace with $preview.fontCSS signal).
		"applyFonts",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() bindFonts must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesApplyThemeGlobal(t *testing.T) {
	// Method 5/10: applyTheme() — legacy updateTheme at studio-engines.js:1396.
	// Reads theme form state (kit / template / palette / image-ratio /
	// custom-template / style classes / color tokens), calls
	// PreviewRuntime.applyTheme(payload) for the cross-frame DOM mutation
	// (per ADR 0008 transitional pattern — see
	// ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md),
	// updates editor-side swatches / kit cards / template cards / custom-
	// template builder, then runs syncControlButtons(document).
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ApplyTheme + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Transitional iframe delegation per ADR 0008 — until slice 6 swaps
		// to $preview.theme.*, the cross-frame mutation rides on the legacy
		// PreviewRuntime contract. Catching drift here before the deletion-
		// window CI clock starts.
		"GoSXStudioPreviewRuntime",
		"applyTheme",
		// Payload fields the legacy populates and the preview consumes
		// (studio-engines.js:1404). Drift in any of these silently breaks
		// the iframe applyTheme contract.
		"kit:",
		"template:",
		"palette:",
		"imageRatio:",
		"customClasses:",
		"styleClasses:",
		"colors:",
		// Theme form inputs the helper reads to compose the payload.
		`name="themeImageRatio"`,
		"data-editor-color-token",
		// DOM contracts the editor-side post-mutation updates touch.
		"data-editor-theme-swatches",
		"data-editor-kit-card",
		"data-editor-template-card",
		"data-custom-template-builder",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() applyTheme must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSyncControlButtonsGlobal(t *testing.T) {
	// Method 6/10: syncControlButtons(root) — legacy syncStyleControlButtons
	// at studio-engines.js:1848. For every [data-studio-style-control] /
	// [data-studio-style-readout] / [data-studio-style-reset] in root:
	//   - control button: aria-pressed + .is-selected vs live form value.
	//   - readout: textContent (ownerLabel) + data-studio-style-inherited
	//     against the kit default.
	//   - reset button: disabled + textContent (Reset / Kit) + aria-label
	//     summarising the impact metadata.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SyncControlButtons + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// DOM contracts the helper queries / mutates.
		"data-studio-style-control",
		"data-studio-style-value",
		"data-studio-style-readout",
		"data-studio-style-reset",
		"data-studio-style-inherited",
		// Visual state contracts the helper toggles.
		"aria-pressed",
		"is-selected",
		// Reset-button labels — the exact strings users see.
		`"Reset"`,
		`"Kit"`,
		`"Inherited"`,
		// The styleImpactFor catalog the reset button's aria-label cites
		// for the per-control summary. studio-style-control-group is the
		// dataset key on the surrounding wrapper that gets the inherited
		// flag.
		"studio-style-control-group",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() syncControlButtons must preserve %q contract", contract)
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
