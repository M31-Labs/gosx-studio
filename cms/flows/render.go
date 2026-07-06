package flows

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx"
)

type StudioOptions struct {
	Class       string
	SelectedKey string
	NewHref     string
	EditHref    func(Definition) string
}

func RenderStudioPanel(catalog []Definition, options StudioOptions) gosx.Node {
	className := strings.TrimSpace(options.Class)
	if className == "" {
		className = "flow-studio"
	}
	header := []gosx.Node{gosx.El("h2", nil, gosx.Text("Flows"))}
	if strings.TrimSpace(options.NewHref) != "" {
		header = append(header, gosx.El("a", gosx.Attrs(
			gosx.Attr("class", className+"__new"),
			gosx.Attr("href", options.NewHref),
		), gosx.Text("New flow")))
	}
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__header")), gosx.Fragment(header...)),
	}
	definitions := Catalog(catalog...)
	if len(definitions) == 0 {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__empty")), gosx.Text("No flows configured.")))
		return gosx.El("section", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
	}
	cards := make([]gosx.Node, 0, len(definitions))
	selectedKey := normalizeKey(options.SelectedKey)
	for _, definition := range definitions {
		cards = append(cards, renderFlowCard(className, definition, selectedKey, options))
	}
	children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("class", className+"__list")), gosx.Fragment(cards...)))
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(children...))
}

func renderFlowCard(className string, definition Definition, selectedKey string, options StudioOptions) gosx.Node {
	classes := className + "__flow"
	if selectedKey != "" && selectedKey == definition.Key {
		classes += " " + className + "__flow--selected"
	}
	children := []gosx.Node{
		gosx.El("h3", nil, gosx.Text(definition.Label)),
	}
	if definition.Description != "" {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__description")), gosx.Text(definition.Description)))
	}
	children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", className+"__meta")),
		gosx.Text(fmt.Sprintf("%d steps / %d actions", len(definition.Steps), len(definition.Actions))),
	))
	if len(definition.Steps) > 0 {
		children = append(children, renderFlowSteps(className, definition.Steps))
	}
	if len(definition.Actions) > 0 {
		children = append(children, renderFlowActions(className, definition.Actions))
	}
	if options.EditHref != nil {
		if href := strings.TrimSpace(options.EditHref(definition)); href != "" {
			children = append(children, gosx.El("a", gosx.Attrs(
				gosx.Attr("class", className+"__edit"),
				gosx.Attr("href", href),
			), gosx.Text("Edit flow")))
		}
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", classes),
		gosx.Attr("data-flow-key", definition.Key),
	), gosx.Fragment(children...))
}

func renderFlowSteps(className string, steps []Step) gosx.Node {
	nodes := make([]gosx.Node, 0, len(steps))
	for _, step := range steps {
		nodes = append(nodes, gosx.El("li", gosx.Attrs(
			gosx.Attr("data-flow-step", step.Key),
		), gosx.Text(step.Label)))
	}
	return gosx.El("ol", gosx.Attrs(gosx.Attr("class", className+"__steps")), gosx.Fragment(nodes...))
}

func renderFlowActions(className string, actions []Action) gosx.Node {
	nodes := make([]gosx.Node, 0, len(actions))
	for _, action := range actions {
		label := action.Label
		if len(action.Fields) > 0 {
			label = fmt.Sprintf("%s (%d fields)", label, len(action.Fields))
		}
		nodes = append(nodes, gosx.El("li", gosx.Attrs(
			gosx.Attr("data-flow-action", action.Key),
			gosx.Attr("data-handler-ref", action.HandlerRef),
		), gosx.Text(label)))
	}
	return gosx.El("ul", gosx.Attrs(gosx.Attr("class", className+"__actions")), gosx.Fragment(nodes...))
}
