package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

func TestInteractionsPanelRendersRevealAndHoverFocusRows(t *testing.T) {
	reveal := core.Interaction{
		Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFadeUp,
		Target: core.CanvasIdentity{Route: "/", BlockKey: "hero"}, Once: true,
	}.Normalize()
	revealDiagnostics := core.InteractionDiagnostics([]core.Interaction{reveal})

	node := RenderInteractionsPanel(InteractionsPanelOptions{
		ID: "interactions", Action: "/authoring", CSRFToken: "token",
		PageKey: "home", PageRoute: "/", ComponentKey: "hero",
		Rows: []InteractionRow{
			InteractionRowFromInteraction(core.InteractionRevealOnScroll, reveal, revealDiagnostics[0]),
			InteractionRowFromInteraction(core.InteractionHoverFocusState, core.Interaction{}, core.InteractionDiagnostic{}),
		},
	})
	html := gosx.RenderHTML(node)

	for _, want := range []string{
		"data-studio-interactions-inspector=\"true\"",
		"data-studio-interaction-card=\"reveal-on-scroll\"",
		"data-studio-interaction-card=\"hover-focus-state\"",
		"data-studio-interaction-attached=\"true\"",
		"data-studio-interaction-attached=\"false\"",
		"gosx_studio_interaction_kind",
		"gosx_studio_interaction_effect",
		"gosx_studio_interaction_duration_ms",
		"gosx_studio_interaction_delay_ms",
		"gosx_studio_interaction_once",
		"Reveal once",
		"Reduced motion: content shows immediately, no animation plays.",
		"Remove", // remove button only for the attached reveal row
		"data-gosx-studio-durable-history=\"true\"",
		"data-studio-target-field=\"interactions.entry\"",
		"data-gosx-studio-operation-kind=\"set-interaction\"",
		"data-gosx-studio-operation-kind=\"remove-interaction\"",
		"data-gosx-studio-interaction-key=\"hero:reveal-on-scroll\"",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
}

// TestInteractionsPanelRowSharesOneDurableFormForAttachAndRemove proves the
// handoff-31 "one product path" design: attach (set-interaction) and remove
// share a SINGLE <form>/operationruntime instance per row, not two, so both
// actions read/advance the same cached target head instead of one racing a
// stale cursor cached by the other. It also proves a never-attached row still
// gets a real, non-blank Target.ControlKey (the deterministic
// "<componentKey>:<kind>" default core.Interaction.Normalize() derives),
// which authoring.OperationRequest.Validate() requires even for a first
// attach.
func TestInteractionsPanelRowSharesOneDurableFormForAttachAndRemove(t *testing.T) {
	node := RenderInteractionsPanel(InteractionsPanelOptions{
		Action: "/authoring", PageKey: "page:home", PageRoute: "/", ComponentKey: "home:hero",
		Rows: []InteractionRow{
			InteractionRowFromInteraction(core.InteractionHoverFocusState, core.Interaction{}, core.InteractionDiagnostic{}),
		},
	})
	html := gosx.RenderHTML(node)

	if got := strings.Count(html, "<form"); got != 1 {
		t.Fatalf("expected exactly one <form> for the row (shared attach/remove), got %d:\n%s", got, html)
	}
	if strings.Contains(html, "Remove") {
		t.Fatal("an unattached row must not render a Remove button")
	}
	if !strings.Contains(html, `data-gosx-studio-interaction-key="home:hero:hover-focus-state"`) {
		t.Fatalf("expected the deterministic default interaction key on an unattached row's button, got:\n%s", html)
	}
	if !strings.Contains(html, `data-gosx-studio-operation-kind="set-interaction"`) {
		t.Fatalf("expected the Add button to carry the set-interaction operation kind, got:\n%s", html)
	}
}

// TestInteractionsPanelAttachedRowRendersTargetHeadForOptimisticConcurrency
// proves an already-attached row's form carries the host-supplied durable
// target head (both as the form's data-studio-target-head attribute
// operationruntime seeds its cursor from, and as the redundant hidden
// gosx_studio_expected_target_head input a no-collaboration plain-POST
// fallback would submit) so a Save on an existing interaction after a fresh
// page load is not rejected as a false conflict against its own blank
// optimistic head.
func TestInteractionsPanelAttachedRowRendersTargetHeadForOptimisticConcurrency(t *testing.T) {
	reveal := core.Interaction{
		Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFadeUp,
		Target: core.CanvasIdentity{Route: "/", BlockKey: "hero"},
	}.Normalize()
	diagnostics := core.InteractionDiagnostics([]core.Interaction{reveal})
	row := InteractionRowFromInteraction(core.InteractionRevealOnScroll, reveal, diagnostics[0])
	row.TargetHead = "head-123"
	node := RenderInteractionsPanel(InteractionsPanelOptions{
		Action: "/authoring", ComponentKey: "hero",
		Rows: []InteractionRow{row},
	})
	html := gosx.RenderHTML(node)
	for _, want := range []string{
		`data-studio-target-head="head-123"`,
		`name="gosx_studio_expected_target_head" value="head-123"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
}

func TestInteractionsPanelSurfacesBlockedReasonForInvalidTarget(t *testing.T) {
	invalid := core.Interaction{Kind: core.InteractionHoverFocusState} // no target identity
	diagnostics := core.InteractionDiagnostics([]core.Interaction{invalid})

	node := RenderInteractionsPanel(InteractionsPanelOptions{
		Rows: []InteractionRow{InteractionRowFromInteraction(core.InteractionHoverFocusState, invalid, diagnostics[0])},
	})
	html := gosx.RenderHTML(node)
	if !strings.Contains(html, "data-studio-interaction-reason=\"true\"") {
		t.Fatalf("expected a blocked-reason element, got:\n%s", html)
	}
	if !strings.Contains(html, "data-studio-interaction-status=\"blocked\"") {
		t.Fatalf("expected the row to carry blocked status, got:\n%s", html)
	}
}

func TestInteractionsPanelHoverRowHasNoOnceControl(t *testing.T) {
	hover := core.Interaction{Kind: core.InteractionHoverFocusState, Target: core.CanvasIdentity{Route: "/", BlockKey: "cta"}}.Normalize()
	diagnostics := core.InteractionDiagnostics([]core.Interaction{hover})
	node := RenderInteractionsPanel(InteractionsPanelOptions{
		Rows: []InteractionRow{InteractionRowFromInteraction(core.InteractionHoverFocusState, hover, diagnostics[0])},
	})
	html := gosx.RenderHTML(node)
	if strings.Contains(html, "Reveal once") {
		t.Fatal("hover-focus-state row must not offer a scroll-only 'reveal once' control")
	}
}

// TestRenderInteractionsPanelCollapsesTargetIntoOneCompactRowWithDisclosure
// guards wave 3A fix (a): a target used to render as a full "Interactions"
// header plus every allow-listed kind's controls, always fully expanded. It
// must now be exactly one compact row (label + one-line summary) with the
// full edit controls tucked behind a closed-by-default <details>.
func TestRenderInteractionsPanelCollapsesTargetIntoOneCompactRowWithDisclosure(t *testing.T) {
	reveal := core.Interaction{
		Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFadeUp,
		Target: core.CanvasIdentity{Route: "/", BlockKey: "hero"},
	}.Normalize()
	diagnostics := core.InteractionDiagnostics([]core.Interaction{reveal})
	node := RenderInteractionsPanel(InteractionsPanelOptions{
		ComponentKey: "home:hero", ComponentLabel: "Hero",
		Rows: []InteractionRow{
			InteractionRowFromInteraction(core.InteractionRevealOnScroll, reveal, diagnostics[0]),
			InteractionRowFromInteraction(core.InteractionHoverFocusState, core.Interaction{}, core.InteractionDiagnostic{}),
		},
	})
	html := gosx.RenderHTML(node)
	for _, want := range []string{
		`class="studio-interactions-target"`,
		`data-studio-interaction-target-attached="true"`,
		`<strong data-studio-interaction-target-label="true">Hero</strong>`,
		`<span data-studio-interaction-target-summary="true">Reveal on scroll: Fade up</span>`,
		`<details class="studio-interactions-target__editor">`,
		`<summary>Edit interactions</summary>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
	// A "raw wall" is a single un-collapsible chunk of always-visible
	// controls. The disclosure that holds the actual Effect/Duration/Delay
	// rows must not carry `open` — a real browser collapses (0 visible
	// selects) until the operator expands it.
	if strings.Contains(html, `<details class="studio-interactions-target__editor" open`) {
		t.Fatal("target editor disclosure must be closed by default")
	}
	// This one target still has no attached "Interactions" header repeated
	// per kind — only one header for the whole card.
	if got := strings.Count(html, "Interactions"); got != 0 {
		t.Fatalf("a single target's compact card must not repeat a standalone \"Interactions\" header, found it %d times:\n%s", got, html)
	}
}

// TestRenderInteractionsPanelUnattachedTargetOffersAddDisclosureLabel guards
// the "+ Add interaction" affordance (punch #24) at the single-target level:
// a target with nothing attached must read as an explicit add affordance,
// not a pre-expanded "Save" control set.
//
// Wave 3B (tamarind, residual #6): the inner disclosure's own summary text
// no longer repeats the outer "+ Add interaction" CTA verbatim (see
// interactionsDisclosureLabel) — it names the target instead
// ("CTA button"), since RenderInteractionsInspector already wraps unattached
// cards in its own shared "+ Add interaction (N elements available)"
// picker, and a second, identically-worded nested toggle read as a
// meaningless repeat with no new information (confirmed live: expanding the
// picker then a target showed "+ ADD INTERACTION (N AVAILABLE)" immediately
// followed by "Hero" / "+ ADD INTERACTION" again).
func TestRenderInteractionsPanelUnattachedTargetOffersAddDisclosureLabel(t *testing.T) {
	node := RenderInteractionsPanel(InteractionsPanelOptions{
		ComponentKey: "home:cta", ComponentLabel: "CTA button",
		Rows: []InteractionRow{
			InteractionRowFromInteraction(core.InteractionHoverFocusState, core.Interaction{}, core.InteractionDiagnostic{}),
		},
	})
	html := gosx.RenderHTML(node)
	for _, want := range []string{
		`data-studio-interaction-target-attached="false"`,
		`<span data-studio-interaction-target-summary="true">Not attached</span>`,
		`<summary>CTA button</summary>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, "<summary>+ Add interaction</summary>") {
		t.Fatalf("the inner per-target disclosure must not repeat the outer picker's own CTA text:\n%s", html)
	}
}

// TestRenderInteractionsInspectorRendersOneHeaderListsAttachedAndTucksTheRest
// guards wave 3A fixes (a): the aggregate entry point a host should switch
// its per-target loop to. Given several targets, it must render the
// "Interactions" header exactly once (not once per target — the raw-wall
// defect), list every target with an attached interaction as a compact card,
// and hide every target with nothing attached behind ONE shared
// "+ Add interaction" picker instead of pre-expanding each one.
func TestRenderInteractionsInspectorRendersOneHeaderListsAttachedAndTucksTheRest(t *testing.T) {
	reveal := core.Interaction{
		Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFade,
		Target: core.CanvasIdentity{Route: "/", BlockKey: "hero"},
	}.Normalize()
	diagnostics := core.InteractionDiagnostics([]core.Interaction{reveal})
	attachedRow := InteractionRowFromInteraction(core.InteractionRevealOnScroll, reveal, diagnostics[0])
	unattachedRow := func() InteractionRow {
		return InteractionRowFromInteraction(core.InteractionHoverFocusState, core.Interaction{}, core.InteractionDiagnostic{})
	}

	node := RenderInteractionsInspector(InteractionsInspectorOptions{
		ID: "studio-interactions",
		Targets: []InteractionsPanelOptions{
			{ComponentKey: "home:hero", ComponentLabel: "Hero", Rows: []InteractionRow{attachedRow}},
			{ComponentKey: "home:cta", ComponentLabel: "CTA button", Rows: []InteractionRow{unattachedRow()}},
			{ComponentKey: "home:footer", ComponentLabel: "Footer", Rows: []InteractionRow{unattachedRow()}},
		},
	})
	html := gosx.RenderHTML(node)

	if got := strings.Count(html, "<h3>Interactions</h3>"); got != 1 {
		t.Fatalf("expected exactly one \"Interactions\" header for the whole inspector, got %d:\n%s", got, html)
	}
	if got := strings.Count(html, `data-studio-interactions-target="true"`); got != 3 {
		t.Fatalf("expected all 3 targets to render as cards (attached inline, unattached behind the picker), got %d:\n%s", got, html)
	}
	for _, want := range []string{
		`data-studio-interactions-list="true"`,
		`data-studio-interaction-target-label="true">Hero</strong>`,
		`data-studio-interactions-add="true"`,
		`<summary>+ Add interaction (2 elements available)</summary>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
	listStart := strings.Index(html, `data-studio-interactions-list="true"`)
	addStart := strings.Index(html, `data-studio-interactions-add="true"`)
	if listStart < 0 || addStart < 0 || listStart > addStart {
		t.Fatalf("expected the attached-target list before the add picker, got:\n%s", html)
	}
	listSection := html[listStart:addStart]
	for _, unattachedLabel := range []string{"CTA button", "Footer"} {
		if strings.Contains(listSection, unattachedLabel) {
			t.Fatalf("unattached target %q must be tucked behind the add picker, not listed inline:\n%s", unattachedLabel, listSection)
		}
	}
}

// TestRenderInteractionsInspectorAllUnattachedShowsEmptyStateAndPicker guards
// the other edge: when nothing on the page has an interaction attached yet,
// the inspector must say so plainly rather than render an empty list, while
// still offering the add picker.
func TestRenderInteractionsInspectorAllUnattachedShowsEmptyStateAndPicker(t *testing.T) {
	node := RenderInteractionsInspector(InteractionsInspectorOptions{
		Targets: []InteractionsPanelOptions{
			{ComponentKey: "home:hero", ComponentLabel: "Hero", Rows: []InteractionRow{
				InteractionRowFromInteraction(core.InteractionHoverFocusState, core.Interaction{}, core.InteractionDiagnostic{}),
			}},
		},
	})
	html := gosx.RenderHTML(node)
	if !strings.Contains(html, `<p class="empty">No elements have interactions attached yet.</p>`) {
		t.Fatalf("expected an explicit empty state, got:\n%s", html)
	}
	if !strings.Contains(html, `<summary>+ Add interaction (1 element available)</summary>`) {
		t.Fatalf("expected the singular add-picker label, got:\n%s", html)
	}
	if strings.Contains(html, `data-studio-interactions-list="true"`) {
		t.Fatal("must not render an empty attached-target list wrapper when nothing is attached")
	}
}
