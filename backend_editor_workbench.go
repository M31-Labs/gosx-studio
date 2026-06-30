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

	SiteNavigator                   gosx.Node
	SiteNavigatorView               map[string]any
	HomeLayers                      map[string]any
	HomeLayerSelection              gosx.Node
	HomeLayerSelectionView          map[string]any
	BlockLayoutEngineHost           gosx.Node
	BlockLibraryPanel               gosx.Node
	BlockLibraryPanelView           map[string]any
	SiteMapEngine                   gosx.Node
	SiteMapCanvas                   gosx.Node
	InspectorChrome                 gosx.Node
	InspectorChromeView             map[string]any
	HomeInspector                   gosx.Node
	HomeInspectorView               map[string]any
	HomeInspectorContentFields      map[string]any
	LookPanel                       gosx.Node
	LookPanelView                   map[string]any
	BrandPanel                      map[string]any
	BrandFields                     map[string]any
	BrandMediaPicker                gosx.Node
	BrandMediaPickerView            map[string]any
	NavigationPanel                 gosx.Node
	NavigationPanelView             map[string]any
	CheckoutPanel                   gosx.Node
	CheckoutPanelView               map[string]any
	PublishPanel                    gosx.Node
	PublishPanelView                map[string]any
	PreviewSharePanel               map[string]any
	ActivityPanel                   map[string]any
	RevisionHistory                 map[string]any
	AdvancedPanel                   map[string]any
	FlowDesigner                    gosx.Node
	FlowDesignerView                map[string]any
	AdvancedToolsPanel              gosx.Node
	AdvancedToolsPanelView          map[string]any
	AdvancedWorkspaceFieldPanel     gosx.Node
	AdvancedWorkspaceFieldPanelView map[string]any
	AdvancedCalendarFieldPanel      gosx.Node
	AdvancedCalendarFieldPanelView  map[string]any
	AdvancedTypeFieldPanel          gosx.Node
	AdvancedTypeFieldPanelView      map[string]any
	AdvancedSettingsPanel           gosx.Node
	AdvancedSettingsPanelView       map[string]any
}

func RenderBackendEditorWorkbench(props BackendEditorWorkbenchProps) gosx.Node {
	toolbar := props.Toolbar
	if workbenchNodeEmpty(toolbar) {
		toolbar = RenderBackendEditorWorkbenchToolbar(props.View)
	}
	canvasBar := props.CanvasBar
	if workbenchNodeEmpty(canvasBar) {
		canvasBar = RenderBackendEditorWorkbenchCanvasBar(props.View)
	}
	canvasStatus := props.CanvasStatus
	if workbenchNodeEmpty(canvasStatus) {
		canvasStatus = RenderBackendEditorWorkbenchCanvasStatus(props.View)
	}

	options := WorkbenchFrameOptions{
		AuthoringURL:   props.AuthoringURL,
		ResizableRails: true,
		Toolbar:        toolbar,
		LeftRail:       []gosx.Node{RenderBackendEditorLeftRail(props.LeftRail...)},
		Board:          []gosx.Node{RenderBackendEditorBoard(props.Board...)},
		RightRail:      []gosx.Node{RenderBackendEditorRightRail(props.RightRail...)},
	}
	if !workbenchNodeEmpty(canvasBar) {
		options.CanvasBar = []gosx.Node{canvasBar}
	}
	if !workbenchNodeEmpty(canvasStatus) {
		options.CanvasStatus = []gosx.Node{canvasStatus}
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-frame-slot", "true")),
		RenderWorkbenchFrame(props.View, options),
	)
}

func RenderBackendEditorWorkbenchToolbar(view map[string]any) gosx.Node {
	return RenderWorkbenchToolbar(view, WorkbenchToolbarOptions{
		Class:        "editor-toolbar",
		ActionsClass: "button-row",
		Controls: []gosx.Node{
			RenderWorkbenchModebar(view, WorkbenchModebarOptions{Class: "studio-modebar"}),
			RenderWorkbenchMetricStrip(view, WorkbenchMetricStripOptions{Class: "studio-context-strip"}),
		},
		CommandPaletteNode: RenderWorkbenchCommandPalette(view, WorkbenchCommandPaletteOptions{Class: "studio-command-palette"}),
		SaveStatusNode: RenderWorkbenchSaveStatus(WorkbenchSaveStatusOptions{
			Class:           "editor-save-status",
			StateClass:      "editor-save-state",
			DetailClass:     "editor-save-detail",
			LastSavedClass:  "editor-save-time",
			DirtyCountClass: "editor-save-count",
		}),
	})
}

func RenderBackendEditorWorkbenchZoomControls(view map[string]any) gosx.Node {
	return RenderWorkbenchZoomControls(view, WorkbenchZoomControlsOptions{Class: "studio-zoombar"})
}

func RenderBackendEditorWorkbenchCanvasTools(view map[string]any) gosx.Node {
	return RenderWorkbenchCanvasTools(view, WorkbenchCanvasToolsOptions{Class: "studio-canvas-tools"})
}

func RenderBackendEditorWorkbenchViewportControls(view map[string]any) gosx.Node {
	return RenderWorkbenchViewportControls(view, WorkbenchViewportControlsOptions{Class: "studio-viewport-switcher"})
}

func RenderBackendEditorWorkbenchCanvasBar(view map[string]any) gosx.Node {
	return RenderWorkbenchCanvasBar(view, WorkbenchCanvasBarOptions{
		Class: "studio-canvas-bar",
		Controls: []gosx.Node{
			RenderBackendEditorWorkbenchCanvasTools(view),
			RenderBackendEditorWorkbenchViewportControls(view),
			RenderBackendEditorWorkbenchZoomControls(view),
		},
	})
}

func RenderBackendEditorWorkbenchCanvasStatus(view map[string]any) gosx.Node {
	return RenderWorkbenchCanvasStatus(view, WorkbenchCanvasStatusOptions{Class: "studio-canvas-status"})
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
	siteNavigator := props.SiteNavigator
	if workbenchNodeEmpty(siteNavigator) {
		siteNavigator = RenderSiteNavigatorPanel(SiteNavigatorPropsFromMap(props.SiteNavigatorView), SiteNavigatorPanelOptions{})
	}
	homeLayerSelection := props.HomeLayerSelection
	if workbenchNodeEmpty(homeLayerSelection) {
		homeLayerSelectionView := props.HomeLayerSelectionView
		if len(homeLayerSelectionView) == 0 {
			homeLayerSelectionView = props.HomeLayers
		}
		homeLayerSelection = RenderHomeLayerSelection(HomeLayerSelectionPropsFromMap(homeLayerSelectionView))
	}
	homeLayersPanel := RenderHomeLayersPanel(props.HomeLayers, HomeLayersPanelOptions{
		PickerNode: homeLayerSelection,
	})
	blockLibraryPanel := props.BlockLibraryPanel
	if workbenchNodeEmpty(blockLibraryPanel) {
		blockLibraryPanel = RenderBlockLibraryPanel(props.BlockLibraryPanelView, BlockLibraryPanelOptions{})
	}
	blockLayout := RenderBlockLayoutEngine(props.HomeLayers, BlockLayoutEngineOptions{
		EngineHostNode: props.BlockLayoutEngineHost,
		LayersNode:     homeLayersPanel,
		LibraryNode:    blockLibraryPanel,
		Kicker:         "Home",
		Title:          "Sections",
	})
	inspectorChrome := props.InspectorChrome
	if workbenchNodeEmpty(inspectorChrome) {
		inspectorChrome = RenderInspectorChromePanel(props.InspectorChromeView, InspectorChromePanelOptions{})
	}
	homeInspector := props.HomeInspector
	if workbenchNodeEmpty(homeInspector) {
		homeInspector = RenderHomeInspectorPanel(props.HomeInspectorView, props.HomeInspectorContentFields, HomeInspectorPanelOptions{})
	}
	lookPanel := props.LookPanel
	if workbenchNodeEmpty(lookPanel) {
		lookPanel = RenderLookPanel(props.LookPanelView, LookPanelOptions{})
	}
	brandMediaPicker := props.BrandMediaPicker
	if workbenchNodeEmpty(brandMediaPicker) {
		brandMediaPicker = RenderBrandMediaPicker(props.BrandMediaPickerView, BrandMediaPickerOptions{})
	}
	brandPanel := RenderBrandPanel(props.BrandPanel, props.BrandFields, BrandPanelOptions{
		MediaPickerNode: brandMediaPicker,
	})
	navigationPanel := props.NavigationPanel
	if workbenchNodeEmpty(navigationPanel) {
		navigationPanel = RenderNavigationPanel(props.NavigationPanelView, NavigationPanelOptions{})
	}
	checkoutPanel := props.CheckoutPanel
	if workbenchNodeEmpty(checkoutPanel) {
		checkoutPanel = RenderCheckoutPanel(props.CheckoutPanelView, CheckoutPanelOptions{})
	}
	flowDesigner := props.FlowDesigner
	if workbenchNodeEmpty(flowDesigner) {
		flowDesigner = RenderFlowDesignerPanel(props.FlowDesignerView, FlowDesignerPanelOptions{})
	}
	advancedToolsPanel := props.AdvancedToolsPanel
	if workbenchNodeEmpty(advancedToolsPanel) {
		advancedToolsPanel = RenderAdvancedToolsPanel(props.AdvancedToolsPanelView, AdvancedToolsPanelOptions{})
	}
	advancedWorkspaceFieldPanel := props.AdvancedWorkspaceFieldPanel
	if workbenchNodeEmpty(advancedWorkspaceFieldPanel) {
		advancedWorkspaceFieldPanel = RenderAdvancedFieldPanel(props.AdvancedWorkspaceFieldPanelView, AdvancedFieldPanelOptions{})
	}
	advancedCalendarFieldPanel := props.AdvancedCalendarFieldPanel
	if workbenchNodeEmpty(advancedCalendarFieldPanel) {
		advancedCalendarFieldPanel = RenderAdvancedFieldPanel(props.AdvancedCalendarFieldPanelView, AdvancedFieldPanelOptions{})
	}
	advancedTypeFieldPanel := props.AdvancedTypeFieldPanel
	if workbenchNodeEmpty(advancedTypeFieldPanel) {
		advancedTypeFieldPanel = RenderAdvancedFieldPanel(props.AdvancedTypeFieldPanelView, AdvancedFieldPanelOptions{})
	}
	advancedSettingsPanel := props.AdvancedSettingsPanel
	if workbenchNodeEmpty(advancedSettingsPanel) {
		advancedSettingsPanel = RenderAdvancedSettingsPanel(props.AdvancedSettingsPanelView, AdvancedSettingsPanelOptions{})
	}
	advancedPanel := RenderAdvancedPanel(props.AdvancedPanel, AdvancedPanelOptions{
		GroupNodes: map[string]gosx.Node{
			"flows":      flowDesigner,
			"tools":      advancedToolsPanel,
			"schema":     advancedWorkspaceFieldPanel,
			"schedule":   advancedCalendarFieldPanel,
			"typography": advancedTypeFieldPanel,
			"settings":   advancedSettingsPanel,
		},
	})
	publishPanel := props.PublishPanel
	if workbenchNodeEmpty(publishPanel) {
		publishPanel = RenderBackendEditorPublishPanelStack(BackendEditorPublishPanelStackProps{
			PublishPanel:    props.PublishPanelView,
			PreviewShare:    props.PreviewSharePanel,
			ActivityPanel:   props.ActivityPanel,
			RevisionHistory: props.RevisionHistory,
		})
	}

	return RenderBackendEditorWorkbenchContent(BackendEditorWorkbenchContentProps{
		View:            props.View,
		AuthoringURL:    props.AuthoringURL,
		Toolbar:         props.Toolbar,
		CanvasBar:       props.CanvasBar,
		CanvasStatus:    props.CanvasStatus,
		SiteNavigator:   siteNavigator,
		BlockLayout:     blockLayout,
		SiteMapEngine:   props.SiteMapEngine,
		SiteMapCanvas:   props.SiteMapCanvas,
		InspectorChrome: inspectorChrome,
		HomeInspector:   homeInspector,
		LookPanel:       lookPanel,
		BrandPanel:      brandPanel,
		NavigationPanel: navigationPanel,
		CheckoutPanel:   checkoutPanel,
		PublishPanel:    publishPanel,
		AdvancedPanel:   advancedPanel,
	})
}

type BackendEditorPublishPanelStackProps struct {
	PublishPanel    map[string]any
	PreviewShare    map[string]any
	ActivityPanel   map[string]any
	RevisionHistory map[string]any
}

func RenderBackendEditorPublishPanelStack(props BackendEditorPublishPanelStackProps) gosx.Node {
	return RenderPublishPanel(props.PublishPanel, PublishPanelOptions{
		PreviewShareNode:    RenderPreviewSharePanel(props.PreviewShare, PreviewSharePanelOptions{}),
		ActivityPanelNode:   RenderActivityPanel(props.ActivityPanel, ActivityPanelOptions{}),
		RevisionHistoryNode: RenderRevisionHistoryPanel(props.RevisionHistory, RevisionHistoryPanelOptions{}),
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
