package sitehost

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplatesCoverEveryKind(t *testing.T) {
	seen := map[string]bool{}
	perKind := map[string]int{}
	for _, template := range Templates() {
		if seen[template.Key] {
			t.Fatalf("duplicate template key %q", template.Key)
		}
		seen[template.Key] = true
		perKind[template.Kind]++
		if PaletteByKey(template.Palette).Key != template.Palette {
			t.Fatalf("%s: unknown palette %q", template.Key, template.Palette)
		}
		if FontPairByKey(template.Fonts).Key != template.Fonts {
			t.Fatalf("%s: unknown fonts %q", template.Key, template.Fonts)
		}
		if ButtonShapeByKey(template.Buttons).Key != template.Buttons || SpacingScaleByKey(template.Spacing).Key != template.Spacing {
			t.Fatalf("%s: unknown shape or spacing", template.Key)
		}
		if template.Band != "" && normalizeSectionStyle(template.Band) != template.Band {
			t.Fatalf("%s: unknown band %q", template.Key, template.Band)
		}
	}
	if len(seen) < 12 {
		t.Fatalf("only %d templates, the gallery needs at least 12", len(seen))
	}
	for _, kind := range SiteKinds() {
		if perKind[kind.Key] < 3 {
			t.Fatalf("kind %q has %d templates, want 3", kind.Key, perKind[kind.Key])
		}
	}
	if templateFor(SetupAnswers{Kind: "food", Template: "rally"}).Key != "bakery" {
		t.Fatal("a template from another kind falls back to the kind's default")
	}
}

func TestTemplateBandShapesTheHomePage(t *testing.T) {
	host, err := Open(Options{DataPath: filepath.Join(t.TempDir(), "site.json"), SiteTitle: "Corner", SiteKind: "food", Seed: true})
	if err != nil {
		t.Fatal(err)
	}
	// Seeding takes the kind's default: Bakery, with a tinted band.
	if host.settings().Metadata["siteTemplate"] != "bakery" {
		t.Fatalf("seed template = %q", host.settings().Metadata["siteTemplate"])
	}
	home := get(t, host.Handler(), "/").Body.String()
	mustContain(t, home, `class="site-section site-section--tinted"`, "the home page has the template's band")
	mustContain(t, home, "--site-radius:8px;", "and the template's rounded corners")
	if strings.Index(home, "site-section--plain") > strings.Index(home, "site-section--tinted") {
		t.Fatal("the opening line stays on the plain section, the band comes after")
	}

	var plain SetupAnswers
	plain.Kind = "simple"
	plain.Template = "plain"
	pages := StarterSiteFor(plain)
	for _, instance := range pages[0].Body.Blocks {
		if instance.Key == blockSection {
			t.Fatal("a template without a band adds no section")
		}
	}
}
