package core

import "strings"

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

func (adapter ResourceAdapter) NormalizedKind() ResourceKind {
	return normalizeResourceKind(adapter.Kind)
}

func (adapter ResourceAdapter) Normalize() ResourceAdapter {
	adapter.Kind = normalizeResourceKind(adapter.Kind)
	adapter.Label = strings.TrimSpace(adapter.Label)
	adapter.Summary = strings.TrimSpace(adapter.Summary)
	adapter.Surface = normalizeResourceSurface(adapter.Surface)
	adapter.Capabilities = normalizeResourceCapabilities(adapter.Capabilities)
	adapter.Bindings = normalizeResourceBindings(adapter.Bindings)
	if adapter.Label == "" {
		adapter.Label = ResourceKindLabel(adapter.Kind)
	}
	return adapter
}

func (adapter ResourceAdapter) CapabilityCount() int {
	return len(adapter.Normalize().Capabilities)
}

func (adapter ResourceAdapter) BindingCount() int {
	return len(adapter.Normalize().Bindings)
}

func normalizeResourceSurface(kind SurfaceKind) SurfaceKind {
	normalized := SurfaceKind(strings.TrimSpace(string(kind)))
	if normalized == "" {
		return SurfaceInspector
	}
	return normalizeSurfaceKind(normalized)
}

func normalizeResourceCapabilities(values []ResourceCapability) []ResourceCapability {
	out := make([]ResourceCapability, 0, len(values))
	for _, value := range values {
		if normalized := ResourceCapability(strings.TrimSpace(string(value))); normalized != "" {
			out = append(out, normalized)
		}
	}
	return out
}

func normalizeResourceBindings(values []ResourceBinding) []ResourceBinding {
	out := make([]ResourceBinding, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Summary = strings.TrimSpace(value.Summary)
		value.Binding = strings.TrimSpace(value.Binding)
		if value.Key == "" || value.Label == "" || value.Binding == "" {
			continue
		}
		out = append(out, value)
	}
	return out
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
