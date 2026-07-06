package sitemap

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

func TestRenderSiteMapBoardUsesSharedRuntimeContracts(t *testing.T) {
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
				Category:       "Page sections",
				Controls: []core.Control{{
					Key:   "caption",
					Label: "Caption",
					Kind:  core.ControlText,
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
	view := AuthoringSiteMapView(authoring.NoCodeAuthoringSurface(core.SiteMap{
		Pages: []core.Page{{
			Key:      "home",
			Label:    "Home",
			Route:    "/",
			Group:    core.PageGroupSite,
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

func TestRenderSiteMapBoardRoundTripsNodePositions(t *testing.T) {
	// Closing the loop with the runtime's gosxstudio:sitemap-node-moved event:
	// the {key,x,y} a host stores are pan-surface-LOCAL pixels, and feeding the
	// same {x,y} back through SiteMapViewOptions.NodePositions must reproduce the
	// exact absolute style and node-x/-y the runtime reads (1:1 round-trip).
	surface := authoring.NoCodeAuthoringSurface(core.SiteMap{Pages: []core.Page{{
		Key:           "home",
		Label:         "Home",
		Route:         "/",
		Group:         core.PageGroupSite,
		GoSXComponent: "HomePage",
		Editable:      true,
		Selected:      true,
	}}})

	view := AuthoringSiteMapView(surface, SiteMapViewOptions{
		NodePositions: map[string]SiteMapNodePosition{
			"page:home": {X: 120, Y: 40},
		},
	})
	html := gosx.RenderHTML(RenderSiteMapBoard(view, SiteMapBoardOptions{}))

	for _, fragment := range []string{
		`data-studio-site-map-node-x="120"`,
		`data-studio-site-map-node-y="40"`,
		`style="position:absolute;left:120px;top:40px"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("positioned site-map board missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderSiteMapBoardWithoutNodePositionsStaysInLaneFlow(t *testing.T) {
	// The default render (no NodePositions) must be layout-neutral: no absolute
	// positioning and no node-x attrs anywhere, so the reference editors are
	// visually unchanged.
	surface := authoring.NoCodeAuthoringSurface(core.SiteMap{Pages: []core.Page{{
		Key:           "home",
		Label:         "Home",
		Route:         "/",
		Group:         core.PageGroupSite,
		GoSXComponent: "HomePage",
		Editable:      true,
		Selected:      true,
	}}})

	view := AuthoringSiteMapView(surface, SiteMapViewOptions{})
	html := gosx.RenderHTML(RenderSiteMapBoard(view, SiteMapBoardOptions{}))

	if !strings.Contains(html, `data-studio-site-map-workspace-node="page:home"`) {
		t.Fatalf("default render should still render the workspace node:\n%s", html)
	}
	if strings.Contains(html, "position:absolute") {
		t.Fatalf("default site-map board must not position nodes absolutely:\n%s", html)
	}
	if strings.Contains(html, "data-studio-site-map-node-x") {
		t.Fatalf("default site-map board must not emit data-studio-site-map-node-x:\n%s", html)
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
