package authoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"m31labs.dev/gosx-studio/core"
)

type InteractionOperationKind string

const (
	InteractionUpsert InteractionOperationKind = "upsert-interaction"
	InteractionDelete InteractionOperationKind = "delete-interaction"
	InteractionUndo   InteractionOperationKind = "undo"
	InteractionRedo   InteractionOperationKind = "redo"
)

var (
	ErrInteractionUnauthorized = errors.New("interaction operation unauthorized")
	ErrInteractionConflict     = errors.New("interaction target stale")
	ErrInteractionIdempotency  = errors.New("interaction operation id reused")
	ErrInteractionNotFound     = errors.New("interaction not found")
	ErrInteractionReferenced   = errors.New("interaction target is referenced")
)

type InteractionState struct {
	SchemaVersion int                              `json:"schemaVersion"`
	Revision      uint64                           `json:"revision"`
	Graphs        map[string]core.InteractionGraph `json:"graphs,omitempty"`
	Heads         map[string]string                `json:"heads,omitempty"`
	Operations    []InteractionOperationRecord     `json:"operations,omitempty"`
}
type InteractionOperation struct {
	SchemaVersion int                      `json:"schemaVersion"`
	ID            string                   `json:"id"`
	Kind          InteractionOperationKind `json:"kind"`
	Graph         *core.InteractionGraph   `json:"graph,omitempty"`
	GraphID       string                   `json:"graphId,omitempty"`
	ExpectedHead  string                   `json:"expectedHead,omitempty"`
	HistoryID     string                   `json:"historyId,omitempty"`
}
type InteractionOperationRecord struct {
	SchemaVersion                            int `json:"schemaVersion"`
	ID, ActorID, ActorLabel                  string
	Kind                                     InteractionOperationKind `json:"kind"`
	TargetKey, RequestHash                   string
	Before, After                            map[string]core.InteractionGraph
	Revision                                 uint64
	PreviousHead, TargetHead, UndoOf, RedoOf string
}
type InteractionApplyOptions struct {
	ActorID, ActorLabel   string
	CanManageInteractions bool
	Reusable              ReusableState
	CanvasIDs             map[string]bool
	AssetState            AssetState
}
type InteractionResult struct {
	State  InteractionState
	Record InteractionOperationRecord
	Graph  *core.InteractionGraph
}
type InteractionStateStore interface {
	LoadInteractionState(context.Context) (InteractionState, error)
	SaveInteractionState(context.Context, InteractionState, uint64) error
}

func CommitInteractionOperation(ctx context.Context, store InteractionStateStore, op InteractionOperation, opt InteractionApplyOptions) (InteractionResult, error) {
	if store == nil {
		return InteractionResult{}, core.ErrInvalidInteraction
	}
	s, e := store.LoadInteractionState(ctx)
	if e != nil {
		return InteractionResult{}, e
	}
	rev := s.Revision
	r, e := ApplyInteractionOperation(s, op, opt)
	if e != nil {
		return r, e
	}
	if e = store.SaveInteractionState(ctx, r.State, rev); e != nil {
		return InteractionResult{}, e
	}
	return r, nil
}
func ApplyInteractionOperation(s InteractionState, op InteractionOperation, opt InteractionApplyOptions) (InteractionResult, error) {
	s = normalizeInteractionState(s)
	op = normalizeInteractionOp(op)
	opt.ActorID = strings.TrimSpace(opt.ActorID)
	if opt.ActorID == "" || !opt.CanManageInteractions {
		return InteractionResult{}, ErrInteractionUnauthorized
	}
	if op.ID == "" || op.SchemaVersion != 1 {
		return InteractionResult{}, core.ErrInvalidInteraction
	}
	hash := interactionRequestHash(op)
	for _, r := range s.Operations {
		if r.ActorID == opt.ActorID && r.ID == op.ID {
			if r.RequestHash != hash {
				return InteractionResult{}, ErrInteractionIdempotency
			}
			return InteractionResult{State: s, Record: r}, nil
		}
	}
	target, e := interactionTarget(op, s)
	if e != nil {
		return InteractionResult{}, e
	}
	if op.ExpectedHead != "" && s.Heads[target] != op.ExpectedHead {
		return InteractionResult{}, ErrInteractionConflict
	}
	before := cloneGraphs(s.Graphs)
	undoOf, redoOf := "", ""
	switch op.Kind {
	case InteractionUpsert:
		g := op.Graph.Normalize()
		if e = g.Validate(); e != nil {
			return InteractionResult{}, e
		}
		if e = validateInteractionReferences(g, opt); e != nil {
			return InteractionResult{}, e
		}
		g.Revision = s.Revision + 1
		s.Graphs[g.ID] = g
	case InteractionDelete:
		if _, ok := s.Graphs[op.GraphID]; !ok {
			return InteractionResult{}, ErrInteractionNotFound
		}
		delete(s.Graphs, op.GraphID)
	case InteractionUndo, InteractionRedo:
		src := interactionRecord(s.Operations, op.HistoryID)
		if src == nil || src.ActorID != opt.ActorID || s.Heads[src.TargetKey] != src.TargetHead {
			return InteractionResult{}, ErrInteractionConflict
		}
		target = src.TargetKey
		if op.Kind == InteractionUndo {
			undoOf = src.ID
			s.Graphs = cloneGraphs(src.Before)
		} else {
			if src.Kind != InteractionUndo || src.UndoOf == "" {
				return InteractionResult{}, ErrInteractionConflict
			}
			orig := interactionRecord(s.Operations, src.UndoOf)
			if orig == nil {
				return InteractionResult{}, ErrInteractionConflict
			}
			redoOf = orig.ID
			s.Graphs = cloneGraphs(orig.After)
		}
	default:
		return InteractionResult{}, core.ErrInvalidInteraction
	}
	s.Revision++
	head := interactionHead(op.ID, s.Revision)
	rec := InteractionOperationRecord{SchemaVersion: 1, ID: op.ID, ActorID: opt.ActorID, ActorLabel: strings.TrimSpace(opt.ActorLabel), Kind: op.Kind, TargetKey: target, RequestHash: hash, Before: before, After: cloneGraphs(s.Graphs), Revision: s.Revision, PreviousHead: s.Heads[target], TargetHead: head, UndoOf: undoOf, RedoOf: redoOf}
	s.Heads[target] = head
	s.Operations = append(s.Operations, rec)
	result := InteractionResult{State: s, Record: rec}
	id := op.GraphID
	if op.Graph != nil {
		id = op.Graph.ID
	}
	if g, ok := s.Graphs[id]; ok {
		copy := g
		result.Graph = &copy
	}
	return result, nil
}

func validateInteractionReferences(g core.InteractionGraph, opt InteractionApplyOptions) error {
	t := g.Target
	if t.CanvasID != "" && !opt.CanvasIDs[t.CanvasID] {
		return ErrInteractionNotFound
	}
	if t.DefinitionID != "" {
		if _, ok := opt.Reusable.Definitions[t.DefinitionID]; !ok {
			return ErrInteractionNotFound
		}
	}
	if t.InstanceID != "" {
		if _, ok := opt.Reusable.Instances[t.InstanceID]; !ok {
			return ErrInteractionNotFound
		}
	}
	for _, a := range g.Actions {
		if a.AssetID != "" {
			if _, ok := opt.AssetState.Assets[a.AssetID]; !ok {
				return ErrInteractionNotFound
			}
		}
	}
	return nil
}
func ValidateInteractionReferenceDeletion(s InteractionState, kind, id string) error {
	for _, g := range s.Graphs {
		if kind == "canvas" && g.Target.CanvasID == id || kind == "definition" && g.Target.DefinitionID == id || kind == "instance" && g.Target.InstanceID == id {
			return ErrInteractionReferenced
		}
		if kind == "asset" {
			for _, a := range g.Actions {
				if a.AssetID == id {
					return ErrInteractionReferenced
				}
			}
		}
	}
	return nil
}
func normalizeInteractionState(s InteractionState) InteractionState {
	if s.SchemaVersion == 0 {
		s.SchemaVersion = 1
	}
	if s.Graphs == nil {
		s.Graphs = map[string]core.InteractionGraph{}
	}
	if s.Heads == nil {
		s.Heads = map[string]string{}
	}
	s.Graphs = cloneGraphs(s.Graphs)
	s.Heads = cloneInteractionStringMap(s.Heads)
	s.Operations = append([]InteractionOperationRecord(nil), s.Operations...)
	return s
}
func normalizeInteractionOp(op InteractionOperation) InteractionOperation {
	if op.SchemaVersion == 0 {
		op.SchemaVersion = 1
	}
	op.ID = strings.TrimSpace(op.ID)
	op.GraphID = strings.TrimSpace(op.GraphID)
	op.ExpectedHead = strings.TrimSpace(op.ExpectedHead)
	op.HistoryID = strings.TrimSpace(op.HistoryID)
	if op.Graph != nil {
		g := op.Graph.Normalize()
		op.Graph = &g
	}
	return op
}
func interactionTarget(op InteractionOperation, s InteractionState) (string, error) {
	switch op.Kind {
	case InteractionUpsert:
		if op.Graph == nil {
			return "", core.ErrInvalidInteraction
		}
		return "interaction:" + op.Graph.ID, nil
	case InteractionDelete:
		return "interaction:" + op.GraphID, nil
	case InteractionUndo, InteractionRedo:
		r := interactionRecord(s.Operations, op.HistoryID)
		if r == nil {
			return "", ErrInteractionNotFound
		}
		return r.TargetKey, nil
	}
	return "", core.ErrInvalidInteraction
}
func interactionRecord(rs []InteractionOperationRecord, id string) *InteractionOperationRecord {
	for i := range rs {
		if rs[i].ID == id {
			return &rs[i]
		}
	}
	return nil
}
func interactionRequestHash(op InteractionOperation) string {
	op.ExpectedHead = ""
	raw, _ := json.Marshal(op)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func interactionHead(id string, rev uint64) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", id, rev)))
	return hex.EncodeToString(sum[:])
}
func cloneGraphs(in map[string]core.InteractionGraph) map[string]core.InteractionGraph {
	raw, _ := json.Marshal(in)
	out := map[string]core.InteractionGraph{}
	_ = json.Unmarshal(raw, &out)
	return out
}
func cloneInteractionStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
