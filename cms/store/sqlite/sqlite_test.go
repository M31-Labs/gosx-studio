package sqlite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"m31labs.dev/gosx-admin/blockstudio"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/cms/store/memory"
)

func body(text string) blockstudio.Document {
	return blockstudio.Document{Version: 1, Kind: "body", Blocks: []blockstudio.BlockInstance{{
		ID: "p-0", Key: "paragraph", Enabled: true, Order: 0,
		Values: blockstudio.Values{"text": blockstudio.Value{Kind: blockstudio.FieldText, String: text}},
	}}}
}

func TestRowsPersistAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "site.db")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveSiteSettings(cmsstore.SiteSettingsInput{Title: "Wildflower", Metadata: cmsstore.Metadata{"k": "v"}}); err != nil {
		t.Fatal(err)
	}
	page, err := store.CreatePage(cmsstore.PageInput{Slug: "home", Title: "Home", Body: body("Hello")})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PreviewPage(page.ID, cmsstore.PageInput{Slug: "home", Title: "Home", Body: body("Draft one"), State: page.State}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PublishPage(page.ID); err != nil {
		t.Fatal(err)
	}
	post, err := store.CreatePost(cmsstore.PostInput{Slug: "news", Title: "News", Tags: []string{"a"}, Body: body("Post")})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PublishPost(post.ID); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for table, want := range map[string]int{"pages": 1, "posts": 1, "settings": 1} {
		if n, _ := store.countRows(ctx, table); n != want {
			t.Fatalf("%s rows = %d, want %d", table, n, want)
		}
	}
	revisions, _ := store.countRows(ctx, "revisions")
	if revisions != len(store.ListRevisions(cmsstore.RevisionFilter{})) || revisions < 3 {
		t.Fatalf("revision rows = %d, listed = %d", revisions, len(store.ListRevisions(cmsstore.RevisionFilter{})))
	}
	store.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	settings, ok, _ := reopened.SiteSettings()
	if !ok || settings.Title != "Wildflower" || settings.Metadata["k"] != "v" {
		t.Fatalf("settings after reopen: %+v %v", settings, ok)
	}
	got, ok, _ := reopened.PageByID(page.ID)
	if !ok || got.State.Publish != cmsstore.PublishStatePublished || got.Body.Blocks[0].Values["text"].String != "Draft one" {
		t.Fatalf("page after reopen: %+v", got)
	}
	if list := reopened.ListRevisions(cmsstore.RevisionFilter{ResourceKind: cmsstore.ResourceKindPage, ResourceID: page.ID}); len(list) < 2 {
		t.Fatalf("page revisions after reopen = %d", len(list))
	}
	posts, _ := reopened.ListPosts(cmsstore.PostFilter{})
	if len(posts) != 1 || posts[0].Tags[0] != "a" {
		t.Fatalf("posts after reopen: %+v", posts)
	}
	// A restore appends a revision and changes the page; the rows follow.
	list := reopened.ListRevisions(cmsstore.RevisionFilter{ResourceKind: cmsstore.ResourceKindPage, ResourceID: page.ID})
	if _, _, err := reopened.RestorePageRevision(page.ID, list[len(list)-1].ID); err != nil {
		t.Fatal(err)
	}
	after, _ := reopened.countRows(ctx, "revisions")
	if after != revisions+1 {
		t.Fatalf("revision rows after restore = %d, want %d", after, revisions+1)
	}
}

func TestSavesAppendOnlyTheNewRevision(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "site.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	page, _ := store.CreatePage(cmsstore.PageInput{Slug: "a", Title: "A", Body: body("x")})
	for i := 0; i < 50; i++ {
		if _, _, err := store.PreviewPage(page.ID, cmsstore.PageInput{Slug: "a", Title: "A", Body: body("edit"), State: page.State}); err != nil {
			t.Fatal(err)
		}
	}
	n, _ := store.countRows(context.Background(), "revisions")
	if n != 50 {
		t.Fatalf("revision rows = %d, want 50", n)
	}
}

func TestNewFromSeedMigratesAJSONSite(t *testing.T) {
	seed := memory.Seed{
		Settings: cmsstore.SiteSettings{Title: "Moved"}, HasSettings: true,
		Pages: []cmsstore.Page{{ID: "page_1", Slug: "home", Title: "Home", Body: body("moved")}},
	}
	path := filepath.Join(t.TempDir(), "site.db")
	store, err := New(path, seed)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	page, ok, _ := reopened.PageBySlug("home")
	if !ok || page.ID != "page_1" || page.Body.Blocks[0].Values["text"].String != "moved" {
		t.Fatalf("migrated page: %+v %v", page, ok)
	}
}

func TestBackupIsAConsistentDatabase(t *testing.T) {
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "site.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	store.CreatePage(cmsstore.PageInput{Slug: "home", Title: "Home", Body: body("keep")})
	var buf bytes.Buffer
	if err := store.Backup(&buf); err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(dir, "copy.db")
	if err := os.WriteFile(copyPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if page, ok, _ := restored.PageBySlug("home"); !ok || page.Title != "Home" {
		t.Fatal("the backup does not open as the same site")
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "backup-") {
			t.Fatalf("temp file left behind: %s", entry.Name())
		}
	}
}
