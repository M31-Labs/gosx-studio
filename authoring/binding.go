package authoring

import (
	"context"
	"strings"

	"m31labs.dev/gosx-studio/core"
)

// This file is the executable half of resource binding: core.BindingResolver
// and core.SiteMap.BindingDiagnostics are pure/host-storage-free. Real hosts
// resolve a component's Binding against live CMS/media/flow state, which
// requires I/O — so the adapter contract that calls into host storage lives
// here (the same layer as AuthoringAdapter), while the shape and
// ready/blocked classification stay in core.

// ResourceBindingOption is one candidate a source picker can list for a given
// resource kind (e.g. one CMS collection, one media asset, one flow).
type ResourceBindingOption struct {
	Value   string
	Label   string
	Summary string
}

// ResourceBindingAdapter is the executable host boundary for one resource
// kind used by a component's Binding. Studio's source picker calls List to
// render browsable options and Validate to compute broken-binding
// diagnostics for readiness and publish review. Saving/selecting a binding
// flows through the ordinary set-field/save-control authoring mutation —
// Binding itself is just a string value on a Control — so this interface only
// covers the read-side operations a host cannot express as pure data.
type ResourceBindingAdapter interface {
	// Kind identifies which core.ResourceKind this adapter serves.
	Kind() core.ResourceKind
	// List returns the candidate bindings for this resource kind, optionally
	// filtered by a free-text query (an empty query returns everything the
	// host is willing to show).
	List(ctx context.Context, query string) ([]ResourceBindingOption, error)
	// Validate reports whether one already-chosen binding still resolves to
	// real content, and a short host-authored reason when it does not.
	Validate(ctx context.Context, binding string) (resolved bool, reason string, err error)
}

func (option ResourceBindingOption) Normalize() ResourceBindingOption {
	option.Value = strings.TrimSpace(option.Value)
	option.Label = strings.TrimSpace(option.Label)
	option.Summary = strings.TrimSpace(option.Summary)
	return option
}

// NormalizeResourceBindingOptions trims and drops any option with no Value.
func NormalizeResourceBindingOptions(options []ResourceBindingOption) []ResourceBindingOption {
	out := make([]ResourceBindingOption, 0, len(options))
	for _, option := range options {
		option = option.Normalize()
		if option.Value == "" {
			continue
		}
		if option.Label == "" {
			option.Label = option.Value
		}
		out = append(out, option)
	}
	return out
}

// ResourceBindingAdapters indexes a set of ResourceBindingAdapter values by
// their declared core.ResourceKind, so a resolver can route a component's
// Binding to the right adapter by Component.Source/binding prefix convention.
type ResourceBindingAdapters map[core.ResourceKind]ResourceBindingAdapter

// SiteMapBindingResolver adapts a ResourceBindingAdapters set into a
// core.BindingResolver for core.SiteMap.BindingDiagnostics. componentResourceKind
// maps one component to the core.ResourceKind whose adapter should validate
// its Binding (hosts typically switch on a binding-string prefix, e.g.
// "products." -> core.ResourceProducts, "flow." -> core.ResourceFlows).
// A component whose resource kind has no registered adapter, or whose
// Validate call errors, is reported unresolved with a diagnostic-friendly
// reason rather than panicking or silently passing.
func SiteMapBindingResolver(ctx context.Context, adapters ResourceBindingAdapters, componentResourceKind func(core.Component) core.ResourceKind) core.BindingResolver {
	return func(component core.Component) (bool, string) {
		if componentResourceKind == nil {
			return false, "No resource kind resolver is configured."
		}
		kind := componentResourceKind(component)
		adapter, ok := adapters[kind]
		if !ok || adapter == nil {
			return false, "No adapter is registered for this resource kind."
		}
		resolved, reason, err := adapter.Validate(ctx, component.Binding)
		if err != nil {
			return false, err.Error()
		}
		return resolved, reason
	}
}
