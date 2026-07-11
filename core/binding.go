package core

import "strings"

// BindingDiagnostic reports whether one placed component's resource Binding
// currently resolves to real host content, media, or a flow. Hosts compute
// Resolved by checking their own CMS/media/flow store through a
// BindingResolver; Studio owns the shape and the ready/blocked/watch
// classification consumed by site-map readiness and publish review. It never
// inspects host storage directly.
type BindingDiagnostic struct {
	PageKey        string
	PageLabel      string
	ComponentKey   string
	ComponentLabel string
	Binding        string
	Source         ComponentSource
	Resolved       bool
	Reason         string
}

// BindingResolver reports whether one component's Binding currently resolves
// to real host content. Hosts implement it against their own CMS/media/flow
// stores (e.g. "does products.collection have a published product?", "does
// flow.contact have a connected handler?"); Studio never inspects host
// storage directly. reason is a short, host-authored explanation surfaced
// when resolved is false (e.g. "No published products in this collection.").
type BindingResolver func(component Component) (resolved bool, reason string)

func (diagnostic BindingDiagnostic) Normalize() BindingDiagnostic {
	diagnostic.PageKey = strings.TrimSpace(diagnostic.PageKey)
	diagnostic.PageLabel = strings.TrimSpace(diagnostic.PageLabel)
	diagnostic.ComponentKey = strings.TrimSpace(diagnostic.ComponentKey)
	diagnostic.ComponentLabel = strings.TrimSpace(diagnostic.ComponentLabel)
	diagnostic.Binding = strings.TrimSpace(diagnostic.Binding)
	diagnostic.Source = normalizeComponentSource(diagnostic.Source)
	diagnostic.Reason = strings.TrimSpace(diagnostic.Reason)
	return diagnostic
}

// Status classifies one binding diagnostic for readiness/publish review: a
// component with no binding at all is not a broken binding (it simply has
// nothing to resolve), a resolved binding is Ready, and an unresolved one is
// Blocked — publish review should stop and surface Reason rather than let an
// operator publish a page with a dangling resource reference.
func (diagnostic BindingDiagnostic) Status() ReadinessStatus {
	diagnostic = diagnostic.Normalize()
	if diagnostic.Binding == "" {
		return ReadinessReady
	}
	if diagnostic.Resolved {
		return ReadinessReady
	}
	return ReadinessBlocked
}

// BindingDiagnostics walks every page/component with a non-empty Binding and
// asks resolve whether it still resolves to real content. Components with no
// binding are skipped entirely — they have nothing to validate.
func (siteMap SiteMap) BindingDiagnostics(resolve BindingResolver) []BindingDiagnostic {
	siteMap = siteMap.Normalize()
	if resolve == nil {
		return nil
	}
	out := []BindingDiagnostic{}
	for _, page := range siteMap.Pages {
		for _, component := range page.Components {
			binding := strings.TrimSpace(component.Binding)
			if binding == "" {
				continue
			}
			resolved, reason := resolve(component)
			out = append(out, BindingDiagnostic{
				PageKey:        page.Key,
				PageLabel:      page.Label,
				ComponentKey:   component.Key,
				ComponentLabel: component.Label,
				Binding:        binding,
				Source:         component.NormalizedSource(),
				Resolved:       resolved,
				Reason:         strings.TrimSpace(reason),
			}.Normalize())
		}
	}
	return out
}

// BrokenBindingDiagnostics filters BindingDiagnostics down to the entries
// publish review must block on.
func BrokenBindingDiagnostics(diagnostics []BindingDiagnostic) []BindingDiagnostic {
	out := make([]BindingDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Status() == ReadinessBlocked {
			out = append(out, diagnostic)
		}
	}
	return out
}
