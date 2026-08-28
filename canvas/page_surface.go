package canvas

import (
	"html"
	"strings"
)

type PageSurfaceTheme struct {
	Class string
	CSS   string
}

type HomePageSurfaceInput struct {
	Key          string
	PaletteClass string
	CSS          string
	Tagline      string
	Headline     string
	Subhead      string
	CTALabel     string
	CTAURL       string
	// DurableHeadlineField, when non-empty, opts the hero headline's
	// contenteditable surface into the durable operation protocol
	// (handoff-31): it is the exact authoring.OperationTarget.Field literal
	// a host's durable set-field ledger already uses for this value (e.g.
	// "hero.headline" — see panels/direct_edit_panel.go's own Field for the
	// SAME value), which is not always identical to this surface's
	// data-studio-field canvas binding ("home.hero.headline"). Left blank,
	// the headline keeps its pre-existing legacy save-control-only behavior
	// unchanged — this is strictly additive/opt-in.
	DurableHeadlineField string
	// DurableHeadlinePageID/-Route/-Component are the rest of the durable
	// OperationTarget address; required together with DurableHeadlineField
	// (the canvas binding's own "home"/"pages" prefix conventions do not
	// reliably derive a ledger's PageID/ComponentKey convention, so a host
	// must supply them explicitly rather than have this package guess).
	DurableHeadlinePageID    string
	DurableHeadlineRoute     string
	DurableHeadlineComponent string
}

type PageArtboardSurfaceInput struct {
	Key           string
	PageKey       string
	Title         string
	Route         string
	Group         string
	Status        string
	Component     string
	Binding       string
	EditableTitle bool
	PaletteClass  string
	CSS           string
	Components    []PageArtboardComponent
	// DurableTitleField, when non-empty, opts this page's editable title
	// into the durable operation protocol (handoff-31) instead of (never in
	// addition to) the legacy update-page mutation — see
	// inlineeditruntime/island_runtime.js's durable dispatch: a
	// durable-enabled commit never also fires the legacy POST. Only a page
	// whose host has actually wired a durable set-field ledger case for its
	// title (e.g. "pages.about.title") should set this; a page without one
	// must leave it blank and keep using update-page, or a durable submit
	// would be rejected server-side as an unknown target.
	DurableTitleField     string
	DurableTitlePageID    string
	DurableTitleComponent string
}

type PageArtboardComponent struct {
	Label  string
	Detail string
}

func RenderHomePageSurfaceMarkup(input HomePageSurfaceInput) string {
	tagline := strings.TrimSpace(input.Tagline)
	headline := firstNonBlank(input.Headline, "Ceramic pieces with evidence of the hand")
	subhead := strings.TrimSpace(input.Subhead)
	ctaLabel := firstNonBlank(input.CTALabel, "Shop available work")
	ctaURL := firstNonBlank(input.CTAURL, "/shop")

	var b strings.Builder
	b.WriteString(`<div class="` + html.EscapeString(strings.TrimSpace(input.PaletteClass)) + `" data-gosx-html-key="` + html.EscapeString(strings.TrimSpace(input.Key)) + `" data-studio-page-surface="home" style="min-height:100%;">`)
	b.WriteString(`<style>` + input.CSS + `</style>`)
	b.WriteString(`<section class="hero gosx-style-block-home-hero">`)
	b.WriteString(`<div class="hero__copy">`)
	b.WriteString(`<p class="kicker" data-studio-field="tagline" data-studio-editable="text" contenteditable="true">` + html.EscapeString(tagline) + `</p>`)
	b.WriteString(`<h1 data-studio-field="home.hero.headline" data-studio-editable="text" contenteditable="true"` + durableAttrs(input.DurableHeadlineField, input.DurableHeadlinePageID, input.DurableHeadlineRoute, input.DurableHeadlineComponent) + `>` + html.EscapeString(headline) + `</h1>`)
	b.WriteString(`<p class="lede" data-studio-field="home.hero.subhead" data-studio-editable="text" contenteditable="true">` + html.EscapeString(subhead) + `</p>`)
	b.WriteString(`<div class="button-row">`)
	b.WriteString(`<a class="button button--primary" href="` + html.EscapeString(ctaURL) + `" data-studio-field="home.hero.ctaLabel" data-studio-editable="text" contenteditable="true">` + html.EscapeString(ctaLabel) + `</a>`)
	b.WriteString(`</div>`)
	b.WriteString(`</div>`)
	b.WriteString(`</section>`)
	b.WriteString(`</div>`)
	return b.String()
}

func RenderPageArtboardSurfaceMarkup(input PageArtboardSurfaceInput) string {
	pageKey := strings.TrimSpace(input.PageKey)
	title := firstNonBlank(input.Title, pageKey, "Untitled page")
	route := firstNonBlank(input.Route, "/"+pageKey)
	status := strings.TrimSpace(input.Status)
	group := strings.TrimSpace(input.Group)
	component := strings.TrimSpace(input.Component)
	binding := strings.TrimSpace(input.Binding)

	var meta []string
	if route != "" {
		meta = append(meta, route)
	}
	if group != "" {
		meta = append(meta, group)
	}
	if status != "" {
		meta = append(meta, status)
	}

	var b strings.Builder
	b.WriteString(`<div class="` + html.EscapeString(strings.TrimSpace(input.PaletteClass)) + `" data-gosx-html-key="` + html.EscapeString(strings.TrimSpace(input.Key)) + `" data-studio-page-surface="` + html.EscapeString(pageKey) + `" style="min-height:100%;">`)
	b.WriteString(`<style>` + input.CSS + `</style>`)
	b.WriteString(`<article class="studio-page-artboard gosx-style-block-page-artboard" style="min-height:100%;padding:28px;box-sizing:border-box;background:var(--color-canvas,#f8f4ec);color:var(--color-ink,#241b14);display:flex;flex-direction:column;gap:20px;">`)
	b.WriteString(`<header style="display:flex;flex-direction:column;gap:10px;">`)
	b.WriteString(`<p class="kicker" style="margin:0;text-transform:uppercase;letter-spacing:0;font-size:12px;">` + html.EscapeString(strings.Join(meta, " / ")) + `</p>`)
	if input.EditableTitle {
		b.WriteString(`<h1 style="margin:0;font-size:42px;line-height:1.02;" data-studio-field="pages.` + html.EscapeString(pageKey) + `.title" data-studio-field-label="Title" data-studio-editable="text" contenteditable="true" data-studio-operation="update-page" data-studio-page-key="` + html.EscapeString(pageKey) + `" data-studio-page-route="` + html.EscapeString(route) + `"` + durableAttrs(input.DurableTitleField, input.DurableTitlePageID, route, input.DurableTitleComponent) + `>` + html.EscapeString(title) + `</h1>`)
	} else {
		b.WriteString(`<h1 style="margin:0;font-size:42px;line-height:1.02;">` + html.EscapeString(title) + `</h1>`)
	}
	b.WriteString(`</header>`)
	b.WriteString(`<section style="display:grid;gap:12px;grid-template-columns:repeat(2,minmax(0,1fr));">`)
	b.WriteString(renderPageArtboardSurfaceInfo("Route", route))
	b.WriteString(renderPageArtboardSurfaceInfo("Component", component))
	b.WriteString(renderPageArtboardSurfaceInfo("Source", binding))
	b.WriteString(renderPageArtboardSurfaceInfo("Status", status))
	b.WriteString(`</section>`)
	if len(input.Components) > 0 {
		b.WriteString(`<section style="display:flex;flex-direction:column;gap:8px;">`)
		for _, component := range input.Components {
			b.WriteString(renderPageArtboardComponentSummary(component))
		}
		b.WriteString(`</section>`)
	}
	b.WriteString(`<div style="margin-top:auto;border-top:1px solid currentColor;padding-top:16px;opacity:.72;">`)
	b.WriteString(`<p style="margin:0;font-size:14px;line-height:1.45;">This page artboard is mounted as a CanvasBoard HTML surface and keyed to ` + html.EscapeString(strings.TrimSpace(input.Key)) + `.</p>`)
	b.WriteString(`</div>`)
	b.WriteString(`</article>`)
	b.WriteString(`</div>`)
	return b.String()
}

// durableAttrs renders the inlineeditruntime durable-history opt-in
// attributes (see island_runtime.js's DURABLE_ATTR/DURABLE_FIELD_ATTR/etc.)
// for a contenteditable surface field. field being blank means the host has
// not wired a durable ledger counterpart for this value yet — every
// attribute is omitted and the field keeps its pre-existing legacy-only
// behavior unchanged.
func durableAttrs(field, pageID, route, component string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	return ` data-gosx-studio-durable-history="true" data-gosx-studio-durable-field="` + html.EscapeString(field) +
		`" data-gosx-studio-durable-page-id="` + html.EscapeString(strings.TrimSpace(pageID)) +
		`" data-gosx-studio-durable-route="` + html.EscapeString(strings.TrimSpace(route)) +
		`" data-gosx-studio-durable-component="` + html.EscapeString(strings.TrimSpace(component)) + `"`
}

func renderPageArtboardSurfaceInfo(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "Not configured"
	}
	return `<div style="border:1px solid currentColor;padding:12px;min-width:0;"><p style="margin:0 0 4px;font-size:11px;text-transform:uppercase;letter-spacing:0;opacity:.72;">` + html.EscapeString(label) + `</p><p style="margin:0;font-size:15px;overflow-wrap:anywhere;">` + html.EscapeString(value) + `</p></div>`
}

func renderPageArtboardComponentSummary(component PageArtboardComponent) string {
	label := firstNonBlank(component.Label, "Component")
	detail := firstNonBlank(component.Detail, "Not configured")
	return `<div style="border:1px solid currentColor;padding:10px 12px;min-width:0;"><p style="margin:0 0 4px;font-size:13px;line-height:1.25;overflow-wrap:anywhere;">` + html.EscapeString(label) + `</p><p style="margin:0;font-size:12px;line-height:1.35;opacity:.72;overflow-wrap:anywhere;">` + html.EscapeString(detail) + `</p></div>`
}
