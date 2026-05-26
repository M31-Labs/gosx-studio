// preview_subscriber.gsx — GoSXStudioPreviewRuntime.* preview-side
// subscriber island.
//
// Phase 3 slice-6 architectural piece. This island is the mount-point
// marker that the storefront layout renders when loaded in the editor
// preview iframe (per ADR 0009's preview-mode bootstrap). The actual
// behavior layer is gosx-studio/previewruntime/subscriber_runtime.js
// (embedded via previewruntime.PreviewSubscriberScript()) — it reads
// $preview.* shared signals delivered by the cross-frame postMessage
// relay and applies the matching DOM mutations inside the iframe.
//
// See:
//   - ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-26-phase-3-slice-6-previewruntime.md
//   - ~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md
//   - ~/.hyphae/spaces/m31labs-gosx/decisions/0009-iframe-transport-postmessage-relay.md
//
// # Mount contract
//
// The storefront layout opts into preview-mode bootstrap via
// island.EnablePreviewBootstrap() (already shipped per ADR 0009; see
// muddy-noni-commerce/app/layout.server.go:26). When the storefront page
// is loaded with ?gosx-preview=1 (the editor's iframe URL augmenter adds
// this query parameter), the gosx tiny WASM build loads, the relay
// initializes, and any island registered here hydrates client-side.
//
// This island is rendered by the storefront layout so the storefront
// page (loaded inside the editor preview iframe) carries the subscriber
// markup. The companion subscriber_runtime.js script — emitted alongside
// the editor bundle — runs at the iframe document load time and registers
// the $preview.* observers via window.__gosx_observe_shared_signal.
//
// # Why a .gsx island vs a raw <script> tag
//
// The .gsx island form gives us:
//   - Server-side rendering of the mount-point markup so the iframe
//     document carries the marker for tests / debugging.
//   - Client-side hydration that fires only when the preview-mode
//     bootstrap is active (the island runtime ignores islands marked
//     hidden when their host page didn't request the bootstrap).
//   - The same authoring shape as slices 1–5's mount islands so the
//     parity matrix stays uniform.
//
// # Slice-6 acceptance check
//
// The slice ships when:
//   - This island renders on the storefront.
//   - subscriber_runtime.js boots inside the preview iframe.
//   - $preview.* signal writes from the editor frame propagate to the
//     iframe via the relay (ADR 0009).
//   - The subscriber applies the corresponding DOM mutation.
//
// Live cross-frame verification requires the gosx + gosx-studio tag
// bumps that include the relay (per ADR 0009 gotcha G1 in
// ~/.hyphae/spaces/m31labs-gosx/lessons/iframe-cross-frame-transport-lessons.md).
// Until those bumps land, parity tests assert the editor-side observable
// (signal writes via window.__gosx_set_shared_signal_json calls) only.

package previewruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// PreviewSubscriberProps lets the consumer pass an opaque selector or
// label identifying the preview document the subscriber is mounted on.
// Defaults to the iframe document when unset.
type PreviewSubscriberProps struct {
	RootSelector string
}

//gosx:island
func PreviewSubscriber(props PreviewSubscriberProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a
	// stable mount event in the compiled output. It also marks the
	// island as having been hydrated on the client; the parity tests
	// can assert this in candidate mode.
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-preview-subscriber-island="true"
		data-gosx-studio-preview-subscriber-root={root}
		data-gosx-studio-preview-subscriber-mounted={mounted.Get()}
		hidden
	></div>
}
