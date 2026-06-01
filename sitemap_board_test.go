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
		`data-studio-site-map-scale="1"`,
		`data-studio-site-map-pan-x="0"`,
		`data-studio-site-map-pan-surface="true"`,
		`data-studio-site-map-zoom-action="in"`,
		`data-studio-site-map-zoom-action="reset"`,
		`data-studio-site-map-zoom-readout`,
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

func TestRenderSiteMapBoardPanSurfaceIsPositionedAnchor(t *testing.T) {
	view := AuthoringSiteMapView(NoCodeAuthoringSurface(SiteMap{
		Pages: []Page{{
			Key:      "home",
			Label:    "Home",
			Route:    "/",
			Group:    PageGroupSite,
			Editable: true,
			Selected: true,
		}},
	}), SiteMapViewOptions{})

	html := gosx.RenderHTML(RenderSiteMapBoard(view, SiteMapBoardOptions{}))

	// The pan surface must be a positioned ancestor so absolutely-positioned
	// nodes anchor to it (drag-to-reposition). Default nodes still flow in
	// lanes, so this style addition must be layout-neutral.
	if !strings.Contains(html, `data-studio-site-map-pan-surface="true" style="position:relative"`) {
		t.Fatalf("pan surface must carry position:relative so nodes can anchor:\n%s", html)
	}
}

func TestRenderSiteMapBoardWorkspaceNodeWithoutPositionStaysInLaneFlow(t *testing.T) {
	// A node map WITHOUT x/y keys must render exactly as today: no absolute
	// position, no node-x/node-y attrs. This keeps the default
	// AuthoringSiteMapView (and the reference editors) visually unchanged.
	node := map[string]any{
		"key":   "page:home",
		"label": "Home",
		"class": "studio-site-map-workspace-node",
	}
	html := gosx.RenderHTML(gosx.Fragment(renderSiteMapBoardWorkspaceNode(node, siteMapBoardState{})...))

	if strings.Contains(html, "data-studio-site-map-node-x") {
		t.Fatalf("node without a saved position must not emit data-studio-site-map-node-x:\n%s", html)
	}
	if strings.Contains(html, "data-studio-site-map-node-y") {
		t.Fatalf("node without a saved position must not emit data-studio-site-map-node-y:\n%s", html)
	}
	if strings.Contains(html, "position:absolute") {
		t.Fatalf("node without a saved position must stay in lane flow (no absolute position):\n%s", html)
	}
}

func TestRenderSiteMapBoardWorkspaceNodeWithPositionRendersAbsolute(t *testing.T) {
	// A node map carrying numeric x/y keys must be rendered absolutely
	// positioned on the pan surface, with the coordinate data attributes the
	// runtime reads/writes during drag-to-reposition.
	node := map[string]any{
		"key":   "page:home",
		"label": "Home",
		"class": "studio-site-map-workspace-node",
		"x":     120.0,
		"y":     48.0,
	}
	html := gosx.RenderHTML(gosx.Fragment(renderSiteMapBoardWorkspaceNode(node, siteMapBoardState{})...))

	for _, fragment := range []string{
		`data-studio-site-map-node-x="120"`,
		`data-studio-site-map-node-y="48"`,
		`style="position:absolute;left:120px;top:48px"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("positioned node missing %q:\n%s", fragment, html)
		}
	}
}
