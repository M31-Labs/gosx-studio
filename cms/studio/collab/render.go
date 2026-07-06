package collab

import (
	"fmt"

	"m31labs.dev/gosx"
)

type RenderProposalOptions struct {
	Class       string
	EmptyTitle  string
	EmptyDetail string
}

type RenderCommentOptions struct {
	Class       string
	EmptyTitle  string
	EmptyDetail string
}

func RenderProposalPanel(snapshot Snapshot, options RenderProposalOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio-proposals")
	proposals := ProposalViews(snapshot)
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__kicker")), gosx.Text("Review")),
				gosx.El("h2", nil, gosx.Text("Proposals")),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", className+"__count")), gosx.Text(fmt.Sprintf("%d pending", pendingProposalCount(proposals)))),
		),
	}
	if len(proposals) == 0 {
		title := firstNonEmpty(options.EmptyTitle, "No proposals")
		detail := firstNonEmpty(options.EmptyDetail, "Human and agent suggestions will appear here before they change the draft.")
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("class", className+"__empty")),
			gosx.El("strong", nil, gosx.Text(title)),
			gosx.El("p", nil, gosx.Text(detail)),
		))
	} else {
		items := make([]gosx.Node, 0, len(proposals))
		for _, proposal := range proposals {
			items = append(items, renderProposalCard(className, proposal))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(items...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-proposals", "true"),
	), gosx.Fragment(children...))
}

func RenderCommentPanel(snapshot Snapshot, options RenderCommentOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio-comments")
	comments := CommentViews(snapshot)
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__kicker")), gosx.Text("Discuss")),
				gosx.El("h2", nil, gosx.Text("Comments")),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", className+"__count")), gosx.Text(fmt.Sprintf("%d open", openCommentCount(comments)))),
		),
	}
	if len(comments) == 0 {
		title := firstNonEmpty(options.EmptyTitle, "No comments")
		detail := firstNonEmpty(options.EmptyDetail, "Canvas and field review notes will appear here.")
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("class", className+"__empty")),
			gosx.El("strong", nil, gosx.Text(title)),
			gosx.El("p", nil, gosx.Text(detail)),
		))
	} else {
		items := make([]gosx.Node, 0, len(comments))
		for _, comment := range comments {
			items = append(items, renderCommentCard(className, comment))
		}
		children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(items...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-comments", "true"),
	), gosx.Fragment(children...))
}

func renderProposalCard(className string, proposal ProposalView) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__card-head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(proposal.Title)),
				gosx.El("span", nil, gosx.Text(proposalStatusMeta(proposal))),
			),
			gosx.El("output", nil, gosx.Text(proposal.StatusLabel)),
		),
	}
	if proposal.Summary != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__summary")), gosx.Text(proposal.Summary)))
	}
	if proposal.ReviewSummary != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__review")), gosx.Text(proposal.ReviewSummary)))
	}
	if len(proposal.Items) > 0 {
		items := make([]gosx.Node, 0, len(proposal.Items))
		for _, item := range proposal.Items {
			items = append(items, gosx.El("li", nil, gosx.Text(item.Summary)))
		}
		children = append(children, gosx.El("ol", gosx.Attrs(gosx.Attr("class", className+"__ops")), gosx.Fragment(items...)))
	}
	if proposal.CanAccept || proposal.CanReject {
		children = append(children, renderProposalActions(className, proposal))
	} else if proposal.DecisionReason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__decision")), gosx.Text(proposal.DecisionReason)))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", className+"__card "+className+"__card--"+string(proposal.Status)),
		gosx.Attr("data-studio-proposal", proposal.ID),
		gosx.Attr("data-studio-proposal-status", string(proposal.Status)),
	), gosx.Fragment(children...))
}

func renderProposalActions(className string, proposal ProposalView) gosx.Node {
	children := []gosx.Node{}
	if proposal.CanAccept {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("class", className+"__accept"),
			gosx.Attr("data-studio-proposal-action", "accept"),
			gosx.Attr("data-studio-proposal-id", proposal.ID),
			gosx.Attr("data-studio-proposal-event", proposal.AcceptEvent),
		), gosx.Text("Accept")))
	}
	if proposal.CanReject {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("class", className+"__reject"),
			gosx.Attr("data-studio-proposal-action", "reject"),
			gosx.Attr("data-studio-proposal-id", proposal.ID),
			gosx.Attr("data-studio-proposal-event", proposal.RejectEvent),
		), gosx.Text("Reject")))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__actions")), gosx.Fragment(children...))
}

func renderCommentCard(className string, comment CommentView) gosx.Node {
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__card-head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(firstNonEmpty(comment.TargetLabel, "Page"))),
				gosx.El("span", nil, gosx.Text(commentMeta(comment))),
			),
			gosx.El("output", nil, gosx.Text(comment.StatusLabel)),
		),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__body")), gosx.Text(comment.Body)),
	}
	if comment.DecisionReason != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__decision")), gosx.Text(comment.DecisionReason)))
	}
	if comment.CanResolve || comment.CanReopen {
		children = append(children, renderCommentActions(className, comment))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", className+"__card "+className+"__card--"+string(comment.Status)),
		gosx.Attr("data-studio-comment", comment.ID),
		gosx.Attr("data-studio-comment-status", string(comment.Status)),
		gosx.Attr("data-studio-comment-block", comment.BlockID),
		gosx.Attr("data-studio-comment-field", comment.Field),
	), gosx.Fragment(children...))
}

func renderCommentActions(className string, comment CommentView) gosx.Node {
	children := []gosx.Node{}
	if comment.CanResolve {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("class", className+"__resolve"),
			gosx.Attr("data-studio-comment-action", "resolve"),
			gosx.Attr("data-studio-comment-id", comment.ID),
			gosx.Attr("data-studio-comment-event", comment.ResolveEvent),
		), gosx.Text("Resolve")))
	}
	if comment.CanReopen {
		children = append(children, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("class", className+"__reopen"),
			gosx.Attr("data-studio-comment-action", "reopen"),
			gosx.Attr("data-studio-comment-id", comment.ID),
			gosx.Attr("data-studio-comment-event", comment.ReopenEvent),
		), gosx.Text("Reopen")))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__actions")), gosx.Fragment(children...))
}

func pendingProposalCount(proposals []ProposalView) int {
	count := 0
	for _, proposal := range proposals {
		if proposal.Status == ProposalPending {
			count++
		}
	}
	return count
}

func openCommentCount(comments []CommentView) int {
	count := 0
	for _, comment := range comments {
		if comment.Status == CommentOpen {
			count++
		}
	}
	return count
}

func proposalStatusMeta(proposal ProposalView) string {
	if proposal.DecisionActorID != "" {
		return fmt.Sprintf("%s by %s", proposal.StatusLabel, proposal.DecisionActorID)
	}
	if proposal.ActorID != "" {
		return fmt.Sprintf("%d operations from %s", proposal.OperationCount, proposal.ActorID)
	}
	return fmt.Sprintf("%d operations", proposal.OperationCount)
}

func commentMeta(comment CommentView) string {
	if comment.DecisionActorID != "" {
		return fmt.Sprintf("%s by %s", comment.StatusLabel, comment.DecisionActorID)
	}
	if comment.ActorID != "" {
		return fmt.Sprintf("From %s", comment.ActorID)
	}
	return "Review note"
}
