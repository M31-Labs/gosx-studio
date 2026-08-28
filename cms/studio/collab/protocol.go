package collab

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

const (
	ProtocolName    = "gosx.studio.collaboration"
	ProtocolVersion = 1
	MaxFrameBytes   = 64 << 10
)

var protocolMessageTypes = map[string]bool{
	"studio.hello": true, "studio.snapshot": true, "studio.presence": true,
	"studio.cursors": true, "studio.selection": true,
	"studio.operation.submit": true, "studio.operation.accepted": true,
	"studio.operation.rejected": true, "studio.sync.resume": true,
	"studio.sync.tail": true,
}

type ErrorCode string

const (
	ErrorInvalidRequest    ErrorCode = "invalid_request"
	ErrorForbidden         ErrorCode = "forbidden"
	ErrorStaleField        ErrorCode = "stale_field"
	ErrorOperationIDReuse  ErrorCode = "operation_id_reuse"
	ErrorHistoryNotFound   ErrorCode = "history_not_found"
	ErrorHistoryActor      ErrorCode = "history_actor_mismatch"
	ErrorBinaryUnsupported ErrorCode = "binary_unsupported"
	ErrorStoreUnavailable  ErrorCode = "store_unavailable"
)

type Envelope struct {
	Protocol string          `json:"protocol"`
	Version  int             `json:"version"`
	Type     string          `json:"type"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

func (e Envelope) Validate() *ProtocolError {
	if e.Protocol != ProtocolName || e.Version != ProtocolVersion {
		return NewProtocolError(ErrorInvalidRequest, "unsupported collaboration protocol")
	}
	if strings.TrimSpace(e.Type) == "" {
		return NewProtocolError(ErrorInvalidRequest, "message type is required")
	}
	if !protocolMessageTypes[e.Type] {
		return NewProtocolError(ErrorInvalidRequest, "unknown collaboration message type")
	}
	if len(e.Payload) > MaxFrameBytes {
		return NewProtocolError(ErrorInvalidRequest, "frame exceeds maximum size")
	}
	return nil
}

func DecodeEnvelope(frame []byte, binary bool) (Envelope, *ProtocolError) {
	if binary {
		return Envelope{}, NewProtocolError(ErrorBinaryUnsupported, "binary collaboration frames are not supported")
	}
	if len(frame) == 0 || len(frame) > MaxFrameBytes {
		return Envelope{}, NewProtocolError(ErrorInvalidRequest, "frame is empty or exceeds maximum size")
	}
	decoder := json.NewDecoder(bytes.NewReader(frame))
	decoder.DisallowUnknownFields()
	var envelope Envelope
	if err := decoder.Decode(&envelope); err != nil {
		return Envelope{}, NewProtocolError(ErrorInvalidRequest, "malformed collaboration frame")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Envelope{}, NewProtocolError(ErrorInvalidRequest, "collaboration frame must contain one object")
	}
	if protocolErr := envelope.Validate(); protocolErr != nil {
		return Envelope{}, protocolErr
	}
	return envelope, nil
}

type OperationSubmit struct {
	Request authoring.OperationRequest `json:"request"`
}

func DecodeOperationSubmit(payload []byte) (OperationSubmit, *ProtocolError) {
	if len(payload) == 0 || len(payload) > MaxFrameBytes {
		return OperationSubmit{}, NewProtocolError(ErrorInvalidRequest, "operation payload is empty or exceeds maximum size")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var submit OperationSubmit
	if err := decoder.Decode(&submit); err != nil {
		return OperationSubmit{}, NewProtocolError(ErrorInvalidRequest, "malformed operation payload")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return OperationSubmit{}, NewProtocolError(ErrorInvalidRequest, "operation payload must contain one object")
	}
	if protocolErr := submit.Validate(); protocolErr != nil {
		return OperationSubmit{}, protocolErr
	}
	return submit, nil
}

func (s OperationSubmit) Validate() *ProtocolError {
	raw, err := json.Marshal(s)
	if err != nil || len(raw) > MaxFrameBytes {
		return NewProtocolError(ErrorInvalidRequest, "operation payload exceeds maximum size")
	}
	if err := s.Request.Validate(); err != nil {
		return NewProtocolError(ErrorInvalidRequest, err.Error())
	}
	target := s.Request.Normalize().Target
	identity := core.CanvasIdentity{
		Route: target.Route, PageID: target.PageID, Field: target.Field,
		BlockKey: target.ComponentKey, NodeID: target.NodeID,
	}.Normalize()
	if !identity.Stable() {
		return NewProtocolError(ErrorInvalidRequest, "canonical canvas locator is required")
	}
	return nil
}

type ProtocolError struct {
	Code           ErrorCode                 `json:"code"`
	Message        string                    `json:"message"`
	OperationID    string                    `json:"operationId,omitempty"`
	CurrentHead    string                    `json:"currentHead,omitempty"`
	CurrentValue   *authoring.OperationValue `json:"currentValue,omitempty"`
	SubmittedValue *authoring.OperationValue `json:"submittedValue,omitempty"`
	Retryable      bool                      `json:"retryable,omitempty"`
}

func NewProtocolError(code ErrorCode, message string) *ProtocolError {
	return &ProtocolError{Code: code, Message: strings.TrimSpace(message)}
}

func (e *ProtocolError) Error() string {
	if e == nil {
		return ""
	}
	return string(e.Code) + ": " + e.Message
}

type OperationAck struct {
	Record     authoring.OperationRecord `json:"record"`
	Sequence   uint64                    `json:"sequence"`
	Idempotent bool                      `json:"idempotent,omitempty"`
}

type ResumeRequest struct {
	AfterSequence uint64 `json:"afterSequence"`
}

type SelectionState struct {
	Route    string `json:"route"`
	PageID   string `json:"pageId,omitempty"`
	Field    string `json:"field,omitempty"`
	BlockKey string `json:"blockKey,omitempty"`
	NodeID   string `json:"nodeId,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Viewport string `json:"viewport,omitempty"`
}

func SelectionFromIdentity(identity core.CanvasIdentity, viewport string) SelectionState {
	identity = identity.Normalize()
	return SelectionState{Route: identity.Route, PageID: identity.PageID, Field: identity.Field, BlockKey: identity.BlockKey, NodeID: identity.NodeID, Kind: identity.Kind, Viewport: strings.TrimSpace(viewport)}
}

func (s SelectionState) Identity() core.CanvasIdentity {
	return core.CanvasIdentity{Route: s.Route, PageID: s.PageID, Field: s.Field, BlockKey: s.BlockKey, NodeID: s.NodeID, Kind: s.Kind}.Normalize()
}

func (s SelectionState) Normalize() SelectionState {
	return SelectionFromIdentity(s.Identity(), s.Viewport)
}

func (s SelectionState) Validate() *ProtocolError {
	if !s.Identity().Stable() {
		return NewProtocolError(ErrorInvalidRequest, "stable shared selection identity is required")
	}
	if len(s.Route) > 2048 || len(s.PageID) > 512 || len(s.Field) > 512 || len(s.BlockKey) > 512 || len(s.NodeID) > 512 {
		return NewProtocolError(ErrorInvalidRequest, "shared selection identity is too long")
	}
	if len(s.Viewport) > 64 {
		return NewProtocolError(ErrorInvalidRequest, "selection viewport is too long")
	}
	return nil
}

type CursorState struct {
	Route    string  `json:"route"`
	Viewport string  `json:"viewport,omitempty"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

func (c CursorState) Normalize() CursorState {
	c.Route = authoring.OperationTarget{Route: c.Route}.Normalize().Route
	c.Viewport = strings.TrimSpace(c.Viewport)
	if c.X < 0 {
		c.X = 0
	}
	if c.X > 1 {
		c.X = 1
	}
	if c.Y < 0 {
		c.Y = 0
	}
	if c.Y > 1 {
		c.Y = 1
	}
	return c
}

func (c CursorState) Validate() *ProtocolError {
	c = c.Normalize()
	if len(c.Route) > 2048 || len(c.Viewport) > 64 {
		return NewProtocolError(ErrorInvalidRequest, "cursor scope is too long")
	}
	return nil
}
