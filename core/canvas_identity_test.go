package core

import "testing"

func TestCanvasIdentityNormalizesRouteAndUsesLeafFirstPrecedence(t *testing.T) {
	identity := (CanvasIdentity{
		Route:    " /about?gosx-preview=1#story ",
		PageID:   "page:about",
		Field:    " pages.about.title ",
		BlockKey: "about-story",
		NodeID:   "node-about-story",
	}).Normalize()
	if identity.Route != "/about" || identity.Field != "pages.about.title" || identity.Kind != "field" {
		t.Fatalf("unexpected normalized identity: %#v", identity)
	}
	if !identity.Stable() {
		t.Fatal("expected route-scoped field identity to be stable")
	}
}

func TestCanvasIdentityFallsBackFromBlockToNodeToPage(t *testing.T) {
	tests := []struct {
		identity CanvasIdentity
		kind     string
	}{
		{CanvasIdentity{Route: "/", BlockKey: "hero", NodeID: "node-hero"}, "block"},
		{CanvasIdentity{Route: "/", NodeID: "node-hero"}, "node"},
		{CanvasIdentity{Route: "/about", PageID: "page:about"}, "page"},
	}
	for _, test := range tests {
		if got := test.identity.Normalize().Kind; got != test.kind {
			t.Fatalf("expected %q, got %q", test.kind, got)
		}
	}
}

func TestCanvasIdentityNormalizesAbsolutePreviewURLToRoutePath(t *testing.T) {
	identity := (CanvasIdentity{Route: "https://noni.example/about?gosx-preview=1", PageID: "page:about"}).Normalize()
	if identity.Route != "/about" {
		t.Fatalf("expected absolute preview URL to normalize to /about, got %q", identity.Route)
	}
}
