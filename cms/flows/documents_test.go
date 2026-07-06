package flows

import (
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

func TestDocumentFromDefinitionNormalizesAndClones(t *testing.T) {
	definition := Contact("contact.submit")
	definition.Steps[0].Blocks = blockstudio.Document{
		Blocks: []blockstudio.BlockInstance{{
			Key:     " Intro ",
			Enabled: true,
			Values: blockstudio.Values{
				"text": {Kind: blockstudio.FieldText, String: "Hello"},
			},
		}},
	}
	doc := DocumentFromDefinition(definition, DocumentOptions{ID: " flow_1 "})
	if doc.ID != "flow_1" || doc.Key != FlowKeyContact || doc.Version != 1 || doc.State != lifecycle.PublishStateDraft {
		t.Fatalf("unexpected document defaults: %#v", doc)
	}
	if doc.Steps[0].Blocks.Kind != DocumentKind+".contact.message" || doc.Steps[0].Blocks.Blocks[0].Key != "intro" {
		t.Fatalf("unexpected normalized step blocks: %#v", doc.Steps[0].Blocks)
	}
	definition.Steps[0].Blocks.Blocks[0].Values["text"] = blockstudio.Value{Kind: blockstudio.FieldText, String: "Changed"}
	if got := doc.Steps[0].Blocks.Blocks[0].Values["text"].String; got != "Hello" {
		t.Fatalf("expected document to clone definition blocks, got %q", got)
	}
}

func TestNormalizeDocumentUsesStepBlockCatalog(t *testing.T) {
	doc := NormalizeDocument(Document{
		Key:   "Custom Flow",
		Label: "Custom flow",
		Steps: []DocumentStep{{
			Label: "Intro",
			Blocks: blockstudio.Document{Blocks: []blockstudio.BlockInstance{{
				Key:     "copy",
				Enabled: true,
				Values:  blockstudio.Values{"body": {Kind: blockstudio.FieldTextarea, String: "Start"}},
			}}},
		}},
		Actions: []DocumentAction{{Key: "submit", Label: "Submit", HandlerRef: "custom.submit"}},
	}, WithStepBlockCatalog("intro", []blockstudio.Definition{{
		Key:       "copy",
		Label:     "Copy",
		DefaultOn: true,
		Fields: []blockstudio.FieldDefinition{
			{Name: "body", Label: "Body", Kind: blockstudio.FieldTextarea, Required: true},
		},
	}}))
	if doc.Key != "custom-flow" || doc.Steps[0].Key != "intro" {
		t.Fatalf("unexpected normalized document keys: %#v", doc)
	}
	if doc.Steps[0].Blocks.Blocks[0].ID != "copy" || doc.Steps[0].Blocks.Blocks[0].Order != 1 {
		t.Fatalf("expected blockstudio normalization, got %#v", doc.Steps[0].Blocks.Blocks[0])
	}
}

func TestStandardDocumentsCoverPresetFlowKinds(t *testing.T) {
	docs := StandardDocuments(HandlerRefs{
		FlowKeyContact:         "contact.submit",
		FlowKeyPurchaseRequest: "purchase.submit",
		FlowKeyCheckoutHandoff: "checkout.continue",
		FlowKeyNewsletter:      "newsletter.submit",
		FlowKeyAppointment:     "appointment.submit",
		FlowKeyScheduleTour:    "tour.submit",
		FlowKeyEnrollment:      "enrollment.submit",
	}, DocumentOptions{})
	if len(docs) != 7 {
		t.Fatalf("expected seven standard documents, got %#v", docs)
	}
	want := []string{
		FlowKeyContact,
		FlowKeyPurchaseRequest,
		FlowKeyCheckoutHandoff,
		FlowKeyNewsletter,
		FlowKeyAppointment,
		FlowKeyScheduleTour,
		FlowKeyEnrollment,
	}
	for index, key := range want {
		if docs[index].Key != key {
			t.Fatalf("expected key %q at index %d, got %#v", key, index, docs)
		}
		if errs := ValidateDocument(docs[index]); len(errs) != 0 {
			t.Fatalf("expected standard document to validate: %#v %#v", docs[index], errs)
		}
	}
}

func TestDraftPublishAndRevisionRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	doc := DocumentFromDefinition(Newsletter("newsletter.submit"), DocumentOptions{ID: "newsletter-flow"})
	draft := NewDraft(doc, "author_1", "rev_old", now)
	if draft.State != lifecycle.DraftStateDraft || draft.Document.Updated != now || draft.BaseRevisionID != "rev_old" {
		t.Fatalf("unexpected draft: %#v", draft)
	}
	result, err := PublishDraft(draft, "author_2", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if result.Publication.Document.State != lifecycle.PublishStatePublished || result.Publication.RevisionID == "" || result.Publication.Document.Updated != now.Add(time.Hour) {
		t.Fatalf("unexpected publication: %#v", result.Publication)
	}
	if result.Revision.ResourceKind != ResourceKind || result.Revision.ResourceID != "newsletter-flow" || result.Revision.Action != ActionPublished {
		t.Fatalf("unexpected revision: %#v", result.Revision)
	}
	decoded, err := DecodeDocumentRevision(result.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Key != FlowKeyNewsletter || decoded.State != lifecycle.PublishStatePublished {
		t.Fatalf("unexpected decoded document: %#v", decoded)
	}
	filter := DocumentRevisionFilter(doc, 5)
	if filter.ResourceKind != ResourceKind || filter.ResourceID != "newsletter-flow" || filter.Limit != 5 {
		t.Fatalf("unexpected revision filter: %#v", filter)
	}
}

func TestSaveConfiguredDraftUpdatesHandlerRefs(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	doc := DocumentFromDefinition(Contact("contact.submit"), DocumentOptions{ID: "contact-flow"})
	draft, err := SaveConfiguredDraft(store, doc, DraftConfig{
		AuthorID:       "author_1",
		BaseRevisionID: "rev_base",
		HandlerRefs:    map[string]string{"submit": "contact.custom_submit"},
		StepLabels:     map[string]string{"message": "Studio message"},
		Now:            now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.AuthorID != "author_1" || draft.BaseRevisionID != "rev_base" || draft.Updated != now {
		t.Fatalf("unexpected configured draft metadata: %#v", draft)
	}
	loaded, ok := store.GetFlowDraft("contact-flow")
	if !ok || loaded.Document.Actions[0].HandlerRef != "contact.custom_submit" || loaded.Document.Steps[0].Label != "Studio message" {
		t.Fatalf("expected configured draft, got %#v %v", loaded, ok)
	}
	doc.Actions[0].HandlerRef = "mutated.after.save"
	again, _ := store.GetFlowDraft(FlowKeyContact)
	if again.Document.Actions[0].HandlerRef != "contact.custom_submit" {
		t.Fatalf("expected stored draft to be isolated from caller mutation: %#v", again)
	}
}

func TestPublishStoredDraftStoresPublication(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	doc := DocumentFromDefinition(Newsletter("newsletter.submit"), DocumentOptions{ID: "newsletter-flow"})
	if _, err := SaveConfiguredDraft(store, doc, DraftConfig{AuthorID: "author_1", Now: now}); err != nil {
		t.Fatal(err)
	}
	result, err := PublishStoredDraft(store, FlowKeyNewsletter, "publisher_1", now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	loaded, ok := store.GetFlowPublication("newsletter-flow")
	if !ok || loaded.RevisionID != result.Revision.ID || loaded.Document.State != lifecycle.PublishStatePublished || loaded.AuthorID != "publisher_1" {
		t.Fatalf("expected stored publication, got %#v %v", loaded, ok)
	}
	if result.Revision.ResourceID != "newsletter-flow" || result.Revision.Action != ActionPublished {
		t.Fatalf("unexpected publish revision: %#v", result.Revision)
	}
}

func TestInstanceFromDocumentBuildsRuntimeDefinition(t *testing.T) {
	doc := DocumentFromDefinition(ScheduleTour("tour.submit"), DocumentOptions{ID: "tour-flow", State: lifecycle.PublishStatePublished})
	instance := InstanceFromDocument(doc, InstanceOptions{ID: "slot_1", RevisionID: "rev_1"})
	if instance.ID != "slot_1" || instance.DocumentID != "tour-flow" || instance.FlowKey != FlowKeyScheduleTour {
		t.Fatalf("unexpected instance identity: %#v", instance)
	}
	if instance.State != lifecycle.PublishStatePublished || instance.Definition.Actions[0].HandlerRef != "tour.submit" {
		t.Fatalf("unexpected instance definition: %#v", instance)
	}
	instance.Source.Actions[0].HandlerRef = "changed"
	if strings.Contains(instance.Definition.Actions[0].HandlerRef, "changed") {
		t.Fatalf("expected instance definition to be isolated from source mutation")
	}
}
