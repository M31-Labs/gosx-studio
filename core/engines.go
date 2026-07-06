package core

import "strings"

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
		value.Kind = NormalizeControlKind(value.Kind)
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
