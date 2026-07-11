package authoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"m31labs.dev/gosx-studio/cms/media"
)

type AssetOperationKind string

const (
	AssetImport         AssetOperationKind = "import-asset"
	AssetUpdateMetadata AssetOperationKind = "update-asset-metadata"
	AssetReplace        AssetOperationKind = "replace-asset"
	AssetDelete         AssetOperationKind = "delete-asset"
	AssetBind           AssetOperationKind = "bind-asset"
	AssetUnbind         AssetOperationKind = "unbind-asset"
	AssetUndo           AssetOperationKind = "undo"
	AssetRedo           AssetOperationKind = "redo"
)

const (
	AssetFieldOperation    = "gosx_studio_asset_operation"
	AssetFieldOperationID  = "gosx_studio_asset_operation_id"
	AssetFieldAssetID      = "gosx_studio_asset_id"
	AssetFieldBindingID    = "gosx_studio_asset_binding_id"
	AssetFieldInstanceID   = "gosx_studio_asset_instance_id"
	AssetFieldDefinitionID = "gosx_studio_asset_definition_id"
	AssetFieldComponentKey = "gosx_studio_asset_component_key"
	AssetFieldProperty     = "gosx_studio_asset_property"
	AssetFieldExpectedHead = "gosx_studio_asset_expected_head"
	AssetFieldHistoryID    = "gosx_studio_asset_history_id"
)

var (
	ErrAssetInvalid      = errors.New("invalid asset operation")
	ErrAssetUnauthorized = errors.New("asset operation is not authorized")
	ErrAssetConflict     = errors.New("asset target is stale")
	ErrAssetIdempotency  = errors.New("asset operation id was reused")
	ErrAssetNotFound     = errors.New("asset not found")
	ErrAssetReferenced   = errors.New("asset is still referenced")
)

type AssetBindingTarget struct {
	InstanceID   string `json:"instanceId,omitempty"`
	DefinitionID string `json:"definitionId,omitempty"`
	ComponentKey string `json:"componentKey,omitempty"`
	Property     string `json:"property"`
}

type AssetBinding struct {
	ID         string                   `json:"id"`
	AssetID    string                   `json:"assetId"`
	Target     AssetBindingTarget       `json:"target"`
	Responsive media.ResponsiveMetadata `json:"responsive,omitempty"`
	Revision   uint64                   `json:"revision"`
}

type AssetState struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Revision      uint64                  `json:"revision"`
	Assets        map[string]media.Asset  `json:"assets,omitempty"`
	Bindings      map[string]AssetBinding `json:"bindings,omitempty"`
	Heads         map[string]string       `json:"heads,omitempty"`
	Operations    []AssetOperationRecord  `json:"operations,omitempty"`
}

type AssetOperation struct {
	SchemaVersion int                `json:"schemaVersion"`
	ID            string             `json:"id"`
	Kind          AssetOperationKind `json:"kind"`
	Asset         *media.Asset       `json:"asset,omitempty"`
	AssetID       string             `json:"assetId,omitempty"`
	Binding       *AssetBinding      `json:"binding,omitempty"`
	BindingID     string             `json:"bindingId,omitempty"`
	ExpectedHead  string             `json:"expectedHead,omitempty"`
	HistoryID     string             `json:"historyId,omitempty"`
}

type AssetSnapshot struct {
	Assets   map[string]media.Asset  `json:"assets,omitempty"`
	Bindings map[string]AssetBinding `json:"bindings,omitempty"`
}
type AssetOperationRecord struct {
	SchemaVersion int                `json:"schemaVersion"`
	ID            string             `json:"id"`
	ActorID       string             `json:"actorId"`
	ActorLabel    string             `json:"actorLabel,omitempty"`
	Kind          AssetOperationKind `json:"kind"`
	TargetKey     string             `json:"targetKey"`
	RequestHash   string             `json:"requestHash"`
	Before        AssetSnapshot      `json:"before"`
	After         AssetSnapshot      `json:"after"`
	Revision      uint64             `json:"revision"`
	PreviousHead  string             `json:"previousHead,omitempty"`
	TargetHead    string             `json:"targetHead"`
	UndoOf        string             `json:"undoOf,omitempty"`
	RedoOf        string             `json:"redoOf,omitempty"`
}
type AssetApplyOptions struct {
	ActorID, ActorLabel            string
	CanManageAssets, CanBindAssets bool
	Reusable                       ReusableState
	Interactions                   InteractionState
}
type AssetOperationResult struct {
	State   AssetState
	Record  AssetOperationRecord
	Asset   *media.Asset
	Binding *AssetBinding
}

type AssetStateStore interface {
	LoadAssetState(context.Context) (AssetState, error)
	SaveAssetState(context.Context, AssetState, uint64) error
}

func CommitAssetOperation(ctx context.Context, store AssetStateStore, operation AssetOperation, options AssetApplyOptions) (AssetOperationResult, error) {
	if store == nil {
		return AssetOperationResult{}, fmt.Errorf("%w: state store is required", ErrAssetInvalid)
	}
	state, err := store.LoadAssetState(ctx)
	if err != nil {
		return AssetOperationResult{}, err
	}
	expected := state.Revision
	result, err := ApplyAssetOperation(state, operation, options)
	if err != nil {
		return AssetOperationResult{}, err
	}
	if err := store.SaveAssetState(ctx, result.State, expected); err != nil {
		return AssetOperationResult{}, err
	}
	return result, nil
}

func ApplyAssetOperation(state AssetState, operation AssetOperation, options AssetApplyOptions) (AssetOperationResult, error) {
	state = normalizeAssetState(state)
	operation = normalizeAssetOperation(operation)
	options.ActorID = strings.TrimSpace(options.ActorID)
	options.ActorLabel = strings.TrimSpace(options.ActorLabel)
	if options.ActorID == "" {
		return AssetOperationResult{}, ErrAssetUnauthorized
	}
	if operation.SchemaVersion != 1 || operation.ID == "" {
		return AssetOperationResult{}, fmt.Errorf("%w: schema and operation id are required", ErrAssetInvalid)
	}
	manage := operation.Kind == AssetImport || operation.Kind == AssetUpdateMetadata || operation.Kind == AssetReplace || operation.Kind == AssetDelete
	bind := operation.Kind == AssetBind || operation.Kind == AssetUnbind
	history := operation.Kind == AssetUndo || operation.Kind == AssetRedo
	if manage && !options.CanManageAssets || bind && !options.CanBindAssets {
		return AssetOperationResult{}, ErrAssetUnauthorized
	}
	if history && !options.CanManageAssets && !options.CanBindAssets {
		return AssetOperationResult{}, ErrAssetUnauthorized
	}
	requestHash := assetRequestHash(operation)
	for _, record := range state.Operations {
		if record.ActorID == options.ActorID && record.ID == operation.ID {
			if record.RequestHash != requestHash {
				return AssetOperationResult{}, ErrAssetIdempotency
			}
			return assetResultFromRecord(state, record), nil
		}
	}
	targetKey, err := assetOperationTargetKey(operation, state)
	if err != nil {
		return AssetOperationResult{}, err
	}
	if operation.ExpectedHead != "" && state.Heads[targetKey] != operation.ExpectedHead {
		return AssetOperationResult{}, ErrAssetConflict
	}
	before := snapshotAssetState(state)
	undoOf, redoOf := "", ""
	if operation.Kind == AssetUndo || operation.Kind == AssetRedo {
		source := assetRecordByID(state.Operations, operation.HistoryID)
		if source == nil || source.ActorID != options.ActorID || state.Heads[source.TargetKey] != source.TargetHead {
			return AssetOperationResult{}, ErrAssetConflict
		}
		if (strings.HasPrefix(source.TargetKey, "asset:") && !options.CanManageAssets) || (strings.HasPrefix(source.TargetKey, "binding:") && !options.CanBindAssets) {
			return AssetOperationResult{}, ErrAssetUnauthorized
		}
		targetKey = source.TargetKey
		if operation.Kind == AssetUndo {
			undoOf = source.ID
			restoreAssetSnapshot(&state, source.Before)
		} else {
			if source.Kind != AssetUndo || source.UndoOf == "" {
				return AssetOperationResult{}, ErrAssetConflict
			}
			original := assetRecordByID(state.Operations, source.UndoOf)
			if original == nil {
				return AssetOperationResult{}, ErrAssetConflict
			}
			redoOf = original.ID
			restoreAssetSnapshot(&state, original.After)
		}
	} else if err := applyAssetMutation(&state, operation, options); err != nil {
		return AssetOperationResult{}, err
	}
	state.Revision++
	head := assetHead(operation.ID, state.Revision)
	record := AssetOperationRecord{SchemaVersion: 1, ID: operation.ID, ActorID: options.ActorID, ActorLabel: options.ActorLabel, Kind: operation.Kind, TargetKey: targetKey, RequestHash: requestHash, Before: before, After: snapshotAssetState(state), Revision: state.Revision, PreviousHead: state.Heads[targetKey], TargetHead: head, UndoOf: undoOf, RedoOf: redoOf}
	state.Heads[targetKey] = head
	state.Operations = append(state.Operations, record)
	result := assetResultFromRecord(state, record)
	if operation.AssetID != "" {
		if asset, ok := state.Assets[operation.AssetID]; ok {
			copy := media.CloneAsset(asset)
			result.Asset = &copy
		}
	}
	if operation.BindingID != "" {
		if binding, ok := state.Bindings[operation.BindingID]; ok {
			copy := cloneAssetBinding(binding)
			result.Binding = &copy
		}
	}
	if operation.Binding != nil {
		if binding, ok := state.Bindings[operation.Binding.ID]; ok {
			copy := cloneAssetBinding(binding)
			result.Binding = &copy
		}
	}
	return result, nil
}

func applyAssetMutation(state *AssetState, op AssetOperation, options AssetApplyOptions) error {
	switch op.Kind {
	case AssetImport:
		if op.Asset == nil {
			return fmt.Errorf("%w: asset is required", ErrAssetInvalid)
		}
		asset := normalizeTypedAsset(*op.Asset)
		if err := validateTypedAsset(asset, true); err != nil {
			return err
		}
		if _, exists := state.Assets[asset.ID]; exists {
			return ErrAssetIdempotency
		}
		asset.Revision = state.Revision + 1
		if asset.Version == 0 {
			asset.Version = 1
		}
		state.Assets[asset.ID] = asset
	case AssetUpdateMetadata:
		current, ok := state.Assets[op.AssetID]
		if !ok || op.Asset == nil {
			return ErrAssetNotFound
		}
		next := normalizeTypedAsset(*op.Asset)
		current.Alt = next.Alt
		current.Caption = next.Caption
		current.License = next.License
		current.Responsive = next.Responsive
		current.FocalX = next.FocalX
		current.FocalY = next.FocalY
		current.Updated = next.Updated
		current.Revision = state.Revision + 1
		if err := validateTypedAsset(current, false); err != nil {
			return err
		}
		state.Assets[current.ID] = current
	case AssetReplace:
		current, ok := state.Assets[op.AssetID]
		if !ok || op.Asset == nil {
			return ErrAssetNotFound
		}
		next := normalizeTypedAsset(*op.Asset)
		next.ID = current.ID
		next.Version = current.Version + 1
		next.Revision = state.Revision + 1
		next.Created = current.Created
		if next.Alt == "" {
			next.Alt = current.Alt
		}
		if len(next.Responsive) == 0 {
			next.Responsive = current.Responsive
		}
		if err := validateTypedAsset(next, true); err != nil {
			return err
		}
		state.Assets[current.ID] = next
	case AssetDelete:
		if _, ok := state.Assets[op.AssetID]; !ok {
			return ErrAssetNotFound
		}
		for _, binding := range state.Bindings {
			if binding.AssetID == op.AssetID {
				return ErrAssetReferenced
			}
		}
		if err := ValidateInteractionReferenceDeletion(options.Interactions, "asset", op.AssetID); err != nil {
			return err
		}
		delete(state.Assets, op.AssetID)
	case AssetBind:
		if op.Binding == nil {
			return fmt.Errorf("%w: binding is required", ErrAssetInvalid)
		}
		binding := cloneAssetBinding(*op.Binding)
		if err := validateAssetBinding(binding, *state, options.Reusable); err != nil {
			return err
		}
		if existing, ok := state.Bindings[binding.ID]; ok {
			binding.Revision = existing.Revision + 1
		} else {
			binding.Revision = 1
		}
		state.Bindings[binding.ID] = binding
	case AssetUnbind:
		if _, ok := state.Bindings[op.BindingID]; !ok {
			return ErrAssetNotFound
		}
		delete(state.Bindings, op.BindingID)
	default:
		return fmt.Errorf("%w: unsupported operation %q", ErrAssetInvalid, op.Kind)
	}
	return nil
}

func normalizeTypedAsset(asset media.Asset) media.Asset {
	input := media.Input{URL: asset.URL, Alt: asset.Alt, Filename: asset.Filename, ContentType: asset.ContentType, Size: asset.Size, Variants: asset.Variants, FocalX: asset.FocalX, FocalY: asset.FocalY, Kind: asset.Kind, ContentHash: asset.ContentHash, Caption: asset.Caption, License: asset.License, Responsive: asset.Responsive}
	asset = media.NormalizeAsset(input, asset)
	asset.ID = strings.TrimSpace(asset.ID)
	asset.ContentHash = strings.ToLower(strings.TrimSpace(asset.ContentHash))
	return asset
}
func validateTypedAsset(asset media.Asset, contentRequired bool) error {
	if !stableAssetID(asset.ID) {
		return fmt.Errorf("%w: stable asset id is required", ErrAssetInvalid)
	}
	if asset.Kind != media.AssetKindImage && asset.Kind != media.AssetKindMedia && asset.Kind != media.AssetKindDocument {
		return fmt.Errorf("%w: unsupported asset kind", ErrAssetInvalid)
	}
	if contentRequired && (asset.URL == "" || asset.Filename == "" || asset.ContentType == "" || asset.Size <= 0 || len(asset.ContentHash) != 64) {
		return fmt.Errorf("%w: stored content metadata and sha256 are required", ErrAssetInvalid)
	}
	if len(asset.ContentHash) > 0 {
		if _, err := hex.DecodeString(asset.ContentHash); err != nil || len(asset.ContentHash) != 64 {
			return fmt.Errorf("%w: invalid content hash", ErrAssetInvalid)
		}
	}
	for key, value := range asset.Responsive {
		validCrop := value.Crop == "" || value.Crop == media.CropCover || value.Crop == media.CropContain || value.Crop == media.CropNone
		if key != "base" && key != "tablet" && key != "mobile" || !validCrop || value.FocalX < 0 || value.FocalX > 1 || value.FocalY < 0 || value.FocalY > 1 {
			return fmt.Errorf("%w: invalid responsive metadata", ErrAssetInvalid)
		}
	}
	return nil
}

func validateAssetBinding(binding AssetBinding, state AssetState, reusable ReusableState) error {
	binding = cloneAssetBinding(binding)
	asset, ok := state.Assets[binding.AssetID]
	if !ok {
		return ErrAssetNotFound
	}
	if !stableAssetID(binding.ID) || !stableAssetProperty(binding.Target.Property) {
		return fmt.Errorf("%w: stable binding id and property are required", ErrAssetInvalid)
	}
	targets := 0
	if binding.Target.InstanceID != "" {
		targets++
		if _, ok := reusable.Instances[binding.Target.InstanceID]; !ok {
			return fmt.Errorf("%w: reusable instance not found", ErrAssetInvalid)
		}
	}
	if binding.Target.DefinitionID != "" {
		targets++
		if _, ok := reusable.Definitions[binding.Target.DefinitionID]; !ok {
			return fmt.Errorf("%w: reusable definition not found", ErrAssetInvalid)
		}
	}
	if binding.Target.ComponentKey != "" {
		targets++
	}
	if targets != 1 {
		return fmt.Errorf("%w: exactly one component target is required", ErrAssetInvalid)
	}
	if asset.Kind == media.AssetKindImage {
		alt := strings.TrimSpace(asset.Alt)
		if base, ok := binding.Responsive["base"]; ok && strings.TrimSpace(base.Alt) != "" {
			alt = strings.TrimSpace(base.Alt)
		}
		if alt == "" {
			return fmt.Errorf("%w: image alt text is required before binding", ErrAssetInvalid)
		}
	}
	return nil
}

func ResolveAssetBinding(state AssetState, reusable ReusableState, instanceID, definitionID, componentKey, property string) (AssetBinding, media.Asset, bool) {
	state = normalizeAssetState(state)
	candidates := []string{}
	for id, binding := range state.Bindings {
		if binding.Target.Property != property {
			continue
		}
		if instanceID != "" && binding.Target.InstanceID == instanceID {
			candidates = append(candidates, "0:"+id)
		} else if definitionID != "" && binding.Target.DefinitionID == definitionID {
			candidates = append(candidates, "1:"+id)
		} else if componentKey != "" && binding.Target.ComponentKey == componentKey {
			candidates = append(candidates, "2:"+id)
		}
	}
	sort.Strings(candidates)
	if len(candidates) == 0 {
		return AssetBinding{}, media.Asset{}, false
	}
	id := strings.SplitN(candidates[0], ":", 2)[1]
	binding := state.Bindings[id]
	asset, ok := state.Assets[binding.AssetID]
	_ = reusable
	return cloneAssetBinding(binding), media.CloneAsset(asset), ok
}

func AssetIDFromContentHash(hash string) string {
	hash = strings.ToLower(strings.TrimSpace(hash))
	if len(hash) < 24 {
		return ""
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return ""
	}
	return "asset_" + hash[:24]
}
func AssetBindingID(target AssetBindingTarget) string {
	target = normalizeAssetBindingTarget(target)
	raw, _ := json.Marshal(target)
	sum := sha256.Sum256(raw)
	return "binding_" + hex.EncodeToString(sum[:12])
}
func normalizeAssetState(state AssetState) AssetState {
	if state.SchemaVersion == 0 {
		state.SchemaVersion = 1
	}
	if state.Assets == nil {
		state.Assets = map[string]media.Asset{}
	}
	if state.Bindings == nil {
		state.Bindings = map[string]AssetBinding{}
	}
	if state.Heads == nil {
		state.Heads = map[string]string{}
	}
	return cloneAssetState(state)
}
func normalizeAssetOperation(op AssetOperation) AssetOperation {
	if op.SchemaVersion == 0 {
		op.SchemaVersion = 1
	}
	op.ID = strings.TrimSpace(op.ID)
	op.AssetID = strings.TrimSpace(op.AssetID)
	op.BindingID = strings.TrimSpace(op.BindingID)
	op.ExpectedHead = strings.TrimSpace(op.ExpectedHead)
	op.HistoryID = strings.TrimSpace(op.HistoryID)
	if op.Asset != nil {
		copy := normalizeTypedAsset(*op.Asset)
		op.Asset = &copy
	}
	if op.Binding != nil {
		copy := cloneAssetBinding(*op.Binding)
		op.Binding = &copy
	}
	return op
}
func normalizeAssetBindingTarget(target AssetBindingTarget) AssetBindingTarget {
	target.InstanceID = strings.TrimSpace(target.InstanceID)
	target.DefinitionID = strings.TrimSpace(target.DefinitionID)
	target.ComponentKey = strings.TrimSpace(target.ComponentKey)
	target.Property = strings.ToLower(strings.TrimSpace(target.Property))
	return target
}
func cloneAssetBinding(binding AssetBinding) AssetBinding {
	binding.ID = strings.TrimSpace(binding.ID)
	binding.AssetID = strings.TrimSpace(binding.AssetID)
	binding.Target = normalizeAssetBindingTarget(binding.Target)
	binding.Responsive = media.NormalizeResponsiveMetadata(binding.Responsive)
	return binding
}
func cloneAssetState(state AssetState) AssetState {
	out := state
	out.Assets = map[string]media.Asset{}
	for id, asset := range state.Assets {
		out.Assets[id] = media.CloneAsset(asset)
	}
	out.Bindings = map[string]AssetBinding{}
	for id, binding := range state.Bindings {
		out.Bindings[id] = cloneAssetBinding(binding)
	}
	out.Heads = map[string]string{}
	for key, head := range state.Heads {
		out.Heads[key] = head
	}
	out.Operations = append([]AssetOperationRecord(nil), state.Operations...)
	return out
}
func snapshotAssetState(state AssetState) AssetSnapshot {
	return AssetSnapshot{Assets: cloneAssetState(state).Assets, Bindings: cloneAssetState(state).Bindings}
}
func restoreAssetSnapshot(state *AssetState, snapshot AssetSnapshot) {
	state.Assets = cloneAssetState(AssetState{Assets: snapshot.Assets}).Assets
	state.Bindings = cloneAssetState(AssetState{Bindings: snapshot.Bindings}).Bindings
}
func assetOperationTargetKey(op AssetOperation, state AssetState) (string, error) {
	switch op.Kind {
	case AssetImport:
		if op.Asset == nil {
			return "", ErrAssetInvalid
		}
		return "asset:" + op.Asset.ID, nil
	case AssetUpdateMetadata, AssetReplace, AssetDelete:
		return "asset:" + op.AssetID, nil
	case AssetBind:
		if op.Binding == nil {
			return "", ErrAssetInvalid
		}
		return "binding:" + op.Binding.ID, nil
	case AssetUnbind:
		return "binding:" + op.BindingID, nil
	case AssetUndo, AssetRedo:
		record := assetRecordByID(state.Operations, op.HistoryID)
		if record == nil {
			return "", ErrAssetNotFound
		}
		return record.TargetKey, nil
	default:
		return "", ErrAssetInvalid
	}
}
func assetRequestHash(op AssetOperation) string {
	copy := op
	copy.ExpectedHead = ""
	raw, _ := json.Marshal(copy)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func assetHead(id string, revision uint64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", id, revision)))
	return hex.EncodeToString(sum[:])
}
func assetRecordByID(rows []AssetOperationRecord, id string) *AssetOperationRecord {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}
func assetResultFromRecord(state AssetState, record AssetOperationRecord) AssetOperationResult {
	return AssetOperationResult{State: cloneAssetState(state), Record: record}
}
func stableAssetID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}
func stableAssetProperty(value string) bool {
	return stableAssetID(strings.ReplaceAll(strings.TrimSpace(value), ":", "_"))
}
