package studio

import "testing"

// lockedFixtureControls returns one editor-configurable control and one
// dev-locked control. The locked control's value/binding still travel with the
// template (so the dev-set wiring is preserved) but the editor must never offer
// it as an editable field.
//
// This is a root-local duplicate of authoring.lockedFixtureControls: the two
// packages both need the fixture (one for authoringSiteMap* view helpers that
// remain at root pending the sitemap slice, the other for authoring.go's
// mutation-boundary control views), and the fixture is small enough that
// duplicating it is simpler and safer than exporting test-only helpers across
// a package boundary.
func lockedFixtureControls() []Control {
	return []Control{
		{
			Key:     "product",
			Label:   "Product",
			Kind:    ControlText,
			Binding: "plugin.stripe.product",
			Value:   "prod_123",
		},
		{
			Key:     "action",
			Label:   "Checkout endpoint",
			Kind:    ControlText,
			Binding: "plugin.stripe.checkout",
			Value:   "/api/checkout",
			Locked:  true,
		},
	}
}

func controlViewKeys(views []map[string]any) []string {
	keys := make([]string, 0, len(views))
	for _, view := range views {
		if key, ok := view["key"].(string); ok {
			keys = append(keys, key)
		}
	}
	return keys
}

func TestControlLockedRawTemplateRetainsLockedControl(t *testing.T) {
	template := ComponentTemplate{
		Key:           "stripe-buy",
		Label:         "Buy button",
		GoSXComponent: "StripeBuyButton",
		Controls:      lockedFixtureControls(),
	}
	normalized := template.Normalize()
	if len(normalized.Controls) != 2 {
		t.Fatalf("normalized template should retain both controls, got %d: %#v", len(normalized.Controls), normalized.Controls)
	}
	var locked *Control
	for i := range normalized.Controls {
		if normalized.Controls[i].Key == "action" {
			locked = &normalized.Controls[i]
		}
	}
	if locked == nil {
		t.Fatalf("locked control should survive normalization: %#v", normalized.Controls)
	}
	if !locked.Locked {
		t.Fatalf("Locked flag should survive normalization: %#v", *locked)
	}
	if locked.Binding != "plugin.stripe.checkout" || locked.Value != "/api/checkout" {
		t.Fatalf("locked control wiring should travel with the template: %#v", *locked)
	}
}

func TestAuthoringSiteMapControlViewsExcludeLocked(t *testing.T) {
	page := Page{Key: "store", Label: "Store", Route: "/store"}
	component := Component{Key: "buy", Label: "Buy", GoSXComponent: "StripeBuyButton", Editable: true}
	views := authoringSiteMapControlViews(page, component, lockedFixtureControls())
	keys := controlViewKeys(views)
	if len(keys) != 1 || keys[0] != "product" {
		t.Fatalf("authoringSiteMapControlViews should expose only the editable control, got %#v", keys)
	}
	if views[0]["locked"] != false {
		t.Fatalf("sitemap control view should expose locked=false, got %#v", views[0]["locked"])
	}
}

func TestAuthoringSiteMapStaticControlViewsExcludeLocked(t *testing.T) {
	views := authoringSiteMapStaticControlViews(lockedFixtureControls())
	keys := controlViewKeys(views)
	if len(keys) != 1 || keys[0] != "product" {
		t.Fatalf("authoringSiteMapStaticControlViews should expose only the editable control, got %#v", keys)
	}
}

func TestAuthoringSiteMapEditableControlViewSkipsLocked(t *testing.T) {
	// Only the locked control exists as a candidate: the editable picker must
	// skip it and return nil rather than offering the dev-locked control.
	pages := []Page{{
		Key:   "store",
		Label: "Store",
		Route: "/store",
		Components: []Component{{
			Key:           "buy",
			Label:         "Buy",
			GoSXComponent: "StripeBuyButton",
			Editable:      true,
			Controls: []Control{{
				Key:     "action",
				Label:   "Checkout endpoint",
				Kind:    ControlText,
				Binding: "plugin.stripe.checkout",
				Value:   "/api/checkout",
				Locked:  true,
			}},
		}},
	}}
	if view := authoringSiteMapEditableControlView(pages); view != nil {
		t.Fatalf("editable control picker should skip locked controls, got %#v", view)
	}
}
