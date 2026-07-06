package studio

// This file is the deprecated compatibility facade for the symbols that used
// to live in activity_panel.go, advanced_panel.go, block_library_panel.go,
// brand_panel.go, brand_media_picker.go, brand_media_picker_island.gsx,
// checkout_panel.go, flow_designer.go, flow_designer_panel.go,
// flow_designer_island.gsx, home_inspector_panel.go, home_layers_panel.go,
// home_layer_selection.go, home_layer_selection_island.gsx, inspector.go,
// inspector_chrome_panel.go, inspector_fields.go, look_panel.go,
// navigation_panel.go, preview_share_panel.go, publish_panel.go,
// revision_history_panel.go, and style_panel.go before Slice 6 of the
// package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) extracted them into
// the m31labs.dev/gosx-studio/panels package: the editor inspector/designer
// panels and their .gsx islands (right/left rail content units).
//
// Every exported type below is a type ALIAS (not a new type), so struct
// literals, embedding, method sets, and assignability are all unchanged for
// existing callers (muddy-noni-commerce, pajaritos-forest-school,
// gosx-cms/studio). Every exported free function is a thin forwarding
// wrapper. Methods on the aliased types are inherited automatically through
// the alias and do not need wrappers here. Every exported constant keeps its
// exact string value (rendered data-studio-*/data-gosx-studio-* attribute
// names and values are part of the rendered-page contract and must not
// change — spec risk R5).
//
// panels imports only core and authoring (see the spec's §1 Import DAG):
// where the moved panel files used to call small map-shape/hidden-input
// helpers whose canonical implementation now lives in sitemap (Slice 5) or
// still lives at root as shell territory (Slice 8), or a block-layout
// open-tag helper now canonical in canvas (Slice 4), panels carries its own
// frozen, byte-identical copies instead of importing those peer/upward
// packages — see panels/support.go.
//
// flow_designer_island.gsx, brand_media_picker_island.gsx, and
// home_layer_selection_island.gsx moved with their package decls rewritten
// to `package panels`; each declares an Island function via the gosx
// toolchain (`//gosx:island`), which is not a linkable Go symbol in either
// the old or new package (no .go file ever referenced FlowDesignerIsland,
// BrandMediaPickerIsland, or HomeLayerSelectionIsland by name — only the
// source-contract tests assert their literal text), so there is no Go-level
// alias to add for them here (matches Slice 5's identical finding for
// site_navigator_island.gsx/SiteNavigatorIsland).
//
// Deprecated: import m31labs.dev/gosx-studio/panels directly; this facade
// will be removed after the v0.6.x compatibility window (see spec §9).

import (
	"m31labs.dev/gosx"

	"m31labs.dev/gosx-studio/panels"
)

// --- Activity panel (panels/activity_panel.go) ---

// Deprecated: use panels.ActivityPanelOptions.
type ActivityPanelOptions = panels.ActivityPanelOptions

// Deprecated: use panels.RenderActivityPanel.
func RenderActivityPanel(view map[string]any, options ActivityPanelOptions) gosx.Node {
	return panels.RenderActivityPanel(view, options)
}

// --- Advanced panel (panels/advanced_panel.go) ---

// Deprecated: use panels.AdvancedPanelOptions.
type AdvancedPanelOptions = panels.AdvancedPanelOptions

// Deprecated: use panels.AdvancedPanelSegments.
type AdvancedPanelSegments = panels.AdvancedPanelSegments

// Deprecated: use panels.AdvancedToolsPanelOptions.
type AdvancedToolsPanelOptions = panels.AdvancedToolsPanelOptions

// Deprecated: use panels.AdvancedFieldPanelOptions.
type AdvancedFieldPanelOptions = panels.AdvancedFieldPanelOptions

// Deprecated: use panels.AdvancedSettingsPanelOptions.
type AdvancedSettingsPanelOptions = panels.AdvancedSettingsPanelOptions

// Deprecated: use panels.RenderAdvancedPanel.
func RenderAdvancedPanel(view map[string]any, options AdvancedPanelOptions) gosx.Node {
	return panels.RenderAdvancedPanel(view, options)
}

// Deprecated: use panels.RenderAdvancedPanelSegments.
func RenderAdvancedPanelSegments(view map[string]any, options AdvancedPanelOptions) AdvancedPanelSegments {
	return panels.RenderAdvancedPanelSegments(view, options)
}

// Deprecated: use panels.RenderAdvancedToolsPanel.
func RenderAdvancedToolsPanel(view map[string]any, options AdvancedToolsPanelOptions) gosx.Node {
	return panels.RenderAdvancedToolsPanel(view, options)
}

// Deprecated: use panels.RenderAdvancedFieldPanel.
func RenderAdvancedFieldPanel(view map[string]any, options AdvancedFieldPanelOptions) gosx.Node {
	return panels.RenderAdvancedFieldPanel(view, options)
}

// Deprecated: use panels.RenderAdvancedSettingsPanel.
func RenderAdvancedSettingsPanel(view map[string]any, options AdvancedSettingsPanelOptions) gosx.Node {
	return panels.RenderAdvancedSettingsPanel(view, options)
}

// --- Block library panel (panels/block_library_panel.go) ---

// Deprecated: use panels.BlockLibraryPanelOptions.
type BlockLibraryPanelOptions = panels.BlockLibraryPanelOptions

// Deprecated: use panels.RenderBlockLibraryPanel.
func RenderBlockLibraryPanel(view map[string]any, options BlockLibraryPanelOptions) gosx.Node {
	return panels.RenderBlockLibraryPanel(view, options)
}

// --- Brand panel (panels/brand_panel.go) ---

// Deprecated: use panels.BrandPanelOptions.
type BrandPanelOptions = panels.BrandPanelOptions

// Deprecated: use panels.BrandPanelSegments.
type BrandPanelSegments = panels.BrandPanelSegments

// Deprecated: use panels.RenderBrandPanel.
func RenderBrandPanel(view map[string]any, fields map[string]any, options BrandPanelOptions) gosx.Node {
	return panels.RenderBrandPanel(view, fields, options)
}

// Deprecated: use panels.RenderBrandPanelSegments.
func RenderBrandPanelSegments(view map[string]any, fields map[string]any, options BrandPanelOptions) BrandPanelSegments {
	return panels.RenderBrandPanelSegments(view, fields, options)
}

// --- Brand media picker (panels/brand_media_picker.go, panels/brand_media_picker_island.gsx) ---

// Deprecated: use panels.BrandMediaPickerAssetItem.
type BrandMediaPickerAssetItem = panels.BrandMediaPickerAssetItem

// Deprecated: use panels.BrandMediaPickerProps.
type BrandMediaPickerProps = panels.BrandMediaPickerProps

// Deprecated: use panels.BrandMediaPickerOptions.
type BrandMediaPickerOptions = panels.BrandMediaPickerOptions

// Deprecated: use panels.RenderBrandMediaPicker.
func RenderBrandMediaPicker(view map[string]any, options BrandMediaPickerOptions) gosx.Node {
	return panels.RenderBrandMediaPicker(view, options)
}

// --- Checkout panel (panels/checkout_panel.go) ---

// Deprecated: use panels.CheckoutPanelOptions.
type CheckoutPanelOptions = panels.CheckoutPanelOptions

// Deprecated: use panels.RenderCheckoutPanel.
func RenderCheckoutPanel(view map[string]any, options CheckoutPanelOptions) gosx.Node {
	return panels.RenderCheckoutPanel(view, options)
}

// --- Flow designer contracts (panels/flow_designer.go) ---

// Deprecated: use panels.FlowDesignerProps.
type FlowDesignerProps = panels.FlowDesignerProps

// Deprecated: use panels.FlowDesignerFlow.
type FlowDesignerFlow = panels.FlowDesignerFlow

// Deprecated: use panels.FlowDesignerReadiness.
type FlowDesignerReadiness = panels.FlowDesignerReadiness

// Deprecated: use panels.FlowDesignerCheck.
type FlowDesignerCheck = panels.FlowDesignerCheck

// Deprecated: use panels.FlowDesignerNode.
type FlowDesignerNode = panels.FlowDesignerNode

// Deprecated: use panels.FlowDesignerStep.
type FlowDesignerStep = panels.FlowDesignerStep

// Deprecated: use panels.FlowDesignerAction.
type FlowDesignerAction = panels.FlowDesignerAction

// Deprecated: use panels.FlowDesignerField.
type FlowDesignerField = panels.FlowDesignerField

// --- Flow designer panel (panels/flow_designer_panel.go, panels/flow_designer_island.gsx) ---

// Deprecated: use panels.FlowDesignerPanelOptions.
type FlowDesignerPanelOptions = panels.FlowDesignerPanelOptions

// Deprecated: use panels.RenderFlowDesignerPanel.
func RenderFlowDesignerPanel(view map[string]any, options FlowDesignerPanelOptions) gosx.Node {
	return panels.RenderFlowDesignerPanel(view, options)
}

// Deprecated: use panels.RenderFlowDesignerHeader.
func RenderFlowDesignerHeader() gosx.Node {
	return panels.RenderFlowDesignerHeader()
}

// Deprecated: use panels.RenderFlowDesignerReadiness.
func RenderFlowDesignerReadiness(flow map[string]any) gosx.Node {
	return panels.RenderFlowDesignerReadiness(flow)
}

// Deprecated: use panels.RenderFlowDesignerGraph.
func RenderFlowDesignerGraph(flow map[string]any) gosx.Node {
	return panels.RenderFlowDesignerGraph(flow)
}

// Deprecated: use panels.RenderFlowDesignerRoute.
func RenderFlowDesignerRoute(flow map[string]any, formID string) gosx.Node {
	return panels.RenderFlowDesignerRoute(flow, formID)
}

// Deprecated: use panels.RenderFlowDesignerActions.
func RenderFlowDesignerActions(flow map[string]any) gosx.Node {
	return panels.RenderFlowDesignerActions(flow)
}

// --- Home inspector panel (panels/home_inspector_panel.go) ---

// Deprecated: use panels.HomeInspectorPanelOptions.
type HomeInspectorPanelOptions = panels.HomeInspectorPanelOptions

// Deprecated: use panels.RenderHomeInspectorPanel.
func RenderHomeInspectorPanel(view map[string]any, contentFields map[string]any, options HomeInspectorPanelOptions) gosx.Node {
	return panels.RenderHomeInspectorPanel(view, contentFields, options)
}

// --- Home layers panel (panels/home_layers_panel.go) ---

// Deprecated: use panels.HomeLayersPanelOptions.
type HomeLayersPanelOptions = panels.HomeLayersPanelOptions

// Deprecated: use panels.HomeLayersPanelSegments.
type HomeLayersPanelSegments = panels.HomeLayersPanelSegments

// Deprecated: use panels.RenderHomeLayersPanelSegments.
func RenderHomeLayersPanelSegments(view map[string]any, options HomeLayersPanelOptions) HomeLayersPanelSegments {
	return panels.RenderHomeLayersPanelSegments(view, options)
}

// Deprecated: use panels.RenderHomeLayersPanel.
func RenderHomeLayersPanel(view map[string]any, options HomeLayersPanelOptions) gosx.Node {
	return panels.RenderHomeLayersPanel(view, options)
}

// --- Home layer selection (panels/home_layer_selection.go, panels/home_layer_selection_island.gsx) ---

// Deprecated: use panels.HomeLayerSelectionItem.
type HomeLayerSelectionItem = panels.HomeLayerSelectionItem

// Deprecated: use panels.HomeLayerSelectionProps.
type HomeLayerSelectionProps = panels.HomeLayerSelectionProps

// Deprecated: use panels.HomeLayerSelectionPropsFromMap.
func HomeLayerSelectionPropsFromMap(view map[string]any) HomeLayerSelectionProps {
	return panels.HomeLayerSelectionPropsFromMap(view)
}

// Deprecated: use panels.RenderHomeLayerSelection.
func RenderHomeLayerSelection(props HomeLayerSelectionProps) gosx.Node {
	return panels.RenderHomeLayerSelection(props)
}

// --- Inspector control widget (panels/inspector.go) ---

// Deprecated: use panels.RenderInspectorControl.
func RenderInspectorControl(control map[string]any, formID, inputName string) gosx.Node {
	return panels.RenderInspectorControl(control, formID, inputName)
}

// --- Inspector chrome panel (panels/inspector_chrome_panel.go) ---

// Deprecated: use panels.InspectorChromePanelOptions.
type InspectorChromePanelOptions = panels.InspectorChromePanelOptions

// Deprecated: use panels.RenderInspectorChromePanel.
func RenderInspectorChromePanel(view map[string]any, options InspectorChromePanelOptions) gosx.Node {
	return panels.RenderInspectorChromePanel(view, options)
}

// --- Inspector field list (panels/inspector_fields.go) ---

// Deprecated: use panels.InspectorFieldListOptions.
type InspectorFieldListOptions = panels.InspectorFieldListOptions

// Deprecated: use panels.InspectorFieldRowOptions.
type InspectorFieldRowOptions = panels.InspectorFieldRowOptions

// Deprecated: use panels.InspectorFieldRowVariantHome.
const InspectorFieldRowVariantHome = panels.InspectorFieldRowVariantHome

// Deprecated: use panels.RenderInspectorFieldList.
func RenderInspectorFieldList(fields []map[string]any, options InspectorFieldListOptions) gosx.Node {
	return panels.RenderInspectorFieldList(fields, options)
}

// Deprecated: use panels.RenderInspectorFieldRow.
func RenderInspectorFieldRow(field map[string]any, options InspectorFieldRowOptions) gosx.Node {
	return panels.RenderInspectorFieldRow(field, options)
}

// --- Look panel (panels/look_panel.go) ---

// Deprecated: use panels.LookPanelOptions.
type LookPanelOptions = panels.LookPanelOptions

// Deprecated: use panels.RenderLookPanel.
func RenderLookPanel(view map[string]any, options LookPanelOptions) gosx.Node {
	return panels.RenderLookPanel(view, options)
}

// --- Navigation panel (panels/navigation_panel.go) ---

// Deprecated: use panels.NavigationPanelOptions.
type NavigationPanelOptions = panels.NavigationPanelOptions

// Deprecated: use panels.RenderNavigationPanel.
func RenderNavigationPanel(view map[string]any, options NavigationPanelOptions) gosx.Node {
	return panels.RenderNavigationPanel(view, options)
}

// --- Preview/share panel (panels/preview_share_panel.go) ---

// Deprecated: use panels.PreviewSharePanelOptions.
type PreviewSharePanelOptions = panels.PreviewSharePanelOptions

// Deprecated: use panels.RenderPreviewSharePanel.
func RenderPreviewSharePanel(view map[string]any, options PreviewSharePanelOptions) gosx.Node {
	return panels.RenderPreviewSharePanel(view, options)
}

// --- Publish panel (panels/publish_panel.go) ---

// Deprecated: use panels.PublishPanelOptions.
type PublishPanelOptions = panels.PublishPanelOptions

// Deprecated: use panels.RenderPublishPanel.
func RenderPublishPanel(view map[string]any, options PublishPanelOptions) gosx.Node {
	return panels.RenderPublishPanel(view, options)
}

// --- Revision history panel (panels/revision_history_panel.go) ---

// Deprecated: use panels.RevisionHistoryPanelOptions.
type RevisionHistoryPanelOptions = panels.RevisionHistoryPanelOptions

// Deprecated: use panels.RenderRevisionHistoryPanel.
func RenderRevisionHistoryPanel(view map[string]any, options RevisionHistoryPanelOptions) gosx.Node {
	return panels.RenderRevisionHistoryPanel(view, options)
}

// --- Style panel (panels/style_panel.go) ---

// Deprecated: use panels.RenderStylePanel.
func RenderStylePanel(view map[string]any, formID, action, csrfToken string) gosx.Node {
	return panels.RenderStylePanel(view, formID, action, csrfToken)
}

// --- Unexported shims for not-yet-moved root files ---
//
// block_library_panel.go also carried an unexported attribute-map render
// helper that workbench_frame.go (shell package territory, Slice 8) calls by
// its original lowercase name. panels exports the equivalent implementation
// as BlockLibraryPanelMapAttrs (see that package's doc comment); this var
// keeps that call site compiling unchanged, and will disappear once
// workbench_frame.go moves to its own subpackage in a later slice (mirrors
// compat_canvas.go's and compat_sitemap.go's identical treatment of this
// exact seam from Slices 4 and 5).
var blockLibraryPanelMapAttrs = panels.BlockLibraryPanelMapAttrs
