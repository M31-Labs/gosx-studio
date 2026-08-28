package media

import (
	"errors"
	"testing"

	"m31labs.dev/gosx-studio/core"
)

func TestNormalizeListDedupesAndAddsFallbackAlt(t *testing.T) {
	items := NormalizeList([]Item{
		{URL: " /media/a.jpg "},
		{URL: "/media/a.jpg", Alt: "Duplicate"},
		{URL: "/media/b.jpg", Alt: "B"},
		{},
	}, "Work", true)
	if len(items) != 2 {
		t.Fatalf("expected two normalized images, got %#v", items)
	}
	if items[0].URL != "/media/a.jpg" || items[0].Alt != "Work" {
		t.Fatalf("unexpected first item: %#v", items[0])
	}
	if items[1].Alt != "B" {
		t.Fatalf("unexpected second item: %#v", items[1])
	}
}

func TestNormalizeListPlaceholder(t *testing.T) {
	items := NormalizeList(nil, "Cup", true)
	if len(items) != 1 || items[0].URL != "/media/placeholder.svg" || items[0].Alt != "Cup" {
		t.Fatalf("unexpected placeholder media: %#v", items)
	}
	if got := NormalizeList(nil, "Cup", false); len(got) != 0 {
		t.Fatalf("did not expect placeholder when disabled: %#v", got)
	}
}

func TestNormalizeAssetAndVariants(t *testing.T) {
	asset := NormalizeAsset(Input{
		URL:         " /media/cup.jpg ",
		ContentType: " image/jpeg ",
		Size:        128,
		Variants: Variants{
			" thumb ": {URL: " /media/cup-thumb.jpg ", ContentType: " image/jpeg ", Width: 400},
			"":        {URL: "/media/empty.jpg"},
			"bad":     {},
		},
	}, Asset{})
	if asset.URL != "/media/cup.jpg" || asset.Filename != "cup.jpg" || asset.ContentType != "image/jpeg" || asset.Size != 128 {
		t.Fatalf("unexpected asset: %#v", asset)
	}
	if len(asset.Variants) != 1 || asset.Variants["thumb"].URL != "/media/cup-thumb.jpg" {
		t.Fatalf("unexpected variants: %#v", asset.Variants)
	}
}

func TestParseAndFormatLines(t *testing.T) {
	items := ParseLines("/media/a.jpg | A\n\n/media/b.jpg")
	if len(items) != 2 || items[0].Alt != "A" || items[1].URL != "/media/b.jpg" {
		t.Fatalf("unexpected parsed media lines: %#v", items)
	}
	if got := FormatLines(items); got != "/media/a.jpg | A\n/media/b.jpg" {
		t.Fatalf("unexpected formatted lines: %q", got)
	}
}

func TestCloneAssetsDeepCopiesVariants(t *testing.T) {
	assets := []Asset{{URL: "/media/a.jpg", Variants: Variants{"thumb": {URL: "/media/thumb.jpg"}}}}
	cloned := CloneAssets(assets)
	cloned[0].Variants["thumb"] = Variant{URL: "/media/changed.jpg"}
	if assets[0].Variants["thumb"].URL != "/media/thumb.jpg" {
		t.Fatalf("expected clone to preserve original variants, got %#v", assets)
	}
}

func TestPickerField(t *testing.T) {
	field := Picker("image", "Image", Item{URL: "/media/a.jpg", Alt: "A"}, []Asset{{URL: "/media/a.jpg"}}, true)
	if field.Name != "image" || field.URL != "/media/a.jpg" || !field.Required || !field.HasAssets || field.AssetCount != 1 {
		t.Fatalf("unexpected picker field: %#v", field)
	}
}

func TestNormalizeFocalPoint(t *testing.T) {
	point := NormalizeFocalPoint(0.25, 1.5)
	if point.X != 0.25 || point.Y != 0 {
		t.Fatalf("unexpected focal point: %#v", point)
	}
}

func TestUploadPolicyAllowsContentAndExtensionTypes(t *testing.T) {
	policy := DefaultUploadPolicy()
	if ext, ok := policy.AllowedExtension("image/jpeg", "forest.jpeg"); !ok || ext != ".jpg" {
		t.Fatalf("expected jpeg content type to be allowed, got %q %v", ext, ok)
	}
	if ext, ok := policy.AllowedExtension("application/octet-stream", "font.woff2"); !ok || ext != ".woff2" {
		t.Fatalf("expected font extension to be allowed, got %q %v", ext, ok)
	}
	if policy.Allows("image/jpeg", "forest.jpg", policy.MaxBytes+1) {
		t.Fatal("expected oversized upload to be rejected")
	}
}

func TestInputFromStoredObject(t *testing.T) {
	input := InputFromStoredObject(StoredObject{
		URL:         " /media/uploads/forest.jpg ",
		Filename:    " forest.jpg ",
		ContentType: " image/jpeg ",
		Size:        120,
	}, " Forest ", Variants{"thumb": {URL: " /media/uploads/forest-thumb.jpg "}})
	if input.URL != "/media/uploads/forest.jpg" || input.Alt != "Forest" || input.Filename != "forest.jpg" || input.ContentType != "image/jpeg" || input.Size != 120 {
		t.Fatalf("unexpected input: %#v", input)
	}
	if input.Variants["thumb"].URL != "/media/uploads/forest-thumb.jpg" {
		t.Fatalf("expected normalized variants, got %#v", input.Variants)
	}
}

func TestNormalizeAssetKeepsCaption(t *testing.T) {
	asset := NormalizeAsset(Input{URL: "/media/cup.jpg", Caption: "  A cup on a shelf.  "}, Asset{})
	if asset.Caption != "A cup on a shelf." {
		t.Fatalf("expected trimmed caption, got %q", asset.Caption)
	}
}

// --- GuardDeleteMediaAsset ---

type stubUsageLookup struct {
	usage map[string][]Usage
}

func (lookup stubUsageLookup) MediaUsage(id string) []Usage { return lookup.usage[id] }

type stubAssetDeleter struct {
	deleted []string
	err     error
}

func (deleter *stubAssetDeleter) DeleteMediaAsset(id string) error {
	if deleter.err != nil {
		return deleter.err
	}
	deleter.deleted = append(deleter.deleted, id)
	return nil
}

func TestGuardDeleteMediaAssetAllowsUnreferencedAsset(t *testing.T) {
	lookup := stubUsageLookup{usage: map[string][]Usage{}}
	deleter := &stubAssetDeleter{}
	usage, err := GuardDeleteMediaAsset(lookup, deleter, "med-1")
	if err != nil {
		t.Fatalf("unexpected error deleting an unreferenced asset: %v", err)
	}
	if len(usage) != 0 {
		t.Fatalf("expected no usage for an unreferenced asset, got %#v", usage)
	}
	if len(deleter.deleted) != 1 || deleter.deleted[0] != "med-1" {
		t.Fatalf("expected the deleter to be called with the asset id, got %#v", deleter.deleted)
	}
}

func TestGuardDeleteMediaAssetRejectsReferencedAsset(t *testing.T) {
	lookup := stubUsageLookup{usage: map[string][]Usage{
		"med-1": {{Kind: "Product", Title: "Hand-thrown mug", ID: "prod-1"}},
	}}
	deleter := &stubAssetDeleter{}
	usage, err := GuardDeleteMediaAsset(lookup, deleter, "med-1")
	if !errors.Is(err, ErrAssetReferenced) {
		t.Fatalf("expected ErrAssetReferenced, got %v", err)
	}
	if len(usage) != 1 || usage[0].Title != "Hand-thrown mug" {
		t.Fatalf("expected the blocking usage to be returned, got %#v", usage)
	}
	if len(deleter.deleted) != 0 {
		t.Fatal("expected the deleter to never be called for a referenced asset")
	}
}

func TestGuardDeleteMediaAssetPropagatesDeleterError(t *testing.T) {
	lookup := stubUsageLookup{usage: map[string][]Usage{}}
	deleter := &stubAssetDeleter{err: errors.New("store unavailable")}
	if _, err := GuardDeleteMediaAsset(lookup, deleter, "med-1"); err == nil || err.Error() != "store unavailable" {
		t.Fatalf("expected the deleter's error to propagate, got %v", err)
	}
}

func TestGuardDeleteMediaAssetRequiresID(t *testing.T) {
	deleter := &stubAssetDeleter{}
	if _, err := GuardDeleteMediaAsset(stubUsageLookup{}, deleter, "  "); err == nil {
		t.Fatal("expected a blank id to be rejected")
	}
	if len(deleter.deleted) != 0 {
		t.Fatal("expected the deleter to never be called for a blank id")
	}
}

// --- AssetReadiness ---

func TestAssetReadinessRequiresAltOnlyWhenReferenced(t *testing.T) {
	unused := Asset{ID: "med-1"}
	if status := AssetReadinessStatus(unused, nil); status != core.ReadinessWatch {
		t.Fatalf("expected an unused asset with no alt text to be Watch, got %v", status)
	}
	referenced := Asset{ID: "med-2"}
	usage := []Usage{{Kind: "Product", Title: "Mug"}}
	if status := AssetReadinessStatus(referenced, usage); status != core.ReadinessBlocked {
		t.Fatalf("expected a referenced asset with no alt text to be Blocked, got %v", status)
	}
	withAlt := Asset{ID: "med-3", Alt: "A hand-thrown mug"}
	if status := AssetReadinessStatus(withAlt, usage); status != core.ReadinessReady {
		t.Fatalf("expected a referenced asset with alt text to be Ready, got %v", status)
	}
}
