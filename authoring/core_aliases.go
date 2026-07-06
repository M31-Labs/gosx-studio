package authoring

import "m31labs.dev/gosx-studio/core"

// Unexported convenience aliases for core helpers used throughout this
// package under their original (pre-restructure) unqualified names. These
// mirror the shim pattern in the root compat_core.go facade: rather than
// rewrite every call site to the fully qualified core.XXX form, the moved
// mutations.go/authoring.go bodies keep calling these lowercase names, which
// now resolve to the single canonical implementation in core.
var (
	firstNonEmpty                  = core.FirstNonEmpty
	countLabel                     = core.CountLabel
	workspaceToken                 = core.WorkspaceToken
	normalizeControlKind           = core.NormalizeControlKind
	normalizeCompositionIntentKind = core.NormalizeCompositionIntentKind
	normalizeCompositionSteps      = core.NormalizeCompositionSteps
)
