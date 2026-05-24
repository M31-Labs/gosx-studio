package studio

import "testing"

func TestDefaultBoundaryNamesStudioAsAuthoringLayer(t *testing.T) {
	boundary := DefaultBoundary()

	if boundary.CMSPackage != PackageCMS {
		t.Fatalf("CMS package = %q, want %q", boundary.CMSPackage, PackageCMS)
	}
	if boundary.AdminPackage != PackageAdmin {
		t.Fatalf("admin package = %q, want %q", boundary.AdminPackage, PackageAdmin)
	}
	if boundary.StudioPackage != PackageStudio {
		t.Fatalf("studio package = %q, want %q", boundary.StudioPackage, PackageStudio)
	}
	if boundary.ProductPackage == "" {
		t.Fatal("product package placeholder should describe the composed product boundary")
	}
}

func TestDefaultFeaturesCoverAuthoringSurfaces(t *testing.T) {
	features := DefaultFeatures()
	if len(features) < 5 {
		t.Fatalf("expected at least 5 default features, got %d", len(features))
	}

	surfaces := map[SurfaceKind]bool{}
	for _, feature := range features {
		if feature.Key == "" || feature.Label == "" || feature.Summary == "" {
			t.Fatalf("feature should include key, label, and summary: %#v", feature)
		}
		surfaces[feature.Surface] = true
	}

	for _, surface := range []SurfaceKind{SurfaceCanvas, SurfaceSiteMap, SurfaceInspector, SurfaceFlow, SurfacePublish} {
		if !surfaces[surface] {
			t.Fatalf("missing default feature for surface %q", surface)
		}
	}
}

func TestDefaultEnginesCoverHeavyStudioInteractions(t *testing.T) {
	engines := DefaultEngines()
	if len(engines) < 3 {
		t.Fatalf("expected default engines, got %#v", engines)
	}

	byKind := map[EngineKind]Engine{}
	for _, engine := range engines {
		if engine.Key == "" || engine.Label == "" || engine.MountID == "" {
			t.Fatalf("engine should include key, label, and mount id: %#v", engine)
		}
		if len(engine.Capabilities) == 0 {
			t.Fatalf("engine should declare capabilities: %#v", engine)
		}
		byKind[engine.Kind] = engine
	}

	for _, kind := range []EngineKind{EngineCanvas, EngineSiteMap, EngineBlockLayout, EngineScene3D} {
		if _, ok := byKind[kind]; !ok {
			t.Fatalf("missing default engine kind %q", kind)
		}
	}
	if !engineHasCapability(byKind[EngineCanvas], CapabilityPanZoom) || !engineHasCapability(byKind[EngineCanvas], CapabilityDragDrop) {
		t.Fatalf("canvas engine should cover pan/zoom and drag/drop: %#v", byKind[EngineCanvas])
	}
	if byKind[EngineScene3D].Surface != SurfaceShowcase3D || !engineHasCapability(byKind[EngineScene3D], CapabilityPopout) {
		t.Fatalf("Scene3D engine should cover showcase pop-out viewing: %#v", byKind[EngineScene3D])
	}
}

func TestDefaultRuntimeContractsExposeEngineAPIs(t *testing.T) {
	contracts := DefaultRuntimeContracts()
	if len(contracts) < 5 {
		t.Fatalf("expected default runtime contracts, got %#v", contracts)
	}

	for _, contract := range contracts {
		contract = contract.Normalize()
		if contract.Key == "" || contract.Label == "" || contract.Global == "" {
			t.Fatalf("runtime contract should include key, label, and global: %#v", contract)
		}
		if contract.MethodCount() == 0 {
			t.Fatalf("runtime contract should expose callable methods: %#v", contract)
		}
		for _, method := range contract.Methods {
			if method.Name == "" || method.Label == "" || method.Summary == "" {
				t.Fatalf("runtime method should include name, label, and summary: %#v", method)
			}
		}
	}

	preview, ok := RuntimeContractByGlobal(contracts, "GoSXStudioPreviewRuntime")
	if !ok {
		t.Fatalf("missing preview runtime contract in %#v", contracts)
	}
	if preview.Surface != SurfaceCanvas || preview.Engine != EngineCanvas {
		t.Fatalf("preview runtime should belong to the canvas engine: %#v", preview)
	}
	for _, method := range []string{"mount", "setBlockVisibility", "applyTextUpdate", "applyTheme", "applyStyleImpact", "applyCSS", "applyFonts", "updateHeaderLogo", "requestInlineEdit", "cycleField"} {
		if !runtimeHasMethod(preview, method) {
			t.Fatalf("preview runtime missing method %q: %#v", method, preview)
		}
	}
	textUpdate, ok := preview.Method("applyTextUpdate")
	if !ok || len(textUpdate.Payload) != 7 {
		t.Fatalf("text update payload = %#v, ok=%v", textUpdate, ok)
	}
	theme, ok := preview.Method("applyTheme")
	if !ok || len(theme.Payload) != 7 {
		t.Fatalf("theme payload = %#v, ok=%v", theme, ok)
	}
	styleImpact, ok := preview.Method("applyStyleImpact")
	if !ok || len(styleImpact.Payload) != 1 {
		t.Fatalf("style impact payload = %#v, ok=%v", styleImpact, ok)
	}
	headerLogo, ok := preview.Method("updateHeaderLogo")
	if !ok || len(headerLogo.Payload) != 5 {
		t.Fatalf("header logo payload = %#v, ok=%v", headerLogo, ok)
	}
	inlineEdit, ok := preview.Method("requestInlineEdit")
	if !ok || len(inlineEdit.Payload) != 2 {
		t.Fatalf("inline edit payload = %#v, ok=%v", inlineEdit, ok)
	}

	workbench, ok := RuntimeContractByGlobal(contracts, "GoSXStudioWorkbenchRuntime")
	if !ok {
		t.Fatalf("missing workbench runtime contract in %#v", contracts)
	}
	if workbench.Surface != SurfaceCanvas || workbench.Engine != EngineCanvas {
		t.Fatalf("workbench runtime should belong to the canvas engine: %#v", workbench)
	}
	for _, method := range []string{
		"bindRailResizers",
		"bindChrome",
		"setMode",
		"syncViewport",
		"activateViewport",
		"currentBreakpoint",
		"setStyleState",
		"syncZoom",
		"activateZoom",
		"toggleRail",
		"toggleFocus",
		"toggleActivity",
		"saveLayout",
		"currentRailWidth",
		"setRailWidth",
	} {
		if !runtimeHasMethod(workbench, method) {
			t.Fatalf("workbench runtime missing method %q: %#v", method, workbench)
		}
	}
	bindChrome, ok := workbench.Method("bindChrome")
	if !ok || len(bindChrome.Payload) != 1 {
		t.Fatalf("bind chrome payload = %#v, ok=%v", bindChrome, ok)
	}
	setMode, ok := workbench.Method("setMode")
	if !ok || len(setMode.Payload) != 3 {
		t.Fatalf("set mode payload = %#v, ok=%v", setMode, ok)
	}
	activateViewport, ok := workbench.Method("activateViewport")
	if !ok || len(activateViewport.Payload) != 2 {
		t.Fatalf("activate viewport payload = %#v, ok=%v", activateViewport, ok)
	}
	currentBreakpoint, ok := workbench.Method("currentBreakpoint")
	if !ok || len(currentBreakpoint.Payload) != 1 {
		t.Fatalf("current breakpoint payload = %#v, ok=%v", currentBreakpoint, ok)
	}
	setStyleState, ok := workbench.Method("setStyleState")
	if !ok || len(setStyleState.Payload) != 2 {
		t.Fatalf("set style state payload = %#v, ok=%v", setStyleState, ok)
	}
	toggleRail, ok := workbench.Method("toggleRail")
	if !ok || len(toggleRail.Payload) != 2 {
		t.Fatalf("toggle rail payload = %#v, ok=%v", toggleRail, ok)
	}
	saveLayout, ok := workbench.Method("saveLayout")
	if !ok || len(saveLayout.Payload) != 1 {
		t.Fatalf("save layout payload = %#v, ok=%v", saveLayout, ok)
	}
	currentRailWidth, ok := workbench.Method("currentRailWidth")
	if !ok || len(currentRailWidth.Payload) != 3 {
		t.Fatalf("current rail width payload = %#v, ok=%v", currentRailWidth, ok)
	}
	setRailWidth, ok := workbench.Method("setRailWidth")
	if !ok || len(setRailWidth.Payload) != 5 {
		t.Fatalf("set rail width payload = %#v, ok=%v", setRailWidth, ok)
	}

	selection, ok := RuntimeContractByGlobal(contracts, "GoSXStudioSelectionRuntime")
	if !ok {
		t.Fatalf("missing selection runtime contract in %#v", contracts)
	}
	if selection.Surface != SurfaceCanvas || selection.Engine != EngineCanvas {
		t.Fatalf("selection runtime should belong to the canvas engine: %#v", selection)
	}
	selectionBind, ok := selection.Method("bind")
	if !ok || len(selectionBind.Payload) != 1 {
		t.Fatalf("selection bind payload = %#v, ok=%v", selectionBind, ok)
	}

	field, ok := RuntimeContractByGlobal(contracts, "GoSXStudioFieldRuntime")
	if !ok {
		t.Fatalf("missing field runtime contract in %#v", contracts)
	}
	if field.Surface != SurfaceInspector || field.Engine != EngineCanvas {
		t.Fatalf("field runtime should belong to the inspector surface and canvas engine: %#v", field)
	}
	for _, method := range []string{"bind", "bindMirroring", "bindClipboard"} {
		if !runtimeHasMethod(field, method) {
			t.Fatalf("field runtime missing method %q: %#v", method, field)
		}
	}
	fieldBind, ok := field.Method("bind")
	if !ok || len(fieldBind.Payload) != 1 {
		t.Fatalf("field bind payload = %#v, ok=%v", fieldBind, ok)
	}

	brand, ok := RuntimeContractByGlobal(contracts, "GoSXStudioBrandRuntime")
	if !ok {
		t.Fatalf("missing brand runtime contract in %#v", contracts)
	}
	if brand.Surface != SurfaceInspector || brand.Engine != EngineCanvas {
		t.Fatalf("brand runtime should belong to the inspector surface and canvas engine: %#v", brand)
	}
	for _, method := range []string{"bindLogo", "updateHeaderLogo"} {
		if !runtimeHasMethod(brand, method) {
			t.Fatalf("brand runtime missing method %q: %#v", method, brand)
		}
	}
	brandLogo, ok := brand.Method("updateHeaderLogo")
	if !ok || len(brandLogo.Payload) != 5 {
		t.Fatalf("brand logo payload = %#v, ok=%v", brandLogo, ok)
	}

	style, ok := RuntimeContractByGlobal(contracts, "GoSXStudioStyleRuntime")
	if !ok {
		t.Fatalf("missing style runtime contract in %#v", contracts)
	}
	if style.Surface != SurfaceInspector || style.Engine != EngineCanvas {
		t.Fatalf("style runtime should belong to the inspector surface and canvas engine: %#v", style)
	}
	for _, method := range []string{"bindTheme", "bindWorkbench", "bindCSS", "bindFonts", "applyTheme", "syncControlButtons", "showImpact", "restoreImpact", "setControlValue", "resetControlValue"} {
		if !runtimeHasMethod(style, method) {
			t.Fatalf("style runtime missing method %q: %#v", method, style)
		}
	}
	bindTheme, ok := style.Method("bindTheme")
	if !ok || len(bindTheme.Payload) != 1 {
		t.Fatalf("bind theme payload = %#v, ok=%v", bindTheme, ok)
	}
	showImpact, ok := style.Method("showImpact")
	if !ok || len(showImpact.Payload) != 3 {
		t.Fatalf("show impact payload = %#v, ok=%v", showImpact, ok)
	}
	setStyle, ok := style.Method("setControlValue")
	if !ok || len(setStyle.Payload) != 2 {
		t.Fatalf("set style payload = %#v, ok=%v", setStyle, ok)
	}

	blockLayout, ok := RuntimeContractByGlobal(contracts, "GoSXStudioBlockLayoutRuntime")
	if !ok {
		t.Fatalf("missing block layout runtime contract in %#v", contracts)
	}
	if blockLayout.Surface != SurfaceCanvas || blockLayout.Engine != EngineBlockLayout {
		t.Fatalf("block layout runtime should belong to the block layout engine: %#v", blockLayout)
	}
	for _, method := range []string{"rows", "rowKey", "rowForKey", "moveRow", "renumber", "selectRow", "commitReorder", "updateBlockLibraryState", "updateVisibilityState"} {
		if !runtimeHasMethod(blockLayout, method) {
			t.Fatalf("block layout runtime missing method %q: %#v", method, blockLayout)
		}
	}
	reorder, ok := blockLayout.Method("commitReorder")
	if !ok || len(reorder.Payload) != 4 {
		t.Fatalf("reorder payload = %#v, ok=%v", reorder, ok)
	}
	visibility, ok := blockLayout.Method("updateVisibilityState")
	if !ok || len(visibility.Payload) != 1 || visibility.Payload[0].Kind != ControlToggle {
		t.Fatalf("visibility payload = %#v, ok=%v", visibility, ok)
	}
}

func TestRuntimeContractNormalizesMethodPayloads(t *testing.T) {
	contract := RuntimeContract{
		Key:     " preview ",
		Label:   " Preview ",
		Global:  " GoSXStudioPreviewRuntime ",
		Surface: SurfaceKind(" canvas "),
		Engine:  EngineKind(" canvas "),
		Methods: []RuntimeMethod{
			{
				Name: " mount ",
				Payload: []RuntimePayloadField{
					{Name: " root ", Label: " Root ", Kind: ControlKind(" source "), Summary: " Mounted root "},
					{Label: "Ignored"},
				},
			},
			{Label: "Ignored"},
		},
	}.Normalize()

	if contract.Key != "preview" || contract.Global != "GoSXStudioPreviewRuntime" {
		t.Fatalf("contract strings = %#v", contract)
	}
	if contract.MethodCount() != 1 {
		t.Fatalf("method count = %d", contract.MethodCount())
	}
	method, ok := contract.Method("mount")
	if !ok {
		t.Fatalf("missing normalized method in %#v", contract.Methods)
	}
	if method.Label != "mount" || len(method.Payload) != 1 {
		t.Fatalf("normalized method = %#v", method)
	}
	if method.Payload[0].Name != "root" || method.Payload[0].Kind != ControlSource || method.Payload[0].Label != "Root" {
		t.Fatalf("normalized payload = %#v", method.Payload[0])
	}
}

func TestCanvasWorkspaceNormalizesNoCodeEditingSurface(t *testing.T) {
	canvas := CanvasWorkspace{
		RouteLabel: " Home ",
		PreviewURL: " / ",
		PreviewShell: CanvasPreviewShell{
			OverlayAttr:         " data-custom-overlay ",
			OutlineTemplateAttr: " data-custom-outline-template ",
			InlineEditClass:     " custom-inline ",
		},
		Viewports: []CanvasViewport{
			{Key: " desktop ", Label: " Desktop ", Width: " 100% "},
			{Key: " tablet ", Label: " Tablet ", Width: " 48rem ", Active: true},
			{Key: " mobile ", Label: " Mobile ", Width: " 24rem ", Active: true},
			{Label: "Ignored"},
		},
		ZoomLevels: []CanvasZoomLevel{
			{Key: " fit ", Label: " Fit "},
			{Key: " 100 ", Label: " 100% ", Scale: 1, Active: true},
			{Key: " 125 ", Label: " 125% ", Scale: 1.25, Active: true},
		},
		Blocks: []CanvasBlock{
			{
				Key:           " hero ",
				Label:         " Hero ",
				Summary:       " Homepage hero ",
				GoSXComponent: " HomeHero ",
				Source:        ComponentSource(" cms "),
				Binding:       " home.hero ",
				Status:        " Editable ",
				Visible:       true,
				Selected:      true,
				Editable:      true,
				Controls: []Control{
					{Key: " headline ", Label: " Headline ", Kind: ControlRichText, Binding: " home.hero.headline "},
					{Key: "", Label: "Ignored"},
				},
			},
			{Key: " products ", Label: " Products ", GoSXComponent: " FeaturedProducts ", Source: ComponentSourcePlugin, Binding: " products.collection "},
			{Label: "Ignored"},
		},
		Actions: []CanvasAction{
			{Kind: CanvasActionInlineText, Enabled: true},
			{Key: "bad", Label: "Custom", Kind: CanvasActionKind("unknown")},
		},
	}.Normalize()

	if canvas.RouteLabel != "Home" || canvas.PreviewURL != "/" {
		t.Fatalf("canvas labels = %#v", canvas)
	}
	if canvas.PreviewShell.OverlayAttr != "data-custom-overlay" || canvas.PreviewShell.OutlineTemplateAttr != "data-custom-outline-template" || canvas.PreviewShell.InlineEditClass != "custom-inline" {
		t.Fatalf("preview shell overrides = %#v", canvas.PreviewShell)
	}
	if canvas.PreviewShell.DockAttr != "data-studio-preview-dock" || canvas.PreviewShell.FieldActionTextAttr != "data-studio-preview-field-action-text" {
		t.Fatalf("preview shell defaults = %#v", canvas.PreviewShell)
	}
	if canvas.BlockCount() != 2 || canvas.VisibleBlockCount() != 1 || canvas.ControlCount() != 1 {
		t.Fatalf("canvas counts = blocks %d visible %d controls %d", canvas.BlockCount(), canvas.VisibleBlockCount(), canvas.ControlCount())
	}
	if canvas.ActiveViewport().Key != "tablet" {
		t.Fatalf("active viewport = %#v", canvas.ActiveViewport())
	}
	if canvas.ActiveZoom().Key != "100" {
		t.Fatalf("active zoom = %#v", canvas.ActiveZoom())
	}
	selected, ok := canvas.SelectedBlock()
	if !ok || selected.Key != "hero" || selected.NormalizedSource() != ComponentSourceCMS {
		t.Fatalf("selected block = %#v, ok=%v", selected, ok)
	}
	if len(canvas.Actions) != 2 || canvas.Actions[0].Key != string(CanvasActionInlineText) || canvas.Actions[1].NormalizedKind() != CanvasActionReveal {
		t.Fatalf("actions = %#v", canvas.Actions)
	}
}

func TestCanvasWorkspaceDefaultsForStudioShell(t *testing.T) {
	canvas := CanvasWorkspace{}.Normalize()

	if canvas.ActiveViewport().Key != "desktop" {
		t.Fatalf("default viewport = %#v", canvas.ActiveViewport())
	}
	if canvas.ActiveZoom().Key != "fit" {
		t.Fatalf("default zoom = %#v", canvas.ActiveZoom())
	}
	if canvas.PreviewShell.OverlayAttr != "data-studio-preview-overlay" || canvas.PreviewShell.InlineEditClass != "is-studio-inline-editing" {
		t.Fatalf("default preview shell = %#v", canvas.PreviewShell)
	}
	actions := canvas.Actions
	if len(actions) != len(DefaultCanvasActions()) {
		t.Fatalf("default actions = %#v", actions)
	}
	if CanvasActionKindLabel(CanvasActionToggleVisibility) != "Hide" {
		t.Fatalf("toggle label = %q", CanvasActionKindLabel(CanvasActionToggleVisibility))
	}
}

func engineHasCapability(engine Engine, capability EngineCapability) bool {
	for _, value := range engine.Capabilities {
		if value == capability {
			return true
		}
	}
	return false
}

func runtimeHasMethod(contract RuntimeContract, name string) bool {
	_, ok := contract.Method(name)
	return ok
}

func TestDefaultResourceAdaptersCoverHostBoundResources(t *testing.T) {
	adapters := DefaultResourceAdapters()
	if len(adapters) < 9 {
		t.Fatalf("expected default resource adapters, got %#v", adapters)
	}

	byKind := map[ResourceKind]ResourceAdapter{}
	for _, adapter := range adapters {
		if adapter.Kind == "" || adapter.Label == "" || adapter.Summary == "" || adapter.Surface == "" {
			t.Fatalf("adapter should include kind, label, summary, and surface: %#v", adapter)
		}
		if adapter.CapabilityCount() == 0 {
			t.Fatalf("adapter should declare capabilities: %#v", adapter)
		}
		if adapter.BindingCount() == 0 {
			t.Fatalf("adapter should expose at least one binding: %#v", adapter)
		}
		byKind[adapter.NormalizedKind()] = adapter
	}

	for _, kind := range []ResourceKind{
		ResourceMedia,
		ResourcePages,
		ResourceProducts,
		ResourceOrders,
		ResourceContacts,
		ResourceSettings,
		ResourceRevisions,
		ResourceLifecycle,
		ResourceFlows,
	} {
		if _, ok := byKind[kind]; !ok {
			t.Fatalf("missing default resource adapter kind %q", kind)
		}
	}

	host := HostConfig{Adapters: adapters}
	lifecycle, ok := host.ResourceAdapter(ResourceLifecycle)
	if !ok || lifecycle.Surface != SurfacePublish {
		t.Fatalf("lifecycle adapter = %#v, ok=%v", lifecycle, ok)
	}

	fallback := ResourceAdapter{Kind: ResourceKind(" unknown "), Label: "Unknown", Summary: "Fallback", Surface: SurfaceInspector}
	if fallback.NormalizedKind() != ResourceMedia {
		t.Fatalf("unknown resource kind should normalize to media, got %q", fallback.NormalizedKind())
	}
	normalized := ResourceAdapter{
		Kind:    ResourceKind(" products "),
		Summary: " Product tools ",
		Surface: SurfaceKind(" site-map "),
		Capabilities: []ResourceCapability{
			ResourceCapability(" read "),
			ResourceCapability(" "),
		},
		Bindings: []ResourceBinding{
			{Key: " collection ", Label: " Collection ", Summary: " Storefront products ", Binding: " products.collection "},
			{Key: "", Label: "Skipped", Binding: "products.skipped"},
		},
	}.Normalize()
	if normalized.Kind != ResourceProducts || normalized.Label != "Products" || normalized.Surface != SurfaceSiteMap {
		t.Fatalf("normalized adapter identity = %#v", normalized)
	}
	if normalized.CapabilityCount() != 1 || normalized.BindingCount() != 1 || normalized.Bindings[0].Binding != "products.collection" {
		t.Fatalf("normalized adapter details = %#v", normalized)
	}
	if ResourceKindLabel(ResourceLifecycle) != "Lifecycle" || ResourceKindLabel(ResourceKind("unknown")) != "Media" {
		t.Fatalf("resource labels did not normalize")
	}
}

func TestSiteMapCountsComposedGoSXComponents(t *testing.T) {
	siteMap := SiteMap{Pages: []Page{
		{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Components: []Component{
				{
					Key:           "hero",
					Label:         "Hero",
					GoSXComponent: "HomeHero",
					Source:        ComponentSourceHost,
					Binding:       "home.section.hero",
					Status:        "Editable",
					Editable:      true,
					Controls: []Control{
						{Key: "headline", Label: "Headline", Kind: ControlText, Binding: "home.hero.headline", Value: "Fresh clay"},
						{Key: "layout", Label: "Layout", Kind: ControlChoice, Binding: "home.hero.layout", Value: "split", Options: []ControlOption{{Value: "split", Label: "Split"}, {Value: "overlay", Label: "Overlay"}}},
					},
				},
				{Key: "products", Label: "Products", GoSXComponent: "FeaturedProducts", Source: ComponentSourceCMS, Binding: "products.collection", Status: "Synced", Editable: true},
			},
		},
		{
			Key:           "product",
			Label:         "Product",
			Route:         "/shop/{slug}",
			Group:         PageGroupCommerce,
			GoSXComponent: "ProductPage",
			Components: []Component{
				{
					Key:           "viewer",
					Label:         "3D viewer",
					GoSXComponent: "Showcase3DViewer",
					Source:        ComponentSourcePlugin,
					Binding:       "showcase3d.model",
					Status:        "Ready",
					Editable:      true,
					Controls: []Control{
						{Key: "model", Label: "Model", Kind: ControlScene3D, Binding: "showcase3d.model"},
					},
				},
			},
		},
	}}

	if siteMap.ComponentCount() != 3 {
		t.Fatalf("component count = %d, want 3", siteMap.ComponentCount())
	}
	if siteMap.Pages[0].ComponentCount() != 2 {
		t.Fatalf("home component count = %d, want 2", siteMap.Pages[0].ComponentCount())
	}
	if siteMap.Pages[1].Components[0].Binding != "showcase3d.model" {
		t.Fatalf("plugin binding = %q", siteMap.Pages[1].Components[0].Binding)
	}
	if siteMap.ControlCount() != 3 {
		t.Fatalf("control count = %d, want 3", siteMap.ControlCount())
	}
	if siteMap.Pages[0].ControlCount() != 2 {
		t.Fatalf("home control count = %d, want 2", siteMap.Pages[0].ControlCount())
	}
	if siteMap.Pages[0].Components[0].SelectionKey("home") != "home.hero" {
		t.Fatalf("selection key = %q", siteMap.Pages[0].Components[0].SelectionKey("home"))
	}
}

func TestSiteMapGroupsPagesForBoardFilters(t *testing.T) {
	siteMap := SiteMap{Pages: []Page{
		{Key: "home", Label: "Home", Route: "/", Group: PageGroupSite, GoSXComponent: "HomePage"},
		{Key: "shop", Label: "Shop", Route: "/shop", Group: PageGroupCommerce, GoSXComponent: "ShopPage"},
		{Key: "blog", Label: "Journal", Route: "/blog", Group: PageGroupContent, GoSXComponent: "JournalPage"},
		{Key: "contact", Label: "Contact", Route: "/contact", Group: PageGroupFlows, GoSXComponent: "ContactPage"},
		{Key: "unknown", Label: "Unknown", Route: "/unknown", Group: PageGroup("private"), GoSXComponent: "UnknownPage"},
	}}

	counts := siteMap.PageGroupCounts()
	if len(counts) != 4 {
		t.Fatalf("group counts = %#v", counts)
	}

	want := map[PageGroup]int{
		PageGroupSite:     2,
		PageGroupCommerce: 1,
		PageGroupContent:  1,
		PageGroupFlows:    1,
	}
	for _, count := range counts {
		if count.Label == "" {
			t.Fatalf("group count should include an editor label: %#v", count)
		}
		if want[count.Group] != count.Count {
			t.Fatalf("group %q count = %d, want %d", count.Group, count.Count, want[count.Group])
		}
		delete(want, count.Group)
	}
	if len(want) != 0 {
		t.Fatalf("missing group counts: %#v", want)
	}
	if PageGroupLabel(PageGroupCommerce) != "Store" {
		t.Fatalf("commerce label = %q", PageGroupLabel(PageGroupCommerce))
	}
	if siteMap.Pages[4].NormalizedGroup() != PageGroupSite {
		t.Fatalf("unknown group should normalize to site, got %q", siteMap.Pages[4].NormalizedGroup())
	}
}

func TestNoCodeControlsExposeEditorFacingBindings(t *testing.T) {
	component := Component{
		Key:           "contact-form",
		Label:         "Contact form",
		Summary:       "Collects customer questions.",
		GoSXComponent: "ContactFormFlow",
		Source:        ComponentSource(" cms "),
		Binding:       "flow.contact",
		Editable:      true,
		Controls: []Control{
			{Key: "title", Label: "Title", Kind: ControlRichText, Binding: "flow.contact.title"},
			{Key: "destination", Label: "Destination", Kind: ControlFlow, Binding: "flow.contact.handler", Advanced: true},
			{Key: "unknown", Label: "Unknown", Kind: ControlKind("custom"), Binding: "host.custom"},
		},
	}

	if component.NormalizedSource() != ComponentSourceCMS {
		t.Fatalf("component source = %q", component.NormalizedSource())
	}
	if ComponentSourceLabel(component.Source) != "CMS" {
		t.Fatalf("component source label = %q", ComponentSourceLabel(component.Source))
	}
	if component.ControlCount() != 3 {
		t.Fatalf("control count = %d", component.ControlCount())
	}
	if component.Controls[0].NormalizedKind() != ControlRichText || ControlKindLabel(component.Controls[0].Kind) != "Rich text" {
		t.Fatalf("rich text control kind = %#v", component.Controls[0])
	}
	if component.Controls[1].NormalizedKind() != ControlFlow || ControlKindLabel(component.Controls[1].Kind) != "Flow" {
		t.Fatalf("flow control kind = %#v", component.Controls[1])
	}
	if component.Controls[2].NormalizedKind() != ControlText {
		t.Fatalf("unknown control kinds should be editor text by default: %#v", component.Controls[2])
	}
}

func TestCompositionLibraryDefinesPageBlueprintsAndPalette(t *testing.T) {
	library := CompositionLibrary{
		PageBlueprints: []PageBlueprint{
			{
				Key:           "landing",
				Label:         "Landing page",
				Summary:       "A focused page with a hero, proof, and a call to action.",
				RoutePattern:  "/new-page",
				Group:         PageGroupContent,
				GoSXComponent: "LandingPage",
				Status:        "Ready",
				Components: []ComponentTemplate{
					{Key: "hero", Label: "Hero", GoSXComponent: "HeroSection", Source: ComponentSourceHost},
					{Key: "cta", Label: "Call to action", GoSXComponent: "CallToAction", Source: ComponentSourceStudio},
				},
			},
		},
		ComponentTemplates: []ComponentTemplate{
			{
				Key:            "showcase-3d",
				Label:          "3D showcase",
				Summary:        "Places an approved generated model on a page.",
				Category:       "Media",
				GoSXComponent:  "Showcase3DViewer",
				Source:         ComponentSourcePlugin,
				DefaultBinding: "showcase3d.model",
				Status:         "Plugin",
				AddLabel:       "Add viewer",
				Controls: []Control{
					{Key: "model", Label: "Model", Kind: ControlScene3D, Binding: "showcase3d.model"},
					{Key: "placement", Label: "Placement", Kind: ControlChoice, Binding: "showcase3d.placement"},
				},
			},
		},
	}
	siteMap := SiteMap{Library: library}

	if library.BlueprintCount() != 1 || library.TemplateCount() != 1 {
		t.Fatalf("library counts = %d blueprints, %d templates", library.BlueprintCount(), library.TemplateCount())
	}
	if siteMap.BlueprintCount() != 1 || siteMap.TemplateCount() != 1 {
		t.Fatalf("site map library counts = %d blueprints, %d templates", siteMap.BlueprintCount(), siteMap.TemplateCount())
	}
	if library.PageBlueprints[0].ComponentCount() != 2 {
		t.Fatalf("blueprint component count = %d", library.PageBlueprints[0].ComponentCount())
	}
	if library.PageBlueprints[0].NormalizedGroup() != PageGroupContent {
		t.Fatalf("blueprint group = %q", library.PageBlueprints[0].NormalizedGroup())
	}
	if library.ComponentTemplates[0].ControlCount() != 2 {
		t.Fatalf("template control count = %d", library.ComponentTemplates[0].ControlCount())
	}
	if library.ComponentTemplates[0].NormalizedSource() != ComponentSourcePlugin {
		t.Fatalf("template source = %q", library.ComponentTemplates[0].NormalizedSource())
	}
}

func TestSiteMapNormalizesPageComposition(t *testing.T) {
	siteMap := SiteMap{Pages: []Page{
		{
			Key:           " home ",
			Label:         " Home ",
			Route:         " / ",
			Group:         PageGroup(" commerce "),
			GoSXComponent: " HomePage ",
			Components: []Component{
				{
					Key:           " hero ",
					Label:         " Hero ",
					Summary:       " Hero copy and image ",
					GoSXComponent: " HomeHero ",
					Source:        ComponentSourceHost,
					Binding:       " home.section.hero ",
					Status:        " Editable ",
					Editable:      true,
					Controls: []Control{
						{Key: " headline ", Label: " Headline ", Kind: ControlText, Binding: " home.hero.headline "},
						{Key: " layout ", Label: " Layout ", Kind: ControlKind(" custom "), Binding: " home.hero.layout ", Options: []ControlOption{{Value: " split ", Label: " Split "}}},
					},
				},
				{Key: "products", Label: "Products", GoSXComponent: "ProductGrid", Source: ComponentSourceCMS, Binding: "products.collection", Status: "Synced", Editable: true},
			},
		},
		{
			Key:           "showcase",
			Label:         "Showcase",
			Route:         "/showcase",
			GoSXComponent: "ShowcasePage",
			Selected:      true,
			Components: []Component{
				{Key: "viewer", Label: "3D viewer", GoSXComponent: "Showcase3DViewer", Source: ComponentSourcePlugin, Binding: "showcase3d.model", Status: "Ready", Editable: true},
			},
		},
	}, Library: CompositionLibrary{
		PageBlueprints: []PageBlueprint{{
			Key:           " landing ",
			Label:         " Landing ",
			Summary:       " Hero and call to action ",
			RoutePattern:  " /new-page ",
			Group:         PageGroup(" content "),
			GoSXComponent: " LandingPage ",
			Components: []ComponentTemplate{{
				Key:           " hero ",
				Label:         " Hero ",
				GoSXComponent: " HomeHero ",
				Source:        ComponentSourceHost,
			}},
		}},
		ComponentTemplates: []ComponentTemplate{{
			Key:            " showcase-3d ",
			Label:          " 3D showcase ",
			Summary:        " Pop-out model viewer ",
			Category:       " Media ",
			GoSXComponent:  " Showcase3DViewer ",
			Source:         ComponentSource(" plugin "),
			DefaultBinding: " showcase3d.model ",
			AddLabel:       " Add viewer ",
			Controls: []Control{
				{Key: " model ", Label: " Model ", Kind: ControlScene3D, Binding: " showcase3d.model "},
			},
		}},
	}}.Normalize()

	if siteMap.ComponentCount() != 3 || siteMap.ControlCount() != 2 || siteMap.BlueprintCount() != 1 || siteMap.TemplateCount() != 1 {
		t.Fatalf("site map counts = components %d controls %d blueprints %d templates %d", siteMap.ComponentCount(), siteMap.ControlCount(), siteMap.BlueprintCount(), siteMap.TemplateCount())
	}
	if siteMap.Pages[0].Key != "home" || siteMap.Pages[0].Route != "/" || siteMap.Pages[0].GroupLabel() != "Store" {
		t.Fatalf("home page was not normalized: %#v", siteMap.Pages[0])
	}
	if siteMap.Pages[0].Components[0].Controls[1].Kind != ControlText || siteMap.Pages[0].Components[0].Controls[1].Options[0].Value != "split" {
		t.Fatalf("nested controls were not normalized: %#v", siteMap.Pages[0].Components[0].Controls)
	}
	if siteMap.Pages[0].Components[0].Controls[1].KindLabel() != "Text" {
		t.Fatalf("control label = %q", siteMap.Pages[0].Components[0].Controls[1].KindLabel())
	}
	selected, ok := siteMap.SelectedPage()
	if !ok || selected.Key != "showcase" {
		t.Fatalf("selected page = %#v, ok=%v", selected, ok)
	}
	if siteMap.Library.PageBlueprints[0].RoutePattern != "/new-page" || siteMap.Library.PageBlueprints[0].GroupLabel() != "Content" {
		t.Fatalf("blueprint was not normalized: %#v", siteMap.Library.PageBlueprints[0])
	}
	if siteMap.Library.ComponentTemplates[0].SourceLabel() != "Plugin" {
		t.Fatalf("template source = %q", siteMap.Library.ComponentTemplates[0].SourceLabel())
	}
}

func TestCompositionIntentDescribesNoCodeDraftOperations(t *testing.T) {
	intent := CompositionIntent{
		Key:                  "add-hero-home",
		Label:                "Add hero to Home",
		Summary:              "Adds a page section to the selected route.",
		Kind:                 CompositionIntentKind(" add-component "),
		TargetPageKey:        "home",
		TargetPageLabel:      "Home",
		TargetRoute:          "/",
		TargetRegion:         "main",
		PageBlueprintKey:     "landing",
		PageBlueprintLabel:   "Landing page",
		ComponentTemplateKey: "hero",
		ComponentLabel:       "Hero",
		GoSXComponent:        "HomeHero",
		Binding:              "home.section.hero",
		Status:               "Draft",
		Steps: []CompositionStep{
			{Key: "target", Label: "Home", Summary: "Selected route", GoSXComponent: "HomePage"},
			{Key: "block", Label: "Hero", Summary: "Reusable page section", GoSXComponent: "HomeHero", Binding: "home.section.hero"},
		},
	}

	if intent.NormalizedKind() != CompositionIntentAddComponent {
		t.Fatalf("intent kind = %q", intent.NormalizedKind())
	}
	if intent.StepCount() != 2 {
		t.Fatalf("step count = %d", intent.StepCount())
	}
	if intent.Steps[1].GoSXComponent != "HomeHero" || intent.Steps[1].Binding != "home.section.hero" {
		t.Fatalf("intent step should expose GoSX component and binding: %#v", intent.Steps[1])
	}
	normalized := CompositionIntent{
		Key:           " add-hero ",
		Label:         " Add hero ",
		Kind:          CompositionIntentKind(" add-component "),
		TargetRoute:   " / ",
		GoSXComponent: " HomeHero ",
		Binding:       " home.section.hero ",
		Steps: []CompositionStep{
			{Key: " target ", Label: " Home ", Summary: " Selected route ", GoSXComponent: " HomePage "},
			{Key: " block ", Label: " Hero ", Summary: " Section ", GoSXComponent: " HomeHero ", Binding: " home.section.hero "},
			{Key: "", Label: "Skipped"},
		},
	}.Normalize()
	if normalized.Key != "add-hero" || normalized.TargetRoute != "/" || normalized.Binding != "home.section.hero" || normalized.StepCount() != 2 {
		t.Fatalf("normalized intent = %#v", normalized)
	}

	createPage := CompositionIntent{Kind: CompositionIntentCreatePage}
	if createPage.NormalizedKind() != CompositionIntentCreatePage {
		t.Fatalf("create page kind = %q", createPage.NormalizedKind())
	}
	unknown := CompositionIntent{Kind: CompositionIntentKind("custom")}
	if unknown.NormalizedKind() != CompositionIntentAddComponent {
		t.Fatalf("unknown intent kind should default to add-component, got %q", unknown.NormalizedKind())
	}
}

func TestCompositionWorkspaceBuildsEditableGraphFromSiteMap(t *testing.T) {
	siteMap := SiteMap{Pages: []Page{
		{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Status:        "Editable",
			Selected:      true,
			Components: []Component{
				{
					Key:           "hero",
					Label:         "Hero",
					Summary:       "Lead section",
					GoSXComponent: "HomeHero",
					Source:        ComponentSourceHost,
					Binding:       "home.section.hero",
					Status:        "Editable",
				},
				{
					Key:           "contact",
					Label:         "Contact form",
					Summary:       "Collect messages",
					GoSXComponent: "ContactFormFlow",
					Source:        ComponentSourceCMS,
					Binding:       "flow.contact",
					Status:        "Connected",
				},
			},
		},
		{
			Key:           "product",
			Label:         "Product",
			Route:         "/shop/{slug}",
			Group:         PageGroupCommerce,
			GoSXComponent: "ProductPage",
			Status:        "Store",
			Components: []Component{
				{
					Key:           "viewer",
					Label:         "3D viewer",
					GoSXComponent: "Showcase3DViewer",
					Source:        ComponentSourcePlugin,
					Binding:       "showcase3d.model",
					Status:        "Plugin",
				},
			},
		},
	}}

	workspace := siteMap.CompositionWorkspace()

	if workspace.LayerCount() != 3 {
		t.Fatalf("layer count = %d, want page layers plus resources", workspace.LayerCount())
	}
	if workspace.NodeCount() != 8 {
		t.Fatalf("node count = %d, want 2 pages + 3 components + 3 resources", workspace.NodeCount())
	}
	if workspace.LinkCount() != 6 {
		t.Fatalf("link count = %d, want contains and binding links", workspace.LinkCount())
	}
	canvas := workspace.CanvasLayout()
	if canvas.ViewBox != "0 0 100 100" || len(canvas.Nodes) != workspace.NodeCount() || len(canvas.Links) != workspace.LinkCount() {
		t.Fatalf("canvas layout = %#v", canvas)
	}

	byKey := map[string]WorkspaceNode{}
	for _, node := range workspace.Nodes {
		byKey[node.Key] = node
	}
	home, ok := byKey["page:home"]
	if !ok {
		t.Fatalf("missing home page node in %#v", workspace.Nodes)
	}
	if home.Kind != WorkspaceNodePage || home.GoSXComponent != "HomePage" || !home.Selected {
		t.Fatalf("home node = %#v", home)
	}
	hero, ok := byKey["component:home:hero"]
	if !ok {
		t.Fatalf("missing hero component node in %#v", workspace.Nodes)
	}
	if hero.Kind != WorkspaceNodeComponent || hero.PageKey != "home" || hero.Binding != "home.section.hero" {
		t.Fatalf("hero node = %#v", hero)
	}
	resource, ok := byKey["resource:flow-contact"]
	if !ok {
		t.Fatalf("missing flow resource node in %#v", workspace.Nodes)
	}
	if resource.Kind != WorkspaceNodeResource || resource.Source != ComponentSourceCMS {
		t.Fatalf("resource node = %#v", resource)
	}

	linkKinds := map[WorkspaceLinkKind]int{}
	for _, link := range workspace.Links {
		linkKinds[link.Kind]++
		if link.FromNodeKey == "" || link.ToNodeKey == "" {
			t.Fatalf("link should connect nodes: %#v", link)
		}
	}
	if linkKinds[WorkspaceLinkContains] != 3 || linkKinds[WorkspaceLinkBinds] != 3 {
		t.Fatalf("link kinds = %#v", linkKinds)
	}
	points := map[string]WorkspaceNodePoint{}
	for _, point := range canvas.Nodes {
		points[point.NodeKey] = point
	}
	if points["page:home"].X != 8 || points["page:home"].Y != 12 {
		t.Fatalf("home point = %#v", points["page:home"])
	}
	if points["component:home:hero"].X != 8 || points["component:home:hero"].Y != 50 {
		t.Fatalf("hero point = %#v", points["component:home:hero"])
	}
	paths := map[string]WorkspaceLinkPath{}
	for _, path := range canvas.Links {
		paths[path.Key] = path
	}
	containsHero := paths["contains:home:hero"]
	if containsHero.Path != "M 8 12 C 16 12, 16 50, 8 50" || containsHero.Kind != WorkspaceLinkContains {
		t.Fatalf("contains path = %#v", containsHero)
	}
	bindsContact := paths["binds:home:contact:flow-contact"]
	if bindsContact.Path == "" || bindsContact.FromNodeKey != "component:home:contact" || bindsContact.ToNodeKey != "resource:flow-contact" {
		t.Fatalf("binding path = %#v", bindsContact)
	}
	if WorkspaceNodeKindLabel(WorkspaceNodeResource) != "Resource" {
		t.Fatalf("resource node label mismatch")
	}
	if WorkspaceLinkKindLabel(WorkspaceLinkBinds) != "Binding" {
		t.Fatalf("binding link label mismatch")
	}
}

func TestFlowReadinessChecksExposeOperatorFriendlyState(t *testing.T) {
	flow := Flow{
		Key:            "contact",
		Label:          "Contact",
		Route:          "/contact",
		HasRoute:       true,
		EmbedTarget:    "contact",
		HasEmbedTarget: true,
		HandlerRef:     "contact.submit",
		CanExecute:     true,
		Steps: []FlowStep{
			{Key: "message", Label: "Message", BlockCount: 1, HasBlocks: true},
		},
		Actions: []FlowAction{
			{
				Key:        "submit",
				Label:      "Submit message",
				HandlerRef: "contact.submit",
				CanExecute: true,
				Fields: []FlowField{
					{Name: "email", Label: "Email", Kind: ControlText, Required: true},
					{Name: "message", Label: "Message", Kind: ControlRichText, Required: true},
				},
			},
		},
	}

	if flow.ReadinessStatus() != ReadinessReady {
		t.Fatalf("ready flow status = %q", flow.ReadinessStatus())
	}
	if flow.ReadinessLabel() != "Ready to publish" {
		t.Fatalf("ready flow label = %q", flow.ReadinessLabel())
	}
	checks := flow.ReadinessChecks()
	if len(checks) != 4 {
		t.Fatalf("readiness checks = %#v", checks)
	}
	if checks[0].Status != ReadinessReady || checks[0].Summary != "contact.submit receives submissions." {
		t.Fatalf("handler check = %#v", checks[0])
	}
	nodes := flow.Nodes()
	if len(nodes) != 4 {
		t.Fatalf("flow nodes = %#v", nodes)
	}
	if nodes[0].Kind != "placement" || nodes[1].Kind != "step" || nodes[2].Kind != "action" || nodes[3].Kind != "publish" {
		t.Fatalf("flow node order = %#v", nodes)
	}
}

func TestFlowReadinessBlocksMissingHandlerButAllowsPlacementReview(t *testing.T) {
	flow := Flow{
		Key:      "newsletter",
		Label:    "Newsletter",
		Route:    "/newsletter",
		HasRoute: true,
		Steps:    []FlowStep{{Key: "signup", Label: "Signup"}},
		Actions:  []FlowAction{{Key: "submit", Label: "Subscribe"}},
	}

	if flow.ReadinessStatus() != ReadinessBlocked {
		t.Fatalf("missing handler should block, got %q", flow.ReadinessStatus())
	}
	checks := flow.ReadinessChecks()
	if checks[0].Status != ReadinessBlocked {
		t.Fatalf("handler check = %#v", checks[0])
	}
	if checks[1].Status != ReadinessWatch {
		t.Fatalf("route-only placement should need review, got %#v", checks[1])
	}
	if ReadinessStatusLabel(ReadinessWatch) != "Review" {
		t.Fatalf("watch label mismatch")
	}
}
