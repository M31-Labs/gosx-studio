package core

import "strings"

type Component struct {
	Key                 string
	TemplateKey         string
	Label               string
	Summary             string
	GoSXComponent       string
	Source              ComponentSource
	Binding             string
	Status              string
	Editable            bool
	Position            int
	CanReorder          bool
	CanDuplicate        bool
	Visible             bool
	CanToggleVisibility bool
	CanDelete           bool
	Controls            []Control
	// DefinitionKey, Overrides, and Detached implement the reusable
	// component/instance model (core/instances.go). A non-empty DefinitionKey
	// makes this placed component a shared instance: EffectiveComponentControls
	// resolves its rendered/editable controls from the matching
	// ComponentDefinition, layering any per-instance Overrides on top, unless
	// Detached is true (in which case Controls is this instance's own frozen
	// copy and it no longer follows the shared definition).
	DefinitionKey string
	Overrides     []ComponentOverride
	Detached      bool
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
	// Locked marks a dev-set control that the editor must never offer for
	// editing (e.g. a plugin checkout endpoint or API-key reference). Its
	// binding/value still travel with the template; editor control-view
	// builders simply exclude it from the editable set.
	Locked  bool `json:"locked,omitempty"`
	Options []ControlOption
}

type ControlOption struct {
	Value string
	Label string
}

type CompositionLibrary struct {
	PageBlueprints     []PageBlueprint
	ComponentTemplates []ComponentTemplate
	// ComponentDefinitions is the site-wide shared/reusable component library
	// (core/instances.go). Placed Component values reference one of these by
	// DefinitionKey to become a shared instance; PaletteEntries() surfaces both
	// ComponentTemplates and ComponentDefinitions as one placeable palette.
	ComponentDefinitions []ComponentDefinition
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

type WorkspaceCanvas struct {
	ViewBox string
	Nodes   []WorkspaceNodePoint
	Links   []WorkspaceLinkPath
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

type WorkspaceNodePoint struct {
	NodeKey  string
	LayerKey string
	X        float64
	Y        float64
}

type WorkspaceLinkPath struct {
	Key         string
	Kind        WorkspaceLinkKind
	FromNodeKey string
	ToNodeKey   string
	Path        string
	FromX       float64
	FromY       float64
	ToX         float64
	ToY         float64
}

func (siteMap SiteMap) CompositionWorkspace() CompositionWorkspace {
	siteMap = siteMap.Normalize()
	workspace := CompositionWorkspace{}
	resourceNodeKeys := map[string]string{}
	resourceNodeIndexes := map[string]int{}
	resourceNodePageKeys := map[string]string{}
	resourceNodeGroups := map[string]PageGroup{}

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
		pageNodeKey := "page:" + WorkspaceToken(pageKey)
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
			componentNodeKey := "component:" + WorkspaceToken(pageKey) + ":" + WorkspaceToken(componentKey)
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
				Key:         "contains:" + WorkspaceToken(pageKey) + ":" + WorkspaceToken(componentKey),
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
				resourceNodeKey = "resource:" + WorkspaceToken(binding)
				resourceNodeKeys[binding] = resourceNodeKey
				resourceNodeIndexes[binding] = len(workspace.Nodes)
				resourceNodePageKeys[binding] = pageKey
				resourceNodeGroups[binding] = page.NormalizedGroup()
				workspace.Nodes = append(workspace.Nodes, WorkspaceNode{
					Key:     resourceNodeKey,
					Label:   binding,
					Summary: ComponentSourceLabel(component.Source) + " binding",
					Kind:    WorkspaceNodeResource,
					PageKey: pageKey,
					Group:   page.NormalizedGroup(),
					Source:  component.NormalizedSource(),
					Binding: binding,
					Status:  ComponentSourceLabel(component.Source),
				})
			} else {
				if resourceNodePageKeys[binding] != pageKey {
					resourceNodePageKeys[binding] = ""
					resourceNodeGroups[binding] = PageGroupUtility
				}
				if index, ok := resourceNodeIndexes[binding]; ok && index >= 0 && index < len(workspace.Nodes) {
					workspace.Nodes[index].PageKey = resourceNodePageKeys[binding]
					workspace.Nodes[index].Group = resourceNodeGroups[binding]
				}
			}
			workspace.Links = append(workspace.Links, WorkspaceLink{
				Key:         "binds:" + WorkspaceToken(pageKey) + ":" + WorkspaceToken(componentKey) + ":" + WorkspaceToken(binding),
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

func (workspace CompositionWorkspace) CanvasLayout() WorkspaceCanvas {
	workspace = workspace.Normalize()
	canvas := WorkspaceCanvas{ViewBox: "0 0 100 100"}
	nodesByKey := map[string]WorkspaceNode{}
	for _, node := range workspace.Nodes {
		nodesByKey[node.Key] = node
	}

	layerCount := len(workspace.Layers)
	pointsByKey := map[string]WorkspaceNodePoint{}
	for layerIndex, layer := range workspace.Layers {
		nodeKeys := make([]string, 0, len(layer.NodeKeys))
		for _, nodeKey := range layer.NodeKeys {
			if _, ok := nodesByKey[nodeKey]; ok {
				nodeKeys = append(nodeKeys, nodeKey)
			}
		}
		nodeCount := len(nodeKeys)
		if nodeCount == 0 {
			continue
		}
		for nodeIndex, nodeKey := range nodeKeys {
			point := WorkspaceNodePoint{
				NodeKey:  nodeKey,
				LayerKey: layer.Key,
				X:        workspaceLayerX(layerIndex, layerCount),
				Y:        workspaceNodeY(nodeIndex, nodeCount),
			}
			canvas.Nodes = append(canvas.Nodes, point)
			pointsByKey[nodeKey] = point
		}
	}

	for _, link := range workspace.Links {
		from, ok := pointsByKey[link.FromNodeKey]
		if !ok {
			continue
		}
		to, ok := pointsByKey[link.ToNodeKey]
		if !ok {
			continue
		}
		canvas.Links = append(canvas.Links, WorkspaceLinkPath{
			Key:         link.Key,
			Kind:        link.NormalizedKind(),
			FromNodeKey: link.FromNodeKey,
			ToNodeKey:   link.ToNodeKey,
			Path:        workspaceLinkPath(from, to),
			FromX:       from.X,
			FromY:       from.Y,
			ToX:         to.X,
			ToY:         to.Y,
		})
	}

	return canvas
}

func (component Component) ControlCount() int {
	return len(component.Controls)
}

func (component Component) Normalize() Component {
	component.Key = strings.TrimSpace(component.Key)
	component.TemplateKey = strings.TrimSpace(component.TemplateKey)
	component.Label = strings.TrimSpace(component.Label)
	component.Summary = strings.TrimSpace(component.Summary)
	component.GoSXComponent = strings.TrimSpace(component.GoSXComponent)
	component.Source = normalizeComponentSource(component.Source)
	component.Binding = strings.TrimSpace(component.Binding)
	component.Status = strings.TrimSpace(component.Status)
	if component.Position < 0 {
		component.Position = 0
	}
	component.Controls = normalizeControls(component.Controls)
	component.DefinitionKey = strings.TrimSpace(component.DefinitionKey)
	component.Overrides = normalizeComponentOverrides(component.Overrides)
	return component
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

func workspaceLayerX(index int, count int) float64 {
	if count <= 1 {
		return 50
	}
	return 8 + (float64(index) * 84 / float64(count-1))
}

func workspaceNodeY(index int, count int) float64 {
	if count <= 1 {
		return 50
	}
	return 12 + (float64(index) * 76 / float64(count-1))
}

func workspaceLinkPath(from WorkspaceNodePoint, to WorkspaceNodePoint) string {
	delta := to.X - from.X
	if delta < 0 {
		delta = -delta
	}
	curve := delta * 0.5
	if curve < 8 {
		curve = 8
	}
	controlFromX := from.X + curve
	controlToX := to.X - curve
	if to.X == from.X {
		controlToX = to.X + curve
	} else if to.X < from.X {
		controlFromX = from.X - curve
		controlToX = to.X + curve
	}
	return "M " + WorkspaceCoord(from.X) + " " + WorkspaceCoord(from.Y) +
		" C " + WorkspaceCoord(controlFromX) + " " + WorkspaceCoord(from.Y) +
		", " + WorkspaceCoord(controlToX) + " " + WorkspaceCoord(to.Y) +
		", " + WorkspaceCoord(to.X) + " " + WorkspaceCoord(to.Y)
}

func (library CompositionLibrary) BlueprintCount() int {
	return len(library.PageBlueprints)
}

func (library CompositionLibrary) Normalize() CompositionLibrary {
	out := CompositionLibrary{
		PageBlueprints:       make([]PageBlueprint, 0, len(library.PageBlueprints)),
		ComponentTemplates:   make([]ComponentTemplate, 0, len(library.ComponentTemplates)),
		ComponentDefinitions: normalizeComponentDefinitions(library.ComponentDefinitions),
	}
	for _, blueprint := range library.PageBlueprints {
		blueprint = blueprint.Normalize()
		if blueprint.Key == "" || blueprint.Label == "" || blueprint.GoSXComponent == "" {
			continue
		}
		out.PageBlueprints = append(out.PageBlueprints, blueprint)
	}
	for _, template := range library.ComponentTemplates {
		template = template.Normalize()
		if template.Key == "" || template.Label == "" || template.GoSXComponent == "" {
			continue
		}
		out.ComponentTemplates = append(out.ComponentTemplates, template)
	}
	return out
}

// DefinitionCount returns the number of reusable shared component definitions
// in the library.
func (library CompositionLibrary) DefinitionCount() int {
	return len(library.ComponentDefinitions)
}

func (library CompositionLibrary) TemplateCount() int {
	return len(library.ComponentTemplates)
}

func (blueprint PageBlueprint) ComponentCount() int {
	return len(blueprint.Components)
}

func (blueprint PageBlueprint) Normalize() PageBlueprint {
	blueprint.Key = strings.TrimSpace(blueprint.Key)
	blueprint.Label = strings.TrimSpace(blueprint.Label)
	blueprint.Summary = strings.TrimSpace(blueprint.Summary)
	blueprint.RoutePattern = strings.TrimSpace(blueprint.RoutePattern)
	blueprint.Group = normalizePageGroup(blueprint.Group)
	blueprint.GoSXComponent = strings.TrimSpace(blueprint.GoSXComponent)
	blueprint.Status = strings.TrimSpace(blueprint.Status)
	blueprint.Components = normalizeComponentTemplates(blueprint.Components)
	return blueprint
}

func (blueprint PageBlueprint) GroupLabel() string {
	return PageGroupLabel(blueprint.Group)
}

func (blueprint PageBlueprint) NormalizedGroup() PageGroup {
	return normalizePageGroup(blueprint.Group)
}

func (template ComponentTemplate) ControlCount() int {
	return len(template.Controls)
}

func (template ComponentTemplate) Normalize() ComponentTemplate {
	template.Key = strings.TrimSpace(template.Key)
	template.Label = strings.TrimSpace(template.Label)
	template.Summary = strings.TrimSpace(template.Summary)
	template.Category = strings.TrimSpace(template.Category)
	template.GoSXComponent = strings.TrimSpace(template.GoSXComponent)
	template.Source = normalizeComponentSource(template.Source)
	template.DefaultBinding = strings.TrimSpace(template.DefaultBinding)
	template.Status = strings.TrimSpace(template.Status)
	template.AddLabel = strings.TrimSpace(template.AddLabel)
	template.Controls = normalizeControls(template.Controls)
	return template
}

func (template ComponentTemplate) SourceLabel() string {
	return ComponentSourceLabel(template.Source)
}

func (template ComponentTemplate) NormalizedSource() ComponentSource {
	return normalizeComponentSource(template.Source)
}

func (intent CompositionIntent) Normalize() CompositionIntent {
	intent.Key = strings.TrimSpace(intent.Key)
	intent.Label = strings.TrimSpace(intent.Label)
	intent.Summary = strings.TrimSpace(intent.Summary)
	intent.Kind = NormalizeCompositionIntentKind(intent.Kind)
	intent.TargetPageKey = strings.TrimSpace(intent.TargetPageKey)
	intent.TargetPageLabel = strings.TrimSpace(intent.TargetPageLabel)
	intent.TargetRoute = strings.TrimSpace(intent.TargetRoute)
	intent.TargetRegion = strings.TrimSpace(intent.TargetRegion)
	intent.PageBlueprintKey = strings.TrimSpace(intent.PageBlueprintKey)
	intent.PageBlueprintLabel = strings.TrimSpace(intent.PageBlueprintLabel)
	intent.ComponentTemplateKey = strings.TrimSpace(intent.ComponentTemplateKey)
	intent.ComponentLabel = strings.TrimSpace(intent.ComponentLabel)
	intent.GoSXComponent = strings.TrimSpace(intent.GoSXComponent)
	intent.Binding = strings.TrimSpace(intent.Binding)
	intent.Status = strings.TrimSpace(intent.Status)
	intent.Steps = NormalizeCompositionSteps(intent.Steps)
	return intent
}

func (intent CompositionIntent) StepCount() int {
	return len(intent.Steps)
}

func (intent CompositionIntent) NormalizedKind() CompositionIntentKind {
	return NormalizeCompositionIntentKind(intent.Kind)
}

func (control Control) NormalizedKind() ControlKind {
	return NormalizeControlKind(control.Kind)
}

func (control Control) Normalize() Control {
	control.Key = strings.TrimSpace(control.Key)
	control.Label = strings.TrimSpace(control.Label)
	control.Kind = NormalizeControlKind(control.Kind)
	control.Binding = strings.TrimSpace(control.Binding)
	control.Value = strings.TrimSpace(control.Value)
	control.Help = strings.TrimSpace(control.Help)
	control.Options = normalizeControlOptions(control.Options)
	return control
}

func (control Control) KindLabel() string {
	return ControlKindLabel(control.Kind)
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

func normalizeControls(values []Control) []Control {
	out := make([]Control, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Kind = NormalizeControlKind(value.Kind)
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

func normalizeComponents(values []Component) []Component {
	out := make([]Component, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.Key == "" || value.Label == "" || value.GoSXComponent == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeComponentTemplates(values []ComponentTemplate) []ComponentTemplate {
	out := make([]ComponentTemplate, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.Key == "" || value.Label == "" || value.GoSXComponent == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func NormalizeCompositionSteps(values []CompositionStep) []CompositionStep {
	out := make([]CompositionStep, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Summary = strings.TrimSpace(value.Summary)
		value.GoSXComponent = strings.TrimSpace(value.GoSXComponent)
		value.Binding = strings.TrimSpace(value.Binding)
		if value.Key == "" || value.Label == "" {
			continue
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

func NormalizeControlKind(kind ControlKind) ControlKind {
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

func NormalizeCompositionIntentKind(kind CompositionIntentKind) CompositionIntentKind {
	switch CompositionIntentKind(strings.TrimSpace(string(kind))) {
	case CompositionIntentCreatePage:
		return CompositionIntentCreatePage
	default:
		return CompositionIntentAddComponent
	}
}
