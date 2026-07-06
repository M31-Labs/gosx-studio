package blocks

import (
	"testing"

	"m31labs.dev/gosx-admin/blockstudio"
)

func TestHomeCatalogNormalizes(t *testing.T) {
	blocks := blockstudio.Normalize(nil, HomeCatalog())
	if len(blocks) != 4 {
		t.Fatalf("expected four home blocks, got %#v", blocks)
	}
	for _, block := range blocks {
		if !block.Enabled {
			t.Fatalf("default home block should be enabled: %#v", block)
		}
	}
	if !blockstudio.KeyAllowed("hero", HomeCatalog()) {
		t.Fatal("expected hero block to be allowed")
	}
}

func TestHomeCatalogSupportsDocuments(t *testing.T) {
	doc := blockstudio.NormalizeDocument(blockstudio.Document{Blocks: []blockstudio.BlockInstance{
		{Key: "blog", Enabled: false, Order: 1},
		{Key: "hero", Enabled: true, Order: 2, Values: blockstudio.Values{
			"headline": {Kind: blockstudio.FieldText, String: "Studio work"},
		}},
	}}, HomeCatalog())
	if len(doc.Blocks) != 4 {
		t.Fatalf("expected document to include all home blocks, got %#v", doc.Blocks)
	}
	if doc.Blocks[0].Key != "blog" || doc.Blocks[1].Key != "hero" {
		t.Fatalf("expected document order to follow saved order, got %#v", doc.Blocks)
	}
	if got := doc.Blocks[1].Values["headline"].String; got != "Studio work" {
		t.Fatalf("expected hero values to survive normalization, got %#v", doc.Blocks[1].Values)
	}

	form := blockstudio.FormDocument(doc, HomeCatalog(), blockstudio.FormOptions{Prefix: "home"})
	if form.CountName != "homeBlockCount" || len(form.Blocks) != 4 {
		t.Fatalf("unexpected home form document: %#v", form)
	}
	if form.Blocks[1].Fields[0].InputName != "homeBlockField1_headline" {
		t.Fatalf("expected hero field names from document form, got %#v", form.Blocks[1].Fields)
	}
}
