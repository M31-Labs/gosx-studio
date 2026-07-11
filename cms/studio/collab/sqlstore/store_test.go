package sqlstore

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/studio/collab"
	"m31labs.dev/gosx-studio/core"
)

func resource() collab.ResourceKey {
	return collab.ResourceKey{TenantID: "tenant", ProjectID: "project", Kind: "site", ID: "main"}
}
func principal(id string) collab.Principal {
	return collab.Principal{ActorID: id, DisplayName: id, Capabilities: map[collab.Capability]bool{collab.CapabilityAuthor: true}}
}
func designer(id string) collab.Principal {
	return collab.Principal{ActorID: id, DisplayName: id, Capabilities: map[collab.Capability]bool{collab.CapabilityDesign: true}}
}
func field(id, field, value, head string) authoring.OperationRequest {
	return authoring.OperationRequest{ID: id, Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Route: "/", PageID: "home", Field: field}, Value: value, ExpectedTargetHead: head}
}
func style(id, value, head string, reset bool) authoring.OperationRequest {
	kind := authoring.OperationSetStyle
	if reset {
		kind = authoring.OperationResetStyle
	}
	return authoring.OperationRequest{ID: id, Kind: kind, Target: authoring.OperationTarget{Route: "/", PageID: "home", ComponentKey: "home:hero", Property: "color", Breakpoint: "base", State: "default"}, Value: value, ExpectedTargetHead: head}
}
func apply(t *testing.T, s *Store, p collab.Principal, r authoring.OperationRequest) (collab.OperationAck, *collab.ProtocolError) {
	t.Helper()
	return s.Apply(context.Background(), collab.ApplyCommand{Resource: resource(), Principal: p, Request: r, Now: time.Date(2026, 7, 11, 1, 2, 3, 0, time.UTC)})
}

// --- Request builders for the new instance/interaction/flow durable
// operation families (mirroring field()/style() above). ---

func sharedField(id, definitionKey, controlKey, value, head string) authoring.OperationRequest {
	return authoring.OperationRequest{ID: id, Kind: authoring.OperationSetSharedField, Target: authoring.OperationTarget{Route: "/", ComponentKey: definitionKey, ControlKey: controlKey, Field: authoring.FieldInstancesSharedField}, Value: value, ExpectedTargetHead: head}
}
func overrideInstance(id, pageKey, componentKey, controlKey, value, head string) authoring.OperationRequest {
	return authoring.OperationRequest{ID: id, Kind: authoring.OperationOverrideInstance, Target: authoring.OperationTarget{Route: "/", PageID: pageKey, ComponentKey: componentKey, ControlKey: controlKey, Field: authoring.FieldInstancesOverride}, Value: value, ExpectedTargetHead: head}
}
func attachment(id string, detach bool, pageKey, componentKey, head string) authoring.OperationRequest {
	kind := authoring.OperationDetachInstance
	if !detach {
		kind = authoring.OperationRestoreInstance
	}
	return authoring.OperationRequest{ID: id, Kind: kind, Target: authoring.OperationTarget{Route: "/", PageID: pageKey, ComponentKey: componentKey, Field: authoring.FieldInstancesAttachment}, ExpectedTargetHead: head}
}
func interactionOp(id string, remove bool, pageKey, componentKey, interactionKey, value, head string) authoring.OperationRequest {
	kind := authoring.OperationSetInteraction
	if remove {
		kind = authoring.OperationRemoveInteraction
	}
	return authoring.OperationRequest{ID: id, Kind: kind, Target: authoring.OperationTarget{Route: "/", PageID: pageKey, ComponentKey: componentKey, ControlKey: interactionKey, Field: authoring.FieldInteractionsEntry}, Value: value, ExpectedTargetHead: head}
}
func flowFieldOp(id string, remove bool, flowKey, actionKey, fieldName, value, head string) authoring.OperationRequest {
	kind := authoring.OperationSetFlowField
	if remove {
		kind = authoring.OperationRemoveFlowField
	}
	return authoring.OperationRequest{ID: id, Kind: kind, Target: authoring.OperationTarget{Route: "/", PageID: flowKey, ComponentKey: actionKey, ControlKey: fieldName, Field: authoring.FieldFlowsField}, Value: value, ExpectedTargetHead: head}
}
func flowActionOp(id, flowKey, actionKey, value, head string) authoring.OperationRequest {
	return authoring.OperationRequest{ID: id, Kind: authoring.OperationSetFlowAction, Target: authoring.OperationTarget{Route: "/", PageID: flowKey, ComponentKey: actionKey, Field: authoring.FieldFlowsAction}, Value: value, ExpectedTargetHead: head}
}
func interactionValue(effect core.InteractionEffect) string {
	return authoring.EncodeInteractionSettings(core.Interaction{Kind: core.InteractionRevealOnScroll, Effect: effect, DurationMS: 250})
}

func TestDifferentFieldsAcceptAndSameFieldStaleIsRecoverable(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a, pe := apply(t, s, principal("a"), field("op-a", "hero.headline", "A", ""))
	if pe != nil || a.Sequence != 1 {
		t.Fatalf("first=%#v err=%v", a, pe)
	}
	b, pe := apply(t, s, principal("b"), field("op-b", "hero.eyebrow", "B", ""))
	if pe != nil || b.Sequence != 2 {
		t.Fatalf("different field=%#v err=%v", b, pe)
	}
	_, pe = apply(t, s, principal("b"), field("op-stale", "hero.headline", "lost", ""))
	if pe == nil || pe.Code != collab.ErrorStaleField || pe.CurrentHead != "op-a" || pe.CurrentValue == nil || pe.CurrentValue.Value != "A" {
		t.Fatalf("stale=%#v", pe)
	}
	attempts, err := s.Attempts(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 3 || attempts[2].Accepted || attempts[2].ErrorCode != collab.ErrorStaleField {
		t.Fatalf("attempts=%#v", attempts)
	}
}

func TestConcurrentDifferentFieldsAcceptAndSameFieldHasOneWinner(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	commands := []collab.ApplyCommand{
		{Resource: resource(), Principal: principal("a"), Request: field("a", "one", "A", "")},
		{Resource: resource(), Principal: principal("b"), Request: field("b", "two", "B", "")},
	}
	var wg sync.WaitGroup
	results := make([]*collab.ProtocolError, len(commands))
	for i := range commands {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, results[i] = s.Apply(ctx, commands[i]) }(i)
	}
	wg.Wait()
	if results[0] != nil || results[1] != nil {
		t.Fatalf("different fields errors=%#v", results)
	}
	same := []collab.ApplyCommand{
		{Resource: resource(), Principal: principal("a"), Request: field("same-a", "same", "A", "")},
		{Resource: resource(), Principal: principal("b"), Request: field("same-b", "same", "B", "")},
	}
	results = make([]*collab.ProtocolError, 2)
	for i := range same {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, results[i] = s.Apply(ctx, same[i]) }(i)
	}
	wg.Wait()
	accepted, stale := 0, 0
	for _, pe := range results {
		if pe == nil {
			accepted++
		} else if pe.Code == collab.ErrorStaleField {
			stale++
		}
	}
	if accepted != 1 || stale != 1 {
		t.Fatalf("same field accepted=%d stale=%d errors=%#v", accepted, stale, results)
	}
}

func TestIdempotentReplayAndAlteredOrOtherActorReuse(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := principal("a")
	request := field("same", "title", "one", "")
	first, pe := apply(t, s, p, request)
	if pe != nil {
		t.Fatal(pe)
	}
	replay, pe := apply(t, s, p, request)
	if pe != nil || !replay.Idempotent || replay.Sequence != first.Sequence {
		t.Fatalf("replay=%#v err=%v", replay, pe)
	}
	changed := request
	changed.Value = "two"
	_, pe = apply(t, s, p, changed)
	if pe == nil || pe.Code != collab.ErrorOperationIDReuse {
		t.Fatalf("altered=%#v", pe)
	}
	_, pe = apply(t, s, principal("b"), request)
	if pe == nil || pe.Code != collab.ErrorOperationIDReuse {
		t.Fatalf("other actor=%#v", pe)
	}
	tail, err := s.Tail(context.Background(), resource(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 1 {
		t.Fatalf("tail=%#v", tail)
	}
	attempts, err := s.Attempts(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 3 || attempts[1].ErrorCode != collab.ErrorOperationIDReuse || attempts[2].ErrorCode != collab.ErrorOperationIDReuse {
		t.Fatalf("attempts=%#v", attempts)
	}
	other := resource()
	other.ID = "other"
	ack, pe := s.Apply(context.Background(), collab.ApplyCommand{Resource: other, Principal: p, Request: request})
	if pe != nil || ack.Sequence != 1 {
		t.Fatalf("same id on other resource=%#v err=%v", ack, pe)
	}
}

func TestUndoCannotOverwriteInterveningSameFieldOperation(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p := principal("a")
	first, pe := apply(t, s, p, field("first", "title", "one", ""))
	if pe != nil {
		t.Fatal(pe)
	}
	_, pe = apply(t, s, p, field("second", "title", "two", "first"))
	if pe != nil {
		t.Fatal(pe)
	}
	undo := authoring.OperationRequest{ID: "unsafe-undo", Kind: authoring.OperationUndo, Target: first.Record.Target, HistoryOperationID: "first", ExpectedTargetHead: "second"}
	_, pe = apply(t, s, p, undo)
	if pe == nil || pe.Code != collab.ErrorStaleField || pe.CurrentHead != "second" {
		t.Fatalf("unsafe undo=%#v", pe)
	}
}

func TestActorScopedUndoAndReopenPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "studio.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	first, pe := apply(t, s, principal("a"), field("first", "title", "one", ""))
	if pe != nil {
		t.Fatal(pe)
	}
	undo := authoring.OperationRequest{ID: "undo-b", Kind: authoring.OperationUndo, Target: first.Record.Target, HistoryOperationID: "first", ExpectedTargetHead: "first"}
	_, pe = apply(t, s, principal("b"), undo)
	if pe == nil || pe.Code != collab.ErrorHistoryActor {
		t.Fatalf("foreign undo=%#v", pe)
	}
	undo.ID = "undo-a"
	ack, pe := apply(t, s, principal("a"), undo)
	if pe != nil || ack.Record.UndoOf != "first" || ack.Record.After.Present {
		t.Fatalf("undo=%#v err=%#v", ack, pe)
	}
	hash1, err := s.StateHash(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hash2, err := s.StateHash(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 {
		t.Fatalf("hash changed %s != %s", hash1, hash2)
	}
	tail, err := s.Tail(context.Background(), resource(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 2 || tail[1].Record.UndoOf != "first" {
		t.Fatalf("tail=%#v", tail)
	}
	outbox, err := s.PendingOutbox(context.Background(), resource(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outbox) != 2 || outbox[1].Sequence != 2 {
		t.Fatalf("outbox=%#v", outbox)
	}
}

func TestAcceptedOperationRollsBackWhenOutboxFails(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, err = s.DB().Exec(`CREATE TRIGGER reject_outbox BEFORE INSERT ON studio_projection_outbox BEGIN SELECT RAISE(ABORT,'forced outbox failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	_, pe := apply(t, s, principal("a"), field("op", "title", "one", ""))
	if pe == nil || pe.Code != collab.ErrorStoreUnavailable {
		t.Fatalf("error=%#v", pe)
	}
	if _, err = s.DB().Exec(`DROP TRIGGER reject_outbox`); err != nil {
		t.Fatal(err)
	}
	ack, pe := apply(t, s, principal("a"), field("op", "title", "one", ""))
	if pe != nil || ack.Sequence != 1 {
		t.Fatalf("post rollback=%#v err=%v", ack, pe)
	}
	attempts, err := s.Attempts(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 || !attempts[0].Accepted {
		t.Fatalf("attempts=%#v", attempts)
	}
}

func TestViewerRefusalIsDurableAndWritesNoOutbox(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	viewer := collab.Principal{ActorID: "viewer", Capabilities: map[collab.Capability]bool{collab.CapabilityView: true}}
	_, pe := apply(t, s, viewer, field("blocked", "title", "no", ""))
	if pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("error=%#v", pe)
	}
	attempts, err := s.Attempts(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 || attempts[0].Accepted {
		t.Fatalf("attempts=%#v", attempts)
	}
	outbox, err := s.PendingOutbox(context.Background(), resource(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outbox) != 0 {
		t.Fatalf("outbox=%#v", outbox)
	}
}

func TestSeedHeadsPreserveExistingDraftAndCreateNoHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "studio.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	heads := []collab.SeedHead{
		{Target: field("seed", "title", "", "").Target, Value: authoring.PresentValue("Existing title")},
		{Target: field("seed-absent", "optional", "", "").Target, Value: authoring.OperationValue{}},
		{Target: authoring.OperationTarget{Route: "/", PageID: "home", ComponentKey: "home:hero", Property: "color", Breakpoint: "mobile", State: "hover"}, Value: authoring.PresentValue("")},
	}
	if err := s.SeedHeads(context.Background(), resource(), heads); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedHeads(context.Background(), resource(), heads); err != nil {
		t.Fatalf("idempotent seed: %v", err)
	}
	attempts, err := s.Attempts(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 0 {
		t.Fatalf("seed attempts=%#v", attempts)
	}
	outbox, err := s.PendingOutbox(context.Background(), resource(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outbox) != 0 {
		t.Fatalf("seed outbox=%#v", outbox)
	}
	tail, err := s.Tail(context.Background(), resource(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 0 {
		t.Fatalf("seed tail=%#v", tail)
	}
	hash1, err := s.StateHash(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hash2, err := s.StateHash(context.Background(), resource())
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 {
		t.Fatalf("seed hash changed across reopen %s != %s", hash1, hash2)
	}
	styleRequest := authoring.OperationRequest{ID: "style-change", Kind: authoring.OperationSetStyle, Target: heads[2].Target, Value: "red"}
	styleAck, pe := apply(t, s, designer("d"), styleRequest)
	if pe != nil || !styleAck.Record.Before.Present || styleAck.Record.Before.Value != "" || styleAck.Record.Target.Breakpoint != "mobile" || styleAck.Record.Target.State != "hover" {
		t.Fatalf("seeded style=%#v err=%v", styleAck, pe)
	}
	undoStyle := authoring.OperationRequest{ID: "style-undo", Kind: authoring.OperationUndo, Target: styleAck.Record.Target, HistoryOperationID: "style-change", ExpectedTargetHead: "style-change"}
	styleRestored, pe := apply(t, s, designer("d"), undoStyle)
	if pe != nil || !styleRestored.Record.After.Present || styleRestored.Record.After.Value != "" {
		t.Fatalf("style undo=%#v err=%v", styleRestored, pe)
	}
	ack, pe := apply(t, s, principal("a"), field("change", "title", "Changed", ""))
	if pe != nil || !ack.Record.Before.Present || ack.Record.Before.Value != "Existing title" {
		t.Fatalf("first op=%#v err=%v", ack, pe)
	}
	undo := authoring.OperationRequest{ID: "undo", Kind: authoring.OperationUndo, Target: ack.Record.Target, HistoryOperationID: "change", ExpectedTargetHead: "change"}
	restored, pe := apply(t, s, principal("a"), undo)
	if pe != nil || !restored.Record.After.Present || restored.Record.After.Value != "Existing title" {
		t.Fatalf("undo=%#v err=%v", restored, pe)
	}
}

func TestSeedHeadsRejectConflictingReseedAndConcurrentSeedIsAtomic(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	target := field("seed", "title", "", "").Target
	if err := s.SeedHeads(context.Background(), resource(), []collab.SeedHead{{Target: target, Value: authoring.PresentValue("one")}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedHeads(context.Background(), resource(), []collab.SeedHead{{Target: target, Value: authoring.PresentValue("two")}}); err == nil {
		t.Fatal("conflicting reseed must fail")
	}
	other := field("seed-other", "other", "", "").Target
	var wg sync.WaitGroup
	errorsSeen := make([]error, 2)
	for i, value := range []string{"alpha", "beta"} {
		wg.Add(1)
		go func(i int, value string) {
			defer wg.Done()
			errorsSeen[i] = s.SeedHeads(context.Background(), resource(), []collab.SeedHead{{Target: other, Value: authoring.PresentValue(value)}})
		}(i, value)
	}
	wg.Wait()
	successes, failures := 0, 0
	for _, err := range errorsSeen {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("concurrent seeds=%#v", errorsSeen)
	}
}

func TestCapabilitiesAreOperationSpecificIncludingUndo(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, pe := apply(t, s, principal("author"), style("author-style", "red", "", false)); pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("author style=%#v", pe)
	}
	if _, pe := apply(t, s, designer("designer"), field("designer-field", "title", "x", "")); pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("designer field=%#v", pe)
	}
	content, pe := apply(t, s, principal("switch"), field("content", "title", "x", ""))
	if pe != nil {
		t.Fatal(pe)
	}
	undoContent := authoring.OperationRequest{ID: "undo-content", Kind: authoring.OperationUndo, Target: content.Record.Target, HistoryOperationID: "content", ExpectedTargetHead: "content"}
	if _, pe = apply(t, s, designer("switch"), undoContent); pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("designer undo content=%#v", pe)
	}
	design, pe := apply(t, s, designer("switch-design"), style("design", "red", "", false))
	if pe != nil {
		t.Fatal(pe)
	}
	undoDesign := authoring.OperationRequest{ID: "undo-design", Kind: authoring.OperationUndo, Target: design.Record.Target, HistoryOperationID: "design", ExpectedTargetHead: "design"}
	if _, pe = apply(t, s, principal("switch-design"), undoDesign); pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("author undo design=%#v", pe)
	}
}

func TestUndoRedoContentStyleAndResetPreservePresence(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	content, pe := apply(t, s, principal("a"), field("content", "title", "one", ""))
	if pe != nil {
		t.Fatal(pe)
	}
	undoContent := authoring.OperationRequest{ID: "content-undo", Kind: authoring.OperationUndo, Target: content.Record.Target, HistoryOperationID: "content", ExpectedTargetHead: "content"}
	undone, pe := apply(t, s, principal("a"), undoContent)
	if pe != nil || undone.Record.After.Present {
		t.Fatalf("content undo=%#v err=%v", undone, pe)
	}
	redoContent := authoring.OperationRequest{ID: "content-redo", Kind: authoring.OperationRedo, Target: content.Record.Target, HistoryOperationID: "content-undo", ExpectedTargetHead: "content-undo"}
	redone, pe := apply(t, s, principal("a"), redoContent)
	if pe != nil || !redone.Record.After.Present || redone.Record.After.Value != "one" || redone.Record.RedoOf != "content" {
		t.Fatalf("content redo=%#v err=%v", redone, pe)
	}

	design, pe := apply(t, s, designer("d"), style("style", "red", "", false))
	if pe != nil {
		t.Fatal(pe)
	}
	undoStyle := authoring.OperationRequest{ID: "style-undo", Kind: authoring.OperationUndo, Target: design.Record.Target, HistoryOperationID: "style", ExpectedTargetHead: "style"}
	styleUndone, pe := apply(t, s, designer("d"), undoStyle)
	if pe != nil || styleUndone.Record.After.Present {
		t.Fatalf("style undo=%#v err=%v", styleUndone, pe)
	}
	redoStyle := authoring.OperationRequest{ID: "style-redo", Kind: authoring.OperationRedo, Target: design.Record.Target, HistoryOperationID: "style-undo", ExpectedTargetHead: "style-undo"}
	styleRedone, pe := apply(t, s, designer("d"), redoStyle)
	if pe != nil || styleRedone.Record.After.Value != "red" || styleRedone.Record.RedoOf != "style" {
		t.Fatalf("style redo=%#v err=%v", styleRedone, pe)
	}

	blue, pe := apply(t, s, designer("d"), style("blue", "blue", "style-redo", false))
	if pe != nil {
		t.Fatal(pe)
	}
	reset, pe := apply(t, s, designer("d"), style("reset", "", "blue", true))
	if pe != nil || reset.Record.After.Present || reset.Record.Before.Value != "blue" {
		t.Fatalf("reset=%#v err=%v", reset, pe)
	}
	undoReset := authoring.OperationRequest{ID: "reset-undo", Kind: authoring.OperationUndo, Target: blue.Record.Target, HistoryOperationID: "reset", ExpectedTargetHead: "reset"}
	restored, pe := apply(t, s, designer("d"), undoReset)
	if pe != nil || restored.Record.After.Value != "blue" {
		t.Fatalf("reset undo=%#v err=%v", restored, pe)
	}
	redoReset := authoring.OperationRequest{ID: "reset-redo", Kind: authoring.OperationRedo, Target: blue.Record.Target, HistoryOperationID: "reset-undo", ExpectedTargetHead: "reset-undo"}
	resetAgain, pe := apply(t, s, designer("d"), redoReset)
	if pe != nil || resetAgain.Record.After.Present || resetAgain.Record.RedoOf != "reset" {
		t.Fatalf("reset redo=%#v err=%v", resetAgain, pe)
	}
}

// --- New durable operation families: instances (shared field / override /
// detach / restore), interactions (set/remove), and flows (field/action). ---

func TestSingletonInstanceAndFlowFamiliesRoundTripUndoRedo(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	shared, pe := apply(t, s, principal("a"), sharedField("shared-1", "hero-card", "title", "Hello", ""))
	if pe != nil || shared.Record.Before.Present || !shared.Record.After.Present || shared.Record.After.Value != "Hello" {
		t.Fatalf("shared field apply=%#v err=%v", shared, pe)
	}
	undoShared := authoring.OperationRequest{ID: "shared-undo", Kind: authoring.OperationUndo, Target: shared.Record.Target, HistoryOperationID: "shared-1", ExpectedTargetHead: "shared-1"}
	undoneShared, pe := apply(t, s, principal("a"), undoShared)
	if pe != nil || undoneShared.Record.After.Value != "" || undoneShared.Record.UndoOf != "shared-1" {
		t.Fatalf("shared field undo=%#v err=%v", undoneShared, pe)
	}
	redoShared := authoring.OperationRequest{ID: "shared-redo", Kind: authoring.OperationRedo, Target: shared.Record.Target, HistoryOperationID: "shared-undo", ExpectedTargetHead: "shared-undo"}
	redoneShared, pe := apply(t, s, principal("a"), redoShared)
	if pe != nil || redoneShared.Record.After.Value != "Hello" || redoneShared.Record.RedoOf != "shared-1" {
		t.Fatalf("shared field redo=%#v err=%v", redoneShared, pe)
	}

	override, pe := apply(t, s, principal("a"), overrideInstance("override-1", "home", "home:hero", "title", "Overridden", ""))
	if pe != nil || override.Record.Before.Present || !override.Record.After.Present || override.Record.After.Value != "Overridden" {
		t.Fatalf("override apply=%#v err=%v", override, pe)
	}
	undoOverride := authoring.OperationRequest{ID: "override-undo", Kind: authoring.OperationUndo, Target: override.Record.Target, HistoryOperationID: "override-1", ExpectedTargetHead: "override-1"}
	undoneOverride, pe := apply(t, s, principal("a"), undoOverride)
	if pe != nil || undoneOverride.Record.After.Value != "" || undoneOverride.Record.UndoOf != "override-1" {
		t.Fatalf("override undo=%#v err=%v", undoneOverride, pe)
	}

	action, pe := apply(t, s, principal("a"), flowActionOp("action-1", "contact", "submit", authoring.EncodeFlowActionSettings("Submit", ""), ""))
	if pe != nil || !action.Record.After.Present {
		t.Fatalf("flow action apply=%#v err=%v", action, pe)
	}
	next, pe := apply(t, s, principal("a"), flowActionOp("action-2", "contact", "submit", authoring.EncodeFlowActionSettings("Submit", "flow.contact.submit"), "action-1"))
	if pe != nil || next.Record.Before.Value != authoring.EncodeFlowActionSettings("Submit", "") {
		t.Fatalf("flow action second apply=%#v err=%v", next, pe)
	}
	undoAction := authoring.OperationRequest{ID: "action-undo", Kind: authoring.OperationUndo, Target: action.Record.Target, HistoryOperationID: "action-2", ExpectedTargetHead: "action-2"}
	undoneAction, pe := apply(t, s, principal("a"), undoAction)
	if pe != nil || undoneAction.Record.After.Value != authoring.EncodeFlowActionSettings("Submit", "") || undoneAction.Record.UndoOf != "action-2" {
		t.Fatalf("flow action undo=%#v err=%v", undoneAction, pe)
	}
}

func TestPairedInstanceInteractionFlowFamiliesRoundTripUndoRedo(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	detach, pe := apply(t, s, principal("a"), attachment("detach-1", true, "home", "home:hero", ""))
	if pe != nil || detach.Record.Before.Present || !detach.Record.After.Present {
		t.Fatalf("detach apply=%#v err=%v", detach, pe)
	}
	restore, pe := apply(t, s, principal("a"), attachment("restore-1", false, "home", "home:hero", "detach-1"))
	if pe != nil || !restore.Record.Before.Present || restore.Record.After.Present {
		t.Fatalf("restore apply=%#v err=%v", restore, pe)
	}
	undoRestore := authoring.OperationRequest{ID: "restore-undo", Kind: authoring.OperationUndo, Target: restore.Record.Target, HistoryOperationID: "restore-1", ExpectedTargetHead: "restore-1"}
	undoneRestore, pe := apply(t, s, principal("a"), undoRestore)
	if pe != nil || !undoneRestore.Record.After.Present || undoneRestore.Record.UndoOf != "restore-1" {
		t.Fatalf("restore undo (back to detached)=%#v err=%v", undoneRestore, pe)
	}
	redoRestore := authoring.OperationRequest{ID: "restore-redo", Kind: authoring.OperationRedo, Target: restore.Record.Target, HistoryOperationID: "restore-undo", ExpectedTargetHead: "restore-undo"}
	redoneRestore, pe := apply(t, s, principal("a"), redoRestore)
	if pe != nil || redoneRestore.Record.After.Present || redoneRestore.Record.RedoOf != "restore-1" {
		t.Fatalf("restore redo=%#v err=%v", redoneRestore, pe)
	}

	set, pe := apply(t, s, designer("d"), interactionOp("interaction-1", false, "home", "home:hero", "home:hero:reveal", interactionValue(core.InteractionEffectFade), ""))
	if pe != nil || set.Record.Before.Present || !set.Record.After.Present {
		t.Fatalf("set interaction apply=%#v err=%v", set, pe)
	}
	removed, pe := apply(t, s, designer("d"), interactionOp("interaction-remove-1", true, "home", "home:hero", "home:hero:reveal", "", "interaction-1"))
	if pe != nil || removed.Record.Before.Value != interactionValue(core.InteractionEffectFade) || removed.Record.After.Present {
		t.Fatalf("remove interaction apply=%#v err=%v", removed, pe)
	}
	undoRemove := authoring.OperationRequest{ID: "interaction-undo", Kind: authoring.OperationUndo, Target: removed.Record.Target, HistoryOperationID: "interaction-remove-1", ExpectedTargetHead: "interaction-remove-1"}
	undoneRemove, pe := apply(t, s, designer("d"), undoRemove)
	if pe != nil || undoneRemove.Record.After.Value != interactionValue(core.InteractionEffectFade) || undoneRemove.Record.UndoOf != "interaction-remove-1" {
		t.Fatalf("remove interaction undo=%#v err=%v", undoneRemove, pe)
	}

	fieldSet, pe := apply(t, s, principal("a"), flowFieldOp("field-1", false, "contact", "submit", "email", authoring.EncodeFlowFieldSettings(core.FlowField{Label: "Email", Kind: core.ControlText, Required: true}), ""))
	if pe != nil || fieldSet.Record.Before.Present || !fieldSet.Record.After.Present {
		t.Fatalf("set flow field apply=%#v err=%v", fieldSet, pe)
	}
	fieldRemoved, pe := apply(t, s, principal("a"), flowFieldOp("field-remove-1", true, "contact", "submit", "email", "", "field-1"))
	if pe != nil || fieldRemoved.Record.After.Present {
		t.Fatalf("remove flow field apply=%#v err=%v", fieldRemoved, pe)
	}
	undoFieldRemove := authoring.OperationRequest{ID: "field-undo", Kind: authoring.OperationUndo, Target: fieldRemoved.Record.Target, HistoryOperationID: "field-remove-1", ExpectedTargetHead: "field-remove-1"}
	undoneField, pe := apply(t, s, principal("a"), undoFieldRemove)
	if pe != nil || !undoneField.Record.After.Present || undoneField.Record.UndoOf != "field-remove-1" {
		t.Fatalf("remove flow field undo=%#v err=%v", undoneField, pe)
	}
}

func TestNewOperationFamiliesStaleBaseIsRejected(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, pe := apply(t, s, principal("a"), sharedField("s1", "hero-card", "title", "one", "")); pe != nil {
		t.Fatal(pe)
	}
	if _, pe := apply(t, s, principal("a"), sharedField("s2", "hero-card", "title", "two", "s1")); pe != nil {
		t.Fatal(pe)
	}
	_, pe := apply(t, s, principal("a"), sharedField("s3-stale", "hero-card", "title", "lost", "s1"))
	if pe == nil || pe.Code != collab.ErrorStaleField || pe.CurrentHead != "s2" {
		t.Fatalf("stale shared field=%#v", pe)
	}

	if _, pe := apply(t, s, designer("d"), interactionOp("i1", false, "home", "home:hero", "k", interactionValue(core.InteractionEffectFade), "")); pe != nil {
		t.Fatal(pe)
	}
	_, pe = apply(t, s, designer("d"), interactionOp("i2-stale", true, "home", "home:hero", "k", "", ""))
	if pe == nil || pe.Code != collab.ErrorStaleField || pe.CurrentHead != "i1" {
		t.Fatalf("stale interaction=%#v", pe)
	}
}

func TestNewOperationFamilyCapabilitiesRequireDesignForInteractionsAuthorForOthers(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if _, pe := apply(t, s, designer("designer-only"), sharedField("shared-forbidden", "hero-card", "title", "x", "")); pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("designer shared field=%#v", pe)
	}
	if _, pe := apply(t, s, designer("designer-only"), flowFieldOp("flow-forbidden", false, "contact", "submit", "email", authoring.EncodeFlowFieldSettings(core.FlowField{Kind: core.ControlText}), "")); pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("designer flow field=%#v", pe)
	}
	if _, pe := apply(t, s, principal("author-only"), interactionOp("interaction-forbidden", false, "home", "home:hero", "k", interactionValue(core.InteractionEffectFade), "")); pe == nil || pe.Code != collab.ErrorForbidden {
		t.Fatalf("author interaction=%#v", pe)
	}
	if _, pe := apply(t, s, principal("author-ok"), sharedField("shared-ok", "hero-card", "title", "x", "")); pe != nil {
		t.Fatalf("author shared field should be allowed: %v", pe)
	}
	if _, pe := apply(t, s, designer("designer-ok"), interactionOp("interaction-ok", false, "home", "home:hero", "k", interactionValue(core.InteractionEffectFade), "")); pe != nil {
		t.Fatalf("designer interaction should be allowed: %v", pe)
	}
}

// --- RepairFieldHead: host-callable field-head repair primitive ---

func repairer(id string) collab.Principal {
	return collab.Principal{ActorID: id, DisplayName: id, Capabilities: map[collab.Capability]bool{collab.CapabilityRepair: true}}
}

func TestRepairFieldHeadUnsticksPoisonedTargetAndAudits(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	first, pe := apply(t, s, principal("a"), field("op1", "title", "A", ""))
	if pe != nil {
		t.Fatal(pe)
	}
	second, pe := apply(t, s, principal("a"), field("op2", "title", "B", "op1"))
	if pe != nil {
		t.Fatal(pe)
	}
	// Simulate the poison scenario: a host recovers by reverting to op1's
	// value but the ledger head has already moved to op2, so a plain retry
	// referencing the pre-poison head is stuck.
	_, pe = apply(t, s, principal("a"), field("op3-stuck", "title", "C", "op1"))
	if pe == nil || pe.Code != collab.ErrorStaleField || pe.CurrentHead != "op2" {
		t.Fatalf("expected stuck submit before repair, got %#v", pe)
	}

	result, protocolErr := s.RepairFieldHead(context.Background(), collab.RepairFieldHeadCommand{
		Resource: resource(), Principal: repairer("ops"), Target: first.Record.Target,
		NewHead: "op1", NewValue: authoring.PresentValue("A"), Reason: "poison outbox recovery",
		Now: time.Date(2026, 7, 11, 2, 0, 0, 0, time.UTC),
	})
	if protocolErr != nil {
		t.Fatalf("repair failed: %v", protocolErr)
	}
	if result.OldHead != "op2" || result.NewHead != "op1" || result.NewValue.Value != "A" || result.ActorID != "ops" || result.Reason != "poison outbox recovery" {
		t.Fatalf("repair result=%#v", result)
	}
	if result.TargetKey != first.Record.TargetKey {
		t.Fatalf("repair target key=%q want %q", result.TargetKey, first.Record.TargetKey)
	}

	// The submit path is now un-stuck: a fresh op referencing the repaired
	// head is accepted.
	unstuck, pe := apply(t, s, principal("a"), field("op3-recovered", "title", "C", "op1"))
	if pe != nil || unstuck.Record.Before.Value != "A" {
		t.Fatalf("post-repair submit=%#v err=%v", unstuck, pe)
	}

	history, err := s.RepairHistory(context.Background(), resource(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].OldHead != "op2" || history[0].NewHead != "op1" || history[0].ActorID != "ops" || history[0].Reason != "poison outbox recovery" {
		t.Fatalf("repair history=%#v", history)
	}
	if !history[0].OldValue.Present || history[0].OldValue.Value != "B" || !history[0].NewValue.Present || history[0].NewValue.Value != "A" {
		t.Fatalf("repair history values=%#v", history[0])
	}

	// second's own record is untouched by the repair (it's a bookkeeping
	// overwrite of the field head, not a rewrite of history).
	_ = second
}

func TestRepairFieldHeadRejectsUnauthorizedPrincipal(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, pe := apply(t, s, principal("a"), field("op1", "title", "A", ""))
	if pe != nil {
		t.Fatal(pe)
	}
	_, protocolErr := s.RepairFieldHead(context.Background(), collab.RepairFieldHeadCommand{
		Resource: resource(), Principal: principal("not-a-repairer"), Target: first.Record.Target,
		NewHead: "op1", NewValue: authoring.PresentValue("A"), Reason: "should be rejected",
	})
	if protocolErr == nil || protocolErr.Code != collab.ErrorForbidden {
		t.Fatalf("unauthorized repair=%#v", protocolErr)
	}
	// The field head must be unchanged and no audit row written.
	_, pe = apply(t, s, principal("a"), field("op2-check", "title", "B", "op1"))
	if pe != nil {
		t.Fatalf("field head should still be op1: %v", pe)
	}
	history, err := s.RepairHistory(context.Background(), resource(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 0 {
		t.Fatalf("unauthorized repair must not audit: %#v", history)
	}
}

func TestRepairFieldHeadOnHealthyTargetIsIdempotent(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	first, pe := apply(t, s, principal("a"), field("op1", "title", "A", ""))
	if pe != nil {
		t.Fatal(pe)
	}
	cmd := collab.RepairFieldHeadCommand{
		Resource: resource(), Principal: repairer("ops"), Target: first.Record.Target,
		NewHead: "op1", NewValue: authoring.PresentValue("A"), Reason: "routine health check",
	}
	for i := 0; i < 2; i++ {
		result, protocolErr := s.RepairFieldHead(context.Background(), cmd)
		if protocolErr != nil {
			t.Fatalf("iteration %d: repair failed: %v", i, protocolErr)
		}
		if result.OldHead != "op1" || result.NewHead != "op1" || result.OldValue.Value != "A" || result.NewValue.Value != "A" {
			t.Fatalf("iteration %d: expected a no-op repair, got %#v", i, result)
		}
	}
	// The healthy head must still accept a normal submit afterward.
	next, pe := apply(t, s, principal("a"), field("op2", "title", "B", "op1"))
	if pe != nil || next.Record.Before.Value != "A" {
		t.Fatalf("post-idempotent-repair submit=%#v err=%v", next, pe)
	}
	history, err := s.RepairHistory(context.Background(), resource(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("expected an audit row per repair call, got %#v", history)
	}
}
