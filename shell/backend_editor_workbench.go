package shell

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/canvas"
	"m31labs.dev/gosx-studio/core"
	"m31labs.dev/gosx-studio/panels"
	"m31labs.dev/gosx-studio/sitemap"
)

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
	SiteMapView                     map[string]any
	SiteMapEngineOptions            sitemap.SiteMapEngineOptions
	SiteMapCanvas                   gosx.Node
	SiteMapCanvasSurfaceOptions     canvas.SiteMapCanvasSurfaceOptions
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
	if core.WorkbenchNodeEmpty(toolbar) {
		toolbar = RenderBackendEditorWorkbenchToolbar(props.View)
	}
	canvasBar := props.CanvasBar
	if core.WorkbenchNodeEmpty(canvasBar) {
		canvasBar = RenderBackendEditorWorkbenchCanvasBar(props.View)
	}
	canvasStatus := props.CanvasStatus
	if core.WorkbenchNodeEmpty(canvasStatus) {
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
	if !core.WorkbenchNodeEmpty(canvasBar) {
		options.CanvasBar = []gosx.Node{canvasBar}
	}
	if !core.WorkbenchNodeEmpty(canvasStatus) {
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
	if core.WorkbenchNodeEmpty(siteNavigator) {
		siteNavigator = sitemap.RenderSiteNavigatorPanel(sitemap.SiteNavigatorPropsFromMap(props.SiteNavigatorView), sitemap.SiteNavigatorPanelOptions{})
	}
	homeLayerSelection := props.HomeLayerSelection
	if core.WorkbenchNodeEmpty(homeLayerSelection) {
		homeLayerSelectionView := props.HomeLayerSelectionView
		if len(homeLayerSelectionView) == 0 {
			homeLayerSelectionView = props.HomeLayers
		}
		homeLayerSelection = panels.RenderHomeLayerSelection(panels.HomeLayerSelectionPropsFromMap(homeLayerSelectionView))
	}
	homeLayersPanel := panels.RenderHomeLayersPanel(props.HomeLayers, panels.HomeLayersPanelOptions{
		PickerNode: homeLayerSelection,
	})
	blockLibraryPanel := props.BlockLibraryPanel
	if core.WorkbenchNodeEmpty(blockLibraryPanel) {
		blockLibraryPanel = panels.RenderBlockLibraryPanel(props.BlockLibraryPanelView, panels.BlockLibraryPanelOptions{})
	}
	blockLayout := canvas.RenderBlockLayoutEngine(props.HomeLayers, canvas.BlockLayoutEngineOptions{
		EngineHostNode: props.BlockLayoutEngineHost,
		LayersNode:     homeLayersPanel,
		LibraryNode:    blockLibraryPanel,
		Kicker:         "Home",
		Title:          "Sections",
	})
	inspectorChrome := props.InspectorChrome
	if core.WorkbenchNodeEmpty(inspectorChrome) {
		inspectorChrome = panels.RenderInspectorChromePanel(props.InspectorChromeView, panels.InspectorChromePanelOptions{})
	}
	homeInspector := props.HomeInspector
	if core.WorkbenchNodeEmpty(homeInspector) {
		homeInspector = panels.RenderHomeInspectorPanel(props.HomeInspectorView, props.HomeInspectorContentFields, panels.HomeInspectorPanelOptions{})
	}
	lookPanel := props.LookPanel
	if core.WorkbenchNodeEmpty(lookPanel) {
		lookPanel = panels.RenderLookPanel(props.LookPanelView, panels.LookPanelOptions{})
	}
	brandMediaPicker := props.BrandMediaPicker
	if core.WorkbenchNodeEmpty(brandMediaPicker) {
		brandMediaPicker = panels.RenderBrandMediaPicker(props.BrandMediaPickerView, panels.BrandMediaPickerOptions{})
	}
	brandPanel := panels.RenderBrandPanel(props.BrandPanel, props.BrandFields, panels.BrandPanelOptions{
		MediaPickerNode: brandMediaPicker,
	})
	navigationPanel := props.NavigationPanel
	if core.WorkbenchNodeEmpty(navigationPanel) {
		navigationPanel = panels.RenderNavigationPanel(props.NavigationPanelView, panels.NavigationPanelOptions{})
	}
	checkoutPanel := props.CheckoutPanel
	if core.WorkbenchNodeEmpty(checkoutPanel) {
		checkoutPanel = panels.RenderCheckoutPanel(props.CheckoutPanelView, panels.CheckoutPanelOptions{})
	}
	flowDesigner := props.FlowDesigner
	if core.WorkbenchNodeEmpty(flowDesigner) {
		flowDesigner = panels.RenderFlowDesignerPanel(props.FlowDesignerView, panels.FlowDesignerPanelOptions{})
	}
	advancedToolsPanel := props.AdvancedToolsPanel
	if core.WorkbenchNodeEmpty(advancedToolsPanel) {
		advancedToolsPanel = panels.RenderAdvancedToolsPanel(props.AdvancedToolsPanelView, panels.AdvancedToolsPanelOptions{})
	}
	advancedWorkspaceFieldPanel := props.AdvancedWorkspaceFieldPanel
	if core.WorkbenchNodeEmpty(advancedWorkspaceFieldPanel) {
		advancedWorkspaceFieldPanel = panels.RenderAdvancedFieldPanel(props.AdvancedWorkspaceFieldPanelView, panels.AdvancedFieldPanelOptions{})
	}
	advancedCalendarFieldPanel := props.AdvancedCalendarFieldPanel
	if core.WorkbenchNodeEmpty(advancedCalendarFieldPanel) {
		advancedCalendarFieldPanel = panels.RenderAdvancedFieldPanel(props.AdvancedCalendarFieldPanelView, panels.AdvancedFieldPanelOptions{})
	}
	advancedTypeFieldPanel := props.AdvancedTypeFieldPanel
	if core.WorkbenchNodeEmpty(advancedTypeFieldPanel) {
		advancedTypeFieldPanel = panels.RenderAdvancedFieldPanel(props.AdvancedTypeFieldPanelView, panels.AdvancedFieldPanelOptions{})
	}
	advancedSettingsPanel := props.AdvancedSettingsPanel
	if core.WorkbenchNodeEmpty(advancedSettingsPanel) {
		advancedSettingsPanel = panels.RenderAdvancedSettingsPanel(props.AdvancedSettingsPanelView, panels.AdvancedSettingsPanelOptions{})
	}
	advancedPanel := panels.RenderAdvancedPanel(props.AdvancedPanel, panels.AdvancedPanelOptions{
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
	if core.WorkbenchNodeEmpty(publishPanel) {
		publishPanel = RenderBackendEditorPublishPanelStack(BackendEditorPublishPanelStackProps{
			PublishPanel:    props.PublishPanelView,
			PreviewShare:    props.PreviewSharePanel,
			ActivityPanel:   props.ActivityPanel,
			RevisionHistory: props.RevisionHistory,
		})
	}
	siteMapEngine := props.SiteMapEngine
	if core.WorkbenchNodeEmpty(siteMapEngine) && len(props.SiteMapView) > 0 {
		siteMapEngine = sitemap.RenderSiteMapEngine(props.SiteMapView, props.SiteMapEngineOptions).Surface
	}
	siteMapCanvas := props.SiteMapCanvas
	if core.WorkbenchNodeEmpty(siteMapCanvas) && len(props.SiteMapView) > 0 {
		siteMapCanvas = canvas.RenderSiteMapCanvasSurface(props.SiteMapView, props.SiteMapCanvasSurfaceOptions)
	}

	return RenderBackendEditorWorkbenchContent(BackendEditorWorkbenchContentProps{
		View:            props.View,
		AuthoringURL:    props.AuthoringURL,
		Toolbar:         props.Toolbar,
		CanvasBar:       props.CanvasBar,
		CanvasStatus:    props.CanvasStatus,
		SiteNavigator:   siteNavigator,
		BlockLayout:     blockLayout,
		SiteMapEngine:   siteMapEngine,
		SiteMapCanvas:   siteMapCanvas,
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
	return panels.RenderPublishPanel(props.PublishPanel, panels.PublishPanelOptions{
		PreviewShareNode:    panels.RenderPreviewSharePanel(props.PreviewShare, panels.PreviewSharePanelOptions{}),
		ActivityPanelNode:   panels.RenderActivityPanel(props.ActivityPanel, panels.ActivityPanelOptions{}),
		RevisionHistoryNode: panels.RenderRevisionHistoryPanel(props.RevisionHistory, panels.RevisionHistoryPanelOptions{}),
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
