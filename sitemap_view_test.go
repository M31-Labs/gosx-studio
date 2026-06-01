package studio

import "testing"

func TestSiteMapAuthoringViewProjectsInteractiveEditorPayload(t *testing.T) {
	siteMap := SiteMap{Pages: []Page{{
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
			Label:               "Hero",
			GoSXComponent:       "HomeHero",
			Source:              ComponentSourceHost,
			Binding:             "pages.home.hero",
			Status:              "Visible",
			Editable:            true,
			Visible:             true,
			CanToggleVisibility: true,
			Controls: []Control{{
				Key:     "headline",
				Label:   "Headline",
				Kind:    ControlText,
				Binding: "pages.home.hero.headline",
				Value:   "Forest days",
			}},
		}},
	}}, Library: CompositionLibrary{
		PageBlueprints: []PageBlueprint{{
			Key:           "landing",
			Label:         "Landing page",
			RoutePattern:  "/landing",
			Group:         PageGroupContent,
			GoSXComponent: "LandingPage",
			Status:        "Ready",
		}},
		ComponentTemplates: []ComponentTemplate{{
			Key:            "gallery",
			Label:          "Gallery",
			Category:       "Media",
			GoSXComponent:  "GalleryGrid",
			Source:         ComponentSourceCMS,
			DefaultBinding: "media.gallery",
			Status:         "Draft",
			AddLabel:       "Add gallery",
			Controls: []Control{{
				Key:     "source",
				Label:   "Source",
				Kind:    ControlSource,
				Binding: "media.gallery",
			}},
		}},
	}}

	view := SiteMapAuthoringView(siteMap, SiteMapViewOptions{
		PreviewHref: func(route string) string {
			return "/admin/preview?path=" + route
		},
	})

	if view["pageCountLabel"] != "1 page" || view["componentCountLabel"] != "1 component" || view["controlCountLabel"] != "1 field" {
		t.Fatalf("unexpected labels: %#v", view)
	}
	if view["selectedPageLabel"] != "Home /" || view["defaultWorkspaceNodeKey"] != "component:home:hero" {
		t.Fatalf("unexpected selection summary: %#v", view)
	}
	pages := view["pages"].([]map[string]any)
	home := pages[0]
	if home["href"] != "/admin/preview?path=/" || home["typeLabel"] != "Site page" || home["class"] != "studio-site-map-page is-selected" {
		t.Fatalf("unexpected home page view: %#v", home)
	}
	if home["editable"] != true || view["hasMetadataPage"] != true {
		t.Fatalf("expected editable page metadata projection: %#v", home)
	}
	metadataPage := view["metadataPage"].(map[string]any)
	metadataForm := metadataPage["formValues"].(map[string]string)
	if metadataPage["authoringOperation"] != "update-page" || metadataForm[AuthoringFieldPageKey] != "home" {
		t.Fatalf("unexpected metadata page authoring payload: %#v", metadataPage)
	}
	hero := home["components"].([]map[string]any)[0]
	if hero["selectionKey"] != "home.hero" || hero["sourceSummary"] != "Site-owned editable block" || hero["controlLabel"] != "1 field" {
		t.Fatalf("unexpected hero view: %#v", hero)
	}
	if hero["canToggleVisibility"] != true || hero["visibilityActionLabel"] != "Hide" {
		t.Fatalf("expected hero visibility authoring projection: %#v", hero)
	}
	visibilityPage := view["visibilityComponent"].(map[string]any)
	visibilityForm := visibilityPage["formValues"].(map[string]string)
	if view["hasVisibilityComponent"] != true || visibilityPage["authoringOperation"] != "toggle-visibility" || visibilityForm[AuthoringFieldVisible] != "false" {
		t.Fatalf("unexpected visibility component payload: %#v", visibilityPage)
	}
	headline := hero["controls"].([]map[string]any)[0]
	if headline["authoringOperation"] != "save-control" || headline["authoringBinding"] != "pages.home.hero.headline" || headline["value"] != "Forest days" {
		t.Fatalf("unexpected control authoring metadata: %#v", headline)
	}
	form := headline["formValues"].(map[string]string)
	if form[AuthoringFieldOperation] != "save-control" || form[AuthoringFieldPageKey] != "home" || form[AuthoringFieldComponentKey] != "hero" {
		t.Fatalf("unexpected control form payload: %#v", form)
	}
	blueprints := view["blueprints"].([]map[string]any)
	if blueprints[0]["inputName"] != "studioPageBlueprintIntent" || blueprints[0]["inputID"] != "studioSiteMapBlueprint-landing" {
		t.Fatalf("unexpected blueprint payload: %#v", blueprints[0])
	}
	palette := view["palette"].([]map[string]any)
	if palette[0]["categoryKey"] != "media" || palette[0]["addLabel"] != "Add gallery" || palette[0]["controlLabel"] != "1 field" {
		t.Fatalf("unexpected palette payload: %#v", palette[0])
	}
	intents := view["compositionIntents"].([]map[string]any)
	if len(intents) != 2 {
		t.Fatalf("expected create/add intents, got %#v", intents)
	}
	add := intents[1]
	if add["kindLabel"] != "Block draft" || add["targetPageKey"] != "home" || add["componentTemplateKey"] != "gallery" {
		t.Fatalf("unexpected add-component intent: %#v", add)
	}
	addForm := add["formValues"].(map[string]string)
	if addForm[AuthoringFieldOperation] != "apply-intent" || addForm[AuthoringFieldComponentTemplateKey] != "gallery" {
		t.Fatalf("unexpected add intent form: %#v", addForm)
	}
	workspaceLayers := view["workspaceLayers"].([]map[string]any)
	if len(workspaceLayers) == 0 || workspaceLayers[0]["hasPageComposition"] != true || workspaceLayers[0]["sectionCountLabel"] != "1 section" {
		t.Fatalf("unexpected workspace layer: %#v", workspaceLayers)
	}
	if view["hasWorkspaceCanvasLinks"] != true || view["workspaceCanvasViewBox"] != "0 0 100 100" {
		t.Fatalf("unexpected workspace canvas: %#v", view)
	}
}
