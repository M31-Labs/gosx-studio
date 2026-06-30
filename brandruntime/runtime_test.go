package brandruntime

import (
	"strings"
	"testing"
)

// The brandruntime package is the Go-side surface for the Phase 3 slice-3
// burn-down of GoSXStudioBrandRuntime. It owns:
//  1. The stable feature-flag key consumers may still probe or emit as a
//     host attribute.
//  2. The JS shim that publishes window.GoSXStudioBrandRuntime and delegates
//     directly to the islands.
//  3. The island runtime JS that publishes window.__gosx_brand_runtime_*
//     globals the BridgeShim delegates to.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-3-brandruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the cross-frame signal channel updateHeaderLogo writes through.

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
	// The shim must reference each method-specific island global and the
	// public global it shims. It must dispatch directly to island globals
	// without a feature-flag-off path.
	for _, fragment := range []string{
		"window.GoSXStudioBrandRuntime",
		IslandGlobals.BindLogo,
		IslandGlobals.UpdateHeaderLogo,
		"return island.apply(null, arguments)",
		"return undefined",
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing %q:\n%s", fragment, shim)
		}
	}
	for _, stale := range []string{
		"flag" + "Enabled",
		"FLAG" + "_" + "ATTR",
		"feature flag" + " is on",
		"fallback" + " branch",
		"fallback" + " path",
		"legacy" + " fallback",
		FeatureFlagKey,
	} {
		if strings.Contains(shim, stale) {
			t.Fatalf("BridgeShim() contains stale gate/fallback fragment %q:\n%s", stale, shim)
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
		// updateHeaderLogo sub-behavior: direct
		// $preview.brand.headerLogo signal write. The cross-frame relay
		// (per ADR 0009) delivers the signal to the iframe's
		// preview_subscriber, where the .brand DOM mutation happens
		// locally.
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

func TestBridgeShimAutoMountsOnDocumentReady(t *testing.T) {
	// Pre-2026-05-27 the deleted studio-engines.js bundle auto-mounted every
	// runtime contract at DOMContentLoaded. When the bundle was deleted the
	// per-slice BridgeShim correctly re-published window.GoSXStudioBrandRuntime
	// but the auto-mount call was lost — nothing invoked .bindLogo(document)
	// on page load anymore, so brand-logo URL/width/offset input listeners
	// were never attached on initial load. The v0.4.1 fix landed this same
	// contract for fieldruntime; v0.5.0 restores it across the remaining six
	// islands. updateHeaderLogo is NOT auto-mounted (it's a host-driven
	// preview-side write, not a boot binding). Defer to DOMContentLoaded if
	// the document is still loading so we don't double-mount during SSR
	// hydration.
	shim := string(BridgeShim())
	for _, fragment := range []string{
		"DOMContentLoaded",
		"window.GoSXStudioBrandRuntime.bindLogo",
	} {
		if !strings.Contains(shim, fragment) {
			t.Fatalf("BridgeShim() missing auto-mount fragment %q:\n%s", fragment, shim)
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
