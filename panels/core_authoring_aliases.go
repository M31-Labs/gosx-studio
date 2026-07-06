package panels

import "m31labs.dev/gosx-studio/core"

// Unexported convenience alias for a core helper used in this package under
// its original (pre-restructure) unqualified name. This mirrors the shim
// pattern in the root compat_core.go facade and the precedent set by
// authoring/core_aliases.go and sitemap/core_authoring_aliases.go: rather
// than rewrite the call site to the fully qualified core.FirstNonEmpty
// form, the moved activity_panel.go body keeps calling this lowercase name,
// which now resolves to the single canonical implementation in core.
var firstNonEmpty = core.FirstNonEmpty
