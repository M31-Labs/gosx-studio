package sitemap

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

// TestRenderSiteMapEditableControlPanelDelegatesKind and
// TestEditableControlViewKindPresent moved here from the (pre-restructure)
// root package's inspector_test.go when Slice 6 of the package restructure
// (see .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) extracted
// inspector.go into m31labs.dev/gosx-studio/panels: they exercise the
// site-map's editable-control panel pipeline end-to-end
// (AuthoringSiteMapView -> RenderSiteMapAuthoringPanels), which delegates to
// this package's own frozen copy of the inspector-widget renderer
// (sitemap/support.go's RenderInspectorControl/inspectorWidget). panels
// cannot be imported here (panels sits beside sitemap, not below it, per
// the spec's §1 Import DAG), so this coverage now lives where the
// implementation it exercises actually lives.

// TestRenderSiteMapEditableControlPanelDelegatesKind checks that the panel
// dispatches to the kind-aware widget (rich-text → textarea).
func TestRenderSiteMapEditableControlPanelDelegatesKind(t *testing.T) {
	view := AuthoringSiteMapView(authoring.NoCodeAuthoringSurface(core.SiteMap{
		Pages: []core.Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         core.PageGroupSite,
			GoSXComponent: "HomePage",
			Editable:      true,
			Selected:      true,
			Components: []core.Component{{
				Key:           "hero",
				Label:         "Hero",
				GoSXComponent: "HomeHero",
				Source:        core.ComponentSourceHost,
				Binding:       "home.section.hero",
				Editable:      true,
				Controls: []core.Control{{
					Key:     "bio",
					Label:   "Bio",
					Kind:    core.ControlRichText,
					Binding: "pages.home.hero.bio",
					Value:   "A long paragraph.",
				}},
			}},
		}},
	}), SiteMapViewOptions{})

	html := gosx.RenderHTML(RenderSiteMapAuthoringPanels(view, SiteMapAuthoringPanelsOptions{}))

	// Data-attribute contract must be preserved
	if !strings.Contains(html, `data-gosx-studio-editable-control="true"`) {
		t.Fatalf("panel must still carry data-gosx-studio-editable-control:\n%s", html)
	}
	// rich-text → textarea
	if !strings.Contains(html, "<textarea") {
		t.Fatalf("rich-text control must render <textarea> via RenderInspectorControl:\n%s", html)
	}
	// Cancel button present
	if !strings.Contains(html, `type="reset"`) && !strings.Contains(html, `data-gosx-studio-inspector-cancel`) {
		t.Fatalf("panel must include Cancel button:\n%s", html)
	}
	// No platform copy
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("panel must not inject platform copy:\n%s", html)
	}
}

// TestEditableControlViewKindPresent verifies the projection exposes "kind".
func TestEditableControlViewKindPresent(t *testing.T) {
	view := AuthoringSiteMapView(authoring.NoCodeAuthoringSurface(core.SiteMap{
		Pages: []core.Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         core.PageGroupSite,
			GoSXComponent: "HomePage",
			Editable:      true,
			Selected:      true,
			Components: []core.Component{{
				Key:           "hero",
				Label:         "Hero",
				GoSXComponent: "HomeHero",
				Source:        core.ComponentSourceHost,
				Editable:      true,
				Controls: []core.Control{{
					Key:     "headline",
					Label:   "Headline",
					Kind:    core.ControlRichText,
					Binding: "pages.home.hero.headline",
					Value:   "Forest mornings",
				}},
			}},
		}},
	}), SiteMapViewOptions{})

	ctrl := workbenchViewMap(view, "editableControl")
	if ctrl == nil {
		t.Fatal("editableControl must be present in the view")
	}
	kind, ok := ctrl["kind"].(string)
	if !ok || kind == "" {
		t.Fatalf("editableControl view must carry 'kind' as a non-empty string, got: %v", ctrl["kind"])
	}
	if kind != string(core.ControlRichText) {
		t.Fatalf("editableControl 'kind' must be %q, got %q", string(core.ControlRichText), kind)
	}
}
