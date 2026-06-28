package studio

import (
	"strings"

	"m31labs.dev/gosx-studio/blocklayoutruntime"
	"m31labs.dev/gosx-studio/brandruntime"
	"m31labs.dev/gosx-studio/fieldruntime"
	"m31labs.dev/gosx-studio/previewruntime"
	"m31labs.dev/gosx-studio/selectionruntime"
	"m31labs.dev/gosx-studio/styleruntime"
	"m31labs.dev/gosx-studio/workbenchruntime"
)

const (
	RuntimeRoot                      = "/_gosx/studio"
	StylesheetPath                   = RuntimeRoot + "/studio.css"
	EngineRuntimePath                = RuntimeRoot + "/studio-engines.js"
	WorkbenchRuntimePath             = RuntimeRoot + "/workbench-runtime.js"
	CommandRuntimePath               = RuntimeRoot + "/command-palette.js"
	StateRuntimePath                 = RuntimeRoot + "/state-runtime.js"
	PreviewSubscriberPath            = RuntimeRoot + "/preview-subscriber.js"
	CanvasSelectionBridgePath        = RuntimeRoot + "/canvas-selection-bridge.js"
	CanvasInlineEditPath             = RuntimeRoot + "/canvas-inline-edit.js"
	Canvas2DPainterPath              = RuntimeRoot + "/canvas2d-painter.js"
	CanvasWASMFreeClientPath         = RuntimeRoot + "/canvas-wasm-free-client.js"
	CanvasContextualPanelPath        = RuntimeRoot + "/canvas-contextual-panel.js"
	CanvasDefaultInlineInstallerPath = RuntimeRoot + "/canvas-default-inline-installer.js"

	CanvasEngineName      = "GoSXStudioCanvas"
	SiteMapEngineName     = "GoSXStudioSiteMap"
	FlowDesignerName      = "GoSXStudioFlowDesigner"
	BlockLayoutEngineName = "GoSXStudioBlockLayout"
	Showcase3DEngineName  = "GoSXStudioShowcase3D"
)

// ShellConfig is the reusable host-facing contract for mounting Studio chrome.
//
// Host apps configure labels, resources, actions, permissions, and feature
// flags. Studio owns the authoring surface vocabulary and normalizes missing
// pieces to editor-safe defaults.
type ShellConfig struct {
	Labels       HostLabels
	Canvas       CanvasConfig
	Modes        []ModeConfig
	Resources    []ResourceConfig
	Panels       []PanelConfig
	Engines      []EngineConfig
	Adapters     []ResourceAdapter
	SiteMap      SiteMap
	Actions      []ActionConfig
	Permissions  PermissionConfig
	FeatureFlags map[string]bool
}

type HostLabels struct {
	ProductName  string
	EditorTitle  string
	PreviewLabel string
	PublishLabel string
}

type CanvasConfig struct {
	PreviewShell CanvasPreviewShell
}

type ModeConfig struct {
	Key     string
	Label   string
	Summary string
}

type ResourceConfig struct {
	Key     string
	Label   string
	Kind    string
	Href    string
	Summary string
}

type PanelConfig struct {
	Key      string
	Label    string
	Mode     string
	Summary  string
	Advanced bool
}

type EngineConfig struct {
	Key          string
	Name         string
	MountID      string
	Capabilities []string
}

type ActionConfig struct {
	Key     string
	Label   string
	Href    string
	Method  string
	Primary bool
}

type PermissionConfig struct {
	CanEdit    bool
	CanPublish bool
	CanManage  bool
}

func DefaultShellConfig() ShellConfig {
	return ShellConfig{
		Labels: HostLabels{
			ProductName:  "Website editor",
			EditorTitle:  "Website editor",
			PreviewLabel: "Preview",
			PublishLabel: "Publish",
		},
		Canvas: CanvasConfig{
			PreviewShell: DefaultCanvasPreviewShell(),
		},
		Modes: []ModeConfig{
			{Key: "home", Label: "Home", Summary: "Page content and section layout."},
			{Key: "look", Label: "Look", Summary: "Theme, layout, color, and visual details."},
			{Key: "brand", Label: "Brand", Summary: "Logo, icon, and identity assets."},
			{Key: "publish", Label: "Publish", Summary: "Preview, schedule, checks, and release."},
			{Key: "advanced", Label: "Advanced", Summary: "Flows, tools, fields, schedule, and fonts."},
		},
		Panels: []PanelConfig{
			{Key: "site", Label: "Site", Mode: "home", Summary: "Navigate editable areas."},
			{Key: "inspector", Label: "Selection tools", Mode: "home", Summary: "Edit the selected section."},
			{Key: "style", Label: "Visual editor", Mode: "look", Summary: "Edit the site look."},
			{Key: "brand", Label: "Brand kit", Mode: "brand", Summary: "Edit public brand marks."},
			{Key: "publish", Label: "Release center", Mode: "publish", Summary: "Review and publish changes."},
			{Key: "advanced", Label: "Tool drawer", Mode: "advanced", Summary: "Operator and developer tools.", Advanced: true},
		},
		Engines: []EngineConfig{
			{Key: "canvas", Name: CanvasEngineName, MountID: "gosx-studio-canvas-engine", Capabilities: []string{"canvas", "pointer", "keyboard", "text-input", "animation"}},
			{Key: "site-map", Name: SiteMapEngineName, MountID: "gosx-studio-site-map-engine", Capabilities: []string{"canvas", "pointer", "keyboard", "text-input", "animation"}},
			{Key: "flow-designer", Name: FlowDesignerName, MountID: "gosx-studio-flow-engine", Capabilities: []string{"canvas", "pointer", "keyboard", "text-input", "animation"}},
			{Key: "block-layout", Name: BlockLayoutEngineName, MountID: "gosx-studio-block-layout-engine", Capabilities: []string{"pointer", "keyboard", "text-input", "animation"}},
			{Key: "showcase-3d", Name: Showcase3DEngineName, MountID: "gosx-studio-showcase-3d-engine", Capabilities: []string{"canvas", "webgl", "webgpu", "pointer", "keyboard", "animation"}},
		},
		Adapters: DefaultResourceAdapters(),
		Actions: []ActionConfig{
			{Key: "save", Label: "Save", Method: "POST", Primary: true},
			{Key: "publish", Label: "Publish", Method: "POST", Primary: true},
			{Key: "schedule", Label: "Schedule", Method: "POST"},
			{Key: "restore", Label: "Restore", Method: "POST"},
		},
		Permissions: PermissionConfig{CanEdit: true, CanPublish: true, CanManage: true},
		FeatureFlags: map[string]bool{
			"brand-media-picker": true,
			"publish-review":     true,
			// Phase 3 burn-down flags. Production-default ON after the
			// legacy JS bundles were deleted on 2026-05-27. The flags
			// remain in the registry so host apps that probe them via
			// FeatureEnabled continue to see a stable shape. See
			// ~/.hyphae/spaces/m31labs-gosx/plans/gosx-vm-unification-and-editor-bridge-burn-down.md
			// Phase 3 Section E.
			fieldruntime.FeatureFlagKey:       true,
			selectionruntime.FeatureFlagKey:   true,
			brandruntime.FeatureFlagKey:       true,
			blocklayoutruntime.FeatureFlagKey: true,
			styleruntime.FeatureFlagKey:       true,
			workbenchruntime.FeatureFlagKey:   true,
			previewruntime.FeatureFlagKey:     true,
		},
	}
}

func (config ShellConfig) Normalize() ShellConfig {
	defaults := DefaultShellConfig()
	out := config
	out.Labels.ProductName = firstNonEmpty(out.Labels.ProductName, defaults.Labels.ProductName)
	out.Labels.EditorTitle = firstNonEmpty(out.Labels.EditorTitle, defaults.Labels.EditorTitle)
	out.Labels.PreviewLabel = firstNonEmpty(out.Labels.PreviewLabel, defaults.Labels.PreviewLabel)
	out.Labels.PublishLabel = firstNonEmpty(out.Labels.PublishLabel, defaults.Labels.PublishLabel)
	out.Canvas = out.Canvas.Normalize()
	out.Modes = normalizeModes(out.Modes, defaults.Modes)
	out.Panels = normalizePanels(out.Panels, defaults.Panels)
	out.Engines = normalizeEngines(out.Engines, defaults.Engines)
	out.Adapters = normalizeResourceAdapters(out.Adapters, defaults.Adapters)
	out.SiteMap = out.SiteMap.Normalize()
	out.Actions = normalizeActions(out.Actions, defaults.Actions)
	out.Resources = normalizeResources(out.Resources)
	if out.Permissions == (PermissionConfig{}) {
		out.Permissions = defaults.Permissions
	}
	out.FeatureFlags = normalizeFeatureFlags(out.FeatureFlags, defaults.FeatureFlags)
	return out
}

func (config CanvasConfig) Normalize() CanvasConfig {
	config.PreviewShell = config.PreviewShell.Normalize()
	return config
}

func (config ShellConfig) Resource(key string) (ResourceConfig, bool) {
	key = strings.TrimSpace(key)
	for _, resource := range config.Normalize().Resources {
		if resource.Key == key {
			return resource, true
		}
	}
	return ResourceConfig{}, false
}

func (config ShellConfig) Engine(key string) (EngineConfig, bool) {
	key = strings.TrimSpace(key)
	for _, engine := range config.Normalize().Engines {
		if engine.Key == key {
			return engine, true
		}
	}
	return EngineConfig{}, false
}

func (config ShellConfig) Adapter(kind ResourceKind) (ResourceAdapter, bool) {
	return ResourceAdapterByKind(config.Normalize().Adapters, kind)
}

func (config ShellConfig) FeatureEnabled(key string) bool {
	return config.Normalize().FeatureFlags[strings.TrimSpace(key)]
}

func EngineHostView(engine EngineConfig, className string) map[string]any {
	engine.Key = strings.TrimSpace(engine.Key)
	engine.Name = strings.TrimSpace(engine.Name)
	engine.MountID = strings.TrimSpace(engine.MountID)
	engine.Capabilities = normalizeShellStringList(engine.Capabilities)
	out := map[string]any{
		"key":          engine.Key,
		"name":         engine.Name,
		"mountId":      engine.MountID,
		"capabilities": strings.Join(engine.Capabilities, " "),
	}
	if className = strings.TrimSpace(className); className != "" {
		out["class"] = className
	}
	return out
}

func normalizeModes(values []ModeConfig, defaults []ModeConfig) []ModeConfig {
	if len(values) == 0 {
		values = defaults
	}
	out := make([]ModeConfig, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Summary = strings.TrimSpace(value.Summary)
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeResources(values []ResourceConfig) []ResourceConfig {
	out := make([]ResourceConfig, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Kind = strings.TrimSpace(value.Kind)
		value.Href = strings.TrimSpace(value.Href)
		value.Summary = strings.TrimSpace(value.Summary)
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeResourceAdapters(values []ResourceAdapter, defaults []ResourceAdapter) []ResourceAdapter {
	if len(values) == 0 {
		values = defaults
	}
	out := make([]ResourceAdapter, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizePanels(values []PanelConfig, defaults []PanelConfig) []PanelConfig {
	if len(values) == 0 {
		values = defaults
	}
	out := make([]PanelConfig, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Mode = strings.TrimSpace(value.Mode)
		value.Summary = strings.TrimSpace(value.Summary)
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeEngines(values []EngineConfig, defaults []EngineConfig) []EngineConfig {
	if len(values) == 0 {
		values = defaults
	}
	out := make([]EngineConfig, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Name = strings.TrimSpace(value.Name)
		value.MountID = strings.TrimSpace(value.MountID)
		value.Capabilities = normalizeShellStringList(value.Capabilities)
		if value.Key == "" || value.Name == "" || value.MountID == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeActions(values []ActionConfig, defaults []ActionConfig) []ActionConfig {
	if len(values) == 0 {
		values = defaults
	}
	out := make([]ActionConfig, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Href = strings.TrimSpace(value.Href)
		value.Method = strings.ToUpper(strings.TrimSpace(value.Method))
		if value.Method == "" {
			value.Method = "POST"
		}
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeFeatureFlags(values map[string]bool, defaults map[string]bool) map[string]bool {
	out := make(map[string]bool, len(defaults)+len(values))
	for key, value := range defaults {
		if normalized := strings.TrimSpace(key); normalized != "" {
			out[normalized] = value
		}
	}
	for key, value := range values {
		if normalized := strings.TrimSpace(key); normalized != "" {
			out[normalized] = value
		}
	}
	return out
}

func normalizeShellStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if normalized := strings.TrimSpace(value); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}
