package core

import "testing"

func testSharedDefinitions() []ComponentDefinition {
	return []ComponentDefinition{
		{
			Key:           "testimonial",
			Label:         "Testimonial",
			GoSXComponent: "Testimonial",
			Controls: []Control{
				{Key: "quote", Label: "Quote", Kind: ControlRichText, Value: "Original quote"},
				{Key: "attribution", Label: "Attribution", Kind: ControlText, Value: "Original author"},
			},
		},
	}
}

func TestEffectiveComponentControlsFollowsSharedDefinitionByDefault(t *testing.T) {
	definitions := testSharedDefinitions()
	instanceA := Component{Key: "testimonial-1", DefinitionKey: "testimonial"}
	instanceB := Component{Key: "testimonial-2", DefinitionKey: "testimonial"}

	controlsA := EffectiveComponentControls(instanceA, definitions)
	controlsB := EffectiveComponentControls(instanceB, definitions)
	if len(controlsA) != 2 || controlsA[0].Value != "Original quote" {
		t.Fatalf("unexpected effective controls for instance A: %#v", controlsA)
	}
	if len(controlsB) != 2 || controlsB[0].Value != "Original quote" {
		t.Fatalf("unexpected effective controls for instance B: %#v", controlsB)
	}

	// Editing the shared definition (as if a "set-shared-field" mutation had
	// just run) must be visible to BOTH instances without any per-instance
	// write, because neither instance has diverged.
	definitions[0].Controls[0].Value = "Updated quote"
	controlsA = EffectiveComponentControls(instanceA, definitions)
	controlsB = EffectiveComponentControls(instanceB, definitions)
	if controlsA[0].Value != "Updated quote" || controlsB[0].Value != "Updated quote" {
		t.Fatalf("expected both instances to reflect the shared edit, got A=%q B=%q", controlsA[0].Value, controlsB[0].Value)
	}
}

func TestOverrideComponentControlDivergesOnlyThatInstance(t *testing.T) {
	definitions := testSharedDefinitions()
	instanceA := Component{Key: "testimonial-1", DefinitionKey: "testimonial"}
	instanceB := Component{Key: "testimonial-2", DefinitionKey: "testimonial"}

	instanceA = OverrideComponentControl(instanceA, "quote", "Overridden quote for A only")

	controlsA := EffectiveComponentControls(instanceA, definitions)
	controlsB := EffectiveComponentControls(instanceB, definitions)
	if controlsA[0].Value != "Overridden quote for A only" {
		t.Fatalf("expected instance A's quote override, got %#v", controlsA[0])
	}
	if controlsA[1].Value != "Original author" {
		t.Fatalf("expected instance A's attribution to keep following the shared definition, got %#v", controlsA[1])
	}
	if controlsB[0].Value != "Original quote" {
		t.Fatalf("expected instance B to be unaffected by A's override, got %#v", controlsB[0])
	}
	if !instanceA.IsSharedInstance() {
		t.Fatal("expected an overridden (not detached) instance to remain a shared instance")
	}
	keys := instanceA.OverriddenControlKeys()
	if len(keys) != 1 || keys[0] != "quote" {
		t.Fatalf("unexpected overridden control keys: %#v", keys)
	}

	// A later definition edit still reaches the un-overridden attribution
	// field on the overridden instance.
	definitions[0].Controls[1].Value = "Updated author"
	controlsA = EffectiveComponentControls(instanceA, definitions)
	if controlsA[1].Value != "Updated author" {
		t.Fatalf("expected un-overridden field to keep following shared edits, got %#v", controlsA[1])
	}
}

func TestDetachComponentInstanceFreezesEffectiveValuesAndStopsFollowing(t *testing.T) {
	definitions := testSharedDefinitions()
	instance := Component{Key: "testimonial-1", DefinitionKey: "testimonial"}
	instance = OverrideComponentControl(instance, "quote", "Frozen at detach time")

	detached := DetachComponentInstance(instance, definitions)
	if !detached.Detached {
		t.Fatal("expected Detached to be true after detaching")
	}
	if detached.IsSharedInstance() {
		t.Fatal("a detached instance must not report as a shared instance")
	}
	if len(detached.Overrides) != 0 {
		t.Fatalf("expected overrides to be cleared on detach, got %#v", detached.Overrides)
	}
	if len(detached.Controls) != 2 || detached.Controls[0].Value != "Frozen at detach time" {
		t.Fatalf("expected detach to freeze the effective controls, got %#v", detached.Controls)
	}

	// Further shared definition edits must NOT reach a detached instance.
	definitions[0].Controls[0].Value = "Shared edit after detach"
	stillFrozen := EffectiveComponentControls(detached, definitions)
	if stillFrozen[0].Value != "Frozen at detach time" {
		t.Fatalf("expected detached instance to ignore further shared edits, got %#v", stillFrozen[0])
	}

	// Detaching an already-standalone/detached component is a no-op.
	same := DetachComponentInstance(detached, definitions)
	if same.Detached != detached.Detached || len(same.Controls) != len(detached.Controls) {
		t.Fatalf("expected detaching an already-detached instance to be a no-op, got %#v", same)
	}
}

func TestRestoreComponentInstanceReattachesAndDropsLocalState(t *testing.T) {
	definitions := testSharedDefinitions()
	instance := Component{Key: "testimonial-1", DefinitionKey: "testimonial"}
	instance = OverrideComponentControl(instance, "quote", "Diverged value")
	detached := DetachComponentInstance(instance, definitions)

	restored := RestoreComponentInstance(detached)
	if restored.Detached {
		t.Fatal("expected restore to clear Detached")
	}
	if len(restored.Overrides) != 0 {
		t.Fatalf("expected restore to clear overrides, got %#v", restored.Overrides)
	}
	if !restored.IsSharedInstance() {
		t.Fatal("expected restored instance to be a shared instance again")
	}
	controls := EffectiveComponentControls(restored, definitions)
	if controls[0].Value != "Original quote" {
		t.Fatalf("expected restored instance to mirror the shared definition, got %#v", controls[0])
	}

	// Restoring a component with no DefinitionKey is a no-op.
	standalone := Component{Key: "standalone"}
	if got := RestoreComponentInstance(standalone); got.NormalizedDefinitionKey() != "" {
		t.Fatalf("expected restoring a standalone component to be a no-op, got %#v", got)
	}
}

func TestEffectiveComponentControlsFallsBackWhenDefinitionMissing(t *testing.T) {
	instance := Component{
		Key:           "orphaned",
		DefinitionKey: "missing-definition",
		Controls:      []Control{{Key: "quote", Label: "Quote", Value: "own copy"}},
	}
	controls := EffectiveComponentControls(instance, nil)
	if len(controls) != 1 || controls[0].Value != "own copy" {
		t.Fatalf("expected fallback to the instance's own controls, got %#v", controls)
	}
}

func TestComponentDefinitionByKeyNormalizesAndTrims(t *testing.T) {
	definitions := []ComponentDefinition{{Key: " testimonial ", Label: " Testimonial "}}
	found, ok := ComponentDefinitionByKey(definitions, "testimonial")
	if !ok || found.Label != "Testimonial" {
		t.Fatalf("expected to find and normalize the definition, got %#v ok=%v", found, ok)
	}
	if _, ok := ComponentDefinitionByKey(definitions, "  "); ok {
		t.Fatal("expected an empty key to never match")
	}
}

func TestPaletteEntriesCombinesTemplatesAndSharedDefinitions(t *testing.T) {
	library := CompositionLibrary{
		ComponentTemplates: []ComponentTemplate{
			{Key: "hero", Label: "Hero", GoSXComponent: "Hero", AddLabel: "Add hero"},
		},
		ComponentDefinitions: []ComponentDefinition{
			{Key: "testimonial", Label: "Testimonial", GoSXComponent: "Testimonial"},
		},
	}
	entries := library.PaletteEntries()
	if len(entries) != 2 {
		t.Fatalf("expected two palette entries, got %#v", entries)
	}
	if entries[0].IsShared || entries[0].AddLabel != "Add hero" {
		t.Fatalf("unexpected template entry: %#v", entries[0])
	}
	if !entries[1].IsShared || entries[1].DefinitionKey != "testimonial" || entries[1].Key != "shared:testimonial" {
		t.Fatalf("unexpected shared entry: %#v", entries[1])
	}
	if library.DefinitionCount() != 1 {
		t.Fatalf("expected DefinitionCount to be 1, got %d", library.DefinitionCount())
	}
}

func TestComponentOverrideValueAndClear(t *testing.T) {
	component := Component{Key: "instance"}
	component = OverrideComponentControl(component, "quote", "first")
	component = OverrideComponentControl(component, "quote", "second")
	if len(component.Overrides) != 1 {
		t.Fatalf("expected overriding the same control key to replace, not append, got %#v", component.Overrides)
	}
	value, ok := component.OverrideValue("quote")
	if !ok || value != "second" {
		t.Fatalf("expected the latest override value, got %q ok=%v", value, ok)
	}
	component = ClearComponentOverride(component, "quote")
	if _, ok := component.OverrideValue("quote"); ok {
		t.Fatal("expected the override to be cleared")
	}
}
