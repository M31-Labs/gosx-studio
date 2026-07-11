package engine

import (
	"sort"
	"strings"

	"m31labs.dev/gosx-studio/authoring"
)

// DraftChangeScope classifies one pending operation-log change by the
// authoring domain it belongs to, so the publish surface can group "what
// will publish" instead of showing an undifferentiated operation list.
type DraftChangeScope string

const (
	ScopeContent      DraftChangeScope = "content"
	ScopeStyle        DraftChangeScope = "style"
	ScopeLayout       DraftChangeScope = "layout"
	ScopeInstances    DraftChangeScope = "instances"
	ScopeAssets       DraftChangeScope = "assets"
	ScopeInteractions DraftChangeScope = "interactions"
	ScopeFlows        DraftChangeScope = "flows"
)

// DraftChange is one classified, attributable pending edit derived from the
// durable operation log (authoring.OperationRecord) since a target's last
// publish -- the exact-draft-diff surface spec §3 asks for, deviated from
// the spec's own cms/studio.DraftChange sketch to live in this package (see
// the S3 report "deviations": HARD BOUNDARIES forbid editing cms/studio/,
// matching the precedent S2/S4 set with engine.PublishChecksFromGate).
//
// Before/After mirror authoring.OperationRecord's Before/After
// authoring.OperationValue exactly: BeforeSet/AfterSet is false when the
// underlying OperationValue.Present is false (the field/style had no prior
// value, or the operation cleared it). Before/After are then "" -- callers
// MUST check BeforeSet/AfterSet before treating "" as a real empty-string
// value; this is the honest "unrepresentable, not faked" handling the task
// requires (see TestOperationChangeSetMarksUnsetValuesHonestly).
//
// OpIDs is a slice (not a single OpID, unlike the spec's pseudocode) because
// this slice's collapse semantics (below) already require a change to speak
// for more than one underlying operation id in the general case; today it is
// always a single-element slice (see "Collapse semantics").
type DraftChange struct {
	Scope            DraftChangeScope
	Target           authoring.OperationTarget
	Label            string
	Before           string
	BeforeSet        bool
	After            string
	AfterSet         bool
	ActorID          string
	ActorLabel       string
	OpIDs            []string
	DocumentRevision uint64
}

// OperationChangeSet classifies the durable operation-log records applied
// after sincePublishedRevision into a domain-grouped, actor-attributed
// change set: exactly the edits Engine.Publish is about to promote to live.
//
// # Collapse semantics (spec §3 open question #2 -- conservative choice)
//
// The spec asks whether an operation sequence that nets to no change should
// still show its intermediate ops or collapse to nothing. This slice picks
// the conservative answer stated in the task: collapse ONLY a literal,
// adjacent, exact-inverse PAIR of operations on the same target (same
// authoring.OperationTarget.Key) -- i.e. an edit immediately undone back to
// the exact prior value, byte for byte, on either side (Before/After AND
// their Present flags all match up). Both operations in such a pair vanish
// entirely from the change set; the target then reports no pending change,
// which is exactly "no-op sequence shows as no change."
//
// A longer round trip through a DIFFERENT intermediate value (A -> B -> C ->
// A) does NOT collapse, even though it also nets to zero, because no
// adjacent pair in that chain is each other's exact inverse (the inverse of
// A->B is B->A, not C->A). This is intentional: only the *structural* diff
// (lifecycle.RevisionDiff / DraftPreview.Diff, from Host.DraftDiff) is a net
// state comparison. This operation-log surface is an audit trail of what an
// operator actually did; it only ever elides a change that is exactly
// self-cancelling, never a coincidental net-zero across several edits. This
// keeps the two complementary sources (spec §3) genuinely complementary
// instead of the op-log degenerating into a second copy of the structural
// diff.
//
// Adjacency is scoped per target: an unrelated operation on a DIFFERENT
// target interleaved between two exact-inverse operations on THIS target
// does not prevent the cancellation (each target's own sub-sequence is
// walked independently), matching how a human operator reads the change set
// -- grouped by what changed, not by literal submission order across
// unrelated fields.
//
// # Empty-diff publish behavior
//
// An operation sequence that fully collapses (or an empty ops slice) yields
// an empty []DraftChange. This is presentation-only: Engine.Publish (S1/S2/
// S4) has no gate on an empty draft/change set today, and this slice does
// not add one -- publishing nothing is explicitly ALLOWED, matching Publish's
// existing normative sequence (spec §1.2), which never inspects the diff
// before minting a restore point and calling Host.ApplyPublish. Adding a new
// hard-stop here would be a Publish behavior change no directive in this
// slice requires, and would need to reconcile with the already-tested
// idempotent-replay/optimistic-head-check flow. DraftPreview.HasChanges
// (commands.go) exists precisely so a host/panel can render "nothing to
// publish" advisory copy without Engine itself refusing the call.
func OperationChangeSet(ops []authoring.OperationRecord, sincePublishedRevision uint64) []DraftChange {
	pending := make([]authoring.OperationRecord, 0, len(ops))
	for _, op := range ops {
		op = op.Normalize()
		if op.DocumentRevision <= sincePublishedRevision {
			continue
		}
		pending = append(pending, op)
	}
	sort.SliceStable(pending, func(i, j int) bool {
		if pending[i].DocumentRevision != pending[j].DocumentRevision {
			return pending[i].DocumentRevision < pending[j].DocumentRevision
		}
		return pending[i].Created.Before(pending[j].Created)
	})

	// Per-target stack: appending an op that is the exact inverse of the top
	// of its target's stack pops (cancels) both; anything else pushes. What
	// remains on each stack when the walk finishes is that target's
	// surviving (non-cancelled) operations, in chronological order.
	stacks := map[string][]authoring.OperationRecord{}
	for _, op := range pending {
		key := op.TargetKey
		if key == "" {
			key = op.Target.Key(op.Kind)
		}
		stack := stacks[key]
		if n := len(stack); n > 0 && isExactInverse(stack[n-1], op) {
			stacks[key] = stack[:n-1]
			continue
		}
		stacks[key] = append(stack, op)
	}

	survivors := map[string]bool{}
	for _, stack := range stacks {
		for _, op := range stack {
			survivors[op.ID] = true
		}
	}

	changes := make([]DraftChange, 0, len(pending))
	for _, op := range pending {
		if !survivors[op.ID] {
			continue
		}
		changes = append(changes, draftChangeFromOperation(op))
	}
	return changes
}

// isExactInverse reports whether next exactly reverses prev on the same
// target: next's Before is prev's After and next's After is prev's Before,
// including their Present flags (so a cleared-then-restored value and an
// unset-then-set-then-unset value both count, but a value that merely
// happens to end up textually equal without matching Present flags does
// not). This is the same relationship authoring.OperationRecord.InverseRequest
// constructs for an accepted Undo, generalized to any two operations that
// happen to be each other's exact inverse (not only a literal Undo/Redo of
// one another via UndoOf/RedoOf) -- a manual re-edit back to the exact prior
// value is just as legitimately "no visible net change" as clicking Undo.
func isExactInverse(prev, next authoring.OperationRecord) bool {
	return prev.After.Present == next.Before.Present &&
		prev.After.Value == next.Before.Value &&
		prev.Before.Present == next.After.Present &&
		prev.Before.Value == next.After.Value
}

func draftChangeFromOperation(op authoring.OperationRecord) DraftChange {
	return DraftChange{
		Scope:            classifyOperationScope(op.Target),
		Target:           op.Target,
		Label:            labelForTarget(op.Target),
		Before:           op.Before.Value,
		BeforeSet:        op.Before.Present,
		After:            op.After.Value,
		AfterSet:         op.After.Present,
		ActorID:          op.ActorID,
		ActorLabel:       op.ActorLabel,
		OpIDs:            []string{op.ID},
		DocumentRevision: op.DocumentRevision,
	}
}

// classifyOperationScope maps one operation's target onto a DraftChangeScope.
//
// Style vs. layout vs. content is a REAL classification against the durable
// operation log as it exists today: authoring.OperationTarget.IsStyle()
// (Property != "") already distinguishes a style-shaped target from a
// content field, and authoring.IsResponsiveLayoutProperty further splits the
// curated responsive-layout property subset (authoring/layout_values.go)
// out of general style. This branch is exercised by every SetStyle/
// ResetStyle op the collab durable log (cms/studio/collab) actually persists
// today, including an Undo/Redo of one (Kind is "undo"/"redo" but Target is
// inherited unchanged from the operation it reverses -- see
// cms/studio/collab/sqlstore/store.go's resolveHistory -- so IsStyle() still
// reports correctly for those).
//
// Instances/assets/interactions/flows classify via the dot-namespaced
// Field-prefix convention below (chosen to match the real dotted-path style
// of genuine Field values, not the colon-prefixed AuthoringChange.Key
// convention the mutation handlers use for their own ephemeral summaries,
// which is a different string with a different purpose). The durable-ops
// consolidation made these four domains first-class durable OperationKinds
// carrying exactly this convention -- see authoring/operations.go's Field*
// constants, and cms/studio/collab/sqlstore/lifecycle_classification_test.go
// which proves a real persisted record of each family classifies into the
// right scope here.
func classifyOperationScope(target authoring.OperationTarget) DraftChangeScope {
	target = target.Normalize()
	if target.IsStyle() {
		if authoring.IsResponsiveLayoutProperty(target.Property) {
			return ScopeLayout
		}
		return ScopeStyle
	}
	return scopeForField(target.Field)
}

// Forward-compatible, dot-namespaced Field-prefix convention. See
// classifyOperationScope's doc comment: nothing in the codebase emits these
// today, so this is honestly-documented scaffolding, not an active signal.
const (
	fieldNamespaceInstances    = "instances."
	fieldNamespaceAssets       = "assets."
	fieldNamespaceInteractions = "interactions."
	fieldNamespaceFlows        = "flows."
)

func scopeForField(field string) DraftChangeScope {
	switch {
	case strings.HasPrefix(field, fieldNamespaceInstances):
		return ScopeInstances
	case strings.HasPrefix(field, fieldNamespaceAssets):
		return ScopeAssets
	case strings.HasPrefix(field, fieldNamespaceInteractions):
		return ScopeInteractions
	case strings.HasPrefix(field, fieldNamespaceFlows):
		return ScopeFlows
	default:
		return ScopeContent
	}
}

// labelForTarget builds a short human-readable label for the panel's change
// list. It is best-effort presentation text, not an identifier -- callers
// needing an exact address use DraftChange.Target directly.
func labelForTarget(target authoring.OperationTarget) string {
	target = target.Normalize()
	if target.IsStyle() {
		label := target.Property
		if target.ComponentKey != "" {
			label = target.ComponentKey + " " + label
		}
		if target.Breakpoint != "" && target.Breakpoint != authoring.StyleBreakpointBase {
			label += " (" + target.Breakpoint + ")"
		}
		if target.State != "" && target.State != authoring.StyleStateDefault {
			label += " [" + target.State + "]"
		}
		return strings.TrimSpace(label)
	}
	if target.Field != "" {
		return target.Field
	}
	if target.ComponentKey != "" {
		return target.ComponentKey
	}
	return target.Route
}
