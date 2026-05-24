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
