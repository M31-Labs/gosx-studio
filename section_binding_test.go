package studio

import "testing"

func TestParseSectionFieldBinding(t *testing.T) {
	cases := []struct {
		binding string
		key     string
		field   string
		ok      bool
	}{
		{"home.section.pagehead.eyebrow", "pagehead", "eyebrow", true},
		{"section.collection.category", "collection", "category", true},
		{"home.hero.headline", "", "", false},
		{"tagline", "", "", false},
		{"home.section.pagehead", "", "", false},
	}
	for _, tc := range cases {
		key, field, ok := ParseSectionFieldBinding(tc.binding)
		if ok != tc.ok || key != tc.key || field != tc.field {
			t.Fatalf("ParseSectionFieldBinding(%q) = (%q,%q,%v), want (%q,%q,%v)", tc.binding, key, field, ok, tc.key, tc.field, tc.ok)
		}
	}
	// Round-trip through the inverse builder.
	key, field, ok := ParseSectionFieldBinding(SectionFieldBinding("postlist", "tag"))
	if !ok || key != "postlist" || field != "tag" {
		t.Fatalf("SectionFieldBinding round-trip failed: (%q,%q,%v)", key, field, ok)
	}
}

func TestStyleFormIDPart(t *testing.T) {
	if got := StyleFormIDPart("hero copy_2!"); got != "hero-copy-2-" {
		t.Fatalf("StyleFormIDPart sanitization wrong: %q", got)
	}
}
