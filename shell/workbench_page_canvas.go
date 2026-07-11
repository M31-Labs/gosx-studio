package shell

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type WorkbenchPageCanvasRoute struct {
	Key    string
	Label  string
	URL    string
	Active bool
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

	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", core.FirstNonEmpty(options.Class, "editor-preview-shell studio-page-canvas")),
		gosx.Attr("data-gosx-studio-preview", "true"),
		gosx.Attr("data-gosx-studio-page-canvas", "true"),
		gosx.Attr("data-gosx-studio-preview-url", url),
		gosx.Attr("data-gosx-studio-preview-state", "loading"),
		gosx.Attr("data-studio-preview-viewport", core.FirstNonEmpty(core.WorkbenchViewString(options.View, "viewportKey"), "desktop")),
	),
		gosx.El("header", gosx.Attrs(
			gosx.Attr("class", core.FirstNonEmpty(options.ToolbarClass, "storefront-frame-toolbar studio-page-canvas__toolbar")),
			gosx.Attr("data-studio-preview-toolbar", "true"),
		), gosx.Fragment(toolbarChildren...)),
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
