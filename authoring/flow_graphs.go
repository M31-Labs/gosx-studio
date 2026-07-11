package authoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"m31labs.dev/gosx-studio/cms/flows"
	"strings"
)

type FlowOperationKind string

const (
	FlowUpsert FlowOperationKind = "upsert-flow"
	FlowDelete FlowOperationKind = "delete-flow"
	FlowUndo   FlowOperationKind = "undo"
	FlowRedo   FlowOperationKind = "redo"
)

var (
	ErrFlowUnauthorized = errors.New("flow operation unauthorized")
	ErrFlowConflict     = errors.New("flow target stale")
	ErrFlowIdempotency  = errors.New("flow operation id reused")
	ErrFlowNotFound     = errors.New("flow not found")
	ErrFlowReferenced   = errors.New("flow reference prevents deletion")
)

type FlowState struct {
	SchemaVersion int
	Revision      uint64
	Flows         map[string]flows.Graph
	Heads         map[string]string
	Operations    []FlowOperationRecord
}
type FlowOperation struct {
	SchemaVersion                   int
	ID                              string
	Kind                            FlowOperationKind
	Flow                            *flows.Graph
	FlowID, ExpectedHead, HistoryID string
}
type FlowOperationRecord struct {
	SchemaVersion                            int
	ID, ActorID, ActorLabel                  string
	Kind                                     FlowOperationKind
	TargetKey, RequestHash                   string
	Before, After                            map[string]flows.Graph
	Revision                                 uint64
	PreviousHead, TargetHead, UndoOf, RedoOf string
}
type FlowApplyOptions struct {
	ActorID, ActorLabel string
	CanManageFlows      bool
	Reusable            ReusableState
	Assets              AssetState
	CanvasIDs           map[string]bool
}
type FlowResult struct {
	State  FlowState
	Record FlowOperationRecord
	Flow   *flows.Graph
}
type FlowStateStore interface {
	LoadFlowState(context.Context) (FlowState, error)
	SaveFlowState(context.Context, FlowState, uint64) error
}

func CommitFlowOperation(ctx context.Context, store FlowStateStore, op FlowOperation, opt FlowApplyOptions) (FlowResult, error) {
	if store == nil {
		return FlowResult{}, flows.ErrInvalidGraph
	}
	s, e := store.LoadFlowState(ctx)
	if e != nil {
		return FlowResult{}, e
	}
	rev := s.Revision
	r, e := ApplyFlowOperation(s, op, opt)
	if e != nil {
		return r, e
	}
	if e = store.SaveFlowState(ctx, r.State, rev); e != nil {
		return FlowResult{}, e
	}
	return r, nil
}
func ApplyFlowOperation(s FlowState, op FlowOperation, opt FlowApplyOptions) (FlowResult, error) {
	s = normalizeFlowState(s)
	op = normalizeFlowOp(op)
	opt.ActorID = strings.TrimSpace(opt.ActorID)
	if opt.ActorID == "" || !opt.CanManageFlows {
		return FlowResult{}, ErrFlowUnauthorized
	}
	if op.ID == "" || op.SchemaVersion != 1 {
		return FlowResult{}, flows.ErrInvalidGraph
	}
	hash := flowHash(op)
	for _, r := range s.Operations {
		if r.ActorID == opt.ActorID && r.ID == op.ID {
			if r.RequestHash != hash {
				return FlowResult{}, ErrFlowIdempotency
			}
			return FlowResult{State: s, Record: r}, nil
		}
	}
	target, e := flowTarget(op, s)
	if e != nil {
		return FlowResult{}, e
	}
	if op.ExpectedHead != "" && s.Heads[target] != op.ExpectedHead {
		return FlowResult{}, ErrFlowConflict
	}
	before := cloneFlows(s.Flows)
	undoOf, redoOf := "", ""
	switch op.Kind {
	case FlowUpsert:
		g := op.Flow.Normalize()
		if e = g.Validate(); e != nil {
			return FlowResult{}, e
		}
		if e = validateFlowRefs(g, opt); e != nil {
			return FlowResult{}, e
		}
		g.Revision = s.Revision + 1
		s.Flows[g.ID] = g
	case FlowDelete:
		if _, ok := s.Flows[op.FlowID]; !ok {
			return FlowResult{}, ErrFlowNotFound
		}
		delete(s.Flows, op.FlowID)
	case FlowUndo, FlowRedo:
		src := flowRecord(s.Operations, op.HistoryID)
		if src == nil || src.ActorID != opt.ActorID || s.Heads[src.TargetKey] != src.TargetHead {
			return FlowResult{}, ErrFlowConflict
		}
		target = src.TargetKey
		if op.Kind == FlowUndo {
			undoOf = src.ID
			s.Flows = cloneFlows(src.Before)
		} else {
			if src.Kind != FlowUndo || src.UndoOf == "" {
				return FlowResult{}, ErrFlowConflict
			}
			orig := flowRecord(s.Operations, src.UndoOf)
			if orig == nil {
				return FlowResult{}, ErrFlowConflict
			}
			redoOf = orig.ID
			s.Flows = cloneFlows(orig.After)
		}
	default:
		return FlowResult{}, flows.ErrInvalidGraph
	}
	s.Revision++
	head := flowHead(op.ID, s.Revision)
	rec := FlowOperationRecord{SchemaVersion: 1, ID: op.ID, ActorID: opt.ActorID, ActorLabel: strings.TrimSpace(opt.ActorLabel), Kind: op.Kind, TargetKey: target, RequestHash: hash, Before: before, After: cloneFlows(s.Flows), Revision: s.Revision, PreviousHead: s.Heads[target], TargetHead: head, UndoOf: undoOf, RedoOf: redoOf}
	s.Heads[target] = head
	s.Operations = append(s.Operations, rec)
	return FlowResult{State: s, Record: rec}, nil
}
func validateFlowRefs(g flows.Graph, opt FlowApplyOptions) error {
	for _, s := range g.Steps {
		if s.ComponentID != "" && !opt.CanvasIDs[s.ComponentID] {
			return ErrFlowNotFound
		}
		if s.DefinitionID != "" {
			if _, ok := opt.Reusable.Definitions[s.DefinitionID]; !ok {
				return ErrFlowNotFound
			}
		}
		if s.InstanceID != "" {
			if _, ok := opt.Reusable.Instances[s.InstanceID]; !ok {
				return ErrFlowNotFound
			}
		}
		if s.AssetID != "" {
			if _, ok := opt.Assets.Assets[s.AssetID]; !ok {
				return ErrFlowNotFound
			}
		}
	}
	return nil
}
func ValidateFlowReferenceDeletion(s FlowState, kind, id string) error {
	for _, g := range s.Flows {
		for _, step := range g.Steps {
			if kind == "canvas" && step.ComponentID == id || kind == "definition" && step.DefinitionID == id || kind == "instance" && step.InstanceID == id || kind == "asset" && step.AssetID == id {
				return ErrFlowReferenced
			}
		}
	}
	return nil
}
func normalizeFlowState(s FlowState) FlowState {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = 1
	}
	if s.Flows == nil {
		s.Flows = map[string]flows.Graph{}
	}
	if s.Heads == nil {
		s.Heads = map[string]string{}
	}
	s.Flows = cloneFlows(s.Flows)
	h := map[string]string{}
	for k, v := range s.Heads {
		h[k] = v
	}
	s.Heads = h
	s.Operations = append([]FlowOperationRecord(nil), s.Operations...)
	return s
}
func normalizeFlowOp(op FlowOperation) FlowOperation {
	if op.SchemaVersion == 0 {
		op.SchemaVersion = 1
	}
	op.ID = strings.TrimSpace(op.ID)
	op.FlowID = strings.TrimSpace(op.FlowID)
	op.ExpectedHead = strings.TrimSpace(op.ExpectedHead)
	op.HistoryID = strings.TrimSpace(op.HistoryID)
	if op.Flow != nil {
		g := op.Flow.Normalize()
		op.Flow = &g
	}
	return op
}
func flowTarget(op FlowOperation, s FlowState) (string, error) {
	switch op.Kind {
	case FlowUpsert:
		if op.Flow == nil {
			return "", flows.ErrInvalidGraph
		}
		return "flow:" + op.Flow.ID, nil
	case FlowDelete:
		return "flow:" + op.FlowID, nil
	case FlowUndo, FlowRedo:
		r := flowRecord(s.Operations, op.HistoryID)
		if r == nil {
			return "", ErrFlowNotFound
		}
		return r.TargetKey, nil
	}
	return "", flows.ErrInvalidGraph
}
func flowRecord(rs []FlowOperationRecord, id string) *FlowOperationRecord {
	for i := range rs {
		if rs[i].ID == id {
			return &rs[i]
		}
	}
	return nil
}
func flowHash(op FlowOperation) string {
	op.ExpectedHead = ""
	raw, _ := json.Marshal(op)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func flowHead(id string, rev uint64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", id, rev)))
	return hex.EncodeToString(sum[:])
}
func cloneFlows(in map[string]flows.Graph) map[string]flows.Graph {
	raw, _ := json.Marshal(in)
	out := map[string]flows.Graph{}
	_ = json.Unmarshal(raw, &out)
	return out
}
