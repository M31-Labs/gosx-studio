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

type WorkspaceNodeKind string

const (
	WorkspaceNodePage      WorkspaceNodeKind = "page"
	WorkspaceNodeComponent WorkspaceNodeKind = "component"
	WorkspaceNodeResource  WorkspaceNodeKind = "resource"
)

type WorkspaceLinkKind string

const (
	WorkspaceLinkContains WorkspaceLinkKind = "contains"
	WorkspaceLinkBinds    WorkspaceLinkKind = "binds"
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
	Selected      bool
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

type CompositionWorkspace struct {
	Layers []WorkspaceLayer
	Nodes  []WorkspaceNode
	Links  []WorkspaceLink
}

type WorkspaceLayer struct {
	Key      string
	Label    string
	Summary  string
	NodeKeys []string
}

type WorkspaceNode struct {
	Key           string
	Label         string
	Summary       string
	Kind          WorkspaceNodeKind
	LayerKey      string
	PageKey       string
	Route         string
	Group         PageGroup
	GoSXComponent string
	Source        ComponentSource
	Binding       string
	Status        string
	Selected      bool
}

type WorkspaceLink struct {
	Key         string
	Label       string
	Summary     string
	Kind        WorkspaceLinkKind
	FromNodeKey string
	ToNodeKey   string
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

func (siteMap SiteMap) CompositionWorkspace() CompositionWorkspace {
	workspace := CompositionWorkspace{}
	resourceNodeKeys := map[string]string{}

	for _, page := range siteMap.Pages {
		pageKey := strings.TrimSpace(page.Key)
		if pageKey == "" {
			continue
		}

		layer := WorkspaceLayer{
			Key:     pageKey,
			Label:   strings.TrimSpace(page.Label),
			Summary: strings.TrimSpace(page.Route),
		}
		pageNodeKey := "page:" + workspaceToken(pageKey)
		workspace.Nodes = append(workspace.Nodes, WorkspaceNode{
			Key:           pageNodeKey,
			Label:         strings.TrimSpace(page.Label),
			Summary:       "Route " + strings.TrimSpace(page.Route),
			Kind:          WorkspaceNodePage,
			LayerKey:      pageKey,
			PageKey:       pageKey,
			Route:         strings.TrimSpace(page.Route),
			Group:         page.NormalizedGroup(),
			GoSXComponent: strings.TrimSpace(page.GoSXComponent),
			Status:        strings.TrimSpace(page.Status),
			Selected:      page.Selected,
		})
		layer.NodeKeys = append(layer.NodeKeys, pageNodeKey)

		for _, component := range page.Components {
			componentKey := strings.TrimSpace(component.Key)
			if componentKey == "" {
				continue
			}
			componentNodeKey := "component:" + workspaceToken(pageKey) + ":" + workspaceToken(componentKey)
			workspace.Nodes = append(workspace.Nodes, WorkspaceNode{
				Key:           componentNodeKey,
				Label:         strings.TrimSpace(component.Label),
				Summary:       strings.TrimSpace(component.Summary),
				Kind:          WorkspaceNodeComponent,
				LayerKey:      pageKey,
				PageKey:       pageKey,
				Route:         strings.TrimSpace(page.Route),
				Group:         page.NormalizedGroup(),
				GoSXComponent: strings.TrimSpace(component.GoSXComponent),
				Source:        component.NormalizedSource(),
				Binding:       strings.TrimSpace(component.Binding),
				Status:        strings.TrimSpace(component.Status),
			})
			layer.NodeKeys = append(layer.NodeKeys, componentNodeKey)
			workspace.Links = append(workspace.Links, WorkspaceLink{
				Key:         "contains:" + workspaceToken(pageKey) + ":" + workspaceToken(componentKey),
				Label:       "Contains",
				Summary:     strings.TrimSpace(page.Label) + " contains " + strings.TrimSpace(component.Label),
				Kind:        WorkspaceLinkContains,
				FromNodeKey: pageNodeKey,
				ToNodeKey:   componentNodeKey,
			})

			binding := strings.TrimSpace(component.Binding)
			if binding == "" {
				continue
			}
			resourceNodeKey, ok := resourceNodeKeys[binding]
			if !ok {
				resourceNodeKey = "resource:" + workspaceToken(binding)
				resourceNodeKeys[binding] = resourceNodeKey
				workspace.Nodes = append(workspace.Nodes, WorkspaceNode{
					Key:     resourceNodeKey,
					Label:   binding,
					Summary: ComponentSourceLabel(component.Source) + " binding",
					Kind:    WorkspaceNodeResource,
					Source:  component.NormalizedSource(),
					Binding: binding,
					Status:  ComponentSourceLabel(component.Source),
				})
			}
			workspace.Links = append(workspace.Links, WorkspaceLink{
				Key:         "binds:" + workspaceToken(pageKey) + ":" + workspaceToken(componentKey) + ":" + workspaceToken(binding),
				Label:       "Binds",
				Summary:     strings.TrimSpace(component.Label) + " uses " + binding,
				Kind:        WorkspaceLinkBinds,
				FromNodeKey: componentNodeKey,
				ToNodeKey:   resourceNodeKey,
			})
		}

		workspace.Layers = append(workspace.Layers, layer)
	}

	if len(resourceNodeKeys) > 0 {
		resourceLayer := WorkspaceLayer{
			Key:     "resources",
			Label:   "Resources",
			Summary: "Content, flow, media, and plugin bindings used by the site.",
		}
		for _, node := range workspace.Nodes {
			if node.Kind == WorkspaceNodeResource {
				resourceLayer.NodeKeys = append(resourceLayer.NodeKeys, node.Key)
			}
		}
		workspace.Layers = append(workspace.Layers, resourceLayer)
	}

	return workspace.Normalize()
}

func (workspace CompositionWorkspace) Normalize() CompositionWorkspace {
	out := CompositionWorkspace{
		Layers: make([]WorkspaceLayer, 0, len(workspace.Layers)),
		Nodes:  make([]WorkspaceNode, 0, len(workspace.Nodes)),
		Links:  make([]WorkspaceLink, 0, len(workspace.Links)),
	}
	for _, layer := range workspace.Layers {
		layer.Key = strings.TrimSpace(layer.Key)
		layer.Label = strings.TrimSpace(layer.Label)
		layer.Summary = strings.TrimSpace(layer.Summary)
		if layer.Key == "" || layer.Label == "" {
			continue
		}
		layer.NodeKeys = normalizeWorkspaceNodeKeys(layer.NodeKeys)
		out.Layers = append(out.Layers, layer)
	}
	for _, node := range workspace.Nodes {
		node = node.Normalize()
		if node.Key == "" || node.Label == "" {
			continue
		}
		out.Nodes = append(out.Nodes, node)
	}
	for _, link := range workspace.Links {
		link = link.Normalize()
		if link.Key == "" || link.FromNodeKey == "" || link.ToNodeKey == "" {
			continue
		}
		out.Links = append(out.Links, link)
	}
	return out
}

func (workspace CompositionWorkspace) NodeCount() int {
	return len(workspace.Nodes)
}

func (workspace CompositionWorkspace) LinkCount() int {
	return len(workspace.Links)
}

func (workspace CompositionWorkspace) LayerCount() int {
	return len(workspace.Layers)
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

func (node WorkspaceNode) Normalize() WorkspaceNode {
	node.Key = strings.TrimSpace(node.Key)
	node.Label = strings.TrimSpace(node.Label)
	node.Summary = strings.TrimSpace(node.Summary)
	node.Kind = normalizeWorkspaceNodeKind(node.Kind)
	node.LayerKey = strings.TrimSpace(node.LayerKey)
	node.PageKey = strings.TrimSpace(node.PageKey)
	node.Route = strings.TrimSpace(node.Route)
	node.Group = normalizePageGroup(node.Group)
	node.GoSXComponent = strings.TrimSpace(node.GoSXComponent)
	node.Source = normalizeComponentSource(node.Source)
	node.Binding = strings.TrimSpace(node.Binding)
	node.Status = strings.TrimSpace(node.Status)
	return node
}

func (node WorkspaceNode) NormalizedKind() WorkspaceNodeKind {
	return normalizeWorkspaceNodeKind(node.Kind)
}

func (link WorkspaceLink) Normalize() WorkspaceLink {
	link.Key = strings.TrimSpace(link.Key)
	link.Label = strings.TrimSpace(link.Label)
	link.Summary = strings.TrimSpace(link.Summary)
	link.Kind = normalizeWorkspaceLinkKind(link.Kind)
	link.FromNodeKey = strings.TrimSpace(link.FromNodeKey)
	link.ToNodeKey = strings.TrimSpace(link.ToNodeKey)
	return link
}

func (link WorkspaceLink) NormalizedKind() WorkspaceLinkKind {
	return normalizeWorkspaceLinkKind(link.Kind)
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

func WorkspaceNodeKindLabel(kind WorkspaceNodeKind) string {
	switch normalizeWorkspaceNodeKind(kind) {
	case WorkspaceNodeComponent:
		return "Component"
	case WorkspaceNodeResource:
		return "Resource"
	default:
		return "Page"
	}
}

func WorkspaceLinkKindLabel(kind WorkspaceLinkKind) string {
	switch normalizeWorkspaceLinkKind(kind) {
	case WorkspaceLinkBinds:
		return "Binding"
	default:
		return "Contains"
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

func normalizeWorkspaceNodeKind(kind WorkspaceNodeKind) WorkspaceNodeKind {
	switch WorkspaceNodeKind(strings.TrimSpace(string(kind))) {
	case WorkspaceNodeComponent:
		return WorkspaceNodeComponent
	case WorkspaceNodeResource:
		return WorkspaceNodeResource
	default:
		return WorkspaceNodePage
	}
}

func normalizeWorkspaceLinkKind(kind WorkspaceLinkKind) WorkspaceLinkKind {
	switch WorkspaceLinkKind(strings.TrimSpace(string(kind))) {
	case WorkspaceLinkBinds:
		return WorkspaceLinkBinds
	default:
		return WorkspaceLinkContains
	}
}

func normalizeWorkspaceNodeKeys(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func workspaceToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "node"
	}
	return token
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
