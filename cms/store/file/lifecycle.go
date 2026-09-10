package file

import (
	"m31labs.dev/gosx-studio/cms/lifecycle"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// lifecycle.go re-exports the draft/publish/restore surface of the wrapped
// *memory.Store with the same persist-or-recover discipline the CRUD methods in
// file.go use: take the pre-mutation snapshot, delegate, then write the new
// snapshot to disk and roll memory back if that write fails.
//
// Without these nine methods the file store satisfied cmsstore.Store but not
// cmsstore.LifecycleStore, which left Studio shipping no durable store that
// could preview, publish, or restore anything — so every host wrote that path
// itself. The wrapped memory store already implemented all nine.
var _ cmsstore.LifecycleStore = (*Store)(nil)

func (s *Store) PreviewPage(id string, input cmsstore.PageInput) (cmsstore.Page, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	page, revision, err := s.store.PreviewPage(id, input)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	return page, revision, s.persistMutationLocked(previous)
}

func (s *Store) PublishPage(id string) (cmsstore.Page, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	page, revision, err := s.store.PublishPage(id)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	return page, revision, s.persistMutationLocked(previous)
}

func (s *Store) RestorePageRevision(id, revisionID string) (cmsstore.Page, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	page, revision, err := s.store.RestorePageRevision(id, revisionID)
	if err != nil {
		return cmsstore.Page{}, lifecycle.Revision{}, err
	}
	return page, revision, s.persistMutationLocked(previous)
}

func (s *Store) PreviewPost(id string, input cmsstore.PostInput) (cmsstore.Post, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	post, revision, err := s.store.PreviewPost(id, input)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	return post, revision, s.persistMutationLocked(previous)
}

func (s *Store) PublishPost(id string) (cmsstore.Post, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	post, revision, err := s.store.PublishPost(id)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	return post, revision, s.persistMutationLocked(previous)
}

func (s *Store) RestorePostRevision(id, revisionID string) (cmsstore.Post, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	post, revision, err := s.store.RestorePostRevision(id, revisionID)
	if err != nil {
		return cmsstore.Post{}, lifecycle.Revision{}, err
	}
	return post, revision, s.persistMutationLocked(previous)
}

func (s *Store) PreviewSiteSettings(input cmsstore.SiteSettingsInput) (cmsstore.SiteSettings, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	settings, revision, err := s.store.PreviewSiteSettings(input)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	return settings, revision, s.persistMutationLocked(previous)
}

func (s *Store) PublishSiteSettings() (cmsstore.SiteSettings, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	settings, revision, err := s.store.PublishSiteSettings()
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	return settings, revision, s.persistMutationLocked(previous)
}

func (s *Store) RestoreSiteSettingsRevision(revisionID string) (cmsstore.SiteSettings, lifecycle.Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.store.Snapshot()
	settings, revision, err := s.store.RestoreSiteSettingsRevision(revisionID)
	if err != nil {
		return cmsstore.SiteSettings{}, lifecycle.Revision{}, err
	}
	return settings, revision, s.persistMutationLocked(previous)
}
