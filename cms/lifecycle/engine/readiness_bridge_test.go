package engine

import (
	"context"
	"errors"
	"testing"

	"m31labs.dev/gosx-studio/cms/studio"
	"m31labs.dev/gosx-studio/core"
)

// TestBlockersFromBindingDiagnosticsClassifyBrokenVsResolved proves the
// core/binding.go composition: a resolved binding produces no Blocker at
// all, and a broken one always blocks (core.BindingDiagnostic.Status is
// binary), carrying the host-supplied resolution link.
func TestBlockersFromBindingDiagnosticsClassifyBrokenVsResolved(t *testing.T) {
	diagnostics := []core.BindingDiagnostic{
		{PageKey: "home", ComponentKey: "hero-image", Binding: "media.hero", Resolved: true},
		{PageKey: "home", ComponentKey: "cta-image", Binding: "media.missing", Resolved: false, Reason: "Asset was deleted."},
	}
	blockers := BlockersFromBindingDiagnostics(diagnostics, func(d core.BindingDiagnostic) string {
		return "/studio/pages/" + d.PageKey + "/components/" + d.ComponentKey
	})
	if len(blockers) != 1 {
		t.Fatalf("expected exactly one blocker (resolved binding produces none), got %d: %#v", len(blockers), blockers)
	}
	blocker := blockers[0]
	if !blocker.Blocking {
		t.Fatal("expected a broken binding to be Blocking")
	}
	if blocker.ResolveHref != "/studio/pages/home/components/cta-image" {
		t.Fatalf("expected resolution href to be surfaced, got %q", blocker.ResolveHref)
	}
	if blocker.Detail != "Asset was deleted." {
		t.Fatalf("expected the host-authored reason as detail, got %q", blocker.Detail)
	}
	if blocker.Scope != "binding" {
		t.Fatalf("expected scope binding, got %q", blocker.Scope)
	}
}

// TestBlockersFromInteractionDiagnosticsClassifyInvalidVsValid mirrors the
// binding test for core.InteractionDiagnostic (cms/studio/interaction_readiness.go's
// sibling source).
func TestBlockersFromInteractionDiagnosticsClassifyInvalidVsValid(t *testing.T) {
	diagnostics := []core.InteractionDiagnostic{
		{Key: "reveal-1", Kind: core.InteractionRevealOnScroll, Status: core.ReadinessReady},
		{Key: "hover-1", Kind: core.InteractionHoverFocusState, Status: core.ReadinessBlocked, Reason: "No stable target."},
	}
	blockers := BlockersFromInteractionDiagnostics(diagnostics, func(d core.InteractionDiagnostic) string {
		return "/studio/interactions/" + d.Key
	})
	if len(blockers) != 1 {
		t.Fatalf("expected exactly one blocker (valid interaction produces none), got %d: %#v", len(blockers), blockers)
	}
	blocker := blockers[0]
	if !blocker.Blocking {
		t.Fatal("expected an invalid interaction to be Blocking")
	}
	if blocker.ResolveHref != "/studio/interactions/hover-1" {
		t.Fatalf("expected resolution href to be surfaced, got %q", blocker.ResolveHref)
	}
	if blocker.Detail != "No stable target." {
		t.Fatalf("expected the reason as detail, got %q", blocker.Detail)
	}
	if blocker.Scope != "interaction" {
		t.Fatalf("expected scope interaction, got %q", blocker.Scope)
	}
}

// TestGateBlockingVsAdvisoryClassification is the core boundary test: a gate
// mixing one blocking finding with one advisory (non-blocking) finding must
// report Blocking()==true (any blocking entry hard-stops), and
// PublishChecksFromGate must map the two onto distinct presentation
// statuses -- ReadinessNext for the blocking one, ReadinessWatch for the
// advisory one -- so the panel's clear/watch/next count is driven by the
// exact same flag the server gates on.
func TestGateBlockingVsAdvisoryClassification(t *testing.T) {
	gate := Gate{Blockers: []Blocker{
		{Key: "asset-optimize", Scope: "asset", Summary: "Large hero image", Blocking: false},
		{Key: "binding:home:cta-image", Scope: "binding", Summary: "Broken binding", ResolveHref: "/studio/fix", Blocking: true},
	}}
	if !gate.Blocking() {
		t.Fatal("expected a gate with one blocking finding to report Blocking()==true")
	}

	checks := PublishChecksFromGate(gate)
	if len(checks) != 2 {
		t.Fatalf("expected two checks, got %d", len(checks))
	}
	byKey := map[string]studio.PublishCheck{}
	for _, check := range checks {
		byKey[check.Key] = check
	}
	advisory, ok := byKey["asset-optimize"]
	if !ok {
		t.Fatalf("expected advisory check to survive the bridge, got %#v", checks)
	}
	if advisory.Status != studio.ReadinessWatch {
		t.Fatalf("expected advisory finding to map to ReadinessWatch, got %q", advisory.Status)
	}
	blocking, ok := byKey["binding:home:cta-image"]
	if !ok {
		t.Fatalf("expected blocking check to survive the bridge, got %#v", checks)
	}
	if blocking.Status != studio.ReadinessNext {
		t.Fatalf("expected blocking finding to map to ReadinessNext, got %q", blocking.Status)
	}
	if blocking.Href != "/studio/fix" {
		t.Fatalf("expected resolution href to carry through to the presentation check, got %q", blocking.Href)
	}
}

// TestPublishBlockedByBindingDiagnosticSurfacesResolutionLink is the
// end-to-end server-side enforcement test: a host's Readiness() reports a
// Blocker built from a real core.BindingDiagnostic (via
// BlockersFromBindingDiagnostics), and Engine.Publish must deny the publish
// -- never merely mark a button disabled -- while the returned error still
// carries the actionable ResolveHref so the caller can render "fix this."
func TestPublishBlockedByBindingDiagnosticSurfacesResolutionLink(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})

	diagnostics := []core.BindingDiagnostic{
		{PageKey: "home", ComponentKey: "cta-image", Binding: "media.missing", Resolved: false, Reason: "Asset was deleted."},
	}
	blockers := BlockersFromBindingDiagnostics(diagnostics, func(d core.BindingDiagnostic) string {
		return "/studio/pages/" + d.PageKey + "/components/" + d.ComponentKey
	})
	host.setBlockers(ref, blockers...)

	_, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: publisherActor("actor-1"), OperationID: "op-blocked-binding"})
	var readinessErr ErrReadiness
	if !errors.As(err, &readinessErr) {
		t.Fatalf("expected ErrReadiness, got %v", err)
	}
	if !readinessErr.Gate.Blocking() {
		t.Fatal("expected the gate to report blocking")
	}
	var href string
	for _, blocker := range readinessErr.Gate.Blockers {
		if blocker.Blocking {
			href = blocker.ResolveHref
		}
	}
	if href != "/studio/pages/home/components/cta-image" {
		t.Fatalf("expected the blocking finding's resolution link to be surfaced, got %q", href)
	}
	if host.applyPublishCalls != 0 {
		t.Fatalf("expected no host write when a binding diagnostic blocks publish, got %d calls", host.applyPublishCalls)
	}
	if points := restorePointRevisions(t, e, ref); len(points) != 0 {
		t.Fatalf("expected no restore point minted when publish is blocked, got %d", len(points))
	}
}

// TestPublishProceedsWithAdvisoryOnlyFindings proves the flip side: when
// every reported finding is advisory (Blocking:false), Publish must proceed
// normally -- the button/count may show "watch," but the server does not
// deny the write.
func TestPublishProceedsWithAdvisoryOnlyFindings(t *testing.T) {
	e, host := newTestEngine(t)
	ctx := context.Background()
	ref := testTarget()
	host.setLive(ref, map[string]string{"title": "Original"})
	host.setDraft(ref, map[string]string{"title": "Draft V1"})
	host.setBlockers(ref,
		Blocker{Key: "asset-optimize", Scope: "asset", Summary: "Large hero image", Blocking: false},
		Blocker{Key: "flow-slow", Scope: "interaction", Summary: "Flow calls a slow endpoint", Blocking: false},
	)

	result, err := e.Publish(ctx, PublishCommand{Target: ref, Actor: publisherActor("actor-1"), OperationID: "op-advisory-only"})
	if err != nil {
		t.Fatalf("expected publish to proceed with advisory-only findings, got %v", err)
	}
	if result.RevisionID == "" || result.RestorePointID == "" {
		t.Fatalf("expected a real publish result, got %#v", result)
	}
	if host.applyPublishCalls != 1 {
		t.Fatalf("expected exactly one host write, got %d", host.applyPublishCalls)
	}
	if points := restorePointRevisions(t, e, ref); len(points) != 1 {
		t.Fatalf("expected a restore point minted for the successful publish, got %d", len(points))
	}
}
