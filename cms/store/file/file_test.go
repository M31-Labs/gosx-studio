package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/cms/store/memory"
)

func TestStoreContracts(t *testing.T) {
	var _ cmsstore.Store = (*Store)(nil)
}

func TestNewCreatesParentDirsAndOpenLoadsSnapshot(t *testing.T) {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "nested", "cms", "snapshot.json")

	store, err := New(path, memory.Seed{}, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected snapshot file to be created: %v", err)
	}

	settings, err := store.SaveSiteSettings(cmsstore.SiteSettingsInput{
		Title:    "Studio",
		BaseURL:  "https://example.com/",
		Metadata: cmsstore.Metadata{"owner": "content"},
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.CreatePage(cmsstore.PageInput{
		Title: "Care Guide",
		State: cmsstore.State{Publish: cmsstore.PublishStatePublished},
		Body: blockstudio.Document{Blocks: []blockstudio.BlockInstance{{
			Key: "paragraph",
			Values: blockstudio.Values{
				"text": {Kind: blockstudio.FieldTextarea, String: "Keep notes"},
			},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	post, err := store.CreatePost(cmsstore.PostInput{
		Title: "Launch Notes",
		Tags:  []string{"News"},
	})
	if err != nil {
		t.Fatal(err)
	}
	revision, err := store.SaveRevision(lifecycle.RevisionInput{
		ResourceKind:  cmsstore.ResourceKindPage,
		ResourceID:    page.ID,
		ResourceTitle: page.Title,
		Action:        "page.saved",
		Snapshot:      page,
	})
	if err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, WithClock(func() time.Time { return now.Add(time.Hour) }))
	if err != nil {
		t.Fatal(err)
	}
	loadedSettings, ok, err := reopened.SiteSettings()
	if err != nil || !ok {
		t.Fatalf("expected settings, ok=%v err=%v", ok, err)
	}
	if loadedSettings.ID != settings.ID || loadedSettings.BaseURL != "https://example.com" || loadedSettings.Metadata["owner"] != "content" {
		t.Fatalf("unexpected loaded settings: %#v", loadedSettings)
	}
	loadedPage, ok, err := reopened.PageBySlug("Care Guide")
	if err != nil || !ok {
		t.Fatalf("expected page by slug, ok=%v err=%v", ok, err)
	}
	if loadedPage.ID != page.ID || loadedPage.Body.Blocks[0].Values["text"].String != "Keep notes" {
		t.Fatalf("unexpected loaded page: %#v", loadedPage)
	}
	loadedPost, ok, err := reopened.PostByID(post.ID)
	if err != nil || !ok {
		t.Fatalf("expected post by id, ok=%v err=%v", ok, err)
	}
	if loadedPost.Slug != "Launch-Notes" || len(loadedPost.Tags) != 1 || loadedPost.Tags[0] != "News" {
		t.Fatalf("unexpected loaded post: %#v", loadedPost)
	}
	loadedRevision, ok := reopened.RevisionByID(cmsstore.ResourceKindPage, page.ID, revision.ID)
	if !ok || loadedRevision.ResourceTitle != page.Title {
		t.Fatalf("expected revision by id, got %#v ok=%v", loadedRevision, ok)
	}
}

func TestOpenMissingFileStartsEmptyAndPersistsAfterMutation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.SiteSettings(); err != nil || ok {
		t.Fatalf("expected empty settings, ok=%v err=%v", ok, err)
	}

	if _, err := store.CreatePage(cmsstore.PageInput{Title: "Draft Page"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Pages) != 1 || snapshot.Pages[0].Slug != "Draft-Page" {
		t.Fatalf("expected persisted page snapshot, got %#v", snapshot.Pages)
	}
}

func TestNewReplacesExistingSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	first, err := New(path, memory.Seed{
		Pages: []cmsstore.Page{{ID: "page_seed", Slug: "seed", Title: "Seed"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.CreatePage(cmsstore.PageInput{Title: "Runtime"}); err != nil {
		t.Fatal(err)
	}

	_, err = New(path, memory.Seed{
		Posts: []cmsstore.Post{{ID: "post_seed", Slug: "post-seed", Title: "Post Seed"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := reopened.ListPages(cmsstore.PageFilter{})
	if err != nil {
		t.Fatal(err)
	}
	posts, err := reopened.ListPosts(cmsstore.PostFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 0 || len(posts) != 1 || posts[0].ID != "post_seed" {
		t.Fatalf("expected replacement snapshot, pages=%#v posts=%#v", pages, posts)
	}
}
