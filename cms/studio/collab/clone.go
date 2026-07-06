package collab

import (
	"sort"
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
	admincollab "m31labs.dev/gosx-admin/blockstudio/collab"
)

func normalizeResource(resource Resource) Resource {
	resource.Kind = cleanID(resource.Kind)
	resource.ID = cleanID(resource.ID)
	resource.Title = strings.TrimSpace(resource.Title)
	return resource
}

func resourceKey(resource Resource) string {
	resource = normalizeResource(resource)
	return resource.Kind + ":" + resource.ID
}

func cleanID(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return strings.Trim(value, "-")
}

func cloneDocument(doc blockstudio.Document) blockstudio.Document {
	out := blockstudio.Document{Version: doc.Version, Kind: doc.Kind, Blocks: make([]blockstudio.BlockInstance, 0, len(doc.Blocks))}
	for _, block := range doc.Blocks {
		out.Blocks = append(out.Blocks, cloneBlock(block))
	}
	return out
}

func cloneBlock(block blockstudio.BlockInstance) blockstudio.BlockInstance {
	if len(block.Values) > 0 {
		values := blockstudio.Values{}
		for key, value := range block.Values {
			values[key] = cloneValue(value)
		}
		block.Values = values
	}
	return block
}

func cloneValue(value blockstudio.Value) blockstudio.Value {
	if len(value.List) > 0 {
		list := make([]blockstudio.Value, len(value.List))
		for index, item := range value.List {
			list[index] = cloneValue(item)
		}
		value.List = list
	}
	if len(value.Object) > 0 {
		object := map[string]blockstudio.Value{}
		for key, item := range value.Object {
			object[key] = cloneValue(item)
		}
		value.Object = object
	}
	if value.Media != nil {
		media := *value.Media
		value.Media = &media
	}
	if value.Relation != nil {
		relation := *value.Relation
		value.Relation = &relation
	}
	return value
}

func cloneSuggestions(values []admincollab.Suggestion) []admincollab.Suggestion {
	out := make([]admincollab.Suggestion, len(values))
	copy(out, values)
	for index := range out {
		out[index].Operations = admincollab.NormalizeOperations(out[index].Operations)
	}
	return out
}

func cloneComments(values []admincollab.Comment) []admincollab.Comment {
	out := make([]admincollab.Comment, len(values))
	copy(out, values)
	return out
}

func cloneCommentDecisions(values []CommentDecision) []CommentDecision {
	out := make([]CommentDecision, len(values))
	copy(out, values)
	for index := range out {
		out[index].Actor = admincollab.NormalizeActor(out[index].Actor)
		out[index].Reason = strings.TrimSpace(out[index].Reason)
	}
	return out
}

func cloneProposalDecisions(values []ProposalDecision) []ProposalDecision {
	out := make([]ProposalDecision, len(values))
	copy(out, values)
	for index := range out {
		out[index].Actor = admincollab.NormalizeActor(out[index].Actor)
		out[index].Reason = strings.TrimSpace(out[index].Reason)
		out[index].Review.Actor = admincollab.NormalizeActor(out[index].Review.Actor)
		out[index].Review.Items = append([]admincollab.ReviewItem(nil), out[index].Review.Items...)
		out[index].Review.Findings = append([]admincollab.ReviewFinding(nil), out[index].Review.Findings...)
	}
	return out
}

func cloneReviews(values []admincollab.Review) []admincollab.Review {
	out := make([]admincollab.Review, len(values))
	copy(out, values)
	for index := range out {
		out[index].Actor = admincollab.NormalizeActor(out[index].Actor)
		out[index].Items = append([]admincollab.ReviewItem(nil), out[index].Items...)
		out[index].Findings = append([]admincollab.ReviewFinding(nil), out[index].Findings...)
	}
	return out
}

func sortPresence(values []Presence) []Presence {
	sort.SliceStable(values, func(i, j int) bool {
		return values[i].Actor.ID < values[j].Actor.ID
	})
	return values
}
