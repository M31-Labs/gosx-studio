package sqlstore

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/studio/collab"
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
