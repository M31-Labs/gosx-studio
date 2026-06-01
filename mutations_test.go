package studio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"m31labs.dev/gosx/action"
)

type stubAuthoringAdapter struct {
	calls    int
	mutation AuthoringMutation
	result   AuthoringMutationResult
	err      error
}

func (adapter *stubAuthoringAdapter) ApplyAuthoringMutation(ctx context.Context, mutation AuthoringMutation) (AuthoringMutationResult, error) {
	adapter.calls++
	adapter.mutation = mutation
	return adapter.result, adapter.err
}

func TestAuthoringMutationFromIntentBuildsActionPayload(t *testing.T) {
	intent := CompositionIntent{
		Key:                  "add-component:home:gallery",
		Label:                "Add gallery",
		Kind:                 CompositionIntentAddComponent,
		TargetPageKey:        "home",
		TargetPageLabel:      "Home",
		TargetRoute:          "/",
		TargetRegion:         "main",
		ComponentTemplateKey: "gallery",
		ComponentLabel:       "Gallery",
		Binding:              "media.gallery",
	}

	mutation := AuthoringMutationFromIntent(intent)
	values := mutation.FormValues()

	if mutation.Kind != AuthoringOperationApplyIntent || mutation.IntentKind != CompositionIntentAddComponent {
		t.Fatalf("unexpected mutation kind: %#v", mutation)
	}
	if values[AuthoringFieldOperation] != "apply-intent" || values[AuthoringFieldIntentKey] != "add-component:home:gallery" {
		t.Fatalf("unexpected form values: %#v", values)
	}
	if values[AuthoringFieldPageKey] != "home" || values[AuthoringFieldComponentTemplateKey] != "gallery" || values[AuthoringFieldBinding] != "media.gallery" {
		t.Fatalf("unexpected target values: %#v", values)
	}
	view := AuthoringMutationView(mutation)
	if view["kind"] != "apply-intent" || view["formValues"].(map[string]string)[AuthoringFieldTargetRegion] != "main" {
		t.Fatalf("unexpected mutation view: %#v", view)
	}
}

func TestAuthoringMutationForPageBuildsUpdatePayload(t *testing.T) {
	mutation := AuthoringMutationForPage(Page{
		Key:      "page_philosophy",
		Label:    "Our Philosophy",
		Route:    "/pages/philosophy",
		Editable: true,
	})
	values := mutation.FormValues()

	if mutation.Kind != AuthoringOperationUpdatePage || mutation.PageKey != "page_philosophy" {
		t.Fatalf("unexpected page mutation: %#v", mutation)
	}
	if values[AuthoringFieldOperation] != "update-page" || values[AuthoringFieldPageLabel] != "Our Philosophy" || values[AuthoringFieldPageRoute] != "/pages/philosophy" {
		t.Fatalf("unexpected page form values: %#v", values)
	}
}

func TestAuthoringMutationForComponentVisibilityBuildsTogglePayload(t *testing.T) {
	mutation := AuthoringMutationForComponentVisibility(Page{
		Key:   "home",
		Label: "Home",
		Route: "/",
	}, Component{
		Key:     "contact",
		Label:   "Contact band",
		Binding: "home.section.contact",
	}, true)
	values := mutation.FormValues()

	if mutation.Kind != AuthoringOperationToggleVisibility || mutation.PageKey != "home" || mutation.ComponentKey != "contact" {
		t.Fatalf("unexpected visibility mutation: %#v", mutation)
	}
	if !mutation.HasVisible || !mutation.Visible {
		t.Fatalf("expected explicit visible target: %#v", mutation)
	}
	if values[AuthoringFieldOperation] != "toggle-visibility" || values[AuthoringFieldVisible] != "true" || values[AuthoringFieldBinding] != "home.section.contact" {
		t.Fatalf("unexpected visibility form values: %#v", values)
	}
}

func TestAuthoringMutationForComponentReorderBuildsPositionPayload(t *testing.T) {
	mutation := AuthoringMutationForComponentReorder(Page{
		Key:   "home",
		Label: "Home",
		Route: "/",
	}, Component{
		Key:     "gallery",
		Label:   "Gallery",
		Binding: "home.section.gallery",
	}, 1)
	values := mutation.FormValues()

	if mutation.Kind != AuthoringOperationReorderComponent || mutation.PageKey != "home" || mutation.ComponentKey != "gallery" {
		t.Fatalf("unexpected reorder mutation: %#v", mutation)
	}
	if !mutation.HasPosition || mutation.Position != 1 {
		t.Fatalf("expected explicit target position: %#v", mutation)
	}
	if values[AuthoringFieldOperation] != "reorder-component" || values[AuthoringFieldPosition] != "1" || values[AuthoringFieldBinding] != "home.section.gallery" {
		t.Fatalf("unexpected reorder form values: %#v", values)
	}
}

func TestAuthoringMutationForComponentDeleteBuildsComponentPayload(t *testing.T) {
	mutation := AuthoringMutationForComponentDelete(Page{
		Key:   "home",
		Label: "Home",
		Route: "/",
	}, Component{
		Key:     "gallery",
		Label:   "Gallery",
		Binding: "home.section.gallery",
	})
	values := mutation.FormValues()

	if mutation.Kind != AuthoringOperationDeleteComponent || mutation.PageKey != "home" || mutation.ComponentKey != "gallery" {
		t.Fatalf("unexpected delete mutation: %#v", mutation)
	}
	if values[AuthoringFieldOperation] != "delete-component" || values[AuthoringFieldBinding] != "home.section.gallery" {
		t.Fatalf("unexpected delete form values: %#v", values)
	}
}

func TestAuthoringMutationFromFormNormalizesAndValidates(t *testing.T) {
	mutation, validation := AuthoringMutationFromForm(map[string]string{
		AuthoringFieldOperation:    " save-control ",
		AuthoringFieldPageKey:      " home ",
		AuthoringFieldComponentKey: " hero ",
		AuthoringFieldControlKey:   " headline ",
		AuthoringFieldControlKind:  " text ",
		AuthoringFieldBinding:      " pages.home.hero.headline ",
		AuthoringFieldValue:        " Welcome families ",
	})
	if !validation.OK() {
		t.Fatalf("expected valid mutation, got %#v", validation)
	}
	if mutation.Kind != AuthoringOperationSaveControl || mutation.PageKey != "home" || mutation.ComponentKey != "hero" {
		t.Fatalf("unexpected mutation target: %#v", mutation)
	}
	if mutation.ControlKey != "headline" || mutation.Binding != "pages.home.hero.headline" || mutation.Value != "Welcome families" {
		t.Fatalf("unexpected control mutation: %#v", mutation)
	}

	invalid, validation := AuthoringMutationFromForm(map[string]string{
		AuthoringFieldOperation:    "reorder-component",
		AuthoringFieldPageKey:      "home",
		AuthoringFieldComponentKey: "hero",
		AuthoringFieldPosition:     "later",
	})
	if validation.OK() || invalid.HasPosition {
		t.Fatalf("expected invalid position, mutation=%#v validation=%#v", invalid, validation)
	}
	if validation.FieldErrors[AuthoringFieldPosition] != "Enter a whole-number position." {
		t.Fatalf("unexpected position error: %#v", validation.FieldErrors)
	}

	missing, validation := AuthoringMutationFromForm(map[string]string{
		AuthoringFieldOperation:  "apply-intent",
		AuthoringFieldIntentKey:  "create-page:landing",
		AuthoringFieldIntentKind: string(CompositionIntentCreatePage),
	})
	if validation.OK() || missing.PageBlueprintKey != "" {
		t.Fatalf("expected missing blueprint error, mutation=%#v validation=%#v", missing, validation)
	}
	if validation.FieldErrors[AuthoringFieldPageBlueprintKey] == "" {
		t.Fatalf("expected page blueprint error: %#v", validation.FieldErrors)
	}
}

func TestAuthoringActionHandlerInvokesAdapter(t *testing.T) {
	adapter := &stubAuthoringAdapter{result: AuthoringMutationResult{
		Message:        "Updated hero.",
		PreviewURL:     "/?gosx-preview=1",
		RefreshPreview: true,
		Changes: []AuthoringChange{{
			Key:       "hero-headline",
			Label:     "Hero headline",
			Kind:      "control",
			PageKey:   "home",
			Component: "hero",
			Binding:   "pages.home.hero.headline",
		}},
	}}
	form := url.Values{}
	form.Set(AuthoringFieldOperation, "save-control")
	form.Set(AuthoringFieldPageKey, "home")
	form.Set(AuthoringFieldComponentKey, "hero")
	form.Set(AuthoringFieldControlKey, "headline")
	form.Set(AuthoringFieldBinding, "pages.home.hero.headline")
	form.Set(AuthoringFieldValue, "New headline")

	req := httptest.NewRequest(http.MethodPost, "/gosx/action/authoring", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	action.ServeHandler(rec, req, AuthoringActionHandler(adapter))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if adapter.calls != 1 || adapter.mutation.Value != "New headline" || adapter.mutation.ControlKey != "headline" {
		t.Fatalf("adapter did not receive mutation: calls=%d mutation=%#v", adapter.calls, adapter.mutation)
	}
	var result action.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Message != "Updated hero." {
		t.Fatalf("unexpected action result: %#v", result)
	}
	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data["previewURL"] != "/?gosx-preview=1" || data["refreshPreview"] != true || data["changeCount"].(float64) != 1 {
		t.Fatalf("unexpected result data: %#v", data)
	}
}

func TestAuthoringActionHandlerReturnsValidationFailure(t *testing.T) {
	adapter := &stubAuthoringAdapter{}
	form := url.Values{}
	form.Set(AuthoringFieldOperation, "toggle-visibility")
	form.Set(AuthoringFieldPageKey, "home")
	form.Set(AuthoringFieldComponentKey, "hero")

	req := httptest.NewRequest(http.MethodPost, "/gosx/action/authoring", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	action.ServeHandler(rec, req, AuthoringActionHandler(adapter))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	if adapter.calls != 0 {
		t.Fatalf("adapter should not be called on validation failure")
	}
	var result action.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.OK || result.FieldErrors[AuthoringFieldVisible] == "" {
		t.Fatalf("unexpected validation result: %#v", result)
	}
}
