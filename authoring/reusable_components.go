package authoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"m31labs.dev/gosx-studio/core"
)

type ReusableOperationKind string

const (
	ReusableCreateDefinition    ReusableOperationKind = "create-definition"
	ReusableCreateInstance      ReusableOperationKind = "create-instance"
	ReusableUpdateDefinition    ReusableOperationKind = "update-definition"
	ReusableSetInstanceOverride ReusableOperationKind = "set-instance-override"
	ReusableClearOverride       ReusableOperationKind = "clear-override"
	ReusableDetachInstance      ReusableOperationKind = "detach-instance"
	ReusableRestoreInstance     ReusableOperationKind = "restore-instance"
	ReusableRevertOperation     ReusableOperationKind = "revert-operation"
)

const (
	ReusableFieldOperation     = "gosx_studio_reusable_operation"
	ReusableFieldOperationID   = "gosx_studio_reusable_operation_id"
	ReusableFieldDefinitionID  = "gosx_studio_reusable_definition_id"
	ReusableFieldInstanceID    = "gosx_studio_reusable_instance_id"
	ReusableFieldPageKey       = "gosx_studio_reusable_page_key"
	ReusableFieldRegion        = "gosx_studio_reusable_region"
	ReusableFieldPosition      = "gosx_studio_reusable_position"
	ReusableFieldOverrideKey   = "gosx_studio_reusable_override_key"
	ReusableFieldOverrideValue = "gosx_studio_reusable_override_value"
	ReusableFieldExpectedHead  = "gosx_studio_reusable_expected_head"
)

var (
	ErrReusableConflict     = errors.New("reusable component target is stale")
	ErrReusableUnauthorized = errors.New("reusable component operation is not authorized")
	ErrReusableIdempotency  = errors.New("reusable component operation id was reused")
	ErrReusableNotFound     = errors.New("reusable component target not found")
)

// ReusableState is host-persistable durable state. Hosts store the returned
// state atomically; Studio owns validation, optimistic heads, propagation, and
// actor-attributed history semantics.
type ReusableState struct {
	SchemaVersion int                                 `json:"schemaVersion"`
	Revision      uint64                              `json:"revision"`
	Definitions   map[string]core.ComponentDefinition `json:"definitions,omitempty"`
	Instances     map[string]core.ComponentInstance   `json:"instances,omitempty"`
	Heads         map[string]string                   `json:"heads,omitempty"`
	Operations    []ReusableOperationRecord           `json:"operations,omitempty"`
}

type ReusableOperation struct {
	SchemaVersion int                       `json:"schemaVersion"`
	ID            string                    `json:"id"`
	Kind          ReusableOperationKind     `json:"kind"`
	Definition    *core.ComponentDefinition `json:"definition,omitempty"`
	Instance      *core.ComponentInstance   `json:"instance,omitempty"`
	DefinitionID  string                    `json:"definitionId,omitempty"`
	InstanceID    string                    `json:"instanceId,omitempty"`
	OverrideKey   string                    `json:"overrideKey,omitempty"`
	Override      core.ExplicitOverride     `json:"override,omitempty"`
	ExpectedHead  string                    `json:"expectedHead,omitempty"`
	HistoryID     string                    `json:"historyId,omitempty"`
}

type ReusableTargetSnapshot struct {
	DefinitionID string                            `json:"definitionId,omitempty"`
	Definition   *core.ComponentDefinition         `json:"definition,omitempty"`
	InstanceID   string                            `json:"instanceId,omitempty"`
	Instance     *core.ComponentInstance           `json:"instance,omitempty"`
	Instances    map[string]core.ComponentInstance `json:"instances,omitempty"`
}

type ReusableOperationRecord struct {
	SchemaVersion int                    `json:"schemaVersion"`
	ID            string                 `json:"id"`
	ActorID       string                 `json:"actorId"`
	ActorLabel    string                 `json:"actorLabel,omitempty"`
	Kind          ReusableOperationKind  `json:"kind"`
	TargetKey     string                 `json:"targetKey"`
	RequestHash   string                 `json:"requestHash"`
	Before        ReusableTargetSnapshot `json:"before"`
	After         ReusableTargetSnapshot `json:"after"`
	Revision      uint64                 `json:"revision"`
	PreviousHead  string                 `json:"previousHead,omitempty"`
	TargetHead    string                 `json:"targetHead"`
	UndoOf        string                 `json:"undoOf,omitempty"`
	AffectedIDs   []string               `json:"affectedInstanceIds,omitempty"`
}

type ReusableApplyOptions struct {
	Templates                []core.ComponentTemplate
	Regions                  []string
	ActorID                  string
	ActorLabel               string
	AllowDefinitionMutations bool
}

type ReusableOperationResult struct {
	State      ReusableState
	Record     ReusableOperationRecord
	Definition *core.ComponentDefinition
	Instance   *core.ComponentInstance
}

// ReusableStateStore is the host durability seam. SaveReusableState must use
// expectedRevision as a compare-and-swap guard and persist state plus history
// atomically; this prevents two app processes from accepting the same head.
type ReusableStateStore interface {
	LoadReusableState(context.Context) (ReusableState, error)
	SaveReusableState(context.Context, ReusableState, uint64) error
}

func CommitReusableOperation(ctx context.Context, store ReusableStateStore, operation ReusableOperation, options ReusableApplyOptions) (ReusableOperationResult, error) {
	if store == nil {
		return ReusableOperationResult{}, fmt.Errorf("%w: reusable state store is required", core.ErrReusableInvalid)
	}
	state, err := store.LoadReusableState(ctx)
	if err != nil {
		return ReusableOperationResult{}, err
	}
	expectedRevision := state.Revision
	result, err := ApplyReusableOperation(state, operation, options)
	if err != nil {
		return ReusableOperationResult{}, err
	}
	if err := store.SaveReusableState(ctx, result.State, expectedRevision); err != nil {
		return ReusableOperationResult{}, err
	}
	return result, nil
}

func CommitReusableInverse(ctx context.Context, store ReusableStateStore, historyID, operationID, expectedHead string, options ReusableApplyOptions) (ReusableOperationResult, error) {
	if store == nil {
		return ReusableOperationResult{}, fmt.Errorf("%w: reusable state store is required", core.ErrReusableInvalid)
	}
	state, err := store.LoadReusableState(ctx)
	if err != nil {
		return ReusableOperationResult{}, err
	}
	expectedRevision := state.Revision
	result, err := ApplyReusableInverse(state, historyID, operationID, expectedHead, options)
	if err != nil {
		return ReusableOperationResult{}, err
	}
	if err := store.SaveReusableState(ctx, result.State, expectedRevision); err != nil {
		return ReusableOperationResult{}, err
	}
	return result, nil
}

// ReusableOperationFromForm parses the finite instance-operation vocabulary
// emitted by the shared panel. Definition create/update carry typed Go values
// through host APIs and are intentionally not decoded from client JSON.
func ReusableOperationFromForm(form map[string]string) (ReusableOperation, error) {
	kind := ReusableOperationKind(strings.TrimSpace(form[ReusableFieldOperation]))
	operation := ReusableOperation{
		SchemaVersion: 1,
		ID:            strings.TrimSpace(form[ReusableFieldOperationID]),
		Kind:          kind,
		DefinitionID:  strings.TrimSpace(form[ReusableFieldDefinitionID]),
		InstanceID:    strings.TrimSpace(form[ReusableFieldInstanceID]),
		OverrideKey:   strings.TrimSpace(form[ReusableFieldOverrideKey]),
		ExpectedHead:  strings.TrimSpace(form[ReusableFieldExpectedHead]),
	}
	switch kind {
	case ReusableCreateInstance:
		position, err := strconv.Atoi(strings.TrimSpace(form[ReusableFieldPosition]))
		if err != nil || position < 0 {
			return ReusableOperation{}, fmt.Errorf("%w: valid instance position is required", core.ErrReusableInvalid)
		}
		operation.Instance = &core.ComponentInstance{ID: operation.InstanceID, DefinitionID: operation.DefinitionID, PageKey: strings.TrimSpace(form[ReusableFieldPageKey]), Region: strings.TrimSpace(form[ReusableFieldRegion]), Position: position}
	case ReusableSetInstanceOverride:
		operation.Override = core.ExplicitOverride{Present: true, Value: form[ReusableFieldOverrideValue]}
	case ReusableClearOverride, ReusableDetachInstance, ReusableRestoreInstance:
	case ReusableCreateDefinition, ReusableUpdateDefinition:
		return ReusableOperation{}, fmt.Errorf("%w: definition payloads must use the typed server API", core.ErrReusableInvalid)
	default:
		return ReusableOperation{}, fmt.Errorf("%w: unsupported reusable operation", core.ErrReusableInvalid)
	}
	operation = normalizeReusableOperation(operation)
	if operation.ID == "" {
		return ReusableOperation{}, fmt.Errorf("%w: operation id is required", core.ErrReusableInvalid)
	}
	return operation, nil
}

func ApplyReusableOperation(state ReusableState, operation ReusableOperation, options ReusableApplyOptions) (ReusableOperationResult, error) {
	state = cloneReusableState(state)
	state.ensure()
	operation = normalizeReusableOperation(operation)
	options.ActorID = strings.TrimSpace(options.ActorID)
	options.ActorLabel = strings.TrimSpace(options.ActorLabel)
	if options.ActorID == "" {
		return ReusableOperationResult{}, ErrReusableUnauthorized
	}
	if operation.ID == "" || !supportedReusableOperation(operation.Kind) {
		return ReusableOperationResult{}, fmt.Errorf("%w: operation id and kind are required", core.ErrReusableInvalid)
	}
	if (operation.Kind == ReusableCreateDefinition || operation.Kind == ReusableUpdateDefinition) && !options.AllowDefinitionMutations {
		return ReusableOperationResult{}, ErrReusableUnauthorized
	}
	requestHash := reusableRequestHash(operation)
	for _, record := range state.Operations {
		if record.ActorID == options.ActorID && record.ID == operation.ID {
			if record.RequestHash != requestHash {
				return ReusableOperationResult{}, ErrReusableIdempotency
			}
			return reusableResult(state, record), nil
		}
	}
	targetKey, err := reusableTargetKey(operation)
	if err != nil {
		return ReusableOperationResult{}, err
	}
	currentHead := state.Heads[targetKey]
	if operation.ExpectedHead != "" && operation.ExpectedHead != currentHead {
		return ReusableOperationResult{}, ErrReusableConflict
	}
	before := state.snapshot(operation)
	affected, undoOf, err := state.apply(operation, options)
	if err != nil {
		return ReusableOperationResult{}, err
	}
	state.Revision++
	head := reusableHead(operation.ID, state.Revision, targetKey)
	state.Heads[targetKey] = head
	for _, instanceID := range affected {
		instance := state.Instances[instanceID]
		instance.HeadRevision = state.Revision
		state.Instances[instanceID] = instance
		state.Heads[reusableInstanceTarget(instanceID)] = reusableHead(operation.ID, state.Revision, reusableInstanceTarget(instanceID))
	}
	after := state.snapshot(operation)
	record := ReusableOperationRecord{SchemaVersion: 1, ID: operation.ID, ActorID: options.ActorID, ActorLabel: options.ActorLabel, Kind: operation.Kind, TargetKey: targetKey, RequestHash: requestHash, Before: before, After: after, Revision: state.Revision, PreviousHead: currentHead, TargetHead: head, UndoOf: undoOf, AffectedIDs: append([]string(nil), affected...)}
	state.Operations = append(state.Operations, record)
	return reusableResult(state, record), nil
}

// ApplyReusableInverse applies an actor-scoped compensating operation. The
// original operation remains immutable in history; the inverse receives its
// own actor, revision, optimistic head, and operation record.
func ApplyReusableInverse(state ReusableState, historyID, operationID, expectedHead string, options ReusableApplyOptions) (ReusableOperationResult, error) {
	state = cloneReusableState(state)
	state.ensure()
	actorID := strings.TrimSpace(options.ActorID)
	var source *ReusableOperationRecord
	for index := range state.Operations {
		if state.Operations[index].ID == strings.TrimSpace(historyID) {
			source = &state.Operations[index]
			break
		}
	}
	if source == nil {
		return ReusableOperationResult{}, ErrReusableNotFound
	}
	if actorID == "" || source.ActorID != actorID {
		return ReusableOperationResult{}, ErrReusableUnauthorized
	}
	if strings.HasPrefix(source.TargetKey, "definition:") && !options.AllowDefinitionMutations {
		return ReusableOperationResult{}, ErrReusableUnauthorized
	}
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return ReusableOperationResult{}, fmt.Errorf("%w: inverse operation id is required", core.ErrReusableInvalid)
	}
	requestHash := reusableHashString(operationID + ":" + historyID)
	for _, record := range state.Operations {
		if record.ActorID == actorID && record.ID == operationID {
			if record.Kind != ReusableRevertOperation || record.UndoOf != source.ID || record.RequestHash != requestHash {
				return ReusableOperationResult{}, ErrReusableIdempotency
			}
			return reusableResult(state, record), nil
		}
	}
	if expectedHead == "" || state.Heads[source.TargetKey] != expectedHead || expectedHead != source.TargetHead {
		return ReusableOperationResult{}, ErrReusableConflict
	}
	before := state.snapshotFromTarget(source.TargetKey)
	if err := state.restoreSnapshot(source.Before, source.TargetKey); err != nil {
		return ReusableOperationResult{}, err
	}
	state.Revision++
	head := reusableHead(operationID, state.Revision, source.TargetKey)
	state.Heads[source.TargetKey] = head
	affected := snapshotInstanceIDs(source.Before)
	for _, instanceID := range affected {
		instance := state.Instances[instanceID]
		instance.HeadRevision = state.Revision
		state.Instances[instanceID] = instance
		state.Heads[reusableInstanceTarget(instanceID)] = reusableHead(operationID, state.Revision, reusableInstanceTarget(instanceID))
	}
	after := state.snapshotFromTarget(source.TargetKey)
	record := ReusableOperationRecord{SchemaVersion: 1, ID: operationID, ActorID: actorID, ActorLabel: strings.TrimSpace(options.ActorLabel), Kind: ReusableRevertOperation, TargetKey: source.TargetKey, RequestHash: requestHash, Before: before, After: after, Revision: state.Revision, PreviousHead: expectedHead, TargetHead: head, UndoOf: source.ID, AffectedIDs: affected}
	state.Operations = append(state.Operations, record)
	return reusableResult(state, record), nil
}

func (state *ReusableState) apply(operation ReusableOperation, options ReusableApplyOptions) ([]string, string, error) {
	switch operation.Kind {
	case ReusableCreateDefinition:
		if operation.Definition == nil {
			return nil, "", fmt.Errorf("%w: definition is required", core.ErrReusableInvalid)
		}
		definition := operation.Definition.Normalize()
		if _, exists := state.Definitions[definition.ID]; exists {
			return nil, "", ErrReusableConflict
		}
		if err := definition.Validate(options.Templates); err != nil {
			return nil, "", err
		}
		if err := validateReusableDefinitionValues(definition); err != nil {
			return nil, "", err
		}
		if definition.Version == 0 {
			definition.Version = 1
		}
		definition.UpdatedRevision = state.Revision + 1
		state.Definitions[definition.ID] = definition
		return nil, "", nil
	case ReusableUpdateDefinition:
		if operation.Definition == nil {
			return nil, "", fmt.Errorf("%w: definition is required", core.ErrReusableInvalid)
		}
		definition := operation.Definition.Normalize()
		current, exists := state.Definitions[definition.ID]
		if !exists {
			return nil, "", ErrReusableNotFound
		}
		if definition.Key != current.Key {
			return nil, "", fmt.Errorf("%w: definition key is immutable", core.ErrReusableInvalid)
		}
		if err := definition.Validate(options.Templates); err != nil {
			return nil, "", err
		}
		if err := validateReusableDefinitionValues(definition); err != nil {
			return nil, "", err
		}
		definition.Version = current.Version + 1
		definition.UpdatedRevision = state.Revision + 1
		state.Definitions[definition.ID] = definition
		ids := state.attachedInstanceIDs(definition.ID)
		for _, id := range ids {
			instance := state.Instances[id]
			instance.DefinitionVersion = definition.Version
			state.Instances[id] = instance
		}
		return ids, "", nil
	case ReusableCreateInstance:
		if operation.Instance == nil {
			return nil, "", fmt.Errorf("%w: instance is required", core.ErrReusableInvalid)
		}
		instance := operation.Instance.Normalize()
		if _, exists := state.Instances[instance.ID]; exists {
			return nil, "", ErrReusableConflict
		}
		definition, ok := state.Definitions[instance.DefinitionID]
		if !ok {
			return nil, "", ErrReusableNotFound
		}
		instance.DefinitionVersion = definition.Version
		instance.Detached = false
		instance.Materialized = nil
		if err := instance.Validate(state.Definitions, options.Regions); err != nil {
			return nil, "", err
		}
		instance.HeadRevision = state.Revision + 1
		state.Instances[instance.ID] = instance
		return []string{instance.ID}, "", nil
	case ReusableSetInstanceOverride:
		instance, ok := state.Instances[operation.InstanceID]
		if !ok {
			return nil, "", ErrReusableNotFound
		}
		if !operation.Override.Present || !core.ValidInstanceOverrideKey(operation.OverrideKey) {
			return nil, "", fmt.Errorf("%w: explicit valid override is required", core.ErrReusableInvalid)
		}
		if err := validateReusableOverrideValue(operation.OverrideKey, operation.Override.Value); err != nil {
			return nil, "", err
		}
		if instance.Overrides == nil {
			instance.Overrides = map[string]core.ExplicitOverride{}
		}
		instance.Overrides[operation.OverrideKey] = operation.Override
		state.Instances[instance.ID] = instance
		return []string{instance.ID}, "", nil
	case ReusableClearOverride:
		instance, ok := state.Instances[operation.InstanceID]
		if !ok {
			return nil, "", ErrReusableNotFound
		}
		if !core.ValidInstanceOverrideKey(operation.OverrideKey) {
			return nil, "", fmt.Errorf("%w: valid override key is required", core.ErrReusableInvalid)
		}
		delete(instance.Overrides, operation.OverrideKey)
		state.Instances[instance.ID] = instance
		return []string{instance.ID}, "", nil
	case ReusableDetachInstance:
		instance, ok := state.Instances[operation.InstanceID]
		if !ok {
			return nil, "", ErrReusableNotFound
		}
		if instance.Detached {
			return nil, "", fmt.Errorf("%w: instance is already detached", core.ErrReusableInvalid)
		}
		resolved, err := core.ResolveComponentInstance(instance, state.Definitions)
		if err != nil {
			return nil, "", err
		}
		instance.Detached = true
		instance.Materialized = &resolved
		state.Instances[instance.ID] = instance
		return []string{instance.ID}, "", nil
	case ReusableRestoreInstance:
		instance, ok := state.Instances[operation.InstanceID]
		if !ok {
			return nil, "", ErrReusableNotFound
		}
		if !instance.Detached {
			return nil, "", fmt.Errorf("%w: instance is already attached", core.ErrReusableInvalid)
		}
		definition, ok := state.Definitions[instance.DefinitionID]
		if !ok {
			return nil, "", ErrReusableNotFound
		}
		instance.Detached = false
		instance.Materialized = nil
		instance.DefinitionVersion = definition.Version
		state.Instances[instance.ID] = instance
		return []string{instance.ID}, "", nil
	}
	return nil, "", fmt.Errorf("%w: unsupported reusable operation", core.ErrReusableInvalid)
}

func (state *ReusableState) ensure() {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	if state.Definitions == nil {
		state.Definitions = map[string]core.ComponentDefinition{}
	}
	if state.Instances == nil {
		state.Instances = map[string]core.ComponentInstance{}
	}
	if state.Heads == nil {
		state.Heads = map[string]string{}
	}
}

func (state ReusableState) snapshot(operation ReusableOperation) ReusableTargetSnapshot {
	target, _ := reusableTargetKey(operation)
	return state.snapshotFromTarget(target)
}

func (state ReusableState) snapshotFromTarget(target string) ReusableTargetSnapshot {
	snapshot := ReusableTargetSnapshot{}
	if strings.HasPrefix(target, "definition:") {
		id := strings.TrimPrefix(target, "definition:")
		snapshot.DefinitionID = id
		if definition, ok := state.Definitions[id]; ok {
			copy := definition.Normalize()
			snapshot.Definition = &copy
		}
		snapshot.Instances = map[string]core.ComponentInstance{}
		for instanceID, instance := range state.Instances {
			if instance.DefinitionID == id {
				snapshot.Instances[instanceID] = instance.Normalize()
			}
		}
		return snapshot
	}
	id := strings.TrimPrefix(target, "instance:")
	snapshot.InstanceID = id
	if instance, ok := state.Instances[id]; ok {
		copy := instance.Normalize()
		snapshot.Instance = &copy
	}
	return snapshot
}

func (state *ReusableState) restoreSnapshot(snapshot ReusableTargetSnapshot, target string) error {
	if strings.HasPrefix(target, "definition:") {
		id := strings.TrimPrefix(target, "definition:")
		if snapshot.Definition == nil {
			if len(state.attachedInstanceIDs(id)) > 0 {
				return ErrReusableConflict
			}
			delete(state.Definitions, id)
			return nil
		}
		definition := snapshot.Definition.Normalize()
		if current, ok := state.Definitions[id]; ok {
			definition.Version = current.Version + 1
		}
		definition.UpdatedRevision = state.Revision + 1
		state.Definitions[id] = definition
		for instanceID, instance := range state.Instances {
			if instance.DefinitionID == id && !instance.Detached {
				instance.DefinitionVersion = definition.Version
				state.Instances[instanceID] = instance
			}
		}
		return nil
	}
	id := strings.TrimPrefix(target, "instance:")
	if snapshot.Instance == nil {
		delete(state.Instances, id)
		return nil
	}
	state.Instances[id] = snapshot.Instance.Normalize()
	return nil
}

func (state ReusableState) attachedInstanceIDs(definitionID string) []string {
	ids := []string{}
	for id, instance := range state.Instances {
		if instance.DefinitionID == definitionID && !instance.Detached {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func reusableTargetKey(operation ReusableOperation) (string, error) {
	switch operation.Kind {
	case ReusableCreateDefinition, ReusableUpdateDefinition:
		id := operation.DefinitionID
		if operation.Definition != nil {
			id = operation.Definition.ID
		}
		id = strings.TrimSpace(id)
		if id == "" {
			return "", fmt.Errorf("%w: definition id is required", core.ErrReusableInvalid)
		}
		return "definition:" + id, nil
	case ReusableCreateInstance:
		id := operation.InstanceID
		if operation.Instance != nil {
			id = operation.Instance.ID
		}
		id = strings.TrimSpace(id)
		if id == "" {
			return "", fmt.Errorf("%w: instance id is required", core.ErrReusableInvalid)
		}
		return reusableInstanceTarget(id), nil
	case ReusableSetInstanceOverride, ReusableClearOverride, ReusableDetachInstance, ReusableRestoreInstance:
		if operation.InstanceID == "" {
			return "", fmt.Errorf("%w: instance id is required", core.ErrReusableInvalid)
		}
		return reusableInstanceTarget(operation.InstanceID), nil
	}
	return "", fmt.Errorf("%w: unsupported reusable operation", core.ErrReusableInvalid)
}

func reusableInstanceTarget(id string) string { return "instance:" + strings.TrimSpace(id) }

func normalizeReusableOperation(operation ReusableOperation) ReusableOperation {
	if operation.SchemaVersion == 0 {
		operation.SchemaVersion = 1
	}
	operation.ID = strings.TrimSpace(operation.ID)
	operation.Kind = ReusableOperationKind(strings.TrimSpace(string(operation.Kind)))
	operation.DefinitionID = strings.TrimSpace(operation.DefinitionID)
	operation.InstanceID = strings.TrimSpace(operation.InstanceID)
	operation.OverrideKey = strings.TrimSpace(operation.OverrideKey)
	operation.ExpectedHead = strings.TrimSpace(operation.ExpectedHead)
	operation.HistoryID = strings.TrimSpace(operation.HistoryID)
	if operation.Definition != nil {
		copy := operation.Definition.Normalize()
		operation.Definition = &copy
	}
	if operation.Instance != nil {
		copy := operation.Instance.Normalize()
		operation.Instance = &copy
	}
	return operation
}

func supportedReusableOperation(kind ReusableOperationKind) bool {
	switch kind {
	case ReusableCreateDefinition, ReusableCreateInstance, ReusableUpdateDefinition, ReusableSetInstanceOverride, ReusableClearOverride, ReusableDetachInstance, ReusableRestoreInstance:
		return true
	default:
		return false
	}
}

func reusableResult(state ReusableState, record ReusableOperationRecord) ReusableOperationResult {
	result := ReusableOperationResult{State: cloneReusableState(state), Record: record}
	if strings.HasPrefix(record.TargetKey, "definition:") {
		if definition, ok := state.Definitions[strings.TrimPrefix(record.TargetKey, "definition:")]; ok {
			copy := definition.Normalize()
			result.Definition = &copy
		}
	} else if instance, ok := state.Instances[strings.TrimPrefix(record.TargetKey, "instance:")]; ok {
		copy := instance.Normalize()
		result.Instance = &copy
	}
	return result
}

func cloneReusableState(state ReusableState) ReusableState {
	data, _ := json.Marshal(state)
	var clone ReusableState
	_ = json.Unmarshal(data, &clone)
	clone.ensure()
	return clone
}

func reusableRequestHash(operation ReusableOperation) string {
	operation.ExpectedHead = ""
	data, _ := json.Marshal(operation)
	return reusableHashString(string(data))
}

func reusableHead(id string, revision uint64, target string) string {
	return reusableHashString(fmt.Sprintf("%s:%d:%s", id, revision, target))
}

func reusableHashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func snapshotInstanceIDs(snapshot ReusableTargetSnapshot) []string {
	ids := make([]string, 0, len(snapshot.Instances)+1)
	if snapshot.InstanceID != "" {
		ids = append(ids, snapshot.InstanceID)
	}
	for id := range snapshot.Instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func validateReusableDefinitionValues(definition core.ComponentDefinition) error {
	for scope, properties := range definition.Styles {
		breakpoint, state, ok := reusableScopeParts(scope)
		if !ok || !isSupportedStyleBreakpoint(breakpoint) || !isSupportedStyleState(state) {
			return fmt.Errorf("%w: unsupported definition style scope %q", core.ErrReusableInvalid, scope)
		}
		for property, value := range properties {
			if !IsSupportedStyleProperty(property) {
				return fmt.Errorf("%w: unsupported definition style property %q", core.ErrReusableInvalid, property)
			}
			if reason := styleValueRejection(value); reason != "" {
				return fmt.Errorf("%w: %s", core.ErrReusableInvalid, reason)
			}
		}
	}
	for scope, properties := range definition.Layout {
		breakpoint, state, ok := reusableScopeParts(scope)
		if !ok || !isSupportedStyleBreakpoint(breakpoint) || !isSupportedStyleState(state) {
			return fmt.Errorf("%w: unsupported definition layout scope %q", core.ErrReusableInvalid, scope)
		}
		for property, value := range properties {
			if reason, valid := ValidateLayoutValue(property, value); !valid {
				return fmt.Errorf("%w: %s", core.ErrReusableInvalid, reason)
			}
		}
	}
	return nil
}

func validateReusableOverrideValue(key, value string) error {
	parts := strings.Split(key, ":")
	if len(parts) != 3 || (parts[0] != "style" && parts[0] != "layout") {
		if len(value) > 1<<20 || strings.ContainsRune(value, '\x00') {
			return fmt.Errorf("%w: override value is too large or invalid", core.ErrReusableInvalid)
		}
		return nil
	}
	breakpoint, state, ok := reusableScopeParts(parts[1])
	if !ok || !isSupportedStyleBreakpoint(breakpoint) || !isSupportedStyleState(state) {
		return fmt.Errorf("%w: unsupported override scope", core.ErrReusableInvalid)
	}
	if parts[0] == "layout" {
		if reason, valid := ValidateLayoutValue(parts[2], value); !valid {
			return fmt.Errorf("%w: %s", core.ErrReusableInvalid, reason)
		}
		return nil
	}
	if !IsSupportedStyleProperty(parts[2]) {
		return fmt.Errorf("%w: unsupported style property", core.ErrReusableInvalid)
	}
	if reason := styleValueRejection(value); reason != "" {
		return fmt.Errorf("%w: %s", core.ErrReusableInvalid, reason)
	}
	return nil
}

func reusableScopeParts(scope string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(scope), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return normalizeStyleBreakpoint(parts[0]), normalizeStyleState(parts[1]), true
}
