package file

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestOpenAcceptsValidSnapshotWithTrailingWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	data, err := json.Marshal(Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte(" \n\t")...), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("expected valid snapshot with trailing whitespace to open: %v", err)
	}
	got := store.Snapshot()
	if got.HasSettings || len(got.Pages) != 0 || len(got.Posts) != 0 || len(got.Revisions) != 0 {
		t.Fatalf("expected empty seed, got %#v", got)
	}
}

func TestOpenRejectsEmptyMalformedAndTrailingSnapshotData(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "empty", data: "", want: "snapshot is empty"},
		{name: "whitespace", data: " \n\t", want: "snapshot is empty"},
		{name: "null", data: "null", want: "snapshot must be a JSON object"},
		{name: "partial", data: `{"pages":[`, want: "decode file store snapshot"},
		{name: "multiple", data: "{}\n{}", want: "multiple JSON values"},
		{name: "trailing", data: "{}\nnot-json", want: "trailing data"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "snapshot.json")
			if err := os.WriteFile(path, []byte(test.data), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Open(path)
			if err == nil {
				t.Fatal("expected existing invalid snapshot to be rejected")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
			if test.name == "empty" || test.name == "whitespace" {
				if !errors.Is(err, io.EOF) {
					t.Fatalf("expected empty snapshot error to wrap io.EOF, got %v", err)
				}
			}
		})
	}
}

func TestMutatorsReconcileMemoryAfterPersistenceFailure(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Store) error
	}{
		{
			name: "create page",
			mutate: func(store *Store) error {
				_, err := store.CreatePage(cmsstore.PageInput{Title: "New Page"})
				return err
			},
		},
		{
			name: "update page",
			mutate: func(store *Store) error {
				_, err := store.UpdatePage("page_existing", cmsstore.PageInput{Title: "Updated Page"})
				return err
			},
		},
		{
			name: "create post",
			mutate: func(store *Store) error {
				_, err := store.CreatePost(cmsstore.PostInput{Title: "New Post"})
				return err
			},
		},
		{
			name: "update post",
			mutate: func(store *Store) error {
				_, err := store.UpdatePost("post_existing", cmsstore.PostInput{Title: "Updated Post"})
				return err
			},
		},
		{
			name: "save settings",
			mutate: func(store *Store) error {
				_, err := store.SaveSiteSettings(cmsstore.SiteSettingsInput{Title: "Updated Site"})
				return err
			},
		},
		{
			name: "save revision",
			mutate: func(store *Store) error {
				_, err := store.SaveRevision(lifecycle.RevisionInput{
					ResourceKind: cmsstore.ResourceKindPage,
					ResourceID:   "page_existing",
					Action:       "page.updated",
					Snapshot:     map[string]string{"title": "Updated Page"},
				})
				return err
			},
		},
	}

	for _, failure := range []struct {
		name string
		err  error
	}{
		{name: "write failure", err: errors.New("injected write failure")},
		{name: "rename failure", err: errors.New("injected rename failure")},
	} {
		failure := failure
		t.Run(failure.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "snapshot.json")
			seed := fileStoreTestSeed()
			if _, err := New(path, seed); err != nil {
				t.Fatal(err)
			}
			store, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			want := store.Snapshot()
			store.saveFn = func(string, memory.Seed) error {
				return fmt.Errorf("persist snapshot: %w", failure.err)
			}

			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					if err := test.mutate(store); err == nil {
						t.Fatal("expected persistence failure")
					} else if !errors.Is(err, failure.err) {
						t.Fatalf("expected wrapped %v, got %v", failure.err, err)
					}
					if got := store.Snapshot(); !reflect.DeepEqual(got, want) {
						t.Fatalf("memory diverged after persistence failure: got %#v want %#v", got, want)
					}
					reopened, err := Open(path)
					if err != nil {
						t.Fatalf("persisted snapshot became unreadable: %v", err)
					}
					if got := reopened.Snapshot(); !reflect.DeepEqual(got, want) {
						t.Fatalf("disk state changed after persistence failure: got %#v want %#v", got, want)
					}
				})
			}
		})
	}
}

func TestPostRenameDurabilityFailureReconcilesToCommittedSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	store, err := New(path, memory.Seed{})
	if err != nil {
		t.Fatal(err)
	}
	durabilityErr := errors.New("injected directory sync failure")
	store.syncDirFn = func(string) error {
		return durabilityErr
	}

	page, err := store.CreatePage(cmsstore.PageInput{Title: "Committed Page"})
	if err == nil {
		t.Fatal("expected post-rename durability error")
	}
	var committedErr *CommittedSnapshotError
	if !errors.As(err, &committedErr) {
		t.Fatalf("expected committed snapshot error, got %v", err)
	}
	if !errors.Is(err, durabilityErr) {
		t.Fatalf("expected wrapped durability error, got %v", err)
	}
	if !strings.Contains(err.Error(), "committed before durability error") {
		t.Fatalf("expected actionable committed-state error, got %v", err)
	}
	if page.ID == "" {
		t.Fatal("expected mutation result to retain the created page")
	}

	got, ok, err := store.PageByID(page.ID)
	if err != nil || !ok {
		t.Fatalf("expected committed page in memory, ok=%v err=%v", ok, err)
	}
	if got.Title != "Committed Page" {
		t.Fatalf("expected committed page title, got %#v", got)
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("expected renamed snapshot to remain readable: %v", err)
	}
	if got, ok, err := reopened.PageByID(page.ID); err != nil || !ok || got.Title != "Committed Page" {
		t.Fatalf("expected committed page after reopen, page=%#v ok=%v err=%v", got, ok, err)
	}
}

func TestPersistenceRecoveryFailureFallsBackAndWrapsErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if _, err := New(path, fileStoreTestSeed()); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	want := store.Snapshot()
	persistErr := errors.New("injected persistence failure")
	store.saveFn = func(path string, _ memory.Seed) error {
		if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
			return fmt.Errorf("corrupt snapshot for recovery test: %w", err)
		}
		return fmt.Errorf("injected persist failure: %w", persistErr)
	}

	if _, err := store.CreatePage(cmsstore.PageInput{Title: "Unpersisted Page"}); err == nil {
		t.Fatal("expected persistence failure")
	} else {
		if !errors.Is(err, persistErr) {
			t.Fatalf("expected persistence error to remain wrapped, got %v", err)
		}
		if !strings.Contains(err.Error(), "recover file store memory from disk") {
			t.Fatalf("expected recovery failure context, got %v", err)
		}
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("expected recovery decode error to remain wrapped, got %v", err)
		}
	}
	if got := store.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected pre-mutation memory fallback, got %#v want %#v", got, want)
	}
}

func TestRenameFailureCleansTempAndPreservesPriorSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	if _, err := New(path, fileStoreTestSeed()); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	want := store.Snapshot()
	renameTarget := filepath.Join(dir, "rename-target")
	if err := os.Mkdir(renameTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	// Point the live adapter at a directory so the real save path writes its
	// temp file, then fails at os.Rename without touching the valid snapshot.
	store.path = renameTarget
	if _, err := store.CreatePage(cmsstore.PageInput{Title: "Rename Failure"}); err == nil {
		t.Fatal("expected filesystem rename failure")
	} else {
		if !strings.Contains(err.Error(), "replace file store snapshot") {
			t.Fatalf("expected actionable rename error, got %v", err)
		}
		var renameErr *os.LinkError
		if !errors.As(err, &renameErr) || renameErr.Op != "rename" {
			t.Fatalf("expected underlying rename error, got %v", err)
		}
	}
	if got := store.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected memory rollback after rename failure, got %#v want %#v", got, want)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	tempPrefix := "." + filepath.Base(renameTarget) + ".tmp-"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), tempPrefix) {
			t.Fatalf("rename failure left temp snapshot %q", entry.Name())
		}
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("expected untouched snapshot to reopen: %v", err)
	}
	if got := reopened.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected untouched snapshot after rename failure, got %#v want %#v", got, want)
	}
}

func TestPersistenceFailurePreservesClockAndIDContinuity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	store, err := New(path, fileStoreTestSeed(), WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatal(err)
	}
	persistErr := errors.New("injected persistence failure")
	store.saveFn = func(string, memory.Seed) error {
		return fmt.Errorf("injected persist failure: %w", persistErr)
	}
	if _, err := store.CreatePage(cmsstore.PageInput{Title: "Failed Page"}); !errors.Is(err, persistErr) {
		t.Fatalf("expected failed page persistence error, got %v", err)
	}
	store.saveFn = store.saveSnapshot
	page, err := store.CreatePage(cmsstore.PageInput{Title: "Next Page"})
	if err != nil {
		t.Fatal(err)
	}
	if page.ID != "page_2" || !page.Created.Equal(now) || !page.Updated.Equal(now) {
		t.Fatalf("expected deterministic page id/time after rollback, got %#v", page)
	}

	store.saveFn = func(string, memory.Seed) error {
		return fmt.Errorf("injected persist failure: %w", persistErr)
	}
	if _, err := store.SaveRevision(lifecycle.RevisionInput{
		ResourceKind: cmsstore.ResourceKindPage,
		ResourceID:   page.ID,
		Action:       "page.updated",
		Snapshot:     page,
	}); !errors.Is(err, persistErr) {
		t.Fatalf("expected failed revision persistence error, got %v", err)
	}
	store.saveFn = store.saveSnapshot
	revision, err := store.SaveRevision(lifecycle.RevisionInput{
		ResourceKind: cmsstore.ResourceKindPage,
		ResourceID:   page.ID,
		Action:       "page.updated",
		Snapshot:     page,
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.ID != "rev_2" || !revision.Created.Equal(now) {
		t.Fatalf("expected deterministic revision id/time after rollback, got %#v", revision)
	}
}

func TestReadWaitsForPersistenceRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	if _, err := New(path, fileStoreTestSeed()); err != nil {
		t.Fatal(err)
	}
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	want := store.Snapshot()
	persistStarted := make(chan struct{})
	releasePersist := make(chan struct{})
	persistErr := errors.New("injected persistence failure")
	store.saveFn = func(string, memory.Seed) error {
		close(persistStarted)
		<-releasePersist
		return fmt.Errorf("injected persist failure: %w", persistErr)
	}

	mutationDone := make(chan error, 1)
	go func() {
		_, err := store.CreatePage(cmsstore.PageInput{Title: "Transient Page"})
		mutationDone <- err
	}()
	<-persistStarted

	readStarted := make(chan struct{})
	readDone := make(chan memory.Seed, 1)
	go func() {
		close(readStarted)
		readDone <- store.Snapshot()
	}()
	<-readStarted
	select {
	case got := <-readDone:
		t.Fatalf("read completed before persistence recovery: %#v", got)
	default:
	}

	close(releasePersist)
	if err := <-mutationDone; !errors.Is(err, persistErr) {
		t.Fatalf("expected wrapped persistence failure, got %v", err)
	}
	select {
	case got := <-readDone:
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected read after recovery to see prior state, got %#v want %#v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("read did not complete after persistence recovery")
	}
}

func fileStoreTestSeed() memory.Seed {
	now := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	return memory.Seed{
		Settings: cmsstore.SiteSettings{
			ID:      "site",
			Title:   "Existing Site",
			State:   cmsstore.State{Publish: cmsstore.PublishStateDraft},
			Updated: now,
		},
		HasSettings: true,
		Pages: []cmsstore.Page{{
			ID:      "page_existing",
			Slug:    "existing-page",
			Title:   "Existing Page",
			State:   cmsstore.State{Publish: cmsstore.PublishStateDraft},
			Created: now,
			Updated: now,
		}},
		Posts: []cmsstore.Post{{
			ID:      "post_existing",
			Slug:    "existing-post",
			Title:   "Existing Post",
			State:   cmsstore.State{Publish: cmsstore.PublishStateDraft},
			Created: now,
			Updated: now,
		}},
		Revisions: []lifecycle.Revision{{
			ID:           "revision_existing",
			ResourceKind: cmsstore.ResourceKindPage,
			ResourceID:   "page_existing",
			Action:       "page.saved",
			Snapshot:     json.RawMessage(`{"title":"Existing Page"}`),
			Created:      now,
		}},
	}
}
