package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderNavigationPanelPopulated(t *testing.T) {
	view := map[string]any{
		"class":       "panel editor-panel editor-panel--navigation studio-navigation-panel",
		"panelKey":    "navigation",
		"headerClass": "panel__header",
		"kicker":      "Navigation",
		"title":       "Site navigation",
		"help":        "Relabel, hide, or reorder the storefront menu.",
		"empty":       "Navigation is derived from your published content.",
		"hasItems":    true,
		"action":      "/admin/editor/__actions/navigation",
		"buttonLabel": "Save navigation",
		"formID":      "websiteEditorForm",
		"items": []map[string]any{
			{
				"key":            "nav-0",
				"label":          "Home",
				"href":           "/",
				"enabled":        true,
				"order":          1,
				"labelName":      "navigationLabel0",
				"labelID":        "navigationLabel0",
				"hrefName":       "navigationHref0",
				"orderName":      "navigationOrder0",
				"enabledName":    "navigationEnabled0",
				"enabledValue":   "true",
				"positionLabel":  "Position",
				"labelFieldText": "Label",
				"linkLabel":      "/",
				"showLabel":      "Show in navigation",
			},
			{
				"key":            "nav-1",
				"label":          "Shop",
				"href":           "/shop",
				"enabled":        false,
				"order":          2,
				"labelName":      "navigationLabel1",
				"labelID":        "navigationLabel1",
				"hrefName":       "navigationHref1",
				"orderName":      "navigationOrder1",
				"enabledName":    "navigationEnabled1",
				"enabledValue":   "true",
				"positionLabel":  "Position",
				"labelFieldText": "Label",
				"linkLabel":      "/shop",
				"showLabel":      "Show in navigation",
			},
		},
	}

	html := gosx.RenderHTML(RenderNavigationPanel(view, NavigationPanelOptions{}))

	for _, fragment := range []string{
		`<section class="panel editor-panel editor-panel--navigation studio-navigation-panel" data-panel-key="navigation" data-studio-navigation-panel="true" data-gosx-studio-navigation-panel-renderer="gosx-studio">`,
		`<div class="panel__header">`,
		`<p class="kicker">Navigation</p>`,
		`<h2>Site navigation</h2>`,
		`<p class="field-help">Relabel, hide, or reorder the storefront menu.</p>`,
		`<p class="empty" hidden>Navigation is derived from your published content.</p>`,
		`<div class="inline-form studio-navigation-panel__form">`,
		`<li class="studio-navigation-panel__row" data-studio-navigation-row="nav-0">`,
		`<li class="studio-navigation-panel__row" data-studio-navigation-row="nav-1">`,
		`<label for="navigationLabel0">Label</label>`,
		`<input id="navigationLabel0" name="navigationLabel0" type="text" value="Home" />`,
		`<small class="field-help">/shop</small>`,
		`<input type="hidden" name="navigationHref1" value="/shop" />`,
		`<input name="navigationEnabled0" type="checkbox" value="true" checked />`,
		`<input name="navigationEnabled1" type="checkbox" value="true" />`,
		`<input name="navigationOrder1" type="number" min="1" value="2" />`,
		`<button class="button button--primary" type="submit" form="websiteEditorForm" formaction="/admin/editor/__actions/navigation" formmethod="post" data-studio-submit-action="navigation" data-studio-field-action-formaction="/admin/editor/__actions/navigation">Save navigation</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered navigation panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		"<form",
		"GoSX Studio",
		`<div class="inline-form studio-navigation-panel__form" hidden>`,
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("rendered navigation panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderNavigationPanelEmpty(t *testing.T) {
	view := map[string]any{
		"class":       "panel editor-panel editor-panel--navigation studio-navigation-panel",
		"panelKey":    "navigation",
		"headerClass": "panel__header",
		"kicker":      "Navigation",
		"title":       "Site navigation",
		"help":        "Relabel, hide, or reorder the storefront menu.",
		"empty":       "Navigation is derived from your published content.",
		"action":      "/admin/editor/__actions/navigation",
		"buttonLabel": "Save navigation",
		"formID":      "websiteEditorForm",
	}

	html := gosx.RenderHTML(RenderNavigationPanel(view, NavigationPanelOptions{}))

	for _, fragment := range []string{
		`data-studio-navigation-panel="true"`,
		`data-gosx-studio-navigation-panel-renderer="gosx-studio"`,
		`<p class="empty">Navigation is derived from your published content.</p>`,
		`<div class="inline-form studio-navigation-panel__form" hidden>`,
		`<button class="button button--primary" type="submit" form="websiteEditorForm" formaction="/admin/editor/__actions/navigation" formmethod="post" data-studio-submit-action="navigation" data-studio-field-action-formaction="/admin/editor/__actions/navigation">Save navigation</button>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("empty navigation panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{
		`data-studio-navigation-row=`,
		"<form",
		"GoSX Studio",
	} {
		if strings.Contains(html, notWant) {
			t.Fatalf("empty navigation panel must not include %q:\n%s", notWant, html)
		}
	}
}
