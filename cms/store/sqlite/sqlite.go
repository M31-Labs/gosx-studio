// Package sqlite stores a site in one SQLite file.
//
// It keeps the same in-memory working set as the file store — reads are
// answered from memory and every rule about drafts, publishing, and
// revisions lives in cms/store/memory — but instead of rewriting a whole
// JSON snapshot after each change it writes only the rows that changed, in
// one transaction. Revisions are appended, never rewritten, so a site with a
// long history costs the same per save as a new one. The file is plain
// SQLite: any tool can read it, and a backup is a consistent copy made with
// VACUUM INTO.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
	"m31labs.dev/gosx-studio/cms/store/memory"
)

const schema = `
CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS settings (id INTEGER PRIMARY KEY CHECK (id = 1), data TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS pages (seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT NOT NULL UNIQUE, slug TEXT NOT NULL, data TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS posts (seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT NOT NULL UNIQUE, slug TEXT NOT NULL, data TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS revisions (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	id TEXT NOT NULL UNIQUE,
	resource_kind TEXT NOT NULL,
	resource_id TEXT NOT NULL,
	action TEXT NOT NULL,
	created TEXT NOT NULL,
	data TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS revisions_resource ON revisions (resource_kind, resource_id, seq);
INSERT OR IGNORE INTO meta (key, value) VALUES ('schema', '1');
`

// Store is a site in a SQLite file.
type Store struct {
	mu            sync.RWMutex
	path          string
	db            *sql.DB
	store         *memory.Store
	memoryOptions []memory.Option
}

var _ cmsstore.Store = (*Store)(nil)
var _ cmsstore.LifecycleStore = (*Store)(nil)

// Option configures a store.
type Option func(*Store)

// WithClock sets the clock the in-memory rules use.
func WithClock(clock memory.Clock) Option {
	return func(s *Store) { s.memoryOptions = append(s.memoryOptions, memory.WithClock(clock)) }
}

func openDB(path string) (*sql.DB, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("sqlite: a file path is required")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// One connection: the working set is in memory and every write is one
	// transaction, so more connections would only contend for the lock.
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{"PRAGMA journal_mode=WAL", "PRAGMA synchronous=NORMAL", "PRAGMA busy_timeout=5000", "PRAGMA foreign_keys=ON"} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, err
		}
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// Open loads the site in path, creating an empty one if the file is new.
func Open(path string, opts ...Option) (*Store, error) {
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	s := &Store{path: path, db: db}
	for _, opt := range opts {
		opt(s)
	}
	seed, err := s.load()
	if err != nil {
		db.Close()
		return nil, err
	}
	s.store = memory.New(seed, s.memoryOptions...)
	return s, nil
}

// New creates the file from a seed, replacing whatever it held. It is how a
// JSON site is migrated.
func New(path string, seed memory.Seed, opts ...Option) (*Store, error) {
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	s := &Store{path: path, db: db}
	for _, opt := range opts {
		opt(s)
	}
	s.store = memory.New(seed, s.memoryOptions...)
	if err := s.writeAll(s.store.Snapshot()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Path is the file the store lives in.
func (s *Store) Path() string { return s.path }

// Close releases the file.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// Snapshot is the whole site as a seed.
func (s *Store) Snapshot() memory.Seed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.Snapshot()
}

// Backup writes a consistent copy of the database to w.
func (s *Store) Backup(w io.Writer) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dir := filepath.Dir(s.path)
	if s.path == ":memory:" {
		dir = os.TempDir()
	}
	temp, err := os.CreateTemp(dir, "backup-*.db")
	if err != nil {
		return err
	}
	name := temp.Name()
	temp.Close()
	os.Remove(name)
	defer os.Remove(name)
	if _, err := s.db.Exec("VACUUM INTO ?", name); err != nil {
		return err
	}
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(w, file)
	return err
}

// ---------- rows in, rows out ----------

func (s *Store) load() (memory.Seed, error) {
	var seed memory.Seed
	var raw string
	switch err := s.db.QueryRow("SELECT data FROM settings WHERE id = 1").Scan(&raw); {
	case err == nil:
		if err := json.Unmarshal([]byte(raw), &seed.Settings); err != nil {
			return seed, fmt.Errorf("sqlite: settings row: %w", err)
		}
		seed.HasSettings = true
	case errors.Is(err, sql.ErrNoRows):
	default:
		return seed, err
	}
	rows, err := s.db.Query("SELECT data FROM pages ORDER BY seq")
	if err != nil {
		return seed, err
	}
	for rows.Next() {
		var page cmsstore.Page
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return seed, err
		}
		if err := json.Unmarshal([]byte(raw), &page); err != nil {
			rows.Close()
			return seed, fmt.Errorf("sqlite: page row: %w", err)
		}
		seed.Pages = append(seed.Pages, page)
	}
	rows.Close()
	rows, err = s.db.Query("SELECT data FROM posts ORDER BY seq")
	if err != nil {
		return seed, err
	}
	for rows.Next() {
		var post cmsstore.Post
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return seed, err
		}
		if err := json.Unmarshal([]byte(raw), &post); err != nil {
			rows.Close()
			return seed, fmt.Errorf("sqlite: post row: %w", err)
		}
		seed.Posts = append(seed.Posts, post)
	}
	rows.Close()
	rows, err = s.db.Query("SELECT data FROM revisions ORDER BY seq")
	if err != nil {
		return seed, err
	}
	for rows.Next() {
		var revision lifecycle.Revision
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return seed, err
		}
		if err := json.Unmarshal([]byte(raw), &revision); err != nil {
			rows.Close()
			return seed, fmt.Errorf("sqlite: revision row: %w", err)
		}
		seed.Revisions = append(seed.Revisions, revision)
	}
	rows.Close()
	return seed, rows.Err()
}

func encode(value any) (string, error) {
	raw, err := json.Marshal(value)
	return string(raw), err
}

func upsertPage(tx *sql.Tx, page cmsstore.Page) error {
	raw, err := encode(page)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO pages (id, slug, data) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, data = excluded.data", page.ID, page.Slug, raw)
	return err
}

func upsertPost(tx *sql.Tx, post cmsstore.Post) error {
	raw, err := encode(post)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO posts (id, slug, data) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, data = excluded.data", post.ID, post.Slug, raw)
	return err
}

func upsertSettings(tx *sql.Tx, settings cmsstore.SiteSettings) error {
	raw, err := encode(settings)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO settings (id, data) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data", raw)
	return err
}

func upsertRevision(tx *sql.Tx, revision lifecycle.Revision) error {
	raw, err := encode(revision)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO revisions (id, resource_kind, resource_id, action, created, data) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data",
		revision.ID, revision.ResourceKind, revision.ResourceID, revision.Action, revision.Created.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"), raw)
	return err
}

// writeAll replaces every row with the seed. Used when creating a file.
func (s *Store) writeAll(seed memory.Seed) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"pages", "posts", "revisions", "settings"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return err
		}
	}
	if seed.HasSettings || strings.TrimSpace(seed.Settings.Title) != "" {
		if err := upsertSettings(tx, seed.Settings); err != nil {
			return err
		}
	}
	for _, page := range seed.Pages {
		if err := upsertPage(tx, page); err != nil {
			return err
		}
	}
	for _, post := range seed.Posts {
		if err := upsertPost(tx, post); err != nil {
			return err
		}
	}
	for _, revision := range seed.Revisions {
		if err := upsertRevision(tx, revision); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// writeChanges persists what one mutation changed: settings if different,
// pages and posts whose JSON differs, and revisions from the last one the
// previous snapshot had onward (a publish rewrites the revision it just
// appended, so the last known one is checked too).
func (s *Store) writeChanges(previous, current memory.Seed) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if current.HasSettings {
		before, _ := encode(previous.Settings)
		after, _ := encode(current.Settings)
		if !previous.HasSettings || before != after {
			if err := upsertSettings(tx, current.Settings); err != nil {
				return err
			}
		}
	}
	knownPages := map[string]string{}
	for _, page := range previous.Pages {
		raw, _ := encode(page)
		knownPages[page.ID] = raw
	}
	for _, page := range current.Pages {
		raw, err := encode(page)
		if err != nil {
			return err
		}
		if knownPages[page.ID] != raw {
			if _, err := tx.Exec("INSERT INTO pages (id, slug, data) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, data = excluded.data", page.ID, page.Slug, raw); err != nil {
				return err
			}
		}
	}
	knownPosts := map[string]string{}
	for _, post := range previous.Posts {
		raw, _ := encode(post)
		knownPosts[post.ID] = raw
	}
	for _, post := range current.Posts {
		raw, err := encode(post)
		if err != nil {
			return err
		}
		if knownPosts[post.ID] != raw {
			if _, err := tx.Exec("INSERT INTO posts (id, slug, data) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET slug = excluded.slug, data = excluded.data", post.ID, post.Slug, raw); err != nil {
				return err
			}
		}
	}
	from := len(previous.Revisions) - 1
	if from < 0 {
		from = 0
	}
	for index := from; index < len(current.Revisions); index++ {
		if err := upsertRevision(tx, current.Revisions[index]); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// mutate runs one change against the working set and persists it, rolling
// the working set back if the write fails.
func (s *Store) mutate(change func() error) error {
	previous := s.store.Snapshot()
	if err := change(); err != nil {
		return err
	}
	if err := s.writeChanges(previous, s.store.Snapshot()); err != nil {
		s.store = memory.New(previous, s.memoryOptions...)
		return err
	}
	return nil
}

// ---------- reads ----------

func (s *Store) ListPages(filter cmsstore.PageFilter) ([]cmsstore.Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.ListPages(filter)
}

func (s *Store) PageByID(id string) (cmsstore.Page, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.PageByID(id)
}

func (s *Store) PageBySlug(slug string) (cmsstore.Page, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.PageBySlug(slug)
}

func (s *Store) ListPosts(filter cmsstore.PostFilter) ([]cmsstore.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.ListPosts(filter)
}

func (s *Store) PostByID(id string) (cmsstore.Post, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.PostByID(id)
}

func (s *Store) PostBySlug(slug string) (cmsstore.Post, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.PostBySlug(slug)
}

func (s *Store) SiteSettings() (cmsstore.SiteSettings, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.SiteSettings()
}

func (s *Store) ListRevisions(filter lifecycle.RevisionFilter) []lifecycle.Revision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.ListRevisions(filter)
}

func (s *Store) RevisionByID(resourceKind, resourceID, revisionID string) (lifecycle.Revision, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.store.RevisionByID(resourceKind, resourceID, revisionID)
}

// ---------- writes ----------

func (s *Store) CreatePage(input cmsstore.PageInput) (page cmsstore.Page, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { page, e = s.store.CreatePage(input); return e })
	return page, err
}

func (s *Store) UpdatePage(id string, input cmsstore.PageInput) (page cmsstore.Page, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { page, e = s.store.UpdatePage(id, input); return e })
	return page, err
}

func (s *Store) CreatePost(input cmsstore.PostInput) (post cmsstore.Post, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { post, e = s.store.CreatePost(input); return e })
	return post, err
}

func (s *Store) UpdatePost(id string, input cmsstore.PostInput) (post cmsstore.Post, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { post, e = s.store.UpdatePost(id, input); return e })
	return post, err
}

func (s *Store) SaveSiteSettings(input cmsstore.SiteSettingsInput) (settings cmsstore.SiteSettings, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { settings, e = s.store.SaveSiteSettings(input); return e })
	return settings, err
}

func (s *Store) SaveRevision(input lifecycle.RevisionInput) (revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { revision, e = s.store.SaveRevision(input); return e })
	return revision, err
}

func (s *Store) PreviewPage(id string, input cmsstore.PageInput) (page cmsstore.Page, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { page, revision, e = s.store.PreviewPage(id, input); return e })
	return page, revision, err
}

func (s *Store) PublishPage(id string) (page cmsstore.Page, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { page, revision, e = s.store.PublishPage(id); return e })
	return page, revision, err
}

func (s *Store) RestorePageRevision(id, revisionID string) (page cmsstore.Page, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { page, revision, e = s.store.RestorePageRevision(id, revisionID); return e })
	return page, revision, err
}

func (s *Store) PreviewPost(id string, input cmsstore.PostInput) (post cmsstore.Post, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { post, revision, e = s.store.PreviewPost(id, input); return e })
	return post, revision, err
}

func (s *Store) PublishPost(id string) (post cmsstore.Post, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { post, revision, e = s.store.PublishPost(id); return e })
	return post, revision, err
}

func (s *Store) RestorePostRevision(id, revisionID string) (post cmsstore.Post, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { post, revision, e = s.store.RestorePostRevision(id, revisionID); return e })
	return post, revision, err
}

func (s *Store) PreviewSiteSettings(input cmsstore.SiteSettingsInput) (settings cmsstore.SiteSettings, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { settings, revision, e = s.store.PreviewSiteSettings(input); return e })
	return settings, revision, err
}

func (s *Store) PublishSiteSettings() (settings cmsstore.SiteSettings, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) { settings, revision, e = s.store.PublishSiteSettings(); return e })
	return settings, revision, err
}

func (s *Store) RestoreSiteSettingsRevision(revisionID string) (settings cmsstore.SiteSettings, revision lifecycle.Revision, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.mutate(func() (e error) {
		settings, revision, e = s.store.RestoreSiteSettingsRevision(revisionID)
		return e
	})
	return settings, revision, err
}

// countRows is a test aid: how many rows a table holds.
func (s *Store) countRows(ctx context.Context, table string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&n)
	return n, err
}
