package studio

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
	b.WriteString(`<h1 data-studio-field="home.hero.headline" data-studio-editable="text" contenteditable="true">` + html.EscapeString(headline) + `</h1>`)
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
		b.WriteString(`<h1 style="margin:0;font-size:42px;line-height:1.02;" data-studio-field="pages.` + html.EscapeString(pageKey) + `.title" data-studio-field-label="Title" data-studio-editable="text" contenteditable="true" data-studio-operation="update-page" data-studio-page-key="` + html.EscapeString(pageKey) + `" data-studio-page-route="` + html.EscapeString(route) + `">` + html.EscapeString(title) + `</h1>`)
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
