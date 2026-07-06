package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderAdvancedPanelFull(t *testing.T) {
	html := gosx.RenderHTML(RenderAdvancedPanel(advancedPanelTestView(), AdvancedPanelOptions{
		GroupNodes: map[string]gosx.Node{
			"flows":      gosx.El("section", gosx.Attrs(gosx.Attr("data-flow-placeholder", "true")), gosx.Text("Flow designer")),
			"tools":      gosx.El("section", gosx.Attrs(gosx.Attr("data-tools-placeholder", "true")), gosx.Text("Tools")),
			"schema":     gosx.El("section", gosx.Attrs(gosx.Attr("data-schema-placeholder", "true")), gosx.Text("Schema")),
			"schedule":   gosx.El("section", gosx.Attrs(gosx.Attr("data-schedule-placeholder", "true")), gosx.Text("Schedule")),
			"typography": gosx.El("section", gosx.Attrs(gosx.Attr("data-type-placeholder", "true")), gosx.Text("Type")),
			"settings":   gosx.El("section", gosx.Attrs(gosx.Attr("data-settings-placeholder", "true")), gosx.Text("Settings")),
		},
	}))

	for _, fragment := range []string{
		`<section class="editor-panel editor-panel--advanced-studio studio-advanced-panel" data-studio-advanced-panel="true" data-studio-mode-panel="advanced" data-studio-panel="advanced" data-studio-engine-source="gosx" data-studio-advanced-group-active="flows" data-gosx-studio-advanced-panel-renderer="gosx-studio">`,
		`<header class="studio-advanced-panel__head"><div><p class="kicker">Advanced</p><h2>Tool drawer</h2><p>Keep tools grouped.</p></div></header>`,
		`<input class="studio-advanced-panel__group-input" id="studioAdvancedGroupFlows" type="radio" name="studioAdvancedGroup" value="flows" checked aria-label="Flows" />`,
		`<label class="studio-advanced-panel__group" for="studioAdvancedGroupTypography" data-studio-advanced-group-label="typography"><strong>Fonts</strong><small>Fonts and custom CSS.</small></label>`,
		`<label class="studio-advanced-panel__group" for="studioAdvancedGroupSettings" data-studio-advanced-group-label="settings"><strong>SEO</strong><small>SEO, domains, and integrations.</small></label>`,
		`<section class="studio-advanced-panel__group-slot" data-studio-advanced-group-slot="flows" data-studio-advanced-group-selected="true">`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="flows"><section data-flow-placeholder="true">Flow designer</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="tools"><section data-tools-placeholder="true">Tools</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="schema"><section data-schema-placeholder="true">Schema</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="schedule"><section data-schedule-placeholder="true">Schedule</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="typography"><section data-type-placeholder="true">Type</section></div>`,
		`<div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="settings"><section data-settings-placeholder="true">Settings</section></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("advanced panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(html, notWant) {
			t.Fatalf("advanced panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderAdvancedPanelSegmentsKeepHostedChildrenInsideRoot(t *testing.T) {
	segments := RenderAdvancedPanelSegments(advancedPanelTestView(), AdvancedPanelOptions{})
	html := gosx.RenderHTML(gosx.Fragment(
		segments.RootOpen,
		segments.Controls,
		segments.FlowsSlotOpen,
		gosx.El("section", gosx.Attrs(gosx.Attr("data-flow-placeholder", "true")), gosx.Text("Flow designer")),
		segments.FlowsSlotClose,
		segments.ToolsSlotOpen,
		gosx.El("section", gosx.Attrs(gosx.Attr("data-tools-placeholder", "true")), gosx.Text("Tools")),
		segments.ToolsSlotClose,
		segments.SchemaSlotOpen,
		gosx.El("section", gosx.Attrs(gosx.Attr("data-schema-placeholder", "true")), gosx.Text("Schema")),
		segments.SchemaSlotClose,
		segments.ScheduleSlotOpen,
		gosx.El("section", gosx.Attrs(gosx.Attr("data-schedule-placeholder", "true")), gosx.Text("Schedule")),
		segments.ScheduleSlotClose,
		segments.TypographySlotOpen,
		gosx.El("section", gosx.Attrs(gosx.Attr("data-type-placeholder", "true")), gosx.Text("Type")),
		segments.TypographySlotClose,
		segments.SettingsSlotOpen,
		gosx.El("section", gosx.Attrs(gosx.Attr("data-settings-placeholder", "true")), gosx.Text("Settings")),
		segments.SettingsSlotClose,
		segments.RootClose,
	))

	for _, fragment := range []string{
		`data-gosx-studio-advanced-panel-renderer="gosx-studio"`,
		`class="studio-advanced-panel__group-input"`,
		`<section class="studio-advanced-panel__group-slot" data-studio-advanced-group-slot="flows" data-studio-advanced-group-selected="true"><header><h3>Flows</h3><p>Contact and request forms.</p></header><div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="flows"><section data-flow-placeholder="true">Flow designer</section></div></section>`,
		`<section class="studio-advanced-panel__group-slot" data-studio-advanced-group-slot="tools" data-studio-advanced-group-selected="false"><header><h3>Tools</h3><p>Back-office destinations.</p></header><div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="tools"><section data-tools-placeholder="true">Tools</section></div></section>`,
		`<section class="studio-advanced-panel__group-slot" data-studio-advanced-group-slot="settings" data-studio-advanced-group-selected="false"><header><h3>SEO</h3><p>SEO, domains, and integrations.</p></header><div class="studio-advanced-panel__group-body" data-studio-advanced-group-body="settings"><section data-settings-placeholder="true">Settings</section></div></section>`,
		`</section>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("advanced panel segments missing %q:\n%s", fragment, html)
		}
	}

	rootIndex := strings.Index(html, `data-gosx-studio-advanced-panel-renderer="gosx-studio"`)
	inputIndex := strings.Index(html, `class="studio-advanced-panel__group-input"`)
	flowIndex := strings.Index(html, `data-flow-placeholder="true"`)
	rootCloseIndex := strings.LastIndex(html, `</section>`)
	if rootIndex < 0 || inputIndex < 0 || flowIndex < 0 || rootCloseIndex < 0 || !(rootIndex < inputIndex && inputIndex < flowIndex && flowIndex < rootCloseIndex) {
		t.Fatalf("advanced panel segment order should keep flow placeholder inside root after group inputs:\n%s", html)
	}
}

func TestRenderAdvancedPanelRootAttrOverride(t *testing.T) {
	html := gosx.RenderHTML(RenderAdvancedPanel(advancedPanelTestView(), AdvancedPanelOptions{
		RootAttrs: map[string]any{
			"class":                   "custom-advanced-panel",
			"data-studio-panel":       "custom",
			"data-custom-advanced":    true,
			"aria-label":              "Advanced override",
			"data-studio-extra-panel": "advanced-test",
		},
	}))

	for _, fragment := range []string{
		`class="custom-advanced-panel"`,
		`data-studio-panel="custom"`,
		`data-custom-advanced="true"`,
		`aria-label="Advanced override"`,
		`data-studio-extra-panel="advanced-test"`,
		`data-gosx-studio-advanced-panel-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("advanced root attr override missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderAdvancedToolsPanelPopulatedAndEmpty(t *testing.T) {
	populated := gosx.RenderHTML(RenderAdvancedToolsPanel(advancedToolsPanelTestView(true), AdvancedToolsPanelOptions{}))
	for _, fragment := range []string{
		`data-gosx-studio-advanced-tools-panel-renderer="gosx-studio"`,
		`<section class="studio-resource-adapters" data-studio-resource-adapters="true">`,
		`<article class="studio-resource-adapter-card" data-studio-resource-adapter="cms" data-studio-resource-surface="media">`,
		`<small>Uploads</small>`,
		`<div class="editor-tools-grid"><a class="editor-tool-card" data-gosx-link="true" href="/admin/media"><strong>Media</strong><span>Manage uploads.</span></a></div>`,
	} {
		if !strings.Contains(populated, fragment) {
			t.Fatalf("populated advanced tools panel missing %q:\n%s", fragment, populated)
		}
	}

	empty := gosx.RenderHTML(RenderAdvancedToolsPanel(advancedToolsPanelTestView(false), AdvancedToolsPanelOptions{}))
	for _, fragment := range []string{
		`data-gosx-studio-advanced-tools-panel-renderer="gosx-studio"`,
		`<p class="empty">No resource adapters are connected.</p>`,
		`<p class="empty">No tools are available.</p>`,
	} {
		if !strings.Contains(empty, fragment) {
			t.Fatalf("empty advanced tools panel missing %q:\n%s", fragment, empty)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(populated, notWant) || strings.Contains(empty, notWant) {
			t.Fatalf("advanced tools panel must not include %q:\npopulated=%s\nempty=%s", notWant, populated, empty)
		}
	}
}

func TestRenderAdvancedSettingsPanelPopulatedAndEmpty(t *testing.T) {
	populatedView := map[string]any{
		"class":           "editor-panel studio-advanced-settings-panel",
		"key":             "settings",
		"mode":            "advanced",
		"kicker":          "Advanced",
		"title":           "SEO and integrations",
		"summary":         "Search metadata and connected destinations.",
		"empty":           "No settings fields are registered.",
		"hasFields":       true,
		"fieldList":       gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-advanced-settings-panel__fields")), gosx.Text("Fields")),
		"statusGridClass": "studio-advanced-settings-panel__status-grid",
		"statusCards": []map[string]any{{
			"key":     "redirects",
			"class":   "studio-advanced-settings-panel__status-card is-disabled",
			"title":   "Redirects",
			"summary": "No redirect store is connected.",
			"status":  "Unavailable",
		}},
	}
	populated := gosx.RenderHTML(RenderAdvancedSettingsPanel(populatedView, AdvancedSettingsPanelOptions{}))
	for _, fragment := range []string{
		`data-gosx-studio-advanced-settings-panel-renderer="gosx-studio"`,
		`<section class="editor-panel studio-advanced-settings-panel" data-panel-key="settings" data-studio-panel="settings" data-studio-mode-panel="advanced" data-studio-engine-source="gosx" data-gosx-studio-advanced-settings-panel-renderer="gosx-studio">`,
		`<p class="studio-advanced-settings-panel__summary">Search metadata and connected destinations.</p>`,
		`<div class="studio-advanced-settings-panel__fields">Fields</div>`,
		`data-studio-advanced-settings-status-grid="true"`,
		`data-studio-advanced-settings-status="redirects"`,
		`<small>Unavailable</small>`,
	} {
		if !strings.Contains(populated, fragment) {
			t.Fatalf("populated advanced settings panel missing %q:\n%s", fragment, populated)
		}
	}

	emptyView := map[string]any{
		"class": "editor-panel studio-advanced-settings-panel",
		"key":   "settings",
		"mode":  "advanced",
		"title": "SEO and integrations",
		"empty": "No settings fields are registered.",
	}
	empty := gosx.RenderHTML(RenderAdvancedSettingsPanel(emptyView, AdvancedSettingsPanelOptions{}))
	for _, fragment := range []string{
		`data-gosx-studio-advanced-settings-panel-renderer="gosx-studio"`,
		`<p class="empty">No settings fields are registered.</p>`,
	} {
		if !strings.Contains(empty, fragment) {
			t.Fatalf("empty advanced settings panel missing %q:\n%s", fragment, empty)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(populated, notWant) || strings.Contains(empty, notWant) {
			t.Fatalf("advanced settings panel must not include %q:\npopulated=%s\nempty=%s", notWant, populated, empty)
		}
	}
}

func TestRenderAdvancedFieldPanelPopulatedAndEmpty(t *testing.T) {
	populatedView := advancedFieldPanelTestView(true)
	populated := gosx.RenderHTML(RenderAdvancedFieldPanel(populatedView, AdvancedFieldPanelOptions{}))
	for _, fragment := range []string{
		`data-gosx-studio-advanced-field-panel-renderer="gosx-studio"`,
		`<section class="editor-panel editor-panel--workspace studio-advanced-fields-panel" data-panel-key="workspace-fields" data-studio-panel="workspace-fields" data-studio-mode-panel="advanced" data-studio-engine-source="gosx" data-gosx-studio-advanced-field-panel-renderer="gosx-studio">`,
		`<div class="studio-advanced-fields-panel__fields">Fields</div>`,
	} {
		if !strings.Contains(populated, fragment) {
			t.Fatalf("populated advanced field panel missing %q:\n%s", fragment, populated)
		}
	}

	emptyView := advancedFieldPanelTestView(false)
	empty := gosx.RenderHTML(RenderAdvancedFieldPanel(emptyView, AdvancedFieldPanelOptions{}))
	for _, fragment := range []string{
		`data-gosx-studio-advanced-field-panel-renderer="gosx-studio"`,
		`<p class="empty">No workspace fields are registered.</p>`,
	} {
		if !strings.Contains(empty, fragment) {
			t.Fatalf("empty advanced field panel missing %q:\n%s", fragment, empty)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(populated, notWant) || strings.Contains(empty, notWant) {
			t.Fatalf("advanced field panel must not include %q:\npopulated=%s\nempty=%s", notWant, populated, empty)
		}
	}
}

func advancedPanelTestView() map[string]any {
	return map[string]any{
		"kicker":          "Advanced",
		"title":           "Tool drawer",
		"summary":         "Keep tools grouped.",
		"groupLabel":      "Advanced groups",
		"defaultGroupKey": "flows",
		"groups": []map[string]any{
			{"key": "flows", "inputID": "studioAdvancedGroupFlows", "label": "Flows", "summary": "Contact and request forms.", "selected": true},
			{"key": "tools", "inputID": "studioAdvancedGroupTools", "label": "Tools", "summary": "Back-office destinations."},
			{"key": "schema", "inputID": "studioAdvancedGroupSchema", "label": "Fields", "summary": "Workspace field details."},
			{"key": "schedule", "inputID": "studioAdvancedGroupSchedule", "label": "Schedule", "summary": "Calendar availability details."},
			{"key": "typography", "inputID": "studioAdvancedGroupTypography", "label": "Fonts", "summary": "Fonts and custom CSS."},
			{"key": "settings", "inputID": "studioAdvancedGroupSettings", "label": "SEO", "summary": "SEO, domains, and integrations."},
		},
	}
}

func advancedToolsPanelTestView(populated bool) map[string]any {
	view := map[string]any{
		"class":             "editor-panel editor-panel--tools",
		"key":               "tools",
		"mode":              "advanced",
		"kicker":            "Advanced",
		"title":             "Tools",
		"resourcesTitle":    "Resource adapters",
		"resourcesSummary":  "Connected surfaces.",
		"resourceEmpty":     "No resource adapters are connected.",
		"resourceGridClass": "studio-resource-adapters__grid",
		"empty":             "No tools are available.",
		"gridClass":         "editor-tools-grid",
		"hasAdapters":       populated,
		"hasItems":          populated,
	}
	if populated {
		view["adapters"] = []map[string]any{{
			"kind":            "cms",
			"surface":         "media",
			"label":           "Media",
			"summary":         "Upload surface.",
			"capabilityLabel": "Read/write",
			"bindingLabel":    "Uploads",
		}}
		view["items"] = []map[string]any{{
			"label":   "Media",
			"summary": "Manage uploads.",
			"attrs": map[string]any{
				"class":          "editor-tool-card",
				"href":           "/admin/media",
				"data-gosx-link": true,
			},
		}}
	}
	return view
}

func advancedFieldPanelTestView(populated bool) map[string]any {
	view := map[string]any{
		"class":     "editor-panel editor-panel--workspace studio-advanced-fields-panel",
		"key":       "workspace-fields",
		"mode":      "advanced",
		"kicker":    "Advanced",
		"title":     "Workspace fields",
		"empty":     "No workspace fields are registered.",
		"hasFields": populated,
	}
	if populated {
		view["fieldList"] = gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-advanced-fields-panel__fields")), gosx.Text("Fields"))
	}
	return view
}
