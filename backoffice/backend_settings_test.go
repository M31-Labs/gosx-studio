package backoffice

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendSettingsPagePreservesSettingsFormContract(t *testing.T) {
	html := gosx.RenderHTML(RenderBackendSettingsPage(BackendSettingsProps{
		RevisionHistory: gosx.El("section", gosx.Attrs(gosx.Attr("data-test-settings-revisions", "true")), gosx.Text("Revision history")),
		SaveAction:      "/admin/settings/__actions/save",
		CSRFToken:       "csrf-token",
		SaveStatus: BackendSettingsStatus{
			OK:      true,
			Message: "Settings saved.",
		},
		RestoreStatus: BackendSettingsStatus{
			Submitted: true,
			Message:   "Restore failed.",
		},
		RevisionRestored: true,
		Settings: BackendSettingsValues{
			SiteTitle:          "Muddy Noni",
			StudioName:         "Studio",
			Tagline:            "Ceramics",
			Description:        "Functional ware",
			OwnerName:          "Owner",
			ContactEmail:       "hello@example.com",
			Instagram:          "@muddy",
			WebsiteURL:         "https://muddy.example",
			PrimaryDomain:      "muddy.example",
			TikTok:             "@muddy-tok",
			NewsletterURL:      "https://news.example",
			MetaTitle:          "Meta title",
			MetaImageURL:       "/media/hero.jpg",
			MetaDescription:    "Meta description",
			MetaImageAlt:       "Hero alt",
			ShippingEnabled:    true,
			LocalPickupEnabled: true,
			LocalPickupCopy:    "Pickup in Honolulu.",
			ShippingCountries:  "US, CA",
			ShippingOptions: []BackendSettingsShippingOption{{
				LabelName:   "shippingOptionLabel0",
				Label:       "Ground",
				AmountName:  "shippingOptionAmount0",
				Amount:      "1200",
				DetailName:  "shippingOptionDetail0",
				Detail:      "3-5 days",
				MinName:     "shippingOptionMin0",
				MinSubtotal: "0",
				MaxName:     "shippingOptionMax0",
				MaxSubtotal: "9999",
			}},
		},
		Notifications: BackendSettingsNotifications{
			StatusLabel: "Ready",
			StatusClass: "status status--ready",
			Summary:     "From studio@example.com to owner@example.com",
		},
		Media: []BackendSettingsMediaAsset{{
			URL:      "/media/hero.jpg",
			Filename: "hero.jpg",
			Alt:      "Hero alt",
		}},
	}))

	for _, fragment := range []string{
		`<div class="admin-page" data-gosx-studio-backend-settings-renderer="gosx-studio">`,
		`<section class="admin-heading"><p class="kicker">CMS</p><h1>Site settings</h1></section>`,
		`<datalist id="settings-media-urls"><option value="/media/hero.jpg" label="hero.jpg" data-media-alt="Hero alt">hero.jpg</option></datalist>`,
		`<form class="panel admin-form" method="post" action="/admin/settings/__actions/save">`,
		`<input type="hidden" name="csrf_token" value="csrf-token" />`,
		`<p class="form-status form-status--ok">Settings saved.</p>`,
		`<p class="form-status form-status--error">Restore failed.</p>`,
		`<p class="form-status form-status--ok">Settings restored.</p>`,
		`<input id="siteTitle" name="siteTitle" value="Muddy Noni" />`,
		`<input id="contactEmail" name="contactEmail" value="hello@example.com" type="email" />`,
		`<legend>Notifications</legend>`,
		`<span class="status status--ready">Ready</span>`,
		`<legend>Public links</legend>`,
		`<input id="websiteUrl" name="websiteUrl" value="https://muddy.example" />`,
		`<legend>SEO</legend>`,
		`<input id="metaImageUrl" name="metaImageUrl" list="settings-media-urls" value="/media/hero.jpg" data-media-alt-target="metaImageAlt" />`,
		`<input type="checkbox" name="shippingEnabled" checked />`,
		`<input type="hidden" name="shippingEnabled" value="off" />`,
		`<input type="checkbox" name="localPickupEnabled" checked />`,
		`<input type="hidden" name="localPickupEnabled" value="off" />`,
		`<textarea id="localPickupCopy" name="localPickupCopy" rows="3">Pickup in Honolulu.</textarea>`,
		`<input id="shippingCountries" name="shippingCountries" value="US, CA" />`,
		`<input name="shippingOptionLabel0" value="Ground" placeholder="Label" />`,
		`<input name="shippingOptionAmount0" value="1200" type="number" min="0" placeholder="Cents" />`,
		`<input name="shippingOptionDetail0" value="3-5 days" placeholder="Detail" />`,
		`<input name="shippingOptionMin0" value="0" type="number" min="0" placeholder="Min subtotal" />`,
		`<input name="shippingOptionMax0" value="9999" type="number" min="0" placeholder="Max subtotal" />`,
		`<button class="button button--primary" type="submit">Save settings</button>`,
		`<section data-test-settings-revisions="true">Revision history</section>`,
		`<script src="/_gosx/studio/media-runtime.js" data-gosx-script="managed" data-gosx-script-load="dom" data-gosx-script-loaded="pending" defer></script>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("settings renderer missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderBackendSettingsSplitNodes(t *testing.T) {
	heading := gosx.RenderHTML(RenderBackendSettingsHeading())
	datalist := gosx.RenderHTML(RenderBackendSettingsMediaDatalist(nil))
	form := gosx.RenderHTML(RenderBackendSettingsForm(BackendSettingsProps{}))

	if !strings.Contains(heading, `<section class="admin-heading">`) || strings.Contains(heading, `<form`) {
		t.Fatalf("settings heading is not split cleanly:\n%s", heading)
	}
	if datalist != `<datalist id="settings-media-urls"></datalist>` {
		t.Fatalf("unexpected empty settings datalist:\n%s", datalist)
	}
	if !strings.Contains(form, `<form class="panel admin-form" method="post" action="">`) || strings.Contains(form, `data-gosx-studio-backend-settings-renderer`) {
		t.Fatalf("settings form is not split cleanly:\n%s", form)
	}
	if strings.Contains(form, `/media-picker.js`) || strings.Contains(form, `data-test-settings-revisions`) {
		t.Fatalf("settings form should not include page-level slots/scripts:\n%s", form)
	}
}
