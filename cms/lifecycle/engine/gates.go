package engine

import "strings"

// Blocker is one host-reported publish blocker. Non-blocking blockers are
// advisory ("watch"); Blocking ones hard-stop Publish server-side.
type Blocker struct {
	Key         string
	Scope       string // "asset" | "binding" | "interaction"
	Summary     string
	Detail      string
	ResolveHref string // actionable link into the editor to fix it
	Blocking    bool
}

// Gate is the full set of blockers Engine.Readiness returns for a target.
type Gate struct {
	Blockers []Blocker
}

// Blocking reports whether any blocker in the gate denies publish.
func (g Gate) Blocking() bool {
	for _, blocker := range g.Blockers {
		if blocker.Blocking {
			return true
		}
	}
	return false
}

// ErrReadiness is returned by Publish when Gate.Blocking() is true. The
// panel's button state is advisory only; this is the server-side hard-stop
// that makes the "N/6 clear" count authoritative.
type ErrReadiness struct {
	Gate Gate
}

func (e ErrReadiness) Error() string {
	keys := make([]string, 0, len(e.Gate.Blockers))
	for _, blocker := range e.Gate.Blockers {
		if blocker.Blocking {
			keys = append(keys, blocker.Key)
		}
	}
	if len(keys) == 0 {
		return "publish blocked by readiness"
	}
	return "publish blocked by readiness: " + strings.Join(keys, ", ")
}
