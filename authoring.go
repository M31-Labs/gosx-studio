package studio

import "strings"

// AuthoringSurface is the no-code projection a host can hand to Studio UI.
//
// It keeps the host's GoSX component names and opaque bindings intact while
// presenting page, palette, intent, and graph data in editor-facing terms.
type AuthoringSurface struct {
	SiteMap             SiteMap
	SelectedPage        Page
	HasSelectedPage     bool
	Workspace           CompositionWorkspace
	Canvas              WorkspaceCanvas
	CreatePageIntents   []CompositionIntent
	AddComponentIntents []CompositionIntent
}

// NoCodeAuthoringSurface derives the editor-facing no-code model from a site
// map. Hosts still own persistence; Studio owns the neutral operation language.
func NoCodeAuthoringSurface(siteMap SiteMap) AuthoringSurface {
	siteMap = siteMap.Normalize()
	selectedPage, hasSelectedPage := siteMap.SelectedPage()
	workspace := siteMap.CompositionWorkspace()
	return AuthoringSurface{
		SiteMap:             siteMap,
		SelectedPage:        selectedPage,
		HasSelectedPage:     hasSelectedPage,
		Workspace:           workspace,
		Canvas:              workspace.CanvasLayout(),
		CreatePageIntents:   authoringCreatePageIntents(siteMap.Library),
		AddComponentIntents: authoringAddComponentIntents(selectedPage, hasSelectedPage, siteMap.Library),
	}
}

func (surface AuthoringSurface) Intents() []CompositionIntent {
	out := make([]CompositionIntent, 0, len(surface.CreatePageIntents)+len(surface.AddComponentIntents))
	out = append(out, normalizeCompositionIntents(surface.CreatePageIntents)...)
	out = append(out, normalizeCompositionIntents(surface.AddComponentIntents)...)
	return out
}

func (surface AuthoringSurface) IntentCount() int {
	return len(surface.Intents())
}

func AuthoringSurfaceView(surface AuthoringSurface) map[string]any {
	siteMap := surface.SiteMap.Normalize()
	selectedPage := surface.SelectedPage.Normalize()
	if !surface.HasSelectedPage {
		selectedPage = Page{}
	}
	workspace := surface.Workspace.Normalize()
	canvas := surface.Canvas
	if canvas.ViewBox == "" && (len(canvas.Nodes) == 0 || len(canvas.Links) == 0) {
		canvas = workspace.CanvasLayout()
	}
	createPageIntents := normalizeCompositionIntents(surface.CreatePageIntents)
	addComponentIntents := normalizeCompositionIntents(surface.AddComponentIntents)
	return map[string]any{
		"pages":                  authoringPageViews(siteMap.Pages),
		"pageCount":              len(siteMap.Pages),
		"componentCount":         siteMap.ComponentCount(),
		"controlCount":           siteMap.ControlCount(),
		"groupCounts":            authoringPageGroupCountViews(siteMap.PageGroupCounts()),
		"selectedPage":           authoringPageView(selectedPage),
		"hasSelectedPage":        surface.HasSelectedPage,
		"library":                authoringLibraryView(siteMap.Library),
		"pageBlueprintCount":     siteMap.BlueprintCount(),
		"componentTemplateCount": siteMap.TemplateCount(),
		"createPageIntents":      authoringIntentViews(createPageIntents),
		"addComponentIntents":    authoringIntentViews(addComponentIntents),
		"intents":                authoringIntentViews(append(createPageIntents, addComponentIntents...)),
		"intentCount":            len(createPageIntents) + len(addComponentIntents),
		"mutationFields":         AuthoringFieldNamesView(),
		"workspace":              authoringWorkspaceView(workspace, canvas),
	}
}

func authoringCreatePageIntents(library CompositionLibrary) []CompositionIntent {
	library = library.Normalize()
	out := make([]CompositionIntent, 0, len(library.PageBlueprints))
	for _, blueprint := range library.PageBlueprints {
		out = append(out, authoringCreatePageIntent(blueprint))
	}
	return normalizeCompositionIntents(out)
}

func authoringCreatePageIntent(blueprint PageBlueprint) CompositionIntent {
	blueprint = blueprint.Normalize()
	steps := []CompositionStep{{
		Key:           "route",
		Label:         firstNonEmpty(blueprint.RoutePattern, "New route"),
		Summary:       firstNonEmpty(authoringRouteSummary(blueprint.RoutePattern), "Create page route"),
		GoSXComponent: blueprint.GoSXComponent,
	}}
	for _, component := range blueprint.Components {
		steps = append(steps, CompositionStep{
			Key:           "component:" + component.Key,
			Label:         component.Label,
			Summary:       firstNonEmpty(component.Summary, "Add "+component.Label+" section."),
			GoSXComponent: component.GoSXComponent,
			Binding:       component.DefaultBinding,
		})
	}
	return CompositionIntent{
		Key:                "create-page:" + workspaceToken(blueprint.Key),
		Label:              "Create " + blueprint.Label,
		Summary:            "Adds " + firstNonEmpty(blueprint.RoutePattern, "a new route") + " with " + countLabel(blueprint.ComponentCount(), "building block", "building blocks") + ".",
		Kind:               CompositionIntentCreatePage,
		TargetRoute:        blueprint.RoutePattern,
		TargetRegion:       blueprint.GroupLabel(),
		PageBlueprintKey:   blueprint.Key,
		PageBlueprintLabel: blueprint.Label,
		GoSXComponent:      blueprint.GoSXComponent,
		Status:             firstNonEmpty(blueprint.Status, "Draft"),
		Steps:              steps,
	}.Normalize()
}

func authoringAddComponentIntents(page Page, hasPage bool, library CompositionLibrary) []CompositionIntent {
	if !hasPage {
		return nil
	}
	page = page.Normalize()
	library = library.Normalize()
	out := make([]CompositionIntent, 0, len(library.ComponentTemplates))
	for _, template := range library.ComponentTemplates {
		out = append(out, authoringAddComponentIntent(page, template))
	}
	return normalizeCompositionIntents(out)
}

func authoringAddComponentIntent(page Page, template ComponentTemplate) CompositionIntent {
	template = template.Normalize()
	label := firstNonEmpty(template.AddLabel, "Add "+template.Label)
	steps := []CompositionStep{
		{
			Key:           "target",
			Label:         page.Label,
			Summary:       "Selected route " + page.Route,
			GoSXComponent: page.GoSXComponent,
		},
		{
			Key:           "component",
			Label:         template.Label,
			Summary:       firstNonEmpty(template.Summary, "Add "+template.SourceLabel()+" component."),
			GoSXComponent: template.GoSXComponent,
			Binding:       template.DefaultBinding,
		},
	}
	if template.ControlCount() > 0 {
		steps = append(steps, CompositionStep{
			Key:     "controls",
			Label:   countLabel(template.ControlCount(), "field", "fields"),
			Summary: "Expose no-code controls",
			Binding: template.DefaultBinding,
		})
	}
	return CompositionIntent{
		Key:                  "add-component:" + workspaceToken(page.Key) + ":" + workspaceToken(template.Key),
		Label:                label,
		Summary:              "Places " + template.Label + " on " + firstNonEmpty(page.Route, page.Label) + " and exposes " + countLabel(template.ControlCount(), "field", "fields") + ".",
		Kind:                 CompositionIntentAddComponent,
		TargetPageKey:        page.Key,
		TargetPageLabel:      page.Label,
		TargetRoute:          page.Route,
		TargetRegion:         "Main content",
		ComponentTemplateKey: template.Key,
		ComponentLabel:       template.Label,
		GoSXComponent:        template.GoSXComponent,
		Binding:              template.DefaultBinding,
		Status:               firstNonEmpty(template.Status, "Draft"),
		Steps:                steps,
	}.Normalize()
}

func authoringRouteSummary(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return ""
	}
	return "Create route " + route
}

func authoringPageViews(pages []Page) []map[string]any {
	out := make([]map[string]any, 0, len(pages))
	for _, page := range pages {
		out = append(out, authoringPageView(page))
	}
	return out
}

func authoringPageView(page Page) map[string]any {
	page = page.Normalize()
	return map[string]any{
		"key":            page.Key,
		"label":          page.Label,
		"route":          page.Route,
		"group":          string(page.NormalizedGroup()),
		"groupLabel":     page.GroupLabel(),
		"goSXComponent":  page.GoSXComponent,
		"status":         page.Status,
		"editable":       page.Editable,
		"selected":       page.Selected,
		"componentCount": page.ComponentCount(),
		"controlCount":   page.ControlCount(),
		"components":     authoringComponentViews(page, page.Components),
	}
}

func authoringComponentViews(page Page, components []Component) []map[string]any {
	out := make([]map[string]any, 0, len(components))
	for _, component := range components {
		out = append(out, authoringComponentView(page, component))
	}
	return out
}

func authoringComponentView(page Page, component Component) map[string]any {
	component = component.Normalize()
	return map[string]any{
		"key":                 component.Key,
		"label":               component.Label,
		"summary":             component.Summary,
		"hasSummary":          component.Summary != "",
		"goSXComponent":       component.GoSXComponent,
		"source":              string(component.NormalizedSource()),
		"sourceLabel":         ComponentSourceLabel(component.Source),
		"binding":             component.Binding,
		"status":              component.Status,
		"editable":            component.Editable,
		"position":            component.Position,
		"canReorder":          component.CanReorder,
		"visible":             component.Visible,
		"canToggleVisibility": component.CanToggleVisibility,
		"canDelete":           component.CanDelete,
		"controlCount":        component.ControlCount(),
		"controls":            authoringControlViews(page, component, component.Controls),
	}
}

func authoringControlViews(page Page, component Component, controls []Control) []map[string]any {
	page = page.Normalize()
	component = component.Normalize()
	out := make([]map[string]any, 0, len(controls))
	for _, control := range controls {
		control = control.Normalize()
		if control.Key == "" || control.Label == "" {
			continue
		}
		view := map[string]any{
			"key":       control.Key,
			"label":     control.Label,
			"kind":      string(control.NormalizedKind()),
			"kindLabel": control.KindLabel(),
			"binding":   control.Binding,
			"value":     control.Value,
			"help":      control.Help,
			"required":  control.Required,
			"advanced":  control.Advanced,
			"options":   authoringControlOptionViews(control.Options),
			"inputName": AuthoringFieldValue,
		}
		if page.Key != "" && component.Key != "" {
			mutation := AuthoringMutationForControl(page, component, control)
			view["mutation"] = AuthoringMutationView(mutation)
			view["formValues"] = mutation.FormValues()
		}
		out = append(out, view)
	}
	return out
}

func authoringControlOptionViews(options []ControlOption) []map[string]string {
	out := make([]map[string]string, 0, len(options))
	for _, option := range options {
		option.Value = strings.TrimSpace(option.Value)
		option.Label = strings.TrimSpace(option.Label)
		if option.Value == "" || option.Label == "" {
			continue
		}
		out = append(out, map[string]string{"value": option.Value, "label": option.Label})
	}
	return out
}

func authoringLibraryView(library CompositionLibrary) map[string]any {
	library = library.Normalize()
	return map[string]any{
		"pageBlueprints":     authoringPageBlueprintViews(library.PageBlueprints),
		"componentTemplates": authoringComponentTemplateViews(library.ComponentTemplates),
		"pageBlueprintCount": len(library.PageBlueprints),
		"templateCount":      len(library.ComponentTemplates),
	}
}

func authoringPageBlueprintViews(blueprints []PageBlueprint) []map[string]any {
	out := make([]map[string]any, 0, len(blueprints))
	for _, blueprint := range blueprints {
		blueprint = blueprint.Normalize()
		out = append(out, map[string]any{
			"key":            blueprint.Key,
			"label":          blueprint.Label,
			"summary":        blueprint.Summary,
			"routePattern":   blueprint.RoutePattern,
			"group":          string(blueprint.NormalizedGroup()),
			"groupLabel":     blueprint.GroupLabel(),
			"goSXComponent":  blueprint.GoSXComponent,
			"status":         blueprint.Status,
			"componentCount": blueprint.ComponentCount(),
			"components":     authoringComponentTemplateViews(blueprint.Components),
		})
	}
	return out
}

func authoringComponentTemplateViews(templates []ComponentTemplate) []map[string]any {
	out := make([]map[string]any, 0, len(templates))
	for _, template := range templates {
		template = template.Normalize()
		out = append(out, map[string]any{
			"key":            template.Key,
			"label":          template.Label,
			"summary":        template.Summary,
			"category":       template.Category,
			"goSXComponent":  template.GoSXComponent,
			"source":         string(template.NormalizedSource()),
			"sourceLabel":    template.SourceLabel(),
			"defaultBinding": template.DefaultBinding,
			"status":         template.Status,
			"addLabel":       firstNonEmpty(template.AddLabel, "Add "+template.Label),
			"controlCount":   template.ControlCount(),
			"controls":       authoringControlViews(Page{}, Component{}, template.Controls),
		})
	}
	return out
}

func authoringIntentViews(intents []CompositionIntent) []map[string]any {
	out := make([]map[string]any, 0, len(intents))
	for _, intent := range normalizeCompositionIntents(intents) {
		mutation := AuthoringMutationFromIntent(intent)
		out = append(out, map[string]any{
			"key":                  intent.Key,
			"label":                intent.Label,
			"summary":              intent.Summary,
			"kind":                 string(intent.NormalizedKind()),
			"targetPageKey":        intent.TargetPageKey,
			"targetPageLabel":      intent.TargetPageLabel,
			"targetRoute":          intent.TargetRoute,
			"targetRegion":         intent.TargetRegion,
			"pageBlueprintKey":     intent.PageBlueprintKey,
			"pageBlueprintLabel":   intent.PageBlueprintLabel,
			"componentTemplateKey": intent.ComponentTemplateKey,
			"componentLabel":       intent.ComponentLabel,
			"goSXComponent":        intent.GoSXComponent,
			"binding":              intent.Binding,
			"status":               intent.Status,
			"stepCount":            intent.StepCount(),
			"steps":                authoringStepViews(intent.Steps),
			"mutation":             AuthoringMutationView(mutation),
			"formValues":           mutation.FormValues(),
		})
	}
	return out
}

func authoringStepViews(steps []CompositionStep) []map[string]any {
	out := make([]map[string]any, 0, len(steps))
	for _, step := range normalizeCompositionSteps(steps) {
		out = append(out, map[string]any{
			"key":           step.Key,
			"label":         step.Label,
			"summary":       step.Summary,
			"goSXComponent": step.GoSXComponent,
			"binding":       step.Binding,
		})
	}
	return out
}

func authoringWorkspaceView(workspace CompositionWorkspace, canvas WorkspaceCanvas) map[string]any {
	workspace = workspace.Normalize()
	return map[string]any{
		"layers":     authoringWorkspaceLayerViews(workspace.Layers),
		"nodes":      authoringWorkspaceNodeViews(workspace.Nodes),
		"links":      authoringWorkspaceLinkViews(workspace.Links),
		"nodeCount":  workspace.NodeCount(),
		"linkCount":  workspace.LinkCount(),
		"layerCount": workspace.LayerCount(),
		"canvas":     authoringWorkspaceCanvasView(canvas),
	}
}

func authoringWorkspaceLayerViews(layers []WorkspaceLayer) []map[string]any {
	out := make([]map[string]any, 0, len(layers))
	for _, layer := range layers {
		layer.Key = strings.TrimSpace(layer.Key)
		layer.Label = strings.TrimSpace(layer.Label)
		layer.Summary = strings.TrimSpace(layer.Summary)
		if layer.Key == "" || layer.Label == "" {
			continue
		}
		out = append(out, map[string]any{
			"key":      layer.Key,
			"label":    layer.Label,
			"summary":  layer.Summary,
			"nodeKeys": append([]string(nil), layer.NodeKeys...),
		})
	}
	return out
}

func authoringWorkspaceNodeViews(nodes []WorkspaceNode) []map[string]any {
	out := make([]map[string]any, 0, len(nodes))
	for _, node := range nodes {
		node = node.Normalize()
		if node.Key == "" || node.Label == "" {
			continue
		}
		out = append(out, map[string]any{
			"key":           node.Key,
			"label":         node.Label,
			"summary":       node.Summary,
			"kind":          string(node.NormalizedKind()),
			"kindLabel":     WorkspaceNodeKindLabel(node.Kind),
			"layerKey":      node.LayerKey,
			"pageKey":       node.PageKey,
			"route":         node.Route,
			"group":         string(node.Group),
			"groupLabel":    PageGroupLabel(node.Group),
			"goSXComponent": node.GoSXComponent,
			"source":        string(node.Source),
			"sourceLabel":   ComponentSourceLabel(node.Source),
			"binding":       node.Binding,
			"status":        node.Status,
			"selected":      node.Selected,
		})
	}
	return out
}

func authoringWorkspaceLinkViews(links []WorkspaceLink) []map[string]any {
	out := make([]map[string]any, 0, len(links))
	for _, link := range links {
		link = link.Normalize()
		if link.Key == "" || link.FromNodeKey == "" || link.ToNodeKey == "" {
			continue
		}
		out = append(out, map[string]any{
			"key":         link.Key,
			"label":       link.Label,
			"summary":     link.Summary,
			"kind":        string(link.NormalizedKind()),
			"kindLabel":   WorkspaceLinkKindLabel(link.Kind),
			"fromNodeKey": link.FromNodeKey,
			"toNodeKey":   link.ToNodeKey,
		})
	}
	return out
}

func authoringWorkspaceCanvasView(canvas WorkspaceCanvas) map[string]any {
	return map[string]any{
		"viewBox": canvas.ViewBox,
		"nodes":   authoringWorkspacePointViews(canvas.Nodes),
		"links":   authoringWorkspacePathViews(canvas.Links),
	}
}

func authoringWorkspacePointViews(points []WorkspaceNodePoint) []map[string]any {
	out := make([]map[string]any, 0, len(points))
	for _, point := range points {
		if strings.TrimSpace(point.NodeKey) == "" {
			continue
		}
		out = append(out, map[string]any{
			"nodeKey":  point.NodeKey,
			"layerKey": point.LayerKey,
			"x":        point.X,
			"y":        point.Y,
		})
	}
	return out
}

func authoringWorkspacePathViews(paths []WorkspaceLinkPath) []map[string]any {
	out := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path.Key) == "" {
			continue
		}
		out = append(out, map[string]any{
			"key":         path.Key,
			"kind":        string(path.Kind),
			"kindLabel":   WorkspaceLinkKindLabel(path.Kind),
			"fromNodeKey": path.FromNodeKey,
			"toNodeKey":   path.ToNodeKey,
			"path":        path.Path,
			"fromX":       path.FromX,
			"fromY":       path.FromY,
			"toX":         path.ToX,
			"toY":         path.ToY,
		})
	}
	return out
}

func authoringPageGroupCountViews(counts []PageGroupCount) []map[string]any {
	out := make([]map[string]any, 0, len(counts))
	for _, count := range counts {
		out = append(out, map[string]any{
			"group": string(count.Group),
			"label": count.Label,
			"count": count.Count,
		})
	}
	return out
}

func normalizeCompositionIntents(values []CompositionIntent) []CompositionIntent {
	out := make([]CompositionIntent, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
