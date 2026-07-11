package studio

// This file is the deprecated compatibility facade for the symbols that used
// to live in mutations.go, authoring.go, style_mutations.go,
// component_styles.go, and the style-mutation half of handlers.go before
// Slice 2 of the package restructure (see
// .tiller/scratch/gosx-studio-restructure-spec-v0.1.md) extracted them into
// the m31labs.dev/gosx-studio/authoring package.
//
// Every exported type below is a type ALIAS (not a new type), so struct
// literals, embedding, method sets, and assignability are all unchanged for
// existing callers (muddy-noni-commerce, pajaritos-forest-school,
// gosx-cms/studio). Every exported free function is a thin forwarding
// wrapper. Methods on the aliased types are inherited automatically through
// the alias and do not need wrappers here.
//
// Deprecated: import m31labs.dev/gosx-studio/authoring directly; this facade
// will be removed after the v0.6.x compatibility window (see spec §9).

import (
	"context"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/media"
	"m31labs.dev/gosx/action"
)

type AssetOperationKind = authoring.AssetOperationKind
type AssetBindingTarget = authoring.AssetBindingTarget
type AssetBinding = authoring.AssetBinding
type AssetState = authoring.AssetState
type AssetOperation = authoring.AssetOperation
type AssetOperationRecord = authoring.AssetOperationRecord
type AssetApplyOptions = authoring.AssetApplyOptions
type AssetOperationResult = authoring.AssetOperationResult
type AssetStateStore = authoring.AssetStateStore
type FileAssetStateStore = authoring.FileAssetStateStore
type AssetService = authoring.AssetService

const (
	AssetImport         = authoring.AssetImport
	AssetUpdateMetadata = authoring.AssetUpdateMetadata
	AssetReplace        = authoring.AssetReplace
	AssetDelete         = authoring.AssetDelete
	AssetBind           = authoring.AssetBind
	AssetUnbind         = authoring.AssetUnbind
	AssetUndo           = authoring.AssetUndo
	AssetRedo           = authoring.AssetRedo
)

var (
	ErrAssetInvalid      = authoring.ErrAssetInvalid
	ErrAssetUnauthorized = authoring.ErrAssetUnauthorized
	ErrAssetConflict     = authoring.ErrAssetConflict
	ErrAssetIdempotency  = authoring.ErrAssetIdempotency
	ErrAssetNotFound     = authoring.ErrAssetNotFound
	ErrAssetReferenced   = authoring.ErrAssetReferenced
)

func ApplyAssetOperation(state AssetState, operation AssetOperation, options AssetApplyOptions) (AssetOperationResult, error) {
	return authoring.ApplyAssetOperation(state, operation, options)
}

func CommitAssetOperation(ctx context.Context, store AssetStateStore, operation AssetOperation, options AssetApplyOptions) (AssetOperationResult, error) {
	return authoring.CommitAssetOperation(ctx, store, operation, options)
}

func ResolveAssetBinding(state AssetState, reusable ReusableState, instanceID, definitionID, componentKey, property string) (AssetBinding, media.Asset, bool) {
	return authoring.ResolveAssetBinding(state, reusable, instanceID, definitionID, componentKey, property)
}

type ReusableOperationKind = authoring.ReusableOperationKind
type ReusableState = authoring.ReusableState
type ReusableOperation = authoring.ReusableOperation
type ReusableTargetSnapshot = authoring.ReusableTargetSnapshot
type ReusableOperationRecord = authoring.ReusableOperationRecord
type ReusableApplyOptions = authoring.ReusableApplyOptions
type ReusableOperationResult = authoring.ReusableOperationResult
type ReusableStateStore = authoring.ReusableStateStore

const (
	ReusableCreateDefinition    = authoring.ReusableCreateDefinition
	ReusableCreateInstance      = authoring.ReusableCreateInstance
	ReusableUpdateDefinition    = authoring.ReusableUpdateDefinition
	ReusableSetInstanceOverride = authoring.ReusableSetInstanceOverride
	ReusableClearOverride       = authoring.ReusableClearOverride
	ReusableDetachInstance      = authoring.ReusableDetachInstance
	ReusableRestoreInstance     = authoring.ReusableRestoreInstance
	ReusableRevertOperation     = authoring.ReusableRevertOperation
)

const (
	ReusableFieldOperation     = authoring.ReusableFieldOperation
	ReusableFieldOperationID   = authoring.ReusableFieldOperationID
	ReusableFieldDefinitionID  = authoring.ReusableFieldDefinitionID
	ReusableFieldInstanceID    = authoring.ReusableFieldInstanceID
	ReusableFieldPageKey       = authoring.ReusableFieldPageKey
	ReusableFieldRegion        = authoring.ReusableFieldRegion
	ReusableFieldPosition      = authoring.ReusableFieldPosition
	ReusableFieldOverrideKey   = authoring.ReusableFieldOverrideKey
	ReusableFieldOverrideValue = authoring.ReusableFieldOverrideValue
	ReusableFieldExpectedHead  = authoring.ReusableFieldExpectedHead
)

var (
	ErrReusableConflict     = authoring.ErrReusableConflict
	ErrReusableUnauthorized = authoring.ErrReusableUnauthorized
	ErrReusableIdempotency  = authoring.ErrReusableIdempotency
	ErrReusableNotFound     = authoring.ErrReusableNotFound
)

func ApplyReusableOperation(state ReusableState, operation ReusableOperation, options ReusableApplyOptions) (ReusableOperationResult, error) {
	return authoring.ApplyReusableOperation(state, operation, options)
}

func ApplyReusableInverse(state ReusableState, historyID, operationID, expectedHead string, options ReusableApplyOptions) (ReusableOperationResult, error) {
	return authoring.ApplyReusableInverse(state, historyID, operationID, expectedHead, options)
}

func ReusableOperationFromForm(form map[string]string) (ReusableOperation, error) {
	return authoring.ReusableOperationFromForm(form)
}

func CommitReusableOperation(ctx context.Context, store ReusableStateStore, operation ReusableOperation, options ReusableApplyOptions) (ReusableOperationResult, error) {
	return authoring.CommitReusableOperation(ctx, store, operation, options)
}

func CommitReusableInverse(ctx context.Context, store ReusableStateStore, historyID, operationID, expectedHead string, options ReusableApplyOptions) (ReusableOperationResult, error) {
	return authoring.CommitReusableInverse(ctx, store, historyID, operationID, expectedHead, options)
}

// --- Mutation-boundary types (authoring/mutations.go) ---

// Deprecated: use authoring.AuthoringOperationKind.
type AuthoringOperationKind = authoring.AuthoringOperationKind

// Deprecated: use authoring.AuthoringAdapter.
type AuthoringAdapter = authoring.AuthoringAdapter

// Deprecated: use authoring.AuthoringMutation.
type AuthoringMutation = authoring.AuthoringMutation

// Deprecated: use authoring.AuthoringValidation.
type AuthoringValidation = authoring.AuthoringValidation

// Deprecated: use authoring.AuthoringMutationResult.
type AuthoringMutationResult = authoring.AuthoringMutationResult

// Deprecated: use authoring.AuthoringChange.
type AuthoringChange = authoring.AuthoringChange

// Deprecated: use authoring.AuthoringRefreshFragment.
type AuthoringRefreshFragment = authoring.AuthoringRefreshFragment

const (
	// Deprecated: use authoring.AuthoringOperationApplyIntent.
	AuthoringOperationApplyIntent = authoring.AuthoringOperationApplyIntent
	// Deprecated: use authoring.AuthoringOperationSaveControl.
	AuthoringOperationSaveControl = authoring.AuthoringOperationSaveControl
	// Deprecated: use authoring.AuthoringOperationReorderComponent.
	AuthoringOperationReorderComponent = authoring.AuthoringOperationReorderComponent
	// Deprecated: use authoring.AuthoringOperationToggleVisibility.
	AuthoringOperationToggleVisibility = authoring.AuthoringOperationToggleVisibility
	// Deprecated: use authoring.AuthoringOperationDuplicateComponent.
	AuthoringOperationDuplicateComponent = authoring.AuthoringOperationDuplicateComponent
	// Deprecated: use authoring.AuthoringOperationDeleteComponent.
	AuthoringOperationDeleteComponent = authoring.AuthoringOperationDeleteComponent
	// Deprecated: use authoring.AuthoringOperationUpdatePage.
	AuthoringOperationUpdatePage = authoring.AuthoringOperationUpdatePage
	// Deprecated: use authoring.AuthoringOperationSaveAppearance.
	AuthoringOperationSaveAppearance = authoring.AuthoringOperationSaveAppearance
	// Deprecated: use authoring.AuthoringOperationSetStyle.
	AuthoringOperationSetStyle   = authoring.AuthoringOperationSetStyle
	AuthoringOperationSetField   = authoring.AuthoringOperationSetField
	AuthoringOperationResetStyle = authoring.AuthoringOperationResetStyle
	AuthoringOperationUndo       = authoring.AuthoringOperationUndo
	AuthoringOperationRedo       = authoring.AuthoringOperationRedo
)

// Durable operation protocol aliases for hosts that still import the Studio
// root facade.
type OperationKind = authoring.OperationKind
type OperationTarget = authoring.OperationTarget
type OperationValue = authoring.OperationValue
type OperationRequest = authoring.OperationRequest
type OperationRecord = authoring.OperationRecord

const OperationSchemaVersion = authoring.OperationSchemaVersion
const (
	OperationSetField   = authoring.OperationSetField
	OperationSetStyle   = authoring.OperationSetStyle
	OperationResetStyle = authoring.OperationResetStyle
	OperationUndo       = authoring.OperationUndo
	OperationRedo       = authoring.OperationRedo
)

const (
	// Deprecated: use authoring.AuthoringFieldOperation.
	AuthoringFieldOperation = authoring.AuthoringFieldOperation
	// Deprecated: use authoring.AuthoringFieldIntentKey.
	AuthoringFieldIntentKey = authoring.AuthoringFieldIntentKey
	// Deprecated: use authoring.AuthoringFieldIntentKind.
	AuthoringFieldIntentKind = authoring.AuthoringFieldIntentKind
	// Deprecated: use authoring.AuthoringFieldPageKey.
	AuthoringFieldPageKey = authoring.AuthoringFieldPageKey
	// Deprecated: use authoring.AuthoringFieldPageLabel.
	AuthoringFieldPageLabel = authoring.AuthoringFieldPageLabel
	// Deprecated: use authoring.AuthoringFieldPageRoute.
	AuthoringFieldPageRoute = authoring.AuthoringFieldPageRoute
	// Deprecated: use authoring.AuthoringFieldPageBlueprintKey.
	AuthoringFieldPageBlueprintKey = authoring.AuthoringFieldPageBlueprintKey
	// Deprecated: use authoring.AuthoringFieldComponentKey.
	AuthoringFieldComponentKey = authoring.AuthoringFieldComponentKey
	// Deprecated: use authoring.AuthoringFieldComponentLabel.
	AuthoringFieldComponentLabel = authoring.AuthoringFieldComponentLabel
	// Deprecated: use authoring.AuthoringFieldComponentTemplateKey.
	AuthoringFieldComponentTemplateKey = authoring.AuthoringFieldComponentTemplateKey
	// Deprecated: use authoring.AuthoringFieldControlKey.
	AuthoringFieldControlKey = authoring.AuthoringFieldControlKey
	// Deprecated: use authoring.AuthoringFieldControlKind.
	AuthoringFieldControlKind = authoring.AuthoringFieldControlKind
	// Deprecated: use authoring.AuthoringFieldBinding.
	AuthoringFieldBinding = authoring.AuthoringFieldBinding
	// Deprecated: use authoring.AuthoringFieldTargetRegion.
	AuthoringFieldTargetRegion = authoring.AuthoringFieldTargetRegion
	// Deprecated: use authoring.AuthoringFieldValue.
	AuthoringFieldValue = authoring.AuthoringFieldValue
	// Deprecated: use authoring.AuthoringFieldStyleProperty.
	AuthoringFieldStyleProperty = authoring.AuthoringFieldStyleProperty
	// Deprecated: use authoring.AuthoringFieldStyleValue.
	AuthoringFieldStyleValue = authoring.AuthoringFieldStyleValue
	// Deprecated: use authoring.AuthoringFieldBreakpoint.
	AuthoringFieldBreakpoint = authoring.AuthoringFieldBreakpoint
	// Deprecated: use authoring.AuthoringFieldState.
	AuthoringFieldState = authoring.AuthoringFieldState
	// Deprecated: use authoring.AuthoringFieldPosition.
	AuthoringFieldPosition = authoring.AuthoringFieldPosition
	// Deprecated: use authoring.AuthoringFieldVisible.
	AuthoringFieldVisible            = authoring.AuthoringFieldVisible
	AuthoringFieldOperationID        = authoring.AuthoringFieldOperationID
	AuthoringFieldExpectedRevision   = authoring.AuthoringFieldExpectedRevision
	AuthoringFieldExpectedTargetHead = authoring.AuthoringFieldExpectedTargetHead
	AuthoringFieldHistoryOperationID = authoring.AuthoringFieldHistoryOperationID
)

// Deprecated: use authoring.AuthoringActionHandler.
func AuthoringActionHandler(adapter AuthoringAdapter) action.Handler {
	return authoring.AuthoringActionHandler(adapter)
}

// Deprecated: use authoring.AuthoringMutationFromIntent.
func AuthoringMutationFromIntent(intent CompositionIntent) AuthoringMutation {
	return authoring.AuthoringMutationFromIntent(intent)
}

// Deprecated: use authoring.AuthoringMutationForControl.
func AuthoringMutationForControl(page Page, component Component, control Control) AuthoringMutation {
	return authoring.AuthoringMutationForControl(page, component, control)
}

// Deprecated: use authoring.AuthoringMutationForPage.
func AuthoringMutationForPage(page Page) AuthoringMutation {
	return authoring.AuthoringMutationForPage(page)
}

// Deprecated: use authoring.AuthoringMutationForComponentReorder.
func AuthoringMutationForComponentReorder(page Page, component Component, position int) AuthoringMutation {
	return authoring.AuthoringMutationForComponentReorder(page, component, position)
}

// Deprecated: use authoring.AuthoringMutationForComponentVisibility.
func AuthoringMutationForComponentVisibility(page Page, component Component, visible bool) AuthoringMutation {
	return authoring.AuthoringMutationForComponentVisibility(page, component, visible)
}

// Deprecated: use authoring.AuthoringMutationForComponentDelete.
func AuthoringMutationForComponentDelete(page Page, component Component) AuthoringMutation {
	return authoring.AuthoringMutationForComponentDelete(page, component)
}

// Deprecated: use authoring.AuthoringMutationForComponentDuplicate.
func AuthoringMutationForComponentDuplicate(page Page, component Component, position int) AuthoringMutation {
	return authoring.AuthoringMutationForComponentDuplicate(page, component, position)
}

// Deprecated: use authoring.AuthoringMutationFromForm.
func AuthoringMutationFromForm(form map[string]string) (AuthoringMutation, AuthoringValidation) {
	return authoring.AuthoringMutationFromForm(form)
}

// Deprecated: use authoring.AuthoringMutationFormInputViews.
func AuthoringMutationFormInputViews(mutation AuthoringMutation) []map[string]string {
	return authoring.AuthoringMutationFormInputViews(mutation)
}

// Deprecated: use authoring.AuthoringMutationView.
func AuthoringMutationView(mutation AuthoringMutation) map[string]any {
	return authoring.AuthoringMutationView(mutation)
}

// Deprecated: use authoring.AuthoringFieldNamesView.
func AuthoringFieldNamesView() map[string]string { return authoring.AuthoringFieldNamesView() }

// Deprecated: use authoring.AuthoringRefreshFragments.
func AuthoringRefreshFragments(selectors ...string) []AuthoringRefreshFragment {
	return authoring.AuthoringRefreshFragments(selectors...)
}

// Deprecated: use authoring.AuthoringMutationResultView.
func AuthoringMutationResultView(result AuthoringMutationResult) map[string]any {
	return authoring.AuthoringMutationResultView(result)
}

// --- Authoring surface types (authoring/authoring.go) ---

// Deprecated: use authoring.AuthoringSurface.
type AuthoringSurface = authoring.AuthoringSurface

// Deprecated: use authoring.NoCodeAuthoringSurface.
func NoCodeAuthoringSurface(siteMap SiteMap) AuthoringSurface {
	return authoring.NoCodeAuthoringSurface(siteMap)
}

// Deprecated: use authoring.AuthoringSurfaceView.
func AuthoringSurfaceView(surface AuthoringSurface) map[string]any {
	return authoring.AuthoringSurfaceView(surface)
}

// --- Style mutation grammar (authoring/style_mutations.go) ---

const (
	// Deprecated: use authoring.StyleBreakpointBase.
	StyleBreakpointBase = authoring.StyleBreakpointBase
	// Deprecated: use authoring.StyleBreakpointTablet.
	StyleBreakpointTablet = authoring.StyleBreakpointTablet
	// Deprecated: use authoring.StyleBreakpointMobile.
	StyleBreakpointMobile = authoring.StyleBreakpointMobile
)

const (
	// Deprecated: use authoring.StyleStateDefault.
	StyleStateDefault = authoring.StyleStateDefault
	// Deprecated: use authoring.StyleStateHover.
	StyleStateHover = authoring.StyleStateHover
	// Deprecated: use authoring.StyleStateFocus.
	StyleStateFocus = authoring.StyleStateFocus
	// Deprecated: use authoring.StyleStateActive.
	StyleStateActive = authoring.StyleStateActive
	// Deprecated: use authoring.StyleStateDisabled.
	StyleStateDisabled = authoring.StyleStateDisabled
)

// Deprecated: use authoring.SupportedStyleProperties.
func SupportedStyleProperties() []string { return authoring.SupportedStyleProperties() }

// Deprecated: use authoring.IsSupportedStyleProperty.
func IsSupportedStyleProperty(property string) bool {
	return authoring.IsSupportedStyleProperty(property)
}

type LayoutValueOption = authoring.LayoutValueOption
type LayoutControl = authoring.LayoutControl

func ResponsiveLayoutControls() []LayoutControl { return authoring.ResponsiveLayoutControls() }
func ResponsiveLayoutProperties() []string      { return authoring.ResponsiveLayoutProperties() }
func IsResponsiveLayoutProperty(property string) bool {
	return authoring.IsResponsiveLayoutProperty(property)
}
func ValidateLayoutValue(property, value string) (string, bool) {
	return authoring.ValidateLayoutValue(property, value)
}

// Deprecated: use authoring.SupportedStyleBreakpoints.
func SupportedStyleBreakpoints() []string { return authoring.SupportedStyleBreakpoints() }

// Deprecated: use authoring.SupportedStyleStates.
func SupportedStyleStates() []string { return authoring.SupportedStyleStates() }

// Deprecated: use authoring.AuthoringMutationForStyle.
func AuthoringMutationForStyle(page Page, component Component, property, value, breakpoint, state string) AuthoringMutation {
	return authoring.AuthoringMutationForStyle(page, component, property, value, breakpoint, state)
}

// --- Component style overrides (authoring/component_styles.go) ---

// Deprecated: use authoring.StyleOverrides.
type StyleOverrides = authoring.StyleOverrides

// Deprecated: use authoring.RenderComponentStylesCSS.
func RenderComponentStylesCSS(overrides StyleOverrides) string {
	return authoring.RenderComponentStylesCSS(overrides)
}

// --- Style/appearance mutation handlers (authoring/style_handlers.go) ---

// Deprecated: use authoring.StyleDraftWriter.
type StyleDraftWriter = authoring.StyleDraftWriter

// Deprecated: use authoring.AppearanceWriter.
type AppearanceWriter = authoring.AppearanceWriter

// Deprecated: use authoring.ApplySetStyle.
func ApplySetStyle(m AuthoringMutation, write StyleDraftWriter) (AuthoringMutationResult, error) {
	return authoring.ApplySetStyle(m, write)
}

// Deprecated: use authoring.ApplySaveAppearance.
func ApplySaveAppearance(m AuthoringMutation, allowedKeys []string, write AppearanceWriter) (AuthoringMutationResult, error) {
	return authoring.ApplySaveAppearance(m, allowedKeys, write)
}

// --- Section-field binding grammar (authoring/section_binding.go) ---
//
// section_binding.go and handlers.go stayed at root through Slice 8 because
// the Style/section-field branch added them after the spec's §1 file table
// was written (see the Slice-8 spore); Slice 9 relocates them into
// m31labs.dev/gosx-studio/authoring, their spec §3 home (siblings of
// ApplySetStyle/ApplySaveAppearance — same mutation-boundary apply/parse
// family).

// Deprecated: use authoring.ParseSectionFieldBinding.
func ParseSectionFieldBinding(binding string) (sectionKey, field string, ok bool) {
	return authoring.ParseSectionFieldBinding(binding)
}

// Deprecated: use authoring.SectionFieldBinding.
func SectionFieldBinding(sectionKey, field string) string {
	return authoring.SectionFieldBinding(sectionKey, field)
}

// Deprecated: use authoring.StyleFormIDPart.
func StyleFormIDPart(s string) string { return authoring.StyleFormIDPart(s) }

// --- Section-field mutation handler (authoring/handlers.go) ---

// Deprecated: use authoring.SectionFieldWriter.
type SectionFieldWriter = authoring.SectionFieldWriter

// Deprecated: use authoring.ApplySaveSectionField.
func ApplySaveSectionField(m AuthoringMutation, write SectionFieldWriter, labelForField func(field string) string) (AuthoringMutationResult, error) {
	return authoring.ApplySaveSectionField(m, write, labelForField)
}
