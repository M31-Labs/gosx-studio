package engine

import (
	"sort"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// RevisionPruner is an optional capability a RevisionStore may implement to
// physically remove a trimmed restore point. cms/store.RevisionStore
// (spec §8) intentionally gains no delete method of its own -- retention is
// additive, not a breaking change to the shared interface. A store that also
// implements RevisionPruner (cms/lifecycle/sqlstore.Store does, see
// sqlstore/store.go DeleteRevision) gets real enforcement; a store that
// doesn't makes enforceRetention a no-op (nothing is ever deleted), so
// trimming can never fail a publish/restore -- it is advisory-strength
// housekeeping layered on top of the mandatory restore-point contract, never
// a requirement for the transition itself.
type RevisionPruner interface {
	DeleteRevision(resourceKind, resourceID, revisionID string) error
}

// enforceRetention trims the restore points recorded for ref down to the
// engine's RetentionPolicy: it keeps the newest max(MaxRevisions,
// MinRestorePointKeep) restore points (MinRestorePointKeep is a floor under
// MaxRevisions, never a way to trim more aggressively than the cap) and
// never deletes any id in protect -- e.g. the revision an in-flight Restore
// is reading from (cmd.RevisionID), or the restore point this same
// transition just minted. Returns the ids that were actually removed.
func (e *Engine) enforceRetention(ref TargetRef, protect ...string) ([]string, error) {
	if e.Revisions == nil {
		return nil, nil
	}
	pruner, ok := e.Revisions.(RevisionPruner)
	if !ok {
		return nil, nil
	}

	policy := e.Retain.Normalize()
	keep := policy.MaxRevisions
	if policy.MinRestorePointKeep > keep {
		keep = policy.MinRestorePointKeep
	}

	all := e.Revisions.ListRevisions(lifecycle.RevisionFilter{
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
	})
	restorePoints := make([]lifecycle.Revision, 0, len(all))
	for _, revision := range all {
		if revision.Action == cmsstore.ActionRestorePoint {
			restorePoints = append(restorePoints, revision)
		}
	}
	if len(restorePoints) <= keep {
		return nil, nil
	}

	// ListRevisions already returns newest-first (both the in-memory
	// lifecycle.FilterRevisions helper and sqlstore's own `ORDER BY
	// created_at DESC` sort this way); re-sort defensively so any future
	// RevisionStore that doesn't guarantee ordering still trims the oldest
	// entries rather than the newest.
	sort.SliceStable(restorePoints, func(i, j int) bool {
		return restorePoints[i].Created.After(restorePoints[j].Created)
	})

	protected := make(map[string]bool, len(protect))
	for _, id := range protect {
		if id != "" {
			protected[id] = true
		}
	}

	var trimmed []string
	for _, revision := range restorePoints[keep:] {
		if protected[revision.ID] {
			continue
		}
		if err := pruner.DeleteRevision(revision.ResourceKind, revision.ResourceID, revision.ID); err != nil {
			return trimmed, err
		}
		trimmed = append(trimmed, revision.ID)
	}
	return trimmed, nil
}
