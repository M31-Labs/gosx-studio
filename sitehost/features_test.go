package sitehost

import (
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseFeatures(t *testing.T) {
	if got := ParseFeatures(""); got != nil {
		t.Fatalf("empty = %v", got)
	}
	if got := ParseFeatures("all"); got != nil {
		t.Fatalf("all = %v", got)
	}
	if got := ParseFeatures("Shop, blog,nonsense shop"); !reflect.DeepEqual(got, []string{"blog", "shop"}) {
		t.Fatalf("parsed = %v", got)
	}
}

func TestFeaturesHideWhatThePlanLacks(t *testing.T) {
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Wildflower", SiteKind: "food", Seed: true, NoBackups: true, Features: []string{FeatureBlog, FeatureForms}})
	if err != nil {
		t.Fatal(err)
	}
	handler := host.Handler()
	nav := get(t, handler, "/admin").Body.String()
	for _, present := range []string{`href="/admin/posts"`, `href="/admin/forms"`, `href="/admin/pages"`, `href="/admin/settings"`} {
		mustContain(t, nav, present, "planned things stay in the menu")
	}
	for _, absent := range []string{`href="/admin/shop"`, `href="/admin/stats"`, `href="/admin/users"`, `href="/admin/staging"`} {
		if strings.Contains(nav, absent) {
			t.Fatalf("%s must leave the menu", absent)
		}
	}
	rec := get(t, handler, "/admin/shop")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("shop = %d", rec.Code)
	}
	mustContain(t, rec.Body.String(), "doesn&#39;t include the shop", "and says so plainly")
	if code := get(t, handler, "/admin/posts").Code; code != http.StatusOK {
		t.Fatalf("blog = %d", code)
	}
	if code := get(t, handler, "/admin/staging").Code; code != http.StatusForbidden {
		t.Fatalf("staging = %d", code)
	}
	settings := get(t, handler, "/admin/settings").Body.String()
	if strings.Contains(settings, "ssoIssuer") || strings.Contains(settings, "Connect a domain") {
		t.Fatal("single sign-on and domain settings must not show")
	}
	if strings.Contains(get(t, handler, "/").Body.String(), "stats.js") {
		t.Fatal("no visitor beacon without the stats feature")
	}
	if rec := postJSON(t, handler, "/admin/api/posts/nope", "{}"); rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("planned api = %d", rec.Code)
	}
	if rec := post(t, handler, "/admin/shop", url.Values{}); rec.Code != http.StatusForbidden {
		t.Fatalf("shop write = %d", rec.Code)
	}

	// A site run on its own has everything.
	full, _ := newTestHost(t)
	if !full.featureOn(FeatureSSO) || !full.featureOn(FeatureShop) {
		t.Fatal("no list means every feature")
	}
}
