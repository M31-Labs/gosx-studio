package flows

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
)

func TestFileStorePersistsDocumentsDraftsAndPublications(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "nested", "flows.json")
	doc := DocumentFromDefinition(Contact("contact.submit"), DocumentOptions{ID: "contact-flow"})

	store, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFlowDocument(doc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected save to create parent dirs and file: %v", err)
	}
	reloaded, err := NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	loadedDoc, ok := reloaded.GetFlowDocument("contact-flow")
	if !ok || loadedDoc.Key != FlowKeyContact {
		t.Fatalf("expected persisted document, got %#v %v", loadedDoc, ok)
	}

	draft := NewDraft(doc, "author", "", now)
	draft.Document.Label = "Contact draft"
	if err := store.SaveFlowDraft(draft); err != nil {
		t.Fatal(err)
	}
	reloaded, err = NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	loadedDraft, ok := reloaded.GetFlowDraft(FlowKeyContact)
	if !ok || loadedDraft.Document.Label != "Contact draft" || loadedDraft.State != lifecycle.DraftStateDraft {
		t.Fatalf("expected persisted draft, got %#v %v", loadedDraft, ok)
	}

	publication, err := NewPublication(doc, "publisher", "rev_1", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFlowPublication(publication); err != nil {
		t.Fatal(err)
	}
	reloaded, err = NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	loadedPublication, ok := reloaded.GetFlowPublication("contact-flow")
	if !ok || loadedPublication.RevisionID != "rev_1" || loadedPublication.Document.State != lifecycle.PublishStatePublished {
		t.Fatalf("expected persisted publication, got %#v %v", loadedPublication, ok)
	}
}

func TestFileStoreLoadsExistingDataOverSeedDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flows.json")
	seed := DocumentFromDefinition(Newsletter("newsletter.submit"), DocumentOptions{ID: "newsletter-flow"})

	store, err := NewFileStore(path, seed)
	if err != nil {
		t.Fatal(err)
	}
	seed.Label = "Saved newsletter"
	if err := store.SaveFlowDocument(seed); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewFileStore(path, DocumentFromDefinition(Newsletter("seed.submit"), DocumentOptions{ID: "newsletter-flow"}))
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok := reloaded.GetFlowDocument("newsletter-flow")
	if !ok || loaded.Label != "Saved newsletter" || loaded.Actions[0].HandlerRef != "newsletter.submit" {
		t.Fatalf("expected persisted document to override seed, got %#v %v", loaded, ok)
	}
}

func TestNewFileStoreRejectsEmptyPath(t *testing.T) {
	if _, err := NewFileStore(" "); err == nil {
		t.Fatal("expected empty path error")
	}
}
