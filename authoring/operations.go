package authoring

// This file contains the host-neutral durable authoring protocol. Hosts own
// storage and authorization; Studio owns the shape, normalization and
// conflict/undo semantics. Keep these values JSON stable so persisted host
// records can outlive a Studio release.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"m31labs.dev/gosx-studio/core"
)

const OperationSchemaVersion = 1

type OperationKind string

const (
	OperationSetField   OperationKind = "set-field"
	OperationSetStyle   OperationKind = "set-style"
	OperationResetStyle OperationKind = "reset-style"
	OperationUndo       OperationKind = "undo"
	OperationRedo       OperationKind = "redo"
	// The six operations below make the reusable component/instance model
	// (core/instances.go), the no-code interaction schema (core/interactions.go),
	// and the flow designer's field/action editing (panels/flow_designer.go)
	// first-class durable operations: they carry an expected-base-revision
	// envelope, persist an OperationRecord, and get an actor-scoped
	// Undo/Redo exactly like SetField/SetStyle/ResetStyle already do. Before
	// this, all three domains only ever reported an ephemeral
	// AuthoringMutationResult.Changes summary -- no durable record, no
	// optimistic concurrency, and nothing cms/lifecycle/engine.
	// OperationChangeSet could classify (see its "currently unrepresentable"
	// caveat, now made real by these six kinds' Field values -- see the
	// Field* constants below).
	//
	// OperationSetSharedField edits one control on a shared
	// core.ComponentDefinition (Target.ComponentKey is the definition key,
	// Target.ControlKey is the control key). It has no paired "remove" --
	// like SetField, Before/After are always Present.
	OperationSetSharedField OperationKind = "set-shared-field"
	// OperationOverrideInstance sets one instance-local control override
	// (Target.PageID+ComponentKey address the placed instance,
	// Target.ControlKey the diverging control). Like SetSharedField, this is
	// a SetField-shaped singleton: Before/After are always Present. A
	// precise per-field "undo back to no override" is not represented by
	// this family alone -- see the operations_test.go/report caveat --
	// restore-instance remains the coarse "drop every override" primitive.
	OperationOverrideInstance OperationKind = "override-instance"
	// OperationDetachInstance/OperationRestoreInstance are a paired set/reset
	// family (like SetStyle/ResetStyle) tracking one placed instance's
	// attachment state (Target.PageID+ComponentKey) at
	// FieldInstancesAttachment: Detach sets the field present, Restore clears
	// it. The durable record is a bookkeeping marker for undo/redo and
	// lifecycle change-set purposes -- the actual control-value freeze/drop a
	// real detach/restore performs remains the host's own side effect (see
	// InstanceDetachWriter/InstanceRestoreWriter), applied once the host
	// accepts the operation.
	OperationDetachInstance  OperationKind = "detach-instance"
	OperationRestoreInstance OperationKind = "restore-instance"
	// OperationSetInteraction/OperationRemoveInteraction are a paired
	// set/reset family addressing one attached core.Interaction
	// (Target.PageID+ComponentKey identify the placed component,
	// Target.ControlKey the interaction key) at FieldInteractionsEntry. The
	// request Value is an InteractionSettings JSON payload (see
	// EncodeInteractionSettings/DecodeInteractionSettings); Validate builds
	// the full core.Interaction and re-runs core.Interaction.Validate, so a
	// durable record can never persist behavior outside core's allow-lists.
	OperationSetInteraction    OperationKind = "set-interaction"
	OperationRemoveInteraction OperationKind = "remove-interaction"
	// OperationSetFlowField/OperationRemoveFlowField are a paired set/reset
	// family addressing one flow action's field (Target.PageID is the flow
	// key, Target.ComponentKey the action key, Target.ControlKey the field
	// name) at FieldFlowsField. The request Value is a FlowFieldSettings
	// JSON payload (see EncodeFlowFieldSettings/DecodeFlowFieldSettings).
	OperationSetFlowField    OperationKind = "set-flow-field"
	OperationRemoveFlowField OperationKind = "remove-flow-field"
	// OperationSetFlowAction updates a flow action's label and submit
	// binding (Target.PageID is the flow key, Target.ComponentKey the action
	// key) at FieldFlowsAction. Like SetSharedField, this is a SetField-shaped
	// singleton with no paired remove. The request Value is a
	// FlowActionSettings JSON payload (see
	// EncodeFlowActionSettings/DecodeFlowActionSettings). ApplyTestFlowAction
	// is deliberately NOT part of this durable protocol: it is an isolated,
	// no-persisted-side-effect dry run (see flow_handlers.go), so it has no
	// Before/After to record and nothing to undo.
	OperationSetFlowAction OperationKind = "set-flow-action"
)

// Field values for the six operation kinds above. Each starts with the
// dot-namespaced prefix cms/lifecycle/engine.classifyOperationScope already
// recognizes ("instances."/"interactions."/"flows.") so a persisted record
// using one of these Field values classifies into the matching
// engine.DraftChangeScope. The rest of a target's addressing (which
// definition/instance/interaction/flow+action/control) lives in the
// existing PageID/ComponentKey fields plus the new ControlKey field, not in
// Field itself -- Field is deliberately a fixed per-kind-family literal (not
// a computed, client-supplied string) so OperationRequest.Validate can
// reject a mismatched Field outright, exactly the way SetStyle/ResetStyle
// already trust ComponentKey/Property/Breakpoint/State rather than a
// client-assembled locator string.
const (
	FieldInstancesSharedField = "instances.shared-field"
	FieldInstancesOverride    = "instances.override"
	FieldInstancesAttachment  = "instances.attachment"
	FieldInteractionsEntry    = "interactions.entry"
	FieldFlowsField           = "flows.field"
	FieldFlowsAction          = "flows.action"
)

type OperationTarget struct {
	Route        string `json:"route,omitempty"`
	PageID       string `json:"pageId,omitempty"`
	Field        string `json:"field,omitempty"`
	ComponentKey string `json:"componentKey,omitempty"`
	NodeID       string `json:"nodeId,omitempty"`
	Property     string `json:"property,omitempty"`
	Breakpoint   string `json:"breakpoint,omitempty"`
	State        string `json:"state,omitempty"`
	// ControlKey addresses one named sub-key within a ComponentKey/PageID
	// container: a shared/override instance's control key, an interaction's
	// key, or a flow field's name (see the OperationSetSharedField /
	// OperationOverrideInstance / OperationSetInteraction /
	// OperationSetFlowField doc comments above for which). It plays the same
	// role for those domains that Property plays for style targets.
	ControlKey string `json:"controlKey,omitempty"`
}

func (t OperationTarget) Normalize() OperationTarget {
	t.Route = normalizeRoute(t.Route)
	t.PageID = strings.TrimSpace(t.PageID)
	t.Field = strings.TrimSpace(t.Field)
	t.ComponentKey = strings.TrimSpace(t.ComponentKey)
	t.NodeID = strings.TrimSpace(t.NodeID)
	t.Property = strings.ToLower(strings.TrimSpace(t.Property))
	t.Breakpoint = strings.ToLower(strings.TrimSpace(t.Breakpoint))
	if t.Breakpoint == "" {
		t.Breakpoint = StyleBreakpointBase
	}
	t.State = strings.ToLower(strings.TrimSpace(t.State))
	if t.State == "" {
		t.State = StyleStateDefault
	}
	t.ControlKey = strings.TrimSpace(t.ControlKey)
	return t
}

func (t OperationTarget) IsStyle() bool {
	return t.Property != "" || t.ComponentKey != "" && t.Field == ""
}

// Key is deterministic and collision-safe: use canonical JSON rather than a
// concatenated DOM selector (which is ambiguous in the presence of separators).
func (t OperationTarget) Key(kind OperationKind) string {
	t = t.Normalize()
	b, _ := json.Marshal(struct {
		Target OperationTarget `json:"target"`
	}{t})
	sum := sha256.Sum256(b)
	return "target:" + hex.EncodeToString(sum[:])
}

func (t OperationTarget) String() string {
	t = t.Normalize()
	if t.Field != "" {
		return strings.TrimSuffix(t.Route+"/"+t.PageID+"/"+t.Field, "/")
	}
	return strings.TrimSuffix(t.Route+"/"+t.PageID+"/"+t.ComponentKey+"/"+t.Property+"/"+t.Breakpoint+"/"+t.State, "/")
}

type OperationValue struct {
	Present bool   `json:"present"`
	Value   string `json:"value,omitempty"`
}

func PresentValue(value string) OperationValue { return OperationValue{Present: true, Value: value} }

type OperationRequest struct {
	SchemaVersion            int             `json:"schemaVersion"`
	ID                       string          `json:"id"`
	Kind                     OperationKind   `json:"kind"`
	Target                   OperationTarget `json:"target"`
	Value                    string          `json:"value,omitempty"`
	ExpectedDocumentRevision uint64          `json:"expectedDocumentRevision"`
	ExpectedTargetHead       string          `json:"expectedTargetHead,omitempty"`
	// ExpectedTargetValue is an optional value-level optimistic precondition.
	// A nil pointer omits the check; a non-nil OperationValue preserves both
	// presence and content, including an explicitly-present empty string.
	ExpectedTargetValue *OperationValue `json:"expectedTargetValue,omitempty"`
	HistoryOperationID  string          `json:"historyOperationId,omitempty"`
}

func (r OperationRequest) Normalize() OperationRequest {
	if r.SchemaVersion == 0 {
		r.SchemaVersion = OperationSchemaVersion
	}
	r.ID = strings.TrimSpace(r.ID)
	r.Target = r.Target.Normalize()
	// Values are content; preserve meaningful leading/trailing whitespace. Host
	// validation decides whether a particular field permits it.
	r.ExpectedTargetHead = strings.TrimSpace(r.ExpectedTargetHead)
	r.ExpectedTargetValue = cloneOperationValue(r.ExpectedTargetValue)
	r.HistoryOperationID = strings.TrimSpace(r.HistoryOperationID)
	return r
}

func (r OperationRequest) Validate() error {
	r = r.Normalize()
	if r.SchemaVersion != OperationSchemaVersion {
		return fmt.Errorf("unsupported operation schema version %d", r.SchemaVersion)
	}
	if r.ID == "" {
		return errors.New("operation id is required")
	}
	switch r.Kind {
	case OperationSetField:
		if r.Target.Field == "" {
			return errors.New("field target is required")
		}
	case OperationSetStyle, OperationResetStyle:
		if r.Target.ComponentKey == "" || r.Target.Property == "" {
			return errors.New("component and property targets are required")
		}
		if !isSupportedStyleBreakpoint(r.Target.Breakpoint) || !isSupportedStyleState(r.Target.State) {
			return errors.New("unsupported style scope")
		}
	case OperationUndo, OperationRedo:
		if r.HistoryOperationID == "" {
			return errors.New("history operation id is required")
		}
	case OperationSetSharedField:
		if r.Target.ComponentKey == "" || r.Target.ControlKey == "" {
			return errors.New("shared component and control targets are required")
		}
		if r.Target.Field != FieldInstancesSharedField {
			return errors.New("set-shared-field requires the instances.shared-field target")
		}
	case OperationOverrideInstance:
		if r.Target.PageID == "" || r.Target.ComponentKey == "" || r.Target.ControlKey == "" {
			return errors.New("page, component, and control targets are required")
		}
		if r.Target.Field != FieldInstancesOverride {
			return errors.New("override-instance requires the instances.override target")
		}
	case OperationDetachInstance, OperationRestoreInstance:
		if r.Target.PageID == "" || r.Target.ComponentKey == "" {
			return errors.New("page and component targets are required")
		}
		if r.Target.Field != FieldInstancesAttachment {
			return errors.New("detach/restore instance requires the instances.attachment target")
		}
	case OperationSetInteraction:
		if r.Target.PageID == "" || r.Target.ComponentKey == "" || r.Target.ControlKey == "" {
			return errors.New("page, component, and interaction targets are required")
		}
		if r.Target.Field != FieldInteractionsEntry {
			return errors.New("set-interaction requires the interactions.entry target")
		}
		settings, err := DecodeInteractionSettings(r.Value)
		if err != nil {
			return err
		}
		if errs := settings.Interaction(r.Target).Validate(); len(errs) > 0 {
			return fmt.Errorf("invalid interaction settings: %s", firstMapValue(errs))
		}
	case OperationRemoveInteraction:
		if r.Target.PageID == "" || r.Target.ComponentKey == "" || r.Target.ControlKey == "" {
			return errors.New("page, component, and interaction targets are required")
		}
		if r.Target.Field != FieldInteractionsEntry {
			return errors.New("remove-interaction requires the interactions.entry target")
		}
	case OperationSetFlowField:
		if r.Target.PageID == "" || r.Target.ComponentKey == "" || r.Target.ControlKey == "" {
			return errors.New("flow, action, and field targets are required")
		}
		if r.Target.Field != FieldFlowsField {
			return errors.New("set-flow-field requires the flows.field target")
		}
		settings, err := DecodeFlowFieldSettings(r.Value)
		if err != nil {
			return err
		}
		if settings.Kind != "" && !isFlowFieldControlKind(settings.Kind) {
			return errors.New("unsupported flow field kind")
		}
	case OperationRemoveFlowField:
		if r.Target.PageID == "" || r.Target.ComponentKey == "" || r.Target.ControlKey == "" {
			return errors.New("flow, action, and field targets are required")
		}
		if r.Target.Field != FieldFlowsField {
			return errors.New("remove-flow-field requires the flows.field target")
		}
	case OperationSetFlowAction:
		if r.Target.PageID == "" || r.Target.ComponentKey == "" {
			return errors.New("flow and action targets are required")
		}
		if r.Target.Field != FieldFlowsAction {
			return errors.New("set-flow-action requires the flows.action target")
		}
		if _, err := DecodeFlowActionSettings(r.Value); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported operation kind %q", r.Kind)
	}
	return nil
}

// firstMapValue returns an arbitrary (but deterministic-enough for a single
// error message) value from a field-error map, or "" for an empty map. Go
// map iteration order is unspecified, but Validate callers only need *a*
// concrete problem to report, not every one -- core.Interaction.Validate
// already accumulates every problem in the map for callers that want them
// all.
func firstMapValue(errs map[string]string) string {
	for _, message := range errs {
		return message
	}
	return ""
}

type OperationRecord struct {
	SchemaVersion      int             `json:"schemaVersion"`
	ID                 string          `json:"id"`
	ActorID            string          `json:"actorId"`
	ActorLabel         string          `json:"actorLabel,omitempty"`
	Kind               OperationKind   `json:"kind"`
	Target             OperationTarget `json:"target"`
	TargetKey          string          `json:"targetKey"`
	Before             OperationValue  `json:"before"`
	After              OperationValue  `json:"after"`
	DocumentRevision   uint64          `json:"documentRevision"`
	PreviousTargetHead string          `json:"previousTargetHead,omitempty"`
	TargetHead         string          `json:"targetHead"`
	UndoOf             string          `json:"undoOf,omitempty"`
	RedoOf             string          `json:"redoOf,omitempty"`
	Created            time.Time       `json:"created"`
}

func (r OperationRecord) Normalize() OperationRecord {
	if r.SchemaVersion == 0 {
		r.SchemaVersion = OperationSchemaVersion
	}
	r.ID = strings.TrimSpace(r.ID)
	r.ActorID = strings.TrimSpace(r.ActorID)
	r.ActorLabel = strings.TrimSpace(r.ActorLabel)
	r.Target = r.Target.Normalize()
	if r.TargetKey == "" {
		r.TargetKey = r.Target.Key(r.Kind)
	}
	r.PreviousTargetHead = strings.TrimSpace(r.PreviousTargetHead)
	r.TargetHead = strings.TrimSpace(r.TargetHead)
	r.UndoOf = strings.TrimSpace(r.UndoOf)
	r.RedoOf = strings.TrimSpace(r.RedoOf)
	return r
}

func (r OperationRecord) InverseRequest(id, actor string, revision uint64) OperationRequest {
	kind := SetKindForTarget(r.Target, r.Before.Present)
	return OperationRequest{SchemaVersion: OperationSchemaVersion, ID: id, Kind: kind, Target: r.Target, Value: r.Before.Value, ExpectedDocumentRevision: revision, ExpectedTargetHead: r.TargetHead, HistoryOperationID: r.ID}.Normalize()
}

func (r OperationRecord) CanonicalRequest() OperationRequest {
	kind := r.Kind
	if kind == OperationUndo || kind == OperationRedo {
		kind = SetKindForTarget(r.Target, r.After.Present)
	}
	return OperationRequest{SchemaVersion: r.SchemaVersion, ID: r.ID, Kind: kind, Target: r.Target, Value: r.After.Value}.Normalize()
}

// SetKindForTarget picks the operation kind that would write present's value
// to target's field-head slot. Target's own shape identifies which durable
// family it belongs to (style, instance attachment, interaction, flow
// field/action, or plain content field); present then selects between a
// family's "set" and "clear" member for the families that have both
// (Style/ResetStyle, Detach/RestoreInstance, Set/RemoveInteraction,
// Set/RemoveFlowField). The three singleton families (SetSharedField,
// OverrideInstance, SetFlowAction) ignore present and always return their
// one member, exactly like SetField.
//
// InverseRequest, CanonicalRequest, and cms/studio/collab/sqlstore's
// Undo/Redo replay (resolveHistory) all call this one function so a target's
// family classification can never drift between the three call sites the
// way it could if each reimplemented the same target-shape switch.
func SetKindForTarget(target OperationTarget, present bool) OperationKind {
	target = target.Normalize()
	switch {
	case target.IsStyle():
		if present {
			return OperationSetStyle
		}
		return OperationResetStyle
	case target.Field == FieldInstancesAttachment:
		if present {
			return OperationDetachInstance
		}
		return OperationRestoreInstance
	case target.Field == FieldInstancesSharedField:
		return OperationSetSharedField
	case target.Field == FieldInstancesOverride:
		return OperationOverrideInstance
	case target.Field == FieldInteractionsEntry:
		if present {
			return OperationSetInteraction
		}
		return OperationRemoveInteraction
	case target.Field == FieldFlowsField:
		if present {
			return OperationSetFlowField
		}
		return OperationRemoveFlowField
	case target.Field == FieldFlowsAction:
		return OperationSetFlowAction
	default:
		return OperationSetField
	}
}

// ClearsTarget reports whether kind is the "clear" member of a paired
// set/reset field-head family (ResetStyle, RestoreInstance,
// RemoveInteraction, RemoveFlowField): a durable record for one of these
// kinds always has an absent After value, mirroring how ResetStyle already
// behaves. cms/studio/collab/sqlstore's Apply uses this to decide whether an
// accepted operation's After value (and, for a rejected stale-head
// SubmittedValue) should be recorded as present or absent.
func ClearsTarget(kind OperationKind) bool {
	switch kind {
	case OperationResetStyle, OperationRestoreInstance, OperationRemoveInteraction, OperationRemoveFlowField:
		return true
	default:
		return false
	}
}

func RequestsEqual(a, b OperationRequest) bool {
	a, b = a.Normalize(), b.Normalize()
	a.ID = ""
	b.ID = ""
	a.ExpectedDocumentRevision = 0
	b.ExpectedDocumentRevision = 0
	a.ExpectedTargetHead, b.ExpectedTargetHead = "", ""
	a.ExpectedTargetValue, b.ExpectedTargetValue = nil, nil
	return canonicalJSON(a) == canonicalJSON(b)
}

func canonicalJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func cloneOperationValue(value *OperationValue) *OperationValue {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func normalizeRoute(route string) string {
	route = strings.TrimSpace(route)
	if route == "" {
		return "/"
	}
	if parsed, err := url.Parse(route); err == nil && parsed.Path != "" {
		route = parsed.Path
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	route = strings.TrimRight(route, "/")
	if route == "" {
		return "/"
	}
	return route
}

// --- Typed payloads for the interaction/flow durable operations ---
//
// OperationRequest.Value stays a single string for every kind (JSON-stable,
// like every other field in this file) -- SetInteraction/SetFlowField/
// SetFlowAction encode their structured settings into that string as JSON.
// The addressing identity (which interaction/field/action) lives on
// OperationTarget, not in these payloads, so encoding one never duplicates
// (and can never disagree with) the target that names it.

// InteractionSettings is SetInteraction's typed payload: everything about one
// core.Interaction except its identity (Key/Target), which the operation's
// OperationTarget already carries (ControlKey is the interaction key,
// PageID/ComponentKey is the attached component).
type InteractionSettings struct {
	Kind       core.InteractionKind   `json:"kind"`
	Effect     core.InteractionEffect `json:"effect"`
	DurationMS int                    `json:"durationMs"`
	DelayMS    int                    `json:"delayMs"`
	Once       bool                   `json:"once"`
}

// Interaction rebuilds the full core.Interaction settings describes,
// resolving its identity from target (ControlKey as Key, PageID/ComponentKey
// as the attached CanvasIdentity). The result is normalized but not
// validated -- call .Validate() on it (Validate does, for OperationRequest.
// Validate's own OperationSetInteraction case).
func (settings InteractionSettings) Interaction(target OperationTarget) core.Interaction {
	target = target.Normalize()
	return core.Interaction{
		Key:        target.ControlKey,
		Kind:       settings.Kind,
		Target:     core.CanvasIdentity{Route: target.Route, PageID: target.PageID, BlockKey: target.ComponentKey},
		Effect:     settings.Effect,
		DurationMS: settings.DurationMS,
		DelayMS:    settings.DelayMS,
		Once:       settings.Once,
	}.Normalize()
}

// EncodeInteractionSettings produces the OperationRequest.Value a
// SetInteraction operation carries for interaction (its identity is dropped;
// the caller is expected to set Target.ControlKey/PageID/ComponentKey
// separately).
func EncodeInteractionSettings(interaction core.Interaction) string {
	interaction = interaction.Normalize()
	raw, _ := json.Marshal(InteractionSettings{
		Kind:       interaction.Kind,
		Effect:     interaction.Effect,
		DurationMS: interaction.DurationMS,
		DelayMS:    interaction.DelayMS,
		Once:       interaction.Once,
	})
	return string(raw)
}

// DecodeInteractionSettings parses a SetInteraction operation's Value. An
// empty or malformed value is always an error -- unlike a plain content
// field, there is no meaningful "empty interaction settings".
func DecodeInteractionSettings(value string) (InteractionSettings, error) {
	var settings InteractionSettings
	if strings.TrimSpace(value) == "" {
		return settings, errors.New("interaction settings are required")
	}
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return InteractionSettings{}, fmt.Errorf("malformed interaction settings: %w", err)
	}
	return settings, nil
}

// FlowFieldSettings is SetFlowField's typed payload: a core.FlowField minus
// its Name, which the operation's OperationTarget.ControlKey already carries.
type FlowFieldSettings struct {
	Label    string           `json:"label"`
	Kind     core.ControlKind `json:"kind"`
	Required bool             `json:"required"`
}

// FlowField rebuilds the full core.FlowField settings describes, taking Name
// from target.ControlKey (falling back to it for Label when Label is blank,
// matching ApplySetFlowField's own fallback).
func (settings FlowFieldSettings) FlowField(target OperationTarget) core.FlowField {
	target = target.Normalize()
	label := strings.TrimSpace(settings.Label)
	if label == "" {
		label = target.ControlKey
	}
	return core.FlowField{Name: target.ControlKey, Label: label, Kind: settings.Kind, Required: settings.Required}
}

// EncodeFlowFieldSettings produces the OperationRequest.Value a SetFlowField
// operation carries for field (its Name is dropped; the caller sets
// Target.ControlKey to field.Name separately).
func EncodeFlowFieldSettings(field core.FlowField) string {
	raw, _ := json.Marshal(FlowFieldSettings{Label: field.Label, Kind: field.Kind, Required: field.Required})
	return string(raw)
}

// DecodeFlowFieldSettings parses a SetFlowField operation's Value.
func DecodeFlowFieldSettings(value string) (FlowFieldSettings, error) {
	var settings FlowFieldSettings
	if strings.TrimSpace(value) == "" {
		return settings, errors.New("flow field settings are required")
	}
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return FlowFieldSettings{}, fmt.Errorf("malformed flow field settings: %w", err)
	}
	return settings, nil
}

// FlowActionSettings is SetFlowAction's typed payload: a flow action's label
// and submit binding (handler ref). An empty HandlerRef is legal -- readiness,
// not this operation, blocks publish until one is connected (see
// ApplySetFlowAction).
type FlowActionSettings struct {
	Label      string `json:"label"`
	HandlerRef string `json:"handlerRef"`
}

// EncodeFlowActionSettings produces the OperationRequest.Value a
// SetFlowAction operation carries.
func EncodeFlowActionSettings(label, handlerRef string) string {
	raw, _ := json.Marshal(FlowActionSettings{Label: label, HandlerRef: handlerRef})
	return string(raw)
}

// DecodeFlowActionSettings parses a SetFlowAction operation's Value.
func DecodeFlowActionSettings(value string) (FlowActionSettings, error) {
	var settings FlowActionSettings
	if strings.TrimSpace(value) == "" {
		return settings, errors.New("flow action settings are required")
	}
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return FlowActionSettings{}, fmt.Errorf("malformed flow action settings: %w", err)
	}
	return settings, nil
}
