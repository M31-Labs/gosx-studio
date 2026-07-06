package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderFlowDesignerPanelFull(t *testing.T) {
	html := gosx.RenderHTML(RenderFlowDesignerPanel(flowDesignerPanelTestView(), FlowDesignerPanelOptions{}))

	for _, fragment := range []string{
		`<section class="editor-panel editor-panel--flow-designer studio-flow-designer" data-studio-mode-panel="advanced" data-studio-panel="flow-designer" data-studio-flow-designer-panel="true" data-studio-engine-source="gosx" data-studio-flow-selected="contact" data-gosx-studio-flow-designer-panel-renderer="gosx-studio">`,
		`<header class="studio-panel-heading"><p class="kicker">Advanced</p><h2>Form flows</h2><p>Build and publish the site forms people use to contact, request, subscribe, and continue checkout.</p></header>`,
		`<button type="button" class="studio-flow-card studio-flow-card--ready" data-editor-flow="contact" data-studio-flow-card="contact" data-studio-flow-card-selected="true" data-studio-flow-route="/contact" data-studio-flow-label="Contact" data-studio-flow-status="Ready" aria-pressed="true">`,
		`<output class="studio-flow-summary__dirty" data-studio-flow-dirty-badge="contact" data-studio-flow-dirty-visible="false">Unsaved</output>`,
		`<article class="studio-flow-editor" data-editor-flow="contact" data-studio-flow-editor="contact" data-studio-flow-valid="true" data-studio-flow-readiness-status="ready" data-studio-flow-dirty="false" data-studio-flow-editor-visible="true">`,
		`<output class="studio-flow-editor__state studio-flow-editor__state--ready" data-studio-flow-editor-state="contact" data-studio-flow-editor-state-visible="true">Ready to publish</output>`,
		`<output class="studio-flow-editor__state studio-flow-editor__state--watch" data-studio-flow-editor-state="contact" data-studio-flow-editor-state-visible="false">Unsaved changes</output>`,
		`<button class="button button--secondary" type="submit" form="websiteEditorForm" formaction="/admin/editor/__actions/publishFlow" formmethod="post" name="flowKey" value="contact" data-studio-submit-action="publish-flow">Publish flow</button>`,
		`<div class="studio-flow-readiness studio-flow-readiness--ready" data-studio-flow-readiness="contact">`,
		`<section class="studio-flow-check studio-flow-check--ready" role="listitem" tabindex="0" data-studio-flow-check="handler" data-studio-flow-check-status="ready">`,
		`<div class="studio-flow-graph" role="list" aria-label="Flow map">`,
		`<section class="studio-flow-node studio-flow-node--ready studio-flow-node--action" role="listitem" tabindex="0" data-studio-flow-node="action:submit" data-studio-flow-node-kind="action" data-studio-flow-node-status="ready">`,
		`<input id="studio-flow-route-contact" name="flowContactRoute" value="/contact" form="websiteEditorForm" data-studio-field-source="flow.contact.route" data-studio-field-editable="routing" readonly />`,
		`<input id="studio-flow-handler-contact" name="flowContactHandlerRef" value="contact.submit" form="websiteEditorForm" data-studio-field-source="flow.contact.handlerRef" data-studio-field-editable="flow" data-studio-flow-handler-ref="true" type="hidden" />`,
		`<section class="studio-flow-step" data-studio-flow-step="details" tabindex="0">`,
		`<input id="studio-flow-step-details" name="flowContactStepDetailsLabel" value="Details" form="websiteEditorForm" data-studio-field-source="flow.contact.steps.details.label" data-studio-field-editable="flow" />`,
		`<section class="studio-flow-action" data-studio-flow-action="submit" tabindex="0">`,
		`<span class="studio-flow-field" data-studio-flow-field="email" tabindex="0"><strong>Email</strong><small>Required</small></span>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("flow designer panel missing %q:\n%s", fragment, html)
		}
	}
	for _, notWant := range []string{"<form", "csrf_token"} {
		if strings.Contains(html, notWant) {
			t.Fatalf("flow designer panel must not include %q:\n%s", notWant, html)
		}
	}
}

func TestRenderFlowDesignerPanelMinimal(t *testing.T) {
	html := gosx.RenderHTML(RenderFlowDesignerPanel(map[string]any{}, FlowDesignerPanelOptions{}))

	for _, fragment := range []string{
		`class="editor-panel editor-panel--flow-designer studio-flow-designer"`,
		`data-studio-mode-panel="advanced"`,
		`data-studio-panel="flow-designer"`,
		`data-studio-flow-designer-panel="true"`,
		`data-studio-flow-selected=""`,
		`data-gosx-studio-flow-designer-panel-renderer="gosx-studio"`,
		`<div class="studio-flow-designer__list" role="list" aria-label="Form flows"></div>`,
		`<div class="studio-flow-designer__editors"></div>`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("minimal flow designer panel missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderFlowDesignerPanelRootAttrOverride(t *testing.T) {
	html := gosx.RenderHTML(RenderFlowDesignerPanel(flowDesignerPanelTestView(), FlowDesignerPanelOptions{
		RootAttrs: map[string]any{
			"id":                            "customFlowDesigner",
			"class":                         "custom-flow-panel",
			"data-studio-panel":             "custom-flow",
			"aria-label":                    "Flow designer override",
			"data-flow-designer-extra":      "test",
			"data-studio-flow-selected":     "checkout",
			"data-studio-flow-designer-hot": true,
		},
	}))

	for _, fragment := range []string{
		`id="customFlowDesigner"`,
		`class="custom-flow-panel"`,
		`data-studio-panel="custom-flow"`,
		`aria-label="Flow designer override"`,
		`data-flow-designer-extra="test"`,
		`data-studio-flow-selected="checkout"`,
		`data-studio-flow-designer-hot="true"`,
		`data-gosx-studio-flow-designer-panel-renderer="gosx-studio"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("flow designer root attr override missing %q:\n%s", fragment, html)
		}
	}
}

func flowDesignerPanelTestView() map[string]any {
	return map[string]any{
		"formID":         "websiteEditorForm",
		"publishAction":  "/admin/editor/__actions/publishFlow",
		"defaultFlowKey": "contact",
		"flows": []map[string]any{
			flowDesignerPanelTestFlow("contact", true),
			flowDesignerPanelTestFlow("checkout", false),
		},
	}
}

func flowDesignerPanelTestFlow(key string, ready bool) map[string]any {
	status := "ready"
	if !ready {
		status = "watch"
	}
	label := "Contact"
	if key == "checkout" {
		label = "Checkout"
	}
	return map[string]any{
		"key":                key,
		"label":              label,
		"description":        "Collect customer details.",
		"summary":            "One step, one action",
		"statusLabel":        "Ready",
		"cardClass":          "studio-flow-card studio-flow-card--ready",
		"route":              "/" + key,
		"embedTarget":        "main",
		"handlerRef":         key + ".submit",
		"handlerStatusLabel": "Connected",
		"handlerDetail":      "This flow is wired to the site action that processes submissions.",
		"handlerClass":       "studio-flow-handler studio-flow-handler--ready",
		"handlerInputID":     "studio-flow-handler-" + key,
		"handlerInputName":   "flowContactHandlerRef",
		"handlerSource":      "flow." + key + ".handlerRef",
		"routeInputID":       "studio-flow-route-" + key,
		"routeInputName":     "flowContactRoute",
		"embedInputID":       "studio-flow-embed-" + key,
		"embedInputName":     "flowContactEmbedTarget",
		"readiness": map[string]any{
			"status":     status,
			"label":      "Ready to publish",
			"summary":    "All visible checks are ready for publishing.",
			"canPublish": ready,
			"class":      "studio-flow-readiness studio-flow-readiness--" + status,
			"badgeClass": "studio-flow-editor__state studio-flow-editor__state--" + status,
		},
		"readinessChecks": []map[string]any{
			{
				"key":         "handler",
				"label":       "Submission action",
				"summary":     "Connected.",
				"status":      "ready",
				"statusLabel": "Ready",
				"class":       "studio-flow-check studio-flow-check--ready",
				"tabIndex":    "0",
			},
		},
		"nodes": []map[string]any{
			{
				"key":       "action:submit",
				"label":     "Submit",
				"kind":      "action",
				"kindLabel": "Action",
				"status":    "ready",
				"summary":   "Runs the handler.",
				"class":     "studio-flow-node studio-flow-node--ready studio-flow-node--action",
				"tabIndex":  "0",
			},
		},
		"steps": []map[string]any{
			{
				"key":            "details",
				"label":          "Details",
				"labelInputID":   "studio-flow-step-details",
				"labelInputName": "flowContactStepDetailsLabel",
				"labelSource":    "flow.contact.steps.details.label",
				"bodySource":     "flow.contact.steps.details.body",
				"bodySummary":    "Ask for contact details.",
			},
		},
		"actions": []map[string]any{
			{
				"key":        "submit",
				"label":      "Submit",
				"fieldCount": "1",
				"fields": []map[string]any{
					{"name": "email", "label": "Email", "requiredLabel": "Required"},
				},
			},
		},
	}
}
