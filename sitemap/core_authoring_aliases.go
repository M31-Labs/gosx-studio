package sitemap

import (
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

// Unexported convenience aliases for core/authoring helpers used throughout
// this package under their original (pre-restructure) unqualified names.
// These mirror the shim pattern in the root compat_core.go facade and the
// precedent set by authoring/core_aliases.go: rather than rewrite every call
// site to the fully qualified core.XXX/authoring.XXX form, the moved
// sitemap_view.go/sitemap_board.go bodies keep calling these lowercase
// names, which now resolve to the single canonical implementation in core
// or authoring.
var (
	firstNonEmpty               = core.FirstNonEmpty
	countLabel                  = core.CountLabel
	workspaceToken              = core.WorkspaceToken
	workspaceCoord              = core.WorkspaceCoord
	normalizeCompositionSteps   = core.NormalizeCompositionSteps
	normalizeCompositionIntents = authoring.NormalizeCompositionIntents
)
