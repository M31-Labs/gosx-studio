package engine

import (
	"strings"

	"m31labs.dev/gosx-studio/cms/studio"
	"m31labs.dev/gosx-studio/core"
)

// This file composes the engine's enforceable Gate/Blocker readiness
// mechanism (gates.go, S1) with the pure diagnostics cms/studio and core
// already compute (spec §4, S4): core.BindingDiagnostic (component resource
// bindings, including a flow's submit binding) and core.InteractionDiagnostic
// (the no-code interaction schema). A real HostLifecycle.Readiness
// implementation runs its own diagnostics and converts them with the
// BlockersFrom* helpers below before returning them to Engine -- that is
// what makes a broken binding or an invalid interaction hard-stop Publish
// server-side (Engine.Publish step 2) rather than merely change a button's
// advisory appearance.
//
// cms/studio/interaction_readiness.go already converts these same
// diagnostics straight to studio.PublishCheck for direct panel rendering;
// that path has no server-side teeth. BlockersFromBindingDiagnostics /
// BlockersFromInteractionDiagnostics are the enforcement-side twin: the
// Blocker.Blocking flag they set is exactly what Gate.Blocking() and
// Engine.Publish gate on.

// BlockersFromBindingDiagnostics converts core.BindingDiagnostic entries
// (SiteMap.BindingDiagnostics, or a flow's submit binding checked through
// core.Flow.BindingResolved) into engine.Blocker values a
// HostLifecycle.Readiness implementation can return. A resolved binding (or
// one with no binding at all) is not a finding and is skipped; an unresolved
// binding always blocks (core.BindingDiagnostic.Status is binary -- there is
// no advisory tier for a broken resource reference). hrefFor is optional
// (nil means no ResolveHref); it is host-specific because only the host
// knows its own editor route shape for "jump to this component."
func BlockersFromBindingDiagnostics(diagnostics []core.BindingDiagnostic, hrefFor func(core.BindingDiagnostic) string) []Blocker {
	out := make([]Blocker, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		diagnostic = diagnostic.Normalize()
		if diagnostic.Status() == core.ReadinessReady {
			continue
		}
		summary := "Broken binding: " + diagnostic.Binding
		detail := firstNonEmptyText(diagnostic.Reason, "This binding does not resolve to real content yet.")
		href := ""
		if hrefFor != nil {
			href = strings.TrimSpace(hrefFor(diagnostic))
		}
		out = append(out, Blocker{
			Key:         normalizeBlockerKey("binding", diagnostic.PageKey, diagnostic.ComponentKey),
			Scope:       "binding",
			Summary:     summary,
			Detail:      detail,
			ResolveHref: href,
			Blocking:    true,
		})
	}
	return out
}

// BlockersFromInteractionDiagnostics converts core.InteractionDiagnostic
// entries into engine.Blocker values. A valid interaction is not a finding
// and is skipped; an invalid one always blocks (core.InteractionDiagnostic's
// ReadinessStatus is binary: Ready or Blocked -- an interaction with no
// stable target or an unsupported kind/effect cannot render safely at all,
// never merely a Watch-level nudge). hrefFor is optional, same as above.
func BlockersFromInteractionDiagnostics(diagnostics []core.InteractionDiagnostic, hrefFor func(core.InteractionDiagnostic) string) []Blocker {
	out := make([]Blocker, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Status == core.ReadinessReady {
			continue
		}
		label := core.InteractionKindLabel(diagnostic.Kind)
		summary := label + " cannot render safely yet."
		detail := firstNonEmptyText(diagnostic.Reason, "Attach this interaction to an identified element.")
		key := diagnostic.Key
		if key == "" {
			key = string(diagnostic.Kind)
		}
		href := ""
		if hrefFor != nil {
			href = strings.TrimSpace(hrefFor(diagnostic))
		}
		out = append(out, Blocker{
			Key:         normalizeBlockerKey("interaction", key),
			Scope:       "interaction",
			Summary:     summary,
			Detail:      detail,
			ResolveHref: href,
			Blocking:    true,
		})
	}
	return out
}

// PublishChecksFromGate bridges a readiness Gate to cms/studio's
// presentation vocabulary (spec §4 "Bridge to presentation"): a Blocking
// blocker becomes studio.ReadinessNext (the must-fix-before-publish state,
// matching cms/studio/interaction_readiness.go's own Blocked->Next
// convention), an advisory (non-blocking) blocker becomes
// studio.ReadinessWatch. This is why the panel's "N/6 clear" count is
// authoritative: it is rendered from the exact same Blocker.Blocking flag
// that hard-stops Engine.Publish server-side, not a separate client-side
// judgment call.
//
// The spec sketches this as studio.PublishChecksFromGate; it lives here
// instead (engine importing cms/studio, "compose via imports") because
// cms/studio is out of bounds for this slice.
func PublishChecksFromGate(gate Gate) []studio.PublishCheck {
	checks := make([]studio.PublishCheck, 0, len(gate.Blockers))
	for _, blocker := range gate.Blockers {
		status := studio.ReadinessWatch
		if blocker.Blocking {
			status = studio.ReadinessNext
		}
		check := studio.NewPublishCheck(blocker.Key, blockerLabel(blocker), blocker.Scope, status, blocker.Summary, blocker.Detail)
		if blocker.ResolveHref != "" {
			check = check.WithHref(blocker.ResolveHref)
		}
		checks = append(checks, check)
	}
	return checks
}

func blockerLabel(blocker Blocker) string {
	label := strings.TrimSpace(blocker.Key)
	label = strings.NewReplacer(":", " ", "_", " ", "-", " ").Replace(label)
	label = strings.TrimSpace(label)
	if label == "" {
		return "Readiness check"
	}
	return label
}

func normalizeBlockerKey(parts ...string) string {
	nonEmpty := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return strings.Join(nonEmpty, ":")
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
