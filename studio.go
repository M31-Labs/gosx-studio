package studio

import (
	"strconv"
	"strings"
)

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

type SiteMap struct {
	Pages   []Page
	Library CompositionLibrary
}

type CanvasWorkspace struct {
	RouteLabel   string
	PreviewURL   string
	PreviewShell CanvasPreviewShell
	Viewports    []CanvasViewport
	ZoomLevels   []CanvasZoomLevel
	Blocks       []CanvasBlock
	Actions      []CanvasAction
}

type CanvasViewport struct {
	Key    string
	Label  string
	Width  string
	Active bool
}

type CanvasZoomLevel struct {
	Key    string
	Label  string
	Scale  float64
	Active bool
}

type CanvasBlock struct {
	Key           string
	Label         string
	Summary       string
	GoSXComponent string
	Source        ComponentSource
	Binding       string
	Status        string
	Visible       bool
	Selected      bool
	Editable      bool
	Controls      []Control
}

type CanvasAction struct {
	Key     string
	Label   string
	Summary string
	Kind    CanvasActionKind
	Enabled bool
}

type CanvasPreviewShell struct {
	OverlayAttr         string
	DropLineAttr        string
	DockAttr            string
	DockTitleAttr       string
	FieldCountAttr      string
	ActionAttr          string
	OutlineTemplateAttr string
	OutlineLabelAttr    string
	FieldTemplateAttr   string
	FieldActionTextAttr string
	InlineEditClass     string
	StyleImpactNodeAttr string
	FrameAttr           string
	FrameURLAttr        string
	ViewportAttr        string
	ViewportCurrentAttr string
	ViewportWidthAttr   string
	SelectionLabelAttr  string
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

type Flow struct {
	Key                string
	Label              string
	Description        string
	Summary            string
	Route              string
	EmbedTarget        string
	HasRoute           bool
	HasEmbedTarget     bool
	HandlerRef         string
	CanExecute         bool
	Steps              []FlowStep
	Actions            []FlowAction
	FieldCount         int
	RequiredFieldCount int
}

type FlowStep struct {
	Key        string
	Label      string
	Summary    string
	BlockCount int
	HasBlocks  bool
}

type FlowAction struct {
	Key        string
	Label      string
	HandlerRef string
	CanExecute bool
	Fields     []FlowField
}

type FlowField struct {
	Name     string
	Label    string
	Kind     ControlKind
	Required bool
}

type FlowReadinessCheck struct {
	Key     string
	Label   string
	Summary string
	Status  ReadinessStatus
}

type FlowNode struct {
	Key     string
	Label   string
	Summary string
	Kind    string
	Status  ReadinessStatus
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

type RuntimeContract struct {
	Key     string
	Label   string
	Global  string
	Surface SurfaceKind
	Engine  EngineKind
	Methods []RuntimeMethod
}

type RuntimeMethod struct {
	Name    string
	Label   string
	Summary string
	Payload []RuntimePayloadField
}

type RuntimePayloadField struct {
	Name     string
	Label    string
	Kind     ControlKind
	Summary  string
	Required bool
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
	Runtimes    []RuntimeContract
	Adapters    []ResourceAdapter
	SiteMap     SiteMap
	Canvas      CanvasWorkspace
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
				CapabilityDragDrop,
				CapabilityPanZoom,
				CapabilitySelection,
				CapabilityInlineEdit,
				CapabilityPreview,
				CapabilityPersistence,
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
		{
			Key:     "showcase-3d",
			Label:   "Showcase 3D",
			Kind:    EngineScene3D,
			MountID: "gosx-studio-showcase-3d-engine",
			Surface: SurfaceShowcase3D,
			Capabilities: []EngineCapability{
				CapabilityPreview,
				CapabilityPopout,
			},
		},
	}
}

func DefaultRuntimeContracts() []RuntimeContract {
	return []RuntimeContract{
		{
			Key:     "preview-runtime",
			Label:   "Preview runtime",
			Global:  "GoSXStudioPreviewRuntime",
			Surface: SurfaceCanvas,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "mount",
					Label:   "Mount preview shell",
					Summary: "Bind the GoSX-authored preview shell, overlays, dock controls, and inline editing affordances.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains preview workbench markup."},
					},
				},
				{
					Name:    "setBlockVisibility",
					Label:   "Set block visibility",
					Summary: "Apply visible or hidden state to a GoSX-backed preview block.",
					Payload: []RuntimePayloadField{
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "visible", Label: "Visible", Kind: ControlToggle, Required: true},
					},
				},
				{
					Name:    "applyCSS",
					Label:   "Apply custom CSS",
					Summary: "Apply editor-provided custom CSS to the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "cssText", Label: "CSS text", Kind: ControlRichText},
					},
				},
				{
					Name:    "applyTextUpdate",
					Label:   "Apply text update",
					Summary: "Mirror editor text and attribute edits into the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "sourceKey", Label: "Source key", Kind: ControlText},
						{Name: "frameTarget", Label: "Frame target", Kind: ControlText},
						{Name: "attrTarget", Label: "Attribute target", Kind: ControlText},
						{Name: "attrName", Label: "Attribute name", Kind: ControlText},
						{Name: "attrPrefix", Label: "Attribute prefix", Kind: ControlText},
						{Name: "attrSuffix", Label: "Attribute suffix", Kind: ControlText},
						{Name: "value", Label: "Value", Kind: ControlRichText},
					},
				},
				{
					Name:    "applyTheme",
					Label:   "Apply theme",
					Summary: "Apply selected kit, template, palette, image ratio, style classes, and color tokens to the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "kit", Label: "Kit", Kind: ControlChoice},
						{Name: "template", Label: "Template", Kind: ControlChoice},
						{Name: "palette", Label: "Palette", Kind: ControlChoice},
						{Name: "imageRatio", Label: "Image ratio", Kind: ControlChoice},
						{Name: "customClasses", Label: "Custom classes", Kind: ControlText},
						{Name: "styleClasses", Label: "Style classes", Kind: ControlText},
						{Name: "colors", Label: "Color tokens", Kind: ControlColor},
					},
				},
				{
					Name:    "applyStyleImpact",
					Label:   "Apply style impact",
					Summary: "Clear previous style-impact markers and mark preview nodes affected by a selected style scope.",
					Payload: []RuntimePayloadField{
						{Name: "selector", Label: "Scope selector", Kind: ControlText},
					},
				},
				{
					Name:    "applyFonts",
					Label:   "Apply font CSS",
					Summary: "Apply generated font-face and font-token CSS to the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "cssText", Label: "CSS text", Kind: ControlRichText},
					},
				},
				{
					Name:    "updateHeaderLogo",
					Label:   "Update header logo",
					Summary: "Update live header logo source, alt text, size, and offsets in the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "url", Label: "Logo URL", Kind: ControlMedia},
						{Name: "alt", Label: "Alt text", Kind: ControlText},
						{Name: "width", Label: "Width", Kind: ControlNumber},
						{Name: "x", Label: "X offset", Kind: ControlNumber},
						{Name: "y", Label: "Y offset", Kind: ControlNumber},
					},
				},
				{
					Name:    "requestInlineEdit",
					Label:   "Request inline edit",
					Summary: "Focus a field hotspot and open the inline text editing affordance for the selected block.",
					Payload: []RuntimePayloadField{
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "field", Label: "Field key", Kind: ControlText},
					},
				},
				{
					Name:    "cycleField",
					Label:   "Cycle field",
					Summary: "Move field selection within the selected preview block.",
					Payload: []RuntimePayloadField{
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "direction", Label: "Direction", Kind: ControlNumber, Required: true},
					},
				},
			},
		},
		{
			Key:     "workbench-runtime",
			Label:   "Workbench runtime",
			Global:  "GoSXStudioWorkbenchRuntime",
			Surface: SurfaceCanvas,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "bindRailResizers",
					Label:   "Bind rail resizers",
					Summary: "Bind pointer and keyboard resizing for GoSX-authored workbench rail handles.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the Studio workbench."},
					},
				},
				{
					Name:    "currentRailWidth",
					Label:   "Current rail width",
					Summary: "Read the current width of a Studio workbench rail.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "side", Label: "Rail side", Kind: ControlChoice, Required: true},
						{Name: "handle", Label: "Rail handle", Kind: ControlSource},
					},
				},
				{
					Name:    "setRailWidth",
					Label:   "Set rail width",
					Summary: "Apply a clamped width to a Studio workbench rail and emit live or committed resize events.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "side", Label: "Rail side", Kind: ControlChoice, Required: true},
						{Name: "width", Label: "Rail width", Kind: ControlNumber, Required: true},
						{Name: "handle", Label: "Rail handle", Kind: ControlSource},
						{Name: "committed", Label: "Committed", Kind: ControlToggle},
					},
				},
			},
		},
		{
			Key:     "brand-runtime",
			Label:   "Brand runtime",
			Global:  "GoSXStudioBrandRuntime",
			Surface: SurfaceInspector,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "bindLogo",
					Label:   "Bind logo controls",
					Summary: "Bind logo placement, snap, keyboard nudge, reset, and live-preview controls for the Brand inspector.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the Brand inspector."},
					},
				},
				{
					Name:    "updateHeaderLogo",
					Label:   "Update header logo",
					Summary: "Send Brand inspector logo source, alt text, size, and offsets to the preview runtime.",
					Payload: []RuntimePayloadField{
						{Name: "url", Label: "Logo URL", Kind: ControlMedia},
						{Name: "alt", Label: "Alt text", Kind: ControlText},
						{Name: "width", Label: "Width", Kind: ControlNumber},
						{Name: "x", Label: "X offset", Kind: ControlNumber},
						{Name: "y", Label: "Y offset", Kind: ControlNumber},
					},
				},
			},
		},
		{
			Key:     "block-layout-runtime",
			Label:   "Block layout runtime",
			Global:  "GoSXStudioBlockLayoutRuntime",
			Surface: SurfaceCanvas,
			Engine:  EngineBlockLayout,
			Methods: []RuntimeMethod{
				{
					Name:    "rows",
					Label:   "Rows",
					Summary: "Return the editable block rows in a page layout list.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "rowKey",
					Label:   "Row key",
					Summary: "Read the stable GoSX block key for a layout row.",
					Payload: []RuntimePayloadField{
						{Name: "row", Label: "Layout row", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "rowForKey",
					Label:   "Row for key",
					Summary: "Find a block layout row by stable GoSX block key.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
					},
				},
				{
					Name:    "moveRow",
					Label:   "Move row",
					Summary: "Move a block row within the current layout.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "direction", Label: "Direction", Kind: ControlNumber, Required: true},
					},
				},
				{
					Name:    "renumber",
					Label:   "Renumber rows",
					Summary: "Refresh visible position labels after a block layout change.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "selectRow",
					Label:   "Select row",
					Summary: "Select the row that represents a GoSX-backed page block.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
					},
				},
				{
					Name:    "commitReorder",
					Label:   "Commit reorder",
					Summary: "Commit a drag/drop reorder operation against the current block layout.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "targetKey", Label: "Target block key", Kind: ControlText, Required: true},
						{Name: "position", Label: "Position", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "updateBlockLibraryState",
					Label:   "Update block library state",
					Summary: "Refresh component-palette availability and active state from the current page layout.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the block library."},
					},
				},
				{
					Name:    "updateVisibilityState",
					Label:   "Update visibility state",
					Summary: "Refresh a block row's visibility status, component-palette state, and live preview visibility.",
					Payload: []RuntimePayloadField{
						{Name: "check", Label: "Visibility control", Kind: ControlToggle, Required: true},
					},
				},
			},
		},
	}
}

func DefaultCanvasActions() []CanvasAction {
	return []CanvasAction{
		{Key: string(CanvasActionReveal), Kind: CanvasActionReveal, Label: "Reveal", Summary: "Focus the selected block in the canvas.", Enabled: true},
		{Key: string(CanvasActionMoveUp), Kind: CanvasActionMoveUp, Label: "Move up", Summary: "Move the selected block earlier on the page.", Enabled: true},
		{Key: string(CanvasActionMoveDown), Kind: CanvasActionMoveDown, Label: "Move down", Summary: "Move the selected block later on the page.", Enabled: true},
		{Key: string(CanvasActionInlineText), Kind: CanvasActionInlineText, Label: "Edit text", Summary: "Edit the primary text field for the selection.", Enabled: true},
		{Key: string(CanvasActionContent), Kind: CanvasActionContent, Label: "Content", Summary: "Open content controls for the selected block.", Enabled: true},
		{Key: string(CanvasActionStyle), Kind: CanvasActionStyle, Label: "Style", Summary: "Open style controls for the selected block.", Enabled: true},
		{Key: string(CanvasActionToggleVisibility), Kind: CanvasActionToggleVisibility, Label: "Hide", Summary: "Toggle whether the selected block is visible.", Enabled: true},
	}
}

func DefaultCanvasPreviewShell() CanvasPreviewShell {
	return CanvasPreviewShell{
		OverlayAttr:         "data-studio-preview-overlay",
		DropLineAttr:        "data-studio-preview-drop-line",
		DockAttr:            "data-studio-preview-dock",
		DockTitleAttr:       "data-studio-preview-dock-title",
		FieldCountAttr:      "data-studio-preview-field-count",
		ActionAttr:          "data-studio-preview-action",
		OutlineTemplateAttr: "data-studio-preview-outline-template",
		OutlineLabelAttr:    "data-studio-preview-outline-label",
		FieldTemplateAttr:   "data-studio-preview-field-template",
		FieldActionTextAttr: "data-studio-preview-field-action-text",
		InlineEditClass:     "is-studio-inline-editing",
		StyleImpactNodeAttr: "data-studio-style-impact-node",
		FrameAttr:           "data-studio-preview-frame",
		FrameURLAttr:        "data-studio-preview-src",
		ViewportAttr:        "data-studio-preview-viewport",
		ViewportCurrentAttr: "data-studio-viewport-current",
		ViewportWidthAttr:   "data-studio-viewport-width",
		SelectionLabelAttr:  "data-studio-selection-label",
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

func RuntimeContractByGlobal(contracts []RuntimeContract, global string) (RuntimeContract, bool) {
	global = strings.TrimSpace(global)
	for _, contract := range contracts {
		contract = contract.Normalize()
		if contract.Global == global {
			return contract, true
		}
	}
	return RuntimeContract{}, false
}

func (contract RuntimeContract) Normalize() RuntimeContract {
	contract.Key = strings.TrimSpace(contract.Key)
	contract.Label = strings.TrimSpace(contract.Label)
	contract.Global = strings.TrimSpace(contract.Global)
	contract.Surface = normalizeSurfaceKind(contract.Surface)
	contract.Engine = normalizeEngineKind(contract.Engine)
	contract.Methods = normalizeRuntimeMethods(contract.Methods)
	return contract
}

func (contract RuntimeContract) Method(name string) (RuntimeMethod, bool) {
	name = strings.TrimSpace(name)
	for _, method := range contract.Normalize().Methods {
		if method.Name == name {
			return method, true
		}
	}
	return RuntimeMethod{}, false
}

func (contract RuntimeContract) MethodCount() int {
	return len(contract.Normalize().Methods)
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

func (canvas CanvasWorkspace) Normalize() CanvasWorkspace {
	canvas.RouteLabel = strings.TrimSpace(canvas.RouteLabel)
	canvas.PreviewURL = strings.TrimSpace(canvas.PreviewURL)
	canvas.PreviewShell = canvas.PreviewShell.Normalize()
	canvas.Viewports = normalizeCanvasViewports(canvas.Viewports)
	canvas.ZoomLevels = normalizeCanvasZoomLevels(canvas.ZoomLevels)
	canvas.Blocks = normalizeCanvasBlocks(canvas.Blocks)
	canvas.Actions = normalizeCanvasActions(canvas.Actions)
	return canvas
}

func (shell CanvasPreviewShell) Normalize() CanvasPreviewShell {
	defaults := DefaultCanvasPreviewShell()
	shell.OverlayAttr = firstNonEmpty(strings.TrimSpace(shell.OverlayAttr), defaults.OverlayAttr)
	shell.DropLineAttr = firstNonEmpty(strings.TrimSpace(shell.DropLineAttr), defaults.DropLineAttr)
	shell.DockAttr = firstNonEmpty(strings.TrimSpace(shell.DockAttr), defaults.DockAttr)
	shell.DockTitleAttr = firstNonEmpty(strings.TrimSpace(shell.DockTitleAttr), defaults.DockTitleAttr)
	shell.FieldCountAttr = firstNonEmpty(strings.TrimSpace(shell.FieldCountAttr), defaults.FieldCountAttr)
	shell.ActionAttr = firstNonEmpty(strings.TrimSpace(shell.ActionAttr), defaults.ActionAttr)
	shell.OutlineTemplateAttr = firstNonEmpty(strings.TrimSpace(shell.OutlineTemplateAttr), defaults.OutlineTemplateAttr)
	shell.OutlineLabelAttr = firstNonEmpty(strings.TrimSpace(shell.OutlineLabelAttr), defaults.OutlineLabelAttr)
	shell.FieldTemplateAttr = firstNonEmpty(strings.TrimSpace(shell.FieldTemplateAttr), defaults.FieldTemplateAttr)
	shell.FieldActionTextAttr = firstNonEmpty(strings.TrimSpace(shell.FieldActionTextAttr), defaults.FieldActionTextAttr)
	shell.InlineEditClass = firstNonEmpty(strings.TrimSpace(shell.InlineEditClass), defaults.InlineEditClass)
	shell.StyleImpactNodeAttr = firstNonEmpty(strings.TrimSpace(shell.StyleImpactNodeAttr), defaults.StyleImpactNodeAttr)
	shell.FrameAttr = firstNonEmpty(strings.TrimSpace(shell.FrameAttr), defaults.FrameAttr)
	shell.FrameURLAttr = firstNonEmpty(strings.TrimSpace(shell.FrameURLAttr), defaults.FrameURLAttr)
	shell.ViewportAttr = firstNonEmpty(strings.TrimSpace(shell.ViewportAttr), defaults.ViewportAttr)
	shell.ViewportCurrentAttr = firstNonEmpty(strings.TrimSpace(shell.ViewportCurrentAttr), defaults.ViewportCurrentAttr)
	shell.ViewportWidthAttr = firstNonEmpty(strings.TrimSpace(shell.ViewportWidthAttr), defaults.ViewportWidthAttr)
	shell.SelectionLabelAttr = firstNonEmpty(strings.TrimSpace(shell.SelectionLabelAttr), defaults.SelectionLabelAttr)
	return shell
}

func (canvas CanvasWorkspace) BlockCount() int {
	return len(canvas.Normalize().Blocks)
}

func (canvas CanvasWorkspace) VisibleBlockCount() int {
	count := 0
	for _, block := range canvas.Normalize().Blocks {
		if block.Visible {
			count++
		}
	}
	return count
}

func (canvas CanvasWorkspace) ControlCount() int {
	count := 0
	for _, block := range canvas.Normalize().Blocks {
		count += block.ControlCount()
	}
	return count
}

func (canvas CanvasWorkspace) ActiveViewport() CanvasViewport {
	viewports := canvas.Normalize().Viewports
	for _, viewport := range viewports {
		if viewport.Active {
			return viewport
		}
	}
	if len(viewports) == 0 {
		return CanvasViewport{}
	}
	return viewports[0]
}

func (canvas CanvasWorkspace) ActiveZoom() CanvasZoomLevel {
	levels := canvas.Normalize().ZoomLevels
	for _, level := range levels {
		if level.Active {
			return level
		}
	}
	if len(levels) == 0 {
		return CanvasZoomLevel{}
	}
	return levels[0]
}

func (canvas CanvasWorkspace) SelectedBlock() (CanvasBlock, bool) {
	for _, block := range canvas.Normalize().Blocks {
		if block.Selected {
			return block, true
		}
	}
	return CanvasBlock{}, false
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

func (block CanvasBlock) ControlCount() int {
	return len(block.Controls)
}

func (block CanvasBlock) NormalizedSource() ComponentSource {
	return normalizeComponentSource(block.Source)
}

func (action CanvasAction) NormalizedKind() CanvasActionKind {
	return normalizeCanvasActionKind(action.Kind)
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

func (flow Flow) Normalize() Flow {
	flow.Key = strings.TrimSpace(flow.Key)
	flow.Label = strings.TrimSpace(flow.Label)
	flow.Description = strings.TrimSpace(flow.Description)
	flow.Summary = strings.TrimSpace(flow.Summary)
	flow.Route = strings.TrimSpace(flow.Route)
	flow.EmbedTarget = strings.TrimSpace(flow.EmbedTarget)
	flow.HandlerRef = strings.TrimSpace(flow.HandlerRef)
	flow.Steps = normalizeFlowSteps(flow.Steps)
	flow.Actions = normalizeFlowActions(flow.Actions)
	if flow.FieldCount == 0 {
		for _, action := range flow.Actions {
			flow.FieldCount += len(action.Fields)
		}
	}
	if flow.RequiredFieldCount == 0 {
		for _, action := range flow.Actions {
			for _, field := range action.Fields {
				if field.Required {
					flow.RequiredFieldCount++
				}
			}
		}
	}
	return flow
}

func (flow Flow) ReadinessChecks() []FlowReadinessCheck {
	flow = flow.Normalize()
	return []FlowReadinessCheck{
		{
			Key:     "handler",
			Label:   "Submission action",
			Summary: flowHandlerReadinessSummary(flow),
			Status:  flowHandlerReadinessStatus(flow),
		},
		{
			Key:     "placement",
			Label:   "Public placement",
			Summary: flowPlacementReadinessSummary(flow),
			Status:  flowPlacementReadinessStatus(flow),
		},
		{
			Key:     "structure",
			Label:   "Steps and actions",
			Summary: flowStructureReadinessSummary(flow),
			Status:  flowStructureReadinessStatus(flow),
		},
		{
			Key:     "fields",
			Label:   "Editor fields",
			Summary: flowFieldsReadinessSummary(flow),
			Status:  flowFieldsReadinessStatus(flow),
		},
	}
}

func (flow Flow) ReadinessStatus() ReadinessStatus {
	status := ReadinessReady
	for _, check := range flow.ReadinessChecks() {
		switch normalizeReadinessStatus(check.Status) {
		case ReadinessBlocked:
			return ReadinessBlocked
		case ReadinessWatch:
			status = ReadinessWatch
		}
	}
	return status
}

func (flow Flow) ReadinessLabel() string {
	switch flow.ReadinessStatus() {
	case ReadinessBlocked:
		return "Needs setup"
	case ReadinessWatch:
		return "Review before publish"
	default:
		return "Ready to publish"
	}
}

func (flow Flow) Nodes() []FlowNode {
	flow = flow.Normalize()
	nodes := []FlowNode{
		{
			Key:     "placement",
			Label:   "Placement",
			Summary: flowPlacementReadinessSummary(flow),
			Kind:    "placement",
			Status:  flowPlacementReadinessStatus(flow),
		},
	}
	for _, step := range flow.Steps {
		status := ReadinessReady
		summary := strings.TrimSpace(step.Summary)
		if summary == "" {
			summary = flowStepSummary(step)
		}
		if !step.HasBlocks && step.BlockCount == 0 {
			status = ReadinessWatch
		}
		nodes = append(nodes, FlowNode{
			Key:     "step:" + workspaceToken(step.Key),
			Label:   step.Label,
			Summary: summary,
			Kind:    "step",
			Status:  status,
		})
	}
	for _, action := range flow.Actions {
		status := ReadinessReady
		summary := flowActionSummary(action)
		if !action.CanExecute || strings.TrimSpace(action.HandlerRef) == "" {
			status = ReadinessBlocked
		}
		nodes = append(nodes, FlowNode{
			Key:     "action:" + workspaceToken(action.Key),
			Label:   action.Label,
			Summary: summary,
			Kind:    "action",
			Status:  status,
		})
	}
	nodes = append(nodes, FlowNode{
		Key:     "publish",
		Label:   flow.ReadinessLabel(),
		Summary: "Publish only after the visible checks match the operator intent.",
		Kind:    "publish",
		Status:  flow.ReadinessStatus(),
	})
	return normalizeFlowNodes(nodes)
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

func ReadinessStatusLabel(status ReadinessStatus) string {
	switch normalizeReadinessStatus(status) {
	case ReadinessBlocked:
		return "Needs setup"
	case ReadinessWatch:
		return "Review"
	default:
		return "Ready"
	}
}

func CanvasActionKindLabel(kind CanvasActionKind) string {
	switch normalizeCanvasActionKind(kind) {
	case CanvasActionReveal:
		return "Reveal"
	case CanvasActionMoveUp:
		return "Move up"
	case CanvasActionMoveDown:
		return "Move down"
	case CanvasActionInlineText:
		return "Edit text"
	case CanvasActionContent:
		return "Content"
	case CanvasActionStyle:
		return "Style"
	case CanvasActionToggleVisibility:
		return "Hide"
	default:
		return "Action"
	}
}

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

func normalizeEngineKind(kind EngineKind) EngineKind {
	switch EngineKind(strings.TrimSpace(string(kind))) {
	case EngineSiteMap:
		return EngineSiteMap
	case EngineBlockLayout:
		return EngineBlockLayout
	case EngineFlow:
		return EngineFlow
	case EngineScene3D:
		return EngineScene3D
	default:
		return EngineCanvas
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

func normalizeCanvasActionKind(kind CanvasActionKind) CanvasActionKind {
	switch CanvasActionKind(strings.TrimSpace(string(kind))) {
	case CanvasActionMoveUp:
		return CanvasActionMoveUp
	case CanvasActionMoveDown:
		return CanvasActionMoveDown
	case CanvasActionInlineText:
		return CanvasActionInlineText
	case CanvasActionContent:
		return CanvasActionContent
	case CanvasActionStyle:
		return CanvasActionStyle
	case CanvasActionToggleVisibility:
		return CanvasActionToggleVisibility
	default:
		return CanvasActionReveal
	}
}

func normalizeReadinessStatus(status ReadinessStatus) ReadinessStatus {
	switch ReadinessStatus(strings.TrimSpace(string(status))) {
	case ReadinessWatch:
		return ReadinessWatch
	case ReadinessBlocked:
		return ReadinessBlocked
	default:
		return ReadinessReady
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

func normalizeCanvasViewports(values []CanvasViewport) []CanvasViewport {
	if len(values) == 0 {
		values = []CanvasViewport{
			{Key: "desktop", Label: "Desktop", Width: "100%", Active: true},
			{Key: "tablet", Label: "Tablet", Width: "48rem"},
			{Key: "mobile", Label: "Mobile", Width: "24rem"},
		}
	}
	out := make([]CanvasViewport, 0, len(values))
	activeSet := false
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Width = strings.TrimSpace(value.Width)
		if value.Key == "" || value.Label == "" {
			continue
		}
		if value.Active {
			if activeSet {
				value.Active = false
			} else {
				activeSet = true
			}
		}
		out = append(out, value)
	}
	if len(out) > 0 && !activeSet {
		out[0].Active = true
	}
	return out
}

func normalizeCanvasZoomLevels(values []CanvasZoomLevel) []CanvasZoomLevel {
	if len(values) == 0 {
		values = []CanvasZoomLevel{
			{Key: "fit", Label: "Fit", Active: true},
			{Key: "75", Label: "75%", Scale: 0.75},
			{Key: "100", Label: "100%", Scale: 1},
			{Key: "125", Label: "125%", Scale: 1.25},
		}
	}
	out := make([]CanvasZoomLevel, 0, len(values))
	activeSet := false
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		if value.Key == "" || value.Label == "" {
			continue
		}
		if value.Active {
			if activeSet {
				value.Active = false
			} else {
				activeSet = true
			}
		}
		out = append(out, value)
	}
	if len(out) > 0 && !activeSet {
		out[0].Active = true
	}
	return out
}

func normalizeCanvasBlocks(values []CanvasBlock) []CanvasBlock {
	out := make([]CanvasBlock, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Summary = strings.TrimSpace(value.Summary)
		value.GoSXComponent = strings.TrimSpace(value.GoSXComponent)
		value.Source = normalizeComponentSource(value.Source)
		value.Binding = strings.TrimSpace(value.Binding)
		value.Status = strings.TrimSpace(value.Status)
		value.Controls = normalizeControls(value.Controls)
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeCanvasActions(values []CanvasAction) []CanvasAction {
	if len(values) == 0 {
		values = DefaultCanvasActions()
	}
	out := make([]CanvasAction, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Summary = strings.TrimSpace(value.Summary)
		value.Kind = normalizeCanvasActionKind(value.Kind)
		if value.Key == "" {
			value.Key = string(value.Kind)
		}
		if value.Label == "" {
			value.Label = CanvasActionKindLabel(value.Kind)
		}
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeControls(values []Control) []Control {
	out := make([]Control, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Kind = normalizeControlKind(value.Kind)
		value.Binding = strings.TrimSpace(value.Binding)
		value.Value = strings.TrimSpace(value.Value)
		value.Help = strings.TrimSpace(value.Help)
		value.Options = normalizeControlOptions(value.Options)
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeRuntimeMethods(values []RuntimeMethod) []RuntimeMethod {
	out := make([]RuntimeMethod, 0, len(values))
	for _, value := range values {
		value.Name = strings.TrimSpace(value.Name)
		value.Label = strings.TrimSpace(value.Label)
		value.Summary = strings.TrimSpace(value.Summary)
		value.Payload = normalizeRuntimePayloadFields(value.Payload)
		if value.Name == "" {
			continue
		}
		if value.Label == "" {
			value.Label = value.Name
		}
		out = append(out, value)
	}
	return out
}

func normalizeRuntimePayloadFields(values []RuntimePayloadField) []RuntimePayloadField {
	out := make([]RuntimePayloadField, 0, len(values))
	for _, value := range values {
		value.Name = strings.TrimSpace(value.Name)
		value.Label = strings.TrimSpace(value.Label)
		value.Kind = normalizeControlKind(value.Kind)
		value.Summary = strings.TrimSpace(value.Summary)
		if value.Name == "" {
			continue
		}
		if value.Label == "" {
			value.Label = value.Name
		}
		out = append(out, value)
	}
	return out
}

func normalizeControlOptions(values []ControlOption) []ControlOption {
	out := make([]ControlOption, 0, len(values))
	for _, value := range values {
		value.Value = strings.TrimSpace(value.Value)
		value.Label = strings.TrimSpace(value.Label)
		if value.Value == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeFlowSteps(steps []FlowStep) []FlowStep {
	out := make([]FlowStep, 0, len(steps))
	for _, step := range steps {
		step.Key = strings.TrimSpace(step.Key)
		step.Label = strings.TrimSpace(step.Label)
		step.Summary = strings.TrimSpace(step.Summary)
		if step.Key == "" || step.Label == "" {
			continue
		}
		out = append(out, step)
	}
	return out
}

func normalizeFlowActions(actions []FlowAction) []FlowAction {
	out := make([]FlowAction, 0, len(actions))
	for _, action := range actions {
		action.Key = strings.TrimSpace(action.Key)
		action.Label = strings.TrimSpace(action.Label)
		action.HandlerRef = strings.TrimSpace(action.HandlerRef)
		action.Fields = normalizeFlowFields(action.Fields)
		if action.Key == "" || action.Label == "" {
			continue
		}
		out = append(out, action)
	}
	return out
}

func normalizeFlowFields(fields []FlowField) []FlowField {
	out := make([]FlowField, 0, len(fields))
	for _, field := range fields {
		field.Name = strings.TrimSpace(field.Name)
		field.Label = strings.TrimSpace(field.Label)
		field.Kind = normalizeControlKind(field.Kind)
		if field.Name == "" || field.Label == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func normalizeFlowNodes(nodes []FlowNode) []FlowNode {
	out := make([]FlowNode, 0, len(nodes))
	for _, node := range nodes {
		node.Key = strings.TrimSpace(node.Key)
		node.Label = strings.TrimSpace(node.Label)
		node.Summary = strings.TrimSpace(node.Summary)
		node.Kind = strings.TrimSpace(node.Kind)
		node.Status = normalizeReadinessStatus(node.Status)
		if node.Key == "" || node.Label == "" {
			continue
		}
		out = append(out, node)
	}
	return out
}

func flowHandlerReadinessStatus(flow Flow) ReadinessStatus {
	if flow.HandlerRef == "" {
		return ReadinessBlocked
	}
	if !flow.CanExecute {
		return ReadinessWatch
	}
	return ReadinessReady
}

func flowHandlerReadinessSummary(flow Flow) string {
	if flow.HandlerRef == "" {
		return "Connect the site action that receives this flow."
	}
	if !flow.CanExecute {
		return flow.HandlerRef + " is set; review secondary action handlers."
	}
	return flow.HandlerRef + " receives submissions."
}

func flowPlacementReadinessStatus(flow Flow) ReadinessStatus {
	if flow.HasRoute && flow.Route != "" && flow.HasEmbedTarget && flow.EmbedTarget != "" {
		return ReadinessReady
	}
	if (flow.HasRoute && flow.Route != "") || (flow.HasEmbedTarget && flow.EmbedTarget != "") {
		return ReadinessWatch
	}
	return ReadinessBlocked
}

func flowPlacementReadinessSummary(flow Flow) string {
	hasRoute := flow.HasRoute && flow.Route != ""
	hasEmbed := flow.HasEmbedTarget && flow.EmbedTarget != ""
	switch {
	case hasRoute && hasEmbed:
		return "Visible at " + flow.Route + " and embedded in " + flow.EmbedTarget + "."
	case hasRoute:
		return "Visible at " + flow.Route + "; choose an embed target if it also belongs inside a page."
	case hasEmbed:
		return "Embedded in " + flow.EmbedTarget + "; add a route if it needs a direct page."
	default:
		return "Choose where this flow appears on the site."
	}
}

func flowStructureReadinessStatus(flow Flow) ReadinessStatus {
	if len(flow.Steps) > 0 && len(flow.Actions) > 0 {
		return ReadinessReady
	}
	if len(flow.Steps) > 0 || len(flow.Actions) > 0 {
		return ReadinessWatch
	}
	return ReadinessBlocked
}

func flowStructureReadinessSummary(flow Flow) string {
	return countLabel(len(flow.Steps), "step", "steps") + " and " + countLabel(len(flow.Actions), "action", "actions") + "."
}

func flowFieldsReadinessStatus(flow Flow) ReadinessStatus {
	if flow.FieldCount > 0 {
		return ReadinessReady
	}
	return ReadinessWatch
}

func flowFieldsReadinessSummary(flow Flow) string {
	if flow.FieldCount == 0 {
		return "No fields are exposed to site operators."
	}
	return countLabel(flow.FieldCount, "field", "fields") + ", " + countLabel(flow.RequiredFieldCount, "required", "required") + "."
}

func flowStepSummary(step FlowStep) string {
	if step.BlockCount == 0 {
		return "No body blocks yet."
	}
	return countLabel(step.BlockCount, "body block", "body blocks") + "."
}

func flowActionSummary(action FlowAction) string {
	fieldCount := len(action.Fields)
	if fieldCount == 0 {
		return "No fields; handler " + firstNonEmpty(action.HandlerRef, "not connected") + "."
	}
	return countLabel(fieldCount, "field", "fields") + "; handler " + firstNonEmpty(action.HandlerRef, "not connected") + "."
}

func countLabel(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + plural
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
