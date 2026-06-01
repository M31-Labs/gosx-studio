package studio

import "testing"

func TestAuthoringSiteMapViewStampsNodePositions(t *testing.T) {
	// A host that persisted node coordinates from gosxstudio:sitemap-node-moved
	// supplies them via SiteMapViewOptions.NodePositions. The matching workspace
	// node map must carry numeric x/y (so the board renders it absolutely);
	// nodes without an entry stay untouched (lane flow).
	surface := NoCodeAuthoringSurface(SiteMap{Pages: []Page{{
		Key:           "home",
		Label:         "Home",
		Route:         "/",
		Group:         PageGroupSite,
		GoSXComponent: "HomePage",
		Editable:      true,
		Selected:      true,
		Components: []Component{{
			Key:           "hero",
			Label:         "Hero",
			GoSXComponent: "HomeHero",
			Source:        ComponentSourceHost,
			Editable:      true,
		}},
	}}})

	view := AuthoringSiteMapView(surface, SiteMapViewOptions{
		NodePositions: map[string]SiteMapNodePosition{
			"page:home": {X: 120, Y: 40},
		},
	})

	positioned, unpositioned := findWorkspaceNode(t, view, "page:home"), findWorkspaceNode(t, view, "component:home:hero")

	x, xok := positioned["x"].(float64)
	y, yok := positioned["y"].(float64)
	if !xok || !yok || x != 120 || y != 40 {
		t.Fatalf("positioned node must carry numeric x=120 y=40, got x=%v(%T) y=%v(%T)", positioned["x"], positioned["x"], positioned["y"], positioned["y"])
	}
	if _, ok := unpositioned["x"]; ok {
		t.Fatalf("node without a saved position must not carry x: %#v", unpositioned)
	}
	if _, ok := unpositioned["y"]; ok {
		t.Fatalf("node without a saved position must not carry y: %#v", unpositioned)
	}
}

func TestAuthoringSiteMapViewWithoutNodePositionsLeavesNodesUnpositioned(t *testing.T) {
	// The default view (no NodePositions) must not stamp any coordinates, so the
	// reference editors keep rendering nodes in lane flow.
	surface := NoCodeAuthoringSurface(SiteMap{Pages: []Page{{
		Key:           "home",
		Label:         "Home",
		Route:         "/",
		Group:         PageGroupSite,
		GoSXComponent: "HomePage",
		Editable:      true,
		Selected:      true,
	}}})

	view := AuthoringSiteMapView(surface, SiteMapViewOptions{})

	node := findWorkspaceNode(t, view, "page:home")
	if _, ok := node["x"]; ok {
		t.Fatalf("default view must not stamp x onto nodes: %#v", node)
	}
	if _, ok := node["y"]; ok {
		t.Fatalf("default view must not stamp y onto nodes: %#v", node)
	}
}

func findWorkspaceNode(t *testing.T, view map[string]any, key string) map[string]any {
	t.Helper()
	layers, ok := view["workspaceLayers"].([]map[string]any)
	if !ok {
		t.Fatalf("view missing workspaceLayers: %#v", view["workspaceLayers"])
	}
	for _, layer := range layers {
		nodes, ok := layer["nodes"].([]map[string]any)
		if !ok {
			continue
		}
		for _, node := range nodes {
			if stringMapValue(node, "key") == key {
				return node
			}
		}
	}
	t.Fatalf("workspace node %q not found in any layer", key)
	return nil
}

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
			TemplateKey:         "hero",
			Label:               "Hero",
			GoSXComponent:       "HomeHero",
			Source:              ComponentSourceHost,
			Binding:             "pages.home.hero",
			Status:              "Visible",
			Editable:            true,
			CanReorder:          true,
			CanDuplicate:        true,
			Visible:             true,
			CanToggleVisibility: true,
			CanDelete:           true,
			Controls: []Control{{
				Key:     "headline",
				Label:   "Headline",
				Kind:    ControlText,
				Binding: "pages.home.hero.headline",
				Value:   "Forest days",
			}},
		}, {
			Key:           "gallery",
			Label:         "Gallery",
			GoSXComponent: "GalleryGrid",
			Source:        ComponentSourceCMS,
			Binding:       "media.gallery",
			Status:        "Visible",
			Editable:      true,
			CanReorder:    true,
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

	if view["pageCountLabel"] != "1 page" || view["componentCountLabel"] != "2 components" || view["controlCountLabel"] != "1 field" {
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
	metadataInputs := metadataPage["formInputs"].([]map[string]string)
	if len(metadataInputs) == 0 || metadataInputs[0]["name"] != AuthoringFieldOperation || metadataInputs[0]["value"] != "update-page" {
		t.Fatalf("expected ordered metadata form inputs: %#v", metadataInputs)
	}
	hero := home["components"].([]map[string]any)[0]
	if hero["selectionKey"] != "home.hero" || hero["sourceSummary"] != "Site-owned editable block" || hero["controlLabel"] != "1 field" {
		t.Fatalf("unexpected hero view: %#v", hero)
	}
	if hero["canToggleVisibility"] != true || hero["visibilityActionLabel"] != "Hide" {
		t.Fatalf("expected hero visibility authoring projection: %#v", hero)
	}
	if hero["canDelete"] != true || hero["deleteAuthoringOperation"] != "delete-component" {
		t.Fatalf("expected hero delete authoring projection: %#v", hero)
	}
	if hero["canDuplicate"] != true || hero["duplicateAuthoringOperation"] != "duplicate-component" || hero["duplicateAuthoringTemplate"] != "hero" {
		t.Fatalf("expected hero duplicate authoring projection: %#v", hero)
	}
	if hero["canReorder"] != true || hero["canMoveUp"] != false || hero["canMoveDown"] != true || hero["positionLabel"] != "1" {
		t.Fatalf("expected hero reorder authoring projection: %#v", hero)
	}
	moveDownInputs := hero["moveDownFormInputs"].([]map[string]string)
	if !authoringFormInputsContain(moveDownInputs, AuthoringFieldPosition, "1") {
		t.Fatalf("expected move-down form inputs: %#v", moveDownInputs)
	}
	reorderPage := view["reorderComponent"].(map[string]any)
	reorderForm := reorderPage["formValues"].(map[string]string)
	if view["hasReorderComponent"] != true || reorderPage["authoringOperation"] != "reorder-component" || reorderForm[AuthoringFieldPosition] != "1" {
		t.Fatalf("unexpected reorder component payload: %#v", reorderPage)
	}
	reorderInputs := reorderPage["formInputs"].([]map[string]string)
	if !authoringFormInputsContain(reorderInputs, AuthoringFieldPosition, "1") {
		t.Fatalf("expected reorder component form inputs: %#v", reorderInputs)
	}
	visibilityPage := view["visibilityComponent"].(map[string]any)
	visibilityForm := visibilityPage["formValues"].(map[string]string)
	if view["hasVisibilityComponent"] != true || visibilityPage["authoringOperation"] != "toggle-visibility" || visibilityForm[AuthoringFieldVisible] != "false" {
		t.Fatalf("unexpected visibility component payload: %#v", visibilityPage)
	}
	visibilityInputs := visibilityPage["formInputs"].([]map[string]string)
	if !authoringFormInputsContain(visibilityInputs, AuthoringFieldVisible, "false") {
		t.Fatalf("expected visibility component form inputs: %#v", visibilityInputs)
	}
	deletePage := view["deleteComponent"].(map[string]any)
	deleteForm := deletePage["formValues"].(map[string]string)
	if view["hasDeleteComponent"] != true || deletePage["authoringOperation"] != "delete-component" || deleteForm[AuthoringFieldComponentKey] != "hero" {
		t.Fatalf("unexpected delete component payload: %#v", deletePage)
	}
	deleteInputs := deletePage["formInputs"].([]map[string]string)
	if !authoringFormInputsContain(deleteInputs, AuthoringFieldComponentKey, "hero") {
		t.Fatalf("expected delete component form inputs: %#v", deleteInputs)
	}
	duplicatePage := view["duplicateComponent"].(map[string]any)
	duplicateForm := duplicatePage["formValues"].(map[string]string)
	if view["hasDuplicateComponent"] != true || duplicatePage["authoringOperation"] != "duplicate-component" || duplicateForm[AuthoringFieldComponentTemplateKey] != "hero" || duplicateForm[AuthoringFieldPosition] != "1" {
		t.Fatalf("unexpected duplicate component payload: %#v", duplicatePage)
	}
	duplicateInputs := duplicatePage["formInputs"].([]map[string]string)
	if !authoringFormInputsContain(duplicateInputs, AuthoringFieldComponentTemplateKey, "hero") {
		t.Fatalf("expected duplicate component form inputs: %#v", duplicateInputs)
	}
	headline := hero["controls"].([]map[string]any)[0]
	if headline["authoringOperation"] != "save-control" || headline["authoringBinding"] != "pages.home.hero.headline" || headline["value"] != "Forest days" {
		t.Fatalf("unexpected control authoring metadata: %#v", headline)
	}
	form := headline["formValues"].(map[string]string)
	if form[AuthoringFieldOperation] != "save-control" || form[AuthoringFieldPageKey] != "home" || form[AuthoringFieldComponentKey] != "hero" {
		t.Fatalf("unexpected control form payload: %#v", form)
	}
	controlInputs := headline["formInputs"].([]map[string]string)
	if !authoringFormInputsContain(controlInputs, AuthoringFieldBinding, "pages.home.hero.headline") {
		t.Fatalf("expected control form inputs: %#v", controlInputs)
	}
	editableControl := view["editableControl"].(map[string]any)
	editableControlForm := editableControl["formValues"].(map[string]string)
	if view["hasEditableControl"] != true || editableControl["key"] != "headline" || editableControl["componentKey"] != "hero" || editableControl["inputName"] != AuthoringFieldValue {
		t.Fatalf("unexpected editable control projection: %#v", editableControl)
	}
	if editableControl["actionLabel"] != "Save field" || editableControlForm[AuthoringFieldBinding] != "pages.home.hero.headline" || editableControlForm[AuthoringFieldValue] != "Forest days" {
		t.Fatalf("unexpected editable control form payload: %#v", editableControl)
	}
	editableControlInputs := editableControl["formInputs"].([]map[string]string)
	if !authoringFormInputsContain(editableControlInputs, AuthoringFieldControlKey, "headline") {
		t.Fatalf("expected editable control form inputs: %#v", editableControlInputs)
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
	if add["kindLabel"] != "Block draft" || add["actionLabel"] != "Add section" || add["formID"] != "studioSiteMapIntentForm-add-component-home-gallery" || add["targetPageKey"] != "home" || add["componentTemplateKey"] != "gallery" {
		t.Fatalf("unexpected add-component intent: %#v", add)
	}
	if add["authoringOperation"] != "apply-intent" || add["authoringIntentKind"] != "add-component" || add["authoringTemplate"] != "gallery" {
		t.Fatalf("unexpected add-component authoring payload: %#v", add)
	}
	addForm := add["formValues"].(map[string]string)
	if addForm[AuthoringFieldOperation] != "apply-intent" || addForm[AuthoringFieldComponentTemplateKey] != "gallery" {
		t.Fatalf("unexpected add intent form: %#v", addForm)
	}
	addInputs := add["formInputs"].([]map[string]string)
	if len(addInputs) == 0 || addInputs[0]["name"] != AuthoringFieldOperation || addInputs[0]["value"] != "apply-intent" {
		t.Fatalf("expected ordered add intent form inputs: %#v", addInputs)
	}
	if !authoringFormInputsContain(addInputs, AuthoringFieldComponentTemplateKey, "gallery") {
		t.Fatalf("expected add intent template input: %#v", addInputs)
	}
	workspaceLayers := view["workspaceLayers"].([]map[string]any)
	if len(workspaceLayers) == 0 || workspaceLayers[0]["hasPageComposition"] != true || workspaceLayers[0]["sectionCountLabel"] != "2 sections" {
		t.Fatalf("unexpected workspace layer: %#v", workspaceLayers)
	}
	if view["hasWorkspaceCanvasLinks"] != true || view["workspaceCanvasViewBox"] != "0 0 100 100" {
		t.Fatalf("unexpected workspace canvas: %#v", view)
	}
}
