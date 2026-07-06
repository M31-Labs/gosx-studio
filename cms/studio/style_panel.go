package studio

import (
	"strings"

	"m31labs.dev/gosx"
	cmsstyle "m31labs.dev/gosx-studio/cms/style"
)

type StyleState struct {
	Key    string
	Label  string
	Active bool
}

type StyleScopeOptions struct {
	Class           string
	SystemID        string
	PageLabel       string
	BlockLabel      string
	FieldLabel      string
	BreakpointLabel string
	ValidityLabel   string
	States          []StyleState
}

type StyleControlBinding struct {
	ControlKey string
	FieldName  string
	Label      string
	Wide       bool
	Attrs      []FieldAttribute
}

type StyleRecipeGroup struct {
	Key               string
	Label             string
	VisualClass       string
	VisualMarks       int
	ReadoutControlKey string
	Controls          []StyleControlBinding
}

type StyleImpactOptions struct {
	Kicker     string
	Label      string
	Summary    string
	CountLabel string
	ScopeLabel string
	StateLabel string
}

type StyleWorkbenchOptions struct {
	Class  string
	Kicker string
	Title  string
	Impact StyleImpactOptions
	Groups []StyleRecipeGroup
	Values map[string]string
}

type StyleSelectControlOptions struct {
	Class        string
	SourcePrefix string
	Wide         bool
	Attrs        []FieldAttribute
}

type StyleRadioControlOptions struct {
	Class        string
	Legend       string
	Name         string
	SourcePrefix string
	Attrs        []FieldAttribute
}

func RenderStyleScope(options StyleScopeOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__style-scope")
	states := normalizeStyleStates(options.States)
	stateNodes := make([]gosx.Node, 0, len(states)+1)
	for _, state := range states {
		stateNodes = append(stateNodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-style-state", state.Key),
			gosx.Attr("aria-pressed", boolAttr(state.Active)),
		), gosx.Text(state.Label)))
	}
	stateNodes = append(stateNodes, gosx.El("output", gosx.Attrs(
		gosx.Attr("data-studio-style-state-label", "true"),
	), gosx.Text(activeStyleStateLabel(states))))
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-style-scope", "true"),
		gosx.Attr("aria-label", "Active style scope"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-scope__head")),
			gosx.El("div", nil,
				gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Scope")),
				gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-style-scope-block", "true")), gosx.Text(firstNonEmpty(options.BlockLabel, "Block"))),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-validity", "true")), gosx.Text(firstNonEmpty(options.ValidityLabel, "Ready"))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-scope__grid")),
			styleScopeCell("System", firstNonEmpty(options.SystemID, "site"), "data-studio-style-system-label"),
			styleScopeCell("Page", firstNonEmpty(options.PageLabel, "Home"), ""),
			styleScopeCell("Field", firstNonEmpty(options.FieldLabel, "Block"), "data-studio-style-scope-field"),
			styleScopeCell("Breakpoint", firstNonEmpty(options.BreakpointLabel, "Desktop"), "data-studio-style-breakpoint-label"),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-style-statebar"),
			gosx.Attr("role", "toolbar"),
			gosx.Attr("aria-label", "Style state"),
		), gosx.Fragment(stateNodes...)),
	)
}

func RenderStyleWorkbench(recipes []cmsstyle.RecipeView, options StyleWorkbenchOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "gosx-studio__style-workbench")
	groups := normalizeStyleRecipeGroups(options.Groups, recipes)
	controlViews := styleControlViews(recipes)
	groupNodes := make([]gosx.Node, 0, len(groups))
	for _, group := range groups {
		controlNodes := make([]gosx.Node, 0, len(group.Controls))
		for _, binding := range group.Controls {
			control, ok := controlViews[normalizeKey(binding.ControlKey)]
			if !ok {
				continue
			}
			controlNodes = append(controlNodes, renderStyleControlGroup(control, binding, options.Values))
		}
		if len(controlNodes) == 0 {
			continue
		}
		groupNodes = append(groupNodes, renderStyleRecipeGroup(group, controlViews, options.Values, controlNodes))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-style-workbench", "true"),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-panel-heading")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(firstNonEmpty(options.Kicker, "Recipes"))),
			gosx.El("h3", nil, gosx.Text(firstNonEmpty(options.Title, "Visual system"))),
		),
		renderStyleImpact(options.Impact),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-recipe-grid")), gosx.Fragment(groupNodes...)),
	)
}

func DefaultStyleStates() []StyleState {
	return []StyleState{
		{Key: "default", Label: "Default", Active: true},
		{Key: "hover", Label: "Hover"},
		{Key: "focus", Label: "Focus"},
	}
}

func StyleSelectControls(recipes []cmsstyle.RecipeView, bindings []StyleControlBinding, values map[string]string, options StyleSelectControlOptions) []SelectControl {
	controls := styleControlViews(recipes)
	bindings = normalizeStyleControlBindings(bindings)
	out := make([]SelectControl, 0, len(bindings))
	for _, binding := range bindings {
		control, ok := controls[binding.ControlKey]
		if !ok || len(control.Options) == 0 {
			continue
		}
		fieldName := firstNonEmpty(binding.FieldName, control.Key)
		label := firstNonEmpty(binding.Label, control.Label)
		value := firstNonEmpty(values[fieldName], control.Default)
		attrs := append([]FieldAttribute{}, options.Attrs...)
		attrs = append(attrs, binding.Attrs...)
		attrs = appendStyleSourceAttr(attrs, options.SourcePrefix, fieldName)
		out = append(out, SelectControl{
			ID:      fieldName,
			Name:    fieldName,
			Label:   label,
			Class:   options.Class,
			Wide:    options.Wide || binding.Wide,
			Attrs:   attrs,
			Options: styleStudioOptions(control, value),
		})
	}
	return out
}

func StyleRadioControlGroup(recipes []cmsstyle.RecipeView, binding StyleControlBinding, values map[string]string, options StyleRadioControlOptions) RadioControlGroup {
	bindings := normalizeStyleControlBindings([]StyleControlBinding{binding})
	if len(bindings) == 0 {
		return RadioControlGroup{}
	}
	binding = bindings[0]
	control, ok := styleControlViews(recipes)[binding.ControlKey]
	if !ok || len(control.Options) == 0 {
		return RadioControlGroup{}
	}
	fieldName := firstNonEmpty(options.Name, binding.FieldName, control.Key)
	value := firstNonEmpty(values[fieldName], control.Default)
	attrs := append([]FieldAttribute{}, options.Attrs...)
	attrs = append(attrs, binding.Attrs...)
	attrs = appendStyleSourceAttr(attrs, options.SourcePrefix, fieldName)
	return RadioControlGroup{
		Class:   options.Class,
		Legend:  firstNonEmpty(options.Legend, binding.Label, control.Label),
		Name:    fieldName,
		Attrs:   attrs,
		Options: styleStudioOptions(control, value),
	}
}

func styleScopeCell(label, value, dataAttr string) gosx.Node {
	outputAttrs := []any{}
	if dataAttr != "" {
		outputAttrs = append(outputAttrs, gosx.Attr(dataAttr, "true"))
	}
	return gosx.El("span", nil,
		gosx.El("small", nil, gosx.Text(label)),
		gosx.El("output", gosx.Attrs(outputAttrs...), gosx.Text(value)),
	)
}

func renderStyleImpact(options StyleImpactOptions) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-style-impact"),
		gosx.Attr("data-studio-style-impact-panel", "true"),
		gosx.Attr("aria-live", "polite"),
	),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(firstNonEmpty(options.Kicker, "Preview impact"))),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-style-impact-label", "true")), gosx.Text(firstNonEmpty(options.Label, "No active recipe"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-style-impact-summary", "true")), gosx.Text(firstNonEmpty(options.Summary, "Awaiting scoped change."))),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-impact__meta")),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-impact-count", "true")), gosx.Text(firstNonEmpty(options.CountLabel, "0 affected"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-style-impact-scope", "true")), gosx.Text(firstNonEmpty(options.ScopeLabel, "site > home"))),
			gosx.El("span", gosx.Attrs(gosx.Attr("data-studio-style-impact-state", "true")), gosx.Text(firstNonEmpty(options.StateLabel, "default / desktop"))),
		),
	)
}

func renderStyleRecipeGroup(group StyleRecipeGroup, controls map[string]cmsstyle.ControlView, values map[string]string, controlNodes []gosx.Node) gosx.Node {
	readoutField := styleReadoutField(group, controls)
	readoutControl := styleReadoutControl(group, controls)
	readoutLabel := styleReadoutLabel(readoutControl, values[readoutField])
	visualClass := firstNonEmpty(group.VisualClass, "studio-style-visual--"+group.Key)
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", "studio-style-recipe-card"),
		gosx.Attr("data-studio-style-recipe", group.Key),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-recipe-card__head")),
			gosx.El("strong", nil, gosx.Text(group.Label)),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-style-readout", readoutField)), gosx.Text(readoutLabel)),
		),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "studio-style-visual "+visualClass),
			gosx.Attr("aria-hidden", "true"),
		), gosx.Fragment(styleVisualMarks(group.VisualMarks)...)),
		gosx.Fragment(controlNodes...),
	)
}

func renderStyleControlGroup(control cmsstyle.ControlView, binding StyleControlBinding, values map[string]string) gosx.Node {
	fieldName := firstNonEmpty(binding.FieldName, control.Key)
	label := firstNonEmpty(binding.Label, control.Label)
	optionNodes := make([]gosx.Node, 0, len(control.Options))
	for _, option := range control.Options {
		optionNodes = append(optionNodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-style-control", fieldName),
			gosx.Attr("data-studio-style-value", option.Value),
		), gosx.Text(styleLabel(option.Label, option.Value))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-control-group")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-control-group__head")),
			gosx.El("span", nil, gosx.Text(label)),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("type", "button"),
				gosx.Attr("data-studio-style-reset", fieldName),
			), gosx.Text("Reset")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-style-choice-row")), gosx.Fragment(optionNodes...)),
	)
}

func normalizeStyleStates(states []StyleState) []StyleState {
	if len(states) == 0 {
		states = DefaultStyleStates()
	}
	out := make([]StyleState, 0, len(states))
	hasActive := false
	for _, state := range states {
		state.Key = normalizeKey(state.Key)
		state.Label = strings.TrimSpace(state.Label)
		if state.Key == "" || state.Label == "" {
			continue
		}
		if state.Active {
			if hasActive {
				state.Active = false
			} else {
				hasActive = true
			}
		}
		out = append(out, state)
	}
	if len(out) > 0 && !hasActive {
		out[0].Active = true
	}
	return out
}

func normalizeStyleRecipeGroups(groups []StyleRecipeGroup, recipes []cmsstyle.RecipeView) []StyleRecipeGroup {
	if len(groups) == 0 {
		for _, recipe := range recipes {
			group := StyleRecipeGroup{
				Key:         recipe.Key,
				Label:       recipe.Label,
				VisualMarks: 3,
				Controls:    make([]StyleControlBinding, 0, len(recipe.Controls)),
			}
			for _, control := range recipe.Controls {
				group.Controls = append(group.Controls, StyleControlBinding{ControlKey: control.Key})
			}
			groups = append(groups, group)
		}
	}
	out := make([]StyleRecipeGroup, 0, len(groups))
	for _, group := range groups {
		group.Key = normalizeKey(group.Key)
		group.Label = strings.TrimSpace(group.Label)
		group.VisualClass = strings.TrimSpace(group.VisualClass)
		group.ReadoutControlKey = normalizeKey(group.ReadoutControlKey)
		if group.VisualMarks <= 0 {
			group.VisualMarks = 3
		}
		if group.Key == "" || group.Label == "" {
			continue
		}
		group.Controls = normalizeStyleControlBindings(group.Controls)
		out = append(out, group)
	}
	return out
}

func normalizeStyleControlBindings(bindings []StyleControlBinding) []StyleControlBinding {
	out := make([]StyleControlBinding, 0, len(bindings))
	for _, binding := range bindings {
		binding.ControlKey = normalizeKey(binding.ControlKey)
		binding.FieldName = strings.TrimSpace(binding.FieldName)
		binding.Label = strings.TrimSpace(binding.Label)
		if binding.ControlKey == "" {
			continue
		}
		out = append(out, binding)
	}
	return out
}

func styleControlViews(recipes []cmsstyle.RecipeView) map[string]cmsstyle.ControlView {
	out := map[string]cmsstyle.ControlView{}
	for _, recipe := range recipes {
		for _, control := range recipe.Controls {
			key := normalizeKey(control.Key)
			if key != "" {
				out[key] = control
			}
		}
	}
	return out
}

func styleReadoutField(group StyleRecipeGroup, controls map[string]cmsstyle.ControlView) string {
	controlKey := firstNonEmpty(group.ReadoutControlKey, firstGroupControlKey(group))
	for _, binding := range group.Controls {
		if binding.ControlKey == controlKey {
			return firstNonEmpty(binding.FieldName, binding.ControlKey)
		}
	}
	return controlKey
}

func styleReadoutControl(group StyleRecipeGroup, controls map[string]cmsstyle.ControlView) cmsstyle.ControlView {
	controlKey := firstNonEmpty(group.ReadoutControlKey, firstGroupControlKey(group))
	return controls[controlKey]
}

func styleReadoutLabel(control cmsstyle.ControlView, value string) string {
	value = strings.TrimSpace(value)
	for _, option := range control.Options {
		if option.Value == value {
			return styleLabel(option.Label, option.Value)
		}
	}
	if value == "" && control.Default != "" {
		for _, option := range control.Options {
			if option.Value == control.Default {
				return styleLabel(option.Label, option.Value)
			}
		}
	}
	return styleLabel(value, value)
}

func firstGroupControlKey(group StyleRecipeGroup) string {
	if len(group.Controls) == 0 {
		return ""
	}
	return group.Controls[0].ControlKey
}

func activeStyleStateLabel(states []StyleState) string {
	for _, state := range states {
		if state.Active {
			return state.Label
		}
	}
	return "Default"
}

func styleVisualMarks(count int) []gosx.Node {
	if count <= 0 {
		count = 3
	}
	nodes := make([]gosx.Node, 0, count)
	for i := 0; i < count; i++ {
		nodes = append(nodes, gosx.El("span", nil))
	}
	return nodes
}

func styleLabel(label, fallback string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		label = strings.TrimSpace(fallback)
	}
	label = strings.ReplaceAll(label, "-", " ")
	words := strings.Fields(label)
	for index, word := range words {
		if len(word) == 0 {
			continue
		}
		words[index] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func styleStudioOptions(control cmsstyle.ControlView, value string) []StudioOption {
	value = strings.TrimSpace(value)
	out := make([]StudioOption, 0, len(control.Options))
	hasSelected := false
	for _, option := range control.Options {
		selected := option.Value == value || (value == "" && option.Value == control.Default)
		if selected {
			hasSelected = true
		}
		out = append(out, StudioOption{
			Value:    option.Value,
			Label:    styleLabel(option.Label, option.Value),
			Selected: selected,
			Attrs:    []FieldAttribute{{Name: "data-studio-style-css", Value: firstNonEmpty(option.CSS, option.Value)}},
		})
	}
	if !hasSelected && control.Default != "" {
		for index := range out {
			if out[index].Value == control.Default {
				out[index].Selected = true
				break
			}
		}
	}
	return out
}

func appendStyleSourceAttr(attrs []FieldAttribute, prefix, fieldName string) []FieldAttribute {
	fieldName = strings.TrimSpace(fieldName)
	if fieldName == "" || strings.TrimSpace(prefix) == "" || fieldAttributeValue(attrs, "data-studio-field-source") != "" {
		return attrs
	}
	return append(attrs, FieldAttribute{Name: "data-studio-field-source", Value: inspectorFieldSource(prefix, fieldName)})
}
