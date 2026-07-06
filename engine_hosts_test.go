package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/engine"
)

type recordingStudioEngineRuntime struct {
	calls []engine.Config
}

func (runtime *recordingStudioEngineRuntime) Engine(cfg engine.Config, fallback gosx.Node) gosx.Node {
	runtime.calls = append(runtime.calls, cfg)
	if !fallback.IsZero() {
		return gosx.El("div", gosx.Attrs(gosx.Attr("data-unexpected-fallback", "true")))
	}
	attrs := []any{
		gosx.Attr("id", cfg.MountID),
		gosx.Attr("data-gosx-engine", cfg.Name),
		gosx.Attr("data-gosx-engine-id", "gosx-engine-0"),
		gosx.Attr("data-gosx-engine-kind", string(cfg.Kind)),
		gosx.Attr("data-gosx-enhance", "engine"),
		gosx.Attr("data-gosx-enhance-layer", "runtime"),
		gosx.Attr("data-gosx-fallback", "none"),
	}
	for name, value := range cfg.MountAttrs {
		attrs = append(attrs, gosx.Attr(name, value))
	}
	if len(cfg.Capabilities) > 0 {
		attrs = append(attrs, gosx.Attr("data-gosx-engine-capabilities", strings.Join(siteMapEngineCapabilityNames(cfg.Capabilities), " ")))
	}
	return gosx.El("div", gosx.Attrs(attrs...))
}

func TestRenderStudioEngineHostsUsesEngineRuntimeForSurfaceContract(t *testing.T) {
	runtime := &recordingStudioEngineRuntime{}
	html := gosx.RenderHTML(RenderStudioEngineHosts([]map[string]any{{
		"key":          "flow-designer",
		"name":         FlowDesignerName,
		"mountId":      "gosx-studio-flow-engine",
		"class":        "studio-flow-engine-host",
		"capabilities": []string{"canvas", "pointer", "keyboard", "text-input", "animation"},
	}}, StudioEngineHostsOptions{EngineRuntime: runtime}))

	if len(runtime.calls) != 1 {
		t.Fatalf("runtime call count = %d, want 1", len(runtime.calls))
	}
	cfg := runtime.calls[0]
	if cfg.Name != FlowDesignerName || cfg.Kind != engine.KindSurface || cfg.MountID != "gosx-studio-flow-engine" {
		t.Fatalf("runtime config = %#v", cfg)
	}
	for name, want := range map[string]any{
		"class":                     "studio-flow-engine-host",
		"data-gosx-studio-engine":   "flow-designer",
		"data-studio-engine-role":   "flow-designer",
		"data-studio-engine-source": "gosx",
	} {
		if got := cfg.MountAttrs[name]; got != want {
			t.Fatalf("mount attr %s = %#v, want %#v in %#v", name, got, want, cfg.MountAttrs)
		}
	}

	for _, fragment := range []string{
		`<div class="studio-engine-hosts" aria-hidden="true" data-gosx-studio-engines="true" data-gosx-studio-engine-hosts-renderer="gosx-studio">`,
		`id="gosx-studio-flow-engine"`,
		`data-gosx-engine="GoSXStudioFlowDesigner"`,
		`data-gosx-engine-id="gosx-engine-0"`,
		`data-gosx-engine-kind="surface"`,
		`data-gosx-enhance="engine"`,
		`data-gosx-enhance-layer="runtime"`,
		`data-gosx-fallback="none"`,
		`class="studio-flow-engine-host"`,
		`data-gosx-studio-engine="flow-designer"`,
		`data-studio-engine-role="flow-designer"`,
		`data-studio-engine-source="gosx"`,
		`data-gosx-engine-capabilities="canvas pointer keyboard text-input animation"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("runtime engine hosts render missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderStudioEngineHostsFallbackRendersSurfaceContract(t *testing.T) {
	html := gosx.RenderHTML(RenderStudioEngineHosts([]map[string]any{{
		"key":          "flow-designer",
		"name":         FlowDesignerName,
		"mountId":      "gosx-studio-flow-engine",
		"class":        "studio-flow-engine-host",
		"capabilities": []string{"canvas", "pointer", "keyboard", "text-input", "animation"},
	}}, StudioEngineHostsOptions{}))

	for _, fragment := range []string{
		`<div class="studio-engine-hosts" aria-hidden="true" data-gosx-studio-engines="true" data-gosx-studio-engine-hosts-renderer="gosx-studio">`,
		`id="gosx-studio-flow-engine"`,
		`class="studio-flow-engine-host"`,
		`data-gosx-engine="GoSXStudioFlowDesigner"`,
		`data-gosx-engine-id="static-flow-designer"`,
		`data-gosx-engine-kind="surface"`,
		`data-gosx-enhance="engine"`,
		`data-gosx-enhance-layer="runtime"`,
		`data-gosx-fallback="none"`,
		`data-gosx-studio-engine="flow-designer"`,
		`data-studio-engine-role="flow-designer"`,
		`data-studio-engine-source="gosx"`,
		`data-gosx-engine-capabilities="canvas pointer keyboard text-input animation"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("engine hosts render missing %q:\n%s", fragment, html)
		}
	}
}

func TestRenderStudioEngineHostsRendersMultipleHosts(t *testing.T) {
	html := gosx.RenderHTML(RenderStudioEngineHosts([]map[string]any{
		{
			"key":          "flow-designer",
			"name":         FlowDesignerName,
			"mountId":      "gosx-studio-flow-engine",
			"class":        "studio-flow-engine-host",
			"capabilities": "canvas pointer",
		},
		{
			"key":          "showcase-3d",
			"name":         Showcase3DEngineName,
			"mountId":      "gosx-studio-showcase-3d-engine",
			"class":        "studio-showcase-engine-host",
			"capabilities": []any{"canvas", "webgl", "webgpu", "pointer"},
		},
	}, StudioEngineHostsOptions{}))

	for _, fragment := range []string{
		`data-gosx-studio-engine="flow-designer"`,
		`data-gosx-engine-capabilities="canvas pointer"`,
		`id="gosx-studio-showcase-3d-engine"`,
		`data-gosx-engine="GoSXStudioShowcase3D"`,
		`data-gosx-engine-capabilities="canvas webgl webgpu pointer"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("multiple engine hosts render missing %q:\n%s", fragment, html)
		}
	}
	if strings.Count(html, `data-gosx-engine-kind="surface"`) != 2 {
		t.Fatalf("expected two surface engine hosts:\n%s", html)
	}
}

func TestRenderStudioEngineHostsBlankAndMinimalBehavior(t *testing.T) {
	blankHTML := gosx.RenderHTML(RenderStudioEngineHosts(nil, StudioEngineHostsOptions{}))
	if blankHTML != `<div class="studio-engine-hosts" aria-hidden="true" data-gosx-studio-engines="true" data-gosx-studio-engine-hosts-renderer="gosx-studio"></div>` {
		t.Fatalf("blank render = %s", blankHTML)
	}
	if strings.Contains(blankHTML, "<form") || strings.Contains(blankHTML, "csrf") {
		t.Fatalf("blank render must not emit forms or csrf: %s", blankHTML)
	}

	minimalHTML := gosx.RenderHTML(RenderStudioEngineHosts([]map[string]any{{
		"key":     "custom",
		"name":    "CustomEngine",
		"mountId": "custom-engine",
		"class":   "custom-engine-host",
	}}, StudioEngineHostsOptions{EngineSource: "host-app"}))
	for _, fragment := range []string{
		`id="custom-engine"`,
		`data-gosx-engine="CustomEngine"`,
		`data-studio-engine-source="host-app"`,
	} {
		if !strings.Contains(minimalHTML, fragment) {
			t.Fatalf("minimal render missing %q:\n%s", fragment, minimalHTML)
		}
	}
	if strings.Contains(minimalHTML, "data-gosx-engine-capabilities") {
		t.Fatalf("minimal render should omit capabilities when none are supplied:\n%s", minimalHTML)
	}
}

func TestRenderStudioEngineHostsRootAttrsOverride(t *testing.T) {
	html := gosx.RenderHTML(RenderStudioEngineHosts(nil, StudioEngineHostsOptions{
		Class: "ignored-by-root-override",
		RootAttrs: map[string]any{
			"class":                    "custom-hosts",
			"aria-hidden":              "false",
			"data-gosx-studio-engines": "custom",
			"data-extra":               "yes",
		},
	}))

	for _, fragment := range []string{
		`class="custom-hosts"`,
		`aria-hidden="false"`,
		`data-gosx-studio-engines="custom"`,
		`data-gosx-studio-engine-hosts-renderer="gosx-studio"`,
		`data-extra="yes"`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("root attrs override missing %q:\n%s", fragment, html)
		}
	}
	if strings.Contains(html, "ignored-by-root-override") {
		t.Fatalf("root class override did not win:\n%s", html)
	}
}

func TestRenderStudioEngineHostsDoesNotEmitFormsOrCSRF(t *testing.T) {
	html := gosx.RenderHTML(RenderStudioEngineHosts([]map[string]any{{
		"key":          "flow-designer",
		"name":         FlowDesignerName,
		"mountId":      "gosx-studio-flow-engine",
		"class":        "studio-flow-engine-host",
		"capabilities": "canvas pointer keyboard",
	}}, StudioEngineHostsOptions{}))

	for _, forbidden := range []string{"<form", "</form>", "csrf", "csrf_token"} {
		if strings.Contains(html, forbidden) {
			t.Fatalf("engine host renderer should not emit %q:\n%s", forbidden, html)
		}
	}
}
