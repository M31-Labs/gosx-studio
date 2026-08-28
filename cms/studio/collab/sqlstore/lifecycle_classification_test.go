package sqlstore_test

// This file proves the "currently unrepresentable" caveat
// cms/lifecycle/engine/change_set.go documents for the instances/
// interactions/flows DraftChangeScope values is no longer accurate for the
// instance/interaction/flow durable operation families added alongside it
// (see authoring/operations.go's OperationSetSharedField/OverrideInstance/
// DetachInstance/RestoreInstance/SetInteraction/RemoveInteraction/
// SetFlowField/RemoveFlowField/SetFlowAction kinds): a REAL record, applied
// and read back through the sqlite-backed ledger exactly the way a host
// would, now classifies into the matching engine.DraftChangeScope.
//
// This is an external test package (sqlstore_test, not sqlstore) precisely so
// it can import cms/lifecycle/engine without an import cycle -- see the
// task's HARD BOUNDARY: cms/lifecycle/ is owned by another lane, so this
// test only ever imports engine, never edits it.

import (
	"context"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/lifecycle/engine"
	"m31labs.dev/gosx-studio/cms/studio/collab"
	"m31labs.dev/gosx-studio/cms/studio/collab/sqlstore"
	"m31labs.dev/gosx-studio/core"
)

func classificationResource() collab.ResourceKey {
	return collab.ResourceKey{TenantID: "tenant", ProjectID: "project", Kind: "site", ID: "classification"}
}

func classificationAuthor(id string) collab.Principal {
	return collab.Principal{ActorID: id, DisplayName: id, Capabilities: map[collab.Capability]bool{collab.CapabilityAuthor: true}}
}

func classificationDesigner(id string) collab.Principal {
	return collab.Principal{ActorID: id, DisplayName: id, Capabilities: map[collab.Capability]bool{collab.CapabilityDesign: true}}
}

func TestOperationChangeSetClassifiesRealPersistedInstanceInteractionAndFlowRecords(t *testing.T) {
	store, err := sqlstore.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	now := time.Date(2026, 7, 11, 3, 0, 0, 0, time.UTC)

	// One real, persisted record per family this slice makes durable:
	// instances (set-shared-field), interactions (set-interaction), and
	// flows (set-flow-field). Each round-trips through Store.Apply exactly
	// the way a host's collaboration ledger would accept it.
	sharedAck, protocolErr := store.Apply(ctx, collab.ApplyCommand{
		Resource: classificationResource(), Principal: classificationAuthor("a"), Now: now,
		Request: authoring.OperationRequest{
			ID: "shared-1", Kind: authoring.OperationSetSharedField,
			Target: authoring.OperationTarget{Route: "/", ComponentKey: "hero-card", ControlKey: "title", Field: authoring.FieldInstancesSharedField},
			Value:  "Welcome",
		},
	})
	if protocolErr != nil {
		t.Fatalf("apply shared field: %v", protocolErr)
	}

	interactionAck, protocolErr := store.Apply(ctx, collab.ApplyCommand{
		Resource: classificationResource(), Principal: classificationDesigner("d"), Now: now,
		Request: authoring.OperationRequest{
			ID: "interaction-1", Kind: authoring.OperationSetInteraction,
			Target: authoring.OperationTarget{Route: "/", PageID: "home", ComponentKey: "home:hero", ControlKey: "home:hero:reveal", Field: authoring.FieldInteractionsEntry},
			Value:  authoring.EncodeInteractionSettings(core.Interaction{Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFade, DurationMS: 250}),
		},
	})
	if protocolErr != nil {
		t.Fatalf("apply interaction: %v", protocolErr)
	}

	flowFieldAck, protocolErr := store.Apply(ctx, collab.ApplyCommand{
		Resource: classificationResource(), Principal: classificationAuthor("a"), Now: now,
		Request: authoring.OperationRequest{
			ID: "flow-field-1", Kind: authoring.OperationSetFlowField,
			Target: authoring.OperationTarget{Route: "/", PageID: "contact", ComponentKey: "submit", ControlKey: "email", Field: authoring.FieldFlowsField},
			Value:  authoring.EncodeFlowFieldSettings(core.FlowField{Label: "Email", Kind: core.ControlText, Required: true}),
		},
	})
	if protocolErr != nil {
		t.Fatalf("apply flow field: %v", protocolErr)
	}

	// Read the accepted records back exactly the way a host's publish
	// surface would -- through Tail, not by reusing the in-process Apply
	// return values -- so this is provably a REAL persisted record, not a
	// hand-constructed struct literal.
	tail, err := store.Tail(ctx, classificationResource(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 3 {
		t.Fatalf("expected 3 persisted records, got %d", len(tail))
	}

	records := make([]authoring.OperationRecord, 0, len(tail))
	for _, ack := range tail {
		records = append(records, ack.Record)
	}
	changes := engine.OperationChangeSet(records, 0)
	if len(changes) != 3 {
		t.Fatalf("expected 3 classified changes, got %#v", changes)
	}

	scopeByOpID := map[string]engine.DraftChangeScope{}
	for _, change := range changes {
		if len(change.OpIDs) != 1 {
			t.Fatalf("expected exactly one op id per change, got %#v", change)
		}
		scopeByOpID[change.OpIDs[0]] = change.Scope
	}

	if got := scopeByOpID[sharedAck.Record.ID]; got != engine.ScopeInstances {
		t.Fatalf("set-shared-field classified as %s, want %s", got, engine.ScopeInstances)
	}
	if got := scopeByOpID[interactionAck.Record.ID]; got != engine.ScopeInteractions {
		t.Fatalf("set-interaction classified as %s, want %s", got, engine.ScopeInteractions)
	}
	if got := scopeByOpID[flowFieldAck.Record.ID]; got != engine.ScopeFlows {
		t.Fatalf("set-flow-field classified as %s, want %s", got, engine.ScopeFlows)
	}
}
