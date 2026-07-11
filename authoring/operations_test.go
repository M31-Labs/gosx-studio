package authoring

import (
	"testing"

	"m31labs.dev/gosx-studio/core"
)

func TestOperationTargetKeyIsStableAcrossOperationKinds(t *testing.T) {
	target := OperationTarget{Route: "https://example.test/about?gosx-preview=1#x/", PageID: "page:about", Field: "pages.about.title"}
	if target.Key(OperationSetField) != target.Key(OperationUndo) {
		t.Fatal("target key must identify target, not operation kind")
	}
	if target.Normalize().Route != "/about" {
		t.Fatalf("route=%q", target.Normalize().Route)
	}
}

func TestOperationInverseUsesResetForAbsentStyle(t *testing.T) {
	record := OperationRecord{ID: "op", Kind: OperationSetStyle, Target: OperationTarget{ComponentKey: "home:hero", Property: "color", Breakpoint: "mobile"}, TargetHead: "head", Before: OperationValue{Present: false}}
	request := record.InverseRequest("undo", "actor", 2)
	if request.Kind != OperationResetStyle || request.ExpectedTargetHead != "head" {
		t.Fatalf("inverse=%#v", request)
	}
}

func TestOperationRequestsEqualPreservesContentWhitespace(t *testing.T) {
	a := OperationRequest{ID: "a", Kind: OperationSetField, Target: OperationTarget{Field: "hero.headline"}, Value: "  headline  "}
	b := a
	b.ID = "b"
	if !RequestsEqual(a, b) {
		t.Fatal("canonical requests should match")
	}
	c := b
	c.Value = "headline"
	if RequestsEqual(a, c) {
		t.Fatal("content whitespace must not be discarded")
	}
}

func TestOperationTargetRootRouteNormalizationIsIdempotent(t *testing.T) {
	target := OperationTarget{Route: "/", Field: "title"}.Normalize()
	if target.Route != "/" || target.Normalize() != target {
		t.Fatalf("root normalization must be stable: %#v -> %#v", target, target.Normalize())
	}
}

// --- New durable operation families: set-shared-field / override-instance /
// detach-instance / restore-instance / set-interaction / remove-interaction /
// set-flow-field / remove-flow-field / set-flow-action ---

func validInteractionValue() string {
	return EncodeInteractionSettings(core.Interaction{
		Kind: core.InteractionRevealOnScroll, Effect: core.InteractionEffectFade, DurationMS: 250,
	})
}

func TestNewOperationKindsValidateRequiredTargets(t *testing.T) {
	cases := []struct {
		name    string
		request OperationRequest
		wantErr bool
	}{
		{"shared field ok", OperationRequest{ID: "a", Kind: OperationSetSharedField, Target: OperationTarget{ComponentKey: "hero-card", ControlKey: "title", Field: FieldInstancesSharedField}, Value: "hi"}, false},
		{"shared field missing control", OperationRequest{ID: "a", Kind: OperationSetSharedField, Target: OperationTarget{ComponentKey: "hero-card", Field: FieldInstancesSharedField}, Value: "hi"}, true},
		{"shared field wrong field literal", OperationRequest{ID: "a", Kind: OperationSetSharedField, Target: OperationTarget{ComponentKey: "hero-card", ControlKey: "title", Field: "instances.override"}, Value: "hi"}, true},

		{"override ok", OperationRequest{ID: "a", Kind: OperationOverrideInstance, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "title", Field: FieldInstancesOverride}, Value: "hi"}, false},
		{"override missing page", OperationRequest{ID: "a", Kind: OperationOverrideInstance, Target: OperationTarget{ComponentKey: "home:hero", ControlKey: "title", Field: FieldInstancesOverride}, Value: "hi"}, true},

		{"detach ok", OperationRequest{ID: "a", Kind: OperationDetachInstance, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", Field: FieldInstancesAttachment}}, false},
		{"restore ok", OperationRequest{ID: "a", Kind: OperationRestoreInstance, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", Field: FieldInstancesAttachment}}, false},
		{"detach missing component", OperationRequest{ID: "a", Kind: OperationDetachInstance, Target: OperationTarget{PageID: "home", Field: FieldInstancesAttachment}}, true},
		{"detach wrong field", OperationRequest{ID: "a", Kind: OperationDetachInstance, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", Field: FieldInstancesOverride}}, true},

		{"set interaction ok", OperationRequest{ID: "a", Kind: OperationSetInteraction, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "home:hero:reveal-on-scroll", Field: FieldInteractionsEntry}, Value: validInteractionValue()}, false},
		{"set interaction missing value", OperationRequest{ID: "a", Kind: OperationSetInteraction, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "k", Field: FieldInteractionsEntry}}, true},
		{"set interaction bad kind", OperationRequest{ID: "a", Kind: OperationSetInteraction, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "k", Field: FieldInteractionsEntry}, Value: EncodeInteractionSettings(core.Interaction{Kind: "not-a-kind"})}, true},
		{"remove interaction ok", OperationRequest{ID: "a", Kind: OperationRemoveInteraction, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "k", Field: FieldInteractionsEntry}}, false},

		{"set flow field ok", OperationRequest{ID: "a", Kind: OperationSetFlowField, Target: OperationTarget{PageID: "contact", ComponentKey: "submit", ControlKey: "email", Field: FieldFlowsField}, Value: EncodeFlowFieldSettings(core.FlowField{Label: "Email", Kind: core.ControlText, Required: true})}, false},
		{"set flow field bad kind", OperationRequest{ID: "a", Kind: OperationSetFlowField, Target: OperationTarget{PageID: "contact", ComponentKey: "submit", ControlKey: "email", Field: FieldFlowsField}, Value: EncodeFlowFieldSettings(core.FlowField{Kind: core.ControlRichText})}, true},
		{"remove flow field ok", OperationRequest{ID: "a", Kind: OperationRemoveFlowField, Target: OperationTarget{PageID: "contact", ComponentKey: "submit", ControlKey: "email", Field: FieldFlowsField}}, false},

		{"set flow action ok", OperationRequest{ID: "a", Kind: OperationSetFlowAction, Target: OperationTarget{PageID: "contact", ComponentKey: "submit", Field: FieldFlowsAction}, Value: EncodeFlowActionSettings("Submit", "flow.contact.submit")}, false},
		{"set flow action missing value", OperationRequest{ID: "a", Kind: OperationSetFlowAction, Target: OperationTarget{PageID: "contact", ComponentKey: "submit", Field: FieldFlowsAction}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.request.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSetKindForTargetRoundTripsEveryFamily(t *testing.T) {
	cases := []struct {
		name    string
		target  OperationTarget
		present bool
		want    OperationKind
	}{
		{"style set", OperationTarget{ComponentKey: "home:hero", Property: "color"}, true, OperationSetStyle},
		{"style reset", OperationTarget{ComponentKey: "home:hero", Property: "color"}, false, OperationResetStyle},
		{"content field", OperationTarget{Field: "hero.headline"}, true, OperationSetField},
		{"shared field", OperationTarget{Field: FieldInstancesSharedField}, true, OperationSetSharedField},
		{"shared field ignores present", OperationTarget{Field: FieldInstancesSharedField}, false, OperationSetSharedField},
		{"override instance", OperationTarget{Field: FieldInstancesOverride}, true, OperationOverrideInstance},
		{"attachment detach", OperationTarget{Field: FieldInstancesAttachment}, true, OperationDetachInstance},
		{"attachment restore", OperationTarget{Field: FieldInstancesAttachment}, false, OperationRestoreInstance},
		{"interaction set", OperationTarget{Field: FieldInteractionsEntry}, true, OperationSetInteraction},
		{"interaction remove", OperationTarget{Field: FieldInteractionsEntry}, false, OperationRemoveInteraction},
		{"flow field set", OperationTarget{Field: FieldFlowsField}, true, OperationSetFlowField},
		{"flow field remove", OperationTarget{Field: FieldFlowsField}, false, OperationRemoveFlowField},
		{"flow action", OperationTarget{Field: FieldFlowsAction}, true, OperationSetFlowAction},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SetKindForTarget(tc.target, tc.present); got != tc.want {
				t.Fatalf("SetKindForTarget(%#v, %v) = %s, want %s", tc.target, tc.present, got, tc.want)
			}
		})
	}
}

func TestClearsTargetMatchesPairedFamilies(t *testing.T) {
	clears := []OperationKind{OperationResetStyle, OperationRestoreInstance, OperationRemoveInteraction, OperationRemoveFlowField}
	for _, kind := range clears {
		if !ClearsTarget(kind) {
			t.Fatalf("%s should clear its target", kind)
		}
	}
	keeps := []OperationKind{OperationSetField, OperationSetStyle, OperationSetSharedField, OperationOverrideInstance, OperationDetachInstance, OperationSetInteraction, OperationSetFlowField, OperationSetFlowAction}
	for _, kind := range keeps {
		if ClearsTarget(kind) {
			t.Fatalf("%s should not clear its target", kind)
		}
	}
}

func TestInverseRequestPicksPairedFamilyMember(t *testing.T) {
	record := OperationRecord{ID: "op", Kind: OperationDetachInstance, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", Field: FieldInstancesAttachment}, TargetHead: "head", Before: OperationValue{Present: false}}
	request := record.InverseRequest("undo", "actor", 2)
	if request.Kind != OperationRestoreInstance || request.ExpectedTargetHead != "head" {
		t.Fatalf("inverse=%#v", request)
	}

	record2 := OperationRecord{ID: "op2", Kind: OperationRemoveInteraction, Target: OperationTarget{PageID: "home", ComponentKey: "home:hero", ControlKey: "k", Field: FieldInteractionsEntry}, TargetHead: "head2", Before: OperationValue{Present: true, Value: validInteractionValue()}}
	request2 := record2.InverseRequest("undo2", "actor", 3)
	if request2.Kind != OperationSetInteraction || request2.Value != validInteractionValue() {
		t.Fatalf("inverse2=%#v", request2)
	}

	record3 := OperationRecord{ID: "op3", Kind: OperationSetFlowAction, Target: OperationTarget{PageID: "contact", ComponentKey: "submit", Field: FieldFlowsAction}, TargetHead: "head3", Before: OperationValue{Present: true, Value: "old"}}
	request3 := record3.InverseRequest("undo3", "actor", 4)
	if request3.Kind != OperationSetFlowAction || request3.Value != "old" {
		t.Fatalf("inverse3=%#v", request3)
	}
}

func TestInteractionFlowSettingsEncodeDecodeRoundTrip(t *testing.T) {
	interaction := core.Interaction{Kind: core.InteractionHoverFocusState, Effect: core.InteractionEffectGlow, DurationMS: 400, DelayMS: 100, Once: true}
	encoded := EncodeInteractionSettings(interaction)
	settings, err := DecodeInteractionSettings(encoded)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt := settings.Interaction(OperationTarget{Route: "/", PageID: "home", ComponentKey: "home:hero", ControlKey: "home:hero:hover"})
	if rebuilt.Kind != core.InteractionHoverFocusState || rebuilt.Effect != core.InteractionEffectGlow || rebuilt.DurationMS != 400 || !rebuilt.Once {
		t.Fatalf("rebuilt=%#v", rebuilt)
	}
	if _, err := DecodeInteractionSettings(""); err == nil {
		t.Fatal("empty interaction settings must be rejected")
	}
	if _, err := DecodeInteractionSettings("{not json"); err == nil {
		t.Fatal("malformed interaction settings must be rejected")
	}

	field := core.FlowField{Label: "Email address", Kind: core.ControlText, Required: true}
	encodedField := EncodeFlowFieldSettings(field)
	fieldSettings, err := DecodeFlowFieldSettings(encodedField)
	if err != nil {
		t.Fatal(err)
	}
	rebuiltField := fieldSettings.FlowField(OperationTarget{PageID: "contact", ComponentKey: "submit", ControlKey: "email"})
	if rebuiltField.Name != "email" || rebuiltField.Label != "Email address" || rebuiltField.Kind != core.ControlText || !rebuiltField.Required {
		t.Fatalf("rebuiltField=%#v", rebuiltField)
	}
	if _, err := DecodeFlowFieldSettings(""); err == nil {
		t.Fatal("empty flow field settings must be rejected")
	}

	encodedAction := EncodeFlowActionSettings("Submit", "flow.contact.submit")
	actionSettings, err := DecodeFlowActionSettings(encodedAction)
	if err != nil {
		t.Fatal(err)
	}
	if actionSettings.Label != "Submit" || actionSettings.HandlerRef != "flow.contact.submit" {
		t.Fatalf("actionSettings=%#v", actionSettings)
	}
	if _, err := DecodeFlowActionSettings(""); err == nil {
		t.Fatal("empty flow action settings must be rejected")
	}
}
