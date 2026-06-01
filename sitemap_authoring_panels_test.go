package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderSiteMapAuthoringPanelsUsesSharedAuthoringContracts(t *testing.T) {
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
			}, {
				Key:           "programs",
				TemplateKey:   "programs",
				Label:         "Programs",
				GoSXComponent: "ProgramsGrid",
				Source:        ComponentSourceCMS,
				Binding:       "home.section.programs",
				Status:        "Visible",
				Editable:      true,
				Visible:       true,
				CanReorder:    true,
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
			}},
		},
	}), SiteMapViewOptions{})

	html := gosx.RenderHTML(RenderSiteMapAuthoringPanels(view, SiteMapAuthoringPanelsOptions{
		IncludePageListInCompositionPanel: true,
	}))

	for _, fragment := range []string{
		`data-gosx-studio-authoring-panel-renderer="gosx-studio"`,
		`data-gosx-studio-page-metadata="true"`,
		`data-studio-site-map-page-edit="true"`,
		`name="gosx_studio_page_label" value="Home"`,
		`data-gosx-studio-editable-control="true"`,
		`name="gosx_studio_value" value="Forest mornings"`,
		`data-gosx-studio-component-reorder="true"`,
		`data-studio-site-map-component-reorder="true"`,
		`data-gosx-studio-component-duplicate="true"`,
		`data-gosx-studio-component-visibility="true"`,
		`data-gosx-studio-component-delete="true"`,
		`data-studio-composition-intent-panel="true"`,
		`data-studio-composition-intent="create-page:landing"`,
		`data-gosx-studio-page-list="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered site-map authoring panels missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("shared authoring panels must not inject visible platform copy:\n%s", html)
	}
}
