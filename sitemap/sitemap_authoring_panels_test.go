package sitemap

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

func TestRenderSiteMapAuthoringPanelsUsesSharedAuthoringContracts(t *testing.T) {
	view := AuthoringSiteMapView(authoring.NoCodeAuthoringSurface(core.SiteMap{
		Pages: []core.Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         core.PageGroupSite,
			GoSXComponent: "HomePage",
			Status:        "Editable",
			Editable:      true,
			Selected:      true,
			Components: []core.Component{{
				Key:                 "hero",
				TemplateKey:         "hero",
				Label:               "Hero",
				GoSXComponent:       "HomeHero",
				Source:              core.ComponentSourceHost,
				Binding:             "home.section.hero",
				Status:              "Visible",
				Editable:            true,
				Visible:             true,
				CanReorder:          true,
				CanDuplicate:        true,
				CanToggleVisibility: true,
				CanDelete:           true,
				Controls: []core.Control{{
					Key:     "headline",
					Label:   "Headline",
					Kind:    core.ControlText,
					Binding: "pages.home.hero.headline",
					Value:   "Forest mornings",
				}},
			}, {
				Key:           "programs",
				TemplateKey:   "programs",
				Label:         "Programs",
				GoSXComponent: "ProgramsGrid",
				Source:        core.ComponentSourceCMS,
				Binding:       "home.section.programs",
				Status:        "Visible",
				Editable:      true,
				Visible:       true,
				CanReorder:    true,
			}},
		}},
		Library: core.CompositionLibrary{
			PageBlueprints: []core.PageBlueprint{{
				Key:           "landing",
				Label:         "Landing",
				RoutePattern:  "/landing",
				Group:         core.PageGroupContent,
				GoSXComponent: "LandingPage",
				Components: []core.ComponentTemplate{{
					Key:           "hero",
					Label:         "Hero",
					GoSXComponent: "HomeHero",
				}},
			}},
			ComponentTemplates: []core.ComponentTemplate{{
				Key:            "gallery",
				Label:          "Gallery",
				GoSXComponent:  "GalleryGrid",
				DefaultBinding: "home.section.gallery",
				Source:         core.ComponentSourceCMS,
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

	formsHTML := gosx.RenderHTML(RenderSiteMapAuthoringForms(view, SiteMapAuthoringFormsOptions{
		Action:    "/admin/editor/__actions/authoring",
		CSRFToken: "csrf-token",
	}))
	for _, fragment := range []string{
		`data-gosx-studio-authoring-forms-renderer="gosx-studio"`,
		`data-studio-composition-intent-forms="true"`,
		`id="studioSiteMapIntentForm-create-page-landing"`,
		`action="/admin/editor/__actions/authoring"`,
		`data-gosx-studio-authoring-managed="true"`,
		`data-gosx-enhance="form"`,
		`name="csrf_token" value="csrf-token"`,
		`data-studio-site-map-operation-form="metadata"`,
		`id="studioSiteMapEditableControlForm"`,
		`data-studio-site-map-operation-form="reorder"`,
		`data-studio-site-map-operation-form="duplicate"`,
		`data-studio-site-map-operation-form="visibility"`,
		`data-studio-site-map-operation-form="delete"`,
	} {
		if !strings.Contains(formsHTML, fragment) {
			t.Fatalf("rendered site-map authoring forms missing %q:\n%s", fragment, formsHTML)
		}
	}
	if strings.Contains(formsHTML, "GoSX Studio") {
		t.Fatalf("shared authoring forms must not inject visible platform copy:\n%s", formsHTML)
	}
}
