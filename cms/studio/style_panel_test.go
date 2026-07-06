package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	cmsstyle "m31labs.dev/gosx-studio/cms/style"
)

func TestRenderStyleScope(t *testing.T) {
	html := gosx.RenderHTML(RenderStyleScope(StyleScopeOptions{
		SystemID:        "forest-school",
		PageLabel:       "Programs",
		BlockLabel:      "Hero",
		FieldLabel:      "headline",
		BreakpointLabel: "Mobile",
		States:          []StyleState{{Key: "default", Label: "Default"}, {Key: "focus", Label: "Focus", Active: true}},
	}))
	if !strings.Contains(html, `data-studio-style-scope="true"`) || !strings.Contains(html, `forest-school`) || !strings.Contains(html, `data-studio-style-state="focus"`) || !strings.Contains(html, `aria-pressed="true"`) {
		t.Fatalf("unexpected style scope markup: %s", html)
	}
}

func TestRenderStyleWorkbenchUsesRecipeViewsAndBindings(t *testing.T) {
	recipes := []cmsstyle.RecipeView{{
		Key:   "theme",
		Label: "Theme",
		Controls: []cmsstyle.ControlView{
			{
				Key:   "hero-layout",
				Label: "Hero",
				Options: []cmsstyle.OptionView{
					{Value: "overlay", Label: "Overlay"},
					{Value: "split", Label: "Split"},
				},
				Default: "overlay",
			},
			{
				Key:   "spacing",
				Label: "Sections",
				Options: []cmsstyle.OptionView{
					{Value: "compact", Label: "Compact"},
					{Value: "airy", Label: "Airy"},
				},
				Default: "compact",
			},
		},
	}}
	html := gosx.RenderHTML(RenderStyleWorkbench(recipes, StyleWorkbenchOptions{
		Groups: []StyleRecipeGroup{{
			Key:               "layout",
			Label:             "Page shape",
			VisualClass:       "studio-style-visual--layout",
			VisualMarks:       2,
			ReadoutControlKey: "hero-layout",
			Controls: []StyleControlBinding{
				{ControlKey: "hero-layout", FieldName: "customHeroLayout", Label: "Hero"},
				{ControlKey: "spacing", FieldName: "styleSpacing"},
			},
		}},
		Values: map[string]string{
			"customHeroLayout": "split",
			"styleSpacing":     "airy",
		},
	}))
	if !strings.Contains(html, `data-studio-style-workbench="true"`) || !strings.Contains(html, `data-studio-style-recipe="layout"`) || !strings.Contains(html, `data-studio-style-readout="customHeroLayout"`) {
		t.Fatalf("unexpected workbench shell markup: %s", html)
	}
	if !strings.Contains(html, `data-studio-style-control="customHeroLayout"`) || !strings.Contains(html, `data-studio-style-value="split"`) || !strings.Contains(html, `data-studio-style-reset="styleSpacing"`) {
		t.Fatalf("expected bound style control hooks, got: %s", html)
	}
	if !strings.Contains(html, `>Split</output>`) {
		t.Fatalf("expected readout to use current value label, got: %s", html)
	}
}

func TestStyleSelectControlsMapRecipeBindings(t *testing.T) {
	recipes := []cmsstyle.RecipeView{{
		Key:   "theme",
		Label: "Theme",
		Controls: []cmsstyle.ControlView{
			{
				Key:     "hero-layout",
				Label:   "Hero layout",
				Default: "overlay",
				Options: []cmsstyle.OptionView{
					{Value: "overlay", Label: "Overlay", CSS: "overlay"},
					{Value: "split", Label: "Split", CSS: "split"},
				},
			},
			{
				Key:     "content-width",
				Label:   "Page width",
				Default: "standard",
				Options: []cmsstyle.OptionView{
					{Value: "standard", Label: "Standard", CSS: "standard"},
					{Value: "wide", Label: "Wide", CSS: "wide"},
				},
			},
		},
	}}
	controls := StyleSelectControls(recipes, []StyleControlBinding{
		{
			ControlKey: "hero-layout",
			FieldName:  "customHeroLayout",
			Label:      "Hero",
			Attrs:      []FieldAttribute{{Name: "data-editor-custom-class", Value: "hero"}},
		},
		{
			ControlKey: "content-width",
			FieldName:  "customContentWidth",
			Label:      "Width",
			Wide:       true,
		},
	}, map[string]string{"customHeroLayout": "split"}, StyleSelectControlOptions{SourcePrefix: "style."})
	html := gosx.RenderHTML(gosx.Fragment(
		renderSelectControl(controls[0]),
		renderSelectControl(controls[1]),
	))
	for _, check := range []string{
		`id="customHeroLayout"`,
		`name="customHeroLayout"`,
		`data-editor-custom-class="hero"`,
		`data-studio-field-source="style.customHeroLayout"`,
		`data-studio-style-css="split"`,
		`value="split" selected`,
		`class="field-row field-row--wide"`,
		`id="customContentWidth"`,
		`value="standard" selected`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in generated style controls: %s", check, html)
		}
	}
}

func TestStyleRadioControlGroupMapsRecipeBinding(t *testing.T) {
	recipes := []cmsstyle.RecipeView{{
		Key: "theme",
		Controls: []cmsstyle.ControlView{{
			Key:     "image-ratio",
			Label:   "Image crop",
			Default: "landscape",
			Options: []cmsstyle.OptionView{
				{Value: "landscape", Label: "Landscape"},
				{Value: "square", Label: "Square"},
			},
		}},
	}}
	group := StyleRadioControlGroup(recipes, StyleControlBinding{
		ControlKey: "image-ratio",
		FieldName:  "themeImageRatio",
	}, map[string]string{"themeImageRatio": "square"}, StyleRadioControlOptions{
		SourcePrefix: "style.",
	})
	html := gosx.RenderHTML(renderRadioControlGroup(group))
	for _, check := range []string{
		`class="radio-row"`,
		`data-studio-field-source="style.themeImageRatio"`,
		`<legend>Image crop</legend>`,
		`name="themeImageRatio" value="square" checked`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in generated radio group: %s", check, html)
		}
	}
}
