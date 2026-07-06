package studio

import (
	"testing"
	"time"

	cmsflows "m31labs.dev/gosx-studio/cms/flows"
)

func TestConfiguredFlowDefinitionsPrefersDraftThenPublicationThenDocument(t *testing.T) {
	now := time.Date(2026, 5, 20, 17, 30, 0, 0, time.UTC)
	store := cmsflows.NewMemoryStore()
	base := cmsflows.Catalog(cmsflows.Contact("base.contact"), cmsflows.ScheduleTour("base.tour"), cmsflows.Newsletter("base.newsletter"))

	document := cmsflows.DocumentFromDefinition(cmsflows.Contact("document.contact"), cmsflows.DocumentOptions{ID: cmsflows.FlowKeyContact})
	if err := store.SaveFlowDocument(document); err != nil {
		t.Fatal(err)
	}
	published := cmsflows.DocumentFromDefinition(cmsflows.ScheduleTour("publication.tour"), cmsflows.DocumentOptions{ID: cmsflows.FlowKeyScheduleTour})
	publication, err := cmsflows.NewPublication(published, "studio", "rev_tour", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFlowPublication(publication); err != nil {
		t.Fatal(err)
	}
	draftDoc := cmsflows.DocumentFromDefinition(cmsflows.Newsletter("draft.newsletter"), cmsflows.DocumentOptions{ID: cmsflows.FlowKeyNewsletter})
	if _, err := cmsflows.SaveConfiguredDraft(store, draftDoc, cmsflows.DraftConfig{AuthorID: "studio", Now: now}); err != nil {
		t.Fatal(err)
	}

	configured := ConfiguredFlowDefinitions(store, base)
	contact, _ := cmsflows.Find(configured, cmsflows.FlowKeyContact)
	tour, _ := cmsflows.Find(configured, cmsflows.FlowKeyScheduleTour)
	newsletter, _ := cmsflows.Find(configured, cmsflows.FlowKeyNewsletter)
	if contact.Actions[0].HandlerRef != "document.contact" {
		t.Fatalf("expected document precedence for contact, got %q", contact.Actions[0].HandlerRef)
	}
	if tour.Actions[0].HandlerRef != "publication.tour" {
		t.Fatalf("expected publication precedence for tour, got %q", tour.Actions[0].HandlerRef)
	}
	if newsletter.Actions[0].HandlerRef != "draft.newsletter" {
		t.Fatalf("expected draft precedence for newsletter, got %q", newsletter.Actions[0].HandlerRef)
	}
}

func TestSaveConfiguredFlowDraftsUsesStudioFormNames(t *testing.T) {
	now := time.Date(2026, 5, 20, 17, 40, 0, 0, time.UTC)
	store := cmsflows.NewMemoryStore()
	definitions := []cmsflows.Definition{cmsflows.ScheduleTour("tour.submit")}
	form := map[string]string{
		FlowHandlerRefInputName(cmsflows.FlowKeyScheduleTour):                              "tour.request",
		FlowStepLabelInputName(cmsflows.FlowKeyScheduleTour, "request"):                    "Family details",
		FlowStepLabelInputName(cmsflows.FlowKeyScheduleTour, "not-a-real-step-is-ignored"): "Ignored",
	}
	if err := SaveConfiguredFlowDrafts(store, definitions, form, FlowDraftSaveOptions{AuthorID: "author", Now: now}); err != nil {
		t.Fatal(err)
	}
	draft, ok := store.GetFlowDraft(cmsflows.FlowKeyScheduleTour)
	if !ok {
		t.Fatal("expected draft")
	}
	if draft.AuthorID != "author" || !draft.Updated.Equal(now) {
		t.Fatalf("unexpected draft metadata: %#v", draft)
	}
	definition := cmsflows.DefinitionFromDocument(draft.Document)
	if definition.Actions[0].HandlerRef != "tour.request" {
		t.Fatalf("unexpected handler ref: %q", definition.Actions[0].HandlerRef)
	}
	if definition.Steps[0].Label != "Family details" {
		t.Fatalf("unexpected step labels: %#v", definition.Steps)
	}
}

func TestPublishConfiguredFlowSavesDraftAndPublication(t *testing.T) {
	now := time.Date(2026, 5, 20, 17, 45, 0, 0, time.UTC)
	store := cmsflows.NewMemoryStore()
	definitions := []cmsflows.Definition{cmsflows.Newsletter("newsletter.submit")}
	form := map[string]string{
		FlowHandlerRefInputName(cmsflows.FlowKeyNewsletter): "newsletter.signup",
	}
	result, err := PublishConfiguredFlow(store, definitions, form, FlowPublishOptions{
		FlowKey:  cmsflows.FlowKeyNewsletter,
		AuthorID: "publisher",
		Now:      now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Publication.AuthorID != "publisher" || result.Publication.Document.Actions[0].HandlerRef != "newsletter.signup" {
		t.Fatalf("unexpected publication: %#v", result.Publication)
	}
	if result.Revision.ID == "" || result.Revision.ResourceKind != cmsflows.ResourceKind {
		t.Fatalf("unexpected revision: %#v", result.Revision)
	}
	if _, ok := store.GetFlowPublication(cmsflows.FlowKeyNewsletter); !ok {
		t.Fatal("expected saved publication")
	}
}
