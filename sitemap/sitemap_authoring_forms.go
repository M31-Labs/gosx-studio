package sitemap

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type SiteMapAuthoringFormsOptions struct {
	Action string

	CSRFToken string
	CSRFName  string

	ContainerClass   string
	MetadataFormID   string
	ControlFormID    string
	ReorderFormID    string
	DuplicateFormID  string
	VisibilityFormID string
	DeleteFormID     string
}

func RenderSiteMapAuthoringForms(siteMapView map[string]any, options SiteMapAuthoringFormsOptions) gosx.Node {
	intents := core.WorkbenchViewMapList(siteMapView, "compositionIntents")
	nodes := make([]gosx.Node, 0, len(intents)+6)
	for _, intent := range intents {
		formID := core.WorkbenchViewString(intent, "formID")
		if formID == "" {
			continue
		}
		nodes = append(nodes, renderSiteMapAuthoringForm(formID, options.Action, options, gosx.Attr("data-studio-composition-intent-form", core.WorkbenchViewString(intent, "key"))))
	}
	operationForms := []struct {
		enabled bool
		id      string
		kind    string
	}{
		{core.WorkbenchViewBool(siteMapView, "hasMetadataPage"), core.FirstNonEmpty(options.MetadataFormID, defaultSiteMapMetadataFormID), "metadata"},
		{core.WorkbenchViewBool(siteMapView, "hasEditableControl"), core.FirstNonEmpty(options.ControlFormID, defaultSiteMapControlFormID), "editable-control"},
		{core.WorkbenchViewBool(siteMapView, "hasReorderComponent"), core.FirstNonEmpty(options.ReorderFormID, defaultSiteMapReorderFormID), "reorder"},
		{core.WorkbenchViewBool(siteMapView, "hasDuplicateComponent"), core.FirstNonEmpty(options.DuplicateFormID, defaultSiteMapDuplicateFormID), "duplicate"},
		{core.WorkbenchViewBool(siteMapView, "hasVisibilityComponent"), core.FirstNonEmpty(options.VisibilityFormID, defaultSiteMapVisibilityFormID), "visibility"},
		{core.WorkbenchViewBool(siteMapView, "hasDeleteComponent"), core.FirstNonEmpty(options.DeleteFormID, defaultSiteMapDeleteFormID), "delete"},
	}
	for _, form := range operationForms {
		if !form.enabled {
			continue
		}
		nodes = append(nodes, renderSiteMapAuthoringForm(form.id, options.Action, options, gosx.Attr("data-studio-site-map-operation-form", form.kind)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("hidden", ""),
		gosx.Attr("class", strings.TrimSpace(options.ContainerClass)),
		gosx.Attr("data-studio-composition-intent-forms", "true"),
		gosx.Attr("data-gosx-studio-authoring-forms-renderer", "gosx-studio"),
	), gosx.Fragment(nodes...))
}

func renderSiteMapAuthoringForm(id string, action string, options SiteMapAuthoringFormsOptions, extraAttrs ...any) gosx.Node {
	attrs := []any{
		gosx.Attr("id", strings.TrimSpace(id)),
		gosx.Attr("method", "post"),
		gosx.Attr("data-gosx-studio-authoring-managed", "true"),
		gosx.Attr("data-gosx-form", "true"),
		gosx.Attr("data-gosx-form-state", "idle"),
		gosx.Attr("data-gosx-form-mode", "post"),
		gosx.Attr("data-gosx-enhance", "form"),
		gosx.Attr("data-gosx-fallback", "native-form"),
	}
	if trimmed := strings.TrimSpace(action); trimmed != "" {
		attrs = append(attrs, gosx.Attr("action", trimmed))
	}
	attrs = append(attrs, extraAttrs...)
	children := []gosx.Node{}
	if token := strings.TrimSpace(options.CSRFToken); token != "" {
		children = append(children, gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", core.FirstNonEmpty(options.CSRFName, "csrf_token")),
			gosx.Attr("value", token),
		)))
	}
	return gosx.El("form", gosx.Attrs(attrs...), gosx.Fragment(children...))
}
