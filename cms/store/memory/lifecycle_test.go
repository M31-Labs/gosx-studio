package memory

import (
	"testing"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

func TestPagePreviewPublishAndRestoreRevision(t *testing.T) {
	var _ cmsstore.LifecycleStore = (*Store)(nil)

	clock := steppingClock(
		time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 16, 11, 0, 0, 0, time.UTC),
	)
	store := New(Seed{}, WithClock(clock))

	preview, previewRevision, err := store.PreviewPage("", cmsstore.PageInput{
		Title: "Forest Guide",
		Body:  paragraphDoc("Preview copy"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.ID != "page_1" || preview.State.Draft != cmsstore.DraftStatePreview || preview.State.Publish != cmsstore.PublishStateDraft || preview.State.RevisionID != previewRevision.ID {
		t.Fatalf("unexpected preview page: %#v revision=%#v", preview, previewRevision)
	}
	if previewRevision.Action != cmsstore.ActionPagePreviewSaved || previewRevision.ResourceTitle != "Forest Guide" {
		t.Fatalf("unexpected preview revision: %#v", previewRevision)
	}

	published, publishRevision, err := store.PublishPage(preview.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !published.State.IsPublished() || published.State.PublishedAt == nil || published.State.RevisionID != publishRevision.ID {
		t.Fatalf("unexpected published page: %#v revision=%#v", published, publishRevision)
	}
	if publishRevision.Action != cmsstore.ActionPagePublished {
		t.Fatalf("unexpected publish revision: %#v", publishRevision)
	}

	restored, restoreRevision, err := store.RestorePageRevision(preview.ID, previewRevision.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.State.Draft != cmsstore.DraftStateRollback || restored.State.Publish != cmsstore.PublishStateDraft || restored.State.RevisionID != restoreRevision.ID {
		t.Fatalf("unexpected restored page state: %#v revision=%#v", restored, restoreRevision)
	}
	if restored.Body.Blocks[0].Values["text"].String != "Preview copy" || restoreRevision.Action != cmsstore.ActionPageRestored {
		t.Fatalf("unexpected restored page: %#v revision=%#v", restored, restoreRevision)
	}
	revisions := store.ListRevisions(cmsstore.RevisionFilter{ResourceKind: cmsstore.ResourceKindPage, ResourceID: preview.ID})
	if len(revisions) != 3 || revisions[0].ID != restoreRevision.ID || revisions[2].ID != previewRevision.ID {
		t.Fatalf("expected newest-first revision history, got %#v", revisions)
	}
}

func TestPostPreviewPublishAndRestoreRevision(t *testing.T) {
	store := New(Seed{}, WithClock(steppingClock(
		time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 16, 11, 0, 0, 0, time.UTC),
	)))

	preview, previewRevision, err := store.PreviewPost("", cmsstore.PostInput{
		Title:  "Pack List",
		Author: "Guides",
		Body:   paragraphDoc("Layers"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.ID != "post_1" || preview.State.Draft != cmsstore.DraftStatePreview || preview.State.RevisionID != previewRevision.ID {
		t.Fatalf("unexpected preview post: %#v revision=%#v", preview, previewRevision)
	}

	published, publishRevision, err := store.PublishPost(preview.ID)
	if err != nil {
		t.Fatal(err)
	}
	if published.State.Publish != cmsstore.PublishStatePublished || publishRevision.Action != cmsstore.ActionPostPublished {
		t.Fatalf("unexpected published post: %#v revision=%#v", published, publishRevision)
	}

	restored, restoreRevision, err := store.RestorePostRevision(preview.ID, previewRevision.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.State.Draft != cmsstore.DraftStateRollback || restored.Body.Blocks[0].Values["text"].String != "Layers" || restoreRevision.Action != cmsstore.ActionPostRestored {
		t.Fatalf("unexpected restored post: %#v revision=%#v", restored, restoreRevision)
	}
}

func TestSiteSettingsPreviewPublishAndRestoreRevision(t *testing.T) {
	store := New(Seed{}, WithClock(steppingClock(
		time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 16, 11, 0, 0, 0, time.UTC),
	)))

	preview, previewRevision, err := store.PreviewSiteSettings(cmsstore.SiteSettingsInput{
		Title:       "Pajaritos",
		Description: "Preview",
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.ID != "site" || preview.State.Draft != cmsstore.DraftStatePreview || preview.State.RevisionID != previewRevision.ID {
		t.Fatalf("unexpected settings preview: %#v revision=%#v", preview, previewRevision)
	}

	published, publishRevision, err := store.PublishSiteSettings()
	if err != nil {
		t.Fatal(err)
	}
	if published.State.Publish != cmsstore.PublishStatePublished || publishRevision.Action != cmsstore.ActionSettingsPublished {
		t.Fatalf("unexpected published settings: %#v revision=%#v", published, publishRevision)
	}

	restored, restoreRevision, err := store.RestoreSiteSettingsRevision(previewRevision.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Description != "Preview" || restored.State.Draft != cmsstore.DraftStateRollback || restoreRevision.Action != cmsstore.ActionSettingsRestored {
		t.Fatalf("unexpected restored settings: %#v revision=%#v", restored, restoreRevision)
	}
}

func paragraphDoc(text string) blockstudio.Document {
	return blockstudio.Document{Version: 1, Blocks: []blockstudio.BlockInstance{{
		ID:      "paragraph-1",
		Key:     "paragraph",
		Enabled: true,
		Values: blockstudio.Values{
			"text": {Kind: blockstudio.FieldTextarea, String: text},
		},
	}}}
}

func steppingClock(values ...time.Time) Clock {
	index := 0
	return func() time.Time {
		if len(values) == 0 {
			return time.Time{}
		}
		if index >= len(values) {
			return values[len(values)-1]
		}
		value := values[index]
		index++
		return value
	}
}
