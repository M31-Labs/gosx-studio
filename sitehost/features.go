package sitehost

import (
	"net/http"
	"sort"
	"strings"

	"m31labs.dev/gosx"
)

// features.go lets the platform that runs a site decide which parts of the
// admin the site's plan includes. A site run on its own has everything.
//
// The list is set once, at start (-features or GOSX_SITE_FEATURES). It is
// not a setting the owner can change: their plan is decided where they pay
// for it. What is not included is hidden from the menu and answers with a
// short page saying so, never with a broken screen.

// Feature names the platform may hand a site.
const (
	FeatureBlog    = "blog"
	FeatureForms   = "forms"
	FeatureShop    = "shop"
	FeatureStats   = "stats"
	FeatureDomain  = "domain"
	FeatureTeam    = "team"
	FeatureStaging = "staging"
	FeatureSSO     = "sso"
)

// AllFeatures is every feature a site can have, which is what a site run on
// its own gets.
var AllFeatures = []string{FeatureBlog, FeatureForms, FeatureShop, FeatureStats, FeatureDomain, FeatureTeam, FeatureStaging, FeatureSSO}

// featureLabels are the words the owner sees when something is not part
// of the plan.
var featureLabels = map[string]string{
	FeatureBlog: "the blog", FeatureForms: "forms", FeatureShop: "the shop", FeatureStats: "visitor statistics",
	FeatureDomain: "a custom domain", FeatureTeam: "team accounts", FeatureStaging: "a staging address", FeatureSSO: "single sign-on",
}

// ParseFeatures reads a comma or space separated list. Empty means all;
// unknown names are ignored.
func ParseFeatures(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	known := map[string]bool{}
	for _, name := range AllFeatures {
		known[name] = true
	}
	seen := map[string]bool{}
	out := []string{}
	for _, field := range strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool { return r == ',' || r == ' ' || r == ';' || r == '\n' }) {
		if field == "all" {
			return nil
		}
		if known[field] && !seen[field] {
			seen[field] = true
			out = append(out, field)
		}
	}
	sort.Strings(out)
	return out
}

// featureOn reports whether the plan includes a feature. No list means
// every feature.
func (h *Host) featureOn(name string) bool {
	if len(h.opts.Features) == 0 {
		return true
	}
	for _, feature := range h.opts.Features {
		if feature == name {
			return true
		}
	}
	return false
}

// featureForPath says which feature an admin address belongs to, or "".
func featureForPath(path string) string {
	switch {
	case strings.HasPrefix(path, "/admin/posts"), strings.HasPrefix(path, "/admin/edit/post/"), strings.HasPrefix(path, "/admin/api/posts/"), strings.HasPrefix(path, "/admin/history/post/"):
		return FeatureBlog
	case strings.HasPrefix(path, "/admin/forms"):
		return FeatureForms
	case strings.HasPrefix(path, "/admin/shop"), strings.HasPrefix(path, "/admin/orders"), strings.HasPrefix(path, "/admin/bookings"):
		return FeatureShop
	case strings.HasPrefix(path, "/admin/stats"):
		return FeatureStats
	case strings.HasPrefix(path, "/admin/domain"):
		return FeatureDomain
	case strings.HasPrefix(path, peoplePath):
		return FeatureTeam
	case strings.HasPrefix(path, stagingAdminPath):
		return FeatureStaging
	case strings.HasPrefix(path, "/admin/sso"):
		return FeatureSSO
	}
	return ""
}

// featureGate answers admin addresses outside the plan with a short page.
func (h *Host) featureGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if feature := featureForPath(r.URL.Path); feature != "" && !h.featureOn(feature) {
			if strings.HasPrefix(r.URL.Path, "/admin/api/") {
				writeJSON(w, http.StatusForbidden, editorSaveResult{Message: "This isn't part of your plan."})
				return
			}
			h.writeNotInPlan(w, feature)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Host) writeNotInPlan(w http.ResponseWriter, feature string) {
	body := h.renderAdminShell("dashboard", "Not part of your plan",
		"Your plan doesn't include "+firstNonEmpty(featureLabels[feature], feature)+". Ask whoever runs your hosting to add it.",
		adminStatus{},
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "/admin")), gosx.Text("Back to the dashboard"))))
	h.writeDocument(w, http.StatusForbidden, h.adminMeta("Not part of your plan"), body)
}

// navFeature maps a menu item to the feature it needs, or "".
func navFeature(key string) string {
	switch key {
	case "posts":
		return FeatureBlog
	case "forms":
		return FeatureForms
	case "shop":
		return FeatureShop
	case "stats":
		return FeatureStats
	case "users":
		return FeatureTeam
	case "staging":
		return FeatureStaging
	}
	return ""
}
