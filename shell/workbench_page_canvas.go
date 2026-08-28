package shell

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
	"m31labs.dev/gosx-studio/hostruntime"
)

type WorkbenchPageCanvasRoute struct {
	Key    string
	Label  string
	URL    string
	Active bool
}

type WorkbenchPageCanvasSectionOrderItem struct {
	Key             string
	Label           string
	Locked          bool
	PreviewOptional bool
}

type WorkbenchPageCanvasSectionOrderOptions struct {
	Enabled          bool
	Action           string
	Route            string
	SourceRoute      string
	PageID           string
	Field            string
	ComponentKey     string
	DocumentRevision string
	TargetHead       string
	UndoOperationID  string
	RedoOperationID  string
	StatusLabel      string
	DisabledReason   string
	Items            []WorkbenchPageCanvasSectionOrderItem
}

type WorkbenchPageCanvasOptions struct {
	View         map[string]any
	Class        string
	ToolbarClass string
	FrameClass   string
	RouteClass   string
	URL          string
	Title        string
	IFrameTitle  string
	StatusLabel  string
	Diagnostic   string
	Routes       []WorkbenchPageCanvasRoute
	Controls     []gosx.Node
	Actions      []gosx.Node
	SectionOrder *WorkbenchPageCanvasSectionOrderOptions
}

type WorkbenchPreviewFrameOptions struct {
	Class string
	URL   string
	Title string
}

// RenderWorkbenchPreviewFrame is the single preview-frame attribute contract
// used by both the page canvas and the legacy CMS preview wrapper.
func RenderWorkbenchPreviewFrame(options WorkbenchPreviewFrameOptions) gosx.Node {
	url := core.FirstNonEmpty(strings.TrimSpace(options.URL), "/")
	return gosx.El("iframe", gosx.Attrs(
		gosx.Attr("class", strings.TrimSpace(options.Class)),
		gosx.Attr("src", url),
		gosx.Attr("title", core.FirstNonEmpty(strings.TrimSpace(options.Title), "Page preview")),
		gosx.Attr("data-studio-preview-frame", "true"),
		gosx.Attr("data-studio-preview-src", url),
		gosx.Attr("data-studio-preview-route", normalizedWorkbenchPageCanvasRoute(url)),
	))
}

// RenderWorkbenchPageCanvas renders the host's real same-origin public route.
// The frame is intentionally not a synthetic scene: server-issued identity
// attributes in the public document are the only selection source of truth.
func RenderWorkbenchPageCanvas(options WorkbenchPageCanvasOptions) gosx.Node {
	url := core.FirstNonEmpty(strings.TrimSpace(options.URL), core.WorkbenchViewString(options.View, "previewURL"), "/")
	title := core.FirstNonEmpty(strings.TrimSpace(options.Title), core.WorkbenchViewString(options.View, "previewTitle"), "Page preview")
	iframeTitle := core.FirstNonEmpty(strings.TrimSpace(options.IFrameTitle), core.WorkbenchViewString(options.View, "previewFrameTitle"), title)
	routes := normalizeWorkbenchPageCanvasRoutes(options.Routes, url)
	toolbarChildren := []gosx.Node{
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.RouteClass, "studio-page-canvas__routes")),
			gosx.Attr("role", "toolbar"),
			gosx.Attr("aria-label", "Page route"),
		), renderWorkbenchPageCanvasRouteButtons(routes)),
		RenderWorkbenchViewportControls(options.View, WorkbenchViewportControlsOptions{Class: "studio-viewport-switcher", Label: "Preview viewport"}),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", "studio-selection-readout"),
			gosx.Attr("data-studio-selection-label", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(core.FirstNonEmpty(core.WorkbenchViewString(options.View, "selectionLabel"), "No selection"))),
	}
	toolbarChildren = append(toolbarChildren, options.Controls...)
	toolbarChildren = append(toolbarChildren, options.Actions...)

	canvasChildren := []gosx.Node{
		gosx.El("header", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.ToolbarClass, "storefront-frame-toolbar studio-page-canvas__toolbar")),
			gosx.Attr("data-studio-preview-toolbar", "true"),
		), gosx.Fragment(toolbarChildren...)),
	}
	if options.SectionOrder != nil {
		canvasChildren = append(canvasChildren, renderWorkbenchPageCanvasSectionOrder(*options.SectionOrder, url))
	}
	canvasChildren = append(canvasChildren,
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-page-canvas__stage"),
			gosx.Attr("data-studio-page-canvas-stage", "true"),
		),
			RenderWorkbenchPreviewFrame(WorkbenchPreviewFrameOptions{
				Class: core.FirstNonEmpty(options.FrameClass, "storefront-frame editor-preview-frame"),
				URL:   url,
				Title: iframeTitle,
			}),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", "studio-page-canvas__status"),
			gosx.Attr("data-studio-preview-status", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(core.FirstNonEmpty(options.StatusLabel, "Loading page preview"))),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", "studio-page-canvas__diagnostic"),
			gosx.Attr("data-studio-preview-diagnostic", "true"),
			gosx.Attr("role", "status"),
			gosx.Attr("aria-live", "polite"),
			gosx.Attr("aria-atomic", "true"),
		), gosx.Text(strings.TrimSpace(options.Diagnostic))),
	)

	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "editor-preview-shell studio-page-canvas")),
		gosx.Attr("data-gosx-studio-preview", "true"),
		gosx.Attr("data-gosx-studio-page-canvas", "true"),
		gosx.Attr("data-gosx-studio-preview-url", url),
		gosx.Attr("data-gosx-studio-preview-state", "loading"),
		gosx.Attr("data-studio-preview-viewport", core.FirstNonEmpty(core.WorkbenchViewString(options.View, "viewportKey"), "desktop")),
	), gosx.Fragment(canvasChildren...))
}

func renderWorkbenchPageCanvasSectionOrder(options WorkbenchPageCanvasSectionOrderOptions, previewURL string) gosx.Node {
	route := core.FirstNonEmpty(strings.TrimSpace(options.Route), "/")
	sourceRoute := core.FirstNonEmpty(strings.TrimSpace(options.SourceRoute), route)
	pageID := core.FirstNonEmpty(strings.TrimSpace(options.PageID), "home")
	field := core.FirstNonEmpty(strings.TrimSpace(options.Field), "home.sections.order")
	componentKey := strings.TrimSpace(options.ComponentKey)
	items := normalizeWorkbenchPageCanvasSectionOrderItems(options.Items)
	enabled := options.Enabled && strings.TrimSpace(options.DisabledReason) == "" && len(items) > 0
	attrs := []any{
		gosx.Attr("class", "studio-page-canvas__section-order"),
		gosx.Attr("data-gosx-studio-section-order", "true"),
		gosx.Attr("data-gosx-studio-section-order-enabled", core.BoolAttr(enabled)),
		gosx.Attr("data-gosx-studio-section-order-route", normalizedWorkbenchPageCanvasRoute(route)),
		gosx.Attr("data-gosx-studio-section-order-source-route", normalizedWorkbenchPageCanvasRoute(sourceRoute)),
		gosx.Attr("data-gosx-studio-section-order-preview-route", normalizedWorkbenchPageCanvasRoute(previewURL)),
		gosx.Attr("aria-label", "Section order"),
	}
	if reason := strings.TrimSpace(options.DisabledReason); reason != "" {
		attrs = append(attrs, gosx.Attr("data-gosx-studio-section-order-disabled-reason", reason))
	}
	return gosx.El("aside", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-page-canvas__section-order-header")),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "studio-page-canvas__section-order-title")), gosx.Text("Section order")),
			gosx.El("output", gosx.Attrs(
				gosx.Attr("data-gosx-studio-section-order-status", "true"),
				gosx.Attr("aria-live", "polite"),
				gosx.Attr("role", "status"),
			), gosx.Text(core.FirstNonEmpty(strings.TrimSpace(options.StatusLabel), sectionOrderStatusLabel(enabled, options.DisabledReason)))),
		),
		gosx.El("ol", gosx.Attrs(gosx.Attr("data-gosx-studio-section-order-list", "true")), renderWorkbenchPageCanvasSectionOrderItems(items, enabled)),
		gosx.El("p", gosx.Attrs(
			gosx.Attr("id", "studio-section-order-guidance"),
			gosx.Attr("class", "studio-page-canvas__section-order-guidance"),
		), gosx.Text("Use the drag handle, arrow keys, or Up and Down controls to reorder sections.")),
		renderWorkbenchPageCanvasSectionOrderHistory(options, enabled),
		renderWorkbenchPageCanvasSectionOrderAdapter(options, route, pageID, field, componentKey),
		gosx.El("script", gosx.Attrs(
			gosx.Attr("src", hostruntime.SectionOrderRuntimePath),
			gosx.Attr("defer", "defer"),
			gosx.Attr("data-gosx-studio-section-order-runtime", "true"),
		)),
	)
}

func renderWorkbenchPageCanvasSectionOrderItems(items []WorkbenchPageCanvasSectionOrderItem, enabled bool) gosx.Node {
	nodes := make([]gosx.Node, 0, len(items))
	for index, item := range items {
		key := strings.TrimSpace(item.Key)
		label := core.FirstNonEmpty(strings.TrimSpace(item.Label), key)
		if key == "" {
			continue
		}
		moveUpAttrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-section-order-move", "up"), gosx.Attr("aria-label", "Move "+label+" up")}
		moveDownAttrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-section-order-move", "down"), gosx.Attr("aria-label", "Move "+label+" down")}
		handleAttrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-section-order-handle", "true"), gosx.Attr("aria-label", "Drag "+label), gosx.Attr("aria-describedby", "studio-section-order-guidance")}
		itemUnavailable := item.PreviewOptional
		if !enabled || item.Locked || itemUnavailable || index == 0 {
			moveUpAttrs = append(moveUpAttrs, gosx.Attr("disabled", "disabled"))
		}
		if !enabled || item.Locked || itemUnavailable || index == len(items)-1 {
			moveDownAttrs = append(moveDownAttrs, gosx.Attr("disabled", "disabled"))
		}
		if !enabled || item.Locked || itemUnavailable {
			handleAttrs = append(handleAttrs, gosx.Attr("disabled", "disabled"))
		}
		itemAttrs := []any{
			gosx.Attr("data-gosx-studio-section-order-item", key),
			gosx.Attr("data-gosx-studio-section-order-label", label),
			gosx.Attr("data-gosx-studio-section-order-locked", core.BoolAttr(item.Locked)),
			gosx.Attr("data-gosx-studio-section-order-preview-optional", core.BoolAttr(item.PreviewOptional)),
		}
		nodes = append(nodes, gosx.El("li", gosx.Attrs(
			itemAttrs...,
		),
			gosx.El("button", gosx.Attrs(handleAttrs...), gosx.Text("Drag")),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-gosx-studio-section-order-name", "true")), gosx.Text(label)),
			gosx.El("button", gosx.Attrs(moveUpAttrs...), gosx.Text("Up")),
			gosx.El("button", gosx.Attrs(moveDownAttrs...), gosx.Text("Down")),
		))
	}
	return gosx.Fragment(nodes...)
}

func normalizeWorkbenchPageCanvasSectionOrderItems(items []WorkbenchPageCanvasSectionOrderItem) []WorkbenchPageCanvasSectionOrderItem {
	out := make([]WorkbenchPageCanvasSectionOrderItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item.Key = strings.TrimSpace(item.Key)
		if item.Key == "" {
			continue
		}
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		item.Label = core.FirstNonEmpty(strings.TrimSpace(item.Label), item.Key)
		out = append(out, item)
	}
	return out
}

func renderWorkbenchPageCanvasSectionOrderHistory(options WorkbenchPageCanvasSectionOrderOptions, enabled bool) gosx.Node {
	undoAttrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-section-order-history", "undo"), gosx.Attr("data-gosx-studio-history-operation-id", strings.TrimSpace(options.UndoOperationID))}
	redoAttrs := []any{gosx.Attr("type", "button"), gosx.Attr("data-gosx-studio-section-order-history", "redo"), gosx.Attr("data-gosx-studio-history-operation-id", strings.TrimSpace(options.RedoOperationID))}
	if !enabled || strings.TrimSpace(options.UndoOperationID) == "" {
		undoAttrs = append(undoAttrs, gosx.Attr("disabled", "disabled"))
	}
	if !enabled || strings.TrimSpace(options.RedoOperationID) == "" {
		redoAttrs = append(redoAttrs, gosx.Attr("disabled", "disabled"))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-section-order-controls", "true")),
		gosx.El("button", gosx.Attrs(undoAttrs...), gosx.Text("Undo reorder")),
		gosx.El("button", gosx.Attrs(redoAttrs...), gosx.Text("Redo reorder")),
	)
}

func renderWorkbenchPageCanvasSectionOrderAdapter(options WorkbenchPageCanvasSectionOrderOptions, route, pageID, field, componentKey string) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("data-gosx-studio-section-order-form", "true"),
		gosx.Attr("data-gosx-studio-operation-action", strings.TrimSpace(options.Action)),
		gosx.Attr("data-studio-document-revision", strings.TrimSpace(options.DocumentRevision)),
		gosx.Attr("data-studio-target-head", strings.TrimSpace(options.TargetHead)),
		gosx.Attr("data-studio-target-route", normalizedWorkbenchPageCanvasRoute(route)),
		gosx.Attr("data-studio-target-page-id", pageID),
		gosx.Attr("data-studio-target-field", field),
		gosx.Attr("data-studio-target-component", componentKey),
		gosx.Attr("hidden", "hidden"),
	))
}

func sectionOrderStatusLabel(enabled bool, disabledReason string) string {
	if reason := strings.TrimSpace(disabledReason); reason != "" {
		return reason
	}
	if enabled {
		return "Drag sections or use Up and Down."
	}
	return "Section ordering is read-only for this route."
}

func normalizeWorkbenchPageCanvasRoutes(routes []WorkbenchPageCanvasRoute, activeURL string) []WorkbenchPageCanvasRoute {
	activeRoute := normalizedWorkbenchPageCanvasRoute(activeURL)
	out := make([]WorkbenchPageCanvasRoute, 0, len(routes)+1)
	seen := map[string]bool{}
	activeSet := false
	for _, route := range routes {
		route.Key = core.NormalizeKey(route.Key)
		route.Label = strings.TrimSpace(route.Label)
		route.URL = strings.TrimSpace(route.URL)
		path := normalizedWorkbenchPageCanvasRoute(route.URL)
		if route.URL == "" || route.Label == "" || seen[path] {
			continue
		}
		if route.Key == "" {
			route.Key = core.NormalizeKey(route.Label)
		}
		if route.Active || path == activeRoute {
			route.Active = !activeSet
			activeSet = true
		}
		seen[path] = true
		out = append(out, route)
	}
	if len(out) == 0 {
		out = append(out, WorkbenchPageCanvasRoute{Key: "page", Label: "Page", URL: activeURL, Active: true})
		return out
	}
	if !activeSet {
		out[0].Active = true
	}
	return out
}

func normalizedWorkbenchPageCanvasRoute(url string) string {
	return (core.CanvasIdentity{Route: url}).Normalize().Route
}

func renderWorkbenchPageCanvasRouteButtons(routes []WorkbenchPageCanvasRoute) gosx.Node {
	nodes := make([]gosx.Node, 0, len(routes))
	for _, route := range routes {
		nodes = append(nodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-preview-route", route.URL),
			gosx.Attr("data-studio-preview-route-key", route.Key),
			gosx.Attr("aria-pressed", core.BoolAttr(route.Active)),
		), gosx.Text(route.Label)))
	}
	return gosx.Fragment(nodes...)
}
