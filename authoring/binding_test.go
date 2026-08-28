package authoring

import (
	"context"
	"errors"
	"strings"
	"testing"

	"m31labs.dev/gosx-studio/core"
)

type stubBindingAdapter struct {
	kind      core.ResourceKind
	options   []ResourceBindingOption
	resolved  bool
	reason    string
	returnErr error
}

func (adapter stubBindingAdapter) Kind() core.ResourceKind { return adapter.kind }
func (adapter stubBindingAdapter) List(ctx context.Context, query string) ([]ResourceBindingOption, error) {
	return adapter.options, nil
}
func (adapter stubBindingAdapter) Validate(ctx context.Context, binding string) (bool, string, error) {
	if adapter.returnErr != nil {
		return false, "", adapter.returnErr
	}
	return adapter.resolved, adapter.reason, nil
}

func TestNormalizeResourceBindingOptionsTrimsAndFillsLabel(t *testing.T) {
	options := NormalizeResourceBindingOptions([]ResourceBindingOption{
		{Value: " products.featured ", Label: " Featured "},
		{Value: "products.new"},
		{Value: "   "},
	})
	if len(options) != 2 {
		t.Fatalf("expected the blank-value option to be dropped, got %#v", options)
	}
	if options[0].Value != "products.featured" || options[0].Label != "Featured" {
		t.Fatalf("unexpected first option: %#v", options[0])
	}
	if options[1].Label != "products.new" {
		t.Fatalf("expected a missing label to fall back to the value, got %#v", options[1])
	}
}

func TestSiteMapBindingResolverRoutesToRegisteredAdapter(t *testing.T) {
	adapters := ResourceBindingAdapters{
		core.ResourceProducts: stubBindingAdapter{kind: core.ResourceProducts, resolved: true},
	}
	resolveKind := func(component core.Component) core.ResourceKind {
		if strings.HasPrefix(component.Binding, "products.") {
			return core.ResourceProducts
		}
		return core.ResourceMedia
	}
	resolver := SiteMapBindingResolver(context.Background(), adapters, resolveKind)
	resolved, reason := resolver(core.Component{Binding: "products.collection"})
	if !resolved || reason != "" {
		t.Fatalf("expected resolved binding with no reason, got resolved=%v reason=%q", resolved, reason)
	}
}

func TestSiteMapBindingResolverReportsMissingAdapter(t *testing.T) {
	resolveKind := func(component core.Component) core.ResourceKind { return core.ResourceFlows }
	resolver := SiteMapBindingResolver(context.Background(), ResourceBindingAdapters{}, resolveKind)
	resolved, reason := resolver(core.Component{Binding: "flow.contact"})
	if resolved || reason == "" {
		t.Fatalf("expected an unresolved binding with a reason, got resolved=%v reason=%q", resolved, reason)
	}
}

func TestSiteMapBindingResolverPropagatesAdapterError(t *testing.T) {
	adapters := ResourceBindingAdapters{
		core.ResourceFlows: stubBindingAdapter{kind: core.ResourceFlows, returnErr: errors.New("store unavailable")},
	}
	resolveKind := func(component core.Component) core.ResourceKind { return core.ResourceFlows }
	resolver := SiteMapBindingResolver(context.Background(), adapters, resolveKind)
	resolved, reason := resolver(core.Component{Binding: "flow.contact"})
	if resolved || reason != "store unavailable" {
		t.Fatalf("expected the adapter error surfaced as the reason, got resolved=%v reason=%q", resolved, reason)
	}
}

func TestSiteMapBindingResolverIntegratesWithSiteMapDiagnostics(t *testing.T) {
	siteMap := core.SiteMap{Pages: []core.Page{{
		Key: "shop", Label: "Shop", Route: "/shop", GoSXComponent: "ShopPage",
		Components: []core.Component{
			{Key: "product-grid", Label: "Product grid", GoSXComponent: "ProductGrid", Binding: "products.collection"},
		},
	}}}
	adapters := ResourceBindingAdapters{
		core.ResourceProducts: stubBindingAdapter{kind: core.ResourceProducts, resolved: false, reason: "No published products in this collection."},
	}
	resolveKind := func(component core.Component) core.ResourceKind { return core.ResourceProducts }
	resolver := SiteMapBindingResolver(context.Background(), adapters, resolveKind)

	diagnostics := siteMap.BindingDiagnostics(resolver)
	broken := core.BrokenBindingDiagnostics(diagnostics)
	if len(broken) != 1 || broken[0].Reason != "No published products in this collection." {
		t.Fatalf("expected the empty collection to be a broken binding, got %#v", broken)
	}
}
