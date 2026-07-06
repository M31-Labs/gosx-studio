package collab

import (
	"strings"

	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
)

type ProposalView struct {
	ID              string
	Title           string
	Summary         string
	ActorID         string
	ActorKind       string
	Status          ProposalStatus
	StatusLabel     string
	OperationCount  int
	ReviewSummary   string
	CanAccept       bool
	CanReject       bool
	AcceptEvent     string
	RejectEvent     string
	DecisionActorID string
	DecisionReason  string
	Items           []ProposalItemView
}

type ProposalItemView struct {
	OperationID string
	Kind        string
	Summary     string
}

type CommentView struct {
	ID              string
	Body            string
	ActorID         string
	ActorKind       string
	TargetLabel     string
	BlockID         string
	Field           string
	Status          CommentStatus
	StatusLabel     string
	CanResolve      bool
	CanReopen       bool
	ResolveEvent    string
	ReopenEvent     string
	DecisionActorID string
	DecisionReason  string
}

func SnapshotView(snapshot Snapshot) map[string]any {
	proposals := ProposalViews(snapshot)
	comments := CommentViews(snapshot)
	pending, accepted, rejected := proposalCounts(proposals)
	openComments, resolvedComments := commentCounts(comments)
	return map[string]any{
		"resource":             snapshot.Resource,
		"proposalCount":        len(proposals),
		"pendingCount":         pending,
		"acceptedCount":        accepted,
		"rejectedCount":        rejected,
		"proposals":            proposalViewMaps(proposals),
		"presenceCount":        len(snapshot.Presence),
		"commentCount":         len(comments),
		"openCommentCount":     openComments,
		"resolvedCommentCount": resolvedComments,
		"comments":             commentViewMaps(comments),
		"reviewCount":          len(snapshot.Reviews),
		"updated":              snapshot.Updated,
	}
}

func ProposalViews(snapshot Snapshot) []ProposalView {
	decisions := latestDecisions(snapshot.ProposalDecisions)
	reviews := reviewsByTransaction(snapshot.Reviews)
	out := make([]ProposalView, 0, len(snapshot.Suggestions))
	for _, suggestion := range snapshot.Suggestions {
		decision, decided := decisions[suggestion.ID]
		status := ProposalPending
		if decided {
			status = normalizeProposalStatus(decision.Status)
		}
		review := reviews[suggestion.ID]
		view := ProposalView{
			ID:             suggestion.ID,
			Title:          firstNonEmpty(suggestion.Title, "Untitled suggestion"),
			Summary:        strings.TrimSpace(suggestion.Summary),
			ActorID:        suggestion.ActorID,
			ActorKind:      string(suggestion.ActorKind),
			Status:         status,
			StatusLabel:    proposalStatusLabel(status),
			OperationCount: len(admincollab.NormalizeOperations(suggestion.Operations)),
			ReviewSummary:  review.Summary,
			CanAccept:      status == ProposalPending,
			CanReject:      status == ProposalPending,
			AcceptEvent:    "studio.acceptSuggestion",
			RejectEvent:    "studio.rejectSuggestion",
			Items:          proposalItemViews(review.Items),
		}
		if decided {
			view.DecisionActorID = decision.Actor.ID
			view.DecisionReason = strings.TrimSpace(decision.Reason)
			if decision.Review.Summary != "" {
				view.ReviewSummary = decision.Review.Summary
				view.Items = proposalItemViews(decision.Review.Items)
			}
		}
		out = append(out, view)
	}
	return out
}

func CommentViews(snapshot Snapshot) []CommentView {
	decisions := latestCommentDecisions(snapshot.CommentDecisions)
	out := make([]CommentView, 0, len(snapshot.Comments))
	for _, comment := range snapshot.Comments {
		decision, decided := decisions[comment.ID]
		status := CommentOpen
		if decided {
			status = normalizeCommentStatus(decision.Status)
		}
		view := CommentView{
			ID:           comment.ID,
			Body:         strings.TrimSpace(comment.Body),
			ActorID:      comment.ActorID,
			ActorKind:    string(comment.ActorKind),
			TargetLabel:  targetLabel(comment.Target),
			BlockID:      comment.Target.BlockID,
			Field:        comment.Target.Field,
			Status:       status,
			StatusLabel:  commentStatusLabel(status),
			CanResolve:   status == CommentOpen,
			CanReopen:    status == CommentResolved,
			ResolveEvent: "studio.resolveComment",
			ReopenEvent:  "studio.reopenComment",
		}
		if decided {
			view.DecisionActorID = decision.Actor.ID
			view.DecisionReason = strings.TrimSpace(decision.Reason)
		}
		out = append(out, view)
	}
	return out
}

func commentViewMaps(comments []CommentView) []map[string]any {
	out := make([]map[string]any, 0, len(comments))
	for _, comment := range comments {
		out = append(out, map[string]any{
			"id":              comment.ID,
			"body":            comment.Body,
			"actorID":         comment.ActorID,
			"actorKind":       comment.ActorKind,
			"targetLabel":     comment.TargetLabel,
			"blockID":         comment.BlockID,
			"field":           comment.Field,
			"status":          string(comment.Status),
			"statusLabel":     comment.StatusLabel,
			"canResolve":      comment.CanResolve,
			"canReopen":       comment.CanReopen,
			"resolveEvent":    comment.ResolveEvent,
			"reopenEvent":     comment.ReopenEvent,
			"decisionActorID": comment.DecisionActorID,
			"decisionReason":  comment.DecisionReason,
		})
	}
	return out
}

func proposalViewMaps(proposals []ProposalView) []map[string]any {
	out := make([]map[string]any, 0, len(proposals))
	for _, proposal := range proposals {
		out = append(out, map[string]any{
			"id":              proposal.ID,
			"title":           proposal.Title,
			"summary":         proposal.Summary,
			"actorID":         proposal.ActorID,
			"actorKind":       proposal.ActorKind,
			"status":          string(proposal.Status),
			"statusLabel":     proposal.StatusLabel,
			"operationCount":  proposal.OperationCount,
			"reviewSummary":   proposal.ReviewSummary,
			"canAccept":       proposal.CanAccept,
			"canReject":       proposal.CanReject,
			"acceptEvent":     proposal.AcceptEvent,
			"rejectEvent":     proposal.RejectEvent,
			"decisionActorID": proposal.DecisionActorID,
			"decisionReason":  proposal.DecisionReason,
			"items":           proposalItemViewMaps(proposal.Items),
		})
	}
	return out
}

func proposalItemViews(items []admincollab.ReviewItem) []ProposalItemView {
	out := make([]ProposalItemView, 0, len(items))
	for _, item := range items {
		out = append(out, ProposalItemView{
			OperationID: item.OperationID,
			Kind:        string(item.Kind),
			Summary:     item.Summary,
		})
	}
	return out
}

func proposalItemViewMaps(items []ProposalItemView) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"operationID": item.OperationID,
			"kind":        item.Kind,
			"summary":     item.Summary,
		})
	}
	return out
}

func proposalCounts(proposals []ProposalView) (pending, accepted, rejected int) {
	for _, proposal := range proposals {
		switch proposal.Status {
		case ProposalAccepted:
			accepted++
		case ProposalRejected:
			rejected++
		default:
			pending++
		}
	}
	return pending, accepted, rejected
}

func latestDecisions(decisions []ProposalDecision) map[string]ProposalDecision {
	out := map[string]ProposalDecision{}
	for _, decision := range decisions {
		decision.SuggestionID = strings.TrimSpace(decision.SuggestionID)
		if decision.SuggestionID == "" {
			continue
		}
		decision.Status = normalizeProposalStatus(decision.Status)
		out[decision.SuggestionID] = decision
	}
	return out
}

func latestCommentDecisions(decisions []CommentDecision) map[string]CommentDecision {
	out := map[string]CommentDecision{}
	for _, decision := range decisions {
		decision.CommentID = strings.TrimSpace(decision.CommentID)
		if decision.CommentID == "" {
			continue
		}
		decision.Status = normalizeCommentStatus(decision.Status)
		out[decision.CommentID] = decision
	}
	return out
}

func reviewsByTransaction(reviews []admincollab.Review) map[string]admincollab.Review {
	out := map[string]admincollab.Review{}
	for _, review := range reviews {
		key := strings.TrimSpace(review.TransactionID)
		if key != "" {
			out[key] = review
		}
	}
	return out
}

func normalizeProposalStatus(status ProposalStatus) ProposalStatus {
	switch status {
	case ProposalAccepted, ProposalRejected:
		return status
	default:
		return ProposalPending
	}
}

func proposalStatusLabel(status ProposalStatus) string {
	switch normalizeProposalStatus(status) {
	case ProposalAccepted:
		return "Accepted"
	case ProposalRejected:
		return "Rejected"
	default:
		return "Pending"
	}
}

func commentCounts(comments []CommentView) (open, resolved int) {
	for _, comment := range comments {
		if comment.Status == CommentResolved {
			resolved++
		} else {
			open++
		}
	}
	return open, resolved
}

func commentStatusLabel(status CommentStatus) string {
	if normalizeCommentStatus(status) == CommentResolved {
		return "Resolved"
	}
	return "Open"
}

func targetLabel(target admincollab.Target) string {
	blockID := strings.TrimSpace(target.BlockID)
	field := strings.TrimSpace(target.Field)
	scope := strings.TrimSpace(target.Scope)
	scopeID := strings.TrimSpace(target.ScopeID)
	selector := strings.TrimSpace(target.Selector)
	switch {
	case blockID != "" && field != "":
		return blockID + " / " + field
	case blockID != "":
		return blockID
	case scope != "" && scopeID != "":
		return scope + " / " + scopeID
	case selector != "":
		return selector
	default:
		return "Page"
	}
}
