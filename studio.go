package studio

const (
	PackageCMS    = "github.com/odvcencio/gosx-cms"
	PackageAdmin  = "github.com/odvcencio/gosx-admin"
	PackageStudio = "github.com/M31-Labs/gosx-studio"
)

type SurfaceKind string

const (
	SurfaceCanvas     SurfaceKind = "canvas"
	SurfaceSiteMap    SurfaceKind = "site-map"
	SurfaceInspector  SurfaceKind = "inspector"
	SurfaceFlow       SurfaceKind = "flow"
	SurfacePublish    SurfaceKind = "publish"
	SurfaceShowcase3D SurfaceKind = "showcase-3d"
)

type ComponentSource string

const (
	ComponentSourceHost   ComponentSource = "host"
	ComponentSourceCMS    ComponentSource = "cms"
	ComponentSourcePlugin ComponentSource = "plugin"
	ComponentSourceStudio ComponentSource = "studio"
)

type EngineKind string

const (
	EngineCanvas      EngineKind = "canvas"
	EngineSiteMap     EngineKind = "site-map"
	EngineBlockLayout EngineKind = "block-layout"
	EngineFlow        EngineKind = "flow"
	EngineScene3D     EngineKind = "scene-3d"
)

type EngineCapability string

const (
	CapabilityDragDrop    EngineCapability = "drag-drop"
	CapabilityPanZoom     EngineCapability = "pan-zoom"
	CapabilitySelection   EngineCapability = "selection"
	CapabilityInlineEdit  EngineCapability = "inline-edit"
	CapabilityPreview     EngineCapability = "preview"
	CapabilityPopout      EngineCapability = "popout"
	CapabilityPersistence EngineCapability = "persistence"
)

type SiteMap struct {
	Pages []Page
}

type Page struct {
	Key           string
	Label         string
	Route         string
	GoSXComponent string
	Status        string
	Components    []Component
}

type Component struct {
	Key           string
	Label         string
	GoSXComponent string
	Source        ComponentSource
	Binding       string
	Status        string
	Editable      bool
}

type Engine struct {
	Key          string
	Label        string
	Kind         EngineKind
	MountID      string
	Surface      SurfaceKind
	Capabilities []EngineCapability
}

type Boundary struct {
	CMSPackage     string
	AdminPackage   string
	StudioPackage  string
	ProductPackage string
}

func DefaultBoundary() Boundary {
	return Boundary{
		CMSPackage:     PackageCMS,
		AdminPackage:   PackageAdmin,
		StudioPackage:  PackageStudio,
		ProductPackage: "future product package consuming cms, admin, and studio",
	}
}

type HostConfig struct {
	ProductName string
	EditorLabel string
	Features    []Feature
	Engines     []Engine
	SiteMap     SiteMap
}

type Feature struct {
	Key     string
	Label   string
	Surface SurfaceKind
	Summary string
}

func DefaultFeatures() []Feature {
	return []Feature{
		{
			Key:     "site-canvas",
			Label:   "Site canvas",
			Surface: SurfaceCanvas,
			Summary: "No-code page structure and preview editing surface.",
		},
		{
			Key:     "site-map",
			Label:   "Site map",
			Surface: SurfaceSiteMap,
			Summary: "Editable page graph where routes are composed from GoSX components.",
		},
		{
			Key:     "site-inspector",
			Label:   "Inspector",
			Surface: SurfaceInspector,
			Summary: "Plain-language controls for selection, style, content, and publish readiness.",
		},
		{
			Key:     "site-flows",
			Label:   "Flows",
			Surface: SurfaceFlow,
			Summary: "Visual automation and lifecycle flow authoring.",
		},
		{
			Key:     "publish-lifecycle",
			Label:   "Publish",
			Surface: SurfacePublish,
			Summary: "Review, schedule, and publish controls for non-technical site operators.",
		},
	}
}

func DefaultEngines() []Engine {
	return []Engine{
		{
			Key:     "canvas",
			Label:   "Canvas",
			Kind:    EngineCanvas,
			MountID: "gosx-studio-canvas-engine",
			Surface: SurfaceCanvas,
			Capabilities: []EngineCapability{
				CapabilitySelection,
				CapabilityInlineEdit,
				CapabilityPreview,
			},
		},
		{
			Key:     "site-map",
			Label:   "Site map",
			Kind:    EngineSiteMap,
			MountID: "gosx-studio-site-map-engine",
			Surface: SurfaceSiteMap,
			Capabilities: []EngineCapability{
				CapabilityPanZoom,
				CapabilitySelection,
				CapabilityPersistence,
			},
		},
		{
			Key:     "block-layout",
			Label:   "Block layout",
			Kind:    EngineBlockLayout,
			MountID: "gosx-studio-block-layout-engine",
			Surface: SurfaceCanvas,
			Capabilities: []EngineCapability{
				CapabilityDragDrop,
				CapabilitySelection,
				CapabilityPersistence,
			},
		},
	}
}

func (siteMap SiteMap) ComponentCount() int {
	count := 0
	for _, page := range siteMap.Pages {
		count += len(page.Components)
	}
	return count
}

func (page Page) ComponentCount() int {
	return len(page.Components)
}
