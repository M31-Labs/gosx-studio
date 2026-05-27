// field_clipboard.gsx — GoSXStudioFieldRuntime.bindClipboard island.
//
// Phase 3 slice-1 island implementation of the JS function bindFieldClipboard in
// the legacy bundle (removed 2026-05-27).
//
// Behavior mirrored from the legacy implementation:
//   - Idempotent document-level click handler for [data-studio-copy-target].
//   - Reads the selector from data-studio-copy-target; copies the target's
//     .value || .textContent via navigator.clipboard.writeText with an
//     execCommand("copy") fallback.
//   - Toggles data-studio-copy-state ("ok" / "error" / "idle") and
//     button.textContent ("Copied" / "Copy failed" / original label) for
//     1600ms then restores.
//   - Preserves the original label in data-studio-copy-label.
//
// Clipboard interactions don't fit a signal-first reactive model — they are
// transient user affordances with browser-API side effects. The island is
// therefore a mount-point pattern; the runtime bundle's island_runtime.js
// publishes window.__gosx_field_runtime_island_bindClipboard, which the
// BridgeShim in runtime.go delegates to when the feature flag is on.

package fieldruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// FieldClipboardProps lets the consumer scope the click handler to a root
// (matches legacy bindFieldClipboard(root) signature). Defaults to document.
type FieldClipboardProps struct {
	RootSelector string
}

//gosx:island
func FieldClipboard(props FieldClipboardProps) gosx.Node {
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-field-clipboard-island="true"
		data-gosx-studio-field-clipboard-root={root}
		data-gosx-studio-field-clipboard-mounted={mounted.Get()}
		hidden
	></div>
}
