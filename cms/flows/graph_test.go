package flows

import "testing"

func graphFixture() Graph {
	return Graph{ID: "signup", Name: "Signup", InitialStepID: "form", Steps: []FlowStep{{ID: "form", Label: "Form", Kind: StepForm, Fields: []FieldRule{{Name: "email", Label: "Email", Type: "email", Required: true}}}, {ID: "done", Label: "Done", Kind: StepSuccess}, {ID: "error", Label: "Error", Kind: StepError}}, Transitions: []FlowTransition{{ID: "ok", FromStepID: "form", ToStepID: "done", Kind: TransitionSuccess}, {ID: "bad", FromStepID: "form", ToStepID: "error", Kind: TransitionError}}}
}
func TestGraphValidationAndBranches(t *testing.T) {
	g := graphFixture()
	if e := g.Validate(); e != nil {
		t.Fatal(e)
	}
	if len(g.TransitionsFrom("form")) != 2 {
		t.Fatal("branches missing")
	}
}
func TestGraphRejectsUnsafeActionsReferencesAndURLs(t *testing.T) {
	for _, mutate := range []func(*Graph){func(g *Graph) {
		g.Transitions[0].Navigation = NavigationPage
		g.Transitions[0].Destination = "javascript:alert(1)"
	}, func(g *Graph) {
		g.Transitions[0].Navigation = NavigationSection
		g.Transitions[0].Destination = "#x > *"
	}, func(g *Graph) { g.Steps[0].ComponentID = "a"; g.Steps[0].InstanceID = "b" }, func(g *Graph) { g.Steps[0].Fields[0].Pattern = "<script>" }, func(g *Graph) { g.Transitions[0].Kind = TransitionKind("execute-js") }} {
		g := graphFixture()
		mutate(&g)
		if e := g.Validate(); e == nil {
			t.Fatalf("accepted %#v", g)
		}
	}
}
