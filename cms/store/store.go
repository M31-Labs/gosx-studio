package store

import (
	"strings"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

const (
	ResourceKindPage         = "page"
	ResourceKindPost         = "post"
	ResourceKindSiteSettings = "site_settings"

	ActionPagePreviewSaved     = "page.preview_saved"
	ActionPagePublished        = "page.published"
	ActionPageRestored         = "page.restored"
	ActionPostPreviewSaved     = "post.preview_saved"
	ActionPostPublished        = "post.published"
	ActionPostRestored         = "post.restored"
	ActionSettingsPreviewSaved = "settings.preview_saved"
	ActionSettingsPublished    = "settings.published"
	ActionSettingsRestored     = "settings.restored"
)

type DraftState = lifecycle.DraftState
type PublishState = lifecycle.PublishState
type Revision = lifecycle.Revision
type RevisionFilter = lifecycle.RevisionFilter
type RevisionInput = lifecycle.RevisionInput

const (
	DraftStateDraft    = lifecycle.DraftStateDraft
	DraftStatePreview  = lifecycle.DraftStatePreview
	DraftStateRollback = lifecycle.DraftStateRollback

	PublishStateDraft     = lifecycle.PublishStateDraft
	PublishStatePublished = lifecycle.PublishStatePublished
)

type State struct {
	Draft       DraftState   `json:"draftState,omitempty"`
	Publish     PublishState `json:"publishState"`
	PublishedAt *time.Time   `json:"publishedAt,omitempty"`
	RevisionID  string       `json:"revisionId,omitempty"`
}

type Metadata map[string]string

type Page struct {
	ID          string               `json:"id"`
	Slug        string               `json:"slug"`
	Title       string               `json:"title"`
	Description string               `json:"description,omitempty"`
	Body        blockstudio.Document `json:"body"`
	State       State                `json:"state"`
	Metadata    Metadata             `json:"metadata,omitempty"`
	Created     time.Time            `json:"created,omitempty"`
	Updated     time.Time            `json:"updated,omitempty"`
}

type PageInput struct {
	Slug        string
	Title       string
	Description string
	Body        blockstudio.Document
	State       State
	Metadata    Metadata
}

type PageFilter struct {
	Slug    string
	Publish PublishState
	Limit   int
}

type Post struct {
	ID       string               `json:"id"`
	Slug     string               `json:"slug"`
	Title    string               `json:"title"`
	Excerpt  string               `json:"excerpt,omitempty"`
	Author   string               `json:"author,omitempty"`
	Tags     []string             `json:"tags,omitempty"`
	Body     blockstudio.Document `json:"body"`
	State    State                `json:"state"`
	Metadata Metadata             `json:"metadata,omitempty"`
	Created  time.Time            `json:"created,omitempty"`
	Updated  time.Time            `json:"updated,omitempty"`
}

type PostInput struct {
	Slug     string
	Title    string
	Excerpt  string
	Author   string
	Tags     []string
	Body     blockstudio.Document
	State    State
	Metadata Metadata
}

type PostFilter struct {
	Slug    string
	Tag     string
	Publish PublishState
	Limit   int
}

type SiteSettings struct {
	ID          string    `json:"id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	BaseURL     string    `json:"baseUrl,omitempty"`
	Locale      string    `json:"locale,omitempty"`
	State       State     `json:"state"`
	Metadata    Metadata  `json:"metadata,omitempty"`
	Updated     time.Time `json:"updated,omitempty"`
}

type SiteSettingsInput struct {
	Title       string
	Description string
	BaseURL     string
	Locale      string
	State       State
	Metadata    Metadata
}

type PageStore interface {
	ListPages(PageFilter) ([]Page, error)
	PageByID(string) (Page, bool, error)
	PageBySlug(string) (Page, bool, error)
	CreatePage(PageInput) (Page, error)
	UpdatePage(string, PageInput) (Page, error)
}

type PostStore interface {
	ListPosts(PostFilter) ([]Post, error)
	PostByID(string) (Post, bool, error)
	PostBySlug(string) (Post, bool, error)
	CreatePost(PostInput) (Post, error)
	UpdatePost(string, PostInput) (Post, error)
}

type SiteSettingsStore interface {
	SiteSettings() (SiteSettings, bool, error)
	SaveSiteSettings(SiteSettingsInput) (SiteSettings, error)
}

type RevisionStore interface {
	ListRevisions(lifecycle.RevisionFilter) []lifecycle.Revision
	RevisionByID(resourceKind, resourceID, revisionID string) (lifecycle.Revision, bool)
	SaveRevision(lifecycle.RevisionInput) (lifecycle.Revision, error)
}

type PageLifecycleStore interface {
	PreviewPage(id string, input PageInput) (Page, lifecycle.Revision, error)
	PublishPage(id string) (Page, lifecycle.Revision, error)
	RestorePageRevision(id, revisionID string) (Page, lifecycle.Revision, error)
}

type PostLifecycleStore interface {
	PreviewPost(id string, input PostInput) (Post, lifecycle.Revision, error)
	PublishPost(id string) (Post, lifecycle.Revision, error)
	RestorePostRevision(id, revisionID string) (Post, lifecycle.Revision, error)
}

type SiteSettingsLifecycleStore interface {
	PreviewSiteSettings(input SiteSettingsInput) (SiteSettings, lifecycle.Revision, error)
	PublishSiteSettings() (SiteSettings, lifecycle.Revision, error)
	RestoreSiteSettingsRevision(revisionID string) (SiteSettings, lifecycle.Revision, error)
}

type LifecycleStore interface {
	PageLifecycleStore
	PostLifecycleStore
	SiteSettingsLifecycleStore
}

type Store interface {
	PageStore
	PostStore
	SiteSettingsStore
	RevisionStore
}

func NormalizeState(state State) State {
	if state.Draft == "" {
		state.Draft = DraftStateDraft
	}
	if state.Publish == "" {
		state.Publish = PublishStateDraft
	}
	state.RevisionID = strings.TrimSpace(state.RevisionID)
	if state.Publish != PublishStatePublished {
		state.PublishedAt = nil
	}
	return state
}

func (state State) IsPublished() bool {
	return state.Publish == PublishStatePublished
}

func (state State) IsDraft() bool {
	return state.Publish == PublishStateDraft
}

func NormalizePage(input PageInput, page Page, now time.Time) Page {
	page.Slug = NormalizeSlug(input.Slug)
	page.Title = strings.TrimSpace(input.Title)
	page.Description = strings.TrimSpace(input.Description)
	page.Body = CloneDocument(input.Body)
	page.State = NormalizeState(input.State)
	page.Metadata = NormalizeMetadata(input.Metadata)
	if page.Created.IsZero() {
		page.Created = now
	}
	page.Updated = now
	return page
}

func NormalizePost(input PostInput, post Post, now time.Time) Post {
	post.Slug = NormalizeSlug(input.Slug)
	post.Title = strings.TrimSpace(input.Title)
	post.Excerpt = strings.TrimSpace(input.Excerpt)
	post.Author = strings.TrimSpace(input.Author)
	post.Tags = NormalizeTags(input.Tags)
	post.Body = CloneDocument(input.Body)
	post.State = NormalizeState(input.State)
	post.Metadata = NormalizeMetadata(input.Metadata)
	if post.Created.IsZero() {
		post.Created = now
	}
	post.Updated = now
	return post
}

func NormalizeSiteSettings(input SiteSettingsInput, settings SiteSettings, now time.Time) SiteSettings {
	settings.Title = strings.TrimSpace(input.Title)
	settings.Description = strings.TrimSpace(input.Description)
	settings.BaseURL = strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	settings.Locale = strings.TrimSpace(input.Locale)
	settings.State = NormalizeState(input.State)
	settings.Metadata = NormalizeMetadata(input.Metadata)
	settings.Updated = now
	return settings
}

func NormalizeSlug(slug string) string {
	slug = strings.Trim(strings.TrimSpace(slug), "/")
	if slug == "" {
		return ""
	}
	return strings.Join(strings.Fields(slug), "-")
}

func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	return out
}

func ClonePage(page Page) Page {
	page.Body = CloneDocument(page.Body)
	page.Metadata = CloneMetadata(page.Metadata)
	return page
}

func ClonePages(pages []Page) []Page {
	out := make([]Page, len(pages))
	for i, page := range pages {
		out[i] = ClonePage(page)
	}
	return out
}

func ClonePost(post Post) Post {
	post.Tags = append([]string(nil), post.Tags...)
	post.Body = CloneDocument(post.Body)
	post.Metadata = CloneMetadata(post.Metadata)
	return post
}

func ClonePosts(posts []Post) []Post {
	out := make([]Post, len(posts))
	for i, post := range posts {
		out[i] = ClonePost(post)
	}
	return out
}

func CloneSiteSettings(settings SiteSettings) SiteSettings {
	settings.Metadata = CloneMetadata(settings.Metadata)
	return settings
}

func CloneMetadata(metadata Metadata) Metadata {
	if metadata == nil {
		return nil
	}
	out := Metadata{}
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func NormalizeMetadata(metadata Metadata) Metadata {
	if metadata == nil {
		return nil
	}
	out := Metadata{}
	for key, value := range metadata {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func CloneDocument(doc blockstudio.Document) blockstudio.Document {
	doc.Blocks = cloneBlocks(doc.Blocks)
	return doc
}

func cloneBlocks(blocks []blockstudio.BlockInstance) []blockstudio.BlockInstance {
	if blocks == nil {
		return nil
	}
	out := make([]blockstudio.BlockInstance, len(blocks))
	for i, block := range blocks {
		out[i] = block
		out[i].Values = cloneValues(block.Values)
	}
	return out
}

func cloneValues(values blockstudio.Values) blockstudio.Values {
	if values == nil {
		return nil
	}
	out := blockstudio.Values{}
	for key, value := range values {
		out[key] = cloneValue(value)
	}
	return out
}

func cloneValue(value blockstudio.Value) blockstudio.Value {
	value.List = cloneValueList(value.List)
	value.Object = cloneValueObject(value.Object)
	if value.Media != nil {
		media := *value.Media
		value.Media = &media
	}
	if value.Relation != nil {
		relation := *value.Relation
		value.Relation = &relation
	}
	return value
}

func cloneValueList(values []blockstudio.Value) []blockstudio.Value {
	if values == nil {
		return nil
	}
	out := make([]blockstudio.Value, len(values))
	for i, value := range values {
		out[i] = cloneValue(value)
	}
	return out
}

func cloneValueObject(values map[string]blockstudio.Value) map[string]blockstudio.Value {
	if values == nil {
		return nil
	}
	out := map[string]blockstudio.Value{}
	for key, value := range values {
		out[key] = cloneValue(value)
	}
	return out
}
