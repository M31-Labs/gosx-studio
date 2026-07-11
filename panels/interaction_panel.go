package panels

import (
	"fmt"
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
	"sort"
	"strings"
)

type InteractionPanelOptions struct {
	ID, Action, CSRFToken, SelectedGraphID string
	Graphs                                 []core.InteractionGraph
	Target                                 core.InteractionTarget
	OperationIDs, Heads                    map[string]string
}

func RenderInteractionPanel(o InteractionPanelOptions) gosx.Node {
	if strings.TrimSpace(o.ID) == "" {
		o.ID = "studio-interactions"
	}
	graphs := append([]core.InteractionGraph(nil), o.Graphs...)
	sort.Slice(graphs, func(i, j int) bool { return graphs[i].ID < graphs[j].ID })
	rows := []gosx.Node{}
	for _, g := range graphs {
		rows = append(rows, gosx.El("article", gosx.Attrs(gosx.Attr("data-studio-interaction", g.ID)), gosx.El("h3", nil, gosx.Text(g.ID)), gosx.El("p", nil, gosx.Text(string(g.Event)+" · "+fmt.Sprintf("%d ordered actions", len(g.Actions)))), gosx.El("a", gosx.Attrs(gosx.Attr("href", "#"+o.ID+"-editor"), gosx.Attr("aria-label", "Edit interaction "+g.ID)), gosx.Text("Edit"))))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("id", o.ID), gosx.Attr("class", "editor-panel studio-interactions"), gosx.Attr("data-studio-interaction-panel", "true")), gosx.El("header", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Interactions")), gosx.El("h2", nil, gosx.Text("Behavior graph")), gosx.El("output", gosx.Attrs(gosx.Attr("aria-live", "polite")), gosx.Text(fmt.Sprintf("%d interactions", len(graphs))))), gosx.El("p", nil, gosx.Text("Actions run in the displayed order. Hover behaviors also run from keyboard focus, and reduced-motion preferences remove delays and smooth scrolling.")), gosx.El("div", gosx.Attrs(gosx.Attr("role", "list"), gosx.Attr("aria-label", "Configured interactions")), gosx.Fragment(rows...)), renderInteractionEditor(o))
}
func renderInteractionEditor(o InteractionPanelOptions) gosx.Node {
	events := []string{"click", "submit", "hover", "focus", "viewport-enter"}
	eventNodes := []gosx.Node{}
	for _, v := range events {
		eventNodes = append(eventNodes, gosx.El("option", gosx.Attrs(gosx.Attr("value", v)), gosx.Text(v)))
	}
	actions := []string{"show", "hide", "toggle-class", "navigate", "scroll", "focus", "set-variant", "play-media", "pause-media"}
	actionNodes := []gosx.Node{}
	for _, v := range actions {
		actionNodes = append(actionNodes, gosx.El("option", gosx.Attrs(gosx.Attr("value", v)), gosx.Text(v)))
	}
	children := []gosx.Node{hidden("gosx_studio_interaction_operation", string(authoring.InteractionUpsert)), hidden("gosx_studio_interaction_operation_id", o.OperationIDs["upsert"]), hidden("gosx_studio_interaction_expected_head", o.Heads["interaction:"+o.SelectedGraphID]), gosx.El("label", nil, gosx.Text("Stable interaction ID"), gosx.El("input", gosx.Attrs(gosx.Attr("name", "interaction_id"), gosx.Attr("required", "required"), gosx.Attr("pattern", "[A-Za-z0-9_.-]+"), gosx.Attr("value", o.SelectedGraphID)))), gosx.El("label", nil, gosx.Text("Event"), gosx.El("select", gosx.Attrs(gosx.Attr("name", "interaction_event")), gosx.Fragment(eventNodes...))), gosx.El("fieldset", nil, gosx.El("legend", nil, gosx.Text("Ordered action 1")), gosx.El("label", nil, gosx.Text("Action"), gosx.El("select", gosx.Attrs(gosx.Attr("name", "interaction_action_kind")), gosx.Fragment(actionNodes...))), gosx.El("label", nil, gosx.Text("Target element ID"), gosx.El("input", gosx.Attrs(gosx.Attr("name", "interaction_target_id"), gosx.Attr("pattern", "[A-Za-z0-9_.-]+")))), gosx.El("label", nil, gosx.Text("Safe value"), gosx.El("input", gosx.Attrs(gosx.Attr("name", "interaction_value"))))), gosx.El("label", nil, gosx.Text("Condition"), gosx.El("select", gosx.Attrs(gosx.Attr("name", "interaction_condition")), gosx.El("option", gosx.Attrs(gosx.Attr("value", "always")), gosx.Text("Always (accessible default)")), gosx.El("option", gosx.Attrs(gosx.Attr("value", "no-reduced-motion")), gosx.Text("Only without reduced motion")))), gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Save interaction"))}
	if o.CSRFToken != "" {
		children = append(children, hidden("csrf_token", o.CSRFToken))
	}
	return gosx.El("form", gosx.Attrs(gosx.Attr("id", o.ID+"-editor"), gosx.Attr("method", "post"), gosx.Attr("action", o.Action), gosx.Attr("data-studio-interaction-form", "true")), gosx.Fragment(children...))
}
