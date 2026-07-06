package blocks

import (
	"testing"

	"m31labs.dev/gosx-admin/blockstudio"
)

func TestBodyCatalogNormalizes(t *testing.T) {
	doc := blockstudio.NormalizeDocument(blockstudio.Document{}, BodyCatalog())
	if len(doc.Blocks) != 8 {
		t.Fatalf("expected eight body blocks, got %#v", doc.Blocks)
	}
	if doc.Blocks[0].Key != "paragraph" || !doc.Blocks[0].Enabled {
		t.Fatalf("expected default paragraph first and enabled, got %#v", doc.Blocks[0])
	}
	for _, block := range doc.Blocks[1:] {
		if block.Enabled {
			t.Fatalf("only paragraph should be default-on, got %#v", doc.Blocks)
		}
	}
	if got := doc.Blocks[1].Values["level"].String; got != "2" {
		t.Fatalf("expected heading default level to normalize, got %#v", doc.Blocks[1].Values)
	}
}

func TestBodyCatalogRepeatableDuplicatesSurvive(t *testing.T) {
	doc := blockstudio.NormalizeDocument(blockstudio.Document{Blocks: []blockstudio.BlockInstance{
		{Key: "paragraph", Enabled: true, Order: 2, Values: blockstudio.Values{
			"text": {Kind: blockstudio.FieldTextarea, String: "Second paragraph."},
		}},
		{Key: "paragraph", Enabled: true, Order: 1, Values: blockstudio.Values{
			"text": {Kind: blockstudio.FieldTextarea, String: "First paragraph."},
		}},
	}}, BodyCatalog())
	if len(doc.Blocks) != 9 {
		t.Fatalf("expected duplicate paragraph plus missing catalog blocks, got %#v", doc.Blocks)
	}
	if doc.Blocks[0].Key != "paragraph" || doc.Blocks[1].Key != "paragraph" {
		t.Fatalf("expected repeatable paragraphs to survive normalization, got %#v", doc.Blocks)
	}
	if doc.Blocks[0].ID == doc.Blocks[1].ID {
		t.Fatalf("expected duplicate paragraphs to receive distinct IDs, got %#v", doc.Blocks[:2])
	}
	if got := doc.Blocks[0].Values["text"].String; got != "First paragraph." {
		t.Fatalf("expected paragraph values to follow normalized order, got %#v", doc.Blocks[0].Values)
	}
}

func TestBodyCatalogFormFieldNames(t *testing.T) {
	doc := blockstudio.NormalizeDocument(blockstudio.Document{Blocks: []blockstudio.BlockInstance{
		{Key: "image", Enabled: true, Order: 1, Values: blockstudio.Values{
			"url": {Kind: blockstudio.FieldURL, String: "/media/work.jpg"},
			"alt": {Kind: blockstudio.FieldText, String: "Work"},
		}},
	}}, BodyCatalog())
	form := blockstudio.FormDocument(doc, BodyCatalog(), blockstudio.FormOptions{Prefix: "body"})
	if form.CountName != "bodyBlockCount" || len(form.Blocks) != 8 {
		t.Fatalf("unexpected body form document: %#v", form)
	}
	block := form.Blocks[0]
	if block.KeyName != "bodyBlockKey0" || block.EnabledName != "bodyBlockEnabled0" || block.OrderName != "bodyBlockOrder0" {
		t.Fatalf("unexpected body block field names: %#v", block)
	}
	if len(block.Fields) != 2 || block.Fields[0].InputName != "bodyBlockField0_url" || block.Fields[1].InputName != "bodyBlockField0_alt" {
		t.Fatalf("expected image field names with body prefix, got %#v", block.Fields)
	}
}

func TestBodyCatalogValidatesRequiredFields(t *testing.T) {
	errs := blockstudio.ValidateDocument(blockstudio.Document{Blocks: []blockstudio.BlockInstance{
		{Key: "paragraph", Enabled: true, Values: blockstudio.Values{
			"text": {Kind: blockstudio.FieldTextarea},
		}},
		{Key: "button", Enabled: true, Values: blockstudio.Values{
			"label": {Kind: blockstudio.FieldText, String: "Read more"},
			"href":  {Kind: blockstudio.FieldURL},
		}},
	}}, BodyCatalog())
	if errs["blocks[0].values.text"] == "" {
		t.Fatalf("expected paragraph text to be required, got %#v", errs)
	}
	if errs["blocks[1].values.href"] == "" {
		t.Fatalf("expected button href to be required, got %#v", errs)
	}
	if errs["blocks[1].values.label"] != "" {
		t.Fatalf("did not expect populated button label error, got %#v", errs)
	}
}
