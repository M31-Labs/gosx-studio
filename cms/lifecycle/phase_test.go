package lifecycle

import "testing"

func TestDerivePhase(t *testing.T) {
	tests := []struct {
		name           string
		draft          DraftState
		publish        PublishState
		decision       PublishDecision
		hasDecision    bool
		readinessClear bool
		want           Phase
	}{
		{
			name:    "fresh draft, no decision",
			draft:   DraftStateDraft,
			publish: PublishStateDraft,
			want:    PhaseDraft,
		},
		{
			name:        "pending decision opens review",
			draft:       DraftStatePreview,
			publish:     PublishStateDraft,
			decision:    PublishDecision{Status: DecisionPending},
			hasDecision: true,
			want:        PhaseReview,
		},
		{
			name:           "approved and readiness clear is ready",
			draft:          DraftStatePreview,
			publish:        PublishStateDraft,
			decision:       PublishDecision{Status: DecisionApproved},
			hasDecision:    true,
			readinessClear: true,
			want:           PhaseReady,
		},
		{
			name:           "approved but readiness blocked stays draft",
			draft:          DraftStatePreview,
			publish:        PublishStateDraft,
			decision:       PublishDecision{Status: DecisionApproved},
			hasDecision:    true,
			readinessClear: false,
			want:           PhaseDraft,
		},
		{
			name:    "published live state wins over no decision",
			draft:   DraftStateDraft,
			publish: PublishStatePublished,
			want:    PhasePublished,
		},
		{
			name:        "published live state wins over stale approved decision",
			draft:       DraftStateDraft,
			publish:     PublishStatePublished,
			decision:    PublishDecision{Status: DecisionApproved},
			hasDecision: true,
			want:        PhasePublished,
		},
		{
			name:        "new pending review on top of published live reopens review",
			draft:       DraftStatePreview,
			publish:     PublishStatePublished,
			decision:    PublishDecision{Status: DecisionPending},
			hasDecision: true,
			want:        PhaseReview,
		},
		{
			name:        "rejected decision stays draft",
			draft:       DraftStatePreview,
			publish:     PublishStateDraft,
			decision:    PublishDecision{Status: DecisionRejected},
			hasDecision: true,
			want:        PhaseDraft,
		},
		{
			name:    "rollback wins regardless of publish state",
			draft:   DraftStateRollback,
			publish: PublishStatePublished,
			want:    PhaseRolledBack,
		},
		{
			name:        "rollback wins regardless of pending decision",
			draft:       DraftStateRollback,
			publish:     PublishStateDraft,
			decision:    PublishDecision{Status: DecisionPending},
			hasDecision: true,
			want:        PhaseRolledBack,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := DerivePhase(test.draft, test.publish, test.decision, test.hasDecision, test.readinessClear)
			if got != test.want {
				t.Fatalf("DerivePhase() = %q, want %q", got, test.want)
			}
		})
	}
}
