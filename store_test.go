package studio

import (
	"testing"

	"m31labs.dev/gosx-cms/lifecycle"
	"m31labs.dev/gosx-cms/media"
	"m31labs.dev/gosx-studio/fieldruntime"
	"m31labs.dev/gosx-studio/selectionruntime"
)

type stubStore struct{}

func (stubStore) LeftPanels() []Panel {
	return []Panel{NewPanel("site tree", "Site tree", "Editable page outline.")}
}

func (stubStore) RightPanels() []Panel {
	return []Panel{NewPanel("inspector", "Inspector", "Selected section controls.")}
}

func (stubStore) MediaAssets() []media.Asset {
	return []media.Asset{{ID: "asset_1", URL: "/media/forest.jpg"}}
}

func (stubStore) Revisions() []lifecycle.Revision {
	return []lifecycle.Revision{{ID: "rev_1", ResourceKind: "page", ResourceID: "home"}}
}

func (stubStore) Readiness() ShellReadiness {
	return NewShellReadiness(NewShellReadinessItem("content", "Content", ShellReadinessReady, "Ready", "All required copy is present."))
}

func (stubStore) Adapters() []ResourceAdapter {
	return []ResourceAdapter{{
		Kind:         ResourcePages,
		Label:        "Pages",
		Summary:      "Host page drafts and routes.",
		Surface:      SurfaceSiteMap,
		Capabilities: []ResourceCapability{ResourceList, ResourceRead},
		Bindings:     []ResourceBinding{{Key: "routes", Label: "Routes", Binding: "pages.routes"}},
	}}
}

func TestHostShellComposesStoreAndShellConfig(t *testing.T) {
	shell := HostShell(stubStore{}, ShellConfig{
		Labels: HostLabels{EditorTitle: "Forest site editor"},
		Resources: []ResourceConfig{{
			Key:     "programs",
			Label:   "Programs",
			Kind:    "content",
			Href:    "/admin/programs",
			Summary: "Nature-school programs and cohorts.",
		}},
		Actions: []ActionConfig{
			{Key: "save", Label: "Save", Href: "/admin/editor?action=save", Primary: true},
			{Key: "publish", Label: "Publish", Href: "/admin/editor?action=publish", Primary: true},
			{Key: "restore", Label: "Restore", Href: "/admin/editor?action=restore"},
		},
		SiteMap: SiteMap{Pages: []Page{
			{Key: "home", Label: "Home", Route: "/", GoSXComponent: "HomePage"},
			{
				Key:           "contact",
				Label:         "Family contact",
				Route:         "/contact",
				GoSXComponent: "ContactPage",
				Selected:      true,
				Components: []Component{
					{Key: "hero", Label: "Hero", GoSXComponent: "HeroSection", Controls: []Control{{Key: "headline", Label: "Headline", Kind: ControlText}}},
					{Key: "form", Label: "Contact form", GoSXComponent: "ContactForm"},
				},
			},
		}},
	})

	if shell.Title != "Forest site editor" || shell.PreviewURL != "/contact" {
		t.Fatalf("unexpected shell identity: %#v", shell)
	}
	if shell.SaveAction != "/admin/editor?action=save" || shell.RestoreAction != "/admin/editor?action=restore" {
		t.Fatalf("expected action URLs from shell config: %#v", shell)
	}
	if len(shell.Actions) != 3 || shell.Actions[0].Key != "save" || !shell.Actions[0].Primary {
		t.Fatalf("expected normalized configured actions: %#v", shell.Actions)
	}
	if len(shell.Navigation) != 1 || shell.Navigation[0].Key != "programs" || len(shell.Navigation[0].Actions) != 1 {
		t.Fatalf("expected resource navigation section: %#v", shell.Navigation)
	}
	if len(shell.Metrics) != 3 || shell.Metrics[0].Value != 2 || shell.Metrics[1].Value != 2 || shell.Metrics[2].Value != 1 {
		t.Fatalf("expected site-map metrics: %#v", shell.Metrics)
	}
	if len(shell.Modes) != 5 || shell.Modes[0].Key != "home" || !shell.Modes[0].Active {
		t.Fatalf("expected default editor modes: %#v", shell.Modes)
	}
	if shell.Canvas.RouteLabel != "Family contact" || shell.Canvas.SelectionLabel != "2 editable sections" || shell.Canvas.Zoom != "fit" {
		t.Fatalf("expected selected page canvas labels: %#v", shell.Canvas)
	}
	if shell.MediaCount != 1 || shell.RevisionCount != 1 || shell.Readiness.Summary() != "1/1 ready" {
		t.Fatalf("expected store-backed counts and readiness: %#v", shell)
	}

	view := View(shell)
	resources := view["resources"].([]map[string]any)
	adapters := view["adapters"].([]map[string]any)
	if len(resources) != 1 || resources[0]["href"] != "/admin/programs" || resources[0]["hasHref"] != true {
		t.Fatalf("expected resource views in extras: %#v", resources)
	}
	if len(adapters) != 1 || adapters[0]["kind"] != string(ResourcePages) || len(adapters[0]["bindings"].([]map[string]any)) != 1 {
		t.Fatalf("expected store adapter views in extras: %#v", adapters)
	}
	authoring := view["authoring"].(map[string]any)
	if authoring["pageCount"] != 2 || authoring["componentCount"] != 2 || authoring["controlCount"] != 1 {
		t.Fatalf("expected no-code authoring counts in host shell: %#v", authoring)
	}
	selectedPage := authoring["selectedPage"].(map[string]any)
	if selectedPage["key"] != "contact" || selectedPage["route"] != "/contact" {
		t.Fatalf("expected selected page authoring projection: %#v", selectedPage)
	}
	workspace := authoring["workspace"].(map[string]any)
	if workspace["nodeCount"] != 4 || workspace["linkCount"] != 2 {
		t.Fatalf("expected authoring workspace graph: %#v", workspace)
	}
}

func TestHostShellFallsBackToShellAdaptersWithoutStore(t *testing.T) {
	shell := HostShell(nil, ShellConfig{
		Labels: HostLabels{EditorTitle: "Website editor"},
		Adapters: []ResourceAdapter{{
			Kind:     ResourceMedia,
			Label:    "Media",
			Bindings: []ResourceBinding{{Key: "assets", Label: "Assets", Binding: "media.assets"}},
		}},
	})
	view := View(shell)
	adapters := view["adapters"].([]map[string]any)
	if len(adapters) != 1 || adapters[0]["kind"] != string(ResourceMedia) {
		t.Fatalf("expected shell adapter views: %#v", adapters)
	}
	if shell.PreviewURL != "/" || shell.Canvas.SelectionLabel != "No selection" {
		t.Fatalf("expected empty-shell fallbacks: %#v", shell)
	}
}

func TestFeatureFlagAttrsRendersEnabledRuntimeFlags(t *testing.T) {
	attrs := FeatureFlagAttrs(ShellConfig{FeatureFlags: map[string]bool{
		fieldruntime.FeatureFlagKey:     true,
		selectionruntime.FeatureFlagKey: false,
		"publish-review":                true,
	}})

	fieldAttr := "data-gosx-studio-feature-flag-" + fieldruntime.FeatureFlagKey
	selectionAttr := "data-gosx-studio-feature-flag-" + selectionruntime.FeatureFlagKey
	if attrs[fieldAttr] != "true" {
		t.Fatalf("expected enabled field runtime flag attr: %#v", attrs)
	}
	if _, ok := attrs[selectionAttr]; ok {
		t.Fatalf("disabled runtime flag should not render: %#v", attrs)
	}
	if _, ok := attrs["data-gosx-studio-feature-flag-publish-review"]; ok {
		t.Fatalf("non-runtime feature flag should not render as runtime attr: %#v", attrs)
	}
}
