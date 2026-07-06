package lifecycle

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewRevisionSnapshotsInput(t *testing.T) {
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	revision, err := NewRevision(RevisionInput{
		ID:            "rev_1",
		ResourceKind:  " page ",
		ResourceID:    " page_1 ",
		ResourceTitle: " Care ",
		Action:        " page.updated ",
		Summary:       " Before update ",
		Snapshot:      map[string]string{"title": "Care guide"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if revision.ResourceKind != "page" || revision.ResourceID != "page_1" || revision.Action != "page.updated" || revision.Created != now {
		t.Fatalf("unexpected revision: %#v", revision)
	}
	if !strings.Contains(string(revision.Snapshot), "Care guide") {
		t.Fatalf("expected snapshot payload, got %s", revision.Snapshot)
	}
}

func TestNewRevisionRejectsIncompleteInput(t *testing.T) {
	if _, err := NewRevision(RevisionInput{ResourceKind: "page", Snapshot: map[string]string{"title": "Care"}}, time.Now()); err == nil {
		t.Fatal("expected missing resource id/action error")
	}
	if _, err := NewRevision(RevisionInput{ResourceKind: "page", ResourceID: "p1", Action: "page.updated"}, time.Now()); err == nil {
		t.Fatal("expected missing snapshot error")
	}
}

func TestFilterRevisionsSortsLimitsAndClones(t *testing.T) {
	old := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
	newer := old.Add(time.Hour)
	revisions := []Revision{
		{ID: "old", ResourceKind: "page", ResourceID: "p1", Snapshot: json.RawMessage(`{"title":"old"}`), Created: old},
		{ID: "new", ResourceKind: "page", ResourceID: "p1", Snapshot: json.RawMessage(`{"title":"new"}`), Created: newer},
		{ID: "other", ResourceKind: "blog", ResourceID: "b1", Snapshot: json.RawMessage(`{"title":"other"}`), Created: newer},
	}
	filtered := FilterRevisions(revisions, Filter{ResourceKind: "page", ResourceID: "p1", Limit: 1})
	if len(filtered) != 1 || filtered[0].ID != "new" {
		t.Fatalf("unexpected filtered revisions: %#v", filtered)
	}
	filtered[0].Snapshot[10] = 'X'
	if string(revisions[1].Snapshot) != `{"title":"new"}` {
		t.Fatalf("expected filtered snapshots to be cloned, got %s", revisions[1].Snapshot)
	}
}

func TestFindAndTrimRevisions(t *testing.T) {
	revisions := []Revision{
		{ID: "1", ResourceKind: "page", ResourceID: "p1", Snapshot: json.RawMessage(`{"n":1}`)},
		{ID: "2", ResourceKind: "page", ResourceID: "p1", Snapshot: json.RawMessage(`{"n":2}`)},
		{ID: "3", ResourceKind: "page", ResourceID: "p1", Snapshot: json.RawMessage(`{"n":3}`)},
	}
	found, ok := FindRevision(revisions, "page", "p1", "2")
	if !ok || found.ID != "2" {
		t.Fatalf("expected revision 2, got %#v %v", found, ok)
	}
	trimmed := TrimRevisions(revisions, 2)
	if len(trimmed) != 2 || trimmed[0].ID != "2" || trimmed[1].ID != "3" {
		t.Fatalf("unexpected trim result: %#v", trimmed)
	}
}

func TestDecodeSnapshot(t *testing.T) {
	revision := Revision{Snapshot: json.RawMessage(`{"title":"Care guide"}`)}
	decoded, err := DecodeSnapshot[struct {
		Title string `json:"title"`
	}](revision)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Title != "Care guide" {
		t.Fatalf("unexpected decoded snapshot: %#v", decoded)
	}
}

func TestNormalizeRevisionFillsDefaults(t *testing.T) {
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	revision := NormalizeRevision(Revision{
		ResourceKind: " page ",
		ResourceID:   " p1 ",
		Action:       " page.updated ",
		Snapshot:     json.RawMessage(`{"title":"Care"}`),
	}, now, func(prefix string) string {
		return prefix + "_1"
	})
	if revision.ID != "rev_1" || revision.ResourceKind != "page" || revision.ResourceID != "p1" || revision.Action != "page.updated" || !revision.Created.Equal(now) {
		t.Fatalf("unexpected normalized revision: %#v", revision)
	}
}

func TestActionLabel(t *testing.T) {
	if got := ActionLabel("page.revision_restore"); got != "revision restore" {
		t.Fatalf("unexpected page action label: %q", got)
	}
	if got := ActionLabel(""); got != "saved" {
		t.Fatalf("unexpected empty action label: %q", got)
	}
}
