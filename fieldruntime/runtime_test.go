package fieldruntime

import (
	"strings"
	"testing"
)

// The fieldruntime package is the Go-side surface for the Phase 3 slice-1
// island implementation of GoSXStudioFieldRuntime. It owns:
//   1. The feature-flag key (retained for host-probe API stability — the
//      island path always runs post-2026-05-27 since the legacy JS bundle
//      was deleted then).
//   2. The JS shim served alongside the island runtime so the
//      window.GoSXStudioFieldRuntime methods dispatch to the island
//      globals.
//   3. (Future) Helper Go APIs the consumer can call to mount the islands
//      into its editor route.
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-25-phase-3-slice-1-fieldruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the cross-frame signal channel the mirroring island writes through.

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if !strings.HasSuffix(FeatureFlagKey, "-runtime-islands") {
		t.Fatalf("FeatureFlagKey %q should end with -runtime-islands per Phase 3 convention", FeatureFlagKey)
	}
}

func TestBridgeShimDelegatesToIslandGlobals(t *testing.T) {
	shim := string(BridgeShim())
	if shim == "" {
		t.Fatal("BridgeShim() must return a non-empty JS snippet")
	}
	// The shim must reference the global the islands publish themselves at,
	// must reference window.GoSXStudioFieldRuntime (the public global it
	// shims), and must reference the feature flag so the legacy path stays
	// reachable when the flag is off.
	for _, fragment := range []string{
		"window.GoSXStudioFieldRuntime",
		"__gosx_field_runtime_island_bind",
		"__gosx_field_runtime_island_bindMirroring",
		"__gosx_field_runtime_island_bindClipboard",
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
		"bindFieldRuntime",
		"bindFieldMirroring",
		"bindFieldClipboard",
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
	// Globals published here are the delegation targets BridgeShim consults.
	for _, fragment := range []string{
		"window.__gosx_field_runtime_island_bind ",
		"window.__gosx_field_runtime_island_bindMirroring ",
		"window.__gosx_field_runtime_island_bindClipboard ",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("IslandRuntimeJS() missing global assignment %q", fragment)
		}
	}
	// The mirroring path must route through $preview.field.* shared signals
	// per ADR 0008.
	if !strings.Contains(body, "$preview.field.") {
		t.Fatalf("IslandRuntimeJS() mirroring must use $preview.field.* shared signals (see ADR 0008)")
	}
	// Clipboard must preserve the data-studio-copy-target contract from
	// the legacy bindFieldClipboard implementation.
	for _, contract := range []string{
		"data-studio-copy-target",
		"data-studio-copy-state",
		"data-studio-copy-label",
		"navigator.clipboard",
		"execCommand",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() clipboard contract missing %q", contract)
		}
	}
	// Mirroring must preserve the data-editor-source / data-editor-frame-*
	// attribute contract from legacy bindFieldMirroring.
	for _, contract := range []string{
		"data-editor-source",
		"data-editor-frame-attr-target",
		"data-editor-frame-attr-prefix",
		"data-editor-frame-attr-suffix",
	} {
		if !strings.Contains(body, contract) {
			t.Fatalf("IslandRuntimeJS() mirroring contract missing %q", contract)
		}
	}
}

func TestBundleConcatenatesIslandRuntimeAndShim(t *testing.T) {
	bundle := string(Bundle())
	if bundle == "" {
		t.Fatal("Bundle() must return a non-empty JS snippet")
	}
	// Island runtime must come before the BridgeShim — the shim consults
	// the globals the island runtime publishes. Order matters.
	islandIdx := strings.Index(bundle, "__gosx_field_runtime_island_bind ")
	shimIdx := strings.Index(bundle, "window.GoSXStudioFieldRuntime =")
	if islandIdx < 0 || shimIdx < 0 {
		t.Fatalf("Bundle() missing island runtime or shim:\n%s", bundle)
	}
	if islandIdx > shimIdx {
		t.Fatalf("Bundle() must place island runtime before shim (island=%d shim=%d)", islandIdx, shimIdx)
	}
}
