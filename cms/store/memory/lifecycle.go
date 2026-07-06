package memory

import (
	"fmt"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

func (s *Store) PreviewPage(id string, input cmsstore.PageInput) (cmsstore.Page, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizePageInput(input)
	if err := validatePageInput(input); err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	now := s.now()
	input.State = previewState(input.State)
	page, index, err := s.upsertPageLocked(id, input, now)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	revision, err := s.appendRevisionLocked(cmsstore.ResourceKindPage, page.ID, page.Title, cmsstore.ActionPagePreviewSaved, "Saved page preview.", &page, now)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	page.State.RevisionID = revision.ID
	s.pages[index] = page
	revision.Snapshot, err = lifecycle.Snapshot(page)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = revision
	return cmsstore.ClonePage(page), lifecycle.CloneRevision(revision), nil
}

func (s *Store) PublishPage(id string) (cmsstore.Page, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.pageIndex(id)
	if index < 0 {
		return cmsstore.Page{}, lifecycle.Revision{}, fmt.Errorf("page %q not found", strings.TrimSpace(id))
	}
	now := s.now()
	page := s.pages[index]
	page.State = publishState(page.State, now)
	page.Updated = now
	revision, err := s.appendRevisionLocked(cmsstore.ResourceKindPage, page.ID, page.Title, cmsstore.ActionPagePublished, "Published page.", &page, now)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	page.State.RevisionID = revision.ID
	s.pages[index] = page
	revision.Snapshot, err = lifecycle.Snapshot(page)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = revision
	return cmsstore.ClonePage(page), lifecycle.CloneRevision(revision), nil
}

func (s *Store) RestorePageRevision(id, revisionID string) (cmsstore.Page, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.pageIndex(id)
	if index < 0 {
		return cmsstore.Page{}, lifecycle.Revision{}, fmt.Errorf("page %q not found", strings.TrimSpace(id))
	}
	revision, ok := lifecycle.FindRevision(s.revisions, cmsstore.ResourceKindPage, s.pages[index].ID, revisionID)
	if !ok {
		return cmsstore.Page{}, lifecycle.Revision{}, fmt.Errorf("page revision %q not found", strings.TrimSpace(revisionID))
	}
	page, err := lifecycle.DecodeSnapshot[cmsstore.Page](revision)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	now := s.now()
	page.ID = s.pages[index].ID
	page.State = rollbackState(page.State)
	page.Updated = now
	restore, err := s.appendRevisionLocked(cmsstore.ResourceKindPage, page.ID, page.Title, cmsstore.ActionPageRestored, "Restored page revision.", &page, now)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	page.State.RevisionID = restore.ID
	s.pages[index] = cmsstore.ClonePage(page)
	restore.Snapshot, err = lifecycle.Snapshot(page)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = restore
	return cmsstore.ClonePage(page), lifecycle.CloneRevision(restore), nil
}

func (s *Store) PreviewPost(id string, input cmsstore.PostInput) (cmsstore.Post, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	input = normalizePostInput(input)
	if err := validatePostInput(input); err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	now := s.now()
	input.State = previewState(input.State)
	post, index, err := s.upsertPostLocked(id, input, now)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	revision, err := s.appendRevisionLocked(cmsstore.ResourceKindPost, post.ID, post.Title, cmsstore.ActionPostPreviewSaved, "Saved post preview.", &post, now)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	post.State.RevisionID = revision.ID
	s.posts[index] = post
	revision.Snapshot, err = lifecycle.Snapshot(post)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = revision
	return cmsstore.ClonePost(post), lifecycle.CloneRevision(revision), nil
}

func (s *Store) PublishPost(id string) (cmsstore.Post, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.postIndex(id)
	if index < 0 {
		return cmsstore.Post{}, lifecycle.Revision{}, fmt.Errorf("post %q not found", strings.TrimSpace(id))
	}
	now := s.now()
	post := s.posts[index]
	post.State = publishState(post.State, now)
	post.Updated = now
	revision, err := s.appendRevisionLocked(cmsstore.ResourceKindPost, post.ID, post.Title, cmsstore.ActionPostPublished, "Published post.", &post, now)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	post.State.RevisionID = revision.ID
	s.posts[index] = post
	revision.Snapshot, err = lifecycle.Snapshot(post)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = revision
	return cmsstore.ClonePost(post), lifecycle.CloneRevision(revision), nil
}

func (s *Store) RestorePostRevision(id, revisionID string) (cmsstore.Post, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.postIndex(id)
	if index < 0 {
		return cmsstore.Post{}, lifecycle.Revision{}, fmt.Errorf("post %q not found", strings.TrimSpace(id))
	}
	revision, ok := lifecycle.FindRevision(s.revisions, cmsstore.ResourceKindPost, s.posts[index].ID, revisionID)
	if !ok {
		return cmsstore.Post{}, lifecycle.Revision{}, fmt.Errorf("post revision %q not found", strings.TrimSpace(revisionID))
	}
	post, err := lifecycle.DecodeSnapshot[cmsstore.Post](revision)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	now := s.now()
	post.ID = s.posts[index].ID
	post.State = rollbackState(post.State)
	post.Updated = now
	restore, err := s.appendRevisionLocked(cmsstore.ResourceKindPost, post.ID, post.Title, cmsstore.ActionPostRestored, "Restored post revision.", &post, now)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	post.State.RevisionID = restore.ID
	s.posts[index] = cmsstore.ClonePost(post)
	restore.Snapshot, err = lifecycle.Snapshot(post)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = restore
	return cmsstore.ClonePost(post), lifecycle.CloneRevision(restore), nil
}

func (s *Store) PreviewSiteSettings(input cmsstore.SiteSettingsInput) (cmsstore.SiteSettings, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(input.Title) == "" {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, fmt.Errorf("site settings title is required")
	}
	now := s.now()
	input.State = previewState(input.State)
	settings := cmsstore.NormalizeSiteSettings(input, s.settings, now)
	if strings.TrimSpace(settings.ID) == "" {
		settings.ID = "site"
	}
	revision, err := s.appendRevisionLocked(cmsstore.ResourceKindSiteSettings, settings.ID, settings.Title, cmsstore.ActionSettingsPreviewSaved, "Saved settings preview.", &settings, now)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	settings.State.RevisionID = revision.ID
	s.settings = settings
	s.hasSettings = true
	revision.Snapshot, err = lifecycle.Snapshot(settings)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = revision
	return cmsstore.CloneSiteSettings(settings), lifecycle.CloneRevision(revision), nil
}

func (s *Store) PublishSiteSettings() (cmsstore.SiteSettings, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.hasSettings {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, fmt.Errorf("site settings not found")
	}
	now := s.now()
	settings := s.settings
	settings.State = publishState(settings.State, now)
	settings.Updated = now
	revision, err := s.appendRevisionLocked(cmsstore.ResourceKindSiteSettings, settings.ID, settings.Title, cmsstore.ActionSettingsPublished, "Published settings.", &settings, now)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	settings.State.RevisionID = revision.ID
	s.settings = settings
	revision.Snapshot, err = lifecycle.Snapshot(settings)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = revision
	return cmsstore.CloneSiteSettings(settings), lifecycle.CloneRevision(revision), nil
}

func (s *Store) RestoreSiteSettingsRevision(revisionID string) (cmsstore.SiteSettings, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceID := strings.TrimSpace(s.settings.ID)
	if resourceID == "" {
		resourceID = "site"
	}
	revision, ok := lifecycle.FindRevision(s.revisions, cmsstore.ResourceKindSiteSettings, resourceID, revisionID)
	if !ok {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, fmt.Errorf("settings revision %q not found", strings.TrimSpace(revisionID))
	}
	settings, err := lifecycle.DecodeSnapshot[cmsstore.SiteSettings](revision)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	now := s.now()
	if strings.TrimSpace(settings.ID) == "" {
		settings.ID = resourceID
	}
	settings.State = rollbackState(settings.State)
	settings.Updated = now
	restore, err := s.appendRevisionLocked(cmsstore.ResourceKindSiteSettings, settings.ID, settings.Title, cmsstore.ActionSettingsRestored, "Restored settings revision.", &settings, now)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	settings.State.RevisionID = restore.ID
	s.settings = settings
	s.hasSettings = true
	restore.Snapshot, err = lifecycle.Snapshot(settings)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	s.revisions[len(s.revisions)-1] = restore
	return cmsstore.CloneSiteSettings(settings), lifecycle.CloneRevision(restore), nil
}

func (s *Store) upsertPageLocked(id string, input cmsstore.PageInput, now time.Time) (cmsstore.Page, int, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		page := cmsstore.NormalizePage(input, cmsstore.Page{ID: s.nextID("page")}, now)
		s.pages = append(s.pages, page)
		return page, len(s.pages) - 1, nil
	}
	index := s.pageIndex(id)
	if index < 0 {
		return cmsstore.Page{}, -1, fmt.Errorf("page %q not found", id)
	}
	page := cmsstore.NormalizePage(input, s.pages[index], now)
	s.pages[index] = page
	return page, index, nil
}

func (s *Store) upsertPostLocked(id string, input cmsstore.PostInput, now time.Time) (cmsstore.Post, int, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		post := cmsstore.NormalizePost(input, cmsstore.Post{ID: s.nextID("post")}, now)
		s.posts = append(s.posts, post)
		return post, len(s.posts) - 1, nil
	}
	index := s.postIndex(id)
	if index < 0 {
		return cmsstore.Post{}, -1, fmt.Errorf("post %q not found", id)
	}
	post := cmsstore.NormalizePost(input, s.posts[index], now)
	s.posts[index] = post
	return post, index, nil
}

func (s *Store) appendRevisionLocked(resourceKind, resourceID, resourceTitle, action, summary string, snapshot any, now time.Time) (lifecycle.Revision, error) {
	revision, err := lifecycle.NewRevision(lifecycle.RevisionInput{
		ID:            s.nextID("rev"),
		ResourceKind:  resourceKind,
		ResourceID:    resourceID,
		ResourceTitle: resourceTitle,
		Action:        action,
		Summary:       summary,
		Snapshot:      snapshot,
		Created:       now,
	}, now)
	if err != nil {
		return lifecycle.Revision{}, err
	}
	s.revisions = append(s.revisions, revision)
	return revision, nil
}

func (s *Store) pageIndex(id string) int {
	id = strings.TrimSpace(id)
	for index, page := range s.pages {
		if page.ID == id {
			return index
		}
	}
	return -1
}

func (s *Store) postIndex(id string) int {
	id = strings.TrimSpace(id)
	for index, post := range s.posts {
		if post.ID == id {
			return index
		}
	}
	return -1
}

func previewState(state cmsstore.State) cmsstore.State {
	state.Publish = cmsstore.PublishStateDraft
	state.Draft = cmsstore.DraftStatePreview
	state.PublishedAt = nil
	state.RevisionID = ""
	return cmsstore.NormalizeState(state)
}

func publishState(state cmsstore.State, now time.Time) cmsstore.State {
	state.Publish = cmsstore.PublishStatePublished
	state.Draft = cmsstore.DraftStateDraft
	state.PublishedAt = &now
	state.RevisionID = ""
	return cmsstore.NormalizeState(state)
}

func rollbackState(state cmsstore.State) cmsstore.State {
	state.Draft = cmsstore.DraftStateRollback
	state.RevisionID = ""
	return cmsstore.NormalizeState(state)
}
