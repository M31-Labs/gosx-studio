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

	for _, kind := range []EngineKind{EngineCanvas, EngineSiteMap, EngineBlockLayout} {
		if _, ok := byKind[kind]; !ok {
			t.Fatalf("missing default engine kind %q", kind)
		}
	}
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
