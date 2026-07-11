package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	ErrReusableInvalid          = errors.New("invalid reusable component")
	ErrReusableDefinitionAbsent = errors.New("reusable component definition not found")
)

// ExplicitOverride distinguishes an authored empty value from no override.
// Clearing an override removes the map entry; Present=false is never inferred
// from Value's zero value.
type ExplicitOverride struct {
	Present bool   `json:"present"`
	Value   string `json:"value,omitempty"`
}

// DefinitionStyles and DefinitionLayout preserve authored responsive values
// by scope (for example base/default or mobile/default) and property.
type DefinitionStyles map[string]map[string]string
type DefinitionLayout map[string]map[string]string

// ComponentDefinition is a versioned reusable component authored once and
// referenced by any number of page instances.
type ComponentDefinition struct {
	ID              string           `json:"id"`
	Key             string           `json:"key"`
	Version         uint64           `json:"version"`
	TemplateKey     string           `json:"templateKey"`
	Label           string           `json:"label"`
	Controls        []Control        `json:"controls,omitempty"`
	Styles          DefinitionStyles `json:"styles,omitempty"`
	Layout          DefinitionLayout `json:"layout,omitempty"`
	Binding         ResourceBinding  `json:"binding,omitempty"`
	UpdatedRevision uint64           `json:"updatedRevision"`
}

// ComponentInstance keeps canvas identity instance-based while following a
// reusable definition. Detached instances retain an exact materialized copy.
type ComponentInstance struct {
	ID                string                      `json:"id"`
	DefinitionID      string                      `json:"definitionId"`
	DefinitionVersion uint64                      `json:"definitionVersion"`
	PageKey           string                      `json:"pageKey"`
	Region            string                      `json:"region"`
	Position          int                         `json:"position"`
	Overrides         map[string]ExplicitOverride `json:"overrides,omitempty"`
	Detached          bool                        `json:"detached"`
	Materialized      *ComponentDefinition        `json:"materialized,omitempty"`
	HeadRevision      uint64                      `json:"headRevision"`
}

func (definition ComponentDefinition) Normalize() ComponentDefinition {
	definition.ID = strings.TrimSpace(definition.ID)
	definition.Key = strings.TrimSpace(definition.Key)
	definition.TemplateKey = strings.TrimSpace(definition.TemplateKey)
	definition.Label = strings.TrimSpace(definition.Label)
	definition.Controls = cloneControls(definition.Controls)
	definition.Styles = cloneDefinitionValues(definition.Styles)
	definition.Layout = DefinitionLayout(cloneDefinitionValues(DefinitionStyles(definition.Layout)))
	definition.Binding = normalizeReusableBinding(definition.Binding)
	return definition
}

func (definition ComponentDefinition) Validate(templates []ComponentTemplate) error {
	definition = definition.Normalize()
	if !stableReusableID(definition.ID) || !stableReusableID(definition.Key) {
		return fmt.Errorf("%w: definition id and key must be stable identifiers", ErrReusableInvalid)
	}
	if definition.TemplateKey == "" || definition.Label == "" {
		return fmt.Errorf("%w: template and label are required", ErrReusableInvalid)
	}
	if len(templates) > 0 {
		found := false
		for _, template := range templates {
			if strings.TrimSpace(template.Key) == definition.TemplateKey {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("%w: unknown component template %q", ErrReusableInvalid, definition.TemplateKey)
		}
	}
	for _, control := range definition.Controls {
		if !stableReusableID(control.Key) {
			return fmt.Errorf("%w: control keys must be stable identifiers", ErrReusableInvalid)
		}
	}
	return nil
}

func (instance ComponentInstance) Normalize() ComponentInstance {
	instance.ID = strings.TrimSpace(instance.ID)
	instance.DefinitionID = strings.TrimSpace(instance.DefinitionID)
	instance.PageKey = strings.TrimSpace(instance.PageKey)
	instance.Region = strings.Trim(strings.TrimSpace(instance.Region), "/")
	if instance.Position < 0 {
		instance.Position = 0
	}
	instance.Overrides = cloneOverrides(instance.Overrides)
	if instance.Materialized != nil {
		copy := instance.Materialized.Normalize()
		instance.Materialized = &copy
	}
	return instance
}

func (instance ComponentInstance) Validate(definitions map[string]ComponentDefinition, regions []string) error {
	instance = instance.Normalize()
	if !stableReusableID(instance.ID) || !stableReusableID(instance.DefinitionID) || !stableReusableID(instance.PageKey) || instance.Region == "" {
		return fmt.Errorf("%w: instance id, definition, page, and region are required", ErrReusableInvalid)
	}
	definition, ok := definitions[instance.DefinitionID]
	if !ok && !instance.Detached {
		return ErrReusableDefinitionAbsent
	}
	if ok && !instance.Detached && instance.DefinitionVersion != 0 && instance.DefinitionVersion != definition.Version {
		return fmt.Errorf("%w: instance definition version is stale", ErrReusableInvalid)
	}
	if len(regions) > 0 && !containsReusableValue(regions, instance.Region) {
		return fmt.Errorf("%w: unknown page region %q", ErrReusableInvalid, instance.Region)
	}
	for key, override := range instance.Overrides {
		if !override.Present || !ValidInstanceOverrideKey(key) {
			return fmt.Errorf("%w: invalid explicit override %q", ErrReusableInvalid, key)
		}
	}
	if instance.Detached && instance.Materialized == nil {
		return fmt.Errorf("%w: detached instance requires materialized content", ErrReusableInvalid)
	}
	return nil
}

// CanvasIdentity is stable across definition updates because selection points
// at the instance rather than the current definition version.
func (instance ComponentInstance) CanvasIdentity() string {
	instance = instance.Normalize()
	if instance.ID == "" {
		return ""
	}
	return "instance:" + instance.ID
}

// ResolveComponentInstance returns the exact materialized component definition
// for rendering. Attached instances follow the latest definition; detached
// instances use their preserved snapshot. Explicit overrides are applied last.
func ResolveComponentInstance(instance ComponentInstance, definitions map[string]ComponentDefinition) (ComponentDefinition, error) {
	instance = instance.Normalize()
	var definition ComponentDefinition
	if instance.Detached {
		if instance.Materialized == nil {
			return ComponentDefinition{}, fmt.Errorf("%w: detached instance has no materialization", ErrReusableInvalid)
		}
		definition = instance.Materialized.Normalize()
	} else {
		var ok bool
		definition, ok = definitions[instance.DefinitionID]
		if !ok {
			return ComponentDefinition{}, ErrReusableDefinitionAbsent
		}
		definition = definition.Normalize()
	}
	for key, override := range instance.Overrides {
		if !override.Present {
			continue
		}
		applyDefinitionOverride(&definition, key, override.Value)
	}
	return definition, nil
}

func ValidInstanceOverrideKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "binding" {
		return true
	}
	parts := strings.Split(key, ":")
	if len(parts) == 2 && parts[0] == "control" {
		return stableReusableID(parts[1])
	}
	if len(parts) == 3 && (parts[0] == "style" || parts[0] == "layout") {
		return stableReusableScope(parts[1]) && stableReusableProperty(parts[2])
	}
	return false
}

func applyDefinitionOverride(definition *ComponentDefinition, key, value string) {
	parts := strings.Split(key, ":")
	switch {
	case key == "binding":
		definition.Binding.Binding = value
	case len(parts) == 2 && parts[0] == "control":
		for index := range definition.Controls {
			if definition.Controls[index].Key == parts[1] {
				definition.Controls[index].Value = value
			}
		}
	case len(parts) == 3 && parts[0] == "style":
		setDefinitionValue(definition.Styles, parts[1], parts[2], value)
	case len(parts) == 3 && parts[0] == "layout":
		setDefinitionValue(DefinitionStyles(definition.Layout), parts[1], parts[2], value)
	}
}

func setDefinitionValue(values DefinitionStyles, scope, property, value string) {
	if values[scope] == nil {
		values[scope] = map[string]string{}
	}
	values[scope][property] = value
}

func stableReusableID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func stableReusableScope(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && stableReusableID(parts[0]) && stableReusableID(parts[1])
}

func stableReusableProperty(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || char == '-' {
			continue
		}
		return false
	}
	return true
}

func cloneControls(values []Control) []Control {
	out := append([]Control(nil), values...)
	for index := range out {
		out[index].Options = append([]ControlOption(nil), out[index].Options...)
	}
	return out
}

func cloneDefinitionValues(values DefinitionStyles) DefinitionStyles {
	out := DefinitionStyles{}
	for scope, properties := range values {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		out[scope] = map[string]string{}
		for property, value := range properties {
			property = strings.TrimSpace(property)
			if property != "" {
				out[scope][property] = value
			}
		}
	}
	return out
}

func cloneOverrides(values map[string]ExplicitOverride) map[string]ExplicitOverride {
	out := map[string]ExplicitOverride{}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		out[strings.TrimSpace(key)] = values[key]
	}
	return out
}

func normalizeReusableBinding(binding ResourceBinding) ResourceBinding {
	binding.Key = strings.TrimSpace(binding.Key)
	binding.Label = strings.TrimSpace(binding.Label)
	binding.Summary = strings.TrimSpace(binding.Summary)
	binding.Binding = strings.TrimSpace(binding.Binding)
	return binding
}

func containsReusableValue(values []string, target string) bool {
	target = strings.Trim(strings.TrimSpace(target), "/")
	for _, value := range values {
		if strings.Trim(strings.TrimSpace(value), "/") == target {
			return true
		}
	}
	return false
}
