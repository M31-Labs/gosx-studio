package studio

type HomeLayerSelectionItem struct {
	Key   string
	Label string
}

type HomeLayerSelectionProps struct {
	DefaultSelectedKey   string
	DefaultSelectedLabel string
	Items                []HomeLayerSelectionItem
}
