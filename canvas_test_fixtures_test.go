package studio

// This file preserves test-only fixtures that used to live in
// sitemap_canvas_test.go and engine_hosts_test.go before Slice 4 of the
// package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) moved those files
// into m31labs.dev/gosx-studio/canvas. Two still-root-resident test files
// (backend_editor_workbench_test.go, backend_editor_page_test.go — shell
// package territory, Slice 8) call these unexported helpers by their
// original unqualified names; canvas carries its own equivalents for its own
// test suite (canvas/sitemap_canvas_test.go's siteMapCanvasTestView is a
// hand-built literal since canvas cannot import authoring/sitemap, and
// canvas/engine_hosts_test.go's recordingStudioEngineRuntime is unchanged),
// so this is a small, deliberate duplication rather than a shim, matching the
// same "still-root-resident file needs a helper from a moved file" pattern
// documented in the compat_*.go facades.

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx/engine"
)

// siteMapCanvasTestView builds the same AuthoringSiteMapView shape that
// sitemap_board_test.go exercises, so root-resident render tests are
// validated against the exact data the DOM board renders from.
func siteMapCanvasTestView() map[string]any {
	return AuthoringSiteMapView(NoCodeAuthoringSurface(SiteMap{
		Pages: []Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Status:        "Editable",
			Editable:      true,
			Selected:      true,
			Components: []Component{{
				Key:           "hero",
				TemplateKey:   "hero",
				Label:         "Hero",
				GoSXComponent: "HomeHero",
				Source:        ComponentSourceHost,
				Binding:       "home.section.hero",
				Status:        "Visible",
				Editable:      true,
				Visible:       true,
			}},
		}},
	}), SiteMapViewOptions{})
}

// recordingStudioEngineRuntime is a test double for the StudioEngineRuntime
// interface (canvas.StudioEngineRuntime via the compat_canvas.go alias) that
// records every engine.Config it is asked to render and emits a stable,
// assertable surface contract.
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
