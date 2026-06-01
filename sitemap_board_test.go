package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderSiteMapBoardUsesSharedRuntimeContracts(t *testing.T) {
	view := AuthoringSiteMapView(NoCodeAuthoringSurface(SiteMap{
		Pages: []Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Status:        "Editable",
			Editable:      true,
			Selected:      true,
			Components: []Component{{
				Key:                 "hero",
				TemplateKey:         "hero",
				Label:               "Hero",
				GoSXComponent:       "HomeHero",
				Source:              ComponentSourceHost,
				Binding:             "home.section.hero",
				Status:              "Visible",
				Editable:            true,
				Visible:             true,
				CanReorder:          true,
				CanDuplicate:        true,
				CanToggleVisibility: true,
				CanDelete:           true,
				Controls: []Control{{
					Key:     "headline",
					Label:   "Headline",
					Kind:    ControlText,
					Binding: "pages.home.hero.headline",
					Value:   "Forest mornings",
				}},
			}},
		}},
		Library: CompositionLibrary{
			PageBlueprints: []PageBlueprint{{
				Key:           "landing",
				Label:         "Landing",
				RoutePattern:  "/landing",
				Group:         PageGroupContent,
				GoSXComponent: "LandingPage",
				Components: []ComponentTemplate{{
					Key:           "hero",
					Label:         "Hero",
					GoSXComponent: "HomeHero",
				}},
			}},
			ComponentTemplates: []ComponentTemplate{{
				Key:            "gallery",
				Label:          "Gallery",
				GoSXComponent:  "GalleryGrid",
				DefaultBinding: "home.section.gallery",
				Source:         ComponentSourceCMS,
				Category:       "Page sections",
				Controls: []Control{{
					Key:   "caption",
					Label: "Caption",
					Kind:  ControlText,
				}},
			}},
		},
	}), SiteMapViewOptions{})

	html := gosx.RenderHTML(RenderSiteMapBoard(view, SiteMapBoardOptions{}))

	for _, fragment := range []string{
		`data-gosx-studio-site-map-board-renderer="gosx-studio"`,
		`class="studio-site-map-board"`,
		`data-studio-site-map-detail-control="runtime"`,
		`data-studio-site-map-filter-control="all"`,
		`data-studio-site-map-focus-control="home"`,
		`data-studio-site-map-detail-target="build"`,
		`data-studio-site-map-palette-control="page-sections"`,
		`data-studio-site-map-workspace="true"`,
		`class="studio-site-map-workspace-layer__composition"`,
		`data-studio-site-map-workspace-node="page:home"`,
		`data-studio-site-map-selected-label="true"`,
		`for="studioSiteMapInspectorFocus"`,
		`data-studio-site-map-builder="true"`,
		`data-studio-composition-intent="create-page:landing"`,
		`data-studio-site-map-blueprint="landing"`,
		`data-studio-site-map-component-template="gallery"`,
		`data-studio-site-map-viewport="true"`,
		`data-gosx-studio-authoring-operation="save-control"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered site-map board missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("shared site-map board must not inject visible platform copy:\n%s", html)
	}
}
