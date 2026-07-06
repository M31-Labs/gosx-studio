package studio

import (
	"strings"

	cmsflows "m31labs.dev/gosx-studio/cms/flows"
)

type FlowCardBuildOptions struct {
	CardClassPrefix    string
	ReadyStatusClass   string
	WatchStatusClass   string
	DefaultStatusClass string
	RequiredLabel      string
	OptionalLabel      string
}

type FlowEditorCommandOptions struct {
	EditHrefPrefix string
	EditGroup      string
	EditItemName   string
	EditSummary    string
	PreviewGroup   string
	PreviewSummary string
}

func FlowCardsFromStudioFlows(flows []cmsflows.StudioFlow, options FlowCardBuildOptions) []FlowCard {
	cards := make([]FlowCard, 0, len(flows))
	for _, flow := range flows {
		card := FlowCard{
			Key:                strings.TrimSpace(flow.Key),
			Label:              strings.TrimSpace(flow.Label),
			Description:        strings.TrimSpace(flow.Description),
			Summary:            strings.TrimSpace(flow.Summary),
			StatusClass:        flowCardStatusClass(flow.Status, options),
			StatusLabel:        strings.TrimSpace(flow.StatusLabel),
			CardClass:          flowCardClass(flow.Status, options),
			Route:              strings.TrimSpace(flow.Route),
			EmbedTarget:        strings.TrimSpace(flow.EmbedTarget),
			PrimaryHandlerRef:  strings.TrimSpace(flow.PrimaryAction.HandlerRef),
			RequiredFieldCount: flow.RequiredFieldCount,
			CanExecute:         flow.CanExecute,
			HasRoute:           flow.HasRoute,
			HasEmbedTarget:     flow.HasEmbedTarget,
			HasPrimaryAction:   flow.HasPrimaryAction,
			Steps:              FlowStepsFromStudioSteps(flow.Steps),
			Actions:            FlowActionsFromStudioActions(flow.Actions, options),
		}
		cards = append(cards, card)
	}
	return normalizeFlowCards(cards)
}

func FlowStepsFromStudioSteps(steps []cmsflows.StudioStep) []FlowStep {
	out := make([]FlowStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, FlowStep{
			Key:        strings.TrimSpace(step.Key),
			Label:      strings.TrimSpace(step.Label),
			BlockCount: step.BlockCount,
			HasBlocks:  step.HasBlocks,
		})
	}
	return out
}

func FlowActionsFromStudioActions(actions []cmsflows.StudioAction, options FlowCardBuildOptions) []FlowAction {
	out := make([]FlowAction, 0, len(actions))
	for _, action := range actions {
		out = append(out, FlowAction{
			Key:    strings.TrimSpace(action.Key),
			Label:  strings.TrimSpace(action.Label),
			Fields: FlowFieldsFromStudioFields(action.Fields, options),
		})
	}
	return out
}

func FlowFieldsFromStudioFields(fields []cmsflows.StudioField, options FlowCardBuildOptions) []FlowField {
	out := make([]FlowField, 0, len(fields))
	for _, field := range fields {
		out = append(out, FlowField{
			Name:          strings.TrimSpace(field.Name),
			Label:         strings.TrimSpace(field.Label),
			RequiredLabel: flowFieldRequiredLabel(field.Required, options),
		})
	}
	return out
}

func FlowEditorCommandsFromStudioFlows(flows []cmsflows.StudioFlow, options FlowEditorCommandOptions) []Command {
	commands := make([]Command, 0, len(flows)*2)
	editPrefix := firstNonEmpty(strings.TrimSpace(options.EditHrefPrefix), "#flow=")
	editGroup := firstNonEmpty(strings.TrimSpace(options.EditGroup), "Flows")
	editItemName := firstNonEmpty(strings.TrimSpace(options.EditItemName), "flow")
	editSummary := firstNonEmpty(strings.TrimSpace(options.EditSummary), "Configure handler refs and flow step labels.")
	previewGroup := firstNonEmpty(strings.TrimSpace(options.PreviewGroup), "Preview")
	previewSummary := firstNonEmpty(strings.TrimSpace(options.PreviewSummary), "Open the public route that can host this flow.")
	for _, flow := range flows {
		key := normalizeKey(flow.Key)
		label := strings.TrimSpace(flow.Label)
		if key == "" || label == "" {
			continue
		}
		commands = append(commands, Command{
			Kind:     CommandLink,
			Key:      "edit-flow-" + key,
			Label:    "Edit " + label + " " + editItemName,
			Summary:  editSummary,
			Group:    editGroup,
			Href:     editPrefix + key,
			Keywords: []string{editItemName, "form", "request"},
		})
		route := strings.TrimSpace(flow.Route)
		if flow.HasRoute && route != "" {
			commands = append(commands, Command{
				Kind:     CommandLink,
				Key:      "preview-flow-" + key,
				Label:    "Preview " + label,
				Summary:  previewSummary,
				Group:    previewGroup,
				Href:     route,
				Keywords: []string{"form", "route", "family"},
			})
		}
	}
	return normalizeCommands(commands)
}

func CommandFlowsFromStudioFlows(flows []cmsflows.StudioFlow) []CommandFlow {
	out := make([]CommandFlow, 0, len(flows))
	for _, flow := range flows {
		out = append(out, CommandFlow{
			Key:            strings.TrimSpace(flow.Key),
			Label:          strings.TrimSpace(flow.Label),
			Description:    strings.TrimSpace(flow.Description),
			Route:          strings.TrimSpace(flow.Route),
			EmbedTarget:    strings.TrimSpace(flow.EmbedTarget),
			HasRoute:       flow.HasRoute,
			HasEmbedTarget: flow.HasEmbedTarget,
		})
	}
	return out
}

func ExecutableStudioFlowCount(flows []cmsflows.StudioFlow) int {
	count := 0
	for _, flow := range flows {
		if flow.CanExecute && flow.HasRoute && flow.HasEmbedTarget && strings.TrimSpace(flow.PrimaryAction.HandlerRef) != "" {
			count++
		}
	}
	return count
}

func flowCardStatusClass(status string, options FlowCardBuildOptions) string {
	switch strings.TrimSpace(status) {
	case "ready":
		return firstNonEmpty(options.ReadyStatusClass, "status status--ready")
	case "watch":
		return firstNonEmpty(options.WatchStatusClass, "status status--request")
	default:
		return firstNonEmpty(options.DefaultStatusClass, "status")
	}
}

func flowCardClass(status string, options FlowCardBuildOptions) string {
	prefix := firstNonEmpty(options.CardClassPrefix, "studio-flow-card studio-flow-card--")
	return prefix + strings.TrimSpace(status)
}

func flowFieldRequiredLabel(required bool, options FlowCardBuildOptions) string {
	if required {
		return firstNonEmpty(options.RequiredLabel, "Required")
	}
	return firstNonEmpty(options.OptionalLabel, "Optional")
}
