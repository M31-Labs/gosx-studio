package studio

import "m31labs.dev/gosx-studio/core"

// This file composes core's pure diagnostics — core.BindingDiagnostic (for
// component resource bindings, including a flow's submit binding) and
// core.InteractionDiagnostic (for the no-code interaction schema) — into
// PublishCheck entries. cms/studio's readiness/publish-review vocabulary has
// no "blocked" status of its own (Ready/Watch/Next), so an unresolved binding
// or an invalid interaction target maps to ReadinessNext: the highest-priority
// state PublishReview already treats as a must-fix-before-publish blocker
// (see derivedPublishStatus/highestReadinessStatus in publish_review.go).

// BindingPublishChecks converts binding diagnostics (core.SiteMap.BindingDiagnostics,
// or a flow's submit binding checked directly through core.Flow.BindingResolved)
// into publish-review checks. A resolved binding is Ready; an unresolved one
// blocks publish (ReadinessNext) with its host-authored reason surfaced as the
// check detail.
func BindingPublishChecks(diagnostics []core.BindingDiagnostic) []PublishCheck {
	checks := make([]PublishCheck, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		diagnostic = diagnostic.Normalize()
		status := publishStatusFromCore(diagnostic.Status())
		summary := diagnostic.Binding + " is connected."
		detail := ""
		if status != ReadinessReady {
			summary = "Broken binding: " + diagnostic.Binding
			detail = firstNonEmpty(diagnostic.Reason, "This binding does not resolve to real content yet.")
		}
		checks = append(checks, NewPublishCheck(
			normalizeKey("binding:"+diagnostic.PageKey+":"+diagnostic.ComponentKey),
			firstNonEmpty(diagnostic.ComponentLabel, diagnostic.ComponentKey, "Binding"),
			firstNonEmpty(diagnostic.PageLabel, diagnostic.PageKey, "Site"),
			status, summary, detail,
		))
	}
	return checks
}

// InteractionPublishChecks converts core.InteractionDiagnostics into
// publish-review checks. An interaction with no stable target or an
// unsupported kind/effect blocks publish (ReadinessNext), because it cannot
// render safely at all — never merely a Watch-level nudge.
func InteractionPublishChecks(diagnostics []core.InteractionDiagnostic) []PublishCheck {
	checks := make([]PublishCheck, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		status := publishStatusFromCore(diagnostic.Status)
		label := core.InteractionKindLabel(diagnostic.Kind)
		summary := label + " is ready."
		detail := ""
		if status != ReadinessReady {
			summary = label + " cannot render safely yet."
			detail = firstNonEmpty(diagnostic.Reason, "Attach this interaction to an identified element.")
		}
		key := diagnostic.Key
		if key == "" {
			key = string(diagnostic.Kind)
		}
		checks = append(checks, NewPublishCheck(
			normalizeKey("interaction:"+key),
			label,
			"Interaction",
			status, summary, detail,
		))
	}
	return checks
}

// publishStatusFromCore maps core.ReadinessStatus (Ready/Watch/Blocked) onto
// the cms/studio readiness vocabulary (Ready/Watch/Next). Blocked always
// becomes Next: the highest-priority, must-fix-before-publish state.
func publishStatusFromCore(status core.ReadinessStatus) ReadinessStatus {
	switch status {
	case core.ReadinessBlocked:
		return ReadinessNext
	case core.ReadinessWatch:
		return ReadinessWatch
	default:
		return ReadinessReady
	}
}
