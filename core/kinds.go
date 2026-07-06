package core

import "strings"

const (
	PackageCMS    = "m31labs.dev/gosx-cms"
	PackageAdmin  = "m31labs.dev/gosx-admin"
	PackageStudio = "m31labs.dev/gosx-studio"
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

type ReadinessStatus string

const (
	ReadinessReady   ReadinessStatus = "ready"
	ReadinessWatch   ReadinessStatus = "watch"
	ReadinessBlocked ReadinessStatus = "blocked"
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

type CanvasActionKind string

const (
	CanvasActionReveal           CanvasActionKind = "reveal"
	CanvasActionMoveUp           CanvasActionKind = "move-up"
	CanvasActionMoveDown         CanvasActionKind = "move-down"
	CanvasActionInlineText       CanvasActionKind = "inline-text"
	CanvasActionContent          CanvasActionKind = "content"
	CanvasActionStyle            CanvasActionKind = "style"
	CanvasActionToggleVisibility CanvasActionKind = "toggle-visibility"
)

func normalizeSurfaceKind(kind SurfaceKind) SurfaceKind {
	switch SurfaceKind(strings.TrimSpace(string(kind))) {
	case SurfaceSiteMap:
		return SurfaceSiteMap
	case SurfaceInspector:
		return SurfaceInspector
	case SurfaceFlow:
		return SurfaceFlow
	case SurfacePublish:
		return SurfacePublish
	case SurfaceShowcase3D:
		return SurfaceShowcase3D
	default:
		return SurfaceCanvas
	}
}
