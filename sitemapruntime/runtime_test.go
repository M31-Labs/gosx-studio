package sitemapruntime

import (
	"strings"
	"testing"
)

func TestBundlePublishesSiteMapRuntime(t *testing.T) {
	bundle := string(Bundle())
	for _, fragment := range []string{
		"window.GoSXStudioSiteMapRuntime",
		"data-studio-site-map-filter-control",
		"data-studio-site-map-detail-target",
		"data-studio-site-map-palette-control",
		"data-studio-site-map-selected-node",
		"gosxstudio:site-map-change",
		"gosxstudio:fragments-refreshed",
	} {
		if !strings.Contains(bundle, fragment) {
			t.Fatalf("site-map runtime missing %q", fragment)
		}
	}
}
