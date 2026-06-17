package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

// TestRenderInspectorControlKindDispatch verifies that RenderInspectorControl
// selects the right widget per ControlKind and preserves invariants that
// the panel test also checks.

func TestRenderInspectorControlText(t *testing.T) {
	control := map[string]any{
		"kind":        string(ControlText),
		"value":       "Forest mornings",
		"actionLabel": "Save field",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if strings.Contains(html, "<textarea") {
		t.Fatalf("text kind must render <input>, got textarea:\n%s", html)
	}
	if !strings.Contains(html, `type="text"`) {
		t.Fatalf("text kind must render <input type=\"text\">:\n%s", html)
	}
	if !strings.Contains(html, `value="Forest mornings"`) {
		t.Fatalf("text kind must include value:\n%s", html)
	}
}

func TestRenderInspectorControlRichText(t *testing.T) {
	control := map[string]any{
		"kind":        string(ControlRichText),
		"value":       "A long paragraph.",
		"actionLabel": "Save content",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, "<textarea") {
		t.Fatalf("rich-text kind must render <textarea>:\n%s", html)
	}
	if strings.Contains(html, `type="text"`) {
		t.Fatalf("rich-text kind must not render input type=text:\n%s", html)
	}
	if !strings.Contains(html, "A long paragraph.") {
		t.Fatalf("rich-text textarea must contain value as text content:\n%s", html)
	}
	if !strings.Contains(html, "Save content") {
		t.Fatalf("rich-text must render custom actionLabel:\n%s", html)
	}
}

func TestRenderInspectorControlNumber(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlNumber),
		"value": "42",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `type="number"`) {
		t.Fatalf("number kind must render <input type=\"number\">:\n%s", html)
	}
}

func TestRenderInspectorControlLink(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlLink),
		"value": "https://example.com",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `type="url"`) {
		t.Fatalf("link kind must render <input type=\"url\">:\n%s", html)
	}
}

func TestRenderInspectorControlColor(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlColor),
		"value": "#ff0000",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `type="color"`) {
		t.Fatalf("color kind must render <input type=\"color\">:\n%s", html)
	}
	// Must also have a text mirror
	if !strings.Contains(html, `type="text"`) {
		t.Fatalf("color kind must render a sibling text mirror input:\n%s", html)
	}
}

func TestRenderInspectorControlToggle(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlToggle),
		"value": "true",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, "<select") {
		t.Fatalf("toggle kind must render a <select>:\n%s", html)
	}
}

func TestRenderInspectorControlChoiceWithOptions(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlChoice),
		"value": "b",
		"options": []map[string]string{
			{"value": "a", "label": "Option A"},
			{"value": "b", "label": "Option B"},
		},
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, "<select") {
		t.Fatalf("choice kind with options must render a <select>:\n%s", html)
	}
	if !strings.Contains(html, "Option A") || !strings.Contains(html, "Option B") {
		t.Fatalf("choice <select> must include option labels:\n%s", html)
	}
}

func TestRenderInspectorControlChoiceFallsBackToText(t *testing.T) {
	// No options → fallback to text input
	control := map[string]any{
		"kind":  string(ControlChoice),
		"value": "manual",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `type="text"`) {
		t.Fatalf("choice without options must fall back to text input:\n%s", html)
	}
}

func TestRenderInspectorControlMediaSlot(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlMedia),
		"value": "image.jpg",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `data-gosx-studio-inspector-slot="media"`) {
		t.Fatalf("media kind must render data-gosx-studio-inspector-slot=media:\n%s", html)
	}
	if !strings.Contains(html, "image.jpg") {
		t.Fatalf("media slot must show current value:\n%s", html)
	}
	// The Choose media button must be disabled
	if !strings.Contains(html, "disabled") {
		t.Fatalf("media slot Choose button must be disabled:\n%s", html)
	}
}

func TestRenderInspectorControlUnknownFallsBackToText(t *testing.T) {
	control := map[string]any{
		"kind":  "scene-3d",
		"value": "scene.glb",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `type="text"`) {
		t.Fatalf("unknown kind must fall back to text input:\n%s", html)
	}
	if !strings.Contains(html, `data-gosx-studio-inspector-slot="scene-3d"`) {
		t.Fatalf("unknown kind must carry inspector-slot data-attr:\n%s", html)
	}
}

func TestRenderInspectorControlPreservesHiddenInputs(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlText),
		"value": "hello",
		"formInputs": []map[string]string{
			{"name": "authoring_operation", "value": "update_control"},
			{"name": "page_key", "value": "home"},
		},
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `name="authoring_operation"`) {
		t.Fatalf("must include hidden formInputs:\n%s", html)
	}
	if !strings.Contains(html, `value="update_control"`) {
		t.Fatalf("must include hidden formInputs values:\n%s", html)
	}
}

func TestRenderInspectorControlWrapperDataAttrs(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlRichText),
		"value": "paragraph",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `data-gosx-studio-inspector-control`) {
		t.Fatalf("wrapper must carry data-gosx-studio-inspector-control:\n%s", html)
	}
	if !strings.Contains(html, `data-gosx-studio-inspector-kind="rich-text"`) {
		t.Fatalf("wrapper must carry data-gosx-studio-inspector-kind:\n%s", html)
	}
}

func TestRenderInspectorControlIncludesSaveAndCancelButtons(t *testing.T) {
	control := map[string]any{
		"kind":        string(ControlText),
		"value":       "hello",
		"actionLabel": "Save field",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, `type="submit"`) {
		t.Fatalf("inspector control must include a submit button:\n%s", html)
	}
	if !strings.Contains(html, "Save field") {
		t.Fatalf("inspector control must render actionLabel on Save:\n%s", html)
	}
	// Cancel button: either type=reset or data-gosx-studio-inspector-cancel
	if !strings.Contains(html, `type="reset"`) && !strings.Contains(html, `data-gosx-studio-inspector-cancel`) {
		t.Fatalf("inspector control must include a Cancel button:\n%s", html)
	}
}

func TestRenderInspectorControlDefaultActionLabel(t *testing.T) {
	control := map[string]any{
		"kind":  string(ControlText),
		"value": "hello",
	}
	html := gosx.RenderHTML(RenderInspectorControl(control, "testForm", "gosx_studio_value"))
	if !strings.Contains(html, "Save field") {
		t.Fatalf("inspector control must default actionLabel to 'Save field':\n%s", html)
	}
}

// TestRenderSiteMapEditableControlPanelDelegatesKind checks that the panel
// now dispatches to the kind-aware widget (rich-text → textarea).
func TestRenderSiteMapEditableControlPanelDelegatesKind(t *testing.T) {
	view := AuthoringSiteMapView(NoCodeAuthoringSurface(SiteMap{
		Pages: []Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Editable:      true,
			Selected:      true,
			Components: []Component{{
				Key:           "hero",
				Label:         "Hero",
				GoSXComponent: "HomeHero",
				Source:        ComponentSourceHost,
				Binding:       "home.section.hero",
				Editable:      true,
				Controls: []Control{{
					Key:     "bio",
					Label:   "Bio",
					Kind:    ControlRichText,
					Binding: "pages.home.hero.bio",
					Value:   "A long paragraph.",
				}},
			}},
		}},
	}), SiteMapViewOptions{})

	html := gosx.RenderHTML(RenderSiteMapAuthoringPanels(view, SiteMapAuthoringPanelsOptions{}))

	// Data-attribute contract must be preserved
	if !strings.Contains(html, `data-gosx-studio-editable-control="true"`) {
		t.Fatalf("panel must still carry data-gosx-studio-editable-control:\n%s", html)
	}
	// rich-text → textarea
	if !strings.Contains(html, "<textarea") {
		t.Fatalf("rich-text control must render <textarea> via RenderInspectorControl:\n%s", html)
	}
	// Cancel button present
	if !strings.Contains(html, `type="reset"`) && !strings.Contains(html, `data-gosx-studio-inspector-cancel`) {
		t.Fatalf("panel must include Cancel button:\n%s", html)
	}
	// No platform copy
	if strings.Contains(html, "GoSX Studio") {
		t.Fatalf("panel must not inject platform copy:\n%s", html)
	}
}

// TestEditableControlViewKindPresent verifies the projection exposes "kind".
func TestEditableControlViewKindPresent(t *testing.T) {
	view := AuthoringSiteMapView(NoCodeAuthoringSurface(SiteMap{
		Pages: []Page{{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Editable:      true,
			Selected:      true,
			Components: []Component{{
				Key:           "hero",
				Label:         "Hero",
				GoSXComponent: "HomeHero",
				Source:        ComponentSourceHost,
				Editable:      true,
				Controls: []Control{{
					Key:     "headline",
					Label:   "Headline",
					Kind:    ControlRichText,
					Binding: "pages.home.hero.headline",
					Value:   "Forest mornings",
				}},
			}},
		}},
	}), SiteMapViewOptions{})

	ctrl := workbenchViewMap(view, "editableControl")
	if ctrl == nil {
		t.Fatal("editableControl must be present in the view")
	}
	kind, ok := ctrl["kind"].(string)
	if !ok || kind == "" {
		t.Fatalf("editableControl view must carry 'kind' as a non-empty string, got: %v", ctrl["kind"])
	}
	if kind != string(ControlRichText) {
		t.Fatalf("editableControl 'kind' must be %q, got %q", string(ControlRichText), kind)
	}
}
