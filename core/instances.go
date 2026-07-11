package core

import "strings"

// ComponentDefinition is the shared "master" record backing a reusable
// component. A page's placed Component becomes a shared instance of one
// ComponentDefinition by setting Component.DefinitionKey; every attached
// instance (DefinitionKey set, Detached false) resolves its Controls from the
// definition through EffectiveComponentControls. Editing a definition's
// Controls therefore fans out to every attached instance in a single save —
// no per-instance write is required — while an instance can still diverge
// one field at a time with an override, or fully diverge with Detach.
type ComponentDefinition struct {
	Key           string
	Label         string
	Summary       string
	Category      string
	GoSXComponent string
	Source        ComponentSource
	Controls      []Control
}

// ComponentOverride is one instance-scoped control override. Only controls an
// editor has explicitly overridden appear here; every other control on an
// attached instance keeps resolving from the shared definition.
type ComponentOverride struct {
	ControlKey string
	Value      string
}

func (definition ComponentDefinition) Normalize() ComponentDefinition {
	definition.Key = strings.TrimSpace(definition.Key)
	definition.Label = strings.TrimSpace(definition.Label)
	definition.Summary = strings.TrimSpace(definition.Summary)
	definition.Category = strings.TrimSpace(definition.Category)
	definition.GoSXComponent = strings.TrimSpace(definition.GoSXComponent)
	definition.Source = normalizeComponentSource(definition.Source)
	definition.Controls = normalizeControls(definition.Controls)
	return definition
}

func (definition ComponentDefinition) ControlCount() int {
	return len(definition.Controls)
}

// ControlByKey returns the shared definition's control with the given key.
func (definition ComponentDefinition) ControlByKey(key string) (Control, bool) {
	key = strings.TrimSpace(key)
	for _, control := range definition.Controls {
		if control.Key == key {
			return control, true
		}
	}
	return Control{}, false
}

func (override ComponentOverride) Normalize() ComponentOverride {
	override.ControlKey = strings.TrimSpace(override.ControlKey)
	override.Value = strings.TrimSpace(override.Value)
	return override
}

func normalizeComponentDefinitions(values []ComponentDefinition) []ComponentDefinition {
	out := make([]ComponentDefinition, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

// normalizeComponentOverrides trims every override and de-duplicates by
// ControlKey, keeping the LAST occurrence — callers build the override list by
// appending the newest edit, so the last entry for a key is authoritative.
func normalizeComponentOverrides(values []ComponentOverride) []ComponentOverride {
	byKey := map[string]ComponentOverride{}
	order := make([]string, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.ControlKey == "" {
			continue
		}
		if _, exists := byKey[value.ControlKey]; !exists {
			order = append(order, value.ControlKey)
		}
		byKey[value.ControlKey] = value
	}
	out := make([]ComponentOverride, 0, len(order))
	for _, key := range order {
		out = append(out, byKey[key])
	}
	return out
}

// ComponentDefinitionByKey finds one shared definition by key from a
// (typically CompositionLibrary.ComponentDefinitions) slice.
func ComponentDefinitionByKey(definitions []ComponentDefinition, key string) (ComponentDefinition, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return ComponentDefinition{}, false
	}
	for _, definition := range definitions {
		definition = definition.Normalize()
		if definition.Key == key {
			return definition, true
		}
	}
	return ComponentDefinition{}, false
}

// NormalizedDefinitionKey trims Component.DefinitionKey.
func (component Component) NormalizedDefinitionKey() string {
	return strings.TrimSpace(component.DefinitionKey)
}

// IsSharedInstance reports whether this placed component still follows a
// shared definition (as opposed to being a standalone component or a
// previously-detached, now-independent instance).
func (component Component) IsSharedInstance() bool {
	return component.NormalizedDefinitionKey() != "" && !component.Detached
}

// OverrideValue returns the instance-local override value for one control key
// and whether an override is set for it.
func (component Component) OverrideValue(controlKey string) (string, bool) {
	controlKey = strings.TrimSpace(controlKey)
	if controlKey == "" {
		return "", false
	}
	for _, override := range normalizeComponentOverrides(component.Overrides) {
		if override.ControlKey == controlKey {
			return override.Value, true
		}
	}
	return "", false
}

// OverriddenControlKeys lists the control keys this instance has diverged
// from its shared definition, in a stable order. Panels use it to mark which
// fields carry an "override" badge.
func (component Component) OverriddenControlKeys() []string {
	overrides := normalizeComponentOverrides(component.Overrides)
	out := make([]string, 0, len(overrides))
	for _, override := range overrides {
		out = append(out, override.ControlKey)
	}
	return out
}

// EffectiveComponentControls resolves the controls a canvas/inspector should
// render and save against for one placed component:
//   - a standalone component (no DefinitionKey) or a Detached instance
//     resolves from its own Controls;
//   - an attached shared instance resolves from its definition's Controls,
//     with any per-instance Overrides layered on top control-by-control.
//
// If the referenced definition cannot be found (e.g. it was removed), the
// instance's own Controls are returned as a safe fallback rather than an
// empty list, so the canvas never renders a blank component.
func EffectiveComponentControls(component Component, definitions []ComponentDefinition) []Control {
	component = component.Normalize()
	if !component.IsSharedInstance() {
		return component.Controls
	}
	definition, ok := ComponentDefinitionByKey(definitions, component.DefinitionKey)
	if !ok {
		return component.Controls
	}
	out := make([]Control, 0, len(definition.Controls))
	for _, control := range definition.Controls {
		if value, overridden := component.OverrideValue(control.Key); overridden {
			control.Value = value
		}
		out = append(out, control)
	}
	return out
}

// OverrideComponentControl sets (or replaces) one instance-local control
// override. The instance stays attached to its shared definition — every
// other control keeps following shared edits; only controlKey diverges.
func OverrideComponentControl(component Component, controlKey, value string) Component {
	component = component.Normalize()
	controlKey = strings.TrimSpace(controlKey)
	if controlKey == "" {
		return component
	}
	overrides := append(append([]ComponentOverride{}, component.Overrides...), ComponentOverride{ControlKey: controlKey, Value: value})
	component.Overrides = normalizeComponentOverrides(overrides)
	return component
}

// ClearComponentOverride removes one instance-local override, so that
// control resolves from the shared definition again.
func ClearComponentOverride(component Component, controlKey string) Component {
	component = component.Normalize()
	controlKey = strings.TrimSpace(controlKey)
	out := make([]ComponentOverride, 0, len(component.Overrides))
	for _, override := range normalizeComponentOverrides(component.Overrides) {
		if override.ControlKey == controlKey {
			continue
		}
		out = append(out, override)
	}
	component.Overrides = out
	return component
}

// DetachComponentInstance freezes the instance's currently-effective control
// values as its own independent copy and severs it from further shared
// definition edits. DefinitionKey is preserved (it records provenance and
// lets RestoreComponentInstance re-attach later); Overrides are cleared
// because they are now baked into Controls.
//
// Detaching a component that is not a shared instance (no DefinitionKey, or
// already Detached) is a no-op: it returns the component unchanged.
func DetachComponentInstance(component Component, definitions []ComponentDefinition) Component {
	component = component.Normalize()
	if component.NormalizedDefinitionKey() == "" || component.Detached {
		return component
	}
	component.Controls = EffectiveComponentControls(component, definitions)
	component.Overrides = nil
	component.Detached = true
	return component
}

// RestoreComponentInstance re-attaches a detached (or override-diverged)
// instance to its shared definition, dropping any instance-local overrides
// and the frozen Controls copy so it once again mirrors the definition
// exactly. It is a no-op when the component has no DefinitionKey to restore
// to.
func RestoreComponentInstance(component Component) Component {
	component = component.Normalize()
	if component.NormalizedDefinitionKey() == "" {
		return component
	}
	component.Detached = false
	component.Overrides = nil
	return component
}

// PaletteEntry is one placeable palette item: either a fresh ComponentTemplate
// or an existing reusable ComponentDefinition an operator can place another
// instance of. IsShared/DefinitionKey distinguish the two so the palette
// (and the AddComponent/AddSharedInstance intents it emits) can route
// correctly.
type PaletteEntry struct {
	Key           string
	Label         string
	Summary       string
	Category      string
	GoSXComponent string
	Source        ComponentSource
	Status        string
	AddLabel      string
	IsShared      bool
	DefinitionKey string
}

// PaletteEntries returns the unified, placeable palette: every fresh
// ComponentTemplate followed by every reusable ComponentDefinition currently
// available for another instance.
func (library CompositionLibrary) PaletteEntries() []PaletteEntry {
	library = library.Normalize()
	out := make([]PaletteEntry, 0, len(library.ComponentTemplates)+len(library.ComponentDefinitions))
	for _, template := range library.ComponentTemplates {
		out = append(out, PaletteEntry{
			Key:           template.Key,
			Label:         template.Label,
			Summary:       template.Summary,
			Category:      template.Category,
			GoSXComponent: template.GoSXComponent,
			Source:        template.Source,
			Status:        template.Status,
			AddLabel:      FirstNonEmpty(template.AddLabel, "Add "+template.Label),
		})
	}
	for _, definition := range library.ComponentDefinitions {
		out = append(out, PaletteEntry{
			Key:           "shared:" + definition.Key,
			Label:         definition.Label,
			Summary:       definition.Summary,
			Category:      definition.Category,
			GoSXComponent: definition.GoSXComponent,
			Source:        definition.Source,
			Status:        "Shared",
			AddLabel:      "Add another " + definition.Label,
			IsShared:      true,
			DefinitionKey: definition.Key,
		})
	}
	return out
}

func (entry PaletteEntry) Normalize() PaletteEntry {
	entry.Key = strings.TrimSpace(entry.Key)
	entry.Label = strings.TrimSpace(entry.Label)
	entry.Summary = strings.TrimSpace(entry.Summary)
	entry.Category = strings.TrimSpace(entry.Category)
	entry.GoSXComponent = strings.TrimSpace(entry.GoSXComponent)
	entry.Source = normalizeComponentSource(entry.Source)
	entry.Status = strings.TrimSpace(entry.Status)
	entry.AddLabel = strings.TrimSpace(entry.AddLabel)
	entry.DefinitionKey = strings.TrimSpace(entry.DefinitionKey)
	return entry
}
