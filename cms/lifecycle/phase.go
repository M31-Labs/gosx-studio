package lifecycle

// Phase is a derived (not persisted) coarse lifecycle stage for one
// resource. It is always computed on read from the existing DraftState /
// PublishState pair (the two fields already embedded in every host's
// persisted state, e.g. cmsstore.State.Draft / cmsstore.State.Publish) plus
// the latest PublishDecision. Because Phase is never stored, adding or
// reordering cases here is never a migration.
type Phase string

const (
	// PhaseDraft is a dirty draft with no review opened.
	PhaseDraft Phase = "draft"
	// PhaseReview has a Pending PublishDecision open.
	PhaseReview Phase = "review"
	// PhaseReady is approved and readiness is clear, but not yet applied.
	PhaseReady Phase = "ready"
	// PhasePublished has been applied to live (staging).
	PhasePublished Phase = "published"
	// PhasePromoted records a promotion intent for a prod-shaped
	// environment. DerivePhase never returns this value: promotion is a
	// recorded intent tracked by cms/lifecycle/engine, layered on top of
	// (not derived from) DraftState/PublishState/PublishDecision. A caller
	// that also tracks the latest PromotionIntent overlays PhasePromoted
	// itself once a promotion has been recorded for the current published
	// revision.
	PhasePromoted Phase = "promoted"
	// PhaseRolledBack means the last transition was a restore.
	PhaseRolledBack Phase = "rolled_back"
)

// DerivePhase composes the existing DraftState/PublishState with the latest
// PublishDecision (hasDecision reports whether one exists at all) and the
// current readiness gate state for the resource. DraftStateRollback always
// wins and maps to PhaseRolledBack, matching the existing rollback marker
// hosts already set via RestorePageRevision/RestoreSettingsRevision/etc.
func DerivePhase(draft DraftState, publish PublishState, decision PublishDecision, hasDecision bool, readinessClear bool) Phase {
	if draft == DraftStateRollback {
		return PhaseRolledBack
	}
	if hasDecision && decision.Status == DecisionPending {
		return PhaseReview
	}
	if publish == PublishStatePublished {
		return PhasePublished
	}
	if hasDecision && decision.Status == DecisionApproved && readinessClear {
		return PhaseReady
	}
	return PhaseDraft
}
