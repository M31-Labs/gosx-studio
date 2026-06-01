package studio

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"m31labs.dev/gosx/action"
)

type AuthoringOperationKind string

const (
	AuthoringOperationApplyIntent        AuthoringOperationKind = "apply-intent"
	AuthoringOperationSaveControl        AuthoringOperationKind = "save-control"
	AuthoringOperationReorderComponent   AuthoringOperationKind = "reorder-component"
	AuthoringOperationToggleVisibility   AuthoringOperationKind = "toggle-visibility"
	AuthoringOperationDuplicateComponent AuthoringOperationKind = "duplicate-component"
	AuthoringOperationDeleteComponent    AuthoringOperationKind = "delete-component"
	AuthoringOperationUpdatePage         AuthoringOperationKind = "update-page"
)

const (
	AuthoringFieldOperation            = "gosx_studio_operation"
	AuthoringFieldIntentKey            = "gosx_studio_intent_key"
	AuthoringFieldIntentKind           = "gosx_studio_intent_kind"
	AuthoringFieldPageKey              = "gosx_studio_page_key"
	AuthoringFieldPageLabel            = "gosx_studio_page_label"
	AuthoringFieldPageRoute            = "gosx_studio_page_route"
	AuthoringFieldPageBlueprintKey     = "gosx_studio_page_blueprint_key"
	AuthoringFieldComponentKey         = "gosx_studio_component_key"
	AuthoringFieldComponentLabel       = "gosx_studio_component_label"
	AuthoringFieldComponentTemplateKey = "gosx_studio_component_template_key"
	AuthoringFieldControlKey           = "gosx_studio_control_key"
	AuthoringFieldControlKind          = "gosx_studio_control_kind"
	AuthoringFieldBinding              = "gosx_studio_binding"
	AuthoringFieldTargetRegion         = "gosx_studio_target_region"
	AuthoringFieldValue                = "gosx_studio_value"
	AuthoringFieldPosition             = "gosx_studio_position"
	AuthoringFieldVisible              = "gosx_studio_visible"
)

// AuthoringAdapter is the host-owned mutation boundary for no-code editing.
//
// Studio normalizes editor operations into AuthoringMutation values. Host apps
// decide how those operations load, validate, persist, preview, and publish.
type AuthoringAdapter interface {
	ApplyAuthoringMutation(ctx context.Context, mutation AuthoringMutation) (AuthoringMutationResult, error)
}

type AuthoringMutation struct {
	Kind                 AuthoringOperationKind
	IntentKey            string
	IntentKind           CompositionIntentKind
	PageKey              string
	PageLabel            string
	PageRoute            string
	PageBlueprintKey     string
	ComponentKey         string
	ComponentLabel       string
	ComponentTemplateKey string
	ControlKey           string
	ControlKind          ControlKind
	Binding              string
	TargetRegion         string
	Value                string
	Position             int
	HasPosition          bool
	Visible              bool
	HasVisible           bool
}

type AuthoringValidation struct {
	Message     string
	FieldErrors map[string]string
	Values      map[string]string
}

type AuthoringMutationResult struct {
	Message        string
	RedirectURL    string
	PreviewURL     string
	DraftID        string
	RefreshPreview bool
	FieldErrors    map[string]string
	Values         map[string]string
	Changes        []AuthoringChange
}

type AuthoringChange struct {
	Key       string
	Label     string
	Summary   string
	Kind      string
	PageKey   string
	Component string
	Binding   string
}

func AuthoringActionHandler(adapter AuthoringAdapter) action.Handler {
	return func(ctx *action.Context) error {
		if adapter == nil {
			return action.Error(http.StatusInternalServerError, "Authoring adapter is not configured.")
		}
		mutation, validation := AuthoringMutationFromForm(ctx.FormData)
		if !validation.OK() {
			ctx.ValidationFailure(validation.Message, validation.FieldErrors)
			return nil
		}
		requestContext := context.Background()
		if ctx != nil && ctx.Request != nil {
			requestContext = ctx.Request.Context()
		}
		result, err := adapter.ApplyAuthoringMutation(requestContext, mutation)
		if err != nil {
			return err
		}
		result = result.Normalize()
		if result.HasErrors() {
			message := firstNonEmpty(result.Message, "Please correct the highlighted fields.")
			ctx.ValidationFailure(message, result.FieldErrors)
			return nil
		}
		if result.RedirectURL != "" {
			ctx.Redirect(result.RedirectURL)
			return nil
		}
		return ctx.Success(firstNonEmpty(result.Message, "Saved."), AuthoringMutationResultView(result))
	}
}

func AuthoringMutationFromIntent(intent CompositionIntent) AuthoringMutation {
	intent = intent.Normalize()
	return AuthoringMutation{
		Kind:                 AuthoringOperationApplyIntent,
		IntentKey:            intent.Key,
		IntentKind:           intent.Kind,
		PageKey:              intent.TargetPageKey,
		PageLabel:            intent.TargetPageLabel,
		PageRoute:            intent.TargetRoute,
		PageBlueprintKey:     intent.PageBlueprintKey,
		ComponentTemplateKey: intent.ComponentTemplateKey,
		ComponentLabel:       intent.ComponentLabel,
		Binding:              intent.Binding,
		TargetRegion:         intent.TargetRegion,
	}.Normalize()
}

func AuthoringMutationForControl(page Page, component Component, control Control) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	control = control.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationSaveControl,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
		ControlKey:     control.Key,
		ControlKind:    control.Kind,
		Binding:        control.Binding,
		Value:          control.Value,
	}.Normalize()
}

func AuthoringMutationForPage(page Page) AuthoringMutation {
	page = page.Normalize()
	return AuthoringMutation{
		Kind:      AuthoringOperationUpdatePage,
		PageKey:   page.Key,
		PageLabel: page.Label,
		PageRoute: page.Route,
	}.Normalize()
}

func AuthoringMutationForComponentReorder(page Page, component Component, position int) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationReorderComponent,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
		Binding:        component.Binding,
		Position:       position,
		HasPosition:    true,
	}.Normalize()
}

func AuthoringMutationForComponentVisibility(page Page, component Component, visible bool) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationToggleVisibility,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
		Binding:        component.Binding,
		Visible:        visible,
		HasVisible:     true,
	}.Normalize()
}

func AuthoringMutationForComponentDelete(page Page, component Component) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationDeleteComponent,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
		Binding:        component.Binding,
	}.Normalize()
}

func AuthoringMutationForComponentDuplicate(page Page, component Component, position int) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:                 AuthoringOperationDuplicateComponent,
		PageKey:              page.Key,
		PageLabel:            page.Label,
		PageRoute:            page.Route,
		ComponentKey:         component.Key,
		ComponentLabel:       component.Label,
		ComponentTemplateKey: firstNonEmpty(component.TemplateKey, component.Key),
		Binding:              component.Binding,
		Position:             position,
		HasPosition:          true,
	}.Normalize()
}

func AuthoringMutationFromForm(form map[string]string) (AuthoringMutation, AuthoringValidation) {
	mutation := AuthoringMutation{
		Kind:                 AuthoringOperationKind(formValue(form, AuthoringFieldOperation)),
		IntentKey:            formValue(form, AuthoringFieldIntentKey),
		IntentKind:           CompositionIntentKind(formValue(form, AuthoringFieldIntentKind)),
		PageKey:              formValue(form, AuthoringFieldPageKey),
		PageLabel:            formValue(form, AuthoringFieldPageLabel),
		PageRoute:            formValue(form, AuthoringFieldPageRoute),
		PageBlueprintKey:     formValue(form, AuthoringFieldPageBlueprintKey),
		ComponentKey:         formValue(form, AuthoringFieldComponentKey),
		ComponentLabel:       formValue(form, AuthoringFieldComponentLabel),
		ComponentTemplateKey: formValue(form, AuthoringFieldComponentTemplateKey),
		ControlKey:           formValue(form, AuthoringFieldControlKey),
		ControlKind:          ControlKind(formValue(form, AuthoringFieldControlKind)),
		Binding:              formValue(form, AuthoringFieldBinding),
		TargetRegion:         formValue(form, AuthoringFieldTargetRegion),
		Value:                formValue(form, AuthoringFieldValue),
	}
	validation := AuthoringValidation{Values: cloneStringMap(form)}
	if position, ok, valid := parseAuthoringInt(formValue(form, AuthoringFieldPosition)); ok {
		if valid {
			mutation.Position = position
			mutation.HasPosition = true
		} else {
			validation.AddFieldError(AuthoringFieldPosition, "Enter a whole-number position.")
		}
	}
	if visible, ok, valid := parseAuthoringBool(formValue(form, AuthoringFieldVisible)); ok {
		if valid {
			mutation.Visible = visible
			mutation.HasVisible = true
		} else {
			validation.AddFieldError(AuthoringFieldVisible, "Use true or false.")
		}
	}
	mutation = mutation.Normalize()
	for field, message := range mutation.Validate().FieldErrors {
		validation.AddFieldError(field, message)
	}
	if validation.Message == "" && len(validation.FieldErrors) > 0 {
		validation.Message = "Please correct the highlighted fields."
	}
	return mutation, validation
}

func (mutation AuthoringMutation) Normalize() AuthoringMutation {
	mutation.Kind = normalizeAuthoringOperationKind(mutation.Kind)
	mutation.IntentKey = strings.TrimSpace(mutation.IntentKey)
	if strings.TrimSpace(string(mutation.IntentKind)) != "" {
		mutation.IntentKind = normalizeCompositionIntentKind(mutation.IntentKind)
	}
	mutation.PageKey = strings.TrimSpace(mutation.PageKey)
	mutation.PageLabel = strings.TrimSpace(mutation.PageLabel)
	mutation.PageRoute = strings.TrimSpace(mutation.PageRoute)
	mutation.PageBlueprintKey = strings.TrimSpace(mutation.PageBlueprintKey)
	mutation.ComponentKey = strings.TrimSpace(mutation.ComponentKey)
	mutation.ComponentLabel = strings.TrimSpace(mutation.ComponentLabel)
	mutation.ComponentTemplateKey = strings.TrimSpace(mutation.ComponentTemplateKey)
	if strings.TrimSpace(string(mutation.ControlKind)) != "" {
		mutation.ControlKind = normalizeControlKind(mutation.ControlKind)
	}
	mutation.ControlKey = strings.TrimSpace(mutation.ControlKey)
	mutation.Binding = strings.TrimSpace(mutation.Binding)
	mutation.TargetRegion = strings.TrimSpace(mutation.TargetRegion)
	mutation.Value = strings.TrimSpace(mutation.Value)
	if mutation.Position < 0 {
		mutation.Position = 0
	}
	return mutation
}

func (mutation AuthoringMutation) Validate() AuthoringValidation {
	mutation = mutation.Normalize()
	validation := AuthoringValidation{Values: mutation.FormValues()}
	if mutation.Kind == "" {
		validation.AddFieldError(AuthoringFieldOperation, "Choose an authoring operation.")
		return validation.withDefaultMessage()
	}
	switch mutation.Kind {
	case AuthoringOperationApplyIntent:
		if mutation.IntentKey == "" {
			validation.AddFieldError(AuthoringFieldIntentKey, "Choose an authoring intent.")
		}
		if mutation.IntentKind == "" {
			validation.AddFieldError(AuthoringFieldIntentKind, "Choose an intent kind.")
		}
		switch mutation.IntentKind {
		case CompositionIntentCreatePage:
			if mutation.PageBlueprintKey == "" {
				validation.AddFieldError(AuthoringFieldPageBlueprintKey, "Choose a page blueprint.")
			}
		case CompositionIntentAddComponent:
			if mutation.PageKey == "" {
				validation.AddFieldError(AuthoringFieldPageKey, "Choose a target page.")
			}
			if mutation.ComponentTemplateKey == "" {
				validation.AddFieldError(AuthoringFieldComponentTemplateKey, "Choose a component template.")
			}
		}
	case AuthoringOperationSaveControl:
		requirePageComponent(&validation, mutation)
		if mutation.ControlKey == "" {
			validation.AddFieldError(AuthoringFieldControlKey, "Choose a control.")
		}
	case AuthoringOperationReorderComponent:
		requirePageComponent(&validation, mutation)
		if !mutation.HasPosition {
			validation.AddFieldError(AuthoringFieldPosition, "Choose a component position.")
		}
	case AuthoringOperationToggleVisibility:
		requirePageComponent(&validation, mutation)
		if !mutation.HasVisible {
			validation.AddFieldError(AuthoringFieldVisible, "Choose a visibility state.")
		}
	case AuthoringOperationDuplicateComponent, AuthoringOperationDeleteComponent:
		requirePageComponent(&validation, mutation)
	case AuthoringOperationUpdatePage:
		if mutation.PageKey == "" {
			validation.AddFieldError(AuthoringFieldPageKey, "Choose a page.")
		}
		if mutation.PageLabel == "" && mutation.PageRoute == "" {
			validation.AddFieldError(AuthoringFieldPageLabel, "Enter a page label or route.")
		}
	}
	return validation.withDefaultMessage()
}

func (mutation AuthoringMutation) FormValues() map[string]string {
	mutation = mutation.Normalize()
	values := map[string]string{}
	setFormValue(values, AuthoringFieldOperation, string(mutation.Kind))
	setFormValue(values, AuthoringFieldIntentKey, mutation.IntentKey)
	setFormValue(values, AuthoringFieldIntentKind, string(mutation.IntentKind))
	setFormValue(values, AuthoringFieldPageKey, mutation.PageKey)
	setFormValue(values, AuthoringFieldPageLabel, mutation.PageLabel)
	setFormValue(values, AuthoringFieldPageRoute, mutation.PageRoute)
	setFormValue(values, AuthoringFieldPageBlueprintKey, mutation.PageBlueprintKey)
	setFormValue(values, AuthoringFieldComponentKey, mutation.ComponentKey)
	setFormValue(values, AuthoringFieldComponentLabel, mutation.ComponentLabel)
	setFormValue(values, AuthoringFieldComponentTemplateKey, mutation.ComponentTemplateKey)
	setFormValue(values, AuthoringFieldControlKey, mutation.ControlKey)
	setFormValue(values, AuthoringFieldControlKind, string(mutation.ControlKind))
	setFormValue(values, AuthoringFieldBinding, mutation.Binding)
	setFormValue(values, AuthoringFieldTargetRegion, mutation.TargetRegion)
	setFormValue(values, AuthoringFieldValue, mutation.Value)
	if mutation.HasPosition {
		values[AuthoringFieldPosition] = strconv.Itoa(mutation.Position)
	}
	if mutation.HasVisible {
		values[AuthoringFieldVisible] = strconv.FormatBool(mutation.Visible)
	}
	return values
}

func AuthoringMutationView(mutation AuthoringMutation) map[string]any {
	mutation = mutation.Normalize()
	return map[string]any{
		"kind":                 string(mutation.Kind),
		"intentKey":            mutation.IntentKey,
		"intentKind":           string(mutation.IntentKind),
		"pageKey":              mutation.PageKey,
		"pageLabel":            mutation.PageLabel,
		"pageRoute":            mutation.PageRoute,
		"pageBlueprintKey":     mutation.PageBlueprintKey,
		"componentKey":         mutation.ComponentKey,
		"componentLabel":       mutation.ComponentLabel,
		"componentTemplateKey": mutation.ComponentTemplateKey,
		"controlKey":           mutation.ControlKey,
		"controlKind":          string(mutation.ControlKind),
		"binding":              mutation.Binding,
		"targetRegion":         mutation.TargetRegion,
		"value":                mutation.Value,
		"position":             mutation.Position,
		"hasPosition":          mutation.HasPosition,
		"visible":              mutation.Visible,
		"hasVisible":           mutation.HasVisible,
		"formValues":           mutation.FormValues(),
	}
}

func AuthoringFieldNamesView() map[string]string {
	return map[string]string{
		"operation":            AuthoringFieldOperation,
		"intentKey":            AuthoringFieldIntentKey,
		"intentKind":           AuthoringFieldIntentKind,
		"pageKey":              AuthoringFieldPageKey,
		"pageLabel":            AuthoringFieldPageLabel,
		"pageRoute":            AuthoringFieldPageRoute,
		"pageBlueprintKey":     AuthoringFieldPageBlueprintKey,
		"componentKey":         AuthoringFieldComponentKey,
		"componentLabel":       AuthoringFieldComponentLabel,
		"componentTemplateKey": AuthoringFieldComponentTemplateKey,
		"controlKey":           AuthoringFieldControlKey,
		"controlKind":          AuthoringFieldControlKind,
		"binding":              AuthoringFieldBinding,
		"targetRegion":         AuthoringFieldTargetRegion,
		"value":                AuthoringFieldValue,
		"position":             AuthoringFieldPosition,
		"visible":              AuthoringFieldVisible,
	}
}

func (validation AuthoringValidation) OK() bool {
	return len(validation.FieldErrors) == 0
}

func (validation *AuthoringValidation) AddFieldError(field, message string) {
	if validation.FieldErrors == nil {
		validation.FieldErrors = map[string]string{}
	}
	field = strings.TrimSpace(field)
	message = strings.TrimSpace(message)
	if field == "" || message == "" {
		return
	}
	if _, exists := validation.FieldErrors[field]; exists {
		return
	}
	validation.FieldErrors[field] = message
}

func (validation AuthoringValidation) withDefaultMessage() AuthoringValidation {
	if validation.Message == "" && len(validation.FieldErrors) > 0 {
		validation.Message = "Please correct the highlighted fields."
	}
	if validation.Values == nil {
		validation.Values = map[string]string{}
	}
	return validation
}

func (result AuthoringMutationResult) Normalize() AuthoringMutationResult {
	result.Message = strings.TrimSpace(result.Message)
	result.RedirectURL = strings.TrimSpace(result.RedirectURL)
	result.PreviewURL = strings.TrimSpace(result.PreviewURL)
	result.DraftID = strings.TrimSpace(result.DraftID)
	result.FieldErrors = cloneStringMap(result.FieldErrors)
	result.Values = cloneStringMap(result.Values)
	result.Changes = normalizeAuthoringChanges(result.Changes)
	return result
}

func (result AuthoringMutationResult) HasErrors() bool {
	return len(result.FieldErrors) > 0
}

func AuthoringMutationResultView(result AuthoringMutationResult) map[string]any {
	result = result.Normalize()
	return map[string]any{
		"message":        result.Message,
		"redirectURL":    result.RedirectURL,
		"previewURL":     result.PreviewURL,
		"draftID":        result.DraftID,
		"refreshPreview": result.RefreshPreview,
		"fieldErrors":    cloneStringMap(result.FieldErrors),
		"values":         cloneStringMap(result.Values),
		"changes":        authoringChangeViews(result.Changes),
		"changeCount":    len(result.Changes),
	}
}

func normalizeAuthoringOperationKind(kind AuthoringOperationKind) AuthoringOperationKind {
	switch AuthoringOperationKind(strings.TrimSpace(string(kind))) {
	case AuthoringOperationApplyIntent:
		return AuthoringOperationApplyIntent
	case AuthoringOperationSaveControl:
		return AuthoringOperationSaveControl
	case AuthoringOperationReorderComponent:
		return AuthoringOperationReorderComponent
	case AuthoringOperationToggleVisibility:
		return AuthoringOperationToggleVisibility
	case AuthoringOperationDuplicateComponent:
		return AuthoringOperationDuplicateComponent
	case AuthoringOperationDeleteComponent:
		return AuthoringOperationDeleteComponent
	case AuthoringOperationUpdatePage:
		return AuthoringOperationUpdatePage
	default:
		return ""
	}
}

func requirePageComponent(validation *AuthoringValidation, mutation AuthoringMutation) {
	if mutation.PageKey == "" {
		validation.AddFieldError(AuthoringFieldPageKey, "Choose a target page.")
	}
	if mutation.ComponentKey == "" {
		validation.AddFieldError(AuthoringFieldComponentKey, "Choose a component.")
	}
}

func formValue(form map[string]string, key string) string {
	return strings.TrimSpace(form[key])
}

func setFormValue(values map[string]string, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		values[key] = value
	}
}

func parseAuthoringInt(value string) (int, bool, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, true, false
	}
	return parsed, true, true
}

func parseAuthoringBool(value string) (bool, bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return false, false, true
	case "1", "true", "t", "yes", "y", "on":
		return true, true, true
	case "0", "false", "f", "no", "n", "off":
		return false, true, true
	default:
		return false, true, false
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func normalizeAuthoringChanges(changes []AuthoringChange) []AuthoringChange {
	out := make([]AuthoringChange, 0, len(changes))
	for _, change := range changes {
		change.Key = strings.TrimSpace(change.Key)
		change.Label = strings.TrimSpace(change.Label)
		change.Summary = strings.TrimSpace(change.Summary)
		change.Kind = strings.TrimSpace(change.Kind)
		change.PageKey = strings.TrimSpace(change.PageKey)
		change.Component = strings.TrimSpace(change.Component)
		change.Binding = strings.TrimSpace(change.Binding)
		if change.Key == "" && change.Label == "" && change.Summary == "" {
			continue
		}
		out = append(out, change)
	}
	return out
}

func authoringChangeViews(changes []AuthoringChange) []map[string]any {
	changes = normalizeAuthoringChanges(changes)
	out := make([]map[string]any, 0, len(changes))
	for _, change := range changes {
		out = append(out, map[string]any{
			"key":       change.Key,
			"label":     change.Label,
			"summary":   change.Summary,
			"kind":      change.Kind,
			"pageKey":   change.PageKey,
			"component": change.Component,
			"binding":   change.Binding,
		})
	}
	return out
}
