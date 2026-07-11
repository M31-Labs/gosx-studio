package engine

import (
	"fmt"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// RetentionPolicy governs how many revisions each target keeps. It reuses
// lifecycle.TrimRevisions/DefaultRevisionLimit and adds a floor so trimming
// never drops the newest MinRestorePointKeep restore points out from under
// an operator who might need them.
type RetentionPolicy struct {
	// MaxRevisions defaults to lifecycle.DefaultRevisionLimit (500).
	MaxRevisions int
	// MinRestorePointKeep defaults to 20: restore points below this floor
	// are never trimmed.
	MinRestorePointKeep int
}

// Normalize fills in the defaults described above.
func (r RetentionPolicy) Normalize() RetentionPolicy {
	if r.MaxRevisions <= 0 {
		r.MaxRevisions = lifecycle.DefaultRevisionLimit
	}
	if r.MinRestorePointKeep <= 0 {
		r.MinRestorePointKeep = 20
	}
	return r
}

// restorePointInput builds the RevisionInput for an automatic pre-transition
// restore point: the full snapshot of pre-transition live state for ref,
// addressed by the ordinary Revision.ID machinery and distinguished by
// cmsstore.ActionRestorePoint.
func restorePointInput(ref TargetRef, snapshot any, operationID, summary string, now time.Time, newID func(string) string) cmsstore.RevisionInput {
	id := ""
	if newID != nil {
		id = strings.TrimSpace(newID("restore"))
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = fmt.Sprintf("Automatic restore point (operation %s).", operationID)
	}
	return cmsstore.RevisionInput{
		ID:           id,
		ResourceKind: ref.ResourceKind,
		ResourceID:   ref.ResourceID,
		Action:       cmsstore.ActionRestorePoint,
		Summary:      summary,
		Snapshot:     snapshot,
		Created:      now,
	}
}
