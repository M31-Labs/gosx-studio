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

// TestRenderSiteMapSharedComponentPalettePanelDispatchesPlaceInstance proves
// the composition workspace palette's real, functional "place another
// instance" action: a rendered button referencing a hidden form (by id) that
// RenderSiteMapAuthoringForms independently emits, carrying the durable
// place-instance operation's hidden fields — the SAME
// `form=`/hidden-input/action-endpoint pattern the existing composition
// intent panel already uses, so shared components dispatch through the
// identical mechanism as every other placeable palette entry.
func TestRenderSiteMapSharedComponentPalettePanelDispatchesPlaceInstance(t *testing.T) {
	siteMap := core.SiteMap{Pages: []core.Page{{
		Key: "about", Label: "About", Route: "/about", GoSXComponent: "ContentPage", Editable: true, Selected: true,
		Components: []core.Component{{Key: "testimonial-1", Label: "One", GoSXComponent: "SharedComponentInstance", DefinitionKey: "testimonial"}},
	}}, Library: core.CompositionLibrary{
		ComponentDefinitions: []core.ComponentDefinition{{
			Key: "testimonial", Label: "Testimonial", Summary: "Reusable quote block.",
			Controls: []core.Control{{Key: "quote", Label: "Quote", Kind: core.ControlText}},
		}},
	}}
	view := SiteMapAuthoringView(siteMap, SiteMapViewOptions{})

	panelsHTML := gosx.RenderHTML(RenderSiteMapAuthoringPanels(view, SiteMapAuthoringPanelsOptions{}))
	for _, fragment := range []string{
		`data-studio-shared-component-palette-panel="true"`,
		`data-studio-shared-component-palette-entry="testimonial"`,
		`data-studio-shared-component-place-form="testimonial"`,
		`form="studioSiteMapPlaceInstanceForm-testimonial"`,
		`data-studio-shared-component-instance-count="true"`,
		`Add another Testimonial`,
		`name="gosx_studio_component_template_key" value="testimonial"`,
		`name="gosx_studio_page_key" value="about"`,
	} {
		if !strings.Contains(panelsHTML, fragment) {
			t.Fatalf("rendered shared component palette panel missing %q:\n%s", fragment, panelsHTML)
		}
	}

	formsHTML := gosx.RenderHTML(RenderSiteMapAuthoringForms(view, SiteMapAuthoringFormsOptions{
		Action:    "/admin/editor/__actions/authoring",
		CSRFToken: "csrf-token",
	}))
	for _, fragment := range []string{
		`id="studioSiteMapPlaceInstanceForm-testimonial"`,
		`data-studio-shared-component-place-form="testimonial"`,
		`action="/admin/editor/__actions/authoring"`,
	} {
		if !strings.Contains(formsHTML, fragment) {
			t.Fatalf("rendered shared component place-instance form missing %q:\n%s", fragment, formsHTML)
		}
	}
}

// TestRenderSiteMapSharedComponentPalettePanelAbsentWithoutSharedDefinitions
// proves the new panel adds no markup at all for a library with no reusable
// ComponentDefinitions (existing hosts see zero-byte-identical output).
func TestRenderSiteMapSharedComponentPalettePanelAbsentWithoutSharedDefinitions(t *testing.T) {
	view := SiteMapAuthoringView(core.SiteMap{Pages: []core.Page{{
		Key: "home", Label: "Home", Route: "/", GoSXComponent: "HomePage", Editable: true, Selected: true,
	}}}, SiteMapViewOptions{})
	html := gosx.RenderHTML(RenderSiteMapAuthoringPanels(view, SiteMapAuthoringPanelsOptions{}))
	if strings.Contains(html, "data-studio-shared-component-palette-panel") {
		t.Fatalf("expected no shared component palette panel without any ComponentDefinitions:\n%s", html)
	}
}
