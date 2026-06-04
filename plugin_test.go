package studio

import "testing"

// stripeFixturePlugin builds a minimal compiled-in plugin whose single template
// carries one editor-configurable control and one dev-locked control. It mirrors
// the surfacing shape later tasks use to wire the real Stripe plugin.
func stripeFixturePlugin() Plugin {
	return Plugin{
		Key:     "stripe",
		Label:   "Stripe",
		Summary: "Accept payments with Stripe checkout blocks.",
		Templates: []ComponentTemplate{{
			Key:           "stripe-checkout",
			Label:         "Checkout button",
			GoSXComponent: "StripeCheckoutButton",
			Source:        ComponentSourcePlugin,
			Controls: []Control{
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
			},
		}},
	}
}

func templateByKey(templates []ComponentTemplate, key string) *ComponentTemplate {
	for i := range templates {
		if templates[i].Key == key {
			return &templates[i]
		}
	}
	return nil
}

func TestRegisterPluginAddsTemplateToLibrary(t *testing.T) {
	siteMap := SiteMap{}
	siteMap.Library.RegisterPlugin(stripeFixturePlugin())

	tmpl := templateByKey(siteMap.Library.ComponentTemplates, "stripe-checkout")
	if tmpl == nil {
		t.Fatalf("RegisterPlugin should add the plugin template to the library, got %#v", siteMap.Library.ComponentTemplates)
	}
	if tmpl.NormalizedSource() != ComponentSourcePlugin {
		t.Fatalf("registered template should normalize to plugin source, got %q", tmpl.NormalizedSource())
	}
	// Registration normalizes but does not drop the dev-locked control: its
	// wiring still travels with the template.
	if len(tmpl.Controls) != 2 {
		t.Fatalf("registered template should retain both controls, got %d: %#v", len(tmpl.Controls), tmpl.Controls)
	}
	var locked *Control
	for i := range tmpl.Controls {
		if tmpl.Controls[i].Key == "action" {
			locked = &tmpl.Controls[i]
		}
	}
	if locked == nil || !locked.Locked || locked.Binding != "plugin.stripe.checkout" {
		t.Fatalf("locked control wiring should survive registration, got %#v", locked)
	}
}

func TestRegisterPluginSurfacesTemplateInPalette(t *testing.T) {
	siteMap := SiteMap{}
	siteMap.Library.RegisterPlugin(stripeFixturePlugin())

	palette := authoringSiteMapComponentTemplateViews(siteMap.Library.ComponentTemplates, SiteMapViewOptions{})
	var view map[string]any
	for _, candidate := range palette {
		if candidate["key"] == "stripe-checkout" {
			view = candidate
		}
	}
	if view == nil {
		t.Fatalf("registered plugin template should surface in the editor palette, got %#v", palette)
	}
	// Composes with M6-1: the palette view exposes only the non-locked control,
	// while the controlCount/label still reflects the raw template.
	controls := view["controls"].([]map[string]any)
	keys := controlViewKeys(controls)
	if len(keys) != 1 || keys[0] != "product" {
		t.Fatalf("palette view should expose only the editable control, got %#v", keys)
	}
	if view["controlLabel"] != "2 fields" {
		t.Fatalf("controlLabel should reflect the raw template (both controls), got %#v", view["controlLabel"])
	}
}

func TestRegisterPluginDedupesByTemplateKey(t *testing.T) {
	siteMap := SiteMap{}
	siteMap.Library.RegisterPlugin(stripeFixturePlugin())
	siteMap.Library.RegisterPlugin(stripeFixturePlugin())

	count := 0
	for _, tmpl := range siteMap.Library.ComponentTemplates {
		if tmpl.Key == "stripe-checkout" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("re-registering the same plugin should not duplicate the template, got %d copies", count)
	}
}

func TestRegisterPluginReplacesTemplateByKey(t *testing.T) {
	siteMap := SiteMap{}
	siteMap.Library.RegisterPlugin(stripeFixturePlugin())

	updated := stripeFixturePlugin()
	updated.Templates[0].Label = "Pay now"
	siteMap.Library.RegisterPlugin(updated)

	tmpl := templateByKey(siteMap.Library.ComponentTemplates, "stripe-checkout")
	if tmpl == nil {
		t.Fatalf("template should still be present after re-registration, got %#v", siteMap.Library.ComponentTemplates)
	}
	if tmpl.Label != "Pay now" {
		t.Fatalf("later registration of the same key should replace the template, got label %q", tmpl.Label)
	}
}

func TestRegisterPluginPreservesExistingTemplates(t *testing.T) {
	siteMap := SiteMap{Library: CompositionLibrary{ComponentTemplates: []ComponentTemplate{{
		Key:           "hero",
		Label:         "Hero",
		GoSXComponent: "HeroSection",
	}}}}
	siteMap.Library.RegisterPlugin(stripeFixturePlugin())

	if templateByKey(siteMap.Library.ComponentTemplates, "hero") == nil {
		t.Fatalf("existing templates should be preserved, got %#v", siteMap.Library.ComponentTemplates)
	}
	if templateByKey(siteMap.Library.ComponentTemplates, "stripe-checkout") == nil {
		t.Fatalf("plugin template should be appended, got %#v", siteMap.Library.ComponentTemplates)
	}
}
