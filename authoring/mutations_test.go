package authoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"m31labs.dev/gosx-studio/core"

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
	intent := core.CompositionIntent{
		Key:                  "add-component:home:gallery",
		Label:                "Add gallery",
		Kind:                 core.CompositionIntentAddComponent,
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

	if mutation.Kind != AuthoringOperationApplyIntent || mutation.IntentKind != core.CompositionIntentAddComponent {
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
	inputs := view["formInputs"].([]map[string]string)
	if len(inputs) == 0 || inputs[0]["name"] != AuthoringFieldOperation || inputs[0]["value"] != "apply-intent" {
		t.Fatalf("expected ordered operation form input: %#v", inputs)
	}
	if !authoringFormInputsContain(inputs, AuthoringFieldComponentTemplateKey, "gallery") ||
		!authoringFormInputsContain(inputs, AuthoringFieldBinding, "media.gallery") {
		t.Fatalf("expected intent form inputs to include component and binding values: %#v", inputs)
	}
}

func TestAuthoringMutationForPageBuildsUpdatePayload(t *testing.T) {
	mutation := AuthoringMutationForPage(core.Page{
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
	mutation := AuthoringMutationForComponentVisibility(core.Page{
		Key:   "home",
		Label: "Home",
		Route: "/",
	}, core.Component{
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
	mutation := AuthoringMutationForComponentReorder(core.Page{
		Key:   "home",
		Label: "Home",
		Route: "/",
	}, core.Component{
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
	mutation := AuthoringMutationForComponentDelete(core.Page{
		Key:   "home",
		Label: "Home",
		Route: "/",
	}, core.Component{
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

func TestAuthoringMutationForComponentDuplicateBuildsTemplatePayload(t *testing.T) {
	mutation := AuthoringMutationForComponentDuplicate(core.Page{
		Key:   "home",
		Label: "Home",
		Route: "/",
	}, core.Component{
		Key:         "hero-copy",
		TemplateKey: "hero",
		Label:       "Hero copy",
		Binding:     "home.section.hero-copy",
	}, 2)
	values := mutation.FormValues()

	if mutation.Kind != AuthoringOperationDuplicateComponent || mutation.PageKey != "home" || mutation.ComponentKey != "hero-copy" || mutation.ComponentTemplateKey != "hero" {
		t.Fatalf("unexpected duplicate mutation: %#v", mutation)
	}
	if !mutation.HasPosition || mutation.Position != 2 {
		t.Fatalf("expected explicit duplicate target position: %#v", mutation)
	}
	if values[AuthoringFieldOperation] != "duplicate-component" || values[AuthoringFieldComponentTemplateKey] != "hero" || values[AuthoringFieldPosition] != "2" {
		t.Fatalf("unexpected duplicate form values: %#v", values)
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
		AuthoringFieldIntentKind: string(core.CompositionIntentCreatePage),
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
		FragmentURL:    "/admin/editor",
		RefreshPreview: true,
		Fragments: AuthoringRefreshFragments(
			"[data-studio-composition-intent-forms='true']",
			"[data-studio-site-map-panel='true']",
		),
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
	if data["fragmentURL"] != "/admin/editor" || data["fragmentCount"].(float64) != 2 {
		t.Fatalf("unexpected fragment result data: %#v", data)
	}
	fragments := data["fragments"].([]any)
	firstFragment := fragments[0].(map[string]any)
	if firstFragment["selector"] != "[data-studio-composition-intent-forms='true']" || firstFragment["mode"] != "replace" {
		t.Fatalf("unexpected fragment view: %#v", fragments)
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

func authoringFormInputsContain(inputs []map[string]string, name, value string) bool {
	for _, input := range inputs {
		if input["name"] == name && input["value"] == value {
			return true
		}
	}
	return false
}

// TestSaveAppearanceIsValidOperation asserts that AuthoringOperationSaveAppearance
// is recognized as a valid operation kind (normalizes correctly, does not become "").
func TestSaveAppearanceIsValidOperation(t *testing.T) {
	if AuthoringOperationSaveAppearance == "" {
		t.Fatal("AuthoringOperationSaveAppearance must not be empty")
	}
	if string(AuthoringOperationSaveAppearance) != "save-appearance" {
		t.Fatalf("expected \"save-appearance\", got %q", AuthoringOperationSaveAppearance)
	}
	normalized := normalizeAuthoringOperationKind(AuthoringOperationSaveAppearance)
	if normalized != AuthoringOperationSaveAppearance {
		t.Fatalf("normalizeAuthoringOperationKind must preserve save-appearance, got %q", normalized)
	}
}

// TestSaveAppearanceMutationFromFormRoundTripsColorFields asserts that a
// save-appearance form submission exposes all submitted color fields to the
// host adapter via AuthoringMutation.RawForm.
//
// This is the delivery contract for the host ApplyAuthoringMutation handler:
// read color values from mutation.RawForm["colorAccent"], mutation.RawForm["colorCanvas"], etc.
func TestSaveAppearanceMutationFromFormRoundTripsColorFields(t *testing.T) {
	form := map[string]string{
		AuthoringFieldOperation: "save-appearance",
		"colorAccent":           "#b5651d",
		"colorCanvas":           "#f8f5ef",
		"colorInk":              "#1a1a1a",
	}
	mutation, validation := AuthoringMutationFromForm(form)
	if !validation.OK() {
		t.Fatalf("save-appearance must be valid, got: %#v", validation)
	}
	if mutation.Kind != AuthoringOperationSaveAppearance {
		t.Fatalf("expected Kind=save-appearance, got %q", mutation.Kind)
	}
	// Host adapter reads color values from RawForm.
	if mutation.RawForm == nil {
		t.Fatal("mutation.RawForm must be populated for save-appearance so host can read color fields")
	}
	if mutation.RawForm["colorAccent"] != "#b5651d" {
		t.Fatalf("expected RawForm[colorAccent]=#b5651d, got %q", mutation.RawForm["colorAccent"])
	}
	if mutation.RawForm["colorCanvas"] != "#f8f5ef" {
		t.Fatalf("expected RawForm[colorCanvas]=#f8f5ef, got %q", mutation.RawForm["colorCanvas"])
	}
	if mutation.RawForm["colorInk"] != "#1a1a1a" {
		t.Fatalf("expected RawForm[colorInk]=#1a1a1a, got %q", mutation.RawForm["colorInk"])
	}
}

// TestSaveAppearanceMutationAdapterReceivesColorFields asserts that the authoring
// action handler invokes the adapter with a mutation whose RawForm carries all
// submitted color fields from the HTTP form.
func TestSaveAppearanceMutationAdapterReceivesColorFields(t *testing.T) {
	adapter := &stubAuthoringAdapter{result: AuthoringMutationResult{
		Message:        "Palette saved.",
		RefreshPreview: true,
	}}

	form := url.Values{}
	form.Set(AuthoringFieldOperation, "save-appearance")
	form.Set("colorAccent", "#b5651d")
	form.Set("colorCanvas", "#f8f5ef")
	form.Set("colorInk", "#1a1a1a")

	req := httptest.NewRequest(http.MethodPost, "/gosx/action/authoring", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	action.ServeHandler(rec, req, AuthoringActionHandler(adapter))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if adapter.calls != 1 {
		t.Fatalf("expected 1 adapter call, got %d", adapter.calls)
	}
	if adapter.mutation.Kind != AuthoringOperationSaveAppearance {
		t.Fatalf("adapter must receive save-appearance mutation, got %q", adapter.mutation.Kind)
	}
	// Host reads colors from RawForm
	if adapter.mutation.RawForm["colorAccent"] != "#b5651d" {
		t.Fatalf("adapter.mutation.RawForm must carry colorAccent, got %q", adapter.mutation.RawForm["colorAccent"])
	}
	if adapter.mutation.RawForm["colorCanvas"] != "#f8f5ef" {
		t.Fatalf("adapter.mutation.RawForm must carry colorCanvas, got %q", adapter.mutation.RawForm["colorCanvas"])
	}

	var result action.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Message != "Palette saved." {
		t.Fatalf("unexpected action result: %#v", result)
	}
}

func TestAuthoringMutationResultViewCarriesAuthoritativeDurableValue(t *testing.T) {
	view := AuthoringMutationResultView(AuthoringMutationResult{
		DocumentRevision: 4,
		TargetHead:       "head-4",
		OperationID:      "op-4",
		Value:            "  meaningful content  ",
	})
	if got := view["value"]; got != "  meaningful content  " {
		t.Fatalf("durable value must preserve content exactly, got %#v", got)
	}
}
