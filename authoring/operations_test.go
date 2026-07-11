package authoring

import "testing"

func TestOperationTargetKeyIsStableAcrossOperationKinds(t *testing.T) {
	target := OperationTarget{Route: "https://example.test/about?gosx-preview=1#x/", PageID: "page:about", Field: "pages.about.title"}
	if target.Key(OperationSetField) != target.Key(OperationUndo) {
		t.Fatal("target key must identify target, not operation kind")
	}
	if target.Normalize().Route != "/about" {
		t.Fatalf("route=%q", target.Normalize().Route)
	}
}

func TestOperationInverseUsesResetForAbsentStyle(t *testing.T) {
	record := OperationRecord{ID: "op", Kind: OperationSetStyle, Target: OperationTarget{ComponentKey: "home:hero", Property: "color", Breakpoint: "mobile"}, TargetHead: "head", Before: OperationValue{Present: false}}
	request := record.InverseRequest("undo", "actor", 2)
	if request.Kind != OperationResetStyle || request.ExpectedTargetHead != "head" {
		t.Fatalf("inverse=%#v", request)
	}
}

func TestOperationRequestsEqualPreservesContentWhitespace(t *testing.T) {
	a := OperationRequest{ID: "a", Kind: OperationSetField, Target: OperationTarget{Field: "hero.headline"}, Value: "  headline  "}
	b := a
	b.ID = "b"
	if !RequestsEqual(a, b) {
		t.Fatal("canonical requests should match")
	}
	c := b
	c.Value = "headline"
	if RequestsEqual(a, c) {
		t.Fatal("content whitespace must not be discarded")
	}
}
