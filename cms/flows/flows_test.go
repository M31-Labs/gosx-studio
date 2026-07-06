package flows

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
)

func TestCatalogNormalizesAndDedupes(t *testing.T) {
	catalog := Catalog(
		Definition{Key: "Schedule Request", Label: "Schedule request", Steps: []Step{{Label: "Request"}}, Actions: []Action{{Key: "submit", HandlerRef: "schedule.submit"}}},
		Definition{Key: "schedule-request", Label: "Duplicate", Steps: []Step{{Key: "request"}}},
	)
	if len(catalog) != 1 {
		t.Fatalf("expected one deduped flow, got %#v", catalog)
	}
	flow := catalog[0]
	if flow.Key != "schedule-request" || flow.Steps[0].Key != "request" || flow.Steps[0].Blocks.Version != 1 {
		t.Fatalf("unexpected normalized flow: %#v", flow)
	}
}

func TestValidateRequiresStepsAndHandlers(t *testing.T) {
	errs := Validate(Definition{Key: "contact", Actions: []Action{{Key: "submit"}}})
	if errs["steps"] == "" || errs["actions.submit"] == "" {
		t.Fatalf("expected validation errors, got %#v", errs)
	}
}

func TestFind(t *testing.T) {
	catalog := Catalog(Contact("contact.submit"))
	flow, ok := Find(catalog, "contact")
	if !ok || flow.Label != "Contact" {
		t.Fatalf("expected contact flow, got %#v %v", flow, ok)
	}
}

func TestPresetFlows(t *testing.T) {
	for _, flow := range []Definition{
		Contact("contact.submit"),
		ScheduleRequest("schedule.submit"),
		ScheduleTour("tour.submit"),
		Appointment("appointment.submit"),
		Newsletter("newsletter.submit"),
		PurchaseRequest("purchase.submit"),
		CheckoutHandoff("checkout.continue"),
		Enrollment("enrollment.submit"),
	} {
		if errs := Validate(flow); len(errs) != 0 {
			t.Fatalf("expected preset flow to validate: %#v %#v", flow, errs)
		}
		if len(flow.Actions) != 1 || len(flow.Actions[0].Fields) == 0 {
			t.Fatalf("expected preset flow fields: %#v", flow)
		}
	}
}

func TestRenderStudioPanel(t *testing.T) {
	html := gosx.RenderHTML(RenderStudioPanel([]Definition{
		Contact("contact.submit"),
		ScheduleTour("tour.submit"),
	}, StudioOptions{
		SelectedKey: "schedule-tour",
		NewHref:     "/admin/flows/new",
		EditHref: func(definition Definition) string {
			return "/admin/flows/" + definition.Key
		},
	}))
	for _, want := range []string{
		`class="flow-studio"`,
		`href="/admin/flows/new"`,
		`data-flow-key="schedule-tour"`,
		`flow-studio__flow--selected`,
		`Request a school visit or guided tour.`,
		`data-flow-action="submit"`,
		`data-handler-ref="tour.submit"`,
		`Request tour (4 fields)`,
		`href="/admin/flows/schedule-tour"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected %q in flow panel html: %s", want, html)
		}
	}
}

func TestRenderStudioPanelEmptyState(t *testing.T) {
	html := gosx.RenderHTML(RenderStudioPanel(nil, StudioOptions{Class: "flows"}))
	if !strings.Contains(html, `class="flows"`) || !strings.Contains(html, "No flows configured.") {
		t.Fatalf("expected empty state: %s", html)
	}
}

func TestStudioLibraryDescribesExecutionAndEmbedTargets(t *testing.T) {
	library := StudioLibrary([]Definition{
		Contact("contact.submit"),
		Newsletter("newsletter.submit"),
		Definition{Key: "orphan", Label: "Orphan", Steps: []Step{{Key: "start"}}, Actions: []Action{{Key: "submit"}}},
	}, StudioLibraryOptions{
		Routes:       map[string]string{FlowKeyContact: "/contact"},
		EmbedTargets: map[string]string{FlowKeyContact: "contact"},
	})
	if len(library) != 3 {
		t.Fatalf("expected three flow views, got %#v", library)
	}
	contact := library[0]
	if contact.Key != FlowKeyContact || !contact.CanExecute || contact.Status != "ready" || !contact.HasRoute || contact.Route != "/contact" || !contact.HasEmbedTarget {
		t.Fatalf("unexpected contact flow view: %#v", contact)
	}
	if !contact.HasPrimaryAction || contact.PrimaryAction.HandlerRef != "contact.submit" || contact.RequiredFieldCount != 3 {
		t.Fatalf("unexpected primary action: %#v", contact)
	}
	if len(contact.Actions[0].Fields) != 3 || contact.Actions[0].Fields[1].Name != "email" {
		t.Fatalf("unexpected fields: %#v", contact.Actions[0].Fields)
	}
	if library[1].Status != "watch" || library[1].StatusLabel != "Registered" {
		t.Fatalf("expected executable flow without route to be watch: %#v", library[1])
	}
	if library[2].CanExecute || library[2].Status != "next" {
		t.Fatalf("expected missing handler to need work: %#v", library[2])
	}
}
