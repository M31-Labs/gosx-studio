package flows

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx-admin/workbench"
)

type StudioLibraryOptions struct {
	Routes       map[string]string
	EmbedTargets map[string]string
}

type StudioFlow struct {
	Key                string
	Label              string
	Description        string
	Summary            string
	Status             string
	StatusLabel        string
	Route              string
	HasRoute           bool
	EmbedTarget        string
	HasEmbedTarget     bool
	StepCount          int
	ActionCount        int
	FieldCount         int
	RequiredFieldCount int
	CanExecute         bool
	Steps              []StudioStep
	Actions            []StudioAction
	PrimaryAction      StudioAction
	HasPrimaryAction   bool
}

type StudioStep struct {
	Key        string
	Label      string
	BlockCount int
	HasBlocks  bool
}

type StudioAction struct {
	Key        string
	Label      string
	HandlerRef string
	FieldCount int
	CanExecute bool
	Fields     []StudioField
}

type StudioField struct {
	Name     string
	Label    string
	Kind     string
	Required bool
}

func StudioLibrary(catalog []Definition, options StudioLibraryOptions) []StudioFlow {
	definitions := Catalog(catalog...)
	out := make([]StudioFlow, 0, len(definitions))
	for _, definition := range definitions {
		out = append(out, StudioFlowView(definition, options))
	}
	return out
}

func StudioFlowView(definition Definition, options StudioLibraryOptions) StudioFlow {
	definition = Normalize(definition)
	view := StudioFlow{
		Key:         definition.Key,
		Label:       definition.Label,
		Description: definition.Description,
		StepCount:   len(definition.Steps),
		ActionCount: len(definition.Actions),
		Steps:       make([]StudioStep, 0, len(definition.Steps)),
		Actions:     make([]StudioAction, 0, len(definition.Actions)),
	}
	if route := strings.TrimSpace(options.Routes[definition.Key]); route != "" {
		view.Route = route
		view.HasRoute = true
	}
	if target := strings.TrimSpace(options.EmbedTargets[definition.Key]); target != "" {
		view.EmbedTarget = target
		view.HasEmbedTarget = true
	}
	for _, step := range definition.Steps {
		blockCount := len(step.Blocks.Blocks)
		view.Steps = append(view.Steps, StudioStep{
			Key:        step.Key,
			Label:      step.Label,
			BlockCount: blockCount,
			HasBlocks:  blockCount > 0,
		})
	}
	for _, action := range definition.Actions {
		actionView := studioActionView(action)
		view.Actions = append(view.Actions, actionView)
		view.FieldCount += actionView.FieldCount
		for _, field := range actionView.Fields {
			if field.Required {
				view.RequiredFieldCount++
			}
		}
	}
	if len(view.Actions) > 0 {
		view.PrimaryAction = view.Actions[0]
		view.HasPrimaryAction = true
	}
	view.CanExecute = canExecuteDefinition(definition)
	view.Status = "ready"
	view.StatusLabel = "Ready"
	if !view.CanExecute {
		view.Status = "next"
		view.StatusLabel = "Needs handler"
	} else if !view.HasRoute && !view.HasEmbedTarget {
		view.Status = "watch"
		view.StatusLabel = "Registered"
	}
	view.Summary = fmt.Sprintf("%d steps / %d actions / %d fields", view.StepCount, view.ActionCount, view.FieldCount)
	return view
}

func studioActionView(action Action) StudioAction {
	action.Key = normalizeKey(firstNonEmpty(action.Key, action.Label))
	action.Label = firstNonEmpty(action.Label, action.Key)
	action.HandlerRef = strings.TrimSpace(action.HandlerRef)
	view := StudioAction{
		Key:        action.Key,
		Label:      action.Label,
		HandlerRef: action.HandlerRef,
		FieldCount: len(action.Fields),
		CanExecute: strings.TrimSpace(action.HandlerRef) != "",
		Fields:     make([]StudioField, 0, len(action.Fields)),
	}
	for _, field := range action.Fields {
		view.Fields = append(view.Fields, studioFieldView(field))
	}
	return view
}

func studioFieldView(field workbench.Field) StudioField {
	label := strings.TrimSpace(field.Label)
	if label == "" {
		label = strings.TrimSpace(field.Name)
	}
	return StudioField{
		Name:     strings.TrimSpace(field.Name),
		Label:    label,
		Kind:     string(field.Kind),
		Required: field.Required,
	}
}

func canExecuteDefinition(definition Definition) bool {
	return DocumentCanExecute(DocumentFromDefinition(definition, DocumentOptions{}))
}
