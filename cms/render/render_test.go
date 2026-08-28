package render

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
)

func TestBodyRendersGenericBlocks(t *testing.T) {
	body := "## Studio note\n\n> Clay remembers the hand.\n\n[image: /media/cup.jpg | Cup]\n\n[gallery: /media/a.jpg | A; /media/b.jpg | B]\n\n[button: Shop now | /shop]\n\nPlain paragraph."
	html := gosx.RenderHTML(Body(body, Hooks{}))
	for _, want := range []string{
		"<h2>Studio note</h2>",
		"<blockquote>Clay remembers the hand.</blockquote>",
		`<img src="/media/cup.jpg" alt="Cup" />`,
		`class="media-strip"`,
		`href="/shop"`,
		"<p>Plain paragraph.</p>",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in rendered body: %s", want, html)
		}
	}
}

func TestProductHookRendersAppOwnedCards(t *testing.T) {
	html := gosx.RenderHTML(Body("[product: cloud-slip-bowl]", Hooks{
		Product: func(ctx Context) (gosx.Node, bool) {
			if ctx.Ref != "cloud-slip-bowl" {
				t.Fatalf("unexpected product ref: %#v", ctx)
			}
			return gosx.El("article", gosx.Attrs(gosx.Attr("class", "product-card")), gosx.Text(ctx.Ref)), true
		},
	}))
	if !strings.Contains(html, `class="product-card"`) || !strings.Contains(html, "cloud-slip-bowl") {
		t.Fatalf("expected product hook output, got %s", html)
	}
}

func TestFlowHookRendersFlowEmbeds(t *testing.T) {
	doc := blockstudio.Document{Blocks: []blockstudio.BlockInstance{{
		Key:     content.BlockFlow,
		Enabled: true,
		Values: blockstudio.Values{
			"flowKey": {Kind: blockstudio.FieldText, String: "schedule-tour"},
		},
	}}}
	html := gosx.RenderHTML(Document(doc, Hooks{
		Flow: func(ctx Context) (gosx.Node, bool) {
			return gosx.El("section", gosx.Attrs(gosx.Attr("data-flow", ctx.Ref)), gosx.Text("Flow")), true
		},
	}))
	if !strings.Contains(html, `data-flow="schedule-tour"`) {
		t.Fatalf("expected flow hook output, got %s", html)
	}
}

func TestProductFallbackRendersReference(t *testing.T) {
	html := gosx.RenderHTML(Body("[product: cloud-slip-bowl]", Hooks{}))
	if !strings.Contains(html, "<p>cloud-slip-bowl</p>") {
		t.Fatalf("expected product fallback paragraph, got %s", html)
	}
}

func TestRenderBlockAppliesExplicitImageAndLinkURLContract(t *testing.T) {
	unsafeImage := gosx.RenderHTML(mustRenderBlock(t, Block{
		"isImage": true,
		"url":     "javascript:alert(1)",
		"alt":     `<Cup & vase>`,
	}))
	if strings.Contains(unsafeImage, "javascript:") || strings.Contains(unsafeImage, "<img") {
		t.Fatalf("unsafe image destination should be inert, got %s", unsafeImage)
	}
	for _, want := range []string{
		`class="media-fallback"`,
		`role="img"`,
		`aria-label="&lt;Cup &amp; vase&gt;"`,
		`&lt;Cup &amp; vase&gt;`,
	} {
		if !strings.Contains(unsafeImage, want) {
			t.Fatalf("inert image fallback missing %q: %s", want, unsafeImage)
		}
	}

	unsafeLink := gosx.RenderHTML(mustRenderBlock(t, Block{
		"isButton": true,
		"href":     "data:text/html,alert(1)",
		"label":    `<Buy & learn>`,
	}))
	if strings.Contains(unsafeLink, "href=") || strings.Contains(unsafeLink, "data:") {
		t.Fatalf("unsafe link destination should be inert, got %s", unsafeLink)
	}
	for _, want := range []string{`class="button button--primary"`, `aria-disabled="true"`, `&lt;Buy &amp; learn&gt;`} {
		if !strings.Contains(unsafeLink, want) {
			t.Fatalf("inert link fallback missing %q: %s", want, unsafeLink)
		}
	}

	validLinks := gosx.RenderHTML(Body("[button: Email us | mailto:artist@example.test]", Hooks{}))
	if !strings.Contains(validLinks, `href="mailto:artist@example.test"`) {
		t.Fatalf("mailto link should remain supported: %s", validLinks)
	}
}

func mustRenderBlock(t *testing.T, block Block) gosx.Node {
	t.Helper()
	node, ok := RenderBlock(block, Hooks{})
	if !ok {
		t.Fatalf("RenderBlock(%#v) did not render", block)
	}
	return node
}
