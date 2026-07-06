package studio

import (
	"testing"

	cmsflows "m31labs.dev/gosx-studio/cms/flows"
)

func TestFlowCardsFromStudioFlowsBuildsReusableEditorModels(t *testing.T) {
	flows := []cmsflows.StudioFlow{
		{
			Key:                "contact",
			Label:              "Contact",
			Description:        "Public contact form.",
			Summary:            "1 step / 1 action / 1 field",
			Status:             "ready",
			StatusLabel:        "Ready",
			Route:              "/contact",
			HasRoute:           true,
			EmbedTarget:        "contact-form",
			HasEmbedTarget:     true,
			RequiredFieldCount: 1,
			CanExecute:         true,
			Steps:              []cmsflows.StudioStep{{Key: "message", Label: "Message", BlockCount: 1, HasBlocks: true}},
			Actions:            []cmsflows.StudioAction{{Key: "submit", Label: "Submit", HandlerRef: "contact.submit", Fields: []cmsflows.StudioField{{Name: "email", Label: "Email", Required: true}}}},
			PrimaryAction:      cmsflows.StudioAction{Key: "submit", Label: "Submit", HandlerRef: "contact.submit"},
			HasPrimaryAction:   true,
		},
		{
			Key:         "newsletter",
			Label:       "Newsletter",
			Status:      "watch",
			StatusLabel: "Registered",
			CanExecute:  true,
			Actions:     []cmsflows.StudioAction{{Key: "submit", Label: "Submit", Fields: []cmsflows.StudioField{{Name: "email", Label: "Email"}}}},
		},
	}
	cards := FlowCardsFromStudioFlows(flows, FlowCardBuildOptions{})
	if len(cards) != 2 {
		t.Fatalf("expected two cards, got %#v", cards)
	}
	contact := cards[0]
	if contact.StatusClass != "status status--ready" || contact.CardClass != "studio-flow-card studio-flow-card--ready" {
		t.Fatalf("unexpected contact classes: %#v", contact)
	}
	if contact.PrimaryHandlerRef != "contact.submit" || !contact.CanExecute || !contact.HasRoute || !contact.HasEmbedTarget {
		t.Fatalf("expected executable contact card: %#v", contact)
	}
	if len(contact.Steps) != 1 || contact.Steps[0].Key != "message" || !contact.Steps[0].HasBlocks {
		t.Fatalf("expected step metadata: %#v", contact.Steps)
	}
	if len(contact.Actions) != 1 || len(contact.Actions[0].Fields) != 1 || contact.Actions[0].Fields[0].RequiredLabel != "Required" {
		t.Fatalf("expected field metadata: %#v", contact.Actions)
	}
	if cards[1].StatusClass != "status status--request" || cards[1].Actions[0].Fields[0].RequiredLabel != "Optional" {
		t.Fatalf("unexpected watch flow card defaults: %#v", cards[1])
	}
	commandFlows := CommandFlowsFromStudioFlows(flows)
	if len(commandFlows) != 2 || commandFlows[0].Route != "/contact" || commandFlows[0].EmbedTarget != "contact-form" {
		t.Fatalf("unexpected command flows: %#v", commandFlows)
	}
	flowEditorCommands := FlowEditorCommandsFromStudioFlows(flows, FlowEditorCommandOptions{})
	byKey := map[string]Command{}
	for _, command := range flowEditorCommands {
		byKey[command.Key] = command
	}
	if byKey["edit-flow-contact"].Href != "#flow=contact" || byKey["preview-flow-contact"].Href != "/contact" || byKey["edit-flow-newsletter"].Kind != CommandLink {
		t.Fatalf("unexpected flow editor commands: %#v", flowEditorCommands)
	}
	if _, ok := byKey["preview-flow-newsletter"]; ok {
		t.Fatalf("expected preview commands only for routed flows: %#v", flowEditorCommands)
	}
	clientCommands := FlowEditorCommandsFromStudioFlows(flows[:1], FlowEditorCommandOptions{
		EditGroup:      "Forms",
		EditItemName:   "form",
		EditSummary:    "Update visible form labels.",
		PreviewSummary: "Preview the family form.",
	})
	if len(clientCommands) != 2 || clientCommands[0].Group != "Forms" || clientCommands[0].Label != "Edit Contact form" || clientCommands[0].Summary != "Update visible form labels." {
		t.Fatalf("unexpected client-facing flow editor commands: %#v", clientCommands)
	}
	if ExecutableStudioFlowCount(flows) != 1 {
		t.Fatalf("expected only fully routed, embedded, handler-backed flow to count")
	}
}
