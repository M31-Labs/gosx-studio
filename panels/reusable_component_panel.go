package panels

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
)

// ReusableComponentPanelOptions projects host-persisted reusable state into a
// selected-canvas library and instance inspector. CreateInstanceIDs are host-
// issued stable IDs; the panel never fabricates identity in browser storage.
type ReusableComponentPanelOptions struct {
	ID                 string
	Action             string
	CSRFToken          string
	Definitions        []core.ComponentDefinition
	Instances          []core.ComponentInstance
	SelectedInstanceID string
	TargetPageKey      string
	TargetRegion       string
	NextPosition       int
	CreateInstanceIDs  map[string]string
	OperationIDs       map[string]string
	Heads              map[string]string
}

func RenderReusableComponentPanel(options ReusableComponentPanelOptions) gosx.Node {
	if strings.TrimSpace(options.ID) == "" {
		options.ID = "studio-reusable-components"
	}
	definitions := normalizedReusableDefinitions(options.Definitions)
	instances := normalizedReusableInstances(options.Instances)
	selected, hasSelected := selectedReusableInstance(instances, options.SelectedInstanceID)
	children := []gosx.Node{
		gosx.El("header", gosx.Attrs(gosx.Attr("class", "studio-reusable-panel__head")),
			gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Components")), gosx.El("h2", nil, gosx.Text("Reusable library"))),
			gosx.El("output", nil, gosx.Text(fmt.Sprintf("%d definitions · %d instances", len(definitions), len(instances)))),
		),
		gosx.El("p", nil, gosx.Text("Create linked instances, keep deliberate overrides, or detach an exact materialized copy from the selected canvas object.")),
		renderReusableLibrary(options, definitions, instances),
	}
	if hasSelected {
		children = append(children, renderReusableInstanceInspector(options, definitions, selected))
	} else {
		children = append(children, gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-reusable-instance-empty", "true")), gosx.El("h3", nil, gosx.Text("No reusable instance selected")), gosx.El("p", nil, gosx.Text("Select a linked canvas component to inspect its definition, overrides, and detach state."))))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("id", options.ID),
		gosx.Attr("class", "editor-panel studio-reusable-panel"),
		gosx.Attr("data-studio-reusable-component-panel", "true"),
		gosx.Attr("data-studio-selected-instance", strings.TrimSpace(options.SelectedInstanceID)),
		gosx.Attr("data-gosx-studio-reusable-panel-renderer", "gosx-studio"),
	), gosx.Fragment(children...))
}

func renderReusableLibrary(options ReusableComponentPanelOptions, definitions []core.ComponentDefinition, instances []core.ComponentInstance) gosx.Node {
	children := []gosx.Node{gosx.El("header", nil, gosx.El("h3", nil, gosx.Text("Definitions")), gosx.El("p", nil, gosx.Text("Attached instances receive definition updates while keeping explicit instance overrides.")))}
	if len(definitions) == 0 {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text("No reusable definitions are available from the host store.")))
	}
	for _, definition := range definitions {
		count := 0
		for _, instance := range instances {
			if instance.DefinitionID == definition.ID && !instance.Detached {
				count++
			}
		}
		body := []gosx.Node{
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(definition.Label)),
				gosx.El("code", nil, gosx.Text("v"+strconv.FormatUint(definition.Version, 10))),
			),
			gosx.El("p", nil, gosx.Text(fmt.Sprintf("%s template · %d attached", definition.TemplateKey, count))),
		}
		instanceID := strings.TrimSpace(options.CreateInstanceIDs[definition.ID])
		if instanceID != "" && options.Action != "" && options.TargetPageKey != "" && options.TargetRegion != "" {
			body = append(body, reusableOperationForm(options, authoring.ReusableCreateInstance, instanceID, definition.ID, "",
				[]gosx.Node{hidden(authoring.ReusableFieldPageKey, options.TargetPageKey), hidden(authoring.ReusableFieldRegion, options.TargetRegion), hidden(authoring.ReusableFieldPosition, strconv.Itoa(options.NextPosition))},
				"Create linked instance"))
		}
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("data-studio-reusable-definition", definition.ID), gosx.Attr("data-studio-definition-version", definition.Version), gosx.Attr("data-studio-definition-attached-count", count)), gosx.Fragment(body...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-reusable-library", "true")), gosx.Fragment(children...))
}

func renderReusableInstanceInspector(options ReusableComponentPanelOptions, definitions []core.ComponentDefinition, instance core.ComponentInstance) gosx.Node {
	definition, found := reusableDefinitionByID(definitions, instance.DefinitionID)
	status := "Attached"
	if instance.Detached {
		status = "Detached copy"
	}
	children := []gosx.Node{
		gosx.El("header", nil,
			gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Selected canvas instance")), gosx.El("h3", nil, gosx.Text(reusableInstanceLabel(instance, definition, found)))),
			gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-reusable-instance-state", strings.ToLower(strings.ReplaceAll(status, " ", "-")))), gosx.Text(status)),
		),
		gosx.El("dl", nil,
			gosx.El("div", nil, gosx.El("dt", nil, gosx.Text("Canvas identity")), gosx.El("dd", nil, gosx.El("code", nil, gosx.Text(instance.CanvasIdentity())))),
			gosx.El("div", nil, gosx.El("dt", nil, gosx.Text("Definition")), gosx.El("dd", nil, gosx.Text(instance.DefinitionID+" · v"+strconv.FormatUint(instance.DefinitionVersion, 10)))),
			gosx.El("div", nil, gosx.El("dt", nil, gosx.Text("Placement")), gosx.El("dd", nil, gosx.Text(instance.PageKey+" / "+instance.Region+" · "+strconv.Itoa(instance.Position)))),
		),
		renderReusableOverrides(options, definition, found, instance),
	}
	if options.Action != "" {
		kind, label := authoring.ReusableDetachInstance, "Detach exact copy"
		if instance.Detached {
			kind, label = authoring.ReusableRestoreInstance, "Restore definition link"
		}
		children = append(children, reusableOperationForm(options, kind, instance.ID, instance.DefinitionID, "", nil, label))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("data-studio-reusable-instance-inspector", "true"),
		gosx.Attr("data-studio-canvas-identity", instance.CanvasIdentity()),
		gosx.Attr("data-studio-instance-head-revision", instance.HeadRevision),
	), gosx.Fragment(children...))
}

func renderReusableOverrides(options ReusableComponentPanelOptions, definition core.ComponentDefinition, found bool, instance core.ComponentInstance) gosx.Node {
	children := []gosx.Node{gosx.El("header", nil, gosx.El("h4", nil, gosx.Text("Instance overrides")), gosx.El("p", nil, gosx.Text("Overrides are explicit. Clear one to inherit from the definition again.")))}
	keys := make([]string, 0, len(instance.Overrides))
	for key := range instance.Overrides {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text("This instance fully follows its definition.")))
	}
	for _, key := range keys {
		override := instance.Overrides[key]
		row := []gosx.Node{gosx.El("div", nil, gosx.El("code", nil, gosx.Text(key)), gosx.El("output", nil, gosx.Text(override.Value)))}
		if options.Action != "" {
			row = append(row, reusableOperationForm(options, authoring.ReusableClearOverride, instance.ID, instance.DefinitionID, key, []gosx.Node{hidden(authoring.ReusableFieldOverrideKey, key)}, "Clear override"))
		}
		children = append(children, gosx.El("article", gosx.Attrs(gosx.Attr("data-studio-instance-override", key), gosx.Attr("data-studio-override-present", core.BoolAttr(override.Present))), gosx.Fragment(row...)))
	}
	if options.Action != "" && found {
		choices := reusableOverrideChoices(definition)
		choiceNodes := make([]gosx.Node, 0, len(choices))
		for _, choice := range choices {
			choiceNodes = append(choiceNodes, gosx.El("option", gosx.Attrs(gosx.Attr("value", choice)), gosx.Text(choice)))
		}
		formChildren := reusableOperationInputs(options, authoring.ReusableSetInstanceOverride, instance.ID, instance.DefinitionID, "")
		formChildren = append(formChildren,
			gosx.El("label", nil, gosx.Text("Override target"), gosx.El("select", gosx.Attrs(gosx.Attr("name", authoring.ReusableFieldOverrideKey)), gosx.Fragment(choiceNodes...))),
			gosx.El("label", nil, gosx.Text("Override value"), gosx.El("input", gosx.Attrs(gosx.Attr("name", authoring.ReusableFieldOverrideValue), gosx.Attr("type", "text"), gosx.Attr("value", "")))),
			gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Set explicit override")),
		)
		children = append(children, gosx.El("form", gosx.Attrs(reusableOperationFormAttrs(options)...), gosx.Fragment(formChildren...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("data-studio-reusable-overrides", "true")), gosx.Fragment(children...))
}

func reusableOperationForm(options ReusableComponentPanelOptions, kind authoring.ReusableOperationKind, instanceID, definitionID, discriminator string, extra []gosx.Node, label string) gosx.Node {
	children := reusableOperationInputs(options, kind, instanceID, definitionID, discriminator)
	children = append(children, extra...)
	children = append(children, gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text(label)))
	return gosx.El("form", gosx.Attrs(reusableOperationFormAttrs(options)...), gosx.Fragment(children...))
}

func reusableOperationInputs(options ReusableComponentPanelOptions, kind authoring.ReusableOperationKind, instanceID, definitionID, discriminator string) []gosx.Node {
	target := "instance:" + strings.TrimSpace(instanceID)
	if kind == authoring.ReusableCreateDefinition || kind == authoring.ReusableUpdateDefinition {
		target = "definition:" + strings.TrimSpace(definitionID)
	}
	children := []gosx.Node{
		hidden(authoring.ReusableFieldOperation, string(kind)),
		hidden(authoring.ReusableFieldOperationID, options.OperationIDs[ReusablePanelOperationKey(kind, instanceID, discriminator)]),
		hidden(authoring.ReusableFieldInstanceID, instanceID),
		hidden(authoring.ReusableFieldDefinitionID, definitionID),
		hidden(authoring.ReusableFieldExpectedHead, options.Heads[target]),
	}
	if options.CSRFToken != "" {
		children = append(children, hidden("csrf_token", options.CSRFToken))
	}
	return children
}

// ReusablePanelOperationKey lets hosts issue stable next-operation IDs without
// embedding state generation in the browser. Clear operations include the
// override key discriminator so parallel clear forms never reuse an ID.
func ReusablePanelOperationKey(kind authoring.ReusableOperationKind, instanceID, discriminator string) string {
	return strings.Join([]string{string(kind), strings.TrimSpace(instanceID), strings.TrimSpace(discriminator)}, ":")
}

func reusableOperationFormAttrs(options ReusableComponentPanelOptions) []any {
	return []any{gosx.Attr("method", "post"), gosx.Attr("action", options.Action), gosx.Attr("data-gosx-studio-reusable-operation-form", "true")}
}

func normalizedReusableDefinitions(values []core.ComponentDefinition) []core.ComponentDefinition {
	out := make([]core.ComponentDefinition, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.ID != "" {
			out = append(out, value)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

func normalizedReusableInstances(values []core.ComponentInstance) []core.ComponentInstance {
	out := make([]core.ComponentInstance, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.ID != "" {
			out = append(out, value)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PageKey != out[j].PageKey {
			return out[i].PageKey < out[j].PageKey
		}
		if out[i].Region != out[j].Region {
			return out[i].Region < out[j].Region
		}
		return out[i].Position < out[j].Position
	})
	return out
}

func selectedReusableInstance(values []core.ComponentInstance, id string) (core.ComponentInstance, bool) {
	id = strings.TrimSpace(id)
	for _, value := range values {
		if value.ID == id {
			return value, true
		}
	}
	return core.ComponentInstance{}, false
}

func reusableDefinitionByID(values []core.ComponentDefinition, id string) (core.ComponentDefinition, bool) {
	for _, value := range values {
		if value.ID == id {
			return value, true
		}
	}
	return core.ComponentDefinition{}, false
}

func reusableInstanceLabel(instance core.ComponentInstance, definition core.ComponentDefinition, found bool) string {
	if found && definition.Label != "" {
		return definition.Label
	}
	return instance.ID
}

func reusableOverrideChoices(definition core.ComponentDefinition) []string {
	choices := []string{"binding"}
	for _, control := range definition.Controls {
		if control.Key != "" {
			choices = append(choices, "control:"+control.Key)
		}
	}
	for scope, properties := range definition.Styles {
		for property := range properties {
			choices = append(choices, "style:"+scope+":"+property)
		}
	}
	for scope, properties := range definition.Layout {
		for property := range properties {
			choices = append(choices, "layout:"+scope+":"+property)
		}
	}
	sort.Strings(choices)
	return choices
}
