package previewruntime

import (
	"strings"
	"testing"
)

// The previewruntime package is the Go-side surface for the Phase 3 slice-6
// architectural burn-down of GoSXStudioPreviewRuntime. It owns:
//  1. The feature-flag key consumers add to studio.ShellConfig.FeatureFlags
//     to flip from the legacy JS implementation in assets/preview-runtime.js
//     to the .gsx-authored subscriber + island writer in this package.
//  2. The JS shim that gosx-studio's runtime bundle appends so the
//     window.GoSXStudioPreviewRuntime methods publish to $preview.* shared
//     signals when the flag is on (and fall back to the legacy 1130-line
//     bridge when off).
//  3. The editor-side island_runtime.js that publishes
//     window.__gosx_preview_runtime_island_* globals the BridgeShim
//     delegates to.
//  4. The preview-side subscriber_runtime.js — emitted by
//     PreviewSubscriberScript() — that the storefront mounts when loaded
//     inside the editor preview iframe (per ADR 0009's cross-frame
//     postMessage relay).
//
// See ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-6-previewruntime.md
// for the slice plan; ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
// for the canonical iframe-stays-via-signals decision (status-corrected
// 2026-05-26); and ~/.hyphae/spaces/m31labs-gosx/decisions/0009-iframe-transport-postmessage-relay.md
// for the cross-frame transport that makes Path A deliverable.

func TestFeatureFlagKey(t *testing.T) {
	if FeatureFlagKey == "" {
		t.Fatal("FeatureFlagKey must be set")
	}
	// Naming convention: "<contract>-runtime-islands". Every Phase 3 slice
	// reuses this convention so the consumer's URL-param handling can flip
	// each contract's flag through a single helper.
	if FeatureFlagKey != "preview-runtime-islands" {
		t.Fatalf("FeatureFlagKey must be %q per Phase 3 slice-6 contract; got %q", "preview-runtime-islands", FeatureFlagKey)
	}
	if !strings.HasSuffix(FeatureFlagKey, "-runtime-islands") {
		t.Fatalf("FeatureFlagKey %q should end with -runtime-islands per Phase 3 convention", FeatureFlagKey)
	}
}
