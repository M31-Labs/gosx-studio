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
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx/action"
)

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
	AuthoringOperationSetStyle = authoring.AuthoringOperationSetStyle
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
	AuthoringFieldVisible = authoring.AuthoringFieldVisible
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

// --- Unexported shims for not-yet-moved root files ---
//
// authoring.go also carried an unexported composition-intent normalizer that
// root-only files (not moved until later slices) call by its original
// lowercase name. sitemap_view.go's AuthoringSiteMapView (moving in the
// sitemap slice) is the one remaining caller. This var keeps that call site
// compiling unchanged against the new authoring implementation; it is
// internal wiring, not part of the public facade, and will disappear once
// sitemap_view.go moves to its own subpackage.
var normalizeCompositionIntents = authoring.NormalizeCompositionIntents
