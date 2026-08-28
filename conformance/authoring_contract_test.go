package conformance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
	"m31labs.dev/gosx/action"
)

// contractAuthoringAdapter records calls so the action boundary can prove a
// rejected mutation never reaches host persistence. A real host adapter is
// deliberately not needed for this contract: the handler's validation must
// run before any adapter implementation is invoked.
type contractAuthoringAdapter struct {
	calls int
}

func (a *contractAuthoringAdapter) ApplyAuthoringMutation(context.Context, authoring.AuthoringMutation) (authoring.AuthoringMutationResult, error) {
	a.calls++
	return authoring.AuthoringMutationResult{Message: "unexpected adapter call"}, nil
}

func TestAuthoringActionContractRejectsMalformedMutationsBeforeAdapter(t *testing.T) {
	cases := []struct {
		name  string
		form  url.Values
		field string
	}{
		{
			name:  "unknown operation kind",
			form:  url.Values{authoring.AuthoringFieldOperation: {"not-a-real-operation"}},
			field: authoring.AuthoringFieldOperation,
		},
		{
			name: "missing field target",
			form: url.Values{
				authoring.AuthoringFieldOperation: {string(authoring.AuthoringOperationSetField)},
			},
			field: authoring.AuthoringFieldPageKey,
		},
		{
			name: "unsupported style property",
			form: url.Values{
				authoring.AuthoringFieldOperation:     {string(authoring.AuthoringOperationSetStyle)},
				authoring.AuthoringFieldPageKey:       {"home"},
				authoring.AuthoringFieldComponentKey:  {"home:hero"},
				authoring.AuthoringFieldStyleProperty: {"behavior"},
				authoring.AuthoringFieldStyleValue:    {"block"},
			},
			field: authoring.AuthoringFieldStyleProperty,
		},
		{
			name: "unsafe style value",
			form: url.Values{
				authoring.AuthoringFieldOperation:     {string(authoring.AuthoringOperationSetStyle)},
				authoring.AuthoringFieldPageKey:       {"home"},
				authoring.AuthoringFieldComponentKey:  {"home:hero"},
				authoring.AuthoringFieldStyleProperty: {"color"},
				authoring.AuthoringFieldStyleValue:    {"red;position:fixed"},
			},
			field: authoring.AuthoringFieldStyleValue,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &contractAuthoringAdapter{}
			req := httptest.NewRequest(http.MethodPost, "/gosx/action/authoring", strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("Accept", "application/json")
			rec := httptest.NewRecorder()

			action.ServeHandler(rec, req, authoring.AuthoringActionHandler(adapter))

			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
			}
			if adapter.calls != 0 {
				t.Fatalf("malformed %s reached adapter %d time(s)", tc.name, adapter.calls)
			}
			var result action.Result
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatalf("decode validation result: %v", err)
			}
			if result.OK || result.FieldErrors[tc.field] == "" {
				t.Fatalf("expected actionable %s error, got %#v", tc.field, result)
			}
		})
	}
}

func TestOperationRequestContractRejectsMalformedTargetsAndStyles(t *testing.T) {
	validStyleTarget := authoring.OperationTarget{Route: "/", PageID: "home", ComponentKey: "home:hero", Property: "color"}
	cases := []struct {
		name    string
		request authoring.OperationRequest
	}{
		{
			name:    "unknown kind",
			request: authoring.OperationRequest{ID: "unknown", Kind: authoring.OperationKind("not-a-real-operation"), Target: authoring.OperationTarget{Field: "title"}},
		},
		{
			name:    "missing field target",
			request: authoring.OperationRequest{ID: "missing-field", Kind: authoring.OperationSetField},
		},
		{
			name:    "field cannot carry style property",
			request: authoring.OperationRequest{ID: "field-style", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{Field: "title", Property: "color"}},
		},
		{
			name:    "missing style target",
			request: authoring.OperationRequest{ID: "missing-style", Kind: authoring.OperationSetStyle, Value: "red"},
		},
		{
			name: "unsupported style property",
			request: authoring.OperationRequest{
				ID: "unsupported-style", Kind: authoring.OperationSetStyle,
				Target: authoring.OperationTarget{ComponentKey: "home:hero", Property: "behavior"}, Value: "block",
			},
		},
		{
			name:    "style cannot carry content field",
			request: authoring.OperationRequest{ID: "style-field", Kind: authoring.OperationSetStyle, Target: authoring.OperationTarget{Field: "title", ComponentKey: "home:hero", Property: "color"}, Value: "red"},
		},
		{
			name:    "unsafe style value",
			request: authoring.OperationRequest{ID: "unsafe-style", Kind: authoring.OperationSetStyle, Target: validStyleTarget, Value: "red;position:fixed"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.request.Validate(); err == nil {
				t.Fatalf("expected malformed request to be rejected: %#v", tc.request)
			}
		})
	}
}

func TestOperationRequestContractPreservesValidControlInstanceInteractionAndFlowKinds(t *testing.T) {
	validInteraction := authoring.EncodeInteractionSettings(core.Interaction{
		Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFade, DurationMS: 250,
	})
	validFlowField := authoring.EncodeFlowFieldSettings(core.FlowField{
		Name: "email", Label: "Email", Kind: core.ControlText, Required: true,
	})
	validFlowAction := authoring.EncodeFlowActionSettings("Submit", "flow.contact.submit")
	cases := []struct {
		name    string
		request authoring.OperationRequest
	}{
		{
			name:    "content control",
			request: authoring.OperationRequest{ID: "field", Kind: authoring.OperationSetField, Target: authoring.OperationTarget{PageID: "home", Field: "hero.headline"}, Value: "Hello"},
		},
		{
			name:    "style control",
			request: authoring.OperationRequest{ID: "style", Kind: authoring.OperationSetStyle, Target: authoring.OperationTarget{PageID: "home", ComponentKey: "home:hero", Property: "color"}, Value: "#111111"},
		},
		{
			name:    "shared instance control",
			request: authoring.OperationRequest{ID: "shared", Kind: authoring.OperationSetSharedField, Target: authoring.OperationTarget{ComponentKey: "hero-card", ControlKey: "title", Field: authoring.FieldInstancesSharedField}, Value: "Hello"},
		},
		{
			name:    "instance override",
			request: authoring.OperationRequest{ID: "override", Kind: authoring.OperationOverrideInstance, Target: authoring.OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "title", Field: authoring.FieldInstancesOverride}, Value: "Hello"},
		},
		{
			name:    "instance detach",
			request: authoring.OperationRequest{ID: "detach", Kind: authoring.OperationDetachInstance, Target: authoring.OperationTarget{PageID: "home", ComponentKey: "home:hero", Field: authoring.FieldInstancesAttachment}},
		},
		{
			name:    "instance restore",
			request: authoring.OperationRequest{ID: "restore", Kind: authoring.OperationRestoreInstance, Target: authoring.OperationTarget{PageID: "home", ComponentKey: "home:hero", Field: authoring.FieldInstancesAttachment}},
		},
		{
			name:    "interaction",
			request: authoring.OperationRequest{ID: "interaction", Kind: authoring.OperationSetInteraction, Target: authoring.OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "home:hero:reveal", Field: authoring.FieldInteractionsEntry}, Value: validInteraction},
		},
		{
			name:    "flow field",
			request: authoring.OperationRequest{ID: "flow-field", Kind: authoring.OperationSetFlowField, Target: authoring.OperationTarget{PageID: "contact", ComponentKey: "submit", ControlKey: "email", Field: authoring.FieldFlowsField}, Value: validFlowField},
		},
		{
			name:    "flow action",
			request: authoring.OperationRequest{ID: "flow-action", Kind: authoring.OperationSetFlowAction, Target: authoring.OperationTarget{PageID: "contact", ComponentKey: "submit", Field: authoring.FieldFlowsAction}, Value: validFlowAction},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.request.Validate(); err != nil {
				t.Fatalf("valid operation rejected: %v", err)
			}
		})
	}
}
