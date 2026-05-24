package studio

import "strings"

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

type PageGroup string

const (
	PageGroupSite     PageGroup = "site"
	PageGroupCommerce PageGroup = "commerce"
	PageGroupContent  PageGroup = "content"
	PageGroupFlows    PageGroup = "flows"
	PageGroupUtility  PageGroup = "utility"
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

type ControlKind string

const (
	ControlText     ControlKind = "text"
	ControlRichText ControlKind = "rich-text"
	ControlMedia    ControlKind = "media"
	ControlChoice   ControlKind = "choice"
	ControlToggle   ControlKind = "toggle"
	ControlNumber   ControlKind = "number"
	ControlLink     ControlKind = "link"
	ControlColor    ControlKind = "color"
	ControlSource   ControlKind = "source"
	ControlFlow     ControlKind = "flow"
	ControlScene3D  ControlKind = "scene-3d"
)

type SiteMap struct {
	Pages   []Page
	Library CompositionLibrary
}

type Page struct {
	Key           string
	Label         string
	Route         string
	Group         PageGroup
	GoSXComponent string
	Status        string
	Components    []Component
}

type Component struct {
	Key           string
	Label         string
	Summary       string
	GoSXComponent string
	Source        ComponentSource
	Binding       string
	Status        string
	Editable      bool
	Controls      []Control
}

type Control struct {
	Key      string
	Label    string
	Kind     ControlKind
	Binding  string
	Value    string
	Help     string
	Required bool
	Advanced bool
	Options  []ControlOption
}

type ControlOption struct {
	Value string
	Label string
}

type CompositionLibrary struct {
	PageBlueprints     []PageBlueprint
	ComponentTemplates []ComponentTemplate
}

type PageBlueprint struct {
	Key           string
	Label         string
	Summary       string
	RoutePattern  string
	Group         PageGroup
	GoSXComponent string
	Status        string
	Components    []ComponentTemplate
}

type ComponentTemplate struct {
	Key            string
	Label          string
	Summary        string
	Category       string
	GoSXComponent  string
	Source         ComponentSource
	DefaultBinding string
	Status         string
	AddLabel       string
	Controls       []Control
}

type PageGroupCount struct {
	Group PageGroup
	Label string
	Count int
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

func (siteMap SiteMap) ControlCount() int {
	count := 0
	for _, page := range siteMap.Pages {
		count += page.ControlCount()
	}
	return count
}

func (siteMap SiteMap) PageGroupCounts() []PageGroupCount {
	counts := map[PageGroup]int{}
	for _, page := range siteMap.Pages {
		counts[page.NormalizedGroup()]++
	}

	groups := []PageGroup{
		PageGroupSite,
		PageGroupCommerce,
		PageGroupContent,
		PageGroupFlows,
		PageGroupUtility,
	}
	out := make([]PageGroupCount, 0, len(groups))
	for _, group := range groups {
		count := counts[group]
		if count == 0 {
			continue
		}
		out = append(out, PageGroupCount{
			Group: group,
			Label: PageGroupLabel(group),
			Count: count,
		})
	}
	return out
}

func (siteMap SiteMap) BlueprintCount() int {
	return len(siteMap.Library.PageBlueprints)
}

func (siteMap SiteMap) TemplateCount() int {
	return len(siteMap.Library.ComponentTemplates)
}

func (page Page) ComponentCount() int {
	return len(page.Components)
}

func (page Page) ControlCount() int {
	count := 0
	for _, component := range page.Components {
		count += component.ControlCount()
	}
	return count
}

func (page Page) NormalizedGroup() PageGroup {
	return normalizePageGroup(page.Group)
}

func (component Component) ControlCount() int {
	return len(component.Controls)
}

func (component Component) SelectionKey(pageKey string) string {
	pageKey = strings.TrimSpace(pageKey)
	componentKey := strings.TrimSpace(component.Key)
	if pageKey == "" {
		return componentKey
	}
	if componentKey == "" {
		return pageKey
	}
	return pageKey + "." + componentKey
}

func (component Component) NormalizedSource() ComponentSource {
	return normalizeComponentSource(component.Source)
}

func (library CompositionLibrary) BlueprintCount() int {
	return len(library.PageBlueprints)
}

func (library CompositionLibrary) TemplateCount() int {
	return len(library.ComponentTemplates)
}

func (blueprint PageBlueprint) ComponentCount() int {
	return len(blueprint.Components)
}

func (blueprint PageBlueprint) NormalizedGroup() PageGroup {
	return normalizePageGroup(blueprint.Group)
}

func (template ComponentTemplate) ControlCount() int {
	return len(template.Controls)
}

func (template ComponentTemplate) NormalizedSource() ComponentSource {
	return normalizeComponentSource(template.Source)
}

func (control Control) NormalizedKind() ControlKind {
	return normalizeControlKind(control.Kind)
}

func PageGroupLabel(group PageGroup) string {
	switch normalizePageGroup(group) {
	case PageGroupCommerce:
		return "Store"
	case PageGroupContent:
		return "Content"
	case PageGroupFlows:
		return "Flows"
	case PageGroupUtility:
		return "Utility"
	default:
		return "Site"
	}
}

func ComponentSourceLabel(source ComponentSource) string {
	switch normalizeComponentSource(source) {
	case ComponentSourceCMS:
		return "CMS"
	case ComponentSourcePlugin:
		return "Plugin"
	case ComponentSourceStudio:
		return "Studio"
	default:
		return "Site"
	}
}

func ControlKindLabel(kind ControlKind) string {
	switch normalizeControlKind(kind) {
	case ControlRichText:
		return "Rich text"
	case ControlMedia:
		return "Media"
	case ControlChoice:
		return "Choice"
	case ControlToggle:
		return "Toggle"
	case ControlNumber:
		return "Number"
	case ControlLink:
		return "Link"
	case ControlColor:
		return "Color"
	case ControlSource:
		return "Source"
	case ControlFlow:
		return "Flow"
	case ControlScene3D:
		return "Scene 3D"
	default:
		return "Text"
	}
}

func normalizePageGroup(group PageGroup) PageGroup {
	normalized := PageGroup(strings.TrimSpace(string(group)))
	switch normalized {
	case PageGroupCommerce, PageGroupContent, PageGroupFlows, PageGroupUtility:
		return normalized
	default:
		return PageGroupSite
	}
}

func normalizeComponentSource(source ComponentSource) ComponentSource {
	switch ComponentSource(strings.TrimSpace(string(source))) {
	case ComponentSourceCMS:
		return ComponentSourceCMS
	case ComponentSourcePlugin:
		return ComponentSourcePlugin
	case ComponentSourceStudio:
		return ComponentSourceStudio
	default:
		return ComponentSourceHost
	}
}

func normalizeControlKind(kind ControlKind) ControlKind {
	switch ControlKind(strings.TrimSpace(string(kind))) {
	case ControlRichText:
		return ControlRichText
	case ControlMedia:
		return ControlMedia
	case ControlChoice:
		return ControlChoice
	case ControlToggle:
		return ControlToggle
	case ControlNumber:
		return ControlNumber
	case ControlLink:
		return ControlLink
	case ControlColor:
		return ControlColor
	case ControlSource:
		return ControlSource
	case ControlFlow:
		return ControlFlow
	case ControlScene3D:
		return ControlScene3D
	default:
		return ControlText
	}
}
