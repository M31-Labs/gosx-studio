package file

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"

	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/cms/store/memory"
)

// TestLifecycleStoreContract locks the durable-publish gap closed. Before this,
// cms/store/file was the only durable content store Studio shipped and it
// satisfied cmsstore.Store but NOT cmsstore.LifecycleStore, so every host had
// to write its own preview/publish/restore path. The wrapped *memory.Store
// already implements all nine methods; the file store simply did not re-export
// them.
func TestLifecycleStoreContract(t *testing.T) {
	var _ cmsstore.LifecycleStore = (*Store)(nil)
	var _ cmsstore.PageLifecycleStore = (*Store)(nil)
	var _ cmsstore.PostLifecycleStore = (*Store)(nil)
	var _ cmsstore.SiteSettingsLifecycleStore = (*Store)(nil)
}

func newLifecycleTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "cms.json")
	store, err := New(path, memory.Seed{}, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	return store, path
}

// TestPagePublishSurvivesReopen is the behavior that matters: a publish must
// still be published after a restart, not only inside the process that made it.
func TestPagePublishSurvivesReopen(t *testing.T) {
	store, path := newLifecycleTestStore(t)

	page, err := store.CreatePage(cmsstore.PageInput{Title: "About", Slug: "about"})
	if err != nil {
		t.Fatal(err)
	}

	preview, revision, err := store.PreviewPage(page.ID, cmsstore.PageInput{Title: "About us", Slug: "about"})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Title != "About us" {
		t.Fatalf("preview title = %q, want %q", preview.Title, "About us")
	}
	if revision.ID == "" {
		t.Fatal("preview must record a revision")
	}

	published, publishRevision, err := store.PublishPage(page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if published.State.PublishedAt == nil {
		t.Fatal("PublishPage must set PublishedAt")
	}
	if publishRevision.ID == "" {
		t.Fatal("publish must record a revision")
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := reopened.PageByID(page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("page missing after reopen")
	}
	if got.Title != "About us" {
		t.Fatalf("reopened title = %q, want %q", got.Title, "About us")
	}
	if got.State.PublishedAt == nil {
		t.Fatal("publish did not survive reopen: PublishedAt is nil")
	}
	if len(reopened.ListRevisions(lifecycle.Filter{})) < 2 {
		t.Fatalf("expected preview and publish revisions to persist, got %d", len(reopened.ListRevisions(lifecycle.Filter{})))
	}
}

func TestPageRestoreRevisionSurvivesReopen(t *testing.T) {
	store, path := newLifecycleTestStore(t)

	page, err := store.CreatePage(cmsstore.PageInput{Title: "V1", Slug: "v"})
	if err != nil {
		t.Fatal(err)
	}
	_, first, err := store.PreviewPage(page.ID, cmsstore.PageInput{Title: "V1", Slug: "v"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PreviewPage(page.ID, cmsstore.PageInput{Title: "V2", Slug: "v"}); err != nil {
		t.Fatal(err)
	}

	restored, _, err := store.RestorePageRevision(page.ID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Title != "V1" {
		t.Fatalf("restored title = %q, want %q", restored.Title, "V1")
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := reopened.PageByID(page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("page missing after reopen")
	}
	if got.Title != "V1" {
		t.Fatalf("restore did not survive reopen: title = %q, want %q", got.Title, "V1")
	}
}

func TestPostAndSiteSettingsLifecycleSurviveReopen(t *testing.T) {
	store, path := newLifecycleTestStore(t)

	post, err := store.CreatePost(cmsstore.PostInput{Title: "Hello", Slug: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PreviewPost(post.ID, cmsstore.PostInput{Title: "Hello world", Slug: "hello"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PublishPost(post.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := store.SaveSiteSettings(cmsstore.SiteSettingsInput{Title: "Site"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PreviewSiteSettings(cmsstore.SiteSettingsInput{Title: "Better site"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.PublishSiteSettings(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	gotPost, ok, err := reopened.PostByID(post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("post missing after reopen")
	}
	if gotPost.Title != "Hello world" {
		t.Fatalf("reopened post title = %q, want %q", gotPost.Title, "Hello world")
	}
	if gotPost.State.PublishedAt == nil {
		t.Fatal("post publish did not survive reopen")
	}

	settings, ok, err := reopened.SiteSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("site settings missing after reopen")
	}
	if settings.Title != "Better site" {
		t.Fatalf("reopened settings title = %q, want %q", settings.Title, "Better site")
	}
}

// TestLifecycleMutationFailureRecoversMemory proves the lifecycle methods use
// the same persist-or-recover discipline as the CRUD methods: a failed write to
// disk must not leave the in-memory store ahead of the snapshot on disk.
func TestLifecycleMutationFailureRecoversMemory(t *testing.T) {
	store, _ := newLifecycleTestStore(t)

	page, err := store.CreatePage(cmsstore.PageInput{Title: "Stable", Slug: "stable"})
	if err != nil {
		t.Fatal(err)
	}

	persistErr := errors.New("injected persistence failure")
	store.saveFn = func(string, memory.Seed) error { return persistErr }

	if _, _, err := store.PreviewPage(page.ID, cmsstore.PageInput{Title: "Doomed", Slug: "stable"}); err == nil {
		t.Fatal("expected PreviewPage to surface the persistence failure")
	}

	got, ok, err := store.PageByID(page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("page missing after failed preview")
	}
	if got.Title != "Stable" {
		t.Fatalf("memory not recovered after failed persist: title = %q, want %q", got.Title, "Stable")
	}
}
