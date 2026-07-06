package lifecycle

import (
	"encoding/json"
	"testing"
)

func TestDiffRevisionsSummarizesSnapshotChanges(t *testing.T) {
	from := Revision{
		ID:           "rev_old",
		ResourceKind: "page",
		ResourceID:   "home",
		Snapshot: json.RawMessage(`{
			"title": "Home",
			"hero": {"headline": "Welcome", "subhead": "Forest days"},
			"sections": [{"key": "hero"}, {"key": "contact"}],
			"draft": true
		}`),
	}
	to := Revision{
		ID:           "rev_new",
		ResourceKind: "page",
		ResourceID:   "home",
		Snapshot: json.RawMessage(`{
			"title": "Home",
			"hero": {"headline": "Welcome families", "cta": "Visit"},
			"sections": [{"key": "hero"}],
			"draft": false
		}`),
	}
	diff, err := DiffRevisions(from, to)
	if err != nil {
		t.Fatal(err)
	}
	if diff.FromRevisionID != "rev_old" || diff.ToRevisionID != "rev_new" || diff.ResourceKind != "page" || diff.ResourceID != "home" {
		t.Fatalf("unexpected diff identity: %#v", diff)
	}
	if diff.Summary != "2 changed fields, 1 added field, 2 removed fields." {
		t.Fatalf("unexpected summary: %q", diff.Summary)
	}
	want := []RevisionChange{
		{Path: "draft", Kind: RevisionChangeChanged, Before: "true", After: "false"},
		{Path: "hero.cta", Kind: RevisionChangeAdded, After: "Visit"},
		{Path: "hero.headline", Kind: RevisionChangeChanged, Before: "Welcome", After: "Welcome families"},
		{Path: "hero.subhead", Kind: RevisionChangeRemoved, Before: "Forest days"},
		{Path: "sections[1].key", Kind: RevisionChangeRemoved, Before: "contact"},
	}
	if len(diff.Changes) != len(want) {
		t.Fatalf("expected %d changes, got %#v", len(want), diff.Changes)
	}
	for index, expected := range want {
		if diff.Changes[index] != expected {
			t.Fatalf("unexpected change at %d:\ngot  %#v\nwant %#v", index, diff.Changes[index], expected)
		}
	}
}

func TestDiffRevisionsReportsNoChangesAndDecodeErrors(t *testing.T) {
	one := Revision{ID: "one", Snapshot: json.RawMessage(`{"title":"Same"}`)}
	two := Revision{ID: "two", Snapshot: json.RawMessage(`{"title":"Same"}`)}
	diff, err := DiffRevisions(one, two)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Summary != "No changes." || len(diff.Changes) != 0 {
		t.Fatalf("unexpected no-op diff: %#v", diff)
	}
	if _, err := DiffRevisions(Revision{ID: "bad", Snapshot: json.RawMessage(`{`)}, two); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestRevisionDiffSummary(t *testing.T) {
	summary := RevisionDiffSummary([]RevisionChange{
		{Kind: RevisionChangeAdded},
		{Kind: RevisionChangeRemoved},
		{Kind: RevisionChangeChanged},
	})
	if summary != "1 changed field, 1 added field, 1 removed field." {
		t.Fatalf("unexpected summary: %q", summary)
	}
}
