package studio

import (
	"strconv"
	"strings"
)

type SiteMapViewOptions struct {
	Kicker                string
	Title                 string
	Summary               string
	DefaultGroupKey       string
	DefaultDetailKey      string
	BlueprintInputName    string
	TemplateInputName     string
	WorkspaceInputName    string
	PreviewHref           func(route string) string
	CompositionIntentView func(CompositionIntent) map[string]any
	// NodePositions carries free-placed workspace node coordinates keyed by node
	// key (e.g. "page:home", "component:home:hero"). A host persists these from
	// the runtime's gosxstudio:sitemap-node-moved event (detail {key,x,y}) and
	// feeds them back so dragged nodes re-render where they were left. The
	// coordinates are pan-surface-LOCAL pixels and round-trip 1:1 with that
	// event. Nodes without an entry stay in lane flow (the default).
	NodePositions map[string]SiteMapNodePosition
}

// SiteMapNodePosition is a free-placed workspace node's saved location on the
// pan surface, in pan-surface-LOCAL pixels (the same coordinate space the
// runtime emits in gosxstudio:sitemap-node-moved and writes back into
// data-studio-site-map-node-x/-y).
type SiteMapNodePosition struct {
	X float64
	Y float64
}

func SiteMapAuthoringView(siteMap SiteMap, options SiteMapViewOptions) map[string]any {
	return AuthoringSiteMapView(NoCodeAuthoringSurface(siteMap), options)
}

func AuthoringSiteMapView(surface AuthoringSurface, options SiteMapViewOptions) map[string]any {
	siteMap := surface.SiteMap.Normalize()
	workspace := surface.Workspace.Normalize()
	if workspace.NodeCount() == 0 && len(siteMap.Pages) > 0 {
		workspace = siteMap.CompositionWorkspace()
	}
	canvas := surface.Canvas
	if canvas.ViewBox == "" && (len(canvas.Nodes) == 0 || len(canvas.Links) == 0) {
		canvas = workspace.CanvasLayout()
	}
	selectedPage, hasSelectedPage := siteMap.SelectedPage()
	if surface.HasSelectedPage {
		selectedPage = surface.SelectedPage.Normalize()
		hasSelectedPage = true
	}
	selectedWorkspaceNode, hasSelectedWorkspaceNode := authoringDefaultWorkspaceNodeView(workspace)
	defaultFocusPageKey := stringMapValue(selectedWorkspaceNode, "pageKey")
	if defaultFocusPageKey == "" && hasSelectedPage {
		defaultFocusPageKey = selectedPage.Key
	}
	defaultFocusKey := defaultFocusPageKey
	if defaultFocusKey == "" {
		defaultFocusKey = "all"
	}
	pages := authoringSiteMapPageViews(siteMap.Pages, options)
	metadataPage := authoringSiteMapMetadataPageView(siteMap.Pages)
	reorderComponent := authoringSiteMapReorderComponentView(siteMap.Pages)
	duplicateComponent := authoringSiteMapDuplicateComponentView(siteMap.Pages)
	visibilityComponent := authoringSiteMapVisibilityComponentView(siteMap.Pages)
	deleteComponent := authoringSiteMapDeleteComponentView(siteMap.Pages)
	editableControl := authoringSiteMapEditableControlView(siteMap.Pages)
	sources := authoringSiteMapSourceViews(siteMap)
	blueprints := authoringSiteMapPageBlueprintViews(siteMap.Library.PageBlueprints, options)
	palette := authoringSiteMapComponentTemplateViews(siteMap.Library.ComponentTemplates, options)
	compositionIntents := authoringSiteMapIntentViews(surface.Intents(), options)
	activeIntentCount := 0
	if len(blueprints) > 0 {
		activeIntentCount++
	}
	if hasSelectedPage && len(palette) > 0 {
		activeIntentCount++
	}
	return map[string]any{
		"kicker":                             firstNonEmpty(options.Kicker, "Site map"),
		"title":                              firstNonEmpty(options.Title, "Pages"),
		"summary":                            firstNonEmpty(options.Summary, "Editable pages composed from sections, content, flows, and plugins."),
		"defaultGroupKey":                    firstNonEmpty(options.DefaultGroupKey, "all"),
		"defaultDetailKey":                   firstNonEmpty(options.DefaultDetailKey, "map"),
		"pageCountLabel":                     countLabel(len(pages), "page", "pages"),
		"componentCountLabel":                countLabel(siteMap.ComponentCount(), "component", "components"),
		"controlCountLabel":                  countLabel(siteMap.ControlCount(), "field", "fields"),
		"workspaceNodeCountLabel":            countLabel(workspace.NodeCount(), "node", "nodes"),
		"workspaceLinkCountLabel":            countLabel(workspace.LinkCount(), "connection", "connections"),
		"workspaceLayerCountLabel":           countLabel(workspace.LayerCount(), "layer", "layers"),
		"blueprintCountLabel":                countLabel(len(blueprints), "blueprint", "blueprints"),
		"paletteCountLabel":                  countLabel(len(palette), "building block", "building blocks"),
		"compositionIntentCountLabel":        countLabel(len(compositionIntents), "draft", "drafts"),
		"activeCompositionIntentCountLabel":  countLabel(activeIntentCount, "draft", "drafts"),
		"selectedPageLabel":                  authoringSelectedPageLabel(selectedPage, hasSelectedPage),
		"hasSelectedPage":                    hasSelectedPage,
		"defaultFocusKey":                    defaultFocusKey,
		"defaultBlueprintKey":                authoringFirstItemKey(blueprints),
		"defaultTemplateKey":                 authoringFirstItemKey(palette),
		"hasDefaultWorkspaceNode":            hasSelectedWorkspaceNode,
		"defaultWorkspaceNodeKey":            stringMapValue(selectedWorkspaceNode, "key"),
		"defaultWorkspaceNodeLabel":          stringMapValue(selectedWorkspaceNode, "label"),
		"defaultWorkspaceNodeKind":           stringMapValue(selectedWorkspaceNode, "kindLabel"),
		"defaultWorkspaceNodeSummary":        stringMapValue(selectedWorkspaceNode, "summary"),
		"defaultWorkspaceNodeRoute":          stringMapValue(selectedWorkspaceNode, "route"),
		"defaultWorkspaceNodeSource":         stringMapValue(selectedWorkspaceNode, "sourceLabel"),
		"defaultWorkspaceNodeBinding":        stringMapValue(selectedWorkspaceNode, "bindingLabel"),
		"defaultWorkspaceNodeStatus":         stringMapValue(selectedWorkspaceNode, "statusLabel"),
		"defaultWorkspaceNodeGoSX":           stringMapValue(selectedWorkspaceNode, "gosxComponent"),
		"defaultWorkspaceNodeComponentLabel": stringMapValue(selectedWorkspaceNode, "componentLabel"),
		"pages":                              pages,
		"hasPages":                           len(pages) > 0,
		"metadataPage":                       metadataPage,
		"hasMetadataPage":                    len(metadataPage) > 0,
		"reorderComponent":                   reorderComponent,
		"hasReorderComponent":                len(reorderComponent) > 0,
		"duplicateComponent":                 duplicateComponent,
		"hasDuplicateComponent":              len(duplicateComponent) > 0,
		"visibilityComponent":                visibilityComponent,
		"hasVisibilityComponent":             len(visibilityComponent) > 0,
		"deleteComponent":                    deleteComponent,
		"hasDeleteComponent":                 len(deleteComponent) > 0,
		"editableControl":                    editableControl,
		"hasEditableControl":                 len(editableControl) > 0,
		"empty":                              "No editable pages are configured.",
		"sources":                            sources,
		"hasSources":                         len(sources) > 0,
		"workspaceLayers":                    authoringSiteMapWorkspaceLayerViews(siteMap, workspace, options.NodePositions),
		"workspaceLinks":                     authoringSiteMapWorkspaceLinkViews(workspace),
		"workspaceCanvasViewBox":             canvas.ViewBox,
		"workspaceCanvasLinks":               authoringSiteMapWorkspaceCanvasLinkViews(workspace, canvas),
		"hasWorkspaceCanvasLinks":            len(canvas.Links) > 0,
		"hasWorkspace":                       workspace.NodeCount() > 0,
		"hasWorkspaceLinks":                  workspace.LinkCount() > 0,
		"blueprints":                         blueprints,
		"hasBlueprints":                      len(blueprints) > 0,
		"palette":                            palette,
		"hasPalette":                         len(palette) > 0,
		"compositionIntents":                 compositionIntents,
		"hasCompositionIntents":              len(compositionIntents) > 0,
		"authoring":                          AuthoringSurfaceView(surface),
	}
}

func authoringSiteMapPageViews(pages []Page, options SiteMapViewOptions) []map[string]any {
	out := make([]map[string]any, 0, len(pages))
	for _, page := range pages {
		page = page.Normalize()
		componentCount := page.ComponentCount()
		className := "studio-site-map-page"
		if page.Selected {
			className += " is-selected"
		}
		out = append(out, map[string]any{
			"key":                 page.Key,
			"label":               page.Label,
			"route":               page.Route,
			"href":                authoringPreviewHref(page.Route, options),
			"group":               string(page.NormalizedGroup()),
			"groupLabel":          page.GroupLabel(),
			"typeLabel":           page.GroupLabel() + " page",
			"statusLabel":         page.Status,
			"gosxComponent":       page.GoSXComponent,
			"goSXComponent":       page.GoSXComponent,
			"editable":            page.Editable,
			"componentCountLabel": countLabel(componentCount, "component", "components"),
			"components":          authoringSiteMapComponentViews(page, page.Components),
			"hasComponents":       componentCount > 0,
			"controlCountLabel":   countLabel(page.ControlCount(), "field", "fields"),
			"class":               className,
			"selected":            page.Selected,
		})
	}
	return out
}

func authoringSiteMapMetadataPageView(pages []Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		if !page.Editable {
			continue
		}
		mutation := AuthoringMutationForPage(page)
		return map[string]any{
			"key":                page.Key,
			"label":              page.Label,
			"route":              page.Route,
			"groupLabel":         page.GroupLabel(),
			"statusLabel":        page.Status,
			"authoringOperation": string(mutation.Kind),
			"authoringPageKey":   mutation.PageKey,
			"authoringPageLabel": mutation.PageLabel,
			"authoringPageRoute": mutation.PageRoute,
			"mutation":           AuthoringMutationView(mutation),
			"formValues":         mutation.FormValues(),
			"formInputs":         AuthoringMutationFormInputViews(mutation),
		}
	}
	return nil
}

func authoringSiteMapComponentViews(page Page, components []Component) []map[string]any {
	out := make([]map[string]any, 0, len(components))
	page = page.Normalize()
	for index, component := range components {
		component = component.Normalize()
		component.Position = index
		moveUpMutation := AuthoringMutation{}
		moveDownMutation := AuthoringMutation{}
		moveUpFormValues := map[string]string(nil)
		moveDownFormValues := map[string]string(nil)
		moveUpFormInputs := []map[string]string(nil)
		moveDownFormInputs := []map[string]string(nil)
		canMoveUp := component.CanReorder && index > 0
		canMoveDown := component.CanReorder && index < len(components)-1
		if canMoveUp {
			moveUpMutation = AuthoringMutationForComponentReorder(page, component, index-1)
			moveUpFormValues = moveUpMutation.FormValues()
			moveUpFormInputs = AuthoringMutationFormInputViews(moveUpMutation)
		}
		if canMoveDown {
			moveDownMutation = AuthoringMutationForComponentReorder(page, component, index+1)
			moveDownFormValues = moveDownMutation.FormValues()
			moveDownFormInputs = AuthoringMutationFormInputViews(moveDownMutation)
		}
		visibilityMutation := AuthoringMutation{}
		visibilityFormValues := map[string]string(nil)
		visibilityFormInputs := []map[string]string(nil)
		visibilityActionLabel := ""
		if component.CanToggleVisibility {
			visibilityMutation = AuthoringMutationForComponentVisibility(page, component, !component.Visible)
			visibilityFormValues = visibilityMutation.FormValues()
			visibilityFormInputs = AuthoringMutationFormInputViews(visibilityMutation)
			visibilityActionLabel = "Show"
			if component.Visible {
				visibilityActionLabel = "Hide"
			}
		}
		deleteMutation := AuthoringMutation{}
		deleteFormValues := map[string]string(nil)
		deleteFormInputs := []map[string]string(nil)
		if component.CanDelete {
			deleteMutation = AuthoringMutationForComponentDelete(page, component)
			deleteFormValues = deleteMutation.FormValues()
			deleteFormInputs = AuthoringMutationFormInputViews(deleteMutation)
		}
		duplicateMutation := AuthoringMutation{}
		duplicateFormValues := map[string]string(nil)
		duplicateFormInputs := []map[string]string(nil)
		if component.CanDuplicate {
			duplicateMutation = AuthoringMutationForComponentDuplicate(page, component, index+1)
			duplicateFormValues = duplicateMutation.FormValues()
			duplicateFormInputs = AuthoringMutationFormInputViews(duplicateMutation)
		}
		out = append(out, map[string]any{
			"key":                          component.Key,
			"templateKey":                  component.TemplateKey,
			"selectionKey":                 component.SelectionKey(page.Key),
			"label":                        component.Label,
			"summary":                      firstNonEmpty(component.Summary, authoringComponentDefaultSummary(component)),
			"gosxComponent":                component.GoSXComponent,
			"goSXComponent":                component.GoSXComponent,
			"source":                       string(component.NormalizedSource()),
			"sourceLabel":                  ComponentSourceLabel(component.Source),
			"sourceSummary":                authoringComponentSourceSummary(component),
			"binding":                      component.Binding,
			"statusLabel":                  component.Status,
			"editable":                     component.Editable,
			"position":                     index,
			"positionLabel":                strconv.Itoa(index + 1),
			"canReorder":                   component.CanReorder,
			"canDuplicate":                 component.CanDuplicate,
			"canMoveUp":                    canMoveUp,
			"canMoveDown":                  canMoveDown,
			"moveUpActionLabel":            "Move up",
			"moveDownActionLabel":          "Move down",
			"moveUpAuthoringOperation":     string(moveUpMutation.Kind),
			"moveUpAuthoringPageKey":       moveUpMutation.PageKey,
			"moveUpAuthoringComponent":     moveUpMutation.ComponentKey,
			"moveUpAuthoringPosition":      moveUpMutation.Position,
			"moveUpMutation":               AuthoringMutationView(moveUpMutation),
			"moveUpFormValues":             moveUpFormValues,
			"moveUpFormInputs":             moveUpFormInputs,
			"moveDownAuthoringOperation":   string(moveDownMutation.Kind),
			"moveDownAuthoringPageKey":     moveDownMutation.PageKey,
			"moveDownAuthoringComponent":   moveDownMutation.ComponentKey,
			"moveDownAuthoringPosition":    moveDownMutation.Position,
			"moveDownMutation":             AuthoringMutationView(moveDownMutation),
			"moveDownFormValues":           moveDownFormValues,
			"moveDownFormInputs":           moveDownFormInputs,
			"visible":                      component.Visible,
			"canToggleVisibility":          component.CanToggleVisibility,
			"visibilityActionLabel":        visibilityActionLabel,
			"visibilityAuthoringOperation": string(visibilityMutation.Kind),
			"visibilityAuthoringPageKey":   visibilityMutation.PageKey,
			"visibilityAuthoringComponent": visibilityMutation.ComponentKey,
			"visibilityAuthoringVisible":   visibilityMutation.Visible,
			"visibilityMutation":           AuthoringMutationView(visibilityMutation),
			"visibilityFormValues":         visibilityFormValues,
			"visibilityFormInputs":         visibilityFormInputs,
			"canDelete":                    component.CanDelete,
			"deleteActionLabel":            "Delete",
			"deleteAuthoringOperation":     string(deleteMutation.Kind),
			"deleteAuthoringPageKey":       deleteMutation.PageKey,
			"deleteAuthoringComponent":     deleteMutation.ComponentKey,
			"deleteMutation":               AuthoringMutationView(deleteMutation),
			"deleteFormValues":             deleteFormValues,
			"deleteFormInputs":             deleteFormInputs,
			"duplicateActionLabel":         "Duplicate",
			"duplicateAuthoringOperation":  string(duplicateMutation.Kind),
			"duplicateAuthoringPageKey":    duplicateMutation.PageKey,
			"duplicateAuthoringComponent":  duplicateMutation.ComponentKey,
			"duplicateAuthoringTemplate":   duplicateMutation.ComponentTemplateKey,
			"duplicateAuthoringPosition":   duplicateMutation.Position,
			"duplicateMutation":            AuthoringMutationView(duplicateMutation),
			"duplicateFormValues":          duplicateFormValues,
			"duplicateFormInputs":          duplicateFormInputs,
			"controls":                     authoringSiteMapControlViews(page, component, component.Controls),
			"hasControls":                  component.ControlCount() > 0,
			"controlLabel":                 countLabel(component.ControlCount(), "field", "fields"),
		})
	}
	return out
}

func authoringSiteMapReorderComponentView(pages []Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for index, component := range page.Components {
			component = component.Normalize()
			component.Position = index
			if !component.CanReorder {
				continue
			}
			targetPosition := index + 1
			actionLabel := "Move down"
			if targetPosition >= len(page.Components) {
				if index == 0 {
					continue
				}
				targetPosition = index - 1
				actionLabel = "Move up"
			}
			mutation := AuthoringMutationForComponentReorder(page, component, targetPosition)
			return map[string]any{
				"key":                 component.Key,
				"label":               component.Label,
				"pageKey":             page.Key,
				"pageLabel":           page.Label,
				"binding":             component.Binding,
				"statusLabel":         component.Status,
				"position":            index,
				"positionLabel":       strconv.Itoa(index + 1),
				"targetPosition":      targetPosition,
				"targetPositionLabel": strconv.Itoa(targetPosition + 1),
				"actionLabel":         actionLabel,
				"authoringOperation":  string(mutation.Kind),
				"authoringPageKey":    mutation.PageKey,
				"authoringComponent":  mutation.ComponentKey,
				"authoringPosition":   mutation.Position,
				"mutation":            AuthoringMutationView(mutation),
				"formValues":          mutation.FormValues(),
				"formInputs":          AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func authoringSiteMapDuplicateComponentView(pages []Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for index, component := range page.Components {
			component = component.Normalize()
			component.Position = index
			if !component.CanDuplicate {
				continue
			}
			mutation := AuthoringMutationForComponentDuplicate(page, component, index+1)
			return map[string]any{
				"key":                component.Key,
				"templateKey":        component.TemplateKey,
				"label":              component.Label,
				"pageKey":            page.Key,
				"pageLabel":          page.Label,
				"binding":            component.Binding,
				"statusLabel":        component.Status,
				"position":           index,
				"positionLabel":      strconv.Itoa(index + 1),
				"targetPosition":     mutation.Position,
				"actionLabel":        "Duplicate",
				"authoringOperation": string(mutation.Kind),
				"authoringPageKey":   mutation.PageKey,
				"authoringComponent": mutation.ComponentKey,
				"authoringTemplate":  mutation.ComponentTemplateKey,
				"authoringPosition":  mutation.Position,
				"mutation":           AuthoringMutationView(mutation),
				"formValues":         mutation.FormValues(),
				"formInputs":         AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func authoringSiteMapVisibilityComponentView(pages []Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for _, component := range page.Components {
			component = component.Normalize()
			if !component.CanToggleVisibility {
				continue
			}
			mutation := AuthoringMutationForComponentVisibility(page, component, !component.Visible)
			actionLabel := "Show"
			if component.Visible {
				actionLabel = "Hide"
			}
			return map[string]any{
				"key":                component.Key,
				"label":              component.Label,
				"pageKey":            page.Key,
				"pageLabel":          page.Label,
				"binding":            component.Binding,
				"statusLabel":        component.Status,
				"visible":            component.Visible,
				"nextVisible":        mutation.Visible,
				"actionLabel":        actionLabel,
				"authoringOperation": string(mutation.Kind),
				"authoringPageKey":   mutation.PageKey,
				"authoringComponent": mutation.ComponentKey,
				"authoringVisible":   mutation.Visible,
				"mutation":           AuthoringMutationView(mutation),
				"formValues":         mutation.FormValues(),
				"formInputs":         AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func authoringSiteMapDeleteComponentView(pages []Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for index, component := range page.Components {
			component = component.Normalize()
			component.Position = index
			if !component.CanDelete {
				continue
			}
			mutation := AuthoringMutationForComponentDelete(page, component)
			return map[string]any{
				"key":                component.Key,
				"label":              component.Label,
				"pageKey":            page.Key,
				"pageLabel":          page.Label,
				"binding":            component.Binding,
				"statusLabel":        component.Status,
				"position":           index,
				"positionLabel":      strconv.Itoa(index + 1),
				"actionLabel":        "Delete",
				"authoringOperation": string(mutation.Kind),
				"authoringPageKey":   mutation.PageKey,
				"authoringComponent": mutation.ComponentKey,
				"mutation":           AuthoringMutationView(mutation),
				"formValues":         mutation.FormValues(),
				"formInputs":         AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func authoringSiteMapEditableControlView(pages []Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for _, component := range page.Components {
			component = component.Normalize()
			if !component.Editable {
				continue
			}
			for _, control := range component.Controls {
				control = control.Normalize()
				if control.Key == "" || control.Label == "" {
					continue
				}
				mutation := AuthoringMutationForControl(page, component, control)
				view := authoringSiteMapControlView(control)
				view["pageKey"] = page.Key
				view["pageLabel"] = page.Label
				view["pageRoute"] = page.Route
				view["componentKey"] = component.Key
				view["componentLabel"] = component.Label
				view["componentBinding"] = component.Binding
				view["selectionKey"] = component.SelectionKey(page.Key)
				view["inputName"] = AuthoringFieldValue
				view["actionLabel"] = "Save field"
				view["authoringOperation"] = string(mutation.Kind)
				view["authoringPageKey"] = mutation.PageKey
				view["authoringComponent"] = mutation.ComponentKey
				view["authoringControl"] = mutation.ControlKey
				view["authoringBinding"] = mutation.Binding
				view["authoringControlKind"] = string(mutation.ControlKind)
				view["mutation"] = AuthoringMutationView(mutation)
				view["formValues"] = mutation.FormValues()
				view["formInputs"] = AuthoringMutationFormInputViews(mutation)
				return view
			}
		}
	}
	return nil
}

func authoringSiteMapControlViews(page Page, component Component, controls []Control) []map[string]any {
	out := make([]map[string]any, 0, len(controls))
	for _, control := range controls {
		control = control.Normalize()
		if control.Key == "" || control.Label == "" {
			continue
		}
		mutation := AuthoringMutationForControl(page, component, control)
		view := authoringSiteMapControlView(control)
		view["inputName"] = AuthoringFieldValue
		view["authoringOperation"] = string(mutation.Kind)
		view["authoringPageKey"] = mutation.PageKey
		view["authoringComponent"] = mutation.ComponentKey
		view["authoringControl"] = mutation.ControlKey
		view["authoringBinding"] = mutation.Binding
		view["authoringControlKind"] = string(mutation.ControlKind)
		view["mutation"] = AuthoringMutationView(mutation)
		view["formValues"] = mutation.FormValues()
		view["formInputs"] = AuthoringMutationFormInputViews(mutation)
		out = append(out, view)
	}
	return out
}

func authoringSiteMapStaticControlViews(controls []Control) []map[string]any {
	out := make([]map[string]any, 0, len(controls))
	for _, control := range controls {
		control = control.Normalize()
		if control.Key == "" || control.Label == "" {
			continue
		}
		out = append(out, authoringSiteMapControlView(control))
	}
	return out
}

func authoringSiteMapControlView(control Control) map[string]any {
	return map[string]any{
		"key":       control.Key,
		"label":     control.Label,
		"kind":      string(control.NormalizedKind()),
		"kindLabel": control.KindLabel(),
		"binding":   control.Binding,
		"value":     control.Value,
		"advanced":  control.Advanced,
		"required":  control.Required,
	}
}

func authoringSiteMapPageBlueprintViews(blueprints []PageBlueprint, options SiteMapViewOptions) []map[string]any {
	out := make([]map[string]any, 0, len(blueprints))
	for index, blueprint := range blueprints {
		blueprint = blueprint.Normalize()
		out = append(out, map[string]any{
			"key":                 blueprint.Key,
			"label":               blueprint.Label,
			"summary":             blueprint.Summary,
			"routePattern":        blueprint.RoutePattern,
			"group":               string(blueprint.NormalizedGroup()),
			"groupLabel":          blueprint.GroupLabel(),
			"statusLabel":         blueprint.Status,
			"gosxComponent":       blueprint.GoSXComponent,
			"goSXComponent":       blueprint.GoSXComponent,
			"componentCountLabel": countLabel(blueprint.ComponentCount(), "building block", "building blocks"),
			"components":          authoringSiteMapComponentTemplateViews(blueprint.Components, options),
			"hasComponents":       blueprint.ComponentCount() > 0,
			"inputID":             "studioSiteMapBlueprint-" + workspaceToken(blueprint.Key),
			"inputName":           firstNonEmpty(options.BlueprintInputName, "studioPageBlueprintIntent"),
			"selected":            index == 0,
		})
	}
	return out
}

func authoringSiteMapComponentTemplateViews(templates []ComponentTemplate, options SiteMapViewOptions) []map[string]any {
	out := make([]map[string]any, 0, len(templates))
	for index, template := range templates {
		template = template.Normalize()
		categoryKey := workspaceToken(template.Category)
		if categoryKey == "node" {
			categoryKey = "general"
		}
		out = append(out, map[string]any{
			"key":            template.Key,
			"label":          template.Label,
			"summary":        template.Summary,
			"category":       template.Category,
			"categoryKey":    categoryKey,
			"gosxComponent":  template.GoSXComponent,
			"goSXComponent":  template.GoSXComponent,
			"source":         string(template.NormalizedSource()),
			"sourceLabel":    template.SourceLabel(),
			"defaultBinding": template.DefaultBinding,
			"statusLabel":    template.Status,
			"addLabel":       firstNonEmpty(template.AddLabel, "Add"),
			"controls":       authoringSiteMapStaticControlViews(template.Controls),
			"hasControls":    template.ControlCount() > 0,
			"controlLabel":   countLabel(template.ControlCount(), "field", "fields"),
			"inputID":        "studioSiteMapTemplate-" + workspaceToken(template.Key),
			"inputName":      firstNonEmpty(options.TemplateInputName, "studioComponentTemplateIntent"),
			"selected":       index == 0,
		})
	}
	return out
}

func authoringSiteMapIntentViews(intents []CompositionIntent, options SiteMapViewOptions) []map[string]any {
	out := make([]map[string]any, 0, len(intents))
	for _, intent := range normalizeCompositionIntents(intents) {
		if options.CompositionIntentView != nil {
			out = append(out, options.CompositionIntentView(intent))
			continue
		}
		out = append(out, authoringSiteMapIntentView(intent))
	}
	return out
}

func authoringSiteMapIntentView(intent CompositionIntent) map[string]any {
	intent = intent.Normalize()
	kind := intent.NormalizedKind()
	mutation := AuthoringMutationFromIntent(intent)
	return map[string]any{
		"key":                  intent.Key,
		"label":                intent.Label,
		"summary":              intent.Summary,
		"kind":                 string(kind),
		"kindLabel":            authoringCompositionIntentKindLabel(kind),
		"actionLabel":          authoringCompositionIntentActionLabel(kind),
		"formID":               "studioSiteMapIntentForm-" + workspaceToken(intent.Key),
		"class":                "studio-site-map-intent studio-site-map-intent--" + string(kind),
		"targetPageKey":        intent.TargetPageKey,
		"targetLabel":          firstNonEmpty(intent.TargetPageLabel, intent.PageBlueprintLabel, intent.TargetRoute, "New page"),
		"targetRoute":          intent.TargetRoute,
		"targetRegion":         intent.TargetRegion,
		"pageBlueprintKey":     intent.PageBlueprintKey,
		"pageBlueprintLabel":   intent.PageBlueprintLabel,
		"componentTemplateKey": intent.ComponentTemplateKey,
		"componentLabel":       intent.ComponentLabel,
		"gosxComponent":        intent.GoSXComponent,
		"goSXComponent":        intent.GoSXComponent,
		"binding":              intent.Binding,
		"statusLabel":          firstNonEmpty(intent.Status, "Draft"),
		"stepCountLabel":       countLabel(intent.StepCount(), "step", "steps"),
		"steps":                authoringSiteMapStepViews(intent.Steps),
		"hasSteps":             intent.StepCount() > 0,
		"authoringOperation":   string(mutation.Kind),
		"authoringIntentKey":   mutation.IntentKey,
		"authoringIntentKind":  string(mutation.IntentKind),
		"authoringPageKey":     mutation.PageKey,
		"authoringPageLabel":   mutation.PageLabel,
		"authoringPageRoute":   mutation.PageRoute,
		"authoringBlueprint":   mutation.PageBlueprintKey,
		"authoringTemplate":    mutation.ComponentTemplateKey,
		"authoringComponent":   mutation.ComponentLabel,
		"authoringBinding":     mutation.Binding,
		"authoringRegion":      mutation.TargetRegion,
		"mutation":             AuthoringMutationView(mutation),
		"formValues":           mutation.FormValues(),
		"formInputs":           AuthoringMutationFormInputViews(mutation),
	}
}

func authoringSiteMapStepViews(steps []CompositionStep) []map[string]any {
	out := make([]map[string]any, 0, len(steps))
	for _, step := range normalizeCompositionSteps(steps) {
		out = append(out, map[string]any{
			"key":           step.Key,
			"label":         step.Label,
			"summary":       step.Summary,
			"gosxComponent": step.GoSXComponent,
			"goSXComponent": step.GoSXComponent,
			"binding":       step.Binding,
			"hasBinding":    step.Binding != "",
		})
	}
	return out
}

func authoringSiteMapWorkspaceLayerViews(siteMap SiteMap, workspace CompositionWorkspace, positions map[string]SiteMapNodePosition) []map[string]any {
	siteMap = siteMap.Normalize()
	workspace = workspace.Normalize()
	pagesByKey := map[string]Page{}
	for _, page := range siteMap.Pages {
		pagesByKey[page.Key] = page
	}
	nodesByKey := map[string]WorkspaceNode{}
	for _, node := range workspace.Nodes {
		nodesByKey[node.Key] = node
	}
	out := make([]map[string]any, 0, len(workspace.Layers))
	for _, layer := range workspace.Layers {
		nodes := make([]map[string]any, 0, len(layer.NodeKeys))
		for _, nodeKey := range layer.NodeKeys {
			node, ok := nodesByKey[nodeKey]
			if !ok {
				continue
			}
			nodeView := authoringSiteMapWorkspaceNodeView(node)
			// Stamp a saved free placement (pan-surface-LOCAL px) when the host
			// supplied one for this node key. renderSiteMapBoardWorkspaceNode reads
			// numeric x/y to position the node absolutely; nodes without an entry
			// stay in lane flow, keeping the default view unchanged.
			if position, ok := positions[node.Key]; ok {
				nodeView["x"] = position.X
				nodeView["y"] = position.Y
			}
			nodes = append(nodes, nodeView)
		}
		if len(nodes) == 0 {
			continue
		}
		layerView := map[string]any{
			"key":            layer.Key,
			"label":          layer.Label,
			"summary":        layer.Summary,
			"nodeCountLabel": countLabel(len(nodes), "node", "nodes"),
			"nodes":          nodes,
			"hasNodes":       true,
			"class":          "studio-site-map-workspace-layer studio-site-map-workspace-layer--" + layer.Key,
		}
		if page, ok := pagesByKey[layer.Key]; ok {
			layerView["hasPageComposition"] = true
			layerView["pageRoute"] = page.Route
			layerView["pageComponentLabel"] = componentDisplayLabel(page.GoSXComponent)
			layerView["pageGosxComponent"] = page.GoSXComponent
			layerView["pageGoSXComponent"] = page.GoSXComponent
			layerView["sectionCountLabel"] = countLabel(page.ComponentCount(), "section", "sections")
			layerView["controlCountLabel"] = countLabel(page.ControlCount(), "field", "fields")
		}
		out = append(out, layerView)
	}
	return out
}

func authoringSiteMapWorkspaceNodeView(node WorkspaceNode) map[string]any {
	node = node.Normalize()
	kind := node.NormalizedKind()
	label := authoringWorkspaceNodeLabel(node)
	summary := node.Summary
	if kind == WorkspaceNodeResource {
		summary = ComponentSourceLabel(node.Source) + " resource"
	}
	bindingLabel := authoringBindingLabel(node.Binding)
	className := "studio-site-map-workspace-node studio-site-map-workspace-node--" + string(kind)
	if node.Selected {
		className += " is-selected"
	}
	return map[string]any{
		"key":               node.Key,
		"label":             label,
		"summary":           summary,
		"kind":              string(kind),
		"kindLabel":         authoringWorkspaceNodeKindLabel(kind),
		"layerKey":          node.LayerKey,
		"pageKey":           node.PageKey,
		"route":             node.Route,
		"group":             string(node.Group),
		"groupLabel":        PageGroupLabel(node.Group),
		"gosxComponent":     node.GoSXComponent,
		"goSXComponent":     node.GoSXComponent,
		"componentLabel":    componentDisplayLabel(node.GoSXComponent),
		"source":            string(node.Source),
		"sourceLabel":       ComponentSourceLabel(node.Source),
		"binding":           node.Binding,
		"bindingLabel":      bindingLabel,
		"statusLabel":       firstNonEmpty(node.Status, "Ready"),
		"selected":          node.Selected,
		"inputID":           "studioSiteMapWorkspace-" + workspaceToken(node.Key),
		"inputName":         "studioSiteMapWorkspaceNode",
		"class":             className,
		"hasSummary":        summary != "",
		"hasRoute":          node.Route != "",
		"hasSource":         kind != WorkspaceNodePage,
		"hasBindingLabel":   bindingLabel != "",
		"hasComponentLabel": strings.TrimSpace(node.GoSXComponent) != "",
	}
}

func authoringSiteMapWorkspaceLinkViews(workspace CompositionWorkspace) []map[string]any {
	workspace = workspace.Normalize()
	nodesByKey := authoringWorkspaceNodesByKey(workspace)
	out := make([]map[string]any, 0, len(workspace.Links))
	for _, link := range workspace.Links {
		link = link.Normalize()
		from := nodesByKey[link.FromNodeKey]
		to := nodesByKey[link.ToNodeKey]
		if from.Key == "" || to.Key == "" {
			continue
		}
		kind := link.NormalizedKind()
		out = append(out, map[string]any{
			"key":         link.Key,
			"label":       link.Label,
			"summary":     link.Summary,
			"kind":        string(kind),
			"kindLabel":   WorkspaceLinkKindLabel(kind),
			"fromNodeKey": link.FromNodeKey,
			"toNodeKey":   link.ToNodeKey,
			"fromPageKey": from.PageKey,
			"toPageKey":   to.PageKey,
			"fromGroup":   string(from.Group),
			"toGroup":     string(to.Group),
			"fromLabel":   authoringWorkspaceNodeLabel(from),
			"toLabel":     authoringWorkspaceNodeLabel(to),
			"class":       "studio-site-map-workspace-link studio-site-map-workspace-link--" + string(kind),
		})
	}
	return out
}

func authoringSiteMapWorkspaceCanvasLinkViews(workspace CompositionWorkspace, canvas WorkspaceCanvas) []map[string]any {
	workspace = workspace.Normalize()
	nodesByKey := authoringWorkspaceNodesByKey(workspace)
	linksByKey := map[string]WorkspaceLink{}
	for _, link := range workspace.Links {
		linksByKey[link.Key] = link.Normalize()
	}
	out := make([]map[string]any, 0, len(canvas.Links))
	for _, path := range canvas.Links {
		link := linksByKey[path.Key]
		from := nodesByKey[path.FromNodeKey]
		to := nodesByKey[path.ToNodeKey]
		if link.Key == "" || from.Key == "" || to.Key == "" || strings.TrimSpace(path.Path) == "" {
			continue
		}
		kind := link.NormalizedKind()
		out = append(out, map[string]any{
			"key":         path.Key,
			"path":        path.Path,
			"kind":        string(kind),
			"kindLabel":   WorkspaceLinkKindLabel(kind),
			"fromNodeKey": path.FromNodeKey,
			"toNodeKey":   path.ToNodeKey,
			"fromPageKey": from.PageKey,
			"toPageKey":   to.PageKey,
			"fromGroup":   string(from.Group),
			"toGroup":     string(to.Group),
			"fromLabel":   authoringWorkspaceNodeLabel(from),
			"toLabel":     authoringWorkspaceNodeLabel(to),
			"class":       "studio-site-map-workspace-path studio-site-map-workspace-path--" + string(kind),
		})
	}
	return out
}

func authoringSiteMapSourceViews(siteMap SiteMap) []map[string]any {
	counts := map[ComponentSource]int{}
	for _, page := range siteMap.Normalize().Pages {
		for _, component := range page.Components {
			counts[component.NormalizedSource()]++
		}
	}
	sources := []ComponentSource{ComponentSourceHost, ComponentSourceCMS, ComponentSourcePlugin, ComponentSourceStudio}
	out := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		count := counts[source]
		if count == 0 {
			continue
		}
		out = append(out, map[string]any{
			"key":        string(source),
			"label":      ComponentSourceLabel(source),
			"countLabel": countLabel(count, "component", "components"),
		})
	}
	return out
}

func authoringDefaultWorkspaceNodeView(workspace CompositionWorkspace) (map[string]any, bool) {
	workspace = workspace.Normalize()
	selectedPageKey := ""
	for _, node := range workspace.Nodes {
		if node.Selected && node.NormalizedKind() == WorkspaceNodePage {
			selectedPageKey = strings.TrimSpace(node.PageKey)
			if selectedPageKey == "" {
				selectedPageKey = strings.TrimPrefix(strings.TrimSpace(node.Key), "page:")
			}
			break
		}
	}
	if selectedPageKey != "" {
		for _, node := range workspace.Nodes {
			if node.NormalizedKind() == WorkspaceNodeComponent && strings.TrimSpace(node.PageKey) == selectedPageKey {
				return authoringSiteMapWorkspaceNodeView(node), true
			}
		}
	}
	for _, node := range workspace.Nodes {
		if node.Selected {
			return authoringSiteMapWorkspaceNodeView(node), true
		}
	}
	for _, node := range workspace.Nodes {
		if node.NormalizedKind() == WorkspaceNodeComponent {
			return authoringSiteMapWorkspaceNodeView(node), true
		}
	}
	if len(workspace.Nodes) == 0 {
		return map[string]any{}, false
	}
	return authoringSiteMapWorkspaceNodeView(workspace.Nodes[0]), true
}

func authoringWorkspaceNodesByKey(workspace CompositionWorkspace) map[string]WorkspaceNode {
	out := map[string]WorkspaceNode{}
	for _, node := range workspace.Normalize().Nodes {
		out[node.Key] = node
	}
	return out
}

func authoringFirstItemKey(items []map[string]any) string {
	if len(items) == 0 {
		return ""
	}
	return stringMapValue(items[0], "key")
}

func authoringPreviewHref(route string, options SiteMapViewOptions) string {
	route = strings.TrimSpace(route)
	if route == "" {
		route = "/"
	}
	if options.PreviewHref != nil {
		return options.PreviewHref(route)
	}
	return route
}

func authoringSelectedPageLabel(page Page, ok bool) string {
	if !ok {
		return "No page selected"
	}
	page = page.Normalize()
	return strings.TrimSpace(page.Label + " " + page.Route)
}

func authoringWorkspaceNodeKindLabel(kind WorkspaceNodeKind) string {
	switch kind {
	case WorkspaceNodeComponent:
		return "Section"
	case WorkspaceNodeResource:
		return "Content"
	default:
		return WorkspaceNodeKindLabel(kind)
	}
}

func authoringWorkspaceNodeLabel(node WorkspaceNode) string {
	node = node.Normalize()
	if node.NormalizedKind() == WorkspaceNodeResource {
		return firstNonEmpty(authoringBindingLabel(node.Binding), node.Label)
	}
	return node.Label
}

func authoringBindingLabel(binding string) string {
	binding = strings.TrimSpace(binding)
	if binding == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(binding, "flow."):
		return titleFromToken(strings.TrimPrefix(binding, "flow.")) + " flow"
	case strings.HasPrefix(binding, "flows."):
		return titleFromToken(strings.TrimPrefix(binding, "flows.")) + " flow"
	case strings.Contains(binding, "showcase3d"):
		return "3D model"
	case strings.Contains(binding, "products.collection"):
		return "Product collection"
	case strings.Contains(binding, "products.categories"):
		return "Product categories"
	case strings.Contains(binding, "gallery.collection"):
		return "Gallery works"
	case strings.Contains(binding, "gallery.detail"):
		return "Gallery detail"
	case strings.Contains(binding, "blog.collection"):
		return "Journal posts"
	case strings.Contains(binding, "blog.detail"):
		return "Journal detail"
	case strings.Contains(binding, "contact.links"):
		return "Contact links"
	case strings.Contains(binding, "page.about"):
		return "About page"
	case strings.Contains(binding, "programs.collection"):
		return "Program collection"
	case strings.Contains(binding, "calendar.sessions"):
		return "Calendar sessions"
	}
	parts := strings.FieldsFunc(binding, func(char rune) bool {
		return char == '.' || char == '-' || char == '_'
	})
	if len(parts) == 0 {
		return binding
	}
	return titleFromToken(parts[len(parts)-1]) + " content"
}

func authoringComponentDefaultSummary(component Component) string {
	return strings.TrimSpace(component.Label + " is editable on this site with " + countLabel(component.ControlCount(), "field", "fields") + ".")
}

func authoringComponentSourceSummary(component Component) string {
	switch component.NormalizedSource() {
	case ComponentSourceCMS:
		return "Connected content source"
	case ComponentSourcePlugin:
		return "Plugin-provided block"
	case ComponentSourceStudio:
		return "Studio building block"
	default:
		return "Site-owned editable block"
	}
}

func authoringCompositionIntentKindLabel(kind CompositionIntentKind) string {
	switch kind {
	case CompositionIntentCreatePage:
		return "Page draft"
	default:
		return "Block draft"
	}
}

func authoringCompositionIntentActionLabel(kind CompositionIntentKind) string {
	switch kind {
	case CompositionIntentCreatePage:
		return "Create page"
	case CompositionIntentAddComponent:
		return "Add section"
	default:
		return "Apply draft"
	}
}

func componentDisplayLabel(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	runes := []rune(name)
	var builder strings.Builder
	for index, current := range runes {
		if index > 0 && shouldSplitComponentDisplayLabel(runes[index-1], current, nextComponentRune(runes, index)) {
			builder.WriteByte(' ')
		}
		builder.WriteRune(current)
	}
	return strings.TrimSpace(builder.String())
}

func nextComponentRune(values []rune, index int) rune {
	if index+1 >= len(values) {
		return 0
	}
	return values[index+1]
}

func shouldSplitComponentDisplayLabel(previous rune, current rune, next rune) bool {
	if current >= '0' && current <= '9' {
		return (previous >= 'A' && previous <= 'Z') || (previous >= 'a' && previous <= 'z')
	}
	if current < 'A' || current > 'Z' {
		return false
	}
	return (previous >= 'a' && previous <= 'z') || (previous >= '0' && previous <= '9' && next >= 'a' && next <= 'z') || (previous >= 'A' && previous <= 'Z' && next >= 'a' && next <= 'z')
}

func titleFromToken(token string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(token), func(char rune) bool {
		return char == '.' || char == '-' || char == '_'
	})
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func stringMapValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}
