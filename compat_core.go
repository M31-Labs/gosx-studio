package studio

// This file is the deprecated compatibility facade for the symbols that used
// to live in studio.go and plugin.go before Slice 1 of the package
// restructure (see .tiller/scratch/gosx-studio-restructure-spec-v0.1.md)
// extracted them into the m31labs.dev/gosx-studio/core package.
//
// Every exported type below is a type ALIAS (not a new type), so struct
// literals, embedding, method sets, and assignability are all unchanged for
// existing callers (muddy-noni-commerce, pajaritos-forest-school,
// gosx-cms/studio). Every exported free function is a thin forwarding
// wrapper. Methods on the aliased types are inherited automatically through
// the alias and do not need wrappers here.
//
// Deprecated: import m31labs.dev/gosx-studio/core directly; this facade will
// be removed after the v0.6.x compatibility window (see spec §9).

import "m31labs.dev/gosx-studio/core"

// --- Kind/enum types (core/kinds.go) ---

// Deprecated: use core.SurfaceKind.
type SurfaceKind = core.SurfaceKind

// Deprecated: use core.ComponentSource.
type ComponentSource = core.ComponentSource

// Deprecated: use core.PageGroup.
type PageGroup = core.PageGroup

// Deprecated: use core.EngineKind.
type EngineKind = core.EngineKind

// Deprecated: use core.EngineCapability.
type EngineCapability = core.EngineCapability

// Deprecated: use core.ResourceKind.
type ResourceKind = core.ResourceKind

// Deprecated: use core.ResourceCapability.
type ResourceCapability = core.ResourceCapability

// Deprecated: use core.ControlKind.
type ControlKind = core.ControlKind

// Deprecated: use core.CompositionIntentKind.
type CompositionIntentKind = core.CompositionIntentKind

// Deprecated: use core.ReadinessStatus.
type ReadinessStatus = core.ReadinessStatus

// Deprecated: use core.WorkspaceNodeKind.
type WorkspaceNodeKind = core.WorkspaceNodeKind

// Deprecated: use core.WorkspaceLinkKind.
type WorkspaceLinkKind = core.WorkspaceLinkKind

// Deprecated: use core.CanvasActionKind.
type CanvasActionKind = core.CanvasActionKind

// Deprecated: use core.PackageCMS.
const PackageCMS = core.PackageCMS

// Deprecated: use core.PackageAdmin.
const PackageAdmin = core.PackageAdmin

// Deprecated: use core.PackageStudio.
const PackageStudio = core.PackageStudio

const (
	// Deprecated: use core.SurfaceCanvas.
	SurfaceCanvas = core.SurfaceCanvas
	// Deprecated: use core.SurfaceSiteMap.
	SurfaceSiteMap = core.SurfaceSiteMap
	// Deprecated: use core.SurfaceInspector.
	SurfaceInspector = core.SurfaceInspector
	// Deprecated: use core.SurfaceFlow.
	SurfaceFlow = core.SurfaceFlow
	// Deprecated: use core.SurfacePublish.
	SurfacePublish = core.SurfacePublish
	// Deprecated: use core.SurfaceShowcase3D.
	SurfaceShowcase3D = core.SurfaceShowcase3D
)

const (
	// Deprecated: use core.ComponentSourceHost.
	ComponentSourceHost = core.ComponentSourceHost
	// Deprecated: use core.ComponentSourceCMS.
	ComponentSourceCMS = core.ComponentSourceCMS
	// Deprecated: use core.ComponentSourcePlugin.
	ComponentSourcePlugin = core.ComponentSourcePlugin
	// Deprecated: use core.ComponentSourceStudio.
	ComponentSourceStudio = core.ComponentSourceStudio
)

const (
	// Deprecated: use core.PageGroupSite.
	PageGroupSite = core.PageGroupSite
	// Deprecated: use core.PageGroupCommerce.
	PageGroupCommerce = core.PageGroupCommerce
	// Deprecated: use core.PageGroupContent.
	PageGroupContent = core.PageGroupContent
	// Deprecated: use core.PageGroupFlows.
	PageGroupFlows = core.PageGroupFlows
	// Deprecated: use core.PageGroupUtility.
	PageGroupUtility = core.PageGroupUtility
)

const (
	// Deprecated: use core.EngineCanvas.
	EngineCanvas = core.EngineCanvas
	// Deprecated: use core.EngineSiteMap.
	EngineSiteMap = core.EngineSiteMap
	// Deprecated: use core.EngineBlockLayout.
	EngineBlockLayout = core.EngineBlockLayout
	// Deprecated: use core.EngineFlow.
	EngineFlow = core.EngineFlow
	// Deprecated: use core.EngineScene3D.
	EngineScene3D = core.EngineScene3D
)

const (
	// Deprecated: use core.CapabilityDragDrop.
	CapabilityDragDrop = core.CapabilityDragDrop
	// Deprecated: use core.CapabilityPanZoom.
	CapabilityPanZoom = core.CapabilityPanZoom
	// Deprecated: use core.CapabilitySelection.
	CapabilitySelection = core.CapabilitySelection
	// Deprecated: use core.CapabilityInlineEdit.
	CapabilityInlineEdit = core.CapabilityInlineEdit
	// Deprecated: use core.CapabilityPreview.
	CapabilityPreview = core.CapabilityPreview
	// Deprecated: use core.CapabilityPopout.
	CapabilityPopout = core.CapabilityPopout
	// Deprecated: use core.CapabilityPersistence.
	CapabilityPersistence = core.CapabilityPersistence
)

const (
	// Deprecated: use core.ResourceMedia.
	ResourceMedia = core.ResourceMedia
	// Deprecated: use core.ResourcePages.
	ResourcePages = core.ResourcePages
	// Deprecated: use core.ResourceProducts.
	ResourceProducts = core.ResourceProducts
	// Deprecated: use core.ResourceOrders.
	ResourceOrders = core.ResourceOrders
	// Deprecated: use core.ResourceContacts.
	ResourceContacts = core.ResourceContacts
	// Deprecated: use core.ResourceSettings.
	ResourceSettings = core.ResourceSettings
	// Deprecated: use core.ResourceRevisions.
	ResourceRevisions = core.ResourceRevisions
	// Deprecated: use core.ResourceLifecycle.
	ResourceLifecycle = core.ResourceLifecycle
	// Deprecated: use core.ResourceFlows.
	ResourceFlows = core.ResourceFlows
)

const (
	// Deprecated: use core.ResourceList.
	ResourceList = core.ResourceList
	// Deprecated: use core.ResourceRead.
	ResourceRead = core.ResourceRead
	// Deprecated: use core.ResourceWrite.
	ResourceWrite = core.ResourceWrite
	// Deprecated: use core.ResourceSearch.
	ResourceSearch = core.ResourceSearch
	// Deprecated: use core.ResourcePreview.
	ResourcePreview = core.ResourcePreview
	// Deprecated: use core.ResourcePublish.
	ResourcePublish = core.ResourcePublish
	// Deprecated: use core.ResourceRestore.
	ResourceRestore = core.ResourceRestore
	// Deprecated: use core.ResourceExecute.
	ResourceExecute = core.ResourceExecute
)

const (
	// Deprecated: use core.ControlText.
	ControlText = core.ControlText
	// Deprecated: use core.ControlRichText.
	ControlRichText = core.ControlRichText
	// Deprecated: use core.ControlMedia.
	ControlMedia = core.ControlMedia
	// Deprecated: use core.ControlChoice.
	ControlChoice = core.ControlChoice
	// Deprecated: use core.ControlToggle.
	ControlToggle = core.ControlToggle
	// Deprecated: use core.ControlNumber.
	ControlNumber = core.ControlNumber
	// Deprecated: use core.ControlLink.
	ControlLink = core.ControlLink
	// Deprecated: use core.ControlColor.
	ControlColor = core.ControlColor
	// Deprecated: use core.ControlSource.
	ControlSource = core.ControlSource
	// Deprecated: use core.ControlFlow.
	ControlFlow = core.ControlFlow
	// Deprecated: use core.ControlScene3D.
	ControlScene3D = core.ControlScene3D
)

const (
	// Deprecated: use core.CompositionIntentCreatePage.
	CompositionIntentCreatePage = core.CompositionIntentCreatePage
	// Deprecated: use core.CompositionIntentAddComponent.
	CompositionIntentAddComponent = core.CompositionIntentAddComponent
)

const (
	// Deprecated: use core.ReadinessReady.
	ReadinessReady = core.ReadinessReady
	// Deprecated: use core.ReadinessWatch.
	ReadinessWatch = core.ReadinessWatch
	// Deprecated: use core.ReadinessBlocked.
	ReadinessBlocked = core.ReadinessBlocked
)

const (
	// Deprecated: use core.WorkspaceNodePage.
	WorkspaceNodePage = core.WorkspaceNodePage
	// Deprecated: use core.WorkspaceNodeComponent.
	WorkspaceNodeComponent = core.WorkspaceNodeComponent
	// Deprecated: use core.WorkspaceNodeResource.
	WorkspaceNodeResource = core.WorkspaceNodeResource
)

const (
	// Deprecated: use core.WorkspaceLinkContains.
	WorkspaceLinkContains = core.WorkspaceLinkContains
	// Deprecated: use core.WorkspaceLinkBinds.
	WorkspaceLinkBinds = core.WorkspaceLinkBinds
)

const (
	// Deprecated: use core.CanvasActionReveal.
	CanvasActionReveal = core.CanvasActionReveal
	// Deprecated: use core.CanvasActionMoveUp.
	CanvasActionMoveUp = core.CanvasActionMoveUp
	// Deprecated: use core.CanvasActionMoveDown.
	CanvasActionMoveDown = core.CanvasActionMoveDown
	// Deprecated: use core.CanvasActionInlineText.
	CanvasActionInlineText = core.CanvasActionInlineText
	// Deprecated: use core.CanvasActionContent.
	CanvasActionContent = core.CanvasActionContent
	// Deprecated: use core.CanvasActionStyle.
	CanvasActionStyle = core.CanvasActionStyle
	// Deprecated: use core.CanvasActionToggleVisibility.
	CanvasActionToggleVisibility = core.CanvasActionToggleVisibility
)

// --- Site-map types (core/sitemap.go) ---

// Deprecated: use core.SiteMap.
type SiteMap = core.SiteMap

// Deprecated: use core.Page.
type Page = core.Page

// Deprecated: use core.PageGroupCount.
type PageGroupCount = core.PageGroupCount

// --- Canvas types (core/canvas.go) ---

// Deprecated: use core.CanvasWorkspace.
type CanvasWorkspace = core.CanvasWorkspace

// Deprecated: use core.CanvasViewport.
type CanvasViewport = core.CanvasViewport

// Deprecated: use core.CanvasZoomLevel.
type CanvasZoomLevel = core.CanvasZoomLevel

// Deprecated: use core.CanvasBlock.
type CanvasBlock = core.CanvasBlock

// Deprecated: use core.CanvasAction.
type CanvasAction = core.CanvasAction

// Deprecated: use core.CanvasPreviewShell.
type CanvasPreviewShell = core.CanvasPreviewShell

// --- Composition types (core/composition.go) ---

// Deprecated: use core.Component.
type Component = core.Component

// Deprecated: use core.Control.
type Control = core.Control

// Deprecated: use core.ControlOption.
type ControlOption = core.ControlOption

// Deprecated: use core.CompositionLibrary.
type CompositionLibrary = core.CompositionLibrary

// Deprecated: use core.PageBlueprint.
type PageBlueprint = core.PageBlueprint

// Deprecated: use core.ComponentTemplate.
type ComponentTemplate = core.ComponentTemplate

// Deprecated: use core.ComponentDefinition.
type ComponentDefinition = core.ComponentDefinition

// Deprecated: use core.ComponentOverride.
type ComponentOverride = core.ComponentOverride

// Deprecated: use core.PaletteEntry.
type PaletteEntry = core.PaletteEntry

// Deprecated: use core.CompositionIntent.
type CompositionIntent = core.CompositionIntent

// Deprecated: use core.CompositionStep.
type CompositionStep = core.CompositionStep

// Deprecated: use core.CompositionWorkspace.
type CompositionWorkspace = core.CompositionWorkspace

// Deprecated: use core.WorkspaceCanvas.
type WorkspaceCanvas = core.WorkspaceCanvas

// Deprecated: use core.WorkspaceLayer.
type WorkspaceLayer = core.WorkspaceLayer

// Deprecated: use core.WorkspaceNode.
type WorkspaceNode = core.WorkspaceNode

// Deprecated: use core.WorkspaceLink.
type WorkspaceLink = core.WorkspaceLink

// Deprecated: use core.WorkspaceNodePoint.
type WorkspaceNodePoint = core.WorkspaceNodePoint

// Deprecated: use core.WorkspaceLinkPath.
type WorkspaceLinkPath = core.WorkspaceLinkPath

// --- Flow types (core/flows.go) ---

// Deprecated: use core.Flow.
type Flow = core.Flow

// Deprecated: use core.FlowStep.
type FlowStep = core.FlowStep

// Deprecated: use core.FlowAction.
type FlowAction = core.FlowAction

// Deprecated: use core.FlowField.
type FlowField = core.FlowField

// Deprecated: use core.FlowReadinessCheck.
type FlowReadinessCheck = core.FlowReadinessCheck

// Deprecated: use core.FlowNode.
type FlowNode = core.FlowNode

// --- Engine/runtime-contract types (core/engines.go) ---

// Deprecated: use core.Engine.
type Engine = core.Engine

// Deprecated: use core.RuntimeContract.
type RuntimeContract = core.RuntimeContract

// Deprecated: use core.RuntimeMethod.
type RuntimeMethod = core.RuntimeMethod

// Deprecated: use core.RuntimePayloadField.
type RuntimePayloadField = core.RuntimePayloadField

// Deprecated: use core.Boundary.
type Boundary = core.Boundary

// Deprecated: use core.HostConfig.
type HostConfig = core.HostConfig

// Deprecated: use core.Feature.
type Feature = core.Feature

// --- Resource types (core/resources.go) ---

// Deprecated: use core.ResourceAdapter.
type ResourceAdapter = core.ResourceAdapter

// Deprecated: use core.ResourceBinding.
type ResourceBinding = core.ResourceBinding

// --- Plugin types (core/plugin.go) ---
//
// Plugin and PluginIsland moved to core alongside CompositionLibrary because
// RegisterPlugin (a method on CompositionLibrary) cannot be defined outside
// core once CompositionLibrary becomes a facade alias. PluginRegistry (the
// http-serving island registry, plugin_v2.go) stays at the studio root
// pending the hostruntime slice.

// Deprecated: use core.Plugin.
type Plugin = core.Plugin

// Deprecated: use core.PluginIsland.
type PluginIsland = core.PluginIsland

// --- Default constructors (core/defaults.go) ---

// Deprecated: use core.DefaultBoundary.
func DefaultBoundary() Boundary { return core.DefaultBoundary() }

// Deprecated: use core.DefaultFeatures.
func DefaultFeatures() []Feature { return core.DefaultFeatures() }

// Deprecated: use core.DefaultResourceAdapters.
func DefaultResourceAdapters() []ResourceAdapter { return core.DefaultResourceAdapters() }

// Deprecated: use core.DefaultEngines.
func DefaultEngines() []Engine { return core.DefaultEngines() }

// Deprecated: use core.DefaultRuntimeContracts.
func DefaultRuntimeContracts() []RuntimeContract { return core.DefaultRuntimeContracts() }

// Deprecated: use core.DefaultCanvasActions.
func DefaultCanvasActions() []CanvasAction { return core.DefaultCanvasActions() }

// Deprecated: use core.DefaultCanvasPreviewShell.
func DefaultCanvasPreviewShell() CanvasPreviewShell { return core.DefaultCanvasPreviewShell() }

// --- Lookup/adapter helper functions (core/engines.go, core/resources.go) ---

// Deprecated: use core.ResourceAdapterByKind.
func ResourceAdapterByKind(adapters []ResourceAdapter, kind ResourceKind) (ResourceAdapter, bool) {
	return core.ResourceAdapterByKind(adapters, kind)
}

// Deprecated: use core.RuntimeContractByGlobal.
func RuntimeContractByGlobal(contracts []RuntimeContract, global string) (RuntimeContract, bool) {
	return core.RuntimeContractByGlobal(contracts, global)
}

// --- Label functions (core/labels.go) ---

// Deprecated: use core.PageGroupLabel.
func PageGroupLabel(group PageGroup) string { return core.PageGroupLabel(group) }

// Deprecated: use core.ComponentSourceLabel.
func ComponentSourceLabel(source ComponentSource) string { return core.ComponentSourceLabel(source) }

// Deprecated: use core.ControlKindLabel.
func ControlKindLabel(kind ControlKind) string { return core.ControlKindLabel(kind) }

// Deprecated: use core.WorkspaceNodeKindLabel.
func WorkspaceNodeKindLabel(kind WorkspaceNodeKind) string {
	return core.WorkspaceNodeKindLabel(kind)
}

// Deprecated: use core.WorkspaceLinkKindLabel.
func WorkspaceLinkKindLabel(kind WorkspaceLinkKind) string {
	return core.WorkspaceLinkKindLabel(kind)
}

// Deprecated: use core.ReadinessStatusLabel.
func ReadinessStatusLabel(status ReadinessStatus) string { return core.ReadinessStatusLabel(status) }

// Deprecated: use core.ResourceKindLabel.
func ResourceKindLabel(kind ResourceKind) string { return core.ResourceKindLabel(kind) }

// Deprecated: use core.CanvasActionKindLabel.
func CanvasActionKindLabel(kind CanvasActionKind) string { return core.CanvasActionKindLabel(kind) }

// --- Text helpers (core/text.go) ---
//
// FirstNonEmpty/NormalizeKey/BoolAttr/FmtAny used to be real implementations
// duplicated across readiness.go and options.go (plus an unexported twin of
// FirstNonEmpty in studio.go). They now have a single canonical home in
// core/text.go; every layer, including gosx-cms/studio, uses these wrappers.

// Deprecated: use core.FirstNonEmpty.
func FirstNonEmpty(values ...string) string { return core.FirstNonEmpty(values...) }

// Deprecated: use core.NormalizeKey.
func NormalizeKey(value string) string { return core.NormalizeKey(value) }

// Deprecated: use core.BoolAttr.
func BoolAttr(value bool) string { return core.BoolAttr(value) }

// Deprecated: use core.FmtAny.
func FmtAny(value any) string { return core.FmtAny(value) }

// --- Unexported shims for not-yet-moved root files ---
//
// studio.go also carried unexported helpers that root-only files (not moved
// until later slices) call by their original lowercase names. These vars
// keep those call sites compiling unchanged against the new core
// implementations; they are internal wiring, not part of the public facade,
// and will disappear once the referencing files move to their own
// subpackages in later slices.
var (
	firstNonEmpty                  = core.FirstNonEmpty
	countLabel                     = core.CountLabel
	workspaceToken                 = core.WorkspaceToken
	workspaceCoord                 = core.WorkspaceCoord
	normalizeControlKind           = core.NormalizeControlKind
	normalizeCompositionIntentKind = core.NormalizeCompositionIntentKind
	normalizeCompositionSteps      = core.NormalizeCompositionSteps
)
