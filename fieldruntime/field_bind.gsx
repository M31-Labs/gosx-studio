// field_bind.gsx — GoSXStudioFieldRuntime.bind island.
//
// Phase 3 slice-1 island implementation of the JS function bindFieldRuntime in
// the legacy bundle (removed 2026-05-27). See the slice plan at
// ~/.hyphae/spaces/m31labs-gosx/plans/2026-05-25-phase-3-slice-1-fieldruntime.md
// and the parity matrix at
// ~/.hyphae/spaces/m31labs-gosx/specs/gosx-studio-runtime-parity-matrix.md.
//
// The island is a mount point — the consumer places it inside the Studio
// chrome and the runtime bundle's island_runtime.js publishes
// window.__gosx_field_runtime_island_bind that delegates to the mirroring
// and clipboard islands' implementations.
//
// Post 2026-05-27 the legacy JS bundle is gone; BridgeShim delegates to
// this island directly. See Phase 3 burn-down.

package fieldruntime

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

// FieldBindProps lets the consumer pass an opaque selector identifying the
// inspector root the field runtime should bind under. Defaults to document
// when unset, matching the legacy bindFieldRuntime(document) call site.
type FieldBindProps struct {
	RootSelector string
}

//gosx:island
func FieldBind(props FieldBindProps) gosx.Node {
	// The mounted signal exists so the island runtime can observe a stable
	// mount event in the compiled output. It also marks the island as
	// having been hydrated on the client, which the parity tests assert in
	// candidate mode (see parity_e2e/fieldruntime_test.ts).
	mounted := signal.New(true)
	root := props.RootSelector
	if root == "" {
		root = "document"
	}
	return <div
		data-gosx-studio-field-bind-island="true"
		data-gosx-studio-field-bind-root={root}
		data-gosx-studio-field-bind-mounted={mounted.Get()}
		hidden
	></div>
}
