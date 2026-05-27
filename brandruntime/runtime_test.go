package brandruntime

import (
	"strings"
	"testing"
)

// The brandruntime package is the Go-side surface for the Phase 3 slice-3
// burn-down of GoSXStudioBrandRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip the island path (legacy bundle deleted 2026-05-27)
//     to the .gsx-authored islands in this package.
//  2. The JS shim that the legacy bundle appends so the
//     window.GoSXStudioBrandRuntime methods delegate to the islands when
//     the flag is on.
//  3. The island runtime JS that publishes window.__gosx_brand_runtime_*
//     globals the BridgeShim delegates to.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-3-brandruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the cross-frame signal channel updateHeaderLogo writes through (slice 6
// will swap the transitional legacy-delegation pattern below for a
// $preview.brand.headerLogo subscriber).

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if FeatureFlagKey != "brand-runtime-islands" {
		t.Fatalf("FeatureFlagKey must be %q per Phase 3 slice-3 contract; got %q", "brand-runtime-islands", FeatureFlagKey)
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
	// global it shims (window.GoSXStudioBrandRuntime), and the feature flag
	// so the legacy path stays reachable when the flag is off.
	for _, fragment := range []string{
		"window.GoSXStudioBrandRuntime",
		IslandGlobals.BindLogo,
		IslandGlobals.UpdateHeaderLogo,
		FeatureFlagKey,
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
}

func TestBridgeShimPreservesLegacyPathWhenFlagOff(t *testing.T) {
	shim := string(BridgeShim())
	// The shim is additive — when the island global is missing, the legacy
	// JS path must still run. Sanity check: look for the legacy function
	// names so a future refactor that drops the fallback is caught here.
	for _, legacy := range []string{
		"bindBrandLogo",
		"updateHeaderLogo",
	} {
		if !strings.Contains(shim, legacy) {
			t.Fatalf("BridgeShim() must retain legacy fallback for %q:\n%s", legacy, shim)
		}
	}
}

func TestIslandRuntimeJSPublishesIslandGlobals(t *testing.T) {
	body := string(IslandRuntimeJS())
	if body == "" {
		t.Fatal("IslandRuntimeJS() must return a non-empty JS snippet")
	}
	// The globals published here are the delegation targets BridgeShim consults.
	for _, fragment := range []string{
		"window." + IslandGlobals.BindLogo + " ",
		"window." + IslandGlobals.UpdateHeaderLogo + " ",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing global assignment %q", fragment)
		}
	}
	// The island runtime must preserve the legacy bindBrandLogo DOM
	// attribute contract — every input + handle + reset button it reads —
	// plus its idempotency-guard dataset key. Missing any of these silently
	// breaks the corresponding sub-behavior; the test fails before the
	// parity suite (which can't run until the cross-module bump lands) ever
	// gets a chance to catch the regression.
	for _, contract := range []string{
		// bindLogo sub-behavior — editor-side DOM contract
		"data-editor-brand-preview",
		"data-editor-brand-logo",
		"data-editor-brand-handle",
		"data-editor-logo-url",
		"data-editor-logo-alt",
		"data-editor-logo-width",
		"data-editor-logo-offset-x",
		"data-editor-logo-offset-y",
		"data-editor-logo-snap",
		"data-editor-logo-snap-size",
		"data-editor-logo-reset",
		"data-editor-logo-readout",
		// CSS custom properties the preview updates
		"--brand-logo-width",
		"--brand-logo-offset-x",
		"--brand-logo-offset-y",
		"--editor-logo-grid-size",
		// updateHeaderLogo sub-behavior — slice 6's transitional cleanup
		// (Section G.2) swapped the legacy
		// window.GoSXStudioPreviewRuntime.updateHeaderLogo delegation for
		// a direct $preview.brand.headerLogo signal write. The cross-
		// frame relay (per ADR 0009) delivers the signal to the
		// iframe's preview_subscriber, where the .brand DOM mutation
		// happens locally.
		"$preview.brand.headerLogo",
		// Idempotency guard — distinct from legacy gosxStudioBrandLogoBound
		// so both implementations can coexist on the same DOM during the
		// additive shipping window without double-binding.
		"gosxStudioBrandLogoIslandBound",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() brand contract missing %q", contract)
		}
	}
}

func TestBundleConcatenatesIslandRuntimeAndShim(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	// Island runtime must come before the BridgeShim — the shim consults
	// the global the island runtime publishes. Order matters.
	islandIdx := strings.Index(bundle, IslandGlobals.BindLogo+" ")
	shimIdx := strings.Index(bundle, "window.GoSXStudioBrandRuntime =")
	if islandIdx < 0 || shimIdx < 0 {
		t.Fatalf("Bundle() missing island runtime or shim:\n%s", bundle)
	}
	if islandIdx > shimIdx {
		t.Fatalf("Bundle() must place island runtime before shim (island=%d shim=%d)", islandIdx, shimIdx)
	}
}
