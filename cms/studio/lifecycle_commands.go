package studio

import "strings"

type LifecycleCommandOptions struct {
	ApproveAction     string
	ScheduleAction    string
	ProcessDueAction  string
	Group             string
	ApproveLabel      string
	ScheduleLabel     string
	ProcessDueLabel   string
	ApproveSummary    string
	ScheduleSummary   string
	ProcessDueSummary string
}

func LifecycleCommands(options LifecycleCommandOptions) []Command {
	group := firstNonEmpty(strings.TrimSpace(options.Group), "Lifecycle")
	commands := []Command{}
	if action := strings.TrimSpace(options.ApproveAction); action != "" {
		commands = append(commands, Command{
			Kind:     CommandSubmit,
			Key:      "approve-publish",
			Label:    firstNonEmpty(options.ApproveLabel, "Approve publish"),
			Summary:  firstNonEmpty(options.ApproveSummary, "Record approval for the current site release."),
			Group:    group,
			Href:     action,
			Keywords: []string{"release", "review", "approval"},
		})
	}
	if action := strings.TrimSpace(options.ScheduleAction); action != "" {
		commands = append(commands, Command{
			Kind:     CommandSubmit,
			Key:      "schedule-publish",
			Label:    firstNonEmpty(options.ScheduleLabel, "Schedule publish"),
			Summary:  firstNonEmpty(options.ScheduleSummary, "Schedule the approved site release."),
			Group:    group,
			Href:     action,
			Keywords: []string{"release", "time", "schedule"},
		})
	}
	if action := strings.TrimSpace(options.ProcessDueAction); action != "" {
		commands = append(commands, Command{
			Kind:     CommandSubmit,
			Key:      "run-due-publishes",
			Label:    firstNonEmpty(options.ProcessDueLabel, "Run due publishes"),
			Summary:  firstNonEmpty(options.ProcessDueSummary, "Process release jobs that are ready to publish."),
			Group:    group,
			Href:     action,
			Keywords: []string{"worker", "release", "publish"},
		})
	}
	return normalizeCommands(commands)
}
