package studio

import "testing"

func TestFieldNamePrefixAndDOMIDNormalizeStudioKeys(t *testing.T) {
	if got := FieldNamePrefix("flow", "schedule-tour", "request.label"); got != "FlowScheduleTourRequestLabel" {
		t.Fatalf("unexpected field prefix: %q", got)
	}
	if got := DOMID("studio-flow-handler", "Schedule tour", "request.label"); got != "studio-flow-handler-schedule-tour-request-label" {
		t.Fatalf("unexpected DOM id: %q", got)
	}
	if got := DOMIDPart("  Ready? Yes!  "); got != "ready-yes" {
		t.Fatalf("unexpected DOM id part: %q", got)
	}
}

func TestAssessRequiredFieldsReportsMissingFields(t *testing.T) {
	assessment := AssessRequiredFields([]RequiredField{
		{Label: "site title", Value: "Muddy Noni"},
		{Label: "tagline", Value: " "},
		{Label: "hero headline", Value: ""},
	}, RequiredFieldOptions{
		ReadySummary:        "Copy ready",
		MissingDetailPrefix: "Missing required",
	})
	if assessment.Status != ReadinessWatch || assessment.Summary != "2 missing fields" {
		t.Fatalf("unexpected missing assessment: %#v", assessment)
	}
	if assessment.Detail != "Missing required tagline, hero headline." || assessment.Count != 1 || assessment.Total != 3 {
		t.Fatalf("unexpected missing details: %#v", assessment)
	}
}

func TestAssessRequiredFieldsReportsReadyFields(t *testing.T) {
	assessment := AssessRequiredFields([]RequiredField{
		{Label: "site title", Value: "Muddy Noni"},
	}, RequiredFieldOptions{ReadySummary: "Copy ready", ReadyDetail: "All copy fields are filled."})
	if assessment.Status != ReadinessReady || assessment.Summary != "Copy ready" || assessment.Detail != "All copy fields are filled." {
		t.Fatalf("unexpected ready assessment: %#v", assessment)
	}
}

func TestAssessExecutableFlows(t *testing.T) {
	empty := AssessExecutableFlows(0, 0, FlowExecutionOptions{})
	if empty.Status != ReadinessNext || empty.Summary != "0/0 executable" {
		t.Fatalf("unexpected empty flow assessment: %#v", empty)
	}
	partial := AssessExecutableFlows(3, 1, FlowExecutionOptions{})
	if partial.Status != ReadinessWatch || partial.Summary != "1/3 executable" {
		t.Fatalf("unexpected partial flow assessment: %#v", partial)
	}
	ready := AssessExecutableFlows(2, 4, FlowExecutionOptions{ReadyDetail: "Ready"})
	if ready.Status != ReadinessReady || ready.Summary != "2/2 executable" || ready.Detail != "Ready" {
		t.Fatalf("unexpected ready flow assessment: %#v", ready)
	}
}

func TestExecutableFlowCardCountUsesNormalizedCards(t *testing.T) {
	count := ExecutableFlowCardCount([]FlowCard{
		{Key: "contact", Label: "Contact", CanExecute: true},
		{Key: "tour", Label: "Tour", PrimaryHandlerRef: "tour.submit"},
		{Key: "newsletter", Label: "Newsletter"},
	})
	if count != 2 {
		t.Fatalf("expected two executable flow cards, got %d", count)
	}
}
