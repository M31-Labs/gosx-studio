package backoffice

import "m31labs.dev/gosx"

type BackendSettingsProps struct {
	Class            string
	RevisionHistory  gosx.Node
	SaveAction       string
	CSRFToken        string
	SaveStatus       BackendSettingsStatus
	RestoreStatus    BackendSettingsStatus
	RevisionRestored bool
	Settings         BackendSettingsValues
	Notifications    BackendSettingsNotifications
	Media            []BackendSettingsMediaAsset
}

type BackendSettingsStatus struct {
	Submitted bool
	OK        bool
	Message   string
}

type BackendSettingsValues struct {
	SiteTitle          string
	StudioName         string
	Tagline            string
	Description        string
	OwnerName          string
	ContactEmail       string
	Instagram          string
	WebsiteURL         string
	PrimaryDomain      string
	TikTok             string
	NewsletterURL      string
	MetaTitle          string
	MetaImageURL       string
	MetaDescription    string
	MetaImageAlt       string
	ShippingEnabled    bool
	LocalPickupEnabled bool
	LocalPickupCopy    string
	ShippingCountries  string
	ShippingOptions    []BackendSettingsShippingOption
}

type BackendSettingsNotifications struct {
	StatusLabel string
	StatusClass string
	Summary     string
}

type BackendSettingsMediaAsset struct {
	URL      string
	Filename string
	Alt      string
}

type BackendSettingsShippingOption struct {
	LabelName   string
	Label       string
	AmountName  string
	Amount      string
	DetailName  string
	Detail      string
	MinName     string
	MinSubtotal string
	MaxName     string
	MaxSubtotal string
}

func RenderBackendSettingsPage(props BackendSettingsProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-settings-renderer", "gosx-studio"),
	),
		RenderBackendSettingsContent(props),
	)
}

func RenderBackendSettingsContent(props BackendSettingsProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendSettingsHeading(),
		RenderBackendSettingsMediaDatalist(props.Media),
		RenderBackendSettingsForm(props),
		props.RevisionHistory,
		RenderBackendSettingsScripts(),
	)
}

func RenderBackendSettingsScripts() gosx.Node {
	return gosx.El("script", gosx.Attrs(gosx.Attr("src", "/media-picker.js"), gosx.Attr("defer", true)))
}

func RenderBackendSettingsHeading() gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("CMS")),
		gosx.El("h1", nil, gosx.Text("Site settings")),
	)
}

func RenderBackendSettingsMediaDatalist(media []BackendSettingsMediaAsset) gosx.Node {
	nodes := make([]gosx.Node, 0, len(media))
	for _, asset := range media {
		nodes = append(nodes, gosx.El("option", gosx.Attrs(
			gosx.Attr("value", asset.URL),
			gosx.Attr("label", asset.Filename),
			gosx.Attr("data-media-alt", asset.Alt),
		), gosx.Text(asset.Filename)))
	}
	return gosx.El("datalist", gosx.Attrs(gosx.Attr("id", "settings-media-urls")), gosx.Fragment(nodes...))
}

func RenderBackendSettingsForm(props BackendSettingsProps) gosx.Node {
	settings := props.Settings
	return gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "panel admin-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", props.SaveAction),
	),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "csrf_token"), gosx.Attr("value", props.CSRFToken))),
		gosx.Fragment(renderBackendSettingsStatuses(props)...),
		renderBackendSettingsInput("siteTitle", "Site title", settings.SiteTitle, ""),
		renderBackendSettingsInput("studioName", "Studio name", settings.StudioName, ""),
		renderBackendSettingsInput("tagline", "Tagline", settings.Tagline, "field-row field-row--wide"),
		renderBackendSettingsTextarea("description", "Description", settings.Description, "4", "field-row field-row--wide"),
		renderBackendSettingsInput("ownerName", "Owner", settings.OwnerName, ""),
		renderBackendSettingsInput("contactEmail", "Contact email", settings.ContactEmail, "", gosx.Attr("type", "email")),
		renderBackendSettingsNotifications(props.Notifications),
		renderBackendSettingsInput("instagram", "Instagram", settings.Instagram, ""),
		renderBackendSettingsPublicLinks(settings),
		renderBackendSettingsSEO(settings),
		renderBackendSettingsShippingToggles(settings),
		renderBackendSettingsTextarea("localPickupCopy", "Local pickup copy", settings.LocalPickupCopy, "3", "field-row field-row--wide"),
		renderBackendSettingsInput("shippingCountries", "Allowed shipping countries", settings.ShippingCountries, "field-row field-row--wide"),
		renderBackendSettingsShippingOptions(settings.ShippingOptions),
		gosx.El("button", gosx.Attrs(gosx.Attr("class", "button button--primary"), gosx.Attr("type", "submit")), gosx.Text("Save settings")),
	)
}

func renderBackendSettingsStatuses(props BackendSettingsProps) []gosx.Node {
	nodes := []gosx.Node{}
	if props.SaveStatus.OK {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--ok")), gosx.Text(props.SaveStatus.Message)))
	}
	if props.RestoreStatus.Submitted {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--error")), gosx.Text(props.RestoreStatus.Message)))
	}
	if props.RevisionRestored {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "form-status form-status--ok")), gosx.Text("Settings restored.")))
	}
	return nodes
}

func renderBackendSettingsInput(id, label, value, className string, extraAttrs ...any) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	attrs := []any{gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("value", value)}
	attrs = append(attrs, extraAttrs...)
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("input", gosx.Attrs(attrs...)),
	)
}

func renderBackendSettingsTextarea(id, label, value, rows, className string) gosx.Node {
	if className == "" {
		className = "field-row"
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", id)), gosx.Text(label)),
		gosx.El("textarea", gosx.Attrs(gosx.Attr("id", id), gosx.Attr("name", id), gosx.Attr("rows", rows)), gosx.Text(value)),
	)
}

func renderBackendSettingsNotifications(notifications BackendSettingsNotifications) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("Notifications")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list field-list--stacked field-row--wide")),
			gosx.El("li", nil,
				gosx.El("strong", nil, gosx.Text("Email delivery")),
				gosx.El("span", gosx.Attrs(gosx.Attr("class", notifications.StatusClass)), gosx.Text(notifications.StatusLabel)),
				gosx.El("p", nil, gosx.Text(notifications.Summary)),
			),
		),
	)
}

func renderBackendSettingsPublicLinks(settings BackendSettingsValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("Public links")),
		renderBackendSettingsInput("websiteUrl", "Website URL", settings.WebsiteURL, ""),
		renderBackendSettingsInput("primaryDomain", "Primary domain", settings.PrimaryDomain, ""),
		renderBackendSettingsInput("tiktok", "TikTok", settings.TikTok, ""),
		renderBackendSettingsInput("newsletterUrl", "Newsletter URL", settings.NewsletterURL, ""),
	)
}

func renderBackendSettingsSEO(settings BackendSettingsValues) gosx.Node {
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", "form-section")),
		gosx.El("legend", nil, gosx.Text("SEO")),
		renderBackendSettingsInput("metaTitle", "Meta title", settings.MetaTitle, ""),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "metaImageUrl")), gosx.Text("Meta image URL")),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "metaImageUrl"),
				gosx.Attr("name", "metaImageUrl"),
				gosx.Attr("list", "settings-media-urls"),
				gosx.Attr("value", settings.MetaImageURL),
				gosx.Attr("data-media-alt-target", "metaImageAlt"),
			)),
		),
		renderBackendSettingsTextarea("metaDescription", "Meta description", settings.MetaDescription, "3", "field-row field-row--wide"),
		renderBackendSettingsInput("metaImageAlt", "Meta image alt text", settings.MetaImageAlt, "field-row field-row--wide"),
	)
}

func renderBackendSettingsShippingToggles(settings BackendSettingsValues) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "check-row")),
		gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "shippingEnabled"), gosx.Attr("checked", settings.ShippingEnabled))),
			gosx.Text(" Enable shipping at checkout"),
		),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "shippingEnabled"), gosx.Attr("value", "off"))),
		gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "localPickupEnabled"), gosx.Attr("checked", settings.LocalPickupEnabled))),
			gosx.Text(" Offer local pickup"),
		),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "localPickupEnabled"), gosx.Attr("value", "off"))),
	)
}

func renderBackendSettingsShippingOptions(options []BackendSettingsShippingOption) gosx.Node {
	nodes := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		nodes = append(nodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "shipping-option-row")),
			gosx.El("input", gosx.Attrs(gosx.Attr("name", option.LabelName), gosx.Attr("value", option.Label), gosx.Attr("placeholder", "Label"))),
			gosx.El("input", gosx.Attrs(gosx.Attr("name", option.AmountName), gosx.Attr("value", option.Amount), gosx.Attr("type", "number"), gosx.Attr("min", "0"), gosx.Attr("placeholder", "Cents"))),
			gosx.El("input", gosx.Attrs(gosx.Attr("name", option.DetailName), gosx.Attr("value", option.Detail), gosx.Attr("placeholder", "Detail"))),
			gosx.El("input", gosx.Attrs(gosx.Attr("name", option.MinName), gosx.Attr("value", option.MinSubtotal), gosx.Attr("type", "number"), gosx.Attr("min", "0"), gosx.Attr("placeholder", "Min subtotal"))),
			gosx.El("input", gosx.Attrs(gosx.Attr("name", option.MaxName), gosx.Attr("value", option.MaxSubtotal), gosx.Attr("type", "number"), gosx.Attr("min", "0"), gosx.Attr("placeholder", "Max subtotal"))),
		))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")),
		gosx.El("label", nil, gosx.Text("Shipping options")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "shipping-option-grid")), gosx.Fragment(nodes...)),
	)
}
