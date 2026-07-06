package flows

import (
	"testing"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
)

func TestMemoryStoreDocumentsCloneAndUpsert(t *testing.T) {
	doc := DocumentFromDefinition(Contact("contact.submit"), DocumentOptions{ID: "contact-flow"})
	store := NewMemoryStore(doc)
	loaded, ok := store.GetFlowDocument("contact-flow")
	if !ok || loaded.Key != FlowKeyContact {
		t.Fatalf("expected seeded document, got %#v %v", loaded, ok)
	}
	loaded.Label = "Changed"
	again, _ := store.GetFlowDocument("contact")
	if again.Label == "Changed" {
		t.Fatalf("expected loaded document to be cloned")
	}
	loaded.Label = "Contact us"
	if err := store.SaveFlowDocument(loaded); err != nil {
		t.Fatal(err)
	}
	all := store.ListFlowDocuments()
	if len(all) != 1 || all[0].Label != "Contact us" {
		t.Fatalf("expected upserted document, got %#v", all)
	}
}

func TestMemoryStoreDraftsAndPublications(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	doc := DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow"})
	store := NewMemoryStore()
	draft := NewDraft(doc, "author", "", now)
	if err := store.SaveFlowDraft(draft); err != nil {
		t.Fatal(err)
	}
	loadedDraft, ok := store.GetFlowDraft("tour-flow")
	if !ok || loadedDraft.Document.Key != FlowKeyScheduleTour || loadedDraft.State != lifecycle.DraftStateDraft {
		t.Fatalf("expected draft, got %#v %v", loadedDraft, ok)
	}
	publication, err := NewPublication(doc, "author", "rev_1", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFlowPublication(publication); err != nil {
		t.Fatal(err)
	}
	loadedPublication, ok := store.GetFlowPublication(FlowKeyScheduleTour)
	if !ok || loadedPublication.RevisionID != "rev_1" || loadedPublication.Document.State != lifecycle.PublishStatePublished {
		t.Fatalf("expected publication, got %#v %v", loadedPublication, ok)
	}
	loadedPublication.Document.Label = "Changed"
	again, _ := store.GetFlowPublication("tour-flow")
	if again.Document.Label == "Changed" {
		t.Fatalf("expected publication to be cloned")
	}
}
