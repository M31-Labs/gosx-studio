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
