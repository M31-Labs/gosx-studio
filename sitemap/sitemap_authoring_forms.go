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

// RenderSiteMapAuthoringForms renders the SINGLE canonical hidden-forms
// container for a site-map view: one form per composition intent, one per
// shared-component palette entry (see renderSharedComponentPaletteFormNodes),
// and the fixed operation forms below. A page must mount this (or
// RenderSiteMapEngine, which already calls it) AT MOST ONCE per rendered
// site-map view — every visible submit control anywhere on the page
// references one of these forms by id via `form="..."` (a plain, valid HTML
// cross-DOM reference; the two elements need not be nearby). Rendering a
// SECOND container for a DIFFERENTLY-SCOPED view (e.g. a host-specific
// shared-component palette scoped to a page the main view can't select —
// see RenderSharedComponentPalettePanel's doc comment) must use
// RenderSharedComponentPaletteForms instead, which wraps in its OWN,
// differently-named container — never call this function twice on the same
// page: duplicate `data-studio-composition-intent-forms` containers is a
// real product bug (duplicate form ids make browsers bind the FIRST match,
// silently misrouting any submission whose id collides).
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
	// One hidden form per shared-component palette entry
	// (AuthoringSiteMapComponentDefinitionPaletteViews), matching the
	// composition-intent forms above — renderSiteMapSharedPaletteEntry's
	// visible fields/button reference this same id via `form=`. A host whose
	// main site-map view legitimately carries shared entries gets them here,
	// in the SAME single container, for free.
	nodes = append(nodes, renderSharedComponentPaletteFormNodes(siteMapView, options)...)
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

// RenderSharedComponentPaletteForms renders JUST the hidden `<form id=...>`
// elements for siteMapView's shared-component palette entries
// (siteMapSharedPaletteEntries — the isShared=true "palette" items), wrapped
// in their OWN container (data-studio-shared-component-palette-forms), NEVER
// data-studio-composition-intent-forms. Use this — not
// RenderSiteMapAuthoringForms — for a dedicated, differently-page-scoped
// shared-component palette view mounted ALONGSIDE a main site-map engine
// that already renders its own RenderSiteMapAuthoringForms container; see
// RenderSharedComponentPalettePanel's doc comment for why a host might need
// a second, differently-scoped view in the first place. A host whose main
// site-map view already carries the shared entries (real page selection, no
// second view needed) never calls this function — RenderSiteMapAuthoringForms
// already renders them, in the same single container, via
// renderSharedComponentPaletteFormNodes.
func RenderSharedComponentPaletteForms(siteMapView map[string]any, options SiteMapAuthoringFormsOptions) gosx.Node {
	nodes := renderSharedComponentPaletteFormNodes(siteMapView, options)
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("hidden", ""),
		gosx.Attr("class", strings.TrimSpace(options.ContainerClass)),
		gosx.Attr("data-studio-shared-component-palette-forms", "true"),
		gosx.Attr("data-gosx-studio-authoring-forms-renderer", "gosx-studio"),
	), gosx.Fragment(nodes...))
}

// renderSharedComponentPaletteFormNodes builds the hidden form NODES (no
// wrapping container) for every shared-component palette entry in
// siteMapView — the single implementation both RenderSiteMapAuthoringForms
// (embedded in its own container) and RenderSharedComponentPaletteForms
// (its own, separate container) call, so the two never drift.
func renderSharedComponentPaletteFormNodes(siteMapView map[string]any, options SiteMapAuthoringFormsOptions) []gosx.Node {
	entries := siteMapSharedPaletteEntries(siteMapView)
	nodes := make([]gosx.Node, 0, len(entries))
	for _, entry := range entries {
		formID := core.WorkbenchViewString(entry, "formID")
		if formID == "" {
			continue
		}
		nodes = append(nodes, renderSiteMapAuthoringForm(formID, options.Action, options, gosx.Attr("data-studio-shared-component-place-form", core.WorkbenchViewString(entry, "definitionKey"))))
	}
	return nodes
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
