package studio

import "m31labs.dev/gosx"

type BackendEditorPageProps struct {
	Class               string
	Media               []BackendEditorMediaAsset
	SaveStatus          BackendEditorActionStatus
	AuthoringStatus     BackendEditorActionStatus
	PublishStatus       BackendEditorActionStatus
	PublishFlowStatus   BackendEditorActionStatus
	RestoreStatus       BackendEditorActionStatus
	RevisionRestored    bool
	WorkbenchShell      gosx.Node
	SupportNodes        []gosx.Node
	StylePanelView      map[string]any
	StylePanelFormID    string
	StylePanelAction    string
	StylePanelCSRFToken string
	StylePanelNode      gosx.Node
	EngineHosts         []map[string]any
	EngineRuntime       StudioEngineRuntime
	EngineHostsNode     gosx.Node
	Scripts             BackendEditorScripts
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
	if stylePanel := backendEditorStylePanelNode(props); !stylePanel.IsZero() {
		nodes = append(nodes, stylePanel)
	}
	if !props.EngineHostsNode.IsZero() {
		return append(nodes, props.EngineHostsNode)
	}
	if len(props.EngineHosts) > 0 {
		nodes = append(nodes, RenderStudioEngineHosts(props.EngineHosts, StudioEngineHostsOptions{
			EngineRuntime: props.EngineRuntime,
		}))
	}
	return nodes
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
	return RenderStylePanel(props.StylePanelView, formID, props.StylePanelAction, props.StylePanelCSRFToken)
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
