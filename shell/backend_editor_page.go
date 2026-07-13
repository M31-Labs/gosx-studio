package shell

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/canvas"
	"m31labs.dev/gosx-studio/panels"
	"m31labs.dev/gosx-studio/sitemap"
)

type BackendEditorPageProps struct {
	Class                        string
	Media                        []BackendEditorMediaAsset
	SaveStatus                   BackendEditorActionStatus
	AuthoringStatus              BackendEditorActionStatus
	PublishStatus                BackendEditorActionStatus
	PublishFlowStatus            BackendEditorActionStatus
	RestoreStatus                BackendEditorActionStatus
	RevisionRestored             bool
	WorkbenchShell               gosx.Node
	SupportNodes                 []gosx.Node
	SiteMapView                  map[string]any
	SiteMapAuthoringFormsOptions sitemap.SiteMapAuthoringFormsOptions
	SiteMapAuthoringFormsNode    gosx.Node
	StylePanelView               map[string]any
	StylePanelFormID             string
	StylePanelAction             string
	StylePanelCSRFToken          string
	StylePanelNode               gosx.Node
	EngineHosts                  []map[string]any
	EngineRuntime                canvas.StudioEngineRuntime
	EngineHostsNode              gosx.Node
	Scripts                      BackendEditorScripts
}

type BackendEditorMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendEditorActionStatus struct {
	Submitted bool
	OK        bool
	Message   string
}

type BackendEditorScripts struct {
	WorkbenchRuntime string
	CommandRuntime   string
	StateRuntime     string
	EngineRuntime    string
}

func RenderBackendEditorPage(props BackendEditorPageProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page editor-page"
	}
	children := []gosx.Node{
		RenderBackendEditorMediaDatalist(props.Media),
		RenderBackendEditorStatuses(props),
		props.WorkbenchShell,
		gosx.Fragment(backendEditorSupportNodes(props)...),
		RenderBackendEditorRuntimeScripts(props.Scripts),
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-editor-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func backendEditorSupportNodes(props BackendEditorPageProps) []gosx.Node {
	nodes := append([]gosx.Node{}, props.SupportNodes...)
	if siteMapAuthoringForms := backendEditorSiteMapAuthoringFormsNode(props); !siteMapAuthoringForms.IsZero() {
		nodes = append(nodes, siteMapAuthoringForms)
	}
	if stylePanel := backendEditorStylePanelNode(props); !stylePanel.IsZero() {
		nodes = append(nodes, stylePanel)
	}
	if !props.EngineHostsNode.IsZero() {
		return append(nodes, props.EngineHostsNode)
	}
	if len(props.EngineHosts) > 0 {
		nodes = append(nodes, canvas.RenderStudioEngineHosts(props.EngineHosts, canvas.StudioEngineHostsOptions{
			EngineRuntime: props.EngineRuntime,
		}))
	}
	return nodes
}

func backendEditorSiteMapAuthoringFormsNode(props BackendEditorPageProps) gosx.Node {
	if !props.SiteMapAuthoringFormsNode.IsZero() {
		return props.SiteMapAuthoringFormsNode
	}
	if len(props.SiteMapView) == 0 || props.SiteMapAuthoringFormsOptions.Action == "" {
		return gosx.Node{}
	}
	return sitemap.RenderSiteMapAuthoringForms(props.SiteMapView, props.SiteMapAuthoringFormsOptions)
}

func backendEditorStylePanelNode(props BackendEditorPageProps) gosx.Node {
	if !props.StylePanelNode.IsZero() {
		return props.StylePanelNode
	}
	if len(props.StylePanelView) == 0 || props.StylePanelAction == "" {
		return gosx.Node{}
	}
	formID := props.StylePanelFormID
	if formID == "" {
		formID = "editorStylePaletteForm"
	}
	panel := panels.RenderStylePanel(props.StylePanelView, formID, props.StylePanelAction, props.StylePanelCSRFToken)
	if panel.IsZero() {
		return panel
	}
	return supportModeGatePanel("look", panel)
}

// supportModeGatePanel wraps a support node (rendered as a sibling of the
// workbench <form> — see backendEditorSupportNodes) in the SAME mode-gate
// convention muddy-noni-commerce's own editorGatedModePanel already uses for
// its host-composed support nodes (direct-edit forms, the interactions
// inspector, ...): a data-studio-mode-panel-tagged .editor-panel wrapper,
// carrying data-studio-mode-gate-wrapper="true" so it gets the shared
// support-panel width treatment (studio.css) and so
// workbenchruntime/island_runtime.js's setModeIsland (which widens its
// [data-studio-mode-panel] search to
// [data-gosx-studio-backend-editor-renderer] specifically so it can reach
// these out-of-form siblings) hides it outside the named mode.
//
// Wave 3B (tamarind): the color-token/font style panel
// (backendEditorStylePanelNode) previously rendered unconditionally in every
// mode — it carried no data-studio-mode-panel at all, so it was composed as
// a permanent, un-gated fixture at the bottom of every mode's page (visually
// "dumped at HOME's bottom" simply because HOME is the default mode, but it
// rendered identically under LOOK/PREVIEW/PUBLISH/ADVANCED too — confirmed
// live: stylePanelModeTag was null and stylePanelVis was true in all five
// modes before this fix). It belongs to LOOK (the raw token editor is a
// second, ungrouped surface for the same "visual system" LOOK's own grouped
// RenderLookPanel already owns as the primary one) so it is now gated there.
func supportModeGatePanel(mode string, node gosx.Node) gosx.Node {
	if node.IsZero() {
		return node
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "editor-panel editor-panel--mode-gate"),
		gosx.Attr("data-studio-mode-panel", mode),
		gosx.Attr("data-studio-mode-gate-wrapper", "true"),
	), node)
}

func RenderBackendEditorMediaDatalist(media []BackendEditorMediaAsset) gosx.Node {
	nodes := make([]gosx.Node, 0, len(media))
	for _, asset := range media {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", asset.URL),
			gosx.Attr("label", asset.Filename),
			gosx.Attr("data-media-alt", asset.Alt),
		), gosx.Text(asset.Filename)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "editor-media-urls")), gosx.Fragment(nodes...))
}

func RenderBackendEditorStatuses(props BackendEditorPageProps) gosx.Node {
	nodes := []gosx.Node{}
	for _, status := range []BackendEditorActionStatus{
		props.SaveStatus,
		props.AuthoringStatus,
		props.PublishStatus,
		props.PublishFlowStatus,
	} {
		nodes = append(nodes, renderBackendEditorActionStatus(status)...)
	}
	if props.RestoreStatus.Submitted {
		nodes = append(nodes, renderBackendEditorStatus("form-status form-status--error", props.RestoreStatus.Message))
	}
	if props.RevisionRestored {
		nodes = append(nodes, renderBackendEditorStatus("form-status form-status--ok", "Editor settings restored."))
	}
	return gosx.Fragment(nodes...)
}

func RenderBackendEditorRuntimeScripts(scripts BackendEditorScripts) gosx.Node {
	return gosx.Fragment(
		gosx.El("script", gosx.Attrs(
			gosx.Attr("src", scripts.WorkbenchRuntime),
			gosx.Attr("defer", true),
			gosx.Attr("data-gosx-studio-workbench-runtime", "true"),
		)),
		gosx.El("script", gosx.Attrs(
			gosx.Attr("src", scripts.CommandRuntime),
			gosx.Attr("defer", true),
			gosx.Attr("data-gosx-studio-command-runtime", "true"),
		)),
		gosx.El("script", gosx.Attrs(
			gosx.Attr("src", scripts.StateRuntime),
			gosx.Attr("defer", true),
			gosx.Attr("data-gosx-studio-state-runtime", "true"),
		)),
		gosx.El("script", gosx.Attrs(
			gosx.Attr("src", scripts.EngineRuntime),
			gosx.Attr("defer", true),
			gosx.Attr("data-gosx-studio-engine-runtime", "true"),
		)),
	)
}

func renderBackendEditorActionStatus(status BackendEditorActionStatus) []gosx.Node {
	if status.OK {
		return []gosx.Node{renderBackendEditorStatus("form-status form-status--ok", status.Message)}
	}
	if status.Submitted {
		return []gosx.Node{renderBackendEditorStatus("form-status form-status--error", status.Message)}
	}
	return nil
}

func renderBackendEditorStatus(className, message string) gosx.Node {
	return gosx.El("p", gosx.Attrs(gosx.Attr("class", className)), gosx.Text(message))
}
