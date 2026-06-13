package studio

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

// canvasBundleScript extracts the JSON text content of the inline
// <script type="application/json" data-gosx-canvas-bundle> the canvas engine
// emits. The renderer embeds the bundle via gosx.RawHTML (encoding/json already
// escapes <,>,& as \uXXXX, so no </script> can appear and no HTML unescaping is
// needed). The capture is non-greedy so a future second script can't swallow it.
var canvasBundleScript = regexp.MustCompile(
	`(?s)<script type="application/json" data-gosx-canvas-bundle="[^"]*">(.*?)</script>`,
)

// siteMapCanvasBundle mirrors the painter-facing bundle shape (the JSON the JS
// canvas2d painter consumes). Only the fields this test asserts are modeled;
// json.Unmarshal ignores the rest, which is the point — the inline JSON must
// parse into exactly this structure.
type siteMapCanvasBundle struct {
	Background string `json:"background"`
	Camera     struct {
		Mode string `json:"mode"`
	} `json:"camera"`
	Materials []struct {
		Color string `json:"color"`
	} `json:"materials"`
	Objects []struct {
		ID            string `json:"id"`
		Kind          string `json:"kind"`
		MaterialIndex int    `json:"materialIndex"`
		Bounds        struct {
			MinX float64 `json:"minX"`
			MinY float64 `json:"minY"`
			MaxX float64 `json:"maxX"`
			MaxY float64 `json:"maxY"`
		} `json:"bounds"`
	} `json:"objects"`
	Lines []struct {
		From struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"from"`
		To struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"to"`
		Color string `json:"color"`
	} `json:"lines"`
	Labels []struct {
		Text     string `json:"text"`
		Position struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"position"`
	} `json:"labels"`
}

func parseInlineCanvasBundle(t *testing.T, htmlOut string) siteMapCanvasBundle {
	t.Helper()
	m := canvasBundleScript.FindStringSubmatch(htmlOut)
	if m == nil {
		t.Fatalf("inline canvas bundle script not found in rendered HTML:\n%s", htmlOut)
	}
	var bundle siteMapCanvasBundle
	if err := json.Unmarshal([]byte(m[1]), &bundle); err != nil {
		t.Fatalf("inline canvas bundle JSON did not parse: %v\nJSON: %s", err, m[1])
	}
	return bundle
}

// TestRenderSiteMapCanvasEngineDefaultOmitsInlineServerBundle locks the M2
// default path: enabled non-WASMFree rendering goes through the real
// gosx.CanvasBoard WebGPU backend and does not emit the legacy inline
// server-side bundle.
func TestRenderSiteMapCanvasEngineDefaultOmitsInlineServerBundle(t *testing.T) {
	htmlOut := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{
		Enabled: true,
	}))

	if strings.Contains(htmlOut, "data-gosx-canvas-bundle") {
		t.Fatalf("default enabled canvas engine must not emit an inline bundle:\n%s", htmlOut)
	}
}

// TestRenderSiteMapCanvasEngineWASMFreeEmitsInlineServerBundle is the
// compatibility-path acceptance for gosx-studio: with WASMFree=true the enabled
// canvas engine emits the server-precomputed RenderBundle as inline JSON on the
// surface, and that JSON parses into a bundle whose objects/labels/lines match
// the known site map. The same view yields 3 workspace rects, 3 title labels + 1
// route subtitle label for the single routed page (page:home → "/"), and 2
// links (see TestSiteMapCanvasNodesDeriveFromWorkspaceLayers) — so the bundle
// must carry 3 rect objects, 4 labels, and 2 lines. The route subtitle flows
// through the shared bundle adapter identically to the title labels.
func TestRenderSiteMapCanvasEngineWASMFreeEmitsInlineServerBundle(t *testing.T) {
	htmlOut := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{
		Enabled:  true,
		WASMFree: true,
	}))

	// The script element itself must be present (data-attribute hook the
	// WASM-free client locates) and typed as a JSON data block.
	if !strings.Contains(htmlOut, `<script type="application/json" data-gosx-canvas-bundle="studio-site-map-canvas-board">`) {
		t.Fatalf("expected inline canvas-bundle <script>, not found:\n%s", htmlOut)
	}

	bundle := parseInlineCanvasBundle(t, htmlOut)

	if got := len(bundle.Objects); got != 3 {
		t.Errorf("inline bundle objects (rects) = %d, want 3", got)
	}
	if got := len(bundle.Labels); got != 4 {
		t.Errorf("inline bundle labels = %d, want 4", got)
	}
	if got := len(bundle.Lines); got != 2 {
		t.Errorf("inline bundle lines = %d, want 2", got)
	}

	// Camera is the 2D orthographic mode the painter keys on.
	if bundle.Camera.Mode != "ortho2d" {
		t.Errorf("inline bundle camera mode = %q, want ortho2d", bundle.Camera.Mode)
	}
	// Background propagates from the canvas surface default.
	if bundle.Background != siteMapCanvasDefaultBackground {
		t.Errorf("inline bundle background = %q, want %q", bundle.Background, siteMapCanvasDefaultBackground)
	}

	// Object identity + material wiring: the page-home rect must be present,
	// carry rect kind, real bounds, and a non-empty fill color resolved via
	// materialIndex (the lookup the painter uses).
	var homeRect *struct {
		ID            string
		Kind          string
		MaterialIndex int
		Color         string
		HasArea       bool
	}
	for _, obj := range bundle.Objects {
		if obj.ID != "page:home" {
			continue
		}
		color := ""
		if obj.MaterialIndex >= 0 && obj.MaterialIndex < len(bundle.Materials) {
			color = bundle.Materials[obj.MaterialIndex].Color
		}
		homeRect = &struct {
			ID            string
			Kind          string
			MaterialIndex int
			Color         string
			HasArea       bool
		}{
			ID:            obj.ID,
			Kind:          obj.Kind,
			MaterialIndex: obj.MaterialIndex,
			Color:         color,
			HasArea:       obj.Bounds.MaxX > obj.Bounds.MinX && obj.Bounds.MaxY > obj.Bounds.MinY,
		}
	}
	if homeRect == nil {
		t.Fatalf("inline bundle missing the page:home rect object:\n%s", htmlOut)
	}
	if homeRect.Kind != "rect" {
		t.Errorf("page:home object kind = %q, want rect", homeRect.Kind)
	}
	if !homeRect.HasArea {
		t.Errorf("page:home object has degenerate bounds")
	}
	if strings.TrimSpace(homeRect.Color) == "" {
		t.Errorf("page:home object resolved no fill color via materialIndex %d (materials=%d)", homeRect.MaterialIndex, len(bundle.Materials))
	}

	// Human-readable content survives into the bundle labels.
	if !bundleHasLabel(bundle, "Hero") {
		t.Errorf("inline bundle labels missing the Hero title; labels=%+v", bundle.Labels)
	}
	// The page route subtitle flows through the shared bundle adapter identically
	// to the title labels (M7-3): the routed page's "/" route must survive.
	if !bundleHasLabel(bundle, "/") {
		t.Errorf("inline bundle labels missing the page route subtitle %q; labels=%+v", "/", bundle.Labels)
	}

	// No visible platform copy regression.
	if strings.Contains(htmlOut, "GoSX Studio") {
		t.Fatalf("canvas engine must not inject visible platform copy:\n%s", htmlOut)
	}
}

// TestRenderSiteMapCanvasEngineDisabledEmitsNoBundle confirms the additive
// emission stays behind the Enabled flag: the disabled path emits no inline
// bundle script (and still no canvas), so hosts that never opt in are
// byte-for-byte unchanged.
func TestRenderSiteMapCanvasEngineDisabledEmitsNoBundle(t *testing.T) {
	htmlOut := gosx.RenderHTML(RenderSiteMapCanvasEngine(siteMapCanvasTestView(), SiteMapCanvasOptions{}))
	if strings.Contains(htmlOut, "data-gosx-canvas-bundle") {
		t.Fatalf("disabled canvas engine must not emit an inline bundle:\n%s", htmlOut)
	}
}

func bundleHasLabel(bundle siteMapCanvasBundle, text string) bool {
	for _, label := range bundle.Labels {
		if strings.TrimSpace(label.Text) == text {
			return true
		}
	}
	return false
}
