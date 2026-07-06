package shell

import (
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-cms/lifecycle"
	"m31labs.dev/gosx-cms/media"

	"m31labs.dev/gosx-studio/core"
)

// cmsbridge.go is the single designed seam where the shell package touches the
// gosx-cms and gosx-admin types (media.Asset, lifecycle.Revision,
// blockstudio.Definition). Per the restructure spec §1 "Shell summaries vs
// cms/admin types" seam, the pure BlockSummary/MediaSummary/RevisionSummary
// structs stay in options.go while the Store interface and the three cms/admin
// converters are isolated here — this is the one file the Phase 2 cms fold-in
// rewrites. Keeping it separate means the rest of shell composes chrome without
// reaching into content-storage packages.

// Store is the small contract a host implements to feed Studio with content.
type Store interface {
	LeftPanels() []Panel
	RightPanels() []Panel
	MediaAssets() []media.Asset
	Revisions() []lifecycle.Revision
	Readiness() ShellReadiness
	Adapters() []core.ResourceAdapter
}

func shellBlockSummaries(blocks []blockstudio.Definition) []BlockSummary {
	out := make([]BlockSummary, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, BlockSummary{
			Key:        block.Key,
			Label:      block.Label,
			Summary:    block.Summary,
			Kind:       block.Kind,
			Preview:    block.Preview,
			DefaultOn:  block.DefaultOn,
			Locked:     block.Locked,
			Repeatable: block.Repeatable,
			Icon:       block.Icon,
			FieldCount: len(block.Fields),
		})
	}
	return out
}

func shellMediaSummaries(assets []media.Asset) []MediaSummary {
	out := make([]MediaSummary, 0, len(assets))
	for _, asset := range assets {
		out = append(out, MediaSummary{
			ID:          asset.ID,
			URL:         asset.URL,
			Alt:         asset.Alt,
			Filename:    asset.Filename,
			ContentType: asset.ContentType,
			Size:        asset.Size,
			Archived:    asset.ArchivedAt != nil,
		})
	}
	return out
}

func shellRevisionSummaries(revisions []lifecycle.Revision) []RevisionSummary {
	out := make([]RevisionSummary, 0, len(revisions))
	for index, revision := range revisions {
		created := ""
		if !revision.Created.IsZero() {
			created = revision.Created.Format("2006-01-02T15:04:05Z07:00")
		}
		summary := RevisionSummary{
			ID:            revision.ID,
			ResourceKind:  revision.ResourceKind,
			ResourceID:    revision.ResourceID,
			ResourceTitle: revision.ResourceTitle,
			Action:        revision.Action,
			Summary:       revision.Summary,
			Created:       created,
		}
		if index+1 < len(revisions) {
			if diff, err := lifecycle.DiffRevisions(revisions[index+1], revision); err == nil {
				summary.ChangeSummary = diff.Summary
				summary.ChangeCount = len(diff.Changes)
				summary.HasDiff = true
			}
		}
		out = append(out, summary)
	}
	return out
}
