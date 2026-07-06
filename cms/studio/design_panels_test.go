package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestRenderBrandPanelKeepsLogoHooks(t *testing.T) {
	html := gosx.RenderHTML(RenderBrandPanel(BrandPanelOptions{
		LogoURL:     "/logo.png",
		LogoAlt:     "Studio logo",
		SnapChecked: true,
		SnapSize:    8,
		MediaHref:   "/admin/media",
		Fields: []InspectorField{
			{Kind: InspectorFieldInput, ID: "logoUrl", Name: "logoUrl", Label: "Logo URL", Value: "/logo.png", Wide: true, Attrs: []FieldAttribute{{Name: "data-editor-logo-url", Value: "true"}}},
		},
		LayoutFields: []InspectorField{
			{Kind: InspectorFieldInput, ID: "logoWidth", Name: "logoWidth", Label: "Logo width", Type: "number", Value: "96", Attrs: []FieldAttribute{{Name: "data-editor-logo-width", Value: "true"}}},
		},
	}))
	for _, check := range []string{
		`data-panel-key="brand"`,
		`data-studio-mode-panel="brand"`,
		`data-editor-logo-url="true"`,
		`data-editor-logo-width="true"`,
		`data-editor-logo-snap="true" checked`,
		`data-editor-logo-snap-size="true"`,
		`data-editor-brand-preview="true"`,
		`data-editor-brand-handle="true"`,
		`src="/logo.png" alt="Studio logo" data-editor-brand-logo="true"`,
		`href="/admin/media" data-gosx-link="true"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in brand panel: %s", check, html)
		}
	}
}

func TestRenderStyleSettingsPanelKeepsThemeHooks(t *testing.T) {
	html := gosx.RenderHTML(RenderStyleSettingsPanel(StyleSettingsPanelOptions{
		WorkbenchHTML: `<div data-studio-style-workbench="true"></div>`,
		KitGroup: ChoiceGroup{
			Class:     "template-picker theme-kit-picker",
			GridClass: "template-picker__grid theme-kit-picker__grid",
			Legend:    "Starter kit",
			Cards: []ChoiceCard{{
				Name:    "themeKit",
				Value:   "studio",
				Label:   "Studio",
				Summary: "Default kit",
				Class:   "template-card theme-kit-card",
				Checked: true,
				CardAttrs: []FieldAttribute{
					{Name: "data-editor-kit-card", Value: "studio"},
					{Name: "data-kit-template", Value: "studio"},
					{Name: "data-kit-style-nav", Value: "links"},
				},
				InputAttrs: []FieldAttribute{{Name: "data-editor-kit-input", Value: "true"}},
			}},
		},
		TemplateGroup: ChoiceGroup{
			Legend: "Template",
			Cards: []ChoiceCard{{
				Name:      "themeTemplate",
				Value:     "custom",
				Label:     "Custom",
				Summary:   "Custom layout",
				CardAttrs: []FieldAttribute{{Name: "data-editor-template-card", Value: "custom"}},
			}},
		},
		CustomBuilderClass: "custom-template-builder is-active",
		CustomNameField: InspectorField{
			Kind:  InspectorFieldInput,
			ID:    "customTemplateName",
			Name:  "customTemplateName",
			Label: "Custom name",
			Value: "Custom",
			Wide:  true,
			Attrs: []FieldAttribute{{Name: "data-editor-custom-template-name", Value: "true"}},
		},
		CustomControls: []SelectControl{{
			ID: "customHeroLayout", Name: "customHeroLayout", Label: "Hero layout", Attrs: []FieldAttribute{{Name: "data-editor-custom-class", Value: "hero"}},
			Options: []StudioOption{{Value: "overlay", Label: "Overlay", Selected: true}},
		}},
		Palette: SelectControl{
			ID: "themePalette", Name: "themePalette", Label: "Theme set", Attrs: []FieldAttribute{{Name: "data-editor-theme-preset", Value: "true"}},
			Options: []StudioOption{{Value: "clay", Label: "Clay", Selected: true, Attrs: []FieldAttribute{{Name: "data-color-canvas", Value: "#fff"}}}},
		},
		Swatches: []ColorTokenControl{{Label: "Canvas", Value: "#ffffff"}},
		StyleControls: []SelectControl{{
			ID: "styleNav", Name: "styleNav", Label: "Navigation", Attrs: []FieldAttribute{{Name: "data-editor-style-class", Value: "nav"}},
			Options: []StudioOption{{Value: "links", Label: "Links", Selected: true}},
		}},
		ImageCrop: RadioControlGroup{
			Legend: "Image crop",
			Name:   "themeImageRatio",
			Options: []StudioOption{
				{Value: "landscape", Label: "Landscape", Selected: true},
			},
		},
		ColorTokens: []ColorTokenControl{{Key: "canvas", Name: "colorCanvas", Label: "Canvas", CSSVar: "--color-canvas", Value: "#ffffff"}},
	}))
	for _, check := range []string{
		`data-studio-style-workbench="true"`,
		`data-editor-kit-card="studio"`,
		`data-kit-template="studio"`,
		`data-editor-kit-input="true"`,
		`data-editor-template-card="custom"`,
		`data-custom-template-builder="true"`,
		`data-editor-custom-class="hero"`,
		`data-editor-theme-preset="true"`,
		`data-color-canvas="#fff"`,
		`data-editor-theme-swatches="true"`,
		`data-editor-style-class="nav"`,
		`name="themeImageRatio" value="landscape" checked`,
		`data-editor-color-token="--color-canvas"`,
		`data-editor-color-key="canvas"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in style settings panel: %s", check, html)
		}
	}
}
