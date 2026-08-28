package engine

import (
	"testing"

	"m31labs.dev/gosx-studio/authoring"
)

func styleOp(id string, rev uint64, actor, property, breakpoint string, before, after authoring.OperationValue) authoring.OperationRecord {
	target := authoring.OperationTarget{ComponentKey: "home:hero", Property: property, Breakpoint: breakpoint}
	return authoring.OperationRecord{
		ID: id, ActorID: actor, ActorLabel: actor, Kind: authoring.OperationSetStyle,
		Target: target, DocumentRevision: rev, Before: before, After: after,
	}.Normalize()
}

func fieldOp(id string, rev uint64, actor, field string, before, after authoring.OperationValue) authoring.OperationRecord {
	target := authoring.OperationTarget{Field: field}
	return authoring.OperationRecord{
		ID: id, ActorID: actor, ActorLabel: actor, Kind: authoring.OperationSetField,
		Target: target, DocumentRevision: rev, Before: before, After: after,
	}.Normalize()
}

func TestOperationChangeSetClassifiesMixedDomains(t *testing.T) {
	ops := []authoring.OperationRecord{
		fieldOp("op-content", 1, "actor-a", "hero.headline", authoring.PresentValue("Old headline"), authoring.PresentValue("New headline")),
		styleOp("op-style", 2, "actor-a", "color", authoring.StyleBreakpointBase, authoring.OperationValue{}, authoring.PresentValue("#ff0000")),
		styleOp("op-layout", 3, "actor-a", "display", authoring.StyleBreakpointBase, authoring.PresentValue("block"), authoring.PresentValue("flex")),
		fieldOp("op-instances", 4, "actor-b", "instances.hero.title", authoring.PresentValue("Shared"), authoring.PresentValue("Overridden")),
		fieldOp("op-assets", 5, "actor-b", "assets.hero.image", authoring.PresentValue("img-1"), authoring.PresentValue("img-2")),
		fieldOp("op-interactions", 6, "actor-c", "interactions.hero.reveal", authoring.OperationValue{}, authoring.PresentValue("reveal-on-scroll")),
		fieldOp("op-flows", 7, "actor-c", "flows.contact.submit", authoring.PresentValue("flow.contact.v1"), authoring.PresentValue("flow.contact.v2")),
	}

	changes := OperationChangeSet(ops, 0)
	if len(changes) != len(ops) {
		t.Fatalf("expected every op to survive (no inverse pairs), got %d changes: %#v", len(changes), changes)
	}

	want := map[string]DraftChangeScope{
		"op-content":      ScopeContent,
		"op-style":        ScopeStyle,
		"op-layout":       ScopeLayout,
		"op-instances":    ScopeInstances,
		"op-assets":       ScopeAssets,
		"op-interactions": ScopeInteractions,
		"op-flows":        ScopeFlows,
	}
	for _, change := range changes {
		if len(change.OpIDs) != 1 {
			t.Fatalf("expected a single collapsed op id per surviving change, got %#v", change)
		}
		id := change.OpIDs[0]
		wantScope, ok := want[id]
		if !ok {
			t.Fatalf("unexpected change for unknown op id %q", id)
		}
		if change.Scope != wantScope {
			t.Fatalf("op %q: expected scope %q, got %q", id, wantScope, change.Scope)
		}
	}
}

func TestOperationChangeSetCollapsesExactInversePair(t *testing.T) {
	setRed := styleOp("op-1", 1, "actor-a", "color", authoring.StyleBreakpointBase, authoring.PresentValue("blue"), authoring.PresentValue("red"))
	setBlue := styleOp("op-2", 2, "actor-a", "color", authoring.StyleBreakpointBase, authoring.PresentValue("red"), authoring.PresentValue("blue"))
	unrelated := fieldOp("op-3", 3, "actor-a", "hero.headline", authoring.PresentValue("Old"), authoring.PresentValue("New"))

	changes := OperationChangeSet([]authoring.OperationRecord{setRed, setBlue, unrelated}, 0)
	if len(changes) != 1 {
		t.Fatalf("expected the exact-inverse pair to fully collapse, leaving only the unrelated change, got %d: %#v", len(changes), changes)
	}
	if changes[0].OpIDs[0] != "op-3" {
		t.Fatalf("expected the surviving change to be the unrelated op, got %#v", changes[0])
	}
}

func TestOperationChangeSetDoesNotCollapseNonInverseRoundTrip(t *testing.T) {
	// A -> B -> C -> A: nets to zero overall, but no ADJACENT pair in this
	// chain is each other's exact inverse (inverse of A->B is B->A, not
	// C->A), so the conservative choice keeps every op visible.
	aToB := styleOp("op-1", 1, "actor-a", "color", authoring.StyleBreakpointBase, authoring.PresentValue("A"), authoring.PresentValue("B"))
	bToC := styleOp("op-2", 2, "actor-a", "color", authoring.StyleBreakpointBase, authoring.PresentValue("B"), authoring.PresentValue("C"))
	cToA := styleOp("op-3", 3, "actor-a", "color", authoring.StyleBreakpointBase, authoring.PresentValue("C"), authoring.PresentValue("A"))

	changes := OperationChangeSet([]authoring.OperationRecord{aToB, bToC, cToA}, 0)
	if len(changes) != 3 {
		t.Fatalf("expected all three non-adjacent-inverse ops to survive (conservative collapse), got %d: %#v", len(changes), changes)
	}
}

func TestOperationChangeSetCollapsesAcrossInterleavedUnrelatedTarget(t *testing.T) {
	// The exact-inverse pair on "color" is interleaved with an unrelated
	// "display" edit; per-target adjacency still finds and collapses the
	// pair.
	setRed := styleOp("op-1", 1, "actor-a", "color", authoring.StyleBreakpointBase, authoring.PresentValue("blue"), authoring.PresentValue("red"))
	otherTarget := styleOp("op-2", 2, "actor-a", "display", authoring.StyleBreakpointBase, authoring.PresentValue("block"), authoring.PresentValue("flex"))
	setBlue := styleOp("op-3", 3, "actor-a", "color", authoring.StyleBreakpointBase, authoring.PresentValue("red"), authoring.PresentValue("blue"))

	changes := OperationChangeSet([]authoring.OperationRecord{setRed, otherTarget, setBlue}, 0)
	if len(changes) != 1 {
		t.Fatalf("expected only the unrelated display change to survive, got %d: %#v", len(changes), changes)
	}
	if changes[0].Scope != ScopeLayout || changes[0].OpIDs[0] != "op-2" {
		t.Fatalf("expected the surviving change to be the layout op, got %#v", changes[0])
	}
}

func TestOperationChangeSetPreservesActorAttribution(t *testing.T) {
	ops := []authoring.OperationRecord{
		fieldOp("op-1", 1, "alice", "hero.headline", authoring.PresentValue("Old"), authoring.PresentValue("Alice's edit")),
		styleOp("op-2", 2, "bob", "color", authoring.StyleBreakpointBase, authoring.OperationValue{}, authoring.PresentValue("#000")),
	}
	changes := OperationChangeSet(ops, 0)
	if len(changes) != 2 {
		t.Fatalf("expected both ops to survive, got %d", len(changes))
	}
	byID := map[string]DraftChange{}
	for _, change := range changes {
		byID[change.OpIDs[0]] = change
	}
	if byID["op-1"].ActorID != "alice" {
		t.Fatalf("expected op-1 to attribute alice, got %#v", byID["op-1"])
	}
	if byID["op-2"].ActorID != "bob" {
		t.Fatalf("expected op-2 to attribute bob, got %#v", byID["op-2"])
	}
}

func TestOperationChangeSetMarksUnsetValuesHonestly(t *testing.T) {
	// Before is genuinely unset (the style did not exist before this op) --
	// not the same as "the prior value was an empty string". After is
	// genuinely cleared by a reset (no After value), not "the new value is
	// an empty string" either.
	created := styleOp("op-created", 1, "actor-a", "color", authoring.StyleBreakpointBase, authoring.OperationValue{}, authoring.PresentValue("#fff"))
	cleared := authoring.OperationRecord{
		ID: "op-cleared", ActorID: "actor-a", Kind: authoring.OperationResetStyle,
		Target:           authoring.OperationTarget{ComponentKey: "home:footer", Property: "color", Breakpoint: authoring.StyleBreakpointBase},
		DocumentRevision: 2, Before: authoring.PresentValue("#fff"), After: authoring.OperationValue{},
	}.Normalize()

	changes := OperationChangeSet([]authoring.OperationRecord{created, cleared}, 0)
	if len(changes) != 2 {
		t.Fatalf("expected both ops to survive (different targets), got %d: %#v", len(changes), changes)
	}
	byID := map[string]DraftChange{}
	for _, change := range changes {
		byID[change.OpIDs[0]] = change
	}

	createdChange := byID["op-created"]
	if createdChange.BeforeSet {
		t.Fatalf("expected BeforeSet=false for an op with no prior value, got %#v", createdChange)
	}
	if createdChange.Before != "" {
		t.Fatalf("expected Before to be empty (not faked) when unset, got %q", createdChange.Before)
	}
	if !createdChange.AfterSet || createdChange.After != "#fff" {
		t.Fatalf("expected the new value to be representable, got %#v", createdChange)
	}

	clearedChange := byID["op-cleared"]
	if !clearedChange.BeforeSet || clearedChange.Before != "#fff" {
		t.Fatalf("expected the prior value to be representable, got %#v", clearedChange)
	}
	if clearedChange.AfterSet {
		t.Fatalf("expected AfterSet=false for a reset (cleared) value, got %#v", clearedChange)
	}
	if clearedChange.After != "" {
		t.Fatalf("expected After to be empty (not faked) when cleared, got %q", clearedChange.After)
	}
}

func TestOperationChangeSetExcludesAlreadyPublishedOps(t *testing.T) {
	ops := []authoring.OperationRecord{
		fieldOp("op-published", 1, "actor-a", "hero.headline", authoring.PresentValue("Old"), authoring.PresentValue("Published already")),
		fieldOp("op-pending", 2, "actor-a", "hero.subhead", authoring.PresentValue("Old sub"), authoring.PresentValue("Pending")),
	}
	changes := OperationChangeSet(ops, 1)
	if len(changes) != 1 {
		t.Fatalf("expected exactly the pending op to show, got %d: %#v", len(changes), changes)
	}
	if changes[0].OpIDs[0] != "op-pending" {
		t.Fatalf("expected op-pending to be the only surviving change, got %#v", changes[0])
	}
}

func TestOperationChangeSetSortsByDocumentRevisionRegardlessOfInputOrder(t *testing.T) {
	first := fieldOp("op-first", 1, "actor-a", "hero.headline", authoring.PresentValue("A"), authoring.PresentValue("B"))
	second := fieldOp("op-second", 2, "actor-a", "hero.headline", authoring.PresentValue("B"), authoring.PresentValue("C"))

	// Supplied out of order; the exact-inverse walk must still process by
	// DocumentRevision, not input order, or it would (wrongly) see "second"
	// before "first" and compare against nothing.
	changes := OperationChangeSet([]authoring.OperationRecord{second, first}, 0)
	if len(changes) != 2 {
		t.Fatalf("expected both non-inverse ops to survive regardless of input order, got %d: %#v", len(changes), changes)
	}
	if changes[0].OpIDs[0] != "op-first" || changes[1].OpIDs[0] != "op-second" {
		t.Fatalf("expected output ordered by DocumentRevision, got %#v", changes)
	}
}

func TestOperationChangeSetEmptyOpsYieldsEmptyChangeSet(t *testing.T) {
	if changes := OperationChangeSet(nil, 0); len(changes) != 0 {
		t.Fatalf("expected an empty ops slice to yield an empty change set, got %#v", changes)
	}
}
