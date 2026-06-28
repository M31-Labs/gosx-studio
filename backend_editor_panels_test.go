package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBackendEditorPanelsEmitSurfaceContracts(t *testing.T) {
	view := backendEditorPanelTestView()

	cases := []struct {
		name      string
		html      string
		fragments []string
	}{
		{
			name: "brand",
			html: gosx.RenderHTML(RenderBrandPanel(view)),
			fragments: []string{
				`data-studio-brand-panel="true"`,
				`data-studio-media-picker-island="brand"`,
				`data-editor-brand-logo="true"`,
				`name="studioPickLogoUrl"`,
			},
		},
		{
			name: "publish",
			html: gosx.RenderHTML(RenderPublishPanel(view)),
			fragments: []string{
				`data-studio-publish-panel="true"`,
				`data-studio-preview-share="true"`,
				`data-studio-activity-drawer="true"`,
				`data-studio-revision-history="true"`,
				`data-studio-submit-action="publish"`,
			},
		},
		{
			name: "advanced",
			html: gosx.RenderHTML(RenderAdvancedPanel(view)),
			fragments: []string{
				`data-studio-advanced-panel="true"`,
				`data-studio-flow-designer-panel="true"`,
				`data-studio-resource-adapters="true"`,
				`data-studio-advanced-field-row="true"`,
				`data-studio-flow-card="contact"`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, fragment := range tc.fragments {
				if !strings.Contains(tc.html, fragment) {
					t.Fatalf("%s renderer missing %q:\n%s", tc.name, fragment, tc.html)
				}
			}
			if strings.Contains(tc.html, "GoSX Studio") {
				t.Fatalf("%s renderer must not inject visible platform copy:\n%s", tc.name, tc.html)
			}
		})
	}
}

func TestRenderFlowDesignerPanelEmitsNestedFlowContent(t *testing.T) {
	html := gosx.RenderHTML(RenderFlowDesignerPanel(backendEditorPanelTestView()))
	for _, fragment := range []string{
		`data-studio-flow-node="start"`,
		`data-studio-flow-check="handler"`,
		`data-studio-flow-step="intro"`,
		`data-studio-flow-action="submit"`,
		`data-studio-flow-handler-ref="true"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("flow designer missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderFlowDesignerPanelUsesCSSRadioSelection(t *testing.T) {
	view := backendEditorPanelTestView()
	addNewsletterFlow(view)

	html := gosx.RenderHTML(RenderFlowDesignerPanel(view))
	for _, fragment := range []string{
		`class="studio-flow-designer__input" id="studioFlowSelect-contact" type="radio" name="studioFlowDesignerFlow" value="contact" checked`,
		`class="studio-flow-designer__input" id="studioFlowSelect-newsletter" type="radio" name="studioFlowDesignerFlow" value="newsletter"`,
		`<label for="studioFlowSelect-contact" class="flow-card" data-editor-flow="contact" data-studio-flow-card="contact"`,
		`<label for="studioFlowSelect-newsletter" class="flow-card" data-editor-flow="newsletter" data-studio-flow-card="newsletter"`,
		`data-studio-flow-editor="contact"`,
		`data-studio-flow-editor="newsletter"`,
		`data-studio-flow-editor-visible="true"`,
		`data-studio-flow-editor-visible="false"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("flow designer missing CSS radio selection fragment %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<button type="button" class="flow-card"`) {
		t.Fatalf("flow designer must not render inert button flow cards:\n%s", html)
	}
}

func TestRenderBrandMediaPickerUsesCSSRadioFilters(t *testing.T) {
	view := backendEditorPanelTestView()
	mediaPicker := view["mediaPicker"].(map[string]any)
	mediaPicker["assets"] = []map[string]any{
		{"id": "asset-1", "url": "/logo.png", "alt": "Logo", "filename": "logo.png", "statusLabel": "Ready", "actionLabel": "Use asset", "cardClass": "studio-brand-media-card", "filterGroup": "ready"},
		{"id": "asset-2", "url": "/missing.png", "alt": "", "filename": "missing.png", "statusLabel": "Needs alt", "actionLabel": "Use asset", "cardClass": "studio-brand-media-card studio-brand-media-card--missing-alt", "filterGroup": "missing-alt"},
	}

	html := gosx.RenderHTML(RenderBrandMediaPicker(view))
	for _, fragment := range []string{
		`class="studio-brand-media-picker__filter-input" id="studioBrandMediaFilterAll" type="radio" name="studioBrandMediaFilter" value="all" checked`,
		`class="studio-brand-media-picker__filter-input" id="studioBrandMediaFilterReady" type="radio" name="studioBrandMediaFilter" value="ready"`,
		`class="studio-brand-media-picker__filter-input" id="studioBrandMediaFilterMissingAlt" type="radio" name="studioBrandMediaFilter" value="missing-alt"`,
		`<label for="studioBrandMediaFilterAll" role="button" data-studio-media-filter-option="all"`,
		`<label for="studioBrandMediaFilterReady" role="button" data-studio-media-filter-option="ready"`,
		`<label for="studioBrandMediaFilterMissingAlt" role="button" data-studio-media-filter-option="missing-alt"`,
		`data-studio-media-asset="asset-1" data-studio-media-filter-group="ready"`,
		`data-studio-media-asset="asset-2" data-studio-media-filter-group="missing-alt"`,
		`type="submit" form="editorForm" formaction="/save" name="studioPickLogoUrl"`,
		`type="submit" form="editorForm" formaction="/save" name="studioPickFaviconUrl"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("brand media picker missing CSS radio filter fragment %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, `<button type="button" aria-pressed`) {
		t.Fatalf("brand media picker must not render inert type=button filter controls:\n%s", html)
	}
}

func TestRenderBackendEditorPanelBooleanStateAttrs(t *testing.T) {
	view := backendEditorPanelTestView()
	addNewsletterFlow(view)

	rendered := []string{
		gosx.RenderHTML(RenderPublishPanel(view)),
		gosx.RenderHTML(RenderBrandPanel(view)),
		gosx.RenderHTML(RenderFlowDesignerPanel(view)),
		gosx.RenderHTML(RenderAdvancedPanel(view)),
	}
	html := strings.Join(rendered, "\n")
	for _, fragment := range []string{
		`data-studio-has-draft="false"`,
		`data-studio-brand-group-selected="true"`,
		`data-studio-brand-group-selected="false"`,
		`data-studio-flow-card-selected="true"`,
		`data-studio-flow-card-selected="false"`,
		`data-studio-flow-dirty-visible="false"`,
		`data-studio-flow-valid="true"`,
		`data-studio-flow-valid="false"`,
		`data-studio-flow-dirty="false"`,
		`data-studio-flow-editor-visible="true"`,
		`data-studio-flow-editor-visible="false"`,
		`data-studio-flow-editor-state-visible="true"`,
		`data-studio-flow-editor-state-visible="false"`,
		`data-studio-advanced-group-selected="true"`,
		`data-studio-advanced-group-selected="false"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("rendered panels missing boolean state attr %q:\n%s", fragment, html)
		}
	}
}

func addNewsletterFlow(view map[string]any) {
	flowDesigner := view["flowDesigner"].(map[string]any)
	flows := flowDesigner["flows"].([]map[string]any)
	flows = append(flows, map[string]any{
		"key":                "newsletter",
		"cardClass":          "flow-card",
		"label":              "Newsletter",
		"statusLabel":        "Draft",
		"description":        "Newsletter signup",
		"summary":            "One action",
		"handlerStatusLabel": "Needs handler",
		"route":              "/newsletter",
		"embedTarget":        "newsletter",
		"routeInputID":       "newsletterRoute",
		"routeInputName":     "newsletterRoute",
		"embedInputID":       "newsletterEmbed",
		"embedInputName":     "newsletterEmbed",
		"handlerInputID":     "newsletterHandler",
		"handlerInputName":   "newsletterHandler",
		"handlerRef":         "",
		"handlerSource":      "flow.newsletter.handler",
		"handlerClass":       "handler",
		"handlerDetail":      "Missing",
		"readiness":          map[string]any{"canPublish": false, "status": "watch", "badgeClass": "badge", "label": "Watch", "summary": "Needs handler", "class": "readiness"},
		"readinessChecks":    []map[string]any{},
		"nodes":              []map[string]any{},
		"steps":              []map[string]any{},
		"actions":            []map[string]any{},
	})
	flowDesigner["flows"] = flows
}

func backendEditorPanelTestView() map[string]any {
	return map[string]any{
		"brandPanel": map[string]any{
			"defaultGroupKey": "logo",
			"kicker":          "Brand",
			"title":           "Identity",
			"summary":         "Manage brand assets.",
			"previewLabel":    "Preview",
			"groupLabel":      "Brand groups",
			"groups": []map[string]any{
				{"key": "logo", "inputID": "brandLogo", "label": "Logo", "summary": "Mark", "selected": true},
				{"key": "placement", "inputID": "brandPlacement", "label": "Placement", "summary": "Position"},
				{"key": "files", "inputID": "brandFiles", "label": "Files", "summary": "Media"},
			},
		},
		"brandFields": map[string]any{
			"preview": map[string]any{"cornerLabel": "Logo", "buttonLabel": "Move logo", "logoURL": "/logo.png", "logoAlt": "Logo"},
			"logoFields": []map[string]any{{
				"id": "logoAlt", "label": "Logo alt", "value": "Logo", "isInput": true,
				"rowAttrs":     map[string]any{"class": "field-row"},
				"controlAttrs": map[string]any{"id": "logoAlt", "name": "logoAlt", "value": "Logo"},
			}},
			"layoutFields": []map[string]any{},
			"fileFields":   []map[string]any{},
			"tools":        map[string]any{"snapChecked": true, "gridLabel": "Grid", "snapSize": "8", "resetLabel": "Reset"},
			"media":        map[string]any{"class": "button", "href": "/admin/media", "label": "Open media"},
		},
		"mediaPicker": map[string]any{
			"kicker": "Media", "title": "Assets", "summary": "Choose assets.", "uploadHref": "/admin/media/new", "filterLabel": "Filters", "hasAssets": true, "assetLabel": "Assets", "formID": "editorForm", "saveAction": "/save", "logoPickName": "studioPickLogoUrl", "faviconPickName": "studioPickFaviconUrl",
			"assets": []map[string]any{{"id": "asset-1", "url": "/logo.png", "alt": "Logo", "filename": "logo.png", "statusLabel": "Ready", "actionLabel": "Use asset", "cardClass": "asset-card", "filterGroup": "ready"}},
		},
		"publishPanel": map[string]any{
			"kicker": "Publish", "panelTitle": "Release center", "summary": "Review release.", "countLabel": "1/1 clear", "status": "ready", "formID": "editorForm", "previewHref": "/preview", "hasPublishAction": true, "publishAction": "/publish", "hasScheduleAction": true, "scheduleAction": "/schedule", "scheduleInputID": "publishAt", "scheduleInputName": "publishAt", "scheduleHelp": "Pick a time.", "readyCountLabel": "1", "watchCountLabel": "0", "nextCountLabel": "0", "hasChecks": true, "hasImpacts": true,
			"checks":  []map[string]any{{"key": "copy", "class": "check", "label": "Copy", "scope": "Content", "statusLabel": "Ready", "summary": "Ready", "detail": "Done"}},
			"impacts": []map[string]any{{"key": "home", "class": "impact", "label": "Home", "scope": "Page", "value": "1", "detail": "One page"}},
		},
		"previewShare": map[string]any{"class": "studio-preview-share", "state": "ready", "kicker": "Preview", "title": "Share", "status": "Ready", "hasHref": true, "detail": "Copy URL.", "inputLabel": "Preview URL", "href": "/?preview=1", "copyLabel": "Copy", "openLabel": "Open", "resource": "settings/site", "audience": "review", "expiresLabel": "Soon"},
		"activityPanel": map[string]any{
			"class": "studio-activity", "kicker": "Activity", "title": "Readiness", "score": "1/1", "toggleLabel": "Toggle", "togglePressed": "true",
			"readiness": map[string]any{"readyCount": "1", "watchCount": "0", "nextCount": "0", "items": []map[string]any{{"key": "copy", "class": "ready", "label": "Copy", "summary": "Ready", "statusLabel": "Ready"}}},
			"comments":  map[string]any{"class": "studio-comments", "headClass": "studio-comments__head", "kickerClass": "kicker", "countClass": "count", "emptyClass": "empty", "kicker": "Comments", "title": "Notes", "countLabel": "0", "emptyTitle": "No notes", "emptyDetail": "Nothing yet."},
			"proposals": map[string]any{"class": "studio-proposals", "headClass": "studio-proposals__head", "kickerClass": "kicker", "countClass": "count", "emptyClass": "empty", "kicker": "Proposals", "title": "Suggestions", "countLabel": "0", "emptyTitle": "No proposals", "emptyDetail": "Nothing yet."},
		},
		"revisionHistory": map[string]any{"class": "history", "panelKey": "history", "headerClass": "head", "title": "History", "empty": "No revisions", "hasItems": true, "items": []map[string]any{{"key": "rev-1", "title": "Saved", "actionLabel": "Save", "createdMachine": "2026-01-01T00:00:00Z", "createdLabel": "Jan 1", "formID": "editorForm", "restoreAction": "/restore", "revisionInputName": "revision", "revisionInputValue": "rev-1", "confirm": "Restore?", "buttonLabel": "Restore"}}},
		"advancedPanel":   map[string]any{"defaultGroupKey": "flows", "kicker": "Advanced", "title": "Operations", "summary": "Manage advanced surfaces.", "groupLabel": "Advanced groups", "groups": []map[string]any{{"key": "flows", "inputID": "advancedFlows", "label": "Flows", "summary": "Forms", "selected": true}, {"key": "tools", "inputID": "advancedTools", "label": "Tools", "summary": "Resources"}, {"key": "schema", "inputID": "advancedSchema", "label": "Schema", "summary": "Fields"}}},
		"toolsPanel":      map[string]any{"class": "tools", "key": "tools", "mode": "advanced", "kicker": "Advanced", "title": "Tools", "resourcesTitle": "Resources", "resourcesSummary": "Adapters", "hasAdapters": true, "resourceGridClass": "grid", "adapters": []map[string]any{{"kind": "cms", "surface": "settings", "label": "CMS", "summary": "Settings", "capabilityLabel": "Read", "bindingLabel": "Bound"}}, "hasItems": true, "gridClass": "tool-grid", "items": []map[string]any{{"label": "Media", "summary": "Open media", "attrs": map[string]any{"href": "/admin/media", "class": "tool"}}}},
		"advancedFields":  map[string]any{"workspace": map[string]any{"class": "fields", "key": "workspace-fields", "mode": "advanced", "kicker": "Advanced", "title": "Fields", "hasFields": true, "fields": []map[string]any{{"id": "workspaceName", "label": "Workspace", "value": "Main", "isInput": true, "rowAttrs": map[string]any{"class": "field-row"}, "controlAttrs": map[string]any{"id": "workspaceName", "name": "workspaceName", "value": "Main"}}}}},
		"flowDesigner":    map[string]any{"defaultFlowKey": "contact", "formID": "editorForm", "publishAction": "/publish-flow", "flows": []map[string]any{{"key": "contact", "cardClass": "flow-card", "label": "Contact", "statusLabel": "Ready", "description": "Contact form", "summary": "One step", "handlerStatusLabel": "Handler ready", "route": "/contact", "embedTarget": "contact", "routeInputID": "flowRoute", "routeInputName": "flowRoute", "embedInputID": "flowEmbed", "embedInputName": "flowEmbed", "handlerInputID": "handler", "handlerInputName": "handler", "handlerRef": "contact.handle", "handlerSource": "flow.contact.handler", "handlerClass": "handler", "handlerDetail": "Configured", "readiness": map[string]any{"canPublish": true, "status": "ready", "badgeClass": "badge", "label": "Ready", "summary": "Ready", "class": "readiness"}, "readinessChecks": []map[string]any{{"key": "handler", "class": "check", "status": "ready", "statusLabel": "Ready", "label": "Handler", "summary": "Configured", "tabIndex": "0"}}, "nodes": []map[string]any{{"key": "start", "class": "node", "kind": "start", "kindLabel": "Start", "status": "ready", "label": "Start", "summary": "Begin", "tabIndex": "0"}}, "steps": []map[string]any{{"key": "intro", "label": "Intro", "labelInputID": "stepIntro", "labelInputName": "stepIntro", "labelSource": "flow.contact.steps.intro", "bodySource": "flow.contact.steps.intro.body", "bodySummary": "Say hello"}}, "actions": []map[string]any{{"key": "submit", "label": "Submit", "fieldCount": "1", "fields": []map[string]any{{"name": "email", "label": "Email", "requiredLabel": "Required"}}}}}}},
	}
}
