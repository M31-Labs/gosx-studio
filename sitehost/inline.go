package sitehost

import (
	"net/url"
	"strings"

	"m31labs.dev/gosx"
)

// inline.go is the formatting inside a block of text: bold, italic, links,
// and line breaks.
//
// It is stored as plain text with three markers — **bold**, _italic_, and
// [label](address) — never as HTML. That is a security decision as much as a
// simplicity one: nothing a browser or a bad actor produces can reach the
// page as markup, because the renderer escapes every character of text and
// only ever emits the four tags it knows. The editor turns a contenteditable
// selection into these markers and back.

// renderInline renders one block's text. Unknown or malformed markers
// render as the literal characters, so nothing is ever silently lost.
func renderInline(text string) gosx.Node {
	nodes := make([]gosx.Node, 0, 8)
	var run strings.Builder
	flush := func() {
		if run.Len() > 0 {
			nodes = append(nodes, gosx.Text(run.String()))
			run.Reset()
		}
	}
	i := 0
	for i < len(text) {
		switch {
		case text[i] == '\n':
			flush()
			nodes = append(nodes, gosx.El("br", nil))
			i++
		case strings.HasPrefix(text[i:], "**"):
			end := strings.Index(text[i+2:], "**")
			if end <= 0 {
				run.WriteString("**")
				i += 2
				continue
			}
			flush()
			nodes = append(nodes, gosx.El("strong", nil, renderInline(text[i+2:i+2+end])))
			i += 2 + end + 2
		case text[i] == '_' && (i == 0 || !isWordByte(text[i-1])):
			end := strings.Index(text[i+1:], "_")
			if end <= 0 || (i+1+end+1 < len(text) && isWordByte(text[i+1+end+1])) {
				run.WriteByte('_')
				i++
				continue
			}
			flush()
			nodes = append(nodes, gosx.El("em", nil, renderInline(text[i+1:i+1+end])))
			i += 1 + end + 1
		case text[i] == '[':
			label, href, consumed := parseLink(text[i:])
			if consumed == 0 {
				run.WriteByte('[')
				i++
				continue
			}
			flush()
			nodes = append(nodes, gosx.El("a", gosx.Attrs(linkAttrs(href)...), renderInline(label)))
			i += consumed
		default:
			run.WriteByte(text[i])
			i++
		}
	}
	flush()
	return gosx.Fragment(nodes...)
}

func isWordByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// parseLink reads "[label](address)" at the start of s.
func parseLink(s string) (label, href string, consumed int) {
	close := strings.Index(s, "](")
	if close < 1 {
		return "", "", 0
	}
	end := strings.Index(s[close+2:], ")")
	if end < 0 {
		return "", "", 0
	}
	label = s[1:close]
	href = strings.TrimSpace(s[close+2 : close+2+end])
	if label == "" || href == "" || strings.ContainsAny(label, "\n") {
		return "", "", 0
	}
	if safeLinkHref(href) == "" {
		return "", "", 0
	}
	return label, href, close + 2 + end + 1
}

// safeLinkHref allows relative paths, http(s), mailto, and tel. Anything
// else — javascript:, data:, a bare word — is not a link.
func safeLinkHref(href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.ContainsAny(href, " \t\r\n\"<>") {
		return ""
	}
	if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
		return href
	}
	if strings.HasPrefix(href, "#") {
		return href
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		if parsed.Host == "" {
			return ""
		}
		return href
	case "mailto", "tel":
		return href
	}
	return ""
}

func linkAttrs(href string) []any {
	attrs := []any{gosx.Attr("href", href)}
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		attrs = append(attrs, gosx.Attr("rel", "noopener"))
	}
	return attrs
}

// inlineToPlain strips the markers, for places that need bare text such as
// a page's <title> or a search description. It pairs markers exactly as
// renderInline does, so an underscore inside a word is left alone.
func inlineToPlain(text string) string {
	var b strings.Builder
	i := 0
	for i < len(text) {
		switch {
		case strings.HasPrefix(text[i:], "**"):
			end := strings.Index(text[i+2:], "**")
			if end <= 0 {
				b.WriteString("**")
				i += 2
				continue
			}
			b.WriteString(inlineToPlain(text[i+2 : i+2+end]))
			i += 2 + end + 2
		case text[i] == '_' && (i == 0 || !isWordByte(text[i-1])):
			end := strings.Index(text[i+1:], "_")
			if end <= 0 || (i+1+end+1 < len(text) && isWordByte(text[i+1+end+1])) {
				b.WriteByte('_')
				i++
				continue
			}
			b.WriteString(inlineToPlain(text[i+1 : i+1+end]))
			i += 1 + end + 1
		case text[i] == '[':
			if label, _, consumed := parseLink(text[i:]); consumed > 0 {
				b.WriteString(inlineToPlain(label))
				i += consumed
				continue
			}
			b.WriteByte('[')
			i++
		default:
			b.WriteByte(text[i])
			i++
		}
	}
	return b.String()
}
