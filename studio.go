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

type ResourceKind string

const (
	ResourceMedia     ResourceKind = "media"
	ResourcePages     ResourceKind = "pages"
	ResourceProducts  ResourceKind = "products"
	ResourceOrders    ResourceKind = "orders"
	ResourceContacts  ResourceKind = "contacts"
	ResourceSettings  ResourceKind = "settings"
	ResourceRevisions ResourceKind = "revisions"
	ResourceLifecycle ResourceKind = "lifecycle"
	ResourceFlows     ResourceKind = "flows"
)

type ResourceCapability string

const (
	ResourceList    ResourceCapability = "list"
	ResourceRead    ResourceCapability = "read"
	ResourceWrite   ResourceCapability = "write"
	ResourceSearch  ResourceCapability = "search"
	ResourcePreview ResourceCapability = "preview"
	ResourcePublish ResourceCapability = "publish"
	ResourceRestore ResourceCapability = "restore"
	ResourceExecute ResourceCapability = "execute"
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

type CompositionIntentKind string

const (
	CompositionIntentCreatePage   CompositionIntentKind = "create-page"
	CompositionIntentAddComponent CompositionIntentKind = "add-component"
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

type CompositionIntent struct {
	Key                  string
	Label                string
	Summary              string
	Kind                 CompositionIntentKind
	TargetPageKey        string
	TargetPageLabel      string
	TargetRoute          string
	TargetRegion         string
	PageBlueprintKey     string
	PageBlueprintLabel   string
	ComponentTemplateKey string
	ComponentLabel       string
	GoSXComponent        string
	Binding              string
	Status               string
	Steps                []CompositionStep
}

type CompositionStep struct {
	Key           string
	Label         string
	Summary       string
	GoSXComponent string
	Binding       string
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
	Adapters    []ResourceAdapter
	SiteMap     SiteMap
}

type Feature struct {
	Key     string
	Label   string
	Surface SurfaceKind
	Summary string
}

type ResourceAdapter struct {
	Kind         ResourceKind
	Label        string
	Summary      string
	Surface      SurfaceKind
	Capabilities []ResourceCapability
	Bindings     []ResourceBinding
}

type ResourceBinding struct {
	Key     string
	Label   string
	Summary string
	Binding string
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

func DefaultResourceAdapters() []ResourceAdapter {
	return []ResourceAdapter{
		{
			Kind:    ResourceMedia,
			Label:   "Media",
			Summary: "Images, video, generated assets, alt text, and focal points.",
			Surface: SurfaceInspector,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
				ResourceSearch,
				ResourcePreview,
			},
			Bindings: []ResourceBinding{
				{Key: "assets", Label: "Assets", Binding: "media.assets"},
				{Key: "uploads", Label: "Uploads", Binding: "media.uploads"},
			},
		},
		{
			Kind:    ResourcePages,
			Label:   "Pages",
			Summary: "Routes, page drafts, page templates, previews, and page publishing state.",
			Surface: SurfaceSiteMap,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
				ResourcePreview,
				ResourcePublish,
			},
			Bindings: []ResourceBinding{
				{Key: "routes", Label: "Routes", Binding: "pages.routes"},
				{Key: "drafts", Label: "Drafts", Binding: "pages.drafts"},
			},
		},
		{
			Kind:    ResourceProducts,
			Label:   "Products",
			Summary: "Product collections, categories, availability, and storefront placement.",
			Surface: SurfaceSiteMap,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceSearch,
				ResourcePreview,
			},
			Bindings: []ResourceBinding{
				{Key: "collection", Label: "Collection", Binding: "products.collection"},
				{Key: "categories", Label: "Categories", Binding: "products.categories"},
			},
		},
		{
			Kind:    ResourceOrders,
			Label:   "Orders",
			Summary: "Store activity that can inform storefront messaging without making Studio the order desk.",
			Surface: SurfacePublish,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
			},
			Bindings: []ResourceBinding{
				{Key: "activity", Label: "Activity", Binding: "orders.activity"},
			},
		},
		{
			Kind:    ResourceContacts,
			Label:   "Contacts",
			Summary: "Contact destinations, customer messages, and form routing context.",
			Surface: SurfaceFlow,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
			},
			Bindings: []ResourceBinding{
				{Key: "messages", Label: "Messages", Binding: "contacts.messages"},
				{Key: "destinations", Label: "Destinations", Binding: "contacts.destinations"},
			},
		},
		{
			Kind:    ResourceSettings,
			Label:   "Settings",
			Summary: "Site identity, theme, navigation, checkout, and operational settings.",
			Surface: SurfaceInspector,
			Capabilities: []ResourceCapability{
				ResourceRead,
				ResourceWrite,
				ResourcePreview,
			},
			Bindings: []ResourceBinding{
				{Key: "site", Label: "Site", Binding: "settings.site"},
				{Key: "theme", Label: "Theme", Binding: "settings.theme"},
			},
		},
		{
			Kind:    ResourceRevisions,
			Label:   "Revisions",
			Summary: "Saved versions, compare points, and restore actions.",
			Surface: SurfacePublish,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceRestore,
			},
			Bindings: []ResourceBinding{
				{Key: "history", Label: "History", Binding: "revisions.history"},
			},
		},
		{
			Kind:    ResourceLifecycle,
			Label:   "Lifecycle",
			Summary: "Readiness, scheduling, approvals, preview links, and publish decisions.",
			Surface: SurfacePublish,
			Capabilities: []ResourceCapability{
				ResourceRead,
				ResourceWrite,
				ResourcePreview,
				ResourcePublish,
			},
			Bindings: []ResourceBinding{
				{Key: "readiness", Label: "Readiness", Binding: "lifecycle.readiness"},
				{Key: "schedule", Label: "Schedule", Binding: "lifecycle.schedule"},
			},
		},
		{
			Kind:    ResourceFlows,
			Label:   "Flows",
			Summary: "Visual automations, form handlers, lifecycle actions, and publishable flow drafts.",
			Surface: SurfaceFlow,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
				ResourceExecute,
			},
			Bindings: []ResourceBinding{
				{Key: "library", Label: "Library", Binding: "flows.library"},
				{Key: "drafts", Label: "Drafts", Binding: "flows.drafts"},
			},
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

func (config HostConfig) ResourceAdapter(kind ResourceKind) (ResourceAdapter, bool) {
	return ResourceAdapterByKind(config.Adapters, kind)
}

func ResourceAdapterByKind(adapters []ResourceAdapter, kind ResourceKind) (ResourceAdapter, bool) {
	normalized := normalizeResourceKind(kind)
	for _, adapter := range adapters {
		if normalizeResourceKind(adapter.Kind) == normalized {
			return adapter, true
		}
	}
	return ResourceAdapter{}, false
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

func (adapter ResourceAdapter) NormalizedKind() ResourceKind {
	return normalizeResourceKind(adapter.Kind)
}

func (adapter ResourceAdapter) CapabilityCount() int {
	return len(adapter.Capabilities)
}

func (adapter ResourceAdapter) BindingCount() int {
	return len(adapter.Bindings)
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

func (intent CompositionIntent) StepCount() int {
	return len(intent.Steps)
}

func (intent CompositionIntent) NormalizedKind() CompositionIntentKind {
	return normalizeCompositionIntentKind(intent.Kind)
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

func normalizeResourceKind(kind ResourceKind) ResourceKind {
	switch ResourceKind(strings.TrimSpace(string(kind))) {
	case ResourcePages:
		return ResourcePages
	case ResourceProducts:
		return ResourceProducts
	case ResourceOrders:
		return ResourceOrders
	case ResourceContacts:
		return ResourceContacts
	case ResourceSettings:
		return ResourceSettings
	case ResourceRevisions:
		return ResourceRevisions
	case ResourceLifecycle:
		return ResourceLifecycle
	case ResourceFlows:
		return ResourceFlows
	default:
		return ResourceMedia
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

func normalizeCompositionIntentKind(kind CompositionIntentKind) CompositionIntentKind {
	switch CompositionIntentKind(strings.TrimSpace(string(kind))) {
	case CompositionIntentCreatePage:
		return CompositionIntentCreatePage
	default:
		return CompositionIntentAddComponent
	}
}
