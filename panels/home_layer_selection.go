package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
)

type HomeLayerSelectionItem struct {
	Key   string
	Label string
}

type HomeLayerSelectionProps struct {
	DefaultSelectedKey   string
	DefaultSelectedLabel string
	Items                []HomeLayerSelectionItem
}

func HomeLayerSelectionPropsFromMap(view map[string]any) HomeLayerSelectionProps {
	return HomeLayerSelectionProps{
		DefaultSelectedKey:   core.WorkbenchViewString(view, "defaultSelectedKey"),
		DefaultSelectedLabel: core.WorkbenchViewString(view, "defaultSelectedLabel"),
		Items:                homeLayerSelectionItemsFromMap(view),
	}
}

func RenderHomeLayerSelection(props HomeLayerSelectionProps) gosx.Node {
	children := []gosx.Node{
		gosx.El("output", gosx.Attrs(
			gosx.Attr("data-studio-selection-label", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(props.DefaultSelectedLabel)),
		renderHomeLayerSelectionButtons(props.Items, props.DefaultSelectedKey),
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-home-layer-picker"),
		gosx.Attr("data-studio-home-layer-picker", "true"),
		gosx.Attr("data-studio-home-layer-selected", props.DefaultSelectedKey),
	), gosx.Fragment(children...))
}

func homeLayerSelectionItemsFromMap(view map[string]any) []HomeLayerSelectionItem {
	items := core.WorkbenchViewMapList(view, "items")
	out := make([]HomeLayerSelectionItem, 0, len(items))
	for _, item := range items {
		out = append(out, HomeLayerSelectionItem{
			Key:   core.WorkbenchViewString(item, "key"),
			Label: core.WorkbenchViewString(item, "label"),
		})
	}
	return out
}

func renderHomeLayerSelectionButtons(items []HomeLayerSelectionItem, selectedKey string) gosx.Node {
	buttons := make([]gosx.Node, 0, len(items))
	for _, item := range items {
		buttons = append(buttons, gosx.El("button", gosx.Attrs(
			gosx.Attr("type", "button"),
			gosx.Attr("data-studio-home-layer-pick", item.Key),
			gosx.Attr("aria-pressed", core.BoolAttr(item.Key == selectedKey)),
		), gosx.Text(item.Label)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "studio-home-layer-picker__list"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Select home section"),
	), gosx.Fragment(buttons...))
}
