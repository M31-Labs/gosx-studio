package core

import (
	"strconv"
	"strings"
)

// Interactions are the no-code allow-listed behavior primitives an operator
// can attach to an identified canvas object (core.CanvasIdentity). Every
// interaction is a typed struct — never a script — so the server can validate
// it completely before it is ever rendered or persisted. Two kinds ship in
// this slice: reveal-on-scroll (an entrance animation driven by an
// IntersectionObserver on the published page) and hover-focus-state (a purely
// CSS-driven hover treatment that always ships a matching :focus-visible
// rule, because a mouse-only affordance would otherwise be invisible to
// keyboard users). Scroll-behavior and page-transition primitives are
// intentionally out of scope for this iteration.
type InteractionKind string

const (
	InteractionRevealOnScroll  InteractionKind = "reveal-on-scroll"
	InteractionHoverFocusState InteractionKind = "hover-focus-state"
)

// InteractionTrigger names what activates an interaction. Each InteractionKind
// has exactly one legal trigger (see InteractionTriggerFor), so an author can
// never persist a mismatched kind/trigger pair.
type InteractionTrigger string

const (
	InteractionTriggerScroll InteractionTrigger = "scroll"
	InteractionTriggerHover  InteractionTrigger = "hover"
)

// InteractionEffect is the allow-listed visual treatment for one interaction.
// Every effect is expressed with opacity/transform CSS custom properties so
// it can never remove a focusable element from the accessibility tree or the
// tab order — only visibility/display do that, and no effect here uses them.
type InteractionEffect string

const (
	InteractionEffectFade      InteractionEffect = "fade"
	InteractionEffectFadeUp    InteractionEffect = "fade-up"
	InteractionEffectFadeLeft  InteractionEffect = "fade-left"
	InteractionEffectFadeRight InteractionEffect = "fade-right"
	InteractionEffectLift      InteractionEffect = "lift"
	InteractionEffectGlow      InteractionEffect = "glow"
	InteractionEffectUnderline InteractionEffect = "underline"
	InteractionEffectScale     InteractionEffect = "scale"
)

// ReducedMotionMode names how an interaction behaves under
// prefers-reduced-motion. Studio ships exactly one mode today —
// ReducedMotionImmediate — and Interaction.Normalize always pins it there: the
// reduced-motion fallback is a platform guarantee, not an operator choice.
// The field stays on the struct (rather than being an unconditional global)
// so the runtime and any future modes stay auditable per interaction.
type ReducedMotionMode string

const (
	ReducedMotionImmediate ReducedMotionMode = "immediate"
)

// Interaction is the durable, typed shape of one attached behavior. Target
// reuses CanvasIdentity: the same stable-identity contract selection and
// collaboration already key off of, so an interaction can never attach to an
// ambiguous or unstable DOM address.
type Interaction struct {
	Key           string
	Kind          InteractionKind
	Trigger       InteractionTrigger
	Target        CanvasIdentity
	Effect        InteractionEffect
	DurationMS    int
	DelayMS       int
	Once          bool
	ReducedMotion ReducedMotionMode
}

// AllowedInteractionKinds is the strict allow-list. Nothing outside this set
// can ever normalize to a non-empty Kind.
func AllowedInteractionKinds() []InteractionKind {
	return []InteractionKind{InteractionRevealOnScroll, InteractionHoverFocusState}
}

// InteractionTriggerFor returns the one legal trigger for kind, or "" when
// kind is not in the allow-list.
func InteractionTriggerFor(kind InteractionKind) InteractionTrigger {
	switch normalizeInteractionKind(kind) {
	case InteractionRevealOnScroll:
		return InteractionTriggerScroll
	case InteractionHoverFocusState:
		return InteractionTriggerHover
	default:
		return ""
	}
}

// InteractionEffectOptions returns the allow-listed effects for kind, in
// display order.
func InteractionEffectOptions(kind InteractionKind) []InteractionEffect {
	switch normalizeInteractionKind(kind) {
	case InteractionRevealOnScroll:
		return []InteractionEffect{InteractionEffectFade, InteractionEffectFadeUp, InteractionEffectFadeLeft, InteractionEffectFadeRight}
	case InteractionHoverFocusState:
		return []InteractionEffect{InteractionEffectLift, InteractionEffectGlow, InteractionEffectUnderline, InteractionEffectScale}
	default:
		return nil
	}
}

// InteractionDurationOptionsMS is the allow-listed animation duration set (ms).
func InteractionDurationOptionsMS() []int { return []int{150, 250, 400, 600} }

// InteractionDelayOptionsMS is the allow-listed animation delay set (ms).
func InteractionDelayOptionsMS() []int { return []int{0, 100, 200, 300} }

// DefaultInteractionDurationMS/DefaultInteractionDelayMS name the safe
// defaults a fresh interaction normalizes to when no duration/delay is given.
const (
	DefaultInteractionDurationMS = 250
	DefaultInteractionDelayMS    = 0
)

func (interaction Interaction) Normalize() Interaction {
	interaction.Key = strings.TrimSpace(interaction.Key)
	interaction.Kind = normalizeInteractionKind(interaction.Kind)
	interaction.Target = interaction.Target.Normalize()
	interaction.Trigger = InteractionTriggerFor(interaction.Kind)
	interaction.Effect = normalizeInteractionEffect(interaction.Kind, interaction.Effect)
	interaction.DurationMS = normalizeInteractionDurationMS(interaction.DurationMS)
	// Reduced motion is a platform guarantee, never an authored choice: every
	// interaction collapses to "content is immediately visible/reachable, no
	// animation plays" under prefers-reduced-motion, regardless of Kind/Effect.
	interaction.ReducedMotion = ReducedMotionImmediate
	if interaction.Key == "" && interaction.Kind != "" {
		interaction.Key = defaultInteractionKey(interaction.Target, interaction.Kind)
	}
	return interaction
}

// defaultInteractionKey derives a stable key for the common case of at most
// one interaction of a given kind per attached target, so authoring a reveal
// or hover state never requires the operator to invent an identifier.
func defaultInteractionKey(target CanvasIdentity, kind InteractionKind) string {
	target = target.Normalize()
	locator := firstNonEmptyValue(target.Field, target.BlockKey, target.NodeID, target.PageID)
	if locator == "" {
		return ""
	}
	return locator + ":" + string(kind)
}

func firstNonEmptyValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// Validate reports field-keyed problems that must block persistence. An
// invalid Kind short-circuits (there is nothing meaningful left to check),
// everything else accumulates so the caller can surface every problem at once.
func (interaction Interaction) Validate() map[string]string {
	interaction = interaction.Normalize()
	errs := map[string]string{}
	if interaction.Key == "" {
		errs["key"] = "This interaction needs a key."
	}
	if !interactionKindAllowed(interaction.Kind) {
		errs["kind"] = "Choose a supported interaction."
		return errs
	}
	if !interaction.Target.Stable() {
		errs["target"] = "Attach this interaction to an identified element."
	}
	if !interactionEffectAllowed(interaction.Kind, interaction.Effect) {
		errs["effect"] = "Choose a supported effect for this interaction."
	}
	if !interactionDurationAllowed(interaction.DurationMS) {
		errs["durationMS"] = "Choose a supported duration."
	}
	if !interactionDelayAllowed(interaction.DelayMS) {
		errs["delayMS"] = "Choose a supported delay."
	}
	return errs
}

// Valid reports whether Validate found no problems.
func (interaction Interaction) Valid() bool {
	return len(interaction.Validate()) == 0
}

// ReadinessStatus classifies one interaction for readiness/publish review: a
// valid interaction is Ready, anything that fails Validate is Blocked (never
// merely Watch) because an interaction with no stable target or an
// unsupported kind/effect cannot render safely at all.
func (interaction Interaction) ReadinessStatus() ReadinessStatus {
	if interaction.Valid() {
		return ReadinessReady
	}
	return ReadinessBlocked
}

// InteractionDiagnostic reports one interaction's readiness classification,
// mirroring the shape of BindingDiagnostic. Unlike a binding, interaction
// validity never requires host I/O (identity stability and the allow-lists
// are pure data), so there is no resolver indirection here.
type InteractionDiagnostic struct {
	Key    string
	Kind   InteractionKind
	Target CanvasIdentity
	Status ReadinessStatus
	Reason string
}

// InteractionDiagnostics validates every interaction and reports its
// classification and — for anything not Ready — an actionable reason.
func InteractionDiagnostics(interactions []Interaction) []InteractionDiagnostic {
	out := make([]InteractionDiagnostic, 0, len(interactions))
	for _, interaction := range interactions {
		interaction = interaction.Normalize()
		out = append(out, InteractionDiagnostic{
			Key:    interaction.Key,
			Kind:   interaction.Kind,
			Target: interaction.Target,
			Status: interaction.ReadinessStatus(),
			Reason: firstInteractionProblem(interaction.Validate()),
		}.normalize())
	}
	return out
}

// BlockedInteractionDiagnostics filters InteractionDiagnostics down to the
// entries publish review must block on.
func BlockedInteractionDiagnostics(diagnostics []InteractionDiagnostic) []InteractionDiagnostic {
	out := make([]InteractionDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Status == ReadinessBlocked {
			out = append(out, diagnostic)
		}
	}
	return out
}

func (diagnostic InteractionDiagnostic) normalize() InteractionDiagnostic {
	diagnostic.Key = strings.TrimSpace(diagnostic.Key)
	diagnostic.Kind = normalizeInteractionKind(diagnostic.Kind)
	diagnostic.Target = diagnostic.Target.Normalize()
	diagnostic.Status = normalizeReadinessStatus(diagnostic.Status)
	diagnostic.Reason = strings.TrimSpace(diagnostic.Reason)
	return diagnostic
}

// RenderAttrs returns the canonical data-attribute map a host's page
// rendering should spread onto the identified target element so the
// published/preview runtime (a JS IntersectionObserver for reveal-on-scroll)
// and studio.css (pure CSS for hover-focus-state) recognize and animate it.
// Attribute names are stable and dash-cased for direct HTML use. An invalid
// interaction renders nothing — the host must never mark up something Studio
// could not validate.
func (interaction Interaction) RenderAttrs() map[string]string {
	interaction = interaction.Normalize()
	if !interaction.Valid() {
		return nil
	}
	return map[string]string{
		"data-gosx-interaction":             string(interaction.Kind),
		"data-gosx-interaction-effect":      string(interaction.Effect),
		"data-gosx-interaction-duration-ms": strconv.Itoa(interaction.DurationMS),
		"data-gosx-interaction-delay-ms":    strconv.Itoa(interaction.DelayMS),
		"data-gosx-interaction-once":        strconv.FormatBool(interaction.Once),
	}
}

// InteractionKindLabel/InteractionEffectLabel are the human-facing names the
// interactions inspector and readiness surfaces render.
func InteractionKindLabel(kind InteractionKind) string {
	switch normalizeInteractionKind(kind) {
	case InteractionRevealOnScroll:
		return "Reveal on scroll"
	case InteractionHoverFocusState:
		return "Hover & focus state"
	default:
		return "Unsupported interaction"
	}
}

func InteractionEffectLabel(effect InteractionEffect) string {
	switch effect {
	case InteractionEffectFade:
		return "Fade in"
	case InteractionEffectFadeUp:
		return "Fade up"
	case InteractionEffectFadeLeft:
		return "Fade from left"
	case InteractionEffectFadeRight:
		return "Fade from right"
	case InteractionEffectLift:
		return "Lift"
	case InteractionEffectGlow:
		return "Glow"
	case InteractionEffectUnderline:
		return "Underline"
	case InteractionEffectScale:
		return "Scale"
	default:
		return "Default"
	}
}

func normalizeInteractionKind(kind InteractionKind) InteractionKind {
	switch InteractionKind(strings.TrimSpace(string(kind))) {
	case InteractionRevealOnScroll:
		return InteractionRevealOnScroll
	case InteractionHoverFocusState:
		return InteractionHoverFocusState
	default:
		return ""
	}
}

func interactionKindAllowed(kind InteractionKind) bool {
	return normalizeInteractionKind(kind) != ""
}

// normalizeInteractionEffect only fills in the kind's default effect when
// none was given. An effect that does not belong to this kind (e.g. "lift" on
// a reveal-on-scroll) is left intact — trimmed/lowercased, never
// silently substituted — so Validate can reject the mismatch with a clear
// error instead of the mistake disappearing into a default.
func normalizeInteractionEffect(kind InteractionKind, effect InteractionEffect) InteractionEffect {
	effect = InteractionEffect(strings.ToLower(strings.TrimSpace(string(effect))))
	if effect != "" {
		return effect
	}
	options := InteractionEffectOptions(kind)
	if len(options) > 0 {
		return options[0]
	}
	return ""
}

func interactionEffectAllowed(kind InteractionKind, effect InteractionEffect) bool {
	for _, option := range InteractionEffectOptions(kind) {
		if option == effect {
			return true
		}
	}
	return false
}

// normalizeInteractionDurationMS only fills in the default duration when none
// was given (the zero value). Any other value — including one outside the
// allow-list — passes through untouched so Validate can reject it explicitly;
// 0 is unambiguous here because it is not itself a legal duration option.
func normalizeInteractionDurationMS(durationMS int) int {
	if durationMS == 0 {
		return DefaultInteractionDurationMS
	}
	return durationMS
}

func interactionDurationAllowed(durationMS int) bool {
	for _, option := range InteractionDurationOptionsMS() {
		if option == durationMS {
			return true
		}
	}
	return false
}

func interactionDelayAllowed(delayMS int) bool {
	for _, option := range InteractionDelayOptionsMS() {
		if option == delayMS {
			return true
		}
	}
	return false
}

func firstInteractionProblem(errs map[string]string) string {
	// Field precedence favors identity over cosmetic detail: an operator
	// fixing "attach this to an element" first is more useful than being told
	// about an effect that will not matter until the target is stable.
	for _, field := range []string{"kind", "target", "effect", "durationMS", "delayMS", "key"} {
		if message, ok := errs[field]; ok {
			return message
		}
	}
	for _, message := range errs {
		return message
	}
	return ""
}
