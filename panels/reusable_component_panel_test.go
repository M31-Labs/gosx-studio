package panels

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

func reusablePanelDefinition() core.ComponentDefinition {
	return core.ComponentDefinition{ID: "def-hero", Key: "shared-hero", Version: 3, TemplateKey: "hero", Label: "Shared hero", Controls: []core.Control{{Key: "headline", Label: "Headline", Kind: core.ControlRichText, Value: "Shared"}}, Styles: core.DefinitionStyles{"base/default": {"background-color": "var(--color-canvas)"}}, Layout: core.DefinitionLayout{"mobile/default": {"display": "block"}}}
}

func TestRenderReusableComponentPanelProjectsRealDefinitionAndSelectedInstance(t *testing.T) {
	definition := reusablePanelDefinition()
	instance := core.ComponentInstance{ID: "inst-home-hero", DefinitionID: definition.ID, DefinitionVersion: definition.Version, PageKey: "home", Region: "main", Position: 1, HeadRevision: 9, Overrides: map[string]core.ExplicitOverride{"control:headline": {Present: true, Value: "Only this hero"}}}
	html := gosx.RenderHTML(RenderReusableComponentPanel(ReusableComponentPanelOptions{
		Action: "/admin/editor/__actions/reusable", CSRFToken: "csrf", Definitions: []core.ComponentDefinition{definition}, Instances: []core.ComponentInstance{instance}, SelectedInstanceID: instance.ID,
		TargetPageKey: "home", TargetRegion: "main", NextPosition: 2, CreateInstanceIDs: map[string]string{definition.ID: "inst-next"}, Heads: map[string]string{"instance:" + instance.ID: "head-9"},
		OperationIDs: map[string]string{
			ReusablePanelOperationKey(authoring.ReusableCreateInstance, "inst-next", ""):                "op-create-next",
			ReusablePanelOperationKey(authoring.ReusableSetInstanceOverride, instance.ID, ""):           "op-set-override",
			ReusablePanelOperationKey(authoring.ReusableClearOverride, instance.ID, "control:headline"): "op-clear-headline",
			ReusablePanelOperationKey(authoring.ReusableDetachInstance, instance.ID, ""):                "op-detach",
		},
	}))
	for _, want := range []string{
		`data-studio-reusable-component-panel="true"`,
		`data-studio-selected-instance="inst-home-hero"`,
		`data-studio-reusable-definition="def-hero"`,
		`data-studio-definition-version="3"`,
		`name="gosx_studio_reusable_operation" value="create-instance"`,
		`name="gosx_studio_reusable_instance_id" value="inst-next"`,
		`name="gosx_studio_reusable_operation_id" value="op-create-next"`,
		`data-studio-canvas-identity="instance:inst-home-hero"`,
		`data-studio-instance-override="control:headline"`,
		`name="gosx_studio_reusable_operation" value="clear-override"`,
		`name="gosx_studio_reusable_operation" value="set-instance-override"`,
		`name="gosx_studio_reusable_operation" value="detach-instance"`,
		`name="gosx_studio_reusable_expected_head" value="head-9"`,
		`name="csrf_token" value="csrf"`,
		`Only this hero`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("reusable panel missing %q:\n%s", want, html)
		}
	}
	if strings.Contains(html, "localStorage") || strings.Contains(html, "contenteditable") {
		t.Fatalf("reusable state must be host-backed forms, not browser-fabricated state:\n%s", html)
	}
}

func TestRenderReusableComponentPanelShowsDetachAndRestoreTruthfully(t *testing.T) {
	definition := reusablePanelDefinition()
	materialized := definition.Normalize()
	detached := core.ComponentInstance{ID: "inst-detached", DefinitionID: definition.ID, DefinitionVersion: definition.Version, PageKey: "home", Region: "main", Detached: true, Materialized: &materialized}
	html := gosx.RenderHTML(RenderReusableComponentPanel(ReusableComponentPanelOptions{Action: "/actions/reusable", Definitions: []core.ComponentDefinition{definition}, Instances: []core.ComponentInstance{detached}, SelectedInstanceID: detached.ID}))
	if !strings.Contains(html, `data-studio-reusable-instance-state="detached-copy"`) || !strings.Contains(html, `value="restore-instance"`) || strings.Contains(html, `value="detach-instance"`) {
		t.Fatalf("detached inspector must offer restore only:\n%s", html)
	}
	empty := gosx.RenderHTML(RenderReusableComponentPanel(ReusableComponentPanelOptions{Definitions: []core.ComponentDefinition{definition}}))
	if !strings.Contains(empty, `data-studio-reusable-instance-empty="true"`) || strings.Contains(empty, `data-studio-reusable-instance-inspector="true"`) {
		t.Fatalf("empty selection must not fabricate an instance inspector:\n%s", empty)
	}
}
