package sitehost

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAddAPageStartsFromALayout(t *testing.T) {
	host, handler := newTestHost(t)
	mustContain(t, get(t, handler, "/admin/pages").Body.String(), `name="template" value="landing"`, "the form offers layouts")
	rec := post(t, handler, "/admin/pages", url.Values{"title": {"Why us"}, "slug": {"why-us"}, "template": {"landing"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create = %d %s", rec.Code, rec.Body.String())
	}
	page, _, _ := host.store.PageBySlug("why-us")
	keys := []string{}
	for _, block := range page.Body.Blocks {
		keys = append(keys, block.Key)
	}
	if strings.Join(keys, " ") != "hero features section stats section testimonials cta" {
		t.Fatalf("landing layout = %q", strings.Join(keys, " "))
	}
	if page.Body.Blocks[0].Values["eyebrow"].String != "Why us" || len(page.Body.Blocks[1].Values["items"].List) != 3 {
		t.Fatalf("hero/features: %+v", page.Body.Blocks[0].Values)
	}
	editor := get(t, handler, "/admin/edit/"+page.ID).Body.String()
	mustContain(t, editor, `data-composite="hero"`, "the editor opens on the layout")
	mustContain(t, editor, `data-composite="testimonials"`, "with every section")
	post(t, handler, "/admin/api/pages/"+page.ID+"/publish", url.Values{})
	public := get(t, handler, "/why-us").Body.String()
	mustContain(t, public, "site-hero site-hero--center", "and it publishes")
	mustContain(t, public, "site-section site-section--tinted site-section--align-center", "with its section breaks")

	// Unknown layouts fall back to blank; every layout builds and renders.
	post(t, handler, "/admin/pages", url.Values{"title": {"Plain"}, "template": {"nonsense"}})
	plain, _, _ := host.store.PageBySlug("plain")
	if len(plain.Body.Blocks) != 1 || plain.Body.Blocks[0].Key != "paragraph" {
		t.Fatalf("blank fallback: %+v", plain.Body.Blocks)
	}
	for _, template := range PageTemplates() {
		post(t, handler, "/admin/pages", url.Values{"title": {"T " + template.Key}, "template": {template.Key}})
		created, ok, _ := host.store.PageBySlug("t-" + template.Key)
		if !ok || len(created.Body.Blocks) == 0 {
			t.Fatalf("%s did not build", template.Key)
		}
		if code := get(t, handler, "/admin/edit/"+created.ID).Code; code != http.StatusOK {
			t.Fatalf("%s editor = %d", template.Key, code)
		}
	}
}
