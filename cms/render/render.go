package render

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	"m31labs.dev/gosx-studio/internal/mediaurl"
)

type Block map[string]any

type Context struct {
	Block Block
	Key   string
	Ref   string
}

// Hook output is caller-owned. Custom hooks must apply the same explicit URL
// contract to any src or href they create; built-in block serialization cannot
// inspect or rewrite arbitrary nodes returned by a hook.
type Hook func(Context) (gosx.Node, bool)

type Hooks struct {
	Product Hook
	Flow    Hook
	Unknown Hook
}

func Body(body string, hooks Hooks) gosx.Node {
	return Blocks(content.ViewBlocks(body), hooks)
}

func Document(doc blockstudio.Document, hooks Hooks) gosx.Node {
	return Blocks(content.ViewBlocksFromDocument(doc), hooks)
}

func Blocks(blocks []map[string]any, hooks Hooks) gosx.Node {
	nodes := make([]gosx.Node, 0, len(blocks))
	for _, raw := range blocks {
		block := Block(raw)
		if node, ok := RenderBlock(block, hooks); ok {
			nodes = append(nodes, node)
		}
	}
	return gosx.Fragment(nodes...)
}

func RenderBlock(block Block, hooks Hooks) (gosx.Node, bool) {
	switch {
	case boolField(block, "isHeading"):
		return gosx.El("h2", nil, gosx.Text(stringField(block, "text"))), true
	case boolField(block, "isQuote"):
		return gosx.El("blockquote", nil, gosx.Text(stringField(block, "text"))), true
	case boolField(block, "isImage"):
		return gosx.El("figure", nil, renderImage(stringField(block, "url"), stringField(block, "alt"))), true
	case boolField(block, "isGallery"):
		images := mapSliceField(block, "images")
		children := make([]gosx.Node, 0, len(images))
		for _, image := range images {
			children = append(children, renderImage(stringField(image, "url"), stringField(image, "alt")))
		}
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "media-strip")), gosx.Fragment(children...)), true
	case boolField(block, "isButton"):
		return renderButton(stringField(block, "href"), stringField(block, "label")), true
	case boolField(block, "isProduct"):
		if hooks.Product != nil {
			if node, ok := hooks.Product(Context{Block: block, Key: "product", Ref: stringField(block, "productRef")}); ok {
				return node, true
			}
		}
		if ref := stringField(block, "productRef"); ref != "" {
			return gosx.El("p", nil, gosx.Text(ref)), true
		}
	case boolField(block, "isFlow"):
		if hooks.Flow != nil {
			if node, ok := hooks.Flow(Context{Block: block, Key: "flow", Ref: stringField(block, "flowKey")}); ok {
				return node, true
			}
		}
	case boolField(block, "isParagraph"):
		return gosx.El("p", nil, gosx.Text(stringField(block, "text"))), true
	default:
		if hooks.Unknown != nil {
			return hooks.Unknown(Context{Block: block})
		}
	}
	return gosx.Node{}, false
}

func stringField(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func renderImage(rawURL, alt string) gosx.Node {
	if safeURL, ok := mediaurl.ForImage(rawURL); ok {
		return gosx.El("img", gosx.Attrs(
			gosx.Attr("src", safeURL),
			gosx.Attr("alt", alt),
		))
	}
	label := strings.TrimSpace(alt)
	if label == "" {
		label = "Image unavailable"
	}
	return gosx.El("span", gosx.Attrs(
		gosx.Attr("class", "media-fallback"),
		gosx.Attr("role", "img"),
		gosx.Attr("aria-label", label),
	), gosx.Text(label))
}

func renderButton(rawURL, label string) gosx.Node {
	if safeURL, ok := mediaurl.ForLink(rawURL); ok {
		return gosx.El("a", gosx.Attrs(
			gosx.Attr("class", "button button--primary"),
			gosx.Attr("href", safeURL),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(label))
	}
	return gosx.El("span", gosx.Attrs(
		gosx.Attr("class", "button button--primary"),
		gosx.Attr("aria-disabled", "true"),
	), gosx.Text(label))
}

func boolField(values map[string]any, key string) bool {
	value, ok := values[key]
	if !ok {
		return false
	}
	if typed, ok := value.(bool); ok {
		return typed
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(value)), "true")
}

func mapSliceField(values map[string]any, key string) []map[string]any {
	if values == nil {
		return nil
	}
	if typed, ok := values[key].([]map[string]any); ok {
		return typed
	}
	raw, ok := values[key].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapped, ok := item.(map[string]any); ok {
			out = append(out, mapped)
		}
	}
	return out
}
