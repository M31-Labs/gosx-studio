package studio

import "m31labs.dev/gosx"

type BackendEditorWorkbenchProps struct {
	View         map[string]any
	AuthoringURL string
	Toolbar      gosx.Node
	CanvasBar    gosx.Node
	CanvasStatus gosx.Node
	LeftRail     []gosx.Node
	Board        []gosx.Node
	RightRail    []gosx.Node
}

type BackendEditorWorkbenchContentProps struct {
	View            map[string]any
	AuthoringURL    string
	Toolbar         gosx.Node
	CanvasBar       gosx.Node
	CanvasStatus    gosx.Node
	SiteNavigator   gosx.Node
	BlockLayout     gosx.Node
	SiteMapEngine   gosx.Node
	SiteMapCanvas   gosx.Node
	InspectorChrome gosx.Node
	HomeInspector   gosx.Node
	LookPanel       gosx.Node
	BrandPanel      gosx.Node
	NavigationPanel gosx.Node
	CheckoutPanel   gosx.Node
	PublishPanel    gosx.Node
	AdvancedPanel   gosx.Node
}

type BackendEditorWorkbenchPanelStackProps struct {
	View         map[string]any
	AuthoringURL string
	Toolbar      gosx.Node
	CanvasBar    gosx.Node
	CanvasStatus gosx.Node

	SiteNavigator               gosx.Node
	HomeLayers                  map[string]any
	HomeLayerSelection          gosx.Node
	BlockLayoutEngineHost       gosx.Node
	BlockLibraryPanel           gosx.Node
	SiteMapEngine               gosx.Node
	SiteMapCanvas               gosx.Node
	InspectorChrome             gosx.Node
	HomeInspector               gosx.Node
	LookPanel                   gosx.Node
	BrandPanel                  map[string]any
	BrandFields                 map[string]any
	BrandMediaPicker            gosx.Node
	NavigationPanel             gosx.Node
	CheckoutPanel               gosx.Node
	PublishPanel                gosx.Node
	AdvancedPanel               map[string]any
	FlowDesigner                gosx.Node
	AdvancedToolsPanel          gosx.Node
	AdvancedWorkspaceFieldPanel gosx.Node
	AdvancedCalendarFieldPanel  gosx.Node
	AdvancedTypeFieldPanel      gosx.Node
}

func RenderBackendEditorWorkbench(props BackendEditorWorkbenchProps) gosx.Node {
	options := WorkbenchFrameOptions{
		AuthoringURL:   props.AuthoringURL,
		ResizableRails: true,
		Toolbar:        props.Toolbar,
		LeftRail:       []gosx.Node{RenderBackendEditorLeftRail(props.LeftRail...)},
		Board:          []gosx.Node{RenderBackendEditorBoard(props.Board...)},
		RightRail:      []gosx.Node{RenderBackendEditorRightRail(props.RightRail...)},
	}
	if !workbenchNodeEmpty(props.CanvasBar) {
		options.CanvasBar = []gosx.Node{props.CanvasBar}
	}
	if !workbenchNodeEmpty(props.CanvasStatus) {
		options.CanvasStatus = []gosx.Node{props.CanvasStatus}
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-frame-slot", "true")),
		RenderWorkbenchFrame(props.View, options),
	)
}

func RenderBackendEditorWorkbenchContent(props BackendEditorWorkbenchContentProps) gosx.Node {
	return RenderBackendEditorWorkbench(BackendEditorWorkbenchProps{
		View:         props.View,
		AuthoringURL: props.AuthoringURL,
		Toolbar:      props.Toolbar,
		CanvasBar:    props.CanvasBar,
		CanvasStatus: props.CanvasStatus,
		LeftRail: []gosx.Node{
			props.SiteNavigator,
			props.BlockLayout,
		},
		Board: []gosx.Node{
			props.SiteMapEngine,
			props.SiteMapCanvas,
		},
		RightRail: []gosx.Node{
			props.InspectorChrome,
			props.HomeInspector,
			props.LookPanel,
			props.BrandPanel,
			props.NavigationPanel,
			props.CheckoutPanel,
			props.PublishPanel,
			props.AdvancedPanel,
		},
	})
}

func RenderBackendEditorWorkbenchPanelStack(props BackendEditorWorkbenchPanelStackProps) gosx.Node {
	homeLayersPanel := RenderHomeLayersPanel(props.HomeLayers, HomeLayersPanelOptions{
		PickerNode: props.HomeLayerSelection,
	})
	blockLayout := RenderBlockLayoutEngine(props.HomeLayers, BlockLayoutEngineOptions{
		EngineHostNode: props.BlockLayoutEngineHost,
		LayersNode:     homeLayersPanel,
		LibraryNode:    props.BlockLibraryPanel,
		Kicker:         "Home",
		Title:          "Sections",
	})
	brandPanel := RenderBrandPanel(props.BrandPanel, props.BrandFields, BrandPanelOptions{
		MediaPickerNode: props.BrandMediaPicker,
	})
	advancedPanel := RenderAdvancedPanel(props.AdvancedPanel, AdvancedPanelOptions{
		GroupNodes: map[string]gosx.Node{
			"flows":      props.FlowDesigner,
			"tools":      props.AdvancedToolsPanel,
			"schema":     props.AdvancedWorkspaceFieldPanel,
			"schedule":   props.AdvancedCalendarFieldPanel,
			"typography": props.AdvancedTypeFieldPanel,
		},
	})

	return RenderBackendEditorWorkbenchContent(BackendEditorWorkbenchContentProps{
		View:            props.View,
		AuthoringURL:    props.AuthoringURL,
		Toolbar:         props.Toolbar,
		CanvasBar:       props.CanvasBar,
		CanvasStatus:    props.CanvasStatus,
		SiteNavigator:   props.SiteNavigator,
		BlockLayout:     blockLayout,
		SiteMapEngine:   props.SiteMapEngine,
		SiteMapCanvas:   props.SiteMapCanvas,
		InspectorChrome: props.InspectorChrome,
		HomeInspector:   props.HomeInspector,
		LookPanel:       props.LookPanel,
		BrandPanel:      brandPanel,
		NavigationPanel: props.NavigationPanel,
		CheckoutPanel:   props.CheckoutPanel,
		PublishPanel:    props.PublishPanel,
		AdvancedPanel:   advancedPanel,
	})
}

func RenderBackendEditorLeftRail(nodes ...gosx.Node) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-left-rail-slot", "true")), gosx.Fragment(nodes...))
}

func RenderBackendEditorBoard(nodes ...gosx.Node) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-board-slot", "true")), gosx.Fragment(nodes...))
}

func RenderBackendEditorRightRail(nodes ...gosx.Node) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-right-rail-slot", "true")), gosx.Fragment(nodes...))
}
