package memory

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

type Clock func() time.Time

type Seed struct {
	Settings    cmsstore.SiteSettings
	HasSettings bool
	Pages       []cmsstore.Page
	Posts       []cmsstore.Post
	Revisions   []lifecycle.Revision
}

type Option func(*Store)

type Store struct {
	mu          sync.RWMutex
	now         Clock
	settings    cmsstore.SiteSettings
	hasSettings bool
	pages       []cmsstore.Page
	posts       []cmsstore.Post
	revisions   []lifecycle.Revision
	nextPageID  int
	nextPostID  int
	nextRevID   int
}

var _ cmsstore.Store = (*Store)(nil)

func New(seed Seed, options ...Option) *Store {
	store := &Store{
		now:         time.Now,
		settings:    cmsstore.CloneSiteSettings(seed.Settings),
		hasSettings: seed.HasSettings || strings.TrimSpace(seed.Settings.Title) != "" || strings.TrimSpace(seed.Settings.ID) != "",
		pages:       cmsstore.ClonePages(seed.Pages),
		posts:       cmsstore.ClonePosts(seed.Posts),
		revisions:   lifecycle.CloneRevisions(seed.Revisions),
	}
	store.nextPageID = len(store.pages) + 1
	store.nextPostID = len(store.posts) + 1
	store.nextRevID = len(store.revisions) + 1
	for _, option := range options {
		if option != nil {
			option(store)
		}
	}
	if store.now == nil {
		store.now = time.Now
	}
	return store
}

func WithClock(clock Clock) Option {
	return func(store *Store) {
		store.now = clock
	}
}

func (s *Store) Snapshot() Seed {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Seed{
		Settings:    cmsstore.CloneSiteSettings(s.settings),
		HasSettings: s.hasSettings,
		Pages:       cmsstore.ClonePages(s.pages),
		Posts:       cmsstore.ClonePosts(s.posts),
		Revisions:   lifecycle.CloneRevisions(s.revisions),
	}
}

func (s *Store) ListPages(filter cmsstore.PageFilter) ([]cmsstore.Page, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]cmsstore.Page, 0, len(s.pages))
	slug := cmsstore.NormalizeSlug(filter.Slug)
	for _, page := range s.pages {
		if slug != "" && page.Slug != slug {
			continue
		}
		if filter.Publish != "" && page.State.Publish != filter.Publish {
			continue
		}
		out = append(out, cmsstore.ClonePage(page))
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *Store) PageByID(id string) (cmsstore.Page, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id = strings.TrimSpace(id)
	for _, page := range s.pages {
		if page.ID == id {
			return cmsstore.ClonePage(page), true, nil
		}
	}
	return cmsstore.Page{}, false, nil
}

func (s *Store) PageBySlug(slug string) (cmsstore.Page, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slug = cmsstore.NormalizeSlug(slug)
	for _, page := range s.pages {
		if page.Slug == slug {
			return cmsstore.ClonePage(page), true, nil
		}
	}
	return cmsstore.Page{}, false, nil
}

func (s *Store) CreatePage(input cmsstore.PageInput) (cmsstore.Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input = normalizePageInput(input)
	if err := validatePageInput(input); err != nil {
		return cmsstore.Page{}, err
	}
	now := s.now()
	page := cmsstore.NormalizePage(input, cmsstore.Page{ID: s.nextID("page")}, now)
	s.pages = append(s.pages, page)
	return cmsstore.ClonePage(page), nil
}

func (s *Store) UpdatePage(id string, input cmsstore.PageInput) (cmsstore.Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input = normalizePageInput(input)
	if err := validatePageInput(input); err != nil {
		return cmsstore.Page{}, err
	}
	id = strings.TrimSpace(id)
	for index, page := range s.pages {
		if page.ID == id {
			next := cmsstore.NormalizePage(input, page, s.now())
			s.pages[index] = next
			return cmsstore.ClonePage(next), nil
		}
	}
	return cmsstore.Page{}, fmt.Errorf("page %q not found", id)
}

func (s *Store) ListPosts(filter cmsstore.PostFilter) ([]cmsstore.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]cmsstore.Post, 0, len(s.posts))
	slug := cmsstore.NormalizeSlug(filter.Slug)
	tag := strings.ToLower(strings.TrimSpace(filter.Tag))
	for _, post := range s.posts {
		if slug != "" && post.Slug != slug {
			continue
		}
		if tag != "" && !hasTag(post.Tags, tag) {
			continue
		}
		if filter.Publish != "" && post.State.Publish != filter.Publish {
			continue
		}
		out = append(out, cmsstore.ClonePost(post))
		if filter.Limit > 0 && len(out) >= filter.Limit {
			break
		}
	}
	return out, nil
}

func (s *Store) PostByID(id string) (cmsstore.Post, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id = strings.TrimSpace(id)
	for _, post := range s.posts {
		if post.ID == id {
			return cmsstore.ClonePost(post), true, nil
		}
	}
	return cmsstore.Post{}, false, nil
}

func (s *Store) PostBySlug(slug string) (cmsstore.Post, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	slug = cmsstore.NormalizeSlug(slug)
	for _, post := range s.posts {
		if post.Slug == slug {
			return cmsstore.ClonePost(post), true, nil
		}
	}
	return cmsstore.Post{}, false, nil
}

func (s *Store) CreatePost(input cmsstore.PostInput) (cmsstore.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input = normalizePostInput(input)
	if err := validatePostInput(input); err != nil {
		return cmsstore.Post{}, err
	}
	now := s.now()
	post := cmsstore.NormalizePost(input, cmsstore.Post{ID: s.nextID("post")}, now)
	s.posts = append(s.posts, post)
	return cmsstore.ClonePost(post), nil
}

func (s *Store) UpdatePost(id string, input cmsstore.PostInput) (cmsstore.Post, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	input = normalizePostInput(input)
	if err := validatePostInput(input); err != nil {
		return cmsstore.Post{}, err
	}
	id = strings.TrimSpace(id)
	for index, post := range s.posts {
		if post.ID == id {
			next := cmsstore.NormalizePost(input, post, s.now())
			s.posts[index] = next
			return cmsstore.ClonePost(next), nil
		}
	}
	return cmsstore.Post{}, fmt.Errorf("post %q not found", id)
}

func (s *Store) SiteSettings() (cmsstore.SiteSettings, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cmsstore.CloneSiteSettings(s.settings), s.hasSettings, nil
}

func (s *Store) SaveSiteSettings(input cmsstore.SiteSettingsInput) (cmsstore.SiteSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(input.Title) == "" {
		return cmsstore.SiteSettings{}, fmt.Errorf("site settings title is required")
	}
	s.settings = cmsstore.NormalizeSiteSettings(input, s.settings, s.now())
	if strings.TrimSpace(s.settings.ID) == "" {
		s.settings.ID = "site"
	}
	s.hasSettings = true
	return cmsstore.CloneSiteSettings(s.settings), nil
}

func (s *Store) ListRevisions(filter lifecycle.RevisionFilter) []lifecycle.Revision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return lifecycle.FilterRevisions(s.revisions, filter)
}

func (s *Store) RevisionByID(resourceKind, resourceID, revisionID string) (lifecycle.Revision, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return lifecycle.FindRevision(s.revisions, resourceKind, resourceID, revisionID)
}

func (s *Store) SaveRevision(input lifecycle.RevisionInput) (lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(input.ID) == "" {
		input.ID = s.nextID("rev")
	}
	revision, err := lifecycle.NewRevision(input, s.now())
	if err != nil {
		return lifecycle.Revision{}, err
	}
	s.revisions = append(s.revisions, revision)
	return lifecycle.CloneRevision(revision), nil
}

func (s *Store) nextID(prefix string) string {
	switch prefix {
	case "page":
		id := fmt.Sprintf("page_%d", s.nextPageID)
		s.nextPageID++
		return id
	case "post":
		id := fmt.Sprintf("post_%d", s.nextPostID)
		s.nextPostID++
		return id
	default:
		id := fmt.Sprintf("%s_%d", prefix, s.nextRevID)
		s.nextRevID++
		return id
	}
}

func normalizePageInput(input cmsstore.PageInput) cmsstore.PageInput {
	if strings.TrimSpace(input.Slug) == "" {
		input.Slug = input.Title
	}
	return input
}

func normalizePostInput(input cmsstore.PostInput) cmsstore.PostInput {
	if strings.TrimSpace(input.Slug) == "" {
		input.Slug = input.Title
	}
	return input
}

func validatePageInput(input cmsstore.PageInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("page title is required")
	}
	if cmsstore.NormalizeSlug(input.Slug) == "" {
		return fmt.Errorf("page slug is required")
	}
	return nil
}

func validatePostInput(input cmsstore.PostInput) error {
	if strings.TrimSpace(input.Title) == "" {
		return fmt.Errorf("post title is required")
	}
	if cmsstore.NormalizeSlug(input.Slug) == "" {
		return fmt.Errorf("post slug is required")
	}
	return nil
}

func hasTag(tags []string, tag string) bool {
	for _, candidate := range tags {
		if strings.ToLower(strings.TrimSpace(candidate)) == tag {
			return true
		}
	}
	return false
}
