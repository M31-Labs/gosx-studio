package content

import (
	"strings"
	"testing"

	"m31labs.dev/gosx-admin/blockstudio"
)

func TestViewBlocksParseLegacyText(t *testing.T) {
	blocks := ViewBlocks("## Studio note\n\n> Clay remembers the hand.\n\n[image: /media/cup.jpg | Cup]\n\n[gallery: /media/a.jpg | A; /media/b.jpg | B]\n\n[button: Shop now | /shop]\n\n[product: cloud-slip-bowl]\n\nPlain paragraph.")
	if len(blocks) != 7 {
		t.Fatalf("expected 7 blocks, got %#v", blocks)
	}
	if blocks[0]["isHeading"] != true || blocks[0]["text"] != "Studio note" {
		t.Fatalf("unexpected heading block: %#v", blocks[0])
	}
	if blocks[1]["isQuote"] != true {
		t.Fatalf("unexpected quote block: %#v", blocks[1])
	}
	if blocks[2]["isImage"] != true || blocks[2]["url"] != "/media/cup.jpg" || blocks[2]["alt"] != "Cup" {
		t.Fatalf("unexpected image block: %#v", blocks[2])
	}
	images, ok := blocks[3]["images"].([]map[string]any)
	if !ok || len(images) != 2 || images[1]["alt"] != "B" {
		t.Fatalf("unexpected gallery block: %#v", blocks[3])
	}
	if blocks[4]["isButton"] != true || blocks[4]["label"] != "Shop now" || blocks[4]["href"] != "/shop" {
		t.Fatalf("unexpected button block: %#v", blocks[4])
	}
	if blocks[5]["isProduct"] != true || blocks[5]["productRef"] != "cloud-slip-bowl" {
		t.Fatalf("unexpected product block: %#v", blocks[5])
	}
	if blocks[6]["isParagraph"] != true || blocks[6]["text"] != "Plain paragraph." {
		t.Fatalf("unexpected paragraph block: %#v", blocks[6])
	}
}

func TestParseLegacyV1Document(t *testing.T) {
	body := `{"version":1,"blocks":[{"id":"a","type":"heading","text":"Studio note"},{"id":"b","type":"paragraph","text":"Clay remembers the hand."},{"id":"c","type":"product","productRef":"cloud-slip-bowl"}]}`
	doc := Parse(body)
	if doc.Kind != BodyKind || len(doc.Blocks) != 3 {
		t.Fatalf("unexpected parsed document: %#v", doc)
	}
	blocks := ViewBlocksFromDocument(doc)
	if len(blocks) != 3 || blocks[0]["isHeading"] != true || blocks[2]["productRef"] != "cloud-slip-bowl" {
		t.Fatalf("unexpected parsed view blocks: %#v", blocks)
	}
}

func TestInvalidJSONFallsBackToText(t *testing.T) {
	blocks := ViewBlocks("{not json")
	if len(blocks) != 1 || blocks[0]["isParagraph"] != true || blocks[0]["text"] != "{not json" {
		t.Fatalf("expected malformed JSON to fall back to paragraph parsing, got %#v", blocks)
	}
}

func TestSerializeLegacyV1(t *testing.T) {
	doc := blockstudio.Document{Version: 1, Kind: BodyKind, Blocks: []blockstudio.BlockInstance{
		{ID: "heading-1", Key: BlockHeading, Enabled: true, Values: blockstudio.Values{
			"text": {Kind: blockstudio.FieldText, String: "Studio note"},
		}},
		{ID: "image-2", Key: BlockImage, Enabled: true, Values: blockstudio.Values{
			"url": {Kind: blockstudio.FieldImage, String: "/media/cup.jpg", Media: &blockstudio.MediaValue{URL: "/media/cup.jpg", Alt: "Cup"}},
		}},
		{ID: "hidden", Key: BlockParagraph, Enabled: false, Values: blockstudio.Values{
			"text": {Kind: blockstudio.FieldTextarea, String: "Hidden draft note."},
		}},
	}}
	body, err := SerializeLegacyV1(doc)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"type": "heading"`, `"text": "Studio note"`, `"url": "/media/cup.jpg"`, `"alt": "Cup"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in serialized body: %s", want, body)
		}
	}
}

func TestFlowBlocksHaveViewShape(t *testing.T) {
	blocks := ViewBlocksFromDocument(blockstudio.Document{Blocks: []blockstudio.BlockInstance{
		{Key: BlockFlow, Enabled: true, Values: blockstudio.Values{
			"flowKey": {Kind: blockstudio.FieldText, String: "purchase-request"},
		}},
	}})
	if len(blocks) != 1 || blocks[0]["isFlow"] != true || blocks[0]["flowKey"] != "purchase-request" {
		t.Fatalf("unexpected flow view block: %#v", blocks)
	}
}
