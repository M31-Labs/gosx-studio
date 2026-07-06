package studio

import "testing"

func TestLifecycleCommandsBuildSubmitActions(t *testing.T) {
	commands := LifecycleCommands(LifecycleCommandOptions{
		ApproveAction:    "/admin/editor/__actions/approvePublish",
		ScheduleAction:   "/admin/editor/__actions/schedulePublish",
		ProcessDueAction: "/admin/editor/__actions/processDuePublishes",
	})
	byKey := map[string]Command{}
	for _, command := range commands {
		byKey[command.Key] = command
	}
	if len(commands) != 3 {
		t.Fatalf("expected three lifecycle commands, got %#v", commands)
	}
	if byKey["approve-publish"].Kind != CommandSubmit || byKey["approve-publish"].Href != "/admin/editor/__actions/approvePublish" {
		t.Fatalf("unexpected approve command: %#v", byKey["approve-publish"])
	}
	if byKey["schedule-publish"].Label != "Schedule publish" || byKey["schedule-publish"].Group != "Lifecycle" {
		t.Fatalf("unexpected schedule command: %#v", byKey["schedule-publish"])
	}
	if byKey["run-due-publishes"].Summary == "" || byKey["run-due-publishes"].Href == "" {
		t.Fatalf("unexpected process-due command: %#v", byKey["run-due-publishes"])
	}
}

func TestLifecycleCommandsAllowLabelsAndSkipMissingActions(t *testing.T) {
	commands := LifecycleCommands(LifecycleCommandOptions{
		ApproveAction:  "/approve",
		Group:          "Release",
		ApproveLabel:   "Approve release",
		ApproveSummary: "Approve this release.",
	})
	if len(commands) != 1 {
		t.Fatalf("expected only configured lifecycle commands, got %#v", commands)
	}
	if commands[0].Key != "approve-publish" || commands[0].Label != "Approve release" || commands[0].Group != "Release" || commands[0].Summary != "Approve this release." {
		t.Fatalf("unexpected configured lifecycle command: %#v", commands[0])
	}
}
