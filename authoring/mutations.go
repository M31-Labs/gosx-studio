package authoring

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"m31labs.dev/gosx-studio/core"

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
	// AuthoringOperationSaveAppearance persists brand color tokens (and future
	// appearance fields) in a single save through the draft→publish pipeline.
	// Because this operation carries N color fields with host-defined names
	// rather than a single well-known binding/value pair, the host adapter must
	// read all submitted color values from AuthoringMutation.RawForm (e.g.
	// mutation.RawForm["colorAccent"], mutation.RawForm["colorCanvas"]).
	AuthoringOperationSaveAppearance AuthoringOperationKind = "save-appearance"
	// AuthoringOperationSetStyle sets a single CSS property on one component for a
	// given responsive breakpoint and element state. It is the typed spine of
	// per-element visual styling: the style inspector and the canvas
	// direct-manipulation overlay both emit set-style mutations. The property is
	// validated against a curated allow-list and the value is sanitized
	// server-side (see style_mutations.go), so the browser can never push an
	// illegal or injecting declaration into a page.
	AuthoringOperationSetStyle   AuthoringOperationKind = "set-style"
	AuthoringOperationSetField   AuthoringOperationKind = "set-field"
	AuthoringOperationResetStyle AuthoringOperationKind = "reset-style"
	AuthoringOperationUndo       AuthoringOperationKind = "undo"
	AuthoringOperationRedo       AuthoringOperationKind = "redo"
	// The four operations below implement the reusable component/instance
	// model (see core/instances.go): a shared ComponentDefinition's Controls
	// fan out to every attached Component instance, and one instance can
	// diverge one field at a time (override) or completely (detach/restore).
	//
	// AuthoringOperationSetSharedField edits one control on the shared
	// definition itself, identified by ComponentTemplateKey (the definition
	// key) and ControlKey. It fans out to every attached instance because
	// attached instances always resolve through core.EffectiveComponentControls
	// — no per-instance write follows a shared edit.
	AuthoringOperationSetSharedField AuthoringOperationKind = "set-shared-field"
	// AuthoringOperationOverrideInstance sets one instance-local control
	// override, identified by PageKey+ComponentKey (the placed instance) and
	// ControlKey. Only that control diverges; the rest of the instance keeps
	// following the shared definition.
	AuthoringOperationOverrideInstance AuthoringOperationKind = "override-instance"
	// AuthoringOperationDetachInstance freezes one instance's currently
	// effective values as its own independent copy and stops it from
	// following further shared edits.
	AuthoringOperationDetachInstance AuthoringOperationKind = "detach-instance"
	// AuthoringOperationRestoreInstance re-attaches a detached (or
	// override-diverged) instance to its shared definition, dropping any
	// instance-local state.
	AuthoringOperationRestoreInstance AuthoringOperationKind = "restore-instance"
	// The two operations below implement the no-code interaction schema
	// (core/interactions.go): a typed, allow-listed reveal-on-scroll or
	// hover-focus-state primitive attached to one identified canvas object
	// (PageKey/ComponentKey name the same page+block target the style
	// operations already use). Every field is re-validated server-side against
	// core's allow-lists — the operation carries no script, only enum choices.
	//
	// AuthoringOperationSetInteraction creates or updates one interaction.
	AuthoringOperationSetInteraction AuthoringOperationKind = "set-interaction"
	// AuthoringOperationRemoveInteraction detaches one interaction from its
	// target, identified by InteractionKey.
	AuthoringOperationRemoveInteraction AuthoringOperationKind = "remove-interaction"
	// The four operations below extend the flow designer (panels/flow_designer.go)
	// with durable editing of one action's fields/validation and its submit
	// binding, following the same host-neutral protocol as the instance ops
	// above. FlowFieldRequired doubles as the field's validation rule (see
	// core.FlowAction.ValidatePayload); the flow's submit binding reuses the
	// ordinary Binding field so it composes with core.BindingDiagnostic /
	// core.BindingResolver the same way a component resource binding does.
	//
	// AuthoringOperationSetFlowField adds or updates one field
	// (name/label/kind/required) on a flow action.
	AuthoringOperationSetFlowField AuthoringOperationKind = "set-flow-field"
	// AuthoringOperationRemoveFlowField removes one field from a flow action.
	AuthoringOperationRemoveFlowField AuthoringOperationKind = "remove-flow-field"
	// AuthoringOperationSetFlowAction updates a flow action's label and/or its
	// submit binding (Binding carries the handler ref, e.g. "flow.contact.submit").
	AuthoringOperationSetFlowAction AuthoringOperationKind = "set-flow-action"
	// AuthoringOperationTestFlowAction runs one flow action against submitted
	// test values through an isolated handler (authoring.FlowTestHandler) —
	// never the production handler — so an operator can verify field
	// validation and the submit binding before publish. Test values are
	// host-defined field names, so they travel in RawForm like save-appearance.
	AuthoringOperationTestFlowAction AuthoringOperationKind = "test-flow-action"
	// AuthoringOperationPlaceInstance places another instance of an EXISTING
	// shared ComponentDefinition (identified by ComponentTemplateKey, the same
	// field AuthoringOperationSetSharedField already reuses for a definition
	// key) onto PageKey. It is the durable counterpart of the reusable
	// component/instance family above (core/instances.go) for the ONE
	// operation that family didn't yet have: placement itself. ComponentKey/
	// ComponentLabel are an optional operator-supplied instance key/label —
	// when ComponentKey is empty the host picks one (see
	// InstancePlaceWriter). This is the operation the composition workspace
	// palette (sitemap.AuthoringSiteMapView's "palette" — every
	// ComponentTemplate plus every reusable ComponentDefinition available for
	// another instance) dispatches so shared components are placed from the
	// SAME palette/operation family as everything else, rather than a
	// standalone host-specific action.
	AuthoringOperationPlaceInstance AuthoringOperationKind = "place-instance"
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
	AuthoringFieldStyleProperty        = "gosx_studio_style_property"
	AuthoringFieldStyleValue           = "gosx_studio_style_value"
	AuthoringFieldBreakpoint           = "gosx_studio_breakpoint"
	AuthoringFieldState                = "gosx_studio_state"
	AuthoringFieldPosition             = "gosx_studio_position"
	AuthoringFieldVisible              = "gosx_studio_visible"
	AuthoringFieldOperationID          = "gosx_studio_operation_id"
	AuthoringFieldExpectedRevision     = "gosx_studio_expected_revision"
	AuthoringFieldExpectedTargetHead   = "gosx_studio_expected_target_head"
	AuthoringFieldExpectedTargetValue  = "gosx_studio_expected_target_value"
	AuthoringFieldHistoryOperationID   = "gosx_studio_history_operation_id"
	// Interaction fields (set-interaction / remove-interaction).
	AuthoringFieldInteractionKey        = "gosx_studio_interaction_key"
	AuthoringFieldInteractionKind       = "gosx_studio_interaction_kind"
	AuthoringFieldInteractionEffect     = "gosx_studio_interaction_effect"
	AuthoringFieldInteractionDurationMS = "gosx_studio_interaction_duration_ms"
	AuthoringFieldInteractionDelayMS    = "gosx_studio_interaction_delay_ms"
	AuthoringFieldInteractionOnce       = "gosx_studio_interaction_once"
	// Flow designer fields (set-flow-field / remove-flow-field / set-flow-action / test-flow-action).
	AuthoringFieldFlowKey           = "gosx_studio_flow_key"
	AuthoringFieldFlowActionKey     = "gosx_studio_flow_action_key"
	AuthoringFieldFlowActionLabel   = "gosx_studio_flow_action_label"
	AuthoringFieldFlowFieldName     = "gosx_studio_flow_field_name"
	AuthoringFieldFlowFieldLabel    = "gosx_studio_flow_field_label"
	AuthoringFieldFlowFieldRequired = "gosx_studio_flow_field_required"
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
	IntentKind           core.CompositionIntentKind
	PageKey              string
	PageLabel            string
	PageRoute            string
	PageBlueprintKey     string
	ComponentKey         string
	ComponentLabel       string
	ComponentTemplateKey string
	ControlKey           string
	ControlKind          core.ControlKind
	Binding              string
	TargetRegion         string
	Value                string
	StyleProperty        string
	StyleValue           string
	Breakpoint           string
	State                string
	Position             int
	HasPosition          bool
	Visible              bool
	HasVisible           bool
	// Interaction fields (set-interaction / remove-interaction). Target reuses
	// PageKey/PageRoute/ComponentKey — the same page+block address style
	// operations already use — rather than duplicating core.CanvasIdentity.
	InteractionKey        string
	InteractionKind       core.InteractionKind
	InteractionEffect     core.InteractionEffect
	InteractionDurationMS int
	InteractionDelayMS    int
	InteractionOnce       bool
	HasInteractionOnce    bool
	// Flow designer fields (set-flow-field / remove-flow-field / set-flow-action
	// / test-flow-action). FlowFieldKind reuses ControlKind; a flow action's
	// submit binding reuses Binding.
	FlowKey              string
	FlowActionKey        string
	FlowActionLabel      string
	FlowFieldName        string
	FlowFieldLabel       string
	FlowFieldRequired    bool
	HasFlowFieldRequired bool
	// RawForm carries the complete submitted form map for operations that use
	// host-defined field names rather than standard authoring fields.
	// For AuthoringOperationSaveAppearance the host adapter reads color values
	// directly from RawForm (e.g. mutation.RawForm["colorAccent"]).
	// For all other operations RawForm is nil.
	RawForm map[string]string
	// Durable operation metadata. Actor/capabilities are intentionally absent:
	// hosts derive them from the authenticated request context.
	OperationID        string
	ExpectedRevision   uint64
	ExpectedTargetHead string
	// ExpectedTargetValue is carried as JSON in AuthoringFieldExpectedTargetValue.
	// A nil pointer means the value-level precondition was omitted.
	ExpectedTargetValue *OperationValue
	HistoryOperationID  string
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
	FragmentURL    string
	DraftID        string
	RefreshPreview bool
	FieldErrors    map[string]string
	Values         map[string]string
	Changes        []AuthoringChange
	Fragments      []AuthoringRefreshFragment
	// Durable operation cursors returned by a host after commit. They let the
	// next selected-target edit advance its optimistic expected head without a
	// whole-page reload.
	DocumentRevision uint64
	TargetHead       string
	OperationID      string
	// Value is the authoritative target value after a durable operation. Hosts
	// return it for Undo/Redo so the editor can update the selected control
	// immediately without reloading the entire workbench.
	Value string
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

type AuthoringRefreshFragment struct {
	Key      string
	Selector string
	Mode     string
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

func AuthoringMutationFromIntent(intent core.CompositionIntent) AuthoringMutation {
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

func AuthoringMutationForControl(page core.Page, component core.Component, control core.Control) AuthoringMutation {
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

func AuthoringMutationForPage(page core.Page) AuthoringMutation {
	page = page.Normalize()
	return AuthoringMutation{
		Kind:      AuthoringOperationUpdatePage,
		PageKey:   page.Key,
		PageLabel: page.Label,
		PageRoute: page.Route,
	}.Normalize()
}

func AuthoringMutationForComponentReorder(page core.Page, component core.Component, position int) AuthoringMutation {
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

func AuthoringMutationForComponentVisibility(page core.Page, component core.Component, visible bool) AuthoringMutation {
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

func AuthoringMutationForComponentDelete(page core.Page, component core.Component) AuthoringMutation {
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

func AuthoringMutationForComponentDuplicate(page core.Page, component core.Component, position int) AuthoringMutation {
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

// AuthoringMutationForSharedField edits one control on a shared component
// definition. definitionKey identifies the core.ComponentDefinition (carried
// in ComponentTemplateKey, the same field a duplicate-from-template mutation
// uses to name its source template); control.Key/control.Value name the field
// and its new value.
func AuthoringMutationForSharedField(definitionKey string, control core.Control) AuthoringMutation {
	control = control.Normalize()
	return AuthoringMutation{
		Kind:                 AuthoringOperationSetSharedField,
		ComponentTemplateKey: strings.TrimSpace(definitionKey),
		ControlKey:           control.Key,
		ControlKind:          control.Kind,
		Value:                control.Value,
	}.Normalize()
}

// AuthoringMutationForInstanceOverride sets one instance-local control
// override on a placed shared-component instance, leaving every other
// control on the instance following the shared definition.
func AuthoringMutationForInstanceOverride(page core.Page, component core.Component, control core.Control) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	control = control.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationOverrideInstance,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
		ControlKey:     control.Key,
		ControlKind:    control.Kind,
		Value:          control.Value,
	}.Normalize()
}

// AuthoringMutationForInstanceDetach severs a placed shared-component instance
// from its definition, freezing its currently-effective values as its own.
func AuthoringMutationForInstanceDetach(page core.Page, component core.Component) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationDetachInstance,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
	}.Normalize()
}

// AuthoringMutationForInstanceRestore re-attaches a detached (or
// override-diverged) instance to its shared definition.
func AuthoringMutationForInstanceRestore(page core.Page, component core.Component) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationRestoreInstance,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
	}.Normalize()
}

// AuthoringMutationForPlaceInstance places another instance of an EXISTING
// shared ComponentDefinition (definitionKey, carried in ComponentTemplateKey
// like AuthoringMutationForSharedField) onto page. instanceKey/instanceLabel
// are optional — an empty instanceKey lets the host writer choose one (see
// InstancePlaceWriter).
func AuthoringMutationForPlaceInstance(page core.Page, definitionKey, instanceKey, instanceLabel string) AuthoringMutation {
	page = page.Normalize()
	return AuthoringMutation{
		Kind:                 AuthoringOperationPlaceInstance,
		PageKey:              page.Key,
		PageLabel:            page.Label,
		PageRoute:            page.Route,
		ComponentTemplateKey: strings.TrimSpace(definitionKey),
		ComponentKey:         strings.TrimSpace(instanceKey),
		ComponentLabel:       strings.TrimSpace(instanceLabel),
	}.Normalize()
}

// AuthoringMutationForSetInteraction creates or updates one interaction
// attached to a placed component. The interaction is normalized first so the
// mutation always carries the allow-listed trigger/effect/reduced-motion the
// server would derive anyway.
func AuthoringMutationForSetInteraction(page core.Page, component core.Component, interaction core.Interaction) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	interaction = interaction.Normalize()
	return AuthoringMutation{
		Kind:                  AuthoringOperationSetInteraction,
		PageKey:               page.Key,
		PageLabel:             page.Label,
		PageRoute:             page.Route,
		ComponentKey:          component.Key,
		ComponentLabel:        component.Label,
		InteractionKey:        interaction.Key,
		InteractionKind:       interaction.Kind,
		InteractionEffect:     interaction.Effect,
		InteractionDurationMS: interaction.DurationMS,
		InteractionDelayMS:    interaction.DelayMS,
		InteractionOnce:       interaction.Once,
		HasInteractionOnce:    true,
	}.Normalize()
}

// AuthoringMutationForRemoveInteraction detaches one interaction, identified
// by its key, from a placed component.
func AuthoringMutationForRemoveInteraction(page core.Page, component core.Component, interactionKey string) AuthoringMutation {
	page = page.Normalize()
	component = component.Normalize()
	return AuthoringMutation{
		Kind:           AuthoringOperationRemoveInteraction,
		PageKey:        page.Key,
		PageLabel:      page.Label,
		PageRoute:      page.Route,
		ComponentKey:   component.Key,
		ComponentLabel: component.Label,
		InteractionKey: strings.TrimSpace(interactionKey),
	}.Normalize()
}

// AuthoringMutationForSetFlowField adds or updates one field on a flow
// action. field.Required doubles as that field's validation rule.
func AuthoringMutationForSetFlowField(flowKey, actionKey string, field core.FlowField) AuthoringMutation {
	return AuthoringMutation{
		Kind:                 AuthoringOperationSetFlowField,
		FlowKey:              flowKey,
		FlowActionKey:        actionKey,
		FlowFieldName:        field.Name,
		FlowFieldLabel:       field.Label,
		ControlKind:          field.Kind,
		FlowFieldRequired:    field.Required,
		HasFlowFieldRequired: true,
	}.Normalize()
}

// AuthoringMutationForRemoveFlowField removes one field from a flow action.
func AuthoringMutationForRemoveFlowField(flowKey, actionKey, fieldName string) AuthoringMutation {
	return AuthoringMutation{
		Kind:          AuthoringOperationRemoveFlowField,
		FlowKey:       flowKey,
		FlowActionKey: actionKey,
		FlowFieldName: fieldName,
	}.Normalize()
}

// AuthoringMutationForSetFlowAction updates a flow action's label and/or its
// submit binding (the connected handler ref).
func AuthoringMutationForSetFlowAction(flowKey, actionKey, label, handlerRef string) AuthoringMutation {
	return AuthoringMutation{
		Kind:            AuthoringOperationSetFlowAction,
		FlowKey:         flowKey,
		FlowActionKey:   actionKey,
		FlowActionLabel: label,
		Binding:         handlerRef,
	}.Normalize()
}

// AuthoringMutationForTestFlowAction runs a flow action against test values
// through the isolated test-execution path (authoring.ApplyTestFlowAction).
// values are host-defined field names, so they travel through RawForm.
func AuthoringMutationForTestFlowAction(flowKey, actionKey string, values map[string]string) AuthoringMutation {
	mutation := AuthoringMutation{
		Kind:          AuthoringOperationTestFlowAction,
		FlowKey:       flowKey,
		FlowActionKey: actionKey,
	}.Normalize()
	mutation.RawForm = cloneStringMap(values)
	return mutation
}

func AuthoringMutationFromForm(form map[string]string) (AuthoringMutation, AuthoringValidation) {
	validation := AuthoringValidation{Values: cloneStringMap(form)}
	mutation := AuthoringMutation{
		Kind:                 AuthoringOperationKind(formValue(form, AuthoringFieldOperation)),
		IntentKey:            formValue(form, AuthoringFieldIntentKey),
		IntentKind:           core.CompositionIntentKind(formValue(form, AuthoringFieldIntentKind)),
		PageKey:              formValue(form, AuthoringFieldPageKey),
		PageLabel:            formValue(form, AuthoringFieldPageLabel),
		PageRoute:            formValue(form, AuthoringFieldPageRoute),
		PageBlueprintKey:     formValue(form, AuthoringFieldPageBlueprintKey),
		ComponentKey:         formValue(form, AuthoringFieldComponentKey),
		ComponentLabel:       formValue(form, AuthoringFieldComponentLabel),
		ComponentTemplateKey: formValue(form, AuthoringFieldComponentTemplateKey),
		ControlKey:           formValue(form, AuthoringFieldControlKey),
		ControlKind:          core.ControlKind(formValue(form, AuthoringFieldControlKind)),
		Binding:              formValue(form, AuthoringFieldBinding),
		TargetRegion:         formValue(form, AuthoringFieldTargetRegion),
		Value:                formValue(form, AuthoringFieldValue),
		StyleProperty:        formValue(form, AuthoringFieldStyleProperty),
		StyleValue:           formValue(form, AuthoringFieldStyleValue),
		Breakpoint:           formValue(form, AuthoringFieldBreakpoint),
		State:                formValue(form, AuthoringFieldState),
		OperationID:          formValue(form, AuthoringFieldOperationID),
		ExpectedTargetHead:   formValue(form, AuthoringFieldExpectedTargetHead),
		HistoryOperationID:   formValue(form, AuthoringFieldHistoryOperationID),
		InteractionKey:       formValue(form, AuthoringFieldInteractionKey),
		InteractionKind:      core.InteractionKind(formValue(form, AuthoringFieldInteractionKind)),
		InteractionEffect:    core.InteractionEffect(formValue(form, AuthoringFieldInteractionEffect)),
		FlowKey:              formValue(form, AuthoringFieldFlowKey),
		FlowActionKey:        formValue(form, AuthoringFieldFlowActionKey),
		FlowActionLabel:      formValue(form, AuthoringFieldFlowActionLabel),
		FlowFieldName:        formValue(form, AuthoringFieldFlowFieldName),
		FlowFieldLabel:       formValue(form, AuthoringFieldFlowFieldLabel),
	}
	if raw, present := form[AuthoringFieldExpectedTargetValue]; present {
		raw = strings.TrimSpace(raw)
		var expected OperationValue
		if raw == "" || raw == "null" || !strings.HasPrefix(raw, "{") || json.Unmarshal([]byte(raw), &expected) != nil {
			validation.AddFieldError(AuthoringFieldExpectedTargetValue, "Enter a valid expected target value.")
		} else {
			mutation.ExpectedTargetValue = &expected
		}
	}
	if revision, ok, valid := parseAuthoringInt(formValue(form, AuthoringFieldExpectedRevision)); ok {
		if valid && revision >= 0 {
			mutation.ExpectedRevision = uint64(revision)
		} else {
			validation.AddFieldError(AuthoringFieldExpectedRevision, "Enter a valid document revision.")
		}
	}
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
	if durationMS, ok, valid := parseAuthoringInt(formValue(form, AuthoringFieldInteractionDurationMS)); ok {
		if valid {
			mutation.InteractionDurationMS = durationMS
		} else {
			validation.AddFieldError(AuthoringFieldInteractionDurationMS, "Enter a whole-number duration.")
		}
	}
	if delayMS, ok, valid := parseAuthoringInt(formValue(form, AuthoringFieldInteractionDelayMS)); ok {
		if valid {
			mutation.InteractionDelayMS = delayMS
		} else {
			validation.AddFieldError(AuthoringFieldInteractionDelayMS, "Enter a whole-number delay.")
		}
	}
	if once, ok, valid := parseAuthoringBool(formValue(form, AuthoringFieldInteractionOnce)); ok {
		if valid {
			mutation.InteractionOnce = once
			mutation.HasInteractionOnce = true
		} else {
			validation.AddFieldError(AuthoringFieldInteractionOnce, "Use true or false.")
		}
	}
	if required, ok, valid := parseAuthoringBool(formValue(form, AuthoringFieldFlowFieldRequired)); ok {
		if valid {
			mutation.FlowFieldRequired = required
			mutation.HasFlowFieldRequired = true
		} else {
			validation.AddFieldError(AuthoringFieldFlowFieldRequired, "Use true or false.")
		}
	}
	mutation = mutation.Normalize()
	// For save-appearance and test-flow-action, populate RawForm with the
	// complete submitted form so the host adapter (or ApplyTestFlowAction) can
	// read all host-defined color/appearance or test-value field names.
	if mutation.Kind == AuthoringOperationSaveAppearance || mutation.Kind == AuthoringOperationTestFlowAction {
		mutation.RawForm = cloneStringMap(form)
	}
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
	mutation.StyleProperty = strings.ToLower(strings.TrimSpace(mutation.StyleProperty))
	mutation.StyleValue = strings.TrimSpace(mutation.StyleValue)
	mutation.Breakpoint = strings.TrimSpace(mutation.Breakpoint)
	mutation.State = strings.TrimSpace(mutation.State)
	mutation.OperationID = strings.TrimSpace(mutation.OperationID)
	mutation.ExpectedTargetHead = strings.TrimSpace(mutation.ExpectedTargetHead)
	mutation.ExpectedTargetValue = cloneOperationValue(mutation.ExpectedTargetValue)
	mutation.HistoryOperationID = strings.TrimSpace(mutation.HistoryOperationID)
	if mutation.Kind == AuthoringOperationSetStyle {
		mutation.Breakpoint = normalizeStyleBreakpoint(mutation.Breakpoint)
		mutation.State = normalizeStyleState(mutation.State)
	}
	if mutation.Position < 0 {
		mutation.Position = 0
	}
	// Interaction fields are trimmed but never silently coerced to a default
	// kind/effect here — Validate rejects anything outside the allow-list so
	// the mismatch is reported, not hidden.
	mutation.InteractionKey = strings.TrimSpace(mutation.InteractionKey)
	mutation.InteractionKind = core.InteractionKind(strings.TrimSpace(string(mutation.InteractionKind)))
	mutation.InteractionEffect = core.InteractionEffect(strings.ToLower(strings.TrimSpace(string(mutation.InteractionEffect))))
	if mutation.InteractionDurationMS < 0 {
		mutation.InteractionDurationMS = 0
	}
	if mutation.InteractionDelayMS < 0 {
		mutation.InteractionDelayMS = 0
	}
	mutation.FlowKey = strings.TrimSpace(mutation.FlowKey)
	mutation.FlowActionKey = strings.TrimSpace(mutation.FlowActionKey)
	mutation.FlowActionLabel = strings.TrimSpace(mutation.FlowActionLabel)
	mutation.FlowFieldName = strings.TrimSpace(mutation.FlowFieldName)
	mutation.FlowFieldLabel = strings.TrimSpace(mutation.FlowFieldLabel)
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
		case core.CompositionIntentCreatePage:
			if mutation.PageBlueprintKey == "" {
				validation.AddFieldError(AuthoringFieldPageBlueprintKey, "Choose a page blueprint.")
			}
		case core.CompositionIntentAddComponent:
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
	case AuthoringOperationSaveAppearance:
		// save-appearance carries multiple host-defined appearance fields
		// (color tokens, etc.). No required standard fields beyond the op kind.
		// The host adapter reads values from mutation.RawForm.
	case AuthoringOperationSetStyle:
		validateStyleMutation(&validation, mutation)
	case AuthoringOperationSetField:
		requirePageFieldTarget(&validation, mutation)
	case AuthoringOperationResetStyle:
		validateStyleTarget(&validation, mutation)
	case AuthoringOperationUndo, AuthoringOperationRedo:
		if mutation.HistoryOperationID == "" {
			validation.AddFieldError(AuthoringFieldHistoryOperationID, "Choose a history operation.")
		}
	case AuthoringOperationUpdatePage:
		if mutation.PageKey == "" {
			validation.AddFieldError(AuthoringFieldPageKey, "Choose a page.")
		}
		if mutation.PageLabel == "" && mutation.PageRoute == "" {
			validation.AddFieldError(AuthoringFieldPageLabel, "Enter a page label or route.")
		}
	case AuthoringOperationSetSharedField:
		if mutation.ComponentTemplateKey == "" {
			validation.AddFieldError(AuthoringFieldComponentTemplateKey, "Choose a shared component.")
		}
		if mutation.ControlKey == "" {
			validation.AddFieldError(AuthoringFieldControlKey, "Choose a control.")
		}
	case AuthoringOperationOverrideInstance:
		requirePageComponent(&validation, mutation)
		if mutation.ControlKey == "" {
			validation.AddFieldError(AuthoringFieldControlKey, "Choose a control.")
		}
	case AuthoringOperationDetachInstance, AuthoringOperationRestoreInstance:
		requirePageComponent(&validation, mutation)
	case AuthoringOperationPlaceInstance:
		if mutation.ComponentTemplateKey == "" {
			validation.AddFieldError(AuthoringFieldComponentTemplateKey, "Choose a shared component.")
		}
		if mutation.PageKey == "" {
			validation.AddFieldError(AuthoringFieldPageKey, "Choose a target page.")
		}
	case AuthoringOperationSetInteraction:
		validateSetInteractionMutation(&validation, mutation)
	case AuthoringOperationRemoveInteraction:
		requirePageComponent(&validation, mutation)
		if mutation.InteractionKey == "" {
			validation.AddFieldError(AuthoringFieldInteractionKey, "Choose an interaction.")
		}
	case AuthoringOperationSetFlowField:
		validateSetFlowFieldMutation(&validation, mutation)
	case AuthoringOperationRemoveFlowField:
		requireFlowAction(&validation, mutation)
		if mutation.FlowFieldName == "" {
			validation.AddFieldError(AuthoringFieldFlowFieldName, "Choose a field.")
		}
	case AuthoringOperationSetFlowAction:
		requireFlowAction(&validation, mutation)
	case AuthoringOperationTestFlowAction:
		requireFlowAction(&validation, mutation)
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
	setFormValue(values, AuthoringFieldStyleProperty, mutation.StyleProperty)
	setFormValue(values, AuthoringFieldStyleValue, mutation.StyleValue)
	setFormValue(values, AuthoringFieldBreakpoint, mutation.Breakpoint)
	setFormValue(values, AuthoringFieldState, mutation.State)
	setFormValue(values, AuthoringFieldOperationID, mutation.OperationID)
	if mutation.ExpectedRevision > 0 {
		values[AuthoringFieldExpectedRevision] = strconv.FormatUint(mutation.ExpectedRevision, 10)
	}
	setFormValue(values, AuthoringFieldExpectedTargetHead, mutation.ExpectedTargetHead)
	if mutation.ExpectedTargetValue != nil {
		if raw, err := json.Marshal(mutation.ExpectedTargetValue); err == nil {
			values[AuthoringFieldExpectedTargetValue] = string(raw)
		}
	}
	setFormValue(values, AuthoringFieldHistoryOperationID, mutation.HistoryOperationID)
	if mutation.HasPosition {
		values[AuthoringFieldPosition] = strconv.Itoa(mutation.Position)
	}
	if mutation.HasVisible {
		values[AuthoringFieldVisible] = strconv.FormatBool(mutation.Visible)
	}
	setFormValue(values, AuthoringFieldInteractionKey, mutation.InteractionKey)
	setFormValue(values, AuthoringFieldInteractionKind, string(mutation.InteractionKind))
	setFormValue(values, AuthoringFieldInteractionEffect, string(mutation.InteractionEffect))
	if mutation.InteractionDurationMS > 0 {
		values[AuthoringFieldInteractionDurationMS] = strconv.Itoa(mutation.InteractionDurationMS)
	}
	if mutation.InteractionDelayMS > 0 {
		values[AuthoringFieldInteractionDelayMS] = strconv.Itoa(mutation.InteractionDelayMS)
	}
	if mutation.HasInteractionOnce {
		values[AuthoringFieldInteractionOnce] = strconv.FormatBool(mutation.InteractionOnce)
	}
	setFormValue(values, AuthoringFieldFlowKey, mutation.FlowKey)
	setFormValue(values, AuthoringFieldFlowActionKey, mutation.FlowActionKey)
	setFormValue(values, AuthoringFieldFlowActionLabel, mutation.FlowActionLabel)
	setFormValue(values, AuthoringFieldFlowFieldName, mutation.FlowFieldName)
	setFormValue(values, AuthoringFieldFlowFieldLabel, mutation.FlowFieldLabel)
	if mutation.HasFlowFieldRequired {
		values[AuthoringFieldFlowFieldRequired] = strconv.FormatBool(mutation.FlowFieldRequired)
	}
	return values
}

func AuthoringMutationFormInputViews(mutation AuthoringMutation) []map[string]string {
	values := mutation.FormValues()
	fields := []string{
		AuthoringFieldOperation,
		AuthoringFieldIntentKey,
		AuthoringFieldIntentKind,
		AuthoringFieldPageKey,
		AuthoringFieldPageLabel,
		AuthoringFieldPageRoute,
		AuthoringFieldPageBlueprintKey,
		AuthoringFieldComponentKey,
		AuthoringFieldComponentLabel,
		AuthoringFieldComponentTemplateKey,
		AuthoringFieldControlKey,
		AuthoringFieldControlKind,
		AuthoringFieldBinding,
		AuthoringFieldTargetRegion,
		AuthoringFieldValue,
		AuthoringFieldStyleProperty,
		AuthoringFieldStyleValue,
		AuthoringFieldBreakpoint,
		AuthoringFieldState,
		AuthoringFieldPosition,
		AuthoringFieldVisible,
		AuthoringFieldOperationID,
		AuthoringFieldExpectedRevision,
		AuthoringFieldExpectedTargetHead,
		AuthoringFieldExpectedTargetValue,
		AuthoringFieldHistoryOperationID,
		AuthoringFieldInteractionKey,
		AuthoringFieldInteractionKind,
		AuthoringFieldInteractionEffect,
		AuthoringFieldInteractionDurationMS,
		AuthoringFieldInteractionDelayMS,
		AuthoringFieldInteractionOnce,
		AuthoringFieldFlowKey,
		AuthoringFieldFlowActionKey,
		AuthoringFieldFlowActionLabel,
		AuthoringFieldFlowFieldName,
		AuthoringFieldFlowFieldLabel,
		AuthoringFieldFlowFieldRequired,
	}
	out := make([]map[string]string, 0, len(values))
	for _, field := range fields {
		value, ok := values[field]
		if !ok {
			continue
		}
		out = append(out, map[string]string{
			"name":  field,
			"value": value,
		})
	}
	return out
}

func AuthoringMutationView(mutation AuthoringMutation) map[string]any {
	mutation = mutation.Normalize()
	return map[string]any{
		"kind":                  string(mutation.Kind),
		"intentKey":             mutation.IntentKey,
		"intentKind":            string(mutation.IntentKind),
		"pageKey":               mutation.PageKey,
		"pageLabel":             mutation.PageLabel,
		"pageRoute":             mutation.PageRoute,
		"pageBlueprintKey":      mutation.PageBlueprintKey,
		"componentKey":          mutation.ComponentKey,
		"componentLabel":        mutation.ComponentLabel,
		"componentTemplateKey":  mutation.ComponentTemplateKey,
		"controlKey":            mutation.ControlKey,
		"controlKind":           string(mutation.ControlKind),
		"binding":               mutation.Binding,
		"targetRegion":          mutation.TargetRegion,
		"value":                 mutation.Value,
		"styleProperty":         mutation.StyleProperty,
		"styleValue":            mutation.StyleValue,
		"breakpoint":            mutation.Breakpoint,
		"state":                 mutation.State,
		"position":              mutation.Position,
		"hasPosition":           mutation.HasPosition,
		"visible":               mutation.Visible,
		"hasVisible":            mutation.HasVisible,
		"operationID":           mutation.OperationID,
		"expectedRevision":      mutation.ExpectedRevision,
		"expectedTargetHead":    mutation.ExpectedTargetHead,
		"expectedTargetValue":   mutation.ExpectedTargetValue,
		"historyOperationID":    mutation.HistoryOperationID,
		"interactionKey":        mutation.InteractionKey,
		"interactionKind":       string(mutation.InteractionKind),
		"interactionEffect":     string(mutation.InteractionEffect),
		"interactionDurationMS": mutation.InteractionDurationMS,
		"interactionDelayMS":    mutation.InteractionDelayMS,
		"interactionOnce":       mutation.InteractionOnce,
		"hasInteractionOnce":    mutation.HasInteractionOnce,
		"flowKey":               mutation.FlowKey,
		"flowActionKey":         mutation.FlowActionKey,
		"flowActionLabel":       mutation.FlowActionLabel,
		"flowFieldName":         mutation.FlowFieldName,
		"flowFieldLabel":        mutation.FlowFieldLabel,
		"flowFieldRequired":     mutation.FlowFieldRequired,
		"hasFlowFieldRequired":  mutation.HasFlowFieldRequired,
		"formValues":            mutation.FormValues(),
		"formInputs":            AuthoringMutationFormInputViews(mutation),
	}
}

func AuthoringFieldNamesView() map[string]string {
	return map[string]string{
		"operation":             AuthoringFieldOperation,
		"intentKey":             AuthoringFieldIntentKey,
		"intentKind":            AuthoringFieldIntentKind,
		"pageKey":               AuthoringFieldPageKey,
		"pageLabel":             AuthoringFieldPageLabel,
		"pageRoute":             AuthoringFieldPageRoute,
		"pageBlueprintKey":      AuthoringFieldPageBlueprintKey,
		"componentKey":          AuthoringFieldComponentKey,
		"componentLabel":        AuthoringFieldComponentLabel,
		"componentTemplateKey":  AuthoringFieldComponentTemplateKey,
		"controlKey":            AuthoringFieldControlKey,
		"controlKind":           AuthoringFieldControlKind,
		"binding":               AuthoringFieldBinding,
		"targetRegion":          AuthoringFieldTargetRegion,
		"value":                 AuthoringFieldValue,
		"styleProperty":         AuthoringFieldStyleProperty,
		"styleValue":            AuthoringFieldStyleValue,
		"breakpoint":            AuthoringFieldBreakpoint,
		"state":                 AuthoringFieldState,
		"position":              AuthoringFieldPosition,
		"visible":               AuthoringFieldVisible,
		"operationID":           AuthoringFieldOperationID,
		"expectedRevision":      AuthoringFieldExpectedRevision,
		"expectedTargetHead":    AuthoringFieldExpectedTargetHead,
		"expectedTargetValue":   AuthoringFieldExpectedTargetValue,
		"historyOperationID":    AuthoringFieldHistoryOperationID,
		"interactionKey":        AuthoringFieldInteractionKey,
		"interactionKind":       AuthoringFieldInteractionKind,
		"interactionEffect":     AuthoringFieldInteractionEffect,
		"interactionDurationMS": AuthoringFieldInteractionDurationMS,
		"interactionDelayMS":    AuthoringFieldInteractionDelayMS,
		"interactionOnce":       AuthoringFieldInteractionOnce,
		"flowKey":               AuthoringFieldFlowKey,
		"flowActionKey":         AuthoringFieldFlowActionKey,
		"flowActionLabel":       AuthoringFieldFlowActionLabel,
		"flowFieldName":         AuthoringFieldFlowFieldName,
		"flowFieldLabel":        AuthoringFieldFlowFieldLabel,
		"flowFieldRequired":     AuthoringFieldFlowFieldRequired,
	}
}

func AuthoringRefreshFragments(selectors ...string) []AuthoringRefreshFragment {
	fragments := make([]AuthoringRefreshFragment, 0, len(selectors))
	for _, selector := range selectors {
		fragment := AuthoringRefreshFragment{Selector: selector}.Normalize()
		if fragment.Selector == "" {
			continue
		}
		fragments = append(fragments, fragment)
	}
	return fragments
}

func (fragment AuthoringRefreshFragment) Normalize() AuthoringRefreshFragment {
	fragment.Key = strings.TrimSpace(fragment.Key)
	fragment.Selector = strings.TrimSpace(fragment.Selector)
	fragment.Mode = strings.TrimSpace(fragment.Mode)
	switch fragment.Mode {
	case "", "replace":
		fragment.Mode = "replace"
	case "inner":
		fragment.Mode = "inner"
	default:
		fragment.Mode = "replace"
	}
	return fragment
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
	result.FragmentURL = strings.TrimSpace(result.FragmentURL)
	result.DraftID = strings.TrimSpace(result.DraftID)
	result.TargetHead = strings.TrimSpace(result.TargetHead)
	result.OperationID = strings.TrimSpace(result.OperationID)
	result.FieldErrors = cloneStringMap(result.FieldErrors)
	result.Values = cloneStringMap(result.Values)
	result.Changes = normalizeAuthoringChanges(result.Changes)
	result.Fragments = normalizeAuthoringRefreshFragments(result.Fragments)
	return result
}

func (result AuthoringMutationResult) HasErrors() bool {
	return len(result.FieldErrors) > 0
}

func AuthoringMutationResultView(result AuthoringMutationResult) map[string]any {
	result = result.Normalize()
	return map[string]any{
		"message":          result.Message,
		"redirectURL":      result.RedirectURL,
		"previewURL":       result.PreviewURL,
		"fragmentURL":      result.FragmentURL,
		"draftID":          result.DraftID,
		"refreshPreview":   result.RefreshPreview,
		"fieldErrors":      cloneStringMap(result.FieldErrors),
		"values":           cloneStringMap(result.Values),
		"changes":          authoringChangeViews(result.Changes),
		"changeCount":      len(result.Changes),
		"fragments":        authoringRefreshFragmentViews(result.Fragments),
		"fragmentCount":    len(result.Fragments),
		"documentRevision": result.DocumentRevision,
		"targetHead":       result.TargetHead,
		"operationID":      result.OperationID,
		"value":            result.Value,
	}
}

func normalizeAuthoringRefreshFragments(fragments []AuthoringRefreshFragment) []AuthoringRefreshFragment {
	out := make([]AuthoringRefreshFragment, 0, len(fragments))
	for _, fragment := range fragments {
		fragment = fragment.Normalize()
		if fragment.Selector == "" {
			continue
		}
		out = append(out, fragment)
	}
	return out
}

func authoringRefreshFragmentViews(fragments []AuthoringRefreshFragment) []map[string]string {
	fragments = normalizeAuthoringRefreshFragments(fragments)
	out := make([]map[string]string, 0, len(fragments))
	for _, fragment := range fragments {
		out = append(out, map[string]string{
			"key":      fragment.Key,
			"selector": fragment.Selector,
			"mode":     fragment.Mode,
		})
	}
	return out
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
	case AuthoringOperationSaveAppearance:
		return AuthoringOperationSaveAppearance
	case AuthoringOperationSetStyle:
		return AuthoringOperationSetStyle
	case AuthoringOperationSetField:
		return AuthoringOperationSetField
	case AuthoringOperationResetStyle:
		return AuthoringOperationResetStyle
	case AuthoringOperationUndo:
		return AuthoringOperationUndo
	case AuthoringOperationRedo:
		return AuthoringOperationRedo
	case AuthoringOperationSetSharedField:
		return AuthoringOperationSetSharedField
	case AuthoringOperationOverrideInstance:
		return AuthoringOperationOverrideInstance
	case AuthoringOperationDetachInstance:
		return AuthoringOperationDetachInstance
	case AuthoringOperationRestoreInstance:
		return AuthoringOperationRestoreInstance
	case AuthoringOperationSetInteraction:
		return AuthoringOperationSetInteraction
	case AuthoringOperationRemoveInteraction:
		return AuthoringOperationRemoveInteraction
	case AuthoringOperationSetFlowField:
		return AuthoringOperationSetFlowField
	case AuthoringOperationRemoveFlowField:
		return AuthoringOperationRemoveFlowField
	case AuthoringOperationSetFlowAction:
		return AuthoringOperationSetFlowAction
	case AuthoringOperationTestFlowAction:
		return AuthoringOperationTestFlowAction
	case AuthoringOperationPlaceInstance:
		return AuthoringOperationPlaceInstance
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

// requirePageFieldTarget validates the two supported SetField addressing
// shapes: an explicitly bound page-level field (page + binding), or a
// component-relative control (page + component + control). The latter keeps
// the existing component requirement for ControlKey-only writes while the
// former deliberately permits an empty ComponentKey for page-level fields.
func requirePageFieldTarget(validation *AuthoringValidation, mutation AuthoringMutation) {
	if mutation.PageKey == "" {
		validation.AddFieldError(AuthoringFieldPageKey, "Choose a target page.")
	}
	if mutation.Binding != "" {
		return
	}
	if mutation.ComponentKey == "" {
		validation.AddFieldError(AuthoringFieldComponentKey, "Choose a component.")
	}
	if mutation.ControlKey == "" {
		validation.AddFieldError(AuthoringFieldBinding, "Choose a field target.")
	}
}

// requireFlowAction validates the address shared by every flow-designer
// durable operation: a real flow key and a real action within it.
func requireFlowAction(validation *AuthoringValidation, mutation AuthoringMutation) {
	if mutation.FlowKey == "" {
		validation.AddFieldError(AuthoringFieldFlowKey, "Choose a flow.")
	}
	if mutation.FlowActionKey == "" {
		validation.AddFieldError(AuthoringFieldFlowActionKey, "Choose a flow action.")
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
