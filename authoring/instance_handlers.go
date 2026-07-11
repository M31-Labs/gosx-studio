package authoring

import "strings"

// This file is the host-agnostic body for the reusable component/instance
// operations (core/instances.go): set-shared-field, override-instance,
// detach-instance, and restore-instance. It follows the same shape as
// style_handlers.go and handlers.go — Studio re-validates the mutation shape
// and delegates the actual draft write to a host-supplied writer closure, so
// hosts never have to duplicate field-presence checks or success/error
// AuthoringMutationResult wiring.

// SharedFieldWriter persists one control's new value on a shared component
// definition (identified by definitionKey — core.ComponentDefinition.Key).
// The host implementation fans this out to every attached instance BY
// CONSTRUCTION: instances resolve controls through
// core.EffectiveComponentControls, so writing the definition once is enough —
// no per-instance write follows.
type SharedFieldWriter func(definitionKey, controlKey, value string) error

// InstanceOverrideWriter persists one instance-local control override,
// diverging that single instance from its shared definition for that field
// only. Every other control on the instance keeps following the shared
// definition.
type InstanceOverrideWriter func(pageKey, componentKey, controlKey, value string) error

// InstanceDetachWriter severs one instance from its shared definition,
// freezing its currently-effective values as its own independent copy (see
// core.DetachComponentInstance).
type InstanceDetachWriter func(pageKey, componentKey string) error

// InstanceRestoreWriter re-attaches a detached (or override-diverged)
// instance to its shared definition, dropping instance-local overrides/state
// (see core.RestoreComponentInstance).
type InstanceRestoreWriter func(pageKey, componentKey string) error

// ApplySetSharedField is the host-agnostic body for the set-shared-field
// authoring operation.
func ApplySetSharedField(m AuthoringMutation, write SharedFieldWriter) (AuthoringMutationResult, error) {
	definitionKey := strings.TrimSpace(m.ComponentTemplateKey)
	controlKey := strings.TrimSpace(m.ControlKey)
	if definitionKey == "" || controlKey == "" {
		return AuthoringMutationResult{
			Message:     "Choose a shared component control.",
			FieldErrors: map[string]string{AuthoringFieldComponentTemplateKey: "Choose a shared component."},
			Values:      m.FormValues(),
		}, nil
	}
	if err := write(definitionKey, controlKey, m.Value); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Shared component updated for every instance.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:     "shared:" + definitionKey + ":" + controlKey,
			Label:   controlKey,
			Summary: "Updated every instance of this shared component.",
			Kind:    "shared-field",
			Binding: m.Binding,
		}},
	}, nil
}

// ApplyOverrideInstance is the host-agnostic body for the override-instance
// authoring operation.
func ApplyOverrideInstance(m AuthoringMutation, write InstanceOverrideWriter) (AuthoringMutationResult, error) {
	if strings.TrimSpace(m.PageKey) == "" || strings.TrimSpace(m.ComponentKey) == "" {
		return AuthoringMutationResult{
			Message:     "Choose an instance to override.",
			FieldErrors: map[string]string{AuthoringFieldComponentKey: "Choose a component."},
			Values:      m.FormValues(),
		}, nil
	}
	controlKey := strings.TrimSpace(m.ControlKey)
	if controlKey == "" {
		return AuthoringMutationResult{
			Message:     "Choose a control to override.",
			FieldErrors: map[string]string{AuthoringFieldControlKey: "Choose a control."},
			Values:      m.FormValues(),
		}, nil
	}
	if err := write(m.PageKey, m.ComponentKey, controlKey, m.Value); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Instance override saved.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "override:" + m.ComponentKey + ":" + controlKey,
			Label:     controlKey,
			Summary:   "Overrode " + controlKey + " for this instance only.",
			Kind:      "instance-override",
			PageKey:   m.PageKey,
			Component: m.ComponentKey,
		}},
	}, nil
}

// ApplyDetachInstance is the host-agnostic body for the detach-instance
// authoring operation.
func ApplyDetachInstance(m AuthoringMutation, write InstanceDetachWriter) (AuthoringMutationResult, error) {
	if strings.TrimSpace(m.PageKey) == "" || strings.TrimSpace(m.ComponentKey) == "" {
		return AuthoringMutationResult{
			Message:     "Choose an instance to detach.",
			FieldErrors: map[string]string{AuthoringFieldComponentKey: "Choose a component."},
			Values:      m.FormValues(),
		}, nil
	}
	if err := write(m.PageKey, m.ComponentKey); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Instance detached. It no longer follows shared edits.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "detach:" + m.ComponentKey,
			Label:     "Detach",
			Summary:   "This instance is now independent of its shared component.",
			Kind:      "instance-detach",
			PageKey:   m.PageKey,
			Component: m.ComponentKey,
		}},
	}, nil
}

// ApplyRestoreInstance is the host-agnostic body for the restore-instance
// authoring operation.
func ApplyRestoreInstance(m AuthoringMutation, write InstanceRestoreWriter) (AuthoringMutationResult, error) {
	if strings.TrimSpace(m.PageKey) == "" || strings.TrimSpace(m.ComponentKey) == "" {
		return AuthoringMutationResult{
			Message:     "Choose an instance to restore.",
			FieldErrors: map[string]string{AuthoringFieldComponentKey: "Choose a component."},
			Values:      m.FormValues(),
		}, nil
	}
	if err := write(m.PageKey, m.ComponentKey); err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Instance restored. It follows shared edits again.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "restore:" + m.ComponentKey,
			Label:     "Restore",
			Summary:   "This instance follows its shared component again.",
			Kind:      "instance-restore",
			PageKey:   m.PageKey,
			Component: m.ComponentKey,
		}},
	}, nil
}
