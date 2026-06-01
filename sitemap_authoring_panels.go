package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type SiteMapAuthoringPanelsOptions struct {
	PanelClass       string
	MetadataClass    string
	ControlClass     string
	ReorderClass     string
	DuplicateClass   string
	VisibilityClass  string
	DeleteClass      string
	IntentClass      string
	IntentItemClass  string
	PageListClass    string
	EmptyClass       string
	MetadataFormID   string
	ControlFormID    string
	ReorderFormID    string
	DuplicateFormID  string
	VisibilityFormID string
	DeleteFormID     string

	IncludePageListInCompositionPanel bool
}

const (
	defaultSiteMapMetadataFormID   = "studioSiteMapMetadataForm"
	defaultSiteMapControlFormID    = "studioSiteMapEditableControlForm"
	defaultSiteMapReorderFormID    = "studioSiteMapReorderForm"
	defaultSiteMapDuplicateFormID  = "studioSiteMapDuplicateForm"
	defaultSiteMapVisibilityFormID = "studioSiteMapVisibilityForm"
	defaultSiteMapDeleteFormID     = "studioSiteMapDeleteForm"
)

func RenderSiteMapAuthoringPanels(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	nodes := []gosx.Node{
		renderSiteMapPageMetadataPanel(siteMapView, options),
		renderSiteMapEditableControlPanel(siteMapView, options),
		renderSiteMapReorderPanel(siteMapView, options),
		renderSiteMapDuplicatePanel(siteMapView, options),
		renderSiteMapVisibilityPanel(siteMapView, options),
		renderSiteMapDeletePanel(siteMapView, options),
		renderSiteMapCompositionIntentPanel(siteMapView, options),
	}
	out := make([]gosx.Node, 0, len(nodes))
	for _, node := range nodes {
		if workbenchNodeEmpty(node) {
			continue
		}
		out = append(out, node)
	}
	return gosx.Fragment(out...)
}

func renderSiteMapPageMetadataPanel(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	if !workbenchViewBool(siteMapView, "hasMetadataPage") {
		return gosx.Fragment()
	}
	page := workbenchViewMap(siteMapView, "metadataPage")
	formID := FirstNonEmpty(options.MetadataFormID, defaultSiteMapMetadataFormID)
	children := []gosx.Node{
		renderSiteMapPanelHeader("Page details", workbenchMapString(page, "groupLabel"), workbenchMapString(page, "statusLabel")),
	}
	children = append(children, siteMapHiddenInputs(formID, siteMapInputViews(page, "formInputs"), AuthoringFieldPageLabel, AuthoringFieldPageRoute)...)
	children = append(children,
		gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Title")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "text"),
				gosx.Attr("name", AuthoringFieldPageLabel),
				gosx.Attr("value", workbenchMapString(page, "authoringPageLabel")),
			)),
		),
		gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Route")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "text"),
				gosx.Attr("name", AuthoringFieldPageRoute),
				gosx.Attr("value", workbenchMapString(page, "authoringPageRoute")),
			)),
		),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("form", formID),
			gosx.Attr("type", "submit"),
		), gosx.Text("Save page")),
	)
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.MetadataClass, options.PanelClass, "studio-site-map-page-edit studio-site-map-builder__panel")),
		gosx.Attr("data-gosx-studio-page-metadata", "true"),
		gosx.Attr("data-studio-site-map-page-edit", "true"),
		gosx.Attr("data-studio-site-map-page", workbenchMapString(page, "key")),
		gosx.Attr("data-gosx-studio-authoring-panel-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func renderSiteMapEditableControlPanel(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	if !workbenchViewBool(siteMapView, "hasEditableControl") {
		return gosx.Fragment()
	}
	control := workbenchViewMap(siteMapView, "editableControl")
	inputName := FirstNonEmpty(workbenchMapString(control, "inputName"), AuthoringFieldValue)
	formID := FirstNonEmpty(options.ControlFormID, defaultSiteMapControlFormID)
	children := []gosx.Node{
		renderSiteMapPanelHeader(workbenchMapString(control, "label"), workbenchMapString(control, "componentLabel"), workbenchMapString(control, "kindLabel")),
	}
	children = append(children, siteMapHiddenInputs(formID, siteMapInputViews(control, "formInputs"), inputName)...)
	children = append(children,
		gosx.El("label", nil,
			gosx.El("span", nil, gosx.Text("Value")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("form", formID),
				gosx.Attr("type", "text"),
				gosx.Attr("name", inputName),
				gosx.Attr("value", workbenchMapString(control, "value")),
			)),
		),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("form", formID),
			gosx.Attr("type", "submit"),
		), gosx.Text(FirstNonEmpty(workbenchMapString(control, "actionLabel"), "Save field"))),
	)
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.ControlClass, options.PanelClass, "studio-site-map-page-edit studio-site-map-builder__panel")),
		gosx.Attr("data-gosx-studio-editable-control", "true"),
		gosx.Attr("data-studio-site-map-page", workbenchMapString(control, "pageKey")),
		gosx.Attr("data-studio-site-map-component", workbenchMapString(control, "componentKey")),
		gosx.Attr("data-studio-site-map-control", workbenchMapString(control, "key")),
		gosx.Attr("data-gosx-studio-authoring-panel-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func renderSiteMapReorderPanel(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	return renderSiteMapComponentActionPanel(siteMapView, componentPanelOptions{
		ViewKey:        "reorderComponent",
		EnabledKey:     "hasReorderComponent",
		FormID:         FirstNonEmpty(options.ReorderFormID, defaultSiteMapReorderFormID),
		ClassName:      FirstNonEmpty(options.ReorderClass, options.PanelClass, "studio-site-map-page-edit studio-site-map-builder__panel"),
		Title:          "Section order",
		Empty:          "No section order changes are available.",
		DataAttr:       "data-gosx-studio-component-reorder",
		LegacyDataAttr: "data-studio-site-map-component-reorder",
		OutputPrefix:   "#",
	})
}

func renderSiteMapDuplicatePanel(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	return renderSiteMapComponentActionPanel(siteMapView, componentPanelOptions{
		ViewKey:        "duplicateComponent",
		EnabledKey:     "hasDuplicateComponent",
		FormID:         FirstNonEmpty(options.DuplicateFormID, defaultSiteMapDuplicateFormID),
		ClassName:      FirstNonEmpty(options.DuplicateClass, options.PanelClass, "studio-site-map-page-edit studio-site-map-builder__panel"),
		Title:          "Duplicate section",
		Empty:          "No duplicatable sections are available.",
		DataAttr:       "data-gosx-studio-component-duplicate",
		LegacyDataAttr: "data-studio-site-map-component-duplicate",
		OutputPrefix:   "#",
	})
}

func renderSiteMapVisibilityPanel(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	return renderSiteMapComponentActionPanel(siteMapView, componentPanelOptions{
		ViewKey:        "visibilityComponent",
		EnabledKey:     "hasVisibilityComponent",
		FormID:         FirstNonEmpty(options.VisibilityFormID, defaultSiteMapVisibilityFormID),
		ClassName:      FirstNonEmpty(options.VisibilityClass, options.PanelClass, "studio-site-map-page-edit studio-site-map-builder__panel"),
		Title:          "Section visibility",
		Empty:          "No editable section visibility controls are available.",
		DataAttr:       "data-gosx-studio-component-visibility",
		LegacyDataAttr: "data-studio-site-map-component-visibility",
		StatusKey:      "statusLabel",
	})
}

func renderSiteMapDeletePanel(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	return renderSiteMapComponentActionPanel(siteMapView, componentPanelOptions{
		ViewKey:        "deleteComponent",
		EnabledKey:     "hasDeleteComponent",
		FormID:         FirstNonEmpty(options.DeleteFormID, defaultSiteMapDeleteFormID),
		ClassName:      FirstNonEmpty(options.DeleteClass, options.PanelClass, "studio-site-map-page-edit studio-site-map-builder__panel"),
		Title:          "Remove section",
		Empty:          "No removable sections are available.",
		DataAttr:       "data-gosx-studio-component-delete",
		LegacyDataAttr: "data-studio-site-map-component-delete",
		OutputPrefix:   "#",
	})
}

type componentPanelOptions struct {
	ViewKey        string
	EnabledKey     string
	FormID         string
	ClassName      string
	Title          string
	Empty          string
	DataAttr       string
	LegacyDataAttr string
	StatusKey      string
	OutputPrefix   string
}

func renderSiteMapComponentActionPanel(siteMapView map[string]any, options componentPanelOptions) gosx.Node {
	if !workbenchViewBool(siteMapView, options.EnabledKey) {
		return gosx.Fragment()
	}
	component := workbenchViewMap(siteMapView, options.ViewKey)
	status := workbenchMapString(component, options.StatusKey)
	if status == "" {
		status = options.OutputPrefix + workbenchMapString(component, "positionLabel")
	}
	children := []gosx.Node{
		renderSiteMapPanelHeader(options.Title, workbenchMapString(component, "pageLabel"), status),
	}
	children = append(children, siteMapHiddenInputs(options.FormID, siteMapInputViews(component, "formInputs"))...)
	children = append(children,
		gosx.El("p", nil, gosx.Text(workbenchMapString(component, "label"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("form", options.FormID),
			gosx.Attr("type", "submit"),
		), gosx.Text(FirstNonEmpty(workbenchMapString(component, "actionLabel"), "Apply")+" section")),
	)
	attrs := []any{
		gosx.Attr("class", options.ClassName),
		gosx.Attr(options.DataAttr, "true"),
		gosx.Attr("data-studio-site-map-page", workbenchMapString(component, "pageKey")),
		gosx.Attr("data-studio-site-map-component", workbenchMapString(component, "key")),
		gosx.Attr("data-gosx-studio-authoring-panel-renderer", "gosx-studio"),
	}
	if options.LegacyDataAttr != "" {
		attrs = append(attrs, gosx.Attr(options.LegacyDataAttr, "true"))
	}
	return gosx.El("section", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func renderSiteMapCompositionIntentPanel(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	if !workbenchViewBool(siteMapView, "hasCompositionIntents") {
		return gosx.Fragment()
	}
	children := []gosx.Node{
		renderSiteMapPanelHeader("Ready to apply", workbenchViewString(siteMapView, "selectedPageLabel"), workbenchViewString(siteMapView, "compositionIntentCountLabel")),
	}
	if options.IncludePageListInCompositionPanel {
		children = append(children, renderSiteMapPageList(siteMapView, options))
	}
	for _, intent := range workbenchViewMapList(siteMapView, "compositionIntents") {
		children = append(children, renderSiteMapCompositionIntent(intent, options))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.IntentClass, options.PanelClass, "studio-site-map-page-edit studio-site-map-builder__panel")),
		gosx.Attr("data-gosx-studio-composition-intents", "true"),
		gosx.Attr("data-studio-composition-intent-panel", "true"),
		gosx.Attr("data-gosx-studio-authoring-panel-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func renderSiteMapPageList(siteMapView map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	pages := workbenchViewMapList(siteMapView, "pages")
	if len(pages) == 0 {
		return gosx.El("p", gosx.Attrs(gosx.Attr("class", FirstNonEmpty(options.EmptyClass, "empty"))), gosx.Text("No pages are available yet."))
	}
	children := []gosx.Node{
		renderSiteMapPanelHeader("Current pages", workbenchViewString(siteMapView, "componentCountLabel"), workbenchViewString(siteMapView, "pageCountLabel")),
	}
	for _, page := range pages {
		componentNodes := []gosx.Node{}
		for _, component := range siteMapMapList(page, "components") {
			componentNodes = append(componentNodes, gosx.El("span", gosx.Attrs(
				gosx.Attr("data-studio-site-map-component", workbenchMapString(component, "key")),
			), gosx.Text(workbenchMapString(component, "label"))))
		}
		pageChildren := []gosx.Node{
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(workbenchMapString(page, "label"))),
				gosx.El("small", nil, gosx.Text(workbenchMapString(page, "route"))),
			),
			gosx.El("output", nil, gosx.Text(workbenchMapString(page, "componentCountLabel"))),
		}
		if len(componentNodes) > 0 {
			pageChildren = append(pageChildren, gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-site-map-page-components")), gosx.Fragment(componentNodes...)))
		}
		children = append(children, gosx.El("article", gosx.Attrs(
			gosx.Attr("class", "studio-site-map-page-summary"),
			gosx.Attr("data-studio-site-map-page", workbenchMapString(page, "key")),
		), gosx.Fragment(pageChildren...)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.PageListClass, "studio-site-map-page-list")),
		gosx.Attr("data-gosx-studio-page-list", "true"),
		gosx.Attr("data-gosx-studio-page-list-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func renderSiteMapCompositionIntent(intent map[string]any, options SiteMapAuthoringPanelsOptions) gosx.Node {
	children := []gosx.Node{}
	for _, input := range siteMapInputViews(intent, "formInputs") {
		children = append(children, gosx.El("input", gosx.Attrs(
			gosx.Attr("form", workbenchMapString(intent, "formID")),
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", input["name"]),
			gosx.Attr("value", input["value"]),
		)))
	}
	children = append(children,
		gosx.El("div", nil,
			gosx.El("strong", nil, gosx.Text(workbenchMapString(intent, "label"))),
			gosx.El("small", nil, gosx.Text(workbenchMapString(intent, "summary"))),
		),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("form", workbenchMapString(intent, "formID")),
			gosx.Attr("type", "submit"),
		), gosx.Text(FirstNonEmpty(workbenchMapString(intent, "actionLabel"), "Apply draft"))),
	)
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", FirstNonEmpty(options.IntentItemClass, "studio-site-map-intent-form")),
		gosx.Attr("data-studio-composition-intent-apply", "true"),
		gosx.Attr("data-studio-composition-intent", workbenchMapString(intent, "key")),
		gosx.Attr("data-studio-composition-kind", workbenchMapString(intent, "kind")),
	), gosx.Fragment(children...))
}

func renderSiteMapPanelHeader(title, detail, status string) gosx.Node {
	return gosx.El("header", nil,
		gosx.El("div", nil,
			gosx.El("strong", nil, gosx.Text(title)),
			gosx.El("small", nil, gosx.Text(detail)),
		),
		gosx.El("output", nil, gosx.Text(status)),
	)
}

func siteMapHiddenInputs(formID string, inputs []map[string]string, exclude ...string) []gosx.Node {
	excluded := map[string]bool{}
	for _, name := range exclude {
		name = strings.TrimSpace(name)
		if name != "" {
			excluded[name] = true
		}
	}
	nodes := make([]gosx.Node, 0, len(inputs))
	for _, input := range inputs {
		name := strings.TrimSpace(input["name"])
		if name == "" || excluded[name] {
			continue
		}
		nodes = append(nodes, gosx.El("input", gosx.Attrs(
			gosx.Attr("form", formID),
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", name),
			gosx.Attr("value", input["value"]),
		)))
	}
	return nodes
}

func siteMapInputViews(values map[string]any, key string) []map[string]string {
	switch typed := values[key].(type) {
	case []map[string]string:
		return typed
	case []map[string]any:
		out := make([]map[string]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, map[string]string{
				"name":  workbenchMapString(item, "name"),
				"value": workbenchMapString(item, "value"),
			})
		}
		return out
	default:
		return nil
	}
}

func siteMapMapList(values map[string]any, key string) []map[string]any {
	if values == nil {
		return nil
	}
	switch typed := values[key].(type) {
	case []map[string]any:
		return typed
	case []map[string]string:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			next := make(map[string]any, len(item))
			for key, value := range item {
				next[key] = value
			}
			out = append(out, next)
		}
		return out
	default:
		return nil
	}
}
