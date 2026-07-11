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
)

const OperationSchemaVersion = 1

type OperationKind string

const (
	OperationSetField   OperationKind = "set-field"
	OperationSetStyle   OperationKind = "set-style"
	OperationResetStyle OperationKind = "reset-style"
	OperationUndo       OperationKind = "undo"
	OperationRedo       OperationKind = "redo"
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
	HistoryOperationID       string          `json:"historyOperationId,omitempty"`
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
	default:
		return fmt.Errorf("unsupported operation kind %q", r.Kind)
	}
	return nil
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
	kind := OperationSetField
	if r.Target.Property != "" {
		kind = OperationSetStyle
	}
	if !r.Before.Present {
		if r.Target.Property != "" {
			kind = OperationResetStyle
		}
	}
	return OperationRequest{SchemaVersion: OperationSchemaVersion, ID: id, Kind: kind, Target: r.Target, Value: r.Before.Value, ExpectedDocumentRevision: revision, ExpectedTargetHead: r.TargetHead, HistoryOperationID: r.ID}.Normalize()
}

func (r OperationRecord) CanonicalRequest() OperationRequest {
	kind := r.Kind
	if kind == OperationUndo || kind == OperationRedo {
		kind = OperationSetField
	}
	return OperationRequest{SchemaVersion: r.SchemaVersion, ID: r.ID, Kind: kind, Target: r.Target, Value: r.After.Value}.Normalize()
}

func RequestsEqual(a, b OperationRequest) bool {
	a, b = a.Normalize(), b.Normalize()
	a.ID = ""
	b.ID = ""
	a.ExpectedDocumentRevision = 0
	b.ExpectedDocumentRevision = 0
	a.ExpectedTargetHead, b.ExpectedTargetHead = "", ""
	return canonicalJSON(a) == canonicalJSON(b)
}

func canonicalJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

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
