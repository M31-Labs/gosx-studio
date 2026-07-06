package authoring

import (
	"testing"

	"m31labs.dev/gosx-studio/core"
)

// lockedFixtureControls returns one editor-configurable control and one
// dev-locked control. The locked control's value/binding still travel with the
// template (so the dev-set wiring is preserved) but the editor must never offer
// it as an editable field.
func lockedFixtureControls() []core.Control {
	return []core.Control{
		{
			Key:     "product",
			Label:   "Product",
			Kind:    core.ControlText,
			Binding: "plugin.stripe.product",
			Value:   "prod_123",
		},
		{
			Key:     "action",
			Label:   "Checkout endpoint",
			Kind:    core.ControlText,
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

func TestAuthoringControlViewsExcludeLocked(t *testing.T) {
	views := authoringControlViews(core.Page{}, core.Component{}, lockedFixtureControls())
	keys := controlViewKeys(views)
	if len(keys) != 1 || keys[0] != "product" {
		t.Fatalf("authoringControlViews should expose only the editable control, got %#v", keys)
	}
	if views[0]["locked"] != false {
		t.Fatalf("editable control view should expose locked=false, got %#v", views[0]["locked"])
	}
}

func TestAuthoringComponentTemplateViewsExcludeLocked(t *testing.T) {
	templates := []core.ComponentTemplate{{
		Key:           "stripe-buy",
		Label:         "Buy button",
		GoSXComponent: "StripeBuyButton",
		Controls:      lockedFixtureControls(),
	}}
	views := authoringComponentTemplateViews(templates)
	if len(views) != 1 {
		t.Fatalf("expected one template view, got %d", len(views))
	}
	controls := views[0]["controls"].([]map[string]any)
	keys := controlViewKeys(controls)
	if len(keys) != 1 || keys[0] != "product" {
		t.Fatalf("template view should expose only the editable control, got %#v", keys)
	}
	if views[0]["controlCount"] != 2 {
		t.Fatalf("controlCount should reflect the raw template (both controls), got %#v", views[0]["controlCount"])
	}
}
