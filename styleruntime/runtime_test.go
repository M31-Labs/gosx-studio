package styleruntime

import (
	"strings"
	"testing"
)

// The styleruntime package is the Go-side surface for the Phase 3 slice-5
// burn-down of GoSXStudioStyleRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip the island path (legacy bundle deleted 2026-05-27)
//     to the .gsx-authored islands in this package.
//  2. The JS shim that the legacy bundle appends so the
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
	// Method 1/10: bindTheme(root) — legacy bindTheme at the legacy bundle:1421.
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
	// the legacy bundle:1423). Drift in any of these silently breaks the
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
	// the legacy bundle:1884. Resolves the [data-editor-workbench] form,
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
	// Method 3/10: bindCSS(root) — legacy bindCSS at the legacy bundle:1570.
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
	// Method 4/10: bindFonts(root) — legacy bindFonts at the legacy bundle:1602.
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
	// Method 5/10: applyTheme() — legacy updateTheme at the legacy bundle:1396.
	// Reads theme form state (kit / template / palette / image-ratio /
	// custom-template / style classes / color tokens) and publishes the
	// $preview.theme.* signal family. Slice 6 transitional cleanup
	// (Section G.4) swapped the previous
	// window.GoSXStudioPreviewRuntime.applyTheme delegation for direct
	// $preview.theme.* writes — the cross-frame relay (per ADR 0009)
	// delivers them to the iframe's preview_subscriber which reassembles
	// the payload and applies .site-shell class transitions + color CSS
	// vars. Editor-side swatches / kit cards / template cards / custom-
	// template builder updates remain editor-frame-only.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ApplyTheme + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Slice 6 transitional cleanup — the signal-family the writer
		// publishes. Drift in any of these silently breaks the cross-
		// frame applyTheme contract (subscriber observes these names
		// in gosx-studio/previewruntime/subscriber_runtime.js).
		"$preview.theme.kit",
		"$preview.theme.template",
		"$preview.theme.palette",
		"$preview.theme.imageRatio",
		"$preview.theme.customClasses",
		"$preview.theme.styleClasses",
		"$preview.theme.colors",
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
	// at the legacy bundle:1848. For every [data-studio-style-control] /
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

func TestIslandRuntimeJSPublishesShowImpactGlobal(t *testing.T) {
	// Method 7/10: showImpact(name, value, committed) — legacy showStyleImpact
	// at the legacy bundle:1785. Looks up impact metadata for the named
	// control, publishes $preview.style.impact.selector for the cross-
	// frame mutation (slice 6 transitional cleanup, Section G.5 — was
	// PreviewRuntime.applyStyleImpact), writes the editor-side impact-
	// readout labels (label / summary / count / scope / state), and
	// toggles the impact panel's is-live / has-change classes against
	// the committed flag. The cross-frame relay (per ADR 0009) delivers
	// the selector to the iframe's preview_subscriber, which marks
	// matching nodes via [data-studio-style-impact-node].
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ShowImpact + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// Slice 6 transitional cleanup — the signal carrying the selector
		// across the iframe boundary. Drift breaks the cross-frame contract.
		"$preview.style.impact.selector",
		// The five impact-readout selectors the helper writes (line 1791-1795).
		"data-studio-style-impact-label",
		"data-studio-style-impact-summary",
		"data-studio-style-impact-count",
		"data-studio-style-impact-scope",
		"data-studio-style-impact-state",
		// Impact-panel class state — is-live during hover/focus preview,
		// has-change after a commit.
		"data-studio-style-impact-panel",
		"is-live",
		"has-change",
		// Affected-count suffix and the no-active-recipe text the user sees.
		"affected",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() showImpact must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesRestoreImpactGlobal(t *testing.T) {
	// Method 8/10: restoreImpact() — legacy restoreStyleImpact at
	// the legacy bundle:1803. If a lastStyleImpact is recorded, replays
	// showImpact(name, value, true) to bring the panel back to its committed
	// state; otherwise clears the readouts to a "No active recipe" state
	// and publishes $preview.style.impact.selector := "" (slice 6
	// transitional cleanup, Section G.6 — was
	// PreviewRuntime.applyStyleImpact("")) to clear the iframe-side
	// markers. The cross-frame relay (per ADR 0009) delivers the empty-
	// selector write to the iframe's preview_subscriber.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.RestoreImpact + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// The clear-state readout strings users see when no impact is
		// active. Drift here changes the editor copy silently.
		`"No active recipe"`,
		`"Awaiting scoped change."`,
		`"0 affected"`,
		// The impact-panel classes restoreImpact removes when clearing.
		"is-live",
		"has-change",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() restoreImpact must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesSetControlValueGlobal(t *testing.T) {
	// Method 9/10: setControlValue(name, value) — legacy setStyleControlValue
	// at the legacy bundle:1821. When name targets a custom-* control (other
	// than customTemplateName), force-checks the [name="themeTemplate"]
	// [value="custom"] radio (with input + change events) so the custom
	// template builder activates. Then sets the named control's value (radio
	// match by [value] or namedControl assignment), dispatches input + change
	// events, runs syncControlButtons(document), and finally calls
	// showImpact(name, value, true) to commit the impact panel.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.SetControlValue + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	for _, contract := range []string{
		// The custom-template-radio coercion contract: when name starts with
		// "custom" but isn't "customTemplateName", set themeTemplate=custom.
		`"themeTemplate"`,
		`"custom"`,
		"customTemplateName",
		// Event contract — both input and change are dispatched on the
		// control after the value is set so any downstream listener fires.
		`new Event("input"`,
		`new Event("change"`,
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() setControlValue must preserve %q contract", contract)
		}
	}
}

func TestIslandRuntimeJSPublishesResetControlValueGlobal(t *testing.T) {
	// Method 10/10: resetControlValue(name) — legacy resetStyleControlValue
	// at the legacy bundle:1842. Looks up the kit default for name via
	// kitDefaultForControl and calls setControlValue(name, default). No-op
	// when no default is found.
	body := string(IslandRuntimeJS())
	want := "window." + IslandGlobals.ResetControlValue + " "
	if !strings.Contains(body, want) {
		t.Fatalf("IslandRuntimeJS() missing global assignment %q", want)
	}
	// resetControlValue is a 4-line delegation to setControlValue + the kit-
	// default lookup. The implementation must reference the helper / global
	// it composes — drift here would silently break the reset semantics.
	for _, contract := range []string{
		// The setControlValue island the reset composes.
		"setControlValueIsland",
		// The kit-default lookup helper.
		"kitDefaultForControl",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() resetControlValue must preserve %q contract", contract)
		}
	}
}

func TestBundleConcatenatesIslandRuntimeAndShim(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	// Island runtime must come before the BridgeShim — the shim consults
	// the globals the island runtime publishes. Order matters: if the shim
	// ran first it would close over an undefined global and every dispatch
	// would fall through to the legacy path, silently disabling the slice.
	islandIdx := strings.Index(bundle, IslandGlobals.BindTheme+" ")
	shimIdx := strings.Index(bundle, "window.GoSXStudioStyleRuntime =")
	if islandIdx < 0 || shimIdx < 0 {
		t.Fatalf("Bundle() missing island runtime or shim:\n%s", bundle)
	}
	if islandIdx > shimIdx {
		t.Fatalf("Bundle() must place island runtime before shim (island=%d shim=%d)", islandIdx, shimIdx)
	}
}

