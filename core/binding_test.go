package core

import "testing"

func testBindingSiteMap() SiteMap {
	return SiteMap{Pages: []Page{
		{
			Key:           "shop",
			Label:         "Shop",
			Route:         "/shop",
			GoSXComponent: "ShopPage",
			Components: []Component{
				{Key: "product-grid", Label: "Product grid", GoSXComponent: "ProductGrid", Binding: "products.collection"},
				{Key: "static-hero", Label: "Hero", GoSXComponent: "Hero"}, // no binding: nothing to validate
			},
		},
	}}
}

func TestBindingDiagnosticsResolvesEachBoundComponent(t *testing.T) {
	siteMap := testBindingSiteMap()
	resolve := func(component Component) (bool, string) {
		if component.Binding == "products.collection" {
			return true, ""
		}
		return false, "unexpected binding"
	}
	diagnostics := siteMap.BindingDiagnostics(resolve)
	if len(diagnostics) != 1 {
		t.Fatalf("expected exactly one diagnostic for the one bound component, got %#v", diagnostics)
	}
	if diagnostics[0].ComponentKey != "product-grid" || !diagnostics[0].Resolved {
		t.Fatalf("unexpected diagnostic: %#v", diagnostics[0])
	}
	if diagnostics[0].Status() != ReadinessReady {
		t.Fatalf("expected ready status, got %v", diagnostics[0].Status())
	}
}

func TestBindingDiagnosticsFlagsBrokenBinding(t *testing.T) {
	siteMap := testBindingSiteMap()
	resolve := func(component Component) (bool, string) {
		return false, "No published products in this collection."
	}
	diagnostics := siteMap.BindingDiagnostics(resolve)
	broken := BrokenBindingDiagnostics(diagnostics)
	if len(broken) != 1 {
		t.Fatalf("expected one broken binding, got %#v", broken)
	}
	if broken[0].Status() != ReadinessBlocked {
		t.Fatalf("expected blocked status, got %v", broken[0].Status())
	}
	if broken[0].Reason != "No published products in this collection." {
		t.Fatalf("expected the resolver's reason to be surfaced, got %q", broken[0].Reason)
	}
}

func TestBindingDiagnosticsNilResolverReturnsNil(t *testing.T) {
	if got := testBindingSiteMap().BindingDiagnostics(nil); got != nil {
		t.Fatalf("expected nil diagnostics with no resolver, got %#v", got)
	}
}

func TestBindingDiagnosticUnboundComponentIsReady(t *testing.T) {
	diagnostic := BindingDiagnostic{ComponentKey: "static-hero"}
	if diagnostic.Status() != ReadinessReady {
		t.Fatalf("expected an unbound component to be Ready (nothing to validate), got %v", diagnostic.Status())
	}
}
