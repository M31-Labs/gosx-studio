package sitemap

import (
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
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
	CompositionIntentView func(core.CompositionIntent) map[string]any
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

func SiteMapAuthoringView(siteMap core.SiteMap, options SiteMapViewOptions) map[string]any {
	return AuthoringSiteMapView(authoring.NoCodeAuthoringSurface(siteMap), options)
}

func AuthoringSiteMapView(surface authoring.AuthoringSurface, options SiteMapViewOptions) map[string]any {
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
	editableControl := AuthoringSiteMapEditableControlView(siteMap.Pages)
	sources := authoringSiteMapSourceViews(siteMap)
	blueprints := authoringSiteMapPageBlueprintViews(siteMap.Library.PageBlueprints, options)
	// The composition workspace palette surfaces every fresh ComponentTemplate
	// AND every reusable ComponentDefinition available for another instance
	// (core/instances.go's PaletteEntries() concept) — previously this only
	// ever read ComponentTemplates, so shared components had no placement
	// affordance inside SiteMapAuthoringView at all (they lived in a
	// standalone host companion panel instead). Templates are built first so
	// existing palette[0]/defaultTemplateKey consumers see no behavior change
	// when a library declares no shared definitions.
	palette := AuthoringSiteMapComponentTemplateViews(siteMap.Library.ComponentTemplates, options)
	palette = append(palette, AuthoringSiteMapComponentDefinitionPaletteViews(siteMap.Library.ComponentDefinitions, siteMap.Pages, selectedPage, hasSelectedPage, options)...)
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
		"authoring":                          authoring.AuthoringSurfaceView(surface),
	}
}

func authoringSiteMapPageViews(pages []core.Page, options SiteMapViewOptions) []map[string]any {
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

func authoringSiteMapMetadataPageView(pages []core.Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		if !page.Editable {
			continue
		}
		mutation := authoring.AuthoringMutationForPage(page)
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
			"mutation":           authoring.AuthoringMutationView(mutation),
			"formValues":         mutation.FormValues(),
			"formInputs":         authoring.AuthoringMutationFormInputViews(mutation),
		}
	}
	return nil
}

func authoringSiteMapComponentViews(page core.Page, components []core.Component) []map[string]any {
	out := make([]map[string]any, 0, len(components))
	page = page.Normalize()
	for index, component := range components {
		component = component.Normalize()
		component.Position = index
		moveUpMutation := authoring.AuthoringMutation{}
		moveDownMutation := authoring.AuthoringMutation{}
		moveUpFormValues := map[string]string(nil)
		moveDownFormValues := map[string]string(nil)
		moveUpFormInputs := []map[string]string(nil)
		moveDownFormInputs := []map[string]string(nil)
		canMoveUp := component.CanReorder && index > 0
		canMoveDown := component.CanReorder && index < len(components)-1
		if canMoveUp {
			moveUpMutation = authoring.AuthoringMutationForComponentReorder(page, component, index-1)
			moveUpFormValues = moveUpMutation.FormValues()
			moveUpFormInputs = authoring.AuthoringMutationFormInputViews(moveUpMutation)
		}
		if canMoveDown {
			moveDownMutation = authoring.AuthoringMutationForComponentReorder(page, component, index+1)
			moveDownFormValues = moveDownMutation.FormValues()
			moveDownFormInputs = authoring.AuthoringMutationFormInputViews(moveDownMutation)
		}
		visibilityMutation := authoring.AuthoringMutation{}
		visibilityFormValues := map[string]string(nil)
		visibilityFormInputs := []map[string]string(nil)
		visibilityActionLabel := ""
		if component.CanToggleVisibility {
			visibilityMutation = authoring.AuthoringMutationForComponentVisibility(page, component, !component.Visible)
			visibilityFormValues = visibilityMutation.FormValues()
			visibilityFormInputs = authoring.AuthoringMutationFormInputViews(visibilityMutation)
			visibilityActionLabel = "Show"
			if component.Visible {
				visibilityActionLabel = "Hide"
			}
		}
		deleteMutation := authoring.AuthoringMutation{}
		deleteFormValues := map[string]string(nil)
		deleteFormInputs := []map[string]string(nil)
		if component.CanDelete {
			deleteMutation = authoring.AuthoringMutationForComponentDelete(page, component)
			deleteFormValues = deleteMutation.FormValues()
			deleteFormInputs = authoring.AuthoringMutationFormInputViews(deleteMutation)
		}
		duplicateMutation := authoring.AuthoringMutation{}
		duplicateFormValues := map[string]string(nil)
		duplicateFormInputs := []map[string]string(nil)
		if component.CanDuplicate {
			duplicateMutation = authoring.AuthoringMutationForComponentDuplicate(page, component, index+1)
			duplicateFormValues = duplicateMutation.FormValues()
			duplicateFormInputs = authoring.AuthoringMutationFormInputViews(duplicateMutation)
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
			"sourceLabel":                  core.ComponentSourceLabel(component.Source),
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
			"moveUpMutation":               authoring.AuthoringMutationView(moveUpMutation),
			"moveUpFormValues":             moveUpFormValues,
			"moveUpFormInputs":             moveUpFormInputs,
			"moveDownAuthoringOperation":   string(moveDownMutation.Kind),
			"moveDownAuthoringPageKey":     moveDownMutation.PageKey,
			"moveDownAuthoringComponent":   moveDownMutation.ComponentKey,
			"moveDownAuthoringPosition":    moveDownMutation.Position,
			"moveDownMutation":             authoring.AuthoringMutationView(moveDownMutation),
			"moveDownFormValues":           moveDownFormValues,
			"moveDownFormInputs":           moveDownFormInputs,
			"visible":                      component.Visible,
			"canToggleVisibility":          component.CanToggleVisibility,
			"visibilityActionLabel":        visibilityActionLabel,
			"visibilityAuthoringOperation": string(visibilityMutation.Kind),
			"visibilityAuthoringPageKey":   visibilityMutation.PageKey,
			"visibilityAuthoringComponent": visibilityMutation.ComponentKey,
			"visibilityAuthoringVisible":   visibilityMutation.Visible,
			"visibilityMutation":           authoring.AuthoringMutationView(visibilityMutation),
			"visibilityFormValues":         visibilityFormValues,
			"visibilityFormInputs":         visibilityFormInputs,
			"canDelete":                    component.CanDelete,
			"deleteActionLabel":            "Delete",
			"deleteAuthoringOperation":     string(deleteMutation.Kind),
			"deleteAuthoringPageKey":       deleteMutation.PageKey,
			"deleteAuthoringComponent":     deleteMutation.ComponentKey,
			"deleteMutation":               authoring.AuthoringMutationView(deleteMutation),
			"deleteFormValues":             deleteFormValues,
			"deleteFormInputs":             deleteFormInputs,
			"duplicateActionLabel":         "Duplicate",
			"duplicateAuthoringOperation":  string(duplicateMutation.Kind),
			"duplicateAuthoringPageKey":    duplicateMutation.PageKey,
			"duplicateAuthoringComponent":  duplicateMutation.ComponentKey,
			"duplicateAuthoringTemplate":   duplicateMutation.ComponentTemplateKey,
			"duplicateAuthoringPosition":   duplicateMutation.Position,
			"duplicateMutation":            authoring.AuthoringMutationView(duplicateMutation),
			"duplicateFormValues":          duplicateFormValues,
			"duplicateFormInputs":          duplicateFormInputs,
			"controls":                     AuthoringSiteMapControlViews(page, component, component.Controls),
			"hasControls":                  component.ControlCount() > 0,
			"controlLabel":                 countLabel(component.ControlCount(), "field", "fields"),
		})
	}
	return out
}

func authoringSiteMapReorderComponentView(pages []core.Page) map[string]any {
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
			mutation := authoring.AuthoringMutationForComponentReorder(page, component, targetPosition)
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
				"mutation":            authoring.AuthoringMutationView(mutation),
				"formValues":          mutation.FormValues(),
				"formInputs":          authoring.AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func authoringSiteMapDuplicateComponentView(pages []core.Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for index, component := range page.Components {
			component = component.Normalize()
			component.Position = index
			if !component.CanDuplicate {
				continue
			}
			mutation := authoring.AuthoringMutationForComponentDuplicate(page, component, index+1)
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
				"mutation":           authoring.AuthoringMutationView(mutation),
				"formValues":         mutation.FormValues(),
				"formInputs":         authoring.AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func authoringSiteMapVisibilityComponentView(pages []core.Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for _, component := range page.Components {
			component = component.Normalize()
			if !component.CanToggleVisibility {
				continue
			}
			mutation := authoring.AuthoringMutationForComponentVisibility(page, component, !component.Visible)
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
				"mutation":           authoring.AuthoringMutationView(mutation),
				"formValues":         mutation.FormValues(),
				"formInputs":         authoring.AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func authoringSiteMapDeleteComponentView(pages []core.Page) map[string]any {
	for _, page := range pages {
		page = page.Normalize()
		for index, component := range page.Components {
			component = component.Normalize()
			component.Position = index
			if !component.CanDelete {
				continue
			}
			mutation := authoring.AuthoringMutationForComponentDelete(page, component)
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
				"mutation":           authoring.AuthoringMutationView(mutation),
				"formValues":         mutation.FormValues(),
				"formInputs":         authoring.AuthoringMutationFormInputViews(mutation),
			}
		}
	}
	return nil
}

func AuthoringSiteMapEditableControlView(pages []core.Page) map[string]any {
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
				// Dev-locked controls are never offered as editable fields.
				if control.Locked {
					continue
				}
				mutation := authoring.AuthoringMutationForControl(page, component, control)
				view := authoringSiteMapControlView(control)
				view["pageKey"] = page.Key
				view["pageLabel"] = page.Label
				view["pageRoute"] = page.Route
				view["componentKey"] = component.Key
				view["componentLabel"] = component.Label
				view["componentBinding"] = component.Binding
				view["selectionKey"] = component.SelectionKey(page.Key)
				view["inputName"] = authoring.AuthoringFieldValue
				view["actionLabel"] = "Save field"
				view["authoringOperation"] = string(mutation.Kind)
				view["authoringPageKey"] = mutation.PageKey
				view["authoringComponent"] = mutation.ComponentKey
				view["authoringControl"] = mutation.ControlKey
				view["authoringBinding"] = mutation.Binding
				view["authoringControlKind"] = string(mutation.ControlKind)
				view["mutation"] = authoring.AuthoringMutationView(mutation)
				view["formValues"] = mutation.FormValues()
				view["formInputs"] = authoring.AuthoringMutationFormInputViews(mutation)
				return view
			}
		}
	}
	return nil
}

func AuthoringSiteMapControlViews(page core.Page, component core.Component, controls []core.Control) []map[string]any {
	out := make([]map[string]any, 0, len(controls))
	for _, control := range controls {
		control = control.Normalize()
		if control.Key == "" || control.Label == "" {
			continue
		}
		// Dev-locked controls are never offered as editable fields.
		if control.Locked {
			continue
		}
		mutation := authoring.AuthoringMutationForControl(page, component, control)
		view := authoringSiteMapControlView(control)
		view["inputName"] = authoring.AuthoringFieldValue
		view["authoringOperation"] = string(mutation.Kind)
		view["authoringPageKey"] = mutation.PageKey
		view["authoringComponent"] = mutation.ComponentKey
		view["authoringControl"] = mutation.ControlKey
		view["authoringBinding"] = mutation.Binding
		view["authoringControlKind"] = string(mutation.ControlKind)
		view["mutation"] = authoring.AuthoringMutationView(mutation)
		view["formValues"] = mutation.FormValues()
		view["formInputs"] = authoring.AuthoringMutationFormInputViews(mutation)
		out = append(out, view)
	}
	return out
}

func AuthoringSiteMapStaticControlViews(controls []core.Control) []map[string]any {
	out := make([]map[string]any, 0, len(controls))
	for _, control := range controls {
		control = control.Normalize()
		if control.Key == "" || control.Label == "" {
			continue
		}
		// Dev-locked controls are never offered as editable fields.
		if control.Locked {
			continue
		}
		out = append(out, authoringSiteMapControlView(control))
	}
	return out
}

func authoringSiteMapControlView(control core.Control) map[string]any {
	return map[string]any{
		"key":       control.Key,
		"label":     control.Label,
		"kind":      string(control.NormalizedKind()),
		"kindLabel": control.KindLabel(),
		"binding":   control.Binding,
		"value":     control.Value,
		"advanced":  control.Advanced,
		"required":  control.Required,
		"locked":    control.Locked,
	}
}

func authoringSiteMapPageBlueprintViews(blueprints []core.PageBlueprint, options SiteMapViewOptions) []map[string]any {
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
			"components":          AuthoringSiteMapComponentTemplateViews(blueprint.Components, options),
			"hasComponents":       blueprint.ComponentCount() > 0,
			"inputID":             "studioSiteMapBlueprint-" + workspaceToken(blueprint.Key),
			"inputName":           firstNonEmpty(options.BlueprintInputName, "studioPageBlueprintIntent"),
			"selected":            index == 0,
		})
	}
	return out
}

func AuthoringSiteMapComponentTemplateViews(templates []core.ComponentTemplate, options SiteMapViewOptions) []map[string]any {
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
			"controls":       AuthoringSiteMapStaticControlViews(template.Controls),
			"hasControls":    template.ControlCount() > 0,
			"controlLabel":   countLabel(template.ControlCount(), "field", "fields"),
			"inputID":        "studioSiteMapTemplate-" + workspaceToken(template.Key),
			"inputName":      firstNonEmpty(options.TemplateInputName, "studioComponentTemplateIntent"),
			"selected":       index == 0,
		})
	}
	return out
}

// AuthoringSiteMapComponentDefinitionPaletteViews projects every reusable
// ComponentDefinition (core/instances.go) as a placeable composition
// workspace palette entry, alongside AuthoringSiteMapComponentTemplateViews's
// fresh-component templates — the piece SiteMapAuthoringView's palette was
// missing (it only ever read Library.ComponentTemplates). Each entry carries
// its current attached-instance count (core.Component.IsSharedInstance
// across every page, matching core.EffectiveComponentControls's own
// attached/detached contract) and, when a page is selected, a ready-to-submit
// authoring.AuthoringOperationPlaceInstance mutation (formID/formValues/
// formInputs) a host can render as a real "place another instance" form —
// see sitemap's RenderSiteMapAuthoringForms/RenderSiteMapAuthoringPanels,
// which do exactly that.
func AuthoringSiteMapComponentDefinitionPaletteViews(definitions []core.ComponentDefinition, pages []core.Page, selectedPage core.Page, hasSelectedPage bool, options SiteMapViewOptions) []map[string]any {
	counts := authoringSharedInstanceCounts(pages)
	out := make([]map[string]any, 0, len(definitions))
	for index, definition := range definitions {
		definition = definition.Normalize()
		if definition.Key == "" {
			continue
		}
		categoryKey := workspaceToken(definition.Category)
		if categoryKey == "" || categoryKey == "node" {
			categoryKey = "shared"
		}
		count := counts[definition.Key]
		view := map[string]any{
			"key":                "shared:" + definition.Key,
			"label":              definition.Label,
			"summary":            definition.Summary,
			"category":           definition.Category,
			"categoryKey":        categoryKey,
			"gosxComponent":      definition.GoSXComponent,
			"goSXComponent":      definition.GoSXComponent,
			"source":             string(definition.Source),
			"sourceLabel":        core.ComponentSourceLabel(definition.Source),
			"defaultBinding":     "",
			"statusLabel":        "Shared",
			"actionLabel":        "Add another " + definition.Label,
			"addLabel":           "Add another " + definition.Label,
			"controls":           AuthoringSiteMapStaticControlViews(definition.Controls),
			"hasControls":        definition.ControlCount() > 0,
			"controlLabel":       countLabel(definition.ControlCount(), "field", "fields"),
			"inputID":            "studioSiteMapSharedDefinition-" + workspaceToken(definition.Key),
			"inputName":          firstNonEmpty(options.TemplateInputName, "studioComponentTemplateIntent"),
			"selected":           index == 0,
			"isShared":           true,
			"definitionKey":      definition.Key,
			"instanceCount":      count,
			"instanceCountValue": strconv.Itoa(count),
			"instanceCountLabel": countLabel(count, "instance", "instances"),
			"hasInstances":       count > 0,
			"formID":             "studioSiteMapPlaceInstanceForm-" + workspaceToken(definition.Key),
		}
		if hasSelectedPage {
			mutation := authoring.AuthoringMutation{
				Kind:                 authoring.AuthoringOperationPlaceInstance,
				PageKey:              selectedPage.Key,
				ComponentTemplateKey: definition.Key,
			}
			view["authoringOperation"] = string(mutation.Kind)
			view["authoringPageKey"] = mutation.PageKey
			view["authoringDefinitionKey"] = mutation.ComponentTemplateKey
			view["mutation"] = authoring.AuthoringMutationView(mutation)
			view["formValues"] = mutation.FormValues()
			view["formInputs"] = authoring.AuthoringMutationFormInputViews(mutation)
		}
		out = append(out, view)
	}
	return out
}

// authoringSharedInstanceCounts counts, for each shared ComponentDefinition
// key, how many currently-ATTACHED placed instances exist across pages —
// core.Component.IsSharedInstance()'s own contract (DefinitionKey set, not
// Detached), the same set core.EffectiveComponentControls fans shared edits
// out to.
func authoringSharedInstanceCounts(pages []core.Page) map[string]int {
	counts := map[string]int{}
	for _, page := range pages {
		page = page.Normalize()
		for _, component := range page.Components {
			component = component.Normalize()
			if !component.IsSharedInstance() {
				continue
			}
			counts[component.NormalizedDefinitionKey()]++
		}
	}
	return counts
}

func authoringSiteMapIntentViews(intents []core.CompositionIntent, options SiteMapViewOptions) []map[string]any {
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

func authoringSiteMapIntentView(intent core.CompositionIntent) map[string]any {
	intent = intent.Normalize()
	kind := intent.NormalizedKind()
	mutation := authoring.AuthoringMutationFromIntent(intent)
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
		"mutation":             authoring.AuthoringMutationView(mutation),
		"formValues":           mutation.FormValues(),
		"formInputs":           authoring.AuthoringMutationFormInputViews(mutation),
	}
}

func authoringSiteMapStepViews(steps []core.CompositionStep) []map[string]any {
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

func authoringSiteMapWorkspaceLayerViews(siteMap core.SiteMap, workspace core.CompositionWorkspace, positions map[string]SiteMapNodePosition) []map[string]any {
	siteMap = siteMap.Normalize()
	workspace = workspace.Normalize()
	pagesByKey := map[string]core.Page{}
	for _, page := range siteMap.Pages {
		pagesByKey[page.Key] = page
	}
	nodesByKey := map[string]core.WorkspaceNode{}
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

func authoringSiteMapWorkspaceNodeView(node core.WorkspaceNode) map[string]any {
	node = node.Normalize()
	kind := node.NormalizedKind()
	label := authoringWorkspaceNodeLabel(node)
	summary := node.Summary
	if kind == core.WorkspaceNodeResource {
		summary = core.ComponentSourceLabel(node.Source) + " resource"
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
		"groupLabel":        core.PageGroupLabel(node.Group),
		"gosxComponent":     node.GoSXComponent,
		"goSXComponent":     node.GoSXComponent,
		"componentLabel":    componentDisplayLabel(node.GoSXComponent),
		"source":            string(node.Source),
		"sourceLabel":       core.ComponentSourceLabel(node.Source),
		"binding":           node.Binding,
		"bindingLabel":      bindingLabel,
		"statusLabel":       firstNonEmpty(node.Status, "Ready"),
		"selected":          node.Selected,
		"inputID":           "studioSiteMapWorkspace-" + workspaceToken(node.Key),
		"inputName":         "studioSiteMapWorkspaceNode",
		"class":             className,
		"hasSummary":        summary != "",
		"hasRoute":          node.Route != "",
		"hasSource":         kind != core.WorkspaceNodePage,
		"hasBindingLabel":   bindingLabel != "",
		"hasComponentLabel": strings.TrimSpace(node.GoSXComponent) != "",
	}
}

func authoringSiteMapWorkspaceLinkViews(workspace core.CompositionWorkspace) []map[string]any {
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
			"kindLabel":   core.WorkspaceLinkKindLabel(kind),
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

func authoringSiteMapWorkspaceCanvasLinkViews(workspace core.CompositionWorkspace, canvas core.WorkspaceCanvas) []map[string]any {
	workspace = workspace.Normalize()
	nodesByKey := authoringWorkspaceNodesByKey(workspace)
	linksByKey := map[string]core.WorkspaceLink{}
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
			"kindLabel":   core.WorkspaceLinkKindLabel(kind),
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

func authoringSiteMapSourceViews(siteMap core.SiteMap) []map[string]any {
	counts := map[core.ComponentSource]int{}
	for _, page := range siteMap.Normalize().Pages {
		for _, component := range page.Components {
			counts[component.NormalizedSource()]++
		}
	}
	sources := []core.ComponentSource{core.ComponentSourceHost, core.ComponentSourceCMS, core.ComponentSourcePlugin, core.ComponentSourceStudio}
	out := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		count := counts[source]
		if count == 0 {
			continue
		}
		out = append(out, map[string]any{
			"key":        string(source),
			"label":      core.ComponentSourceLabel(source),
			"countLabel": countLabel(count, "component", "components"),
		})
	}
	return out
}

func authoringDefaultWorkspaceNodeView(workspace core.CompositionWorkspace) (map[string]any, bool) {
	workspace = workspace.Normalize()
	selectedPageKey := ""
	for _, node := range workspace.Nodes {
		if node.Selected && node.NormalizedKind() == core.WorkspaceNodePage {
			selectedPageKey = strings.TrimSpace(node.PageKey)
			if selectedPageKey == "" {
				selectedPageKey = strings.TrimPrefix(strings.TrimSpace(node.Key), "page:")
			}
			break
		}
	}
	if selectedPageKey != "" {
		for _, node := range workspace.Nodes {
			if node.NormalizedKind() == core.WorkspaceNodeComponent && strings.TrimSpace(node.PageKey) == selectedPageKey {
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
		if node.NormalizedKind() == core.WorkspaceNodeComponent {
			return authoringSiteMapWorkspaceNodeView(node), true
		}
	}
	if len(workspace.Nodes) == 0 {
		return map[string]any{}, false
	}
	return authoringSiteMapWorkspaceNodeView(workspace.Nodes[0]), true
}

func authoringWorkspaceNodesByKey(workspace core.CompositionWorkspace) map[string]core.WorkspaceNode {
	out := map[string]core.WorkspaceNode{}
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

func authoringSelectedPageLabel(page core.Page, ok bool) string {
	if !ok {
		return "No page selected"
	}
	page = page.Normalize()
	return strings.TrimSpace(page.Label + " " + page.Route)
}

func authoringWorkspaceNodeKindLabel(kind core.WorkspaceNodeKind) string {
	switch kind {
	case core.WorkspaceNodeComponent:
		return "Section"
	case core.WorkspaceNodeResource:
		return "Content"
	default:
		return core.WorkspaceNodeKindLabel(kind)
	}
}

func authoringWorkspaceNodeLabel(node core.WorkspaceNode) string {
	node = node.Normalize()
	if node.NormalizedKind() == core.WorkspaceNodeResource {
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

func authoringComponentDefaultSummary(component core.Component) string {
	return strings.TrimSpace(component.Label + " is editable on this site with " + countLabel(component.ControlCount(), "field", "fields") + ".")
}

func authoringComponentSourceSummary(component core.Component) string {
	switch component.NormalizedSource() {
	case core.ComponentSourceCMS:
		return "Connected content source"
	case core.ComponentSourcePlugin:
		return "Plugin-provided block"
	case core.ComponentSourceStudio:
		return "Studio building block"
	default:
		return "Site-owned editable block"
	}
}

func authoringCompositionIntentKindLabel(kind core.CompositionIntentKind) string {
	switch kind {
	case core.CompositionIntentCreatePage:
		return "Page draft"
	default:
		return "Block draft"
	}
}

func authoringCompositionIntentActionLabel(kind core.CompositionIntentKind) string {
	switch kind {
	case core.CompositionIntentCreatePage:
		return "Create page"
	case core.CompositionIntentAddComponent:
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
