package sitehost

import (
	"net/url"
	"strconv"
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-studio/cms/content"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

// text.go converts between the block document the store holds and the plain
// text the default host's editing form shows.
//
// The form is deliberately line-based rather than a block palette: a person
// editing their own website should be able to type. Studio's richer block
// editor mounts on the same documents when the workbench route lands.

func values(pairs ...string) blockstudio.Values {
	out := blockstudio.Values{}
	for i := 0; i+1 < len(pairs); i += 2 {
		out[pairs[i]] = text(pairs[i+1])
	}
	return out
}

// documentToText renders a block document as editable lines. Headings keep a
// leading "# " and quotes a leading "> " so the round trip is lossless for the
// block kinds this form supports.
func documentToText(doc blockstudio.Document) string {
	lines := make([]string, 0, len(doc.Blocks))
	for _, instance := range doc.Blocks {
		value := strings.TrimSpace(instance.Values["text"].String)
		switch instance.Key {
		case content.BlockHeading:
			level := strings.TrimSpace(instance.Values["level"].String)
			marker := "#"
			if level == "3" {
				marker = "##"
			} else if level == "4" {
				marker = "###"
			}
			if value != "" {
				lines = append(lines, marker+" "+value)
			}
		case content.BlockQuote:
			if value != "" {
				lines = append(lines, "> "+value)
			}
		case content.BlockButton:
			label := strings.TrimSpace(instance.Values["label"].String)
			target := strings.TrimSpace(instance.Values["url"].String)
			if label != "" {
				lines = append(lines, "["+label+"]("+target+")")
			}
		default:
			if value != "" {
				lines = append(lines, value)
			}
		}
	}
	return strings.Join(lines, "\n\n")
}

// textToDocument parses the editing form's plain text back into blocks.
func textToDocument(body string) blockstudio.Document {
	blocks := make([]blockstudio.BlockInstance, 0, 8)
	order := 0
	for _, raw := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		switch {
		case strings.HasPrefix(line, "### "):
			blocks = append(blocks, block(order, content.BlockHeading,
				values("text", strings.TrimSpace(line[4:]), "level", "4")))
		case strings.HasPrefix(line, "## "):
			blocks = append(blocks, block(order, content.BlockHeading,
				values("text", strings.TrimSpace(line[3:]), "level", "3")))
		case strings.HasPrefix(line, "# "):
			blocks = append(blocks, block(order, content.BlockHeading,
				values("text", strings.TrimSpace(line[2:]), "level", "2")))
		case strings.HasPrefix(line, "> "):
			blocks = append(blocks, block(order, content.BlockQuote,
				values("text", strings.TrimSpace(line[2:]))))
		default:
			if label, target, ok := parseLinkLine(line); ok {
				blocks = append(blocks, block(order, content.BlockButton,
					values("label", label, "url", target)))
				break
			}
			blocks = append(blocks, block(order, content.BlockParagraph, values("text", line)))
		}
		order++
	}
	return document(blocks...)
}

// parseLinkLine recognizes a whole-line "[label](/target)" as a button.
func parseLinkLine(line string) (string, string, bool) {
	if !strings.HasPrefix(line, "[") || !strings.HasSuffix(line, ")") {
		return "", "", false
	}
	close := strings.Index(line, "](")
	if close < 1 {
		return "", "", false
	}
	label := strings.TrimSpace(line[1:close])
	target := strings.TrimSpace(line[close+2 : len(line)-1])
	if label == "" || target == "" {
		return "", "", false
	}
	return label, target, true
}

// normalizeSlug turns a title or a typed address into a safe URL segment.
func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := true
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == ' ' || r == '_' || r == '/':
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func revisionFilterAll() lifecycle.Filter { return lifecycle.Filter{} }

func itoa(value int) string { return strconv.Itoa(value) }

func queryEscape(value string) string { return url.QueryEscape(value) }
