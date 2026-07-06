package studio

// Code generated for the Slice 8 package restructure. DO NOT EDIT by hand.
//
// This file is the deprecated compatibility facade for the symbols that used
// to live in the studio root shell files (shell.go, options.go, store.go,
// cmsbridge.go, readiness.go, runtime_config.go, workbench_*.go,
// backend_editor_*.go) before Slice 8 extracted them into the
// m31labs.dev/gosx-studio/shell package (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md).
//
// Every exported type is a type ALIAS (identity preserved: struct literals,
// embedding, method sets, assignability unchanged for muddy-noni-commerce,
// pajaritos-forest-school, and gosx-cms/studio). Every exported free function
// is a thin forwarding wrapper. Methods on aliased types are inherited through
// the alias and need no wrappers.
//
// Deprecated: import m31labs.dev/gosx-studio/shell directly; this facade will
// be removed after the v0.6.x compatibility window (spec §9).

import (
	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/core"
	shellpkg "m31labs.dev/gosx-studio/shell"
)

// --- backend_editor_page.go ---

// Deprecated: use shell.BackendEditorPageProps.
type BackendEditorPageProps = shellpkg.BackendEditorPageProps

// Deprecated: use shell.BackendEditorMediaAsset.
type BackendEditorMediaAsset = shellpkg.BackendEditorMediaAsset

// Deprecated: use shell.BackendEditorActionStatus.
type BackendEditorActionStatus = shellpkg.BackendEditorActionStatus

// Deprecated: use shell.BackendEditorScripts.
type BackendEditorScripts = shellpkg.BackendEditorScripts

// Deprecated: use shell.RenderBackendEditorPage.
func RenderBackendEditorPage(props BackendEditorPageProps) gosx.Node {
	return shellpkg.RenderBackendEditorPage(props)
}

// Deprecated: use shell.RenderBackendEditorMediaDatalist.
func RenderBackendEditorMediaDatalist(media []BackendEditorMediaAsset) gosx.Node {
	return shellpkg.RenderBackendEditorMediaDatalist(media)
}

// Deprecated: use shell.RenderBackendEditorStatuses.
func RenderBackendEditorStatuses(props BackendEditorPageProps) gosx.Node {
	return shellpkg.RenderBackendEditorStatuses(props)
}

// Deprecated: use shell.RenderBackendEditorRuntimeScripts.
func RenderBackendEditorRuntimeScripts(scripts BackendEditorScripts) gosx.Node {
	return shellpkg.RenderBackendEditorRuntimeScripts(scripts)
}

// --- backend_editor_workbench.go ---

// Deprecated: use shell.BackendEditorWorkbenchProps.
type BackendEditorWorkbenchProps = shellpkg.BackendEditorWorkbenchProps

// Deprecated: use shell.BackendEditorWorkbenchContentProps.
type BackendEditorWorkbenchContentProps = shellpkg.BackendEditorWorkbenchContentProps

// Deprecated: use shell.BackendEditorWorkbenchPanelStackProps.
type BackendEditorWorkbenchPanelStackProps = shellpkg.BackendEditorWorkbenchPanelStackProps

// Deprecated: use shell.BackendEditorPublishPanelStackProps.
type BackendEditorPublishPanelStackProps = shellpkg.BackendEditorPublishPanelStackProps

// Deprecated: use shell.RenderBackendEditorWorkbench.
func RenderBackendEditorWorkbench(props BackendEditorWorkbenchProps) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbench(props)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchToolbar.
func RenderBackendEditorWorkbenchToolbar(view map[string]any) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchToolbar(view)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchZoomControls.
func RenderBackendEditorWorkbenchZoomControls(view map[string]any) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchZoomControls(view)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchCanvasTools.
func RenderBackendEditorWorkbenchCanvasTools(view map[string]any) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchCanvasTools(view)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchViewportControls.
func RenderBackendEditorWorkbenchViewportControls(view map[string]any) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchViewportControls(view)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchCanvasBar.
func RenderBackendEditorWorkbenchCanvasBar(view map[string]any) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchCanvasBar(view)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchCanvasStatus.
func RenderBackendEditorWorkbenchCanvasStatus(view map[string]any) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchCanvasStatus(view)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchContent.
func RenderBackendEditorWorkbenchContent(props BackendEditorWorkbenchContentProps) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchContent(props)
}

// Deprecated: use shell.RenderBackendEditorWorkbenchPanelStack.
func RenderBackendEditorWorkbenchPanelStack(props BackendEditorWorkbenchPanelStackProps) gosx.Node {
	return shellpkg.RenderBackendEditorWorkbenchPanelStack(props)
}

// Deprecated: use shell.RenderBackendEditorPublishPanelStack.
func RenderBackendEditorPublishPanelStack(props BackendEditorPublishPanelStackProps) gosx.Node {
	return shellpkg.RenderBackendEditorPublishPanelStack(props)
}

// Deprecated: use shell.RenderBackendEditorLeftRail.
func RenderBackendEditorLeftRail(nodes ...gosx.Node) gosx.Node {
	return shellpkg.RenderBackendEditorLeftRail(nodes...)
}

// Deprecated: use shell.RenderBackendEditorBoard.
func RenderBackendEditorBoard(nodes ...gosx.Node) gosx.Node {
	return shellpkg.RenderBackendEditorBoard(nodes...)
}

// Deprecated: use shell.RenderBackendEditorRightRail.
func RenderBackendEditorRightRail(nodes ...gosx.Node) gosx.Node {
	return shellpkg.RenderBackendEditorRightRail(nodes...)
}

// --- cmsbridge.go ---

// Deprecated: use shell.Store.
type Store = shellpkg.Store

// --- options.go ---

// Deprecated: use shell.Options.
type Options = shellpkg.Options

// Deprecated: use shell.Action.
type Action = shellpkg.Action

// Deprecated: use shell.Panel.
type Panel = shellpkg.Panel

// Deprecated: use shell.Section.
type Section = shellpkg.Section

// Deprecated: use shell.Metric.
type Metric = shellpkg.Metric

// Deprecated: use shell.Mode.
type Mode = shellpkg.Mode

// Deprecated: use shell.Viewport.
type Viewport = shellpkg.Viewport

// Deprecated: use shell.CanvasSurface.
type CanvasSurface = shellpkg.CanvasSurface

// Deprecated: use shell.BlockSummary.
type BlockSummary = shellpkg.BlockSummary

// Deprecated: use shell.MediaSummary.
type MediaSummary = shellpkg.MediaSummary

// Deprecated: use shell.RevisionSummary.
type RevisionSummary = shellpkg.RevisionSummary

// Deprecated: use shell.Shell.
type Shell = shellpkg.Shell

// Deprecated: use shell.View.
func View(shell Shell) map[string]any {
	return shellpkg.View(shell)
}

// Deprecated: use shell.New.
func New(options Options) Shell {
	return shellpkg.New(options)
}

// Deprecated: use shell.LinkAction.
func LinkAction(key, label, href string) Action {
	return shellpkg.LinkAction(key, label, href)
}

// Deprecated: use shell.PrimaryAction.
func PrimaryAction(key, label, href string) Action {
	return shellpkg.PrimaryAction(key, label, href)
}

// Deprecated: use shell.NewPanel.
func NewPanel(key, label, summary string, children ...gosx.Node) Panel {
	return shellpkg.NewPanel(key, label, summary, children...)
}

// Deprecated: use shell.NewSection.
func NewSection(key, label string, actions ...Action) Section {
	return shellpkg.NewSection(key, label, actions...)
}

// Deprecated: use shell.NewMetric.
func NewMetric(key, label string, value any) Metric {
	return shellpkg.NewMetric(key, label, value)
}

// Deprecated: use shell.NewMode.
func NewMode(key, label string, active bool) Mode {
	return shellpkg.NewMode(key, label, active)
}

// Deprecated: use shell.NewViewport.
func NewViewport(key, label, width string, active bool) Viewport {
	return shellpkg.NewViewport(key, label, width, active)
}

// Deprecated: use shell.Render.
func Render(shell Shell) gosx.Node {
	return shellpkg.Render(shell)
}

// Deprecated: use shell.NormalizeMetrics.
func NormalizeMetrics(metrics []Metric) []Metric {
	return shellpkg.NormalizeMetrics(metrics)
}

// Deprecated: use shell.NormalizeModes.
func NormalizeModes(modes []Mode) []Mode {
	return shellpkg.NormalizeModes(modes)
}

// Deprecated: use shell.NormalizeViewports.
func NormalizeViewports(viewports []Viewport) []Viewport {
	return shellpkg.NormalizeViewports(viewports)
}

// --- readiness.go ---

// Deprecated: use shell.ShellReadinessStatus.
type ShellReadinessStatus = shellpkg.ShellReadinessStatus

// Deprecated: use shell.ShellReadiness.
type ShellReadiness = shellpkg.ShellReadiness

// Deprecated: use shell.ShellReadinessItem.
type ShellReadinessItem = shellpkg.ShellReadinessItem

// Deprecated: use shell.ShellReadinessReady.
const ShellReadinessReady = shellpkg.ShellReadinessReady

// Deprecated: use shell.ShellReadinessWatch.
const ShellReadinessWatch = shellpkg.ShellReadinessWatch

// Deprecated: use shell.ShellReadinessNext.
const ShellReadinessNext = shellpkg.ShellReadinessNext

// Deprecated: use shell.NewShellReadiness.
func NewShellReadiness(items ...ShellReadinessItem) ShellReadiness {
	return shellpkg.NewShellReadiness(items...)
}

// Deprecated: use shell.NewShellReadinessItem.
func NewShellReadinessItem(key, label string, status ShellReadinessStatus, summary, detail string) ShellReadinessItem {
	return shellpkg.NewShellReadinessItem(key, label, status, summary, detail)
}

// Deprecated: use shell.NormalizeShellReadiness.
func NormalizeShellReadiness(readiness ShellReadiness) ShellReadiness {
	return shellpkg.NormalizeShellReadiness(readiness)
}

// Deprecated: use shell.ShellReadinessView.
func ShellReadinessView(readiness ShellReadiness) map[string]any {
	return shellpkg.ShellReadinessView(readiness)
}

// Deprecated: use shell.NormalizeShellReadinessStatus.
func NormalizeShellReadinessStatus(status ShellReadinessStatus) ShellReadinessStatus {
	return shellpkg.NormalizeShellReadinessStatus(status)
}

// Deprecated: use shell.ShellReadinessStatusLabel.
func ShellReadinessStatusLabel(status ShellReadinessStatus) string {
	return shellpkg.ShellReadinessStatusLabel(status)
}

// Deprecated: use shell.ShellReadinessActionLabel.
func ShellReadinessActionLabel(status ShellReadinessStatus) string {
	return shellpkg.ShellReadinessActionLabel(status)
}

// --- runtime_config.go ---

// Deprecated: use shell.RuntimeConfig.
type RuntimeConfig = shellpkg.RuntimeConfig

// Deprecated: use shell.DefaultRuntimeConfig.
func DefaultRuntimeConfig(assetVersion string) RuntimeConfig {
	return shellpkg.DefaultRuntimeConfig(assetVersion)
}

// --- shell.go ---

// Deprecated: use shell.ShellConfig.
type ShellConfig = shellpkg.ShellConfig

// Deprecated: use shell.HostLabels.
type HostLabels = shellpkg.HostLabels

// Deprecated: use shell.CanvasConfig.
type CanvasConfig = shellpkg.CanvasConfig

// Deprecated: use shell.ModeConfig.
type ModeConfig = shellpkg.ModeConfig

// Deprecated: use shell.ResourceConfig.
type ResourceConfig = shellpkg.ResourceConfig

// Deprecated: use shell.PanelConfig.
type PanelConfig = shellpkg.PanelConfig

// Deprecated: use shell.EngineConfig.
type EngineConfig = shellpkg.EngineConfig

// Deprecated: use shell.ActionConfig.
type ActionConfig = shellpkg.ActionConfig

// Deprecated: use shell.PermissionConfig.
type PermissionConfig = shellpkg.PermissionConfig

// Deprecated: use shell.CanvasEngineName.
const CanvasEngineName = shellpkg.CanvasEngineName

// Deprecated: use shell.SiteMapEngineName.
const SiteMapEngineName = shellpkg.SiteMapEngineName

// Deprecated: use shell.FlowDesignerName.
const FlowDesignerName = shellpkg.FlowDesignerName

// Deprecated: use shell.BlockLayoutEngineName.
const BlockLayoutEngineName = shellpkg.BlockLayoutEngineName

// Deprecated: use shell.Showcase3DEngineName.
const Showcase3DEngineName = shellpkg.Showcase3DEngineName

// Deprecated: use shell.DefaultShellConfig.
func DefaultShellConfig() ShellConfig {
	return shellpkg.DefaultShellConfig()
}

// Deprecated: use shell.EngineHostView.
func EngineHostView(engine EngineConfig, className string) map[string]any {
	return shellpkg.EngineHostView(engine, className)
}

// --- store.go ---

// Deprecated: use shell.HostShell.
func HostShell(store Store, shell ShellConfig) Shell {
	return shellpkg.HostShell(store, shell)
}

// Deprecated: use shell.FeatureFlagAttrs.
func FeatureFlagAttrs(shell ShellConfig) map[string]string {
	return shellpkg.FeatureFlagAttrs(shell)
}

// --- workbench_frame.go ---

// Deprecated: use shell.WorkbenchFrameOptions.
type WorkbenchFrameOptions = shellpkg.WorkbenchFrameOptions

// Deprecated: use shell.WorkbenchRailResizerOptions.
type WorkbenchRailResizerOptions = shellpkg.WorkbenchRailResizerOptions

// Deprecated: use shell.WorkbenchFrameSegments.
type WorkbenchFrameSegments = shellpkg.WorkbenchFrameSegments

// Deprecated: use shell.RenderWorkbenchFrame.
func RenderWorkbenchFrame(view map[string]any, options WorkbenchFrameOptions) gosx.Node {
	return shellpkg.RenderWorkbenchFrame(view, options)
}

// Deprecated: use shell.RenderWorkbenchFrameSegments.
func RenderWorkbenchFrameSegments(view map[string]any, options WorkbenchFrameOptions) WorkbenchFrameSegments {
	return shellpkg.RenderWorkbenchFrameSegments(view, options)
}

// Deprecated: use shell.RenderWorkbenchRailResizer.
func RenderWorkbenchRailResizer(view map[string]any, side string, options WorkbenchRailResizerOptions) gosx.Node {
	return shellpkg.RenderWorkbenchRailResizer(view, side, options)
}

// Deprecated: use shell.RenderWorkbenchPreviewDockShell.
func RenderWorkbenchPreviewDockShell(view map[string]any) gosx.Node {
	return shellpkg.RenderWorkbenchPreviewDockShell(view)
}

// --- workbench_shell_view.go ---

// Deprecated: use shell.WorkbenchShellSource.
type WorkbenchShellSource = shellpkg.WorkbenchShellSource

// Deprecated: use shell.WorkbenchShellViewOptions.
type WorkbenchShellViewOptions = shellpkg.WorkbenchShellViewOptions

// Deprecated: use shell.WorkbenchZoomLevel.
type WorkbenchZoomLevel = shellpkg.WorkbenchZoomLevel

// Deprecated: use shell.WorkbenchRailResizer.
type WorkbenchRailResizer = shellpkg.WorkbenchRailResizer

// Deprecated: use shell.WorkbenchShellSourceFromShell.
func WorkbenchShellSourceFromShell(shell Shell) WorkbenchShellSource {
	return shellpkg.WorkbenchShellSourceFromShell(shell)
}

// Deprecated: use shell.WorkbenchShellViewForShell.
func WorkbenchShellViewForShell(shell Shell, options WorkbenchShellViewOptions) map[string]any {
	return shellpkg.WorkbenchShellViewForShell(shell, options)
}

// Deprecated: use shell.WorkbenchShellView.
func WorkbenchShellView(source WorkbenchShellSource, options WorkbenchShellViewOptions) map[string]any {
	return shellpkg.WorkbenchShellView(source, options)
}

// Deprecated: use shell.CanvasPreviewShellView.
func CanvasPreviewShellView(shell core.CanvasPreviewShell, previewURL string) map[string]any {
	return shellpkg.CanvasPreviewShellView(shell, previewURL)
}

// Deprecated: use shell.StudioAttrView.
func StudioAttrView(pairs ...string) map[string]any {
	return shellpkg.StudioAttrView(pairs...)
}

// Deprecated: use shell.WorkbenchZoomLevelViews.
func WorkbenchZoomLevelViews(active string, levels []WorkbenchZoomLevel) []map[string]any {
	return shellpkg.WorkbenchZoomLevelViews(active, levels)
}

// Deprecated: use shell.WorkbenchRailResizerView.
func WorkbenchRailResizerView(resizer WorkbenchRailResizer, side string) map[string]any {
	return shellpkg.WorkbenchRailResizerView(resizer, side)
}

// --- workbench_toolbar.go ---

// Deprecated: use shell.WorkbenchToolbarOptions.
type WorkbenchToolbarOptions = shellpkg.WorkbenchToolbarOptions

// Deprecated: use shell.WorkbenchSaveStatusOptions.
type WorkbenchSaveStatusOptions = shellpkg.WorkbenchSaveStatusOptions

// Deprecated: use shell.WorkbenchHistoryControlsOptions.
type WorkbenchHistoryControlsOptions = shellpkg.WorkbenchHistoryControlsOptions

// Deprecated: use shell.WorkbenchModebarOptions.
type WorkbenchModebarOptions = shellpkg.WorkbenchModebarOptions

// Deprecated: use shell.WorkbenchMetricStripOptions.
type WorkbenchMetricStripOptions = shellpkg.WorkbenchMetricStripOptions

// Deprecated: use shell.WorkbenchZoomControlsOptions.
type WorkbenchZoomControlsOptions = shellpkg.WorkbenchZoomControlsOptions

// Deprecated: use shell.WorkbenchViewportControlsOptions.
type WorkbenchViewportControlsOptions = shellpkg.WorkbenchViewportControlsOptions

// Deprecated: use shell.WorkbenchCanvasToolsOptions.
type WorkbenchCanvasToolsOptions = shellpkg.WorkbenchCanvasToolsOptions

// Deprecated: use shell.WorkbenchCanvasBarOptions.
type WorkbenchCanvasBarOptions = shellpkg.WorkbenchCanvasBarOptions

// Deprecated: use shell.WorkbenchCanvasStatusOptions.
type WorkbenchCanvasStatusOptions = shellpkg.WorkbenchCanvasStatusOptions

// Deprecated: use shell.WorkbenchCommandPaletteOptions.
type WorkbenchCommandPaletteOptions = shellpkg.WorkbenchCommandPaletteOptions

// Deprecated: use shell.RenderWorkbenchToolbar.
func RenderWorkbenchToolbar(view map[string]any, options WorkbenchToolbarOptions) gosx.Node {
	return shellpkg.RenderWorkbenchToolbar(view, options)
}

// Deprecated: use shell.RenderWorkbenchModebar.
func RenderWorkbenchModebar(view map[string]any, options WorkbenchModebarOptions) gosx.Node {
	return shellpkg.RenderWorkbenchModebar(view, options)
}

// Deprecated: use shell.RenderWorkbenchMetricStrip.
func RenderWorkbenchMetricStrip(view map[string]any, options WorkbenchMetricStripOptions) gosx.Node {
	return shellpkg.RenderWorkbenchMetricStrip(view, options)
}

// Deprecated: use shell.RenderWorkbenchZoomControls.
func RenderWorkbenchZoomControls(view map[string]any, options WorkbenchZoomControlsOptions) gosx.Node {
	return shellpkg.RenderWorkbenchZoomControls(view, options)
}

// Deprecated: use shell.RenderWorkbenchViewportControls.
func RenderWorkbenchViewportControls(view map[string]any, options WorkbenchViewportControlsOptions) gosx.Node {
	return shellpkg.RenderWorkbenchViewportControls(view, options)
}

// Deprecated: use shell.RenderWorkbenchCanvasTools.
func RenderWorkbenchCanvasTools(view map[string]any, options WorkbenchCanvasToolsOptions) gosx.Node {
	return shellpkg.RenderWorkbenchCanvasTools(view, options)
}

// Deprecated: use shell.RenderWorkbenchCanvasBar.
func RenderWorkbenchCanvasBar(view map[string]any, options WorkbenchCanvasBarOptions) gosx.Node {
	return shellpkg.RenderWorkbenchCanvasBar(view, options)
}

// Deprecated: use shell.RenderWorkbenchCanvasStatus.
func RenderWorkbenchCanvasStatus(view map[string]any, options WorkbenchCanvasStatusOptions) gosx.Node {
	return shellpkg.RenderWorkbenchCanvasStatus(view, options)
}

// Deprecated: use shell.RenderWorkbenchCommandPalette.
func RenderWorkbenchCommandPalette(view map[string]any, options WorkbenchCommandPaletteOptions) gosx.Node {
	return shellpkg.RenderWorkbenchCommandPalette(view, options)
}

// Deprecated: use shell.RenderWorkbenchSaveStatus.
func RenderWorkbenchSaveStatus(options WorkbenchSaveStatusOptions) gosx.Node {
	return shellpkg.RenderWorkbenchSaveStatus(options)
}

// Deprecated: use shell.RenderWorkbenchHistoryControls.
func RenderWorkbenchHistoryControls(options WorkbenchHistoryControlsOptions) gosx.Node {
	return shellpkg.RenderWorkbenchHistoryControls(options)
}
