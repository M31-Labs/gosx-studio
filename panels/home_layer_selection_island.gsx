package panels

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx/signal"
)

//gosx:island
func HomeLayerSelectionIsland(props HomeLayerSelectionProps) gosx.Node {
	selected := signal.New(props.DefaultSelectedKey)
	selectedLabel := signal.New(props.DefaultSelectedLabel)

	return <div class="studio-home-layer-picker" data-studio-home-layer-selected={selected.Get()}>
		<output data-studio-selection-label="true" aria-live="polite">{selectedLabel.Get()}</output>
		<div class="studio-home-layer-picker__list" role="toolbar" aria-label="Select home section">
			<Each of={props.Items} as="layer">
				<button
					type="button"
					data-studio-home-layer-pick={layer.Key}
					aria-pressed={selected.Get() == layer.Key}
					onClick={func() { selected.Set(layer.Key); selectedLabel.Set(layer.Label) }}
				>{layer.Label}</button>
			</Each>
		</div>
	</div>
}
