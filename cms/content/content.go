package content

import (
	"encoding/json"
	"fmt"
	"strings"

	"m31labs.dev/gosx-admin/blockstudio"
)

const (
	BodyKind = "content.body"

	BlockParagraph = "paragraph"
	BlockHeading   = "heading"
	BlockQuote     = "quote"
	BlockImage     = "image"
	BlockGallery   = "gallery"
	BlockButton    = "button"
	BlockProduct   = "product"
	BlockFlow      = "flow"
)

type LegacyDocument struct {
	Version int           `json:"version"`
	Blocks  []LegacyBlock `json:"blocks"`
}

type LegacyBlock struct {
	ID         string  `json:"id,omitempty"`
	Type       string  `json:"type"`
	Text       string  `json:"text,omitempty"`
	URL        string  `json:"url,omitempty"`
	Alt        string  `json:"alt,omitempty"`
	Label      string  `json:"label,omitempty"`
	Href       string  `json:"href,omitempty"`
	ProductRef string  `json:"productRef,omitempty"`
	FlowKey    string  `json:"flowKey,omitempty"`
	Images     []Image `json:"images,omitempty"`
}

type Image struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

func Parse(body string) blockstudio.Document {
	if doc, ok := ParseLegacyV1(body); ok {
		return doc
	}
	return ParseLegacyText(body)
}

func ParseLegacyV1(body string) (blockstudio.Document, bool) {
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, "{") {
		return blockstudio.Document{}, false
	}
	var legacy LegacyDocument
	if err := json.Unmarshal([]byte(body), &legacy); err != nil {
		return blockstudio.Document{}, false
	}
	return DocumentFromLegacy(legacy), true
}

func ParseLegacyText(body string) blockstudio.Document {
	parts := strings.Split(strings.TrimSpace(body), "\n\n")
	blocks := make([]blockstudio.BlockInstance, 0, len(parts))
	for _, part := range parts {
		if block, ok := blockFromLegacyText(part, len(blocks)+1); ok {
			blocks = append(blocks, block)
		}
	}
	return blockstudio.Document{Version: 1, Kind: BodyKind, Blocks: blocks}
}

func DocumentFromLegacy(legacy LegacyDocument) blockstudio.Document {
	version := legacy.Version
	if version <= 0 {
		version = 1
	}
	doc := blockstudio.Document{Version: version, Kind: BodyKind, Blocks: make([]blockstudio.BlockInstance, 0, len(legacy.Blocks))}
	for _, block := range legacy.Blocks {
		if mapped, ok := blockFromLegacyBlock(block, len(doc.Blocks)+1); ok {
			doc.Blocks = append(doc.Blocks, mapped)
		}
	}
	return doc
}

func LegacyFromDocument(doc blockstudio.Document) LegacyDocument {
	version := doc.Version
	if version <= 0 {
		version = 1
	}
	legacy := LegacyDocument{Version: version, Blocks: make([]LegacyBlock, 0, len(doc.Blocks))}
	for _, block := range doc.Blocks {
		if mapped, ok := legacyBlockFromInstance(block); ok {
			legacy.Blocks = append(legacy.Blocks, mapped)
		}
	}
	return legacy
}

func SerializeLegacyV1(doc blockstudio.Document) (string, error) {
	data, err := json.MarshalIndent(LegacyFromDocument(doc), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ViewBlocks(body string) []map[string]any {
	return ViewBlocksFromDocument(Parse(body))
}

func ViewBlocksFromDocument(doc blockstudio.Document) []map[string]any {
	out := make([]map[string]any, 0, len(doc.Blocks))
	for _, instance := range doc.Blocks {
		if !instance.Enabled {
			continue
		}
		if block := viewBlock(instance); block != nil {
			out = append(out, block)
		}
	}
	return out
}

func blockFromLegacyBlock(input LegacyBlock, order int) (blockstudio.BlockInstance, bool) {
	key := normalizeKey(input.Type)
	text := strings.TrimSpace(input.Text)
	values := blockstudio.Values{}
	switch key {
	case BlockHeading:
		if text == "" {
			return blockstudio.BlockInstance{}, false
		}
		values["text"] = stringValue(blockstudio.FieldText, text)
	case BlockQuote:
		if text == "" {
			return blockstudio.BlockInstance{}, false
		}
		values["text"] = stringValue(blockstudio.FieldTextarea, text)
	case BlockImage:
		url := strings.TrimSpace(input.URL)
		if url == "" {
			return blockstudio.BlockInstance{}, false
		}
		values["url"] = mediaValue(url, strings.TrimSpace(input.Alt))
		values["alt"] = stringValue(blockstudio.FieldText, strings.TrimSpace(input.Alt))
	case BlockGallery:
		images := cleanImages(input.Images)
		if len(images) == 0 {
			return blockstudio.BlockInstance{}, false
		}
		values["images"] = imagesValue(images)
	case BlockButton:
		label := strings.TrimSpace(input.Label)
		href := strings.TrimSpace(input.Href)
		if label == "" || href == "" {
			return blockstudio.BlockInstance{}, false
		}
		values["label"] = stringValue(blockstudio.FieldText, label)
		values["href"] = stringValue(blockstudio.FieldURL, href)
	case BlockProduct:
		ref := strings.TrimSpace(input.ProductRef)
		if ref == "" {
			ref = text
		}
		if ref == "" {
			return blockstudio.BlockInstance{}, false
		}
		values["productRef"] = stringValue(blockstudio.FieldText, ref)
	case BlockFlow:
		flowKey := strings.TrimSpace(input.FlowKey)
		if flowKey == "" {
			flowKey = text
		}
		if flowKey == "" {
			return blockstudio.BlockInstance{}, false
		}
		values["flowKey"] = stringValue(blockstudio.FieldText, flowKey)
	case BlockParagraph:
		if text == "" {
			return blockstudio.BlockInstance{}, false
		}
		values["text"] = stringValue(blockstudio.FieldTextarea, collapseWhitespace(text))
	default:
		if text == "" {
			return blockstudio.BlockInstance{}, false
		}
		key = BlockParagraph
		values["text"] = stringValue(blockstudio.FieldTextarea, collapseWhitespace(text))
	}
	return newInstance(firstNonEmpty(input.ID, instanceID(key, order)), key, order, values), true
}

func blockFromLegacyText(raw string, order int) (blockstudio.BlockInstance, bool) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return blockstudio.BlockInstance{}, false
	}
	key := BlockParagraph
	values := blockstudio.Values{"text": stringValue(blockstudio.FieldTextarea, collapseWhitespace(text))}
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(text, "## "):
		key = BlockHeading
		values = blockstudio.Values{"text": stringValue(blockstudio.FieldText, strings.TrimSpace(strings.TrimPrefix(text, "## ")))}
	case strings.HasPrefix(text, "> "):
		key = BlockQuote
		values = blockstudio.Values{"text": stringValue(blockstudio.FieldTextarea, strings.TrimSpace(strings.TrimPrefix(text, "> ")))}
	case strings.HasPrefix(lower, "[image:") && strings.HasSuffix(text, "]"):
		url, alt := ParseMediaLine(strings.TrimSpace(text[len("[image:") : len(text)-1]))
		if url != "" {
			key = BlockImage
			values = blockstudio.Values{
				"url": mediaValue(url, alt),
				"alt": stringValue(blockstudio.FieldText, alt),
			}
		}
	case strings.HasPrefix(lower, "[gallery:") && strings.HasSuffix(text, "]"):
		inner := strings.TrimSpace(text[len("[gallery:") : len(text)-1])
		lines := strings.FieldsFunc(inner, func(r rune) bool {
			return r == ';' || r == '\n'
		})
		images := make([]Image, 0, len(lines))
		for _, line := range lines {
			url, alt := ParseMediaLine(line)
			if url != "" {
				images = append(images, Image{URL: url, Alt: alt})
			}
		}
		if len(images) > 0 {
			key = BlockGallery
			values = blockstudio.Values{"images": imagesValue(images)}
		}
	case strings.HasPrefix(lower, "[button:") && strings.HasSuffix(text, "]"):
		label, href := ParseMediaLine(strings.TrimSpace(text[len("[button:") : len(text)-1]))
		if label != "" && href != "" {
			key = BlockButton
			values = blockstudio.Values{
				"label": stringValue(blockstudio.FieldText, label),
				"href":  stringValue(blockstudio.FieldURL, href),
			}
		}
	case strings.HasPrefix(lower, "[product:") && strings.HasSuffix(text, "]"):
		ref := strings.TrimSpace(text[len("[product:") : len(text)-1])
		if ref != "" {
			key = BlockProduct
			values = blockstudio.Values{"productRef": stringValue(blockstudio.FieldText, ref)}
		}
	}
	return newInstance(instanceID(key, order), key, order, values), true
}

func viewBlock(instance blockstudio.BlockInstance) map[string]any {
	key := normalizeKey(instance.Key)
	block := emptyViewBlock()
	switch key {
	case BlockHeading:
		text := valueString(instance.Values["text"])
		if text == "" {
			return nil
		}
		block["text"] = text
		block["isParagraph"] = false
		block["isHeading"] = true
	case BlockQuote:
		text := valueString(instance.Values["text"])
		if text == "" {
			return nil
		}
		block["text"] = text
		block["isParagraph"] = false
		block["isQuote"] = true
	case BlockImage:
		url := mediaURL(instance.Values["url"])
		if url == "" {
			url = valueString(instance.Values["url"])
		}
		if url == "" {
			return nil
		}
		alt := firstNonEmpty(valueString(instance.Values["alt"]), mediaAlt(instance.Values["url"]))
		block["text"] = alt
		block["url"] = url
		block["alt"] = alt
		block["isParagraph"] = false
		block["isImage"] = true
	case BlockGallery:
		images := imagesFromValue(instance.Values["images"])
		if len(images) == 0 {
			return nil
		}
		block["images"] = images
		block["isParagraph"] = false
		block["isGallery"] = true
	case BlockButton:
		label := valueString(instance.Values["label"])
		href := valueString(instance.Values["href"])
		if label == "" || href == "" {
			return nil
		}
		block["text"] = label
		block["label"] = label
		block["href"] = href
		block["isParagraph"] = false
		block["isButton"] = true
	case BlockProduct:
		ref := valueString(instance.Values["productRef"])
		if ref == "" {
			ref = valueString(instance.Values["text"])
		}
		if ref == "" {
			return nil
		}
		block["text"] = ref
		block["productRef"] = ref
		block["isParagraph"] = false
		block["isProduct"] = true
	case BlockFlow:
		flowKey := valueString(instance.Values["flowKey"])
		if flowKey == "" {
			return nil
		}
		block["text"] = flowKey
		block["flowKey"] = flowKey
		block["isParagraph"] = false
		block["isFlow"] = true
	default:
		text := valueString(instance.Values["text"])
		if text == "" {
			return nil
		}
		block["text"] = collapseWhitespace(text)
	}
	return block
}

func legacyBlockFromInstance(instance blockstudio.BlockInstance) (LegacyBlock, bool) {
	key := normalizeKey(instance.Key)
	block := LegacyBlock{ID: strings.TrimSpace(instance.ID), Type: key}
	switch key {
	case BlockHeading, BlockQuote, BlockParagraph:
		block.Text = valueString(instance.Values["text"])
	case BlockImage:
		block.URL = firstNonEmpty(mediaURL(instance.Values["url"]), valueString(instance.Values["url"]))
		block.Alt = firstNonEmpty(valueString(instance.Values["alt"]), mediaAlt(instance.Values["url"]))
	case BlockGallery:
		images := imagesFromValue(instance.Values["images"])
		block.Images = make([]Image, 0, len(images))
		for _, image := range images {
			block.Images = append(block.Images, Image{URL: valueStringFromMap(image, "url"), Alt: valueStringFromMap(image, "alt")})
		}
	case BlockButton:
		block.Label = valueString(instance.Values["label"])
		block.Href = valueString(instance.Values["href"])
	case BlockProduct:
		block.ProductRef = valueString(instance.Values["productRef"])
	case BlockFlow:
		block.FlowKey = valueString(instance.Values["flowKey"])
	default:
		text := valueString(instance.Values["text"])
		if text == "" {
			return LegacyBlock{}, false
		}
		block.Type = BlockParagraph
		block.Text = text
	}
	return block, true
}

func emptyViewBlock() map[string]any {
	return map[string]any{
		"text":        "",
		"isParagraph": true,
		"isHeading":   false,
		"isQuote":     false,
		"isImage":     false,
		"isGallery":   false,
		"isButton":    false,
		"isProduct":   false,
		"url":         "",
		"alt":         "",
		"label":       "",
		"href":        "",
		"productRef":  "",
		"flowKey":     "",
		"hasProduct":  false,
		"isFlow":      false,
		"images":      []map[string]any{},
	}
}

func ParseMediaLine(line string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(line), "|", 2)
	url := strings.TrimSpace(parts[0])
	alt := ""
	if len(parts) > 1 {
		alt = strings.TrimSpace(parts[1])
	}
	return url, alt
}

func newInstance(id, key string, order int, values blockstudio.Values) blockstudio.BlockInstance {
	return blockstudio.BlockInstance{
		ID:      strings.TrimSpace(id),
		Key:     normalizeKey(key),
		Enabled: true,
		Order:   order,
		Values:  values,
	}
}

func stringValue(kind blockstudio.FieldKind, value string) blockstudio.Value {
	return blockstudio.Value{Kind: kind, String: strings.TrimSpace(value)}
}

func mediaValue(url, alt string) blockstudio.Value {
	url = strings.TrimSpace(url)
	alt = strings.TrimSpace(alt)
	return blockstudio.Value{
		Kind:   blockstudio.FieldImage,
		String: url,
		Media:  &blockstudio.MediaValue{URL: url, Alt: alt},
	}
}

func imagesValue(images []Image) blockstudio.Value {
	list := make([]blockstudio.Value, 0, len(images))
	for _, image := range cleanImages(images) {
		list = append(list, blockstudio.Value{Object: map[string]blockstudio.Value{
			"url": mediaValue(image.URL, image.Alt),
			"alt": stringValue(blockstudio.FieldText, image.Alt),
		}})
	}
	return blockstudio.Value{List: list}
}

func imagesFromValue(value blockstudio.Value) []map[string]any {
	out := make([]map[string]any, 0, len(value.List))
	for _, item := range value.List {
		url := firstNonEmpty(mediaURL(item.Object["url"]), valueString(item.Object["url"]))
		if url == "" {
			continue
		}
		alt := firstNonEmpty(valueString(item.Object["alt"]), mediaAlt(item.Object["url"]))
		out = append(out, map[string]any{"url": url, "alt": alt})
	}
	return out
}

func cleanImages(images []Image) []Image {
	out := make([]Image, 0, len(images))
	for _, image := range images {
		url := strings.TrimSpace(image.URL)
		if url == "" {
			continue
		}
		out = append(out, Image{URL: url, Alt: strings.TrimSpace(image.Alt)})
	}
	return out
}

func valueString(value blockstudio.Value) string {
	if value.Media != nil && strings.TrimSpace(value.Media.URL) != "" {
		return strings.TrimSpace(value.Media.URL)
	}
	return strings.TrimSpace(value.String)
}

func mediaURL(value blockstudio.Value) string {
	if value.Media == nil {
		return ""
	}
	return strings.TrimSpace(value.Media.URL)
}

func mediaAlt(value blockstudio.Value) string {
	if value.Media == nil {
		return ""
	}
	return strings.TrimSpace(value.Media.Alt)
}

func valueStringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(values[key]))
}

func normalizeKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func instanceID(key string, order int) string {
	if order <= 0 {
		order = 1
	}
	return fmt.Sprintf("%s-%d", normalizeKey(key), order)
}

func collapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
