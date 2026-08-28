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

// InstancePlaceWriter places a new instance of an EXISTING shared
// ComponentDefinition (identified by definitionKey) onto pageKey.
// instanceKey/instanceLabel are the operator-supplied identity for the new
// instance; when instanceKey is empty the host chooses one (e.g. derived
// from definitionKey with a numeric suffix) and returns it, so the mutation
// result can report which instance was actually placed without the caller
// reloading first. Unlike the companion "create a brand-new shared
// component" host action, a writer implementing this MUST fail (a plain Go
// error, propagated by ApplyPlaceInstance exactly like the other instance
// writers) when definitionKey does not already exist — placement always
// targets a definition the palette already listed.
type InstancePlaceWriter func(pageKey, definitionKey, instanceKey, instanceLabel string) (string, error)

// ApplyPlaceInstance is the host-agnostic body for the place-instance
// authoring operation: the composition workspace palette's "place another
// instance" action for a shared ComponentDefinition. It requires an existing
// definition (ComponentTemplateKey) and a target page (PageKey);
// ComponentKey/ComponentLabel are an optional operator-supplied instance
// key/label for the new instance.
func ApplyPlaceInstance(m AuthoringMutation, write InstancePlaceWriter) (AuthoringMutationResult, error) {
	definitionKey := strings.TrimSpace(m.ComponentTemplateKey)
	pageKey := strings.TrimSpace(m.PageKey)
	if definitionKey == "" {
		return AuthoringMutationResult{
			Message:     "Choose a shared component to place.",
			FieldErrors: map[string]string{AuthoringFieldComponentTemplateKey: "Choose a shared component."},
			Values:      m.FormValues(),
		}, nil
	}
	if pageKey == "" {
		return AuthoringMutationResult{
			Message:     "Choose a page to place this instance on.",
			FieldErrors: map[string]string{AuthoringFieldPageKey: "Choose a target page."},
			Values:      m.FormValues(),
		}, nil
	}
	instanceKey, err := write(pageKey, definitionKey, strings.TrimSpace(m.ComponentKey), strings.TrimSpace(m.ComponentLabel))
	if err != nil {
		return AuthoringMutationResult{}, err
	}
	return AuthoringMutationResult{
		Message:        "Instance placed.",
		PreviewURL:     "/",
		RefreshPreview: true,
		Values:         m.FormValues(),
		Changes: []AuthoringChange{{
			Key:       "place:" + instanceKey,
			Label:     "Place instance",
			Summary:   "Placed another instance of this shared component.",
			Kind:      "instance-place",
			PageKey:   pageKey,
			Component: instanceKey,
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
