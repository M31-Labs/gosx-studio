package core

import "testing"

func validGraph() InteractionGraph {
	return InteractionGraph{ID: "graph_a", Target: InteractionTarget{CanvasID: "hero"}, Event: InteractionClick, Enabled: true, Actions: []InteractionAction{{ID: "a1", Kind: InteractionToggleClass, TargetID: "details", Value: "open"}, {ID: "a2", Kind: InteractionFocusAction, TargetID: "details"}}}
}
func TestInteractionValidationAndSerialization(t *testing.T) {
	g := validGraph()
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, err := SerializeInteractionGraphs([]InteractionGraph{g})
	if err != nil || raw == "" {
		t.Fatalf("serialize=%q %v", raw, err)
	}
}
func TestInteractionRejectsCodeURLsSelectorsAndInvalidFiniteValues(t *testing.T) {
	cases := []InteractionGraph{}
	for _, mutate := range []func(*InteractionGraph){func(g *InteractionGraph) { g.Actions[0].Kind = InteractionActionKind("javascript") }, func(g *InteractionGraph) {
		g.Actions[0].Kind = InteractionNavigate
		g.Actions[0].Value = "javascript:alert(1)"
	}, func(g *InteractionGraph) { g.Actions[0].TargetID = "#details > *" }, func(g *InteractionGraph) { g.Actions[0].Value = "js-owned" }, func(g *InteractionGraph) { g.Actions[0].DelayMS = 60001 }, func(g *InteractionGraph) {
		g.Target.CanvasID = ""
		g.Target.InstanceID = "x"
		g.Target.DefinitionID = "y"
	}} {
		g := validGraph()
		mutate(&g)
		cases = append(cases, g)
	}
	for _, g := range cases {
		if err := g.Validate(); err == nil {
			t.Fatalf("accepted %#v", g)
		}
	}
}
func TestInteractionPropagationIncludesInstanceDefinitionAndCanvas(t *testing.T) {
	base := validGraph()
	def := base
	def.ID = "graph_def"
	def.Target = InteractionTarget{DefinitionID: "def"}
	inst := base
	inst.ID = "graph_inst"
	inst.Target = InteractionTarget{InstanceID: "inst"}
	m := map[string]InteractionGraph{base.ID: base, def.ID: def, inst.ID: inst}
	got := InteractionGraphForTarget(m, "hero", "def", "inst")
	if len(got) != 3 || got[0].ID != "graph_a" || got[2].ID != "graph_inst" {
		t.Fatalf("got %#v", got)
	}
}
