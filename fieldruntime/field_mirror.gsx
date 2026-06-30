// field_mirror.gsx — GoSXStudioFieldRuntime.bindMirroring island.
//
// Phase 3 slice-1 island implementation of the JS function bindFieldMirroring in
// the legacy bundle (removed 2026-05-27).
//
// The legacy implementation scanned the inspector for inputs with
// data-editor-source / data-editor-frame-attr-target, attached input event
// listeners, and on every keystroke updated the preview iframe's DOM
// directly through cross-frame element queries.
//
// Under Phase 3, per ADR 0008
// (~/.hyphae/spaces/m31labs-gosx/decisions/0008-iframe-preview-stays-via-shared-signal-portal.md),
// cross-frame state moves through $preview.* shared signals: the editor
// writes $preview.field.<sourceKey> on every keystroke and the preview
// iframe's subscriber island reads + applies. The preview iframe and the
// editor each have their own gosx WASM Bridge instance; shared signals
// cross the boundary via postMessage at <5ms per write (acceptable for
// 16ms / 60fps frame budgets).
//
// This island is the editor-side writer. The reader lives in slice 6
// (preview_subscriber.gsx); signal writes are the sole mirroring delivery
// mechanism now that the old PreviewRuntime branch has been removed.

package fieldruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// FieldMirrorProps mirrors FieldBindProps for symmetry; the runtime takes
// the inspector root selector and operates over the inputs found there.
type FieldMirrorProps struct {
	RootSelector string
}

// $preview.field is the shared-signal namespace per ADR 0008. The compile
// pipeline recognizes the leading "$" as the shared-signal marker (see
// gosx/client/bridge/bridge.go: isSharedSignal). The island runtime writes
// $preview.field.<sourceKey> on every input keystroke; the slice-6 preview
// subscriber reads and applies.

//gosx:island
func FieldMirror(props FieldMirrorProps) gosx.Node {
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-field-mirror-island="true"
		data-gosx-studio-field-mirror-root={root}
		data-gosx-studio-field-mirror-mounted={mounted.Get()}
		data-gosx-studio-field-mirror-signal-prefix="$preview.field."
		hidden
	></div>
}
