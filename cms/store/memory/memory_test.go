package memory

import (
	"strings"
	"testing"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

func TestStoreContracts(t *testing.T) {
	var _ cmsstore.Store = (*Store)(nil)
}

func TestPageCRUDClonesAndFilters(t *testing.T) {
	now := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	store := New(Seed{}, WithClock(func() time.Time { return now }))
	page, err := store.CreatePage(cmsstore.PageInput{
		Title: "Forest Guide",
		State: cmsstore.State{Publish: cmsstore.PublishStatePublished},
		Body: blockstudio.Document{Blocks: []blockstudio.BlockInstance{{
			Key: "paragraph",
			Values: blockstudio.Values{
				"text": {Kind: blockstudio.FieldTextarea, String: "Hello"},
			},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.ID != "page_1" || page.Slug != "Forest-Guide" || page.Created != now || page.Updated != now {
		t.Fatalf("unexpected page: %#v", page)
	}
	page.Body.Blocks[0].Values["text"] = blockstudio.Value{String: "changed"}
	loaded, ok, err := store.PageBySlug("Forest Guide")
	if err != nil || !ok {
		t.Fatalf("expected page by slug, ok=%v err=%v", ok, err)
	}
	if loaded.Body.Blocks[0].Values["text"].String != "Hello" {
		t.Fatalf("expected stored page to be isolated from mutation: %#v", loaded)
	}
	published, err := store.ListPages(cmsstore.PageFilter{Publish: cmsstore.PublishStatePublished, Limit: 1})
	if err != nil || len(published) != 1 {
		t.Fatalf("expected published page, got %#v err=%v", published, err)
	}
	updated, err := store.UpdatePage(page.ID, cmsstore.PageInput{Slug: "forest-care", Title: "Forest Care"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Slug != "forest-care" || updated.Created != now || updated.Updated != now {
		t.Fatalf("unexpected updated page: %#v", updated)
	}
}

func TestPostFiltersAndValidation(t *testing.T) {
	store := New(Seed{})
	if _, err := store.CreatePost(cmsstore.PostInput{}); err == nil {
		t.Fatal("expected invalid post error")
	}
	if _, err := store.CreatePost(cmsstore.PostInput{
		Title: "Packing List",
		Tags:  []string{"Family", "Readiness"},
		State: cmsstore.State{Publish: cmsstore.PublishStatePublished},
	}); err != nil {
		t.Fatal(err)
	}
	posts, err := store.ListPosts(cmsstore.PostFilter{Tag: "family", Publish: cmsstore.PublishStatePublished})
	if err != nil || len(posts) != 1 || posts[0].Slug != "Packing-List" {
		t.Fatalf("unexpected posts: %#v err=%v", posts, err)
	}
}

func TestSettingsAndSnapshotClone(t *testing.T) {
	store := New(Seed{})
	if _, ok, err := store.SiteSettings(); err != nil || ok {
		t.Fatalf("expected no initial settings, ok=%v err=%v", ok, err)
	}
	settings, err := store.SaveSiteSettings(cmsstore.SiteSettingsInput{
		Title:    "Pajaritos",
		Metadata: cmsstore.Metadata{"tagline": "Outdoor learning"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if settings.ID != "site" || settings.Title != "Pajaritos" {
		t.Fatalf("unexpected settings: %#v", settings)
	}
	snapshot := store.Snapshot()
	snapshot.Settings.Metadata["tagline"] = "changed"
	loaded, ok, _ := store.SiteSettings()
	if !ok || loaded.Metadata["tagline"] != "Outdoor learning" {
		t.Fatalf("expected cloned snapshot, got %#v", loaded)
	}
}

func TestRevisionStore(t *testing.T) {
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	store := New(Seed{}, WithClock(func() time.Time { return now }))
	revision, err := store.SaveRevision(lifecycle.RevisionInput{
		ResourceKind: cmsstore.ResourceKindPage,
		ResourceID:   "page_1",
		Action:       "page.saved",
		Snapshot:     map[string]string{"title": "Forest"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(revision.ID, "rev_") || revision.Created != now {
		t.Fatalf("unexpected revision: %#v", revision)
	}
	list := store.ListRevisions(cmsstore.RevisionFilter{ResourceKind: cmsstore.ResourceKindPage, ResourceID: "page_1"})
	if len(list) != 1 || list[0].ID != revision.ID {
		t.Fatalf("unexpected revisions: %#v", list)
	}
	found, ok := store.RevisionByID(cmsstore.ResourceKindPage, "page_1", revision.ID)
	if !ok || found.ID != revision.ID {
		t.Fatalf("expected revision by id, got %#v %v", found, ok)
	}
}
