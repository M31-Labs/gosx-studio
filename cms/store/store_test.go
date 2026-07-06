package store

import (
	"testing"
	"time"

	"m31labs.dev/gosx-admin/blockstudio"
)

func TestNormalizePageTrimsDefaultsAndClones(t *testing.T) {
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	body := blockstudio.Document{
		Version: 1,
		Kind:    "content.body",
		Blocks: []blockstudio.BlockInstance{{
			ID:      "hero",
			Key:     "heading",
			Enabled: true,
			Values: blockstudio.Values{
				"text": {Kind: blockstudio.FieldText, String: " Care "},
				"media": {Media: &blockstudio.MediaValue{
					URL: "/media/care.jpg",
					Alt: "Care",
				}},
			},
		}},
	}

	page := NormalizePage(PageInput{
		Slug:        " /care guide/ ",
		Title:       " Care Guide ",
		Description: " Learn care ",
		Body:        body,
		Metadata:    Metadata{" section ": " pages ", "empty": " "},
	}, Page{ID: "page_1"}, now)

	if page.ID != "page_1" || page.Slug != "care-guide" || page.Title != "Care Guide" || page.State.Publish != PublishStateDraft || page.State.Draft != DraftStateDraft {
		t.Fatalf("unexpected normalized page: %#v", page)
	}
	if page.Created != now || page.Updated != now {
		t.Fatalf("expected timestamps to be set, got created=%s updated=%s", page.Created, page.Updated)
	}
	if len(page.Metadata) != 1 || page.Metadata["section"] != "pages" {
		t.Fatalf("unexpected metadata: %#v", page.Metadata)
	}

	pageMedia := page.Body.Blocks[0].Values["media"]
	pageMedia.Media.Alt = "Changed"
	if body.Blocks[0].Values["media"].Media.Alt != "Care" {
		t.Fatalf("expected body to be cloned, got %#v", body.Blocks[0].Values["media"].Media)
	}
}

func TestNormalizeStatePublishedKeepsDateAndDraftClearsDate(t *testing.T) {
	publishedAt := time.Date(2026, 5, 16, 9, 0, 0, 0, time.UTC)
	published := NormalizeState(State{Publish: PublishStatePublished, PublishedAt: &publishedAt, RevisionID: " rev_1 "})
	if !published.IsPublished() || published.PublishedAt == nil || published.RevisionID != "rev_1" {
		t.Fatalf("unexpected published state: %#v", published)
	}

	draft := NormalizeState(State{Publish: PublishStateDraft, PublishedAt: &publishedAt})
	if draft.PublishedAt != nil || !draft.IsDraft() {
		t.Fatalf("expected draft state to clear published date, got %#v", draft)
	}
}

func TestNormalizePostDedupeTagsAndClone(t *testing.T) {
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	body := blockstudio.Document{
		Blocks: []blockstudio.BlockInstance{{
			Values: blockstudio.Values{
				"items": {List: []blockstudio.Value{{String: "a"}}},
			},
		}},
	}
	post := NormalizePost(PostInput{
		Slug:   "journal/First Post",
		Title:  " First Post ",
		Author: " Editor ",
		Tags:   []string{"News", " news ", "", "Care"},
		Body:   body,
	}, Post{}, now)

	if post.Slug != "journal/First-Post" || post.Author != "Editor" {
		t.Fatalf("unexpected normalized post: %#v", post)
	}
	if len(post.Tags) != 2 || post.Tags[0] != "News" || post.Tags[1] != "Care" {
		t.Fatalf("unexpected tags: %#v", post.Tags)
	}

	postItems := post.Body.Blocks[0].Values["items"]
	postItems.List[0].String = "changed"
	if body.Blocks[0].Values["items"].List[0].String != "a" {
		t.Fatalf("expected nested values to be cloned, got %#v", body.Blocks[0].Values["items"])
	}
}

func TestNormalizeSiteSettingsTrimsAndClonesMetadata(t *testing.T) {
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	settings := NormalizeSiteSettings(SiteSettingsInput{
		Title:       " Studio ",
		Description: " CMS ",
		BaseURL:     " https://example.com/ ",
		Locale:      " en-US ",
		State:       State{Publish: PublishStatePublished},
		Metadata:    Metadata{" owner ": " content "},
	}, SiteSettings{ID: "site"}, now)

	if settings.ID != "site" || settings.Title != "Studio" || settings.BaseURL != "https://example.com" || settings.Locale != "en-US" || settings.State.Publish != PublishStatePublished || settings.Updated != now {
		t.Fatalf("unexpected settings: %#v", settings)
	}
	settings.Metadata["owner"] = "changed"
	cloned := CloneSiteSettings(settings)
	cloned.Metadata["owner"] = "clone"
	if settings.Metadata["owner"] != "changed" {
		t.Fatalf("expected settings metadata to be cloned, got %#v", settings.Metadata)
	}
}

func TestStoreContractsCompile(t *testing.T) {
	var _ PageStore = (*contractStore)(nil)
	var _ PostStore = (*contractStore)(nil)
	var _ SiteSettingsStore = (*contractStore)(nil)
	var _ RevisionStore = (*contractStore)(nil)
	var _ Store = (*contractStore)(nil)
}

type contractStore struct{}

func (*contractStore) ListPages(PageFilter) ([]Page, error) { return nil, nil }
func (*contractStore) PageByID(string) (Page, bool, error)  { return Page{}, false, nil }
func (*contractStore) PageBySlug(string) (Page, bool, error) {
	return Page{}, false, nil
}
func (*contractStore) CreatePage(PageInput) (Page, error)         { return Page{}, nil }
func (*contractStore) UpdatePage(string, PageInput) (Page, error) { return Page{}, nil }

func (*contractStore) ListPosts(PostFilter) ([]Post, error) { return nil, nil }
func (*contractStore) PostByID(string) (Post, bool, error)  { return Post{}, false, nil }
func (*contractStore) PostBySlug(string) (Post, bool, error) {
	return Post{}, false, nil
}
func (*contractStore) CreatePost(PostInput) (Post, error)         { return Post{}, nil }
func (*contractStore) UpdatePost(string, PostInput) (Post, error) { return Post{}, nil }

func (*contractStore) SiteSettings() (SiteSettings, bool, error) {
	return SiteSettings{}, false, nil
}
func (*contractStore) SaveSiteSettings(SiteSettingsInput) (SiteSettings, error) {
	return SiteSettings{}, nil
}

func (*contractStore) ListRevisions(RevisionFilter) []Revision { return nil }
func (*contractStore) RevisionByID(string, string, string) (Revision, bool) {
	return Revision{}, false
}
func (*contractStore) SaveRevision(RevisionInput) (Revision, error) { return Revision{}, nil }
