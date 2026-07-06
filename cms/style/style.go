package style

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

type ScopeKind string
type ControlKind string
type ValueKind string

const (
	ScopeSite      ScopeKind = "site"
	ScopeRoute     ScopeKind = "route"
	ScopeTemplate  ScopeKind = "template"
	ScopeBlock     ScopeKind = "block"
	ScopeFlow      ScopeKind = "flow"
	ScopeComponent ScopeKind = "component"
	ScopeState     ScopeKind = "state"

	ControlChoice    ControlKind = "choice"
	ControlColorRole ControlKind = "color-role"
	ControlFontPair  ControlKind = "font-pair"
	ControlSpacing   ControlKind = "spacing"
	ControlRadius    ControlKind = "radius"
	ControlShadow    ControlKind = "shadow"
	ControlMotion    ControlKind = "motion"
	ControlCrop      ControlKind = "crop"
	ControlAlignment ControlKind = "alignment"

	ValueString ValueKind = "string"
	ValueNumber ValueKind = "number"
	ValueBool   ValueKind = "bool"
)

type DesignSystem struct {
	ID          string
	Label       string
	Brand       Brand
	Tokens      TokenSet
	Recipes     []ComponentRecipe
	Scopes      []StyleScope
	Breakpoints []Breakpoint
}

type Brand struct {
	Name        string
	Description string
	Tone        string
}

type TokenSet struct {
	Colors     map[string]string
	Typography map[string]string
	Spacing    map[string]string
	Radii      map[string]string
	Shadows    map[string]string
	Motion     map[string]string
	Custom     map[string]string
}

type StyleScope struct {
	Kind     ScopeKind
	TargetID string
	Values   map[string]Value
}

type ComponentRecipe struct {
	Key      string
	Label    string
	Controls []StyleControl
	Variants []StyleVariant
}

type StyleControl struct {
	Key      string
	Label    string
	Kind     ControlKind
	Options  []StyleOption
	Default  Value
	Required bool
}

type RecipeView struct {
	Key      string
	Label    string
	Controls []ControlView
	Variants []VariantView
}

type ControlView struct {
	Key        string
	Label      string
	Kind       string
	Options    []OptionView
	Default    string
	HasDefault bool
	Required   bool
}

type OptionView struct {
	Value string
	Label string
	CSS   string
}

type VariantView struct {
	Key    string
	Label  string
	Values map[string]string
}

type StyleOption struct {
	Value string
	Label string
	CSS   string
}

type StyleVariant struct {
	Key    string
	Label  string
	Values map[string]Value
}

type Breakpoint struct {
	Key      string
	Label    string
	MinWidth string
}

type Value struct {
	Kind   ValueKind
	String string
	Number float64
	Bool   bool
}

type ValidationError struct {
	Path    string
	Message string
}

type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err.Path == "" {
			parts = append(parts, err.Message)
			continue
		}
		parts = append(parts, err.Path+": "+err.Message)
	}
	return strings.Join(parts, "; ")
}

func String(value string) Value {
	return Value{Kind: ValueString, String: strings.TrimSpace(value)}
}

func Number(value float64) Value {
	return Value{Kind: ValueNumber, Number: value}
}

func Bool(value bool) Value {
	return Value{Kind: ValueBool, Bool: value}
}

func Normalize(system DesignSystem) DesignSystem {
	system.ID = slug(firstNonEmpty(system.ID, system.Brand.Name, system.Label, "studio"))
	system.Label = strings.TrimSpace(system.Label)
	system.Brand.Name = strings.TrimSpace(system.Brand.Name)
	system.Brand.Description = strings.TrimSpace(system.Brand.Description)
	system.Brand.Tone = strings.TrimSpace(system.Brand.Tone)
	system.Tokens = normalizeTokens(system.Tokens)
	system.Recipes = normalizeRecipes(system.Recipes)
	system.Scopes = normalizeScopes(system.Scopes)
	system.Breakpoints = normalizeBreakpoints(system.Breakpoints)
	return system
}

func Validate(system DesignSystem) ValidationErrors {
	system = Normalize(system)
	var errs ValidationErrors
	if system.ID == "" {
		errs = append(errs, ValidationError{Path: "id", Message: "design system id is required"})
	}
	errs = append(errs, validateTokenContrast(system.Tokens)...)
	controls := controlsByKey(system.Recipes)
	for index, scope := range system.Scopes {
		path := fmt.Sprintf("scopes[%d]", index)
		if !scopeKindAllowed(scope.Kind) {
			errs = append(errs, ValidationError{Path: path + ".kind", Message: "unsupported style scope"})
		}
		if scope.Kind != ScopeSite && strings.TrimSpace(scope.TargetID) == "" {
			errs = append(errs, ValidationError{Path: path + ".targetId", Message: "target id is required"})
		}
		for key, value := range scope.Values {
			control, ok := controls[key]
			if len(controls) > 0 && !ok {
				errs = append(errs, ValidationError{Path: path + ".values." + key, Message: "style control is not defined by a recipe"})
				continue
			}
			if ok && len(control.Options) > 0 && !optionAllowed(value.CSS(), control.Options) {
				errs = append(errs, ValidationError{Path: path + ".values." + key, Message: "choose one of the available options"})
			}
		}
	}
	return errs
}

func CompileCSS(system DesignSystem) (string, ValidationErrors) {
	system = Normalize(system)
	if errs := Validate(system); len(errs) > 0 {
		return "", errs
	}
	var b strings.Builder
	writeRule(&b, systemSelector(system.ID), tokenDeclarations(system.Tokens))
	for _, scope := range system.Scopes {
		declarations := scopeDeclarations(scope)
		if len(declarations) == 0 {
			continue
		}
		writeRule(&b, scopeSelector(system.ID, scope), declarations)
	}
	return strings.TrimSpace(b.String()), nil
}

func RecipeViews(system DesignSystem) []RecipeView {
	system = Normalize(system)
	out := make([]RecipeView, 0, len(system.Recipes))
	for _, recipe := range system.Recipes {
		view := RecipeView{
			Key:      recipe.Key,
			Label:    recipe.Label,
			Controls: make([]ControlView, 0, len(recipe.Controls)),
			Variants: make([]VariantView, 0, len(recipe.Variants)),
		}
		for _, control := range recipe.Controls {
			controlView := ControlView{
				Key:      control.Key,
				Label:    control.Label,
				Kind:     string(control.Kind),
				Required: control.Required,
				Options:  make([]OptionView, 0, len(control.Options)),
			}
			if defaultValue, ok := valueDefault(control.Default); ok {
				controlView.Default = defaultValue
				controlView.HasDefault = true
			}
			for _, option := range control.Options {
				controlView.Options = append(controlView.Options, OptionView{
					Value: option.Value,
					Label: option.Label,
					CSS:   option.CSS,
				})
			}
			view.Controls = append(view.Controls, controlView)
		}
		for _, variant := range recipe.Variants {
			values := map[string]string{}
			for key, value := range variant.Values {
				values[key] = value.CSS()
			}
			if len(values) == 0 {
				values = nil
			}
			view.Variants = append(view.Variants, VariantView{
				Key:    variant.Key,
				Label:  variant.Label,
				Values: values,
			})
		}
		out = append(out, view)
	}
	return out
}

func ScopeClass(scope StyleScope) string {
	scope.Kind = ScopeKind(slug(string(scope.Kind)))
	scope.TargetID = slug(scope.TargetID)
	if scope.Kind == "" || scope.Kind == ScopeSite {
		return ""
	}
	return "gosx-style-" + string(scope.Kind) + "-" + scope.TargetID
}

func (value Value) CSS() string {
	switch value.Kind {
	case ValueNumber:
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", value.Number), "0"), ".")
	case ValueBool:
		if value.Bool {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(value.String)
	}
}

func normalizeTokens(tokens TokenSet) TokenSet {
	return TokenSet{
		Colors:     normalizeValueMap(tokens.Colors, normalizeColor),
		Typography: normalizeValueMap(tokens.Typography, strings.TrimSpace),
		Spacing:    normalizeValueMap(tokens.Spacing, strings.TrimSpace),
		Radii:      normalizeValueMap(tokens.Radii, strings.TrimSpace),
		Shadows:    normalizeValueMap(tokens.Shadows, strings.TrimSpace),
		Motion:     normalizeValueMap(tokens.Motion, strings.TrimSpace),
		Custom:     normalizeValueMap(tokens.Custom, strings.TrimSpace),
	}
}

func normalizeValueMap(input map[string]string, normalize func(string) string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range input {
		key = slug(key)
		value = normalize(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeRecipes(recipes []ComponentRecipe) []ComponentRecipe {
	out := make([]ComponentRecipe, 0, len(recipes))
	seen := map[string]bool{}
	for _, recipe := range recipes {
		recipe.Key = slug(recipe.Key)
		if recipe.Key == "" || seen[recipe.Key] {
			continue
		}
		recipe.Label = firstNonEmpty(recipe.Label, recipe.Key)
		recipe.Controls = normalizeControls(recipe.Controls)
		recipe.Variants = normalizeVariants(recipe.Variants)
		seen[recipe.Key] = true
		out = append(out, recipe)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func normalizeControls(controls []StyleControl) []StyleControl {
	out := make([]StyleControl, 0, len(controls))
	seen := map[string]bool{}
	for _, control := range controls {
		control.Key = slug(control.Key)
		if control.Key == "" || seen[control.Key] {
			continue
		}
		control.Label = firstNonEmpty(control.Label, control.Key)
		control.Kind = ControlKind(slug(string(control.Kind)))
		if control.Kind == "" {
			control.Kind = ControlChoice
		}
		control.Options = normalizeOptions(control.Options)
		control.Default = normalizeValue(control.Default)
		seen[control.Key] = true
		out = append(out, control)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func normalizeOptions(options []StyleOption) []StyleOption {
	out := make([]StyleOption, 0, len(options))
	seen := map[string]bool{}
	for _, option := range options {
		option.Value = slug(option.Value)
		if option.Value == "" || seen[option.Value] {
			continue
		}
		option.Label = firstNonEmpty(option.Label, option.Value)
		option.CSS = strings.TrimSpace(option.CSS)
		seen[option.Value] = true
		out = append(out, option)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Value < out[j].Value })
	return out
}

func normalizeVariants(variants []StyleVariant) []StyleVariant {
	out := make([]StyleVariant, 0, len(variants))
	seen := map[string]bool{}
	for _, variant := range variants {
		variant.Key = slug(variant.Key)
		if variant.Key == "" || seen[variant.Key] {
			continue
		}
		variant.Label = firstNonEmpty(variant.Label, variant.Key)
		variant.Values = normalizeStyleValues(variant.Values)
		seen[variant.Key] = true
		out = append(out, variant)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func normalizeScopes(scopes []StyleScope) []StyleScope {
	out := make([]StyleScope, 0, len(scopes))
	for _, scope := range scopes {
		scope.Kind = ScopeKind(slug(string(scope.Kind)))
		if scope.Kind == "" {
			scope.Kind = ScopeSite
		}
		scope.TargetID = slug(scope.TargetID)
		scope.Values = normalizeStyleValues(scope.Values)
		if scope.Kind != ScopeSite && scope.TargetID == "" && len(scope.Values) == 0 {
			continue
		}
		out = append(out, scope)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].TargetID < out[j].TargetID
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func normalizeStyleValues(values map[string]Value) map[string]Value {
	if len(values) == 0 {
		return nil
	}
	out := map[string]Value{}
	for key, value := range values {
		key = slug(key)
		if key == "" {
			continue
		}
		out[key] = normalizeValue(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeValue(value Value) Value {
	if value.Kind == "" {
		if strings.TrimSpace(value.String) == "" {
			return Value{}
		}
		value.Kind = ValueString
	}
	value.String = strings.TrimSpace(value.String)
	return value
}

func valueDefault(value Value) (string, bool) {
	if value.Kind == "" {
		return "", false
	}
	css := value.CSS()
	if css == "" {
		return "", false
	}
	return css, true
}

func normalizeBreakpoints(breakpoints []Breakpoint) []Breakpoint {
	out := make([]Breakpoint, 0, len(breakpoints))
	seen := map[string]bool{}
	for _, breakpoint := range breakpoints {
		breakpoint.Key = slug(breakpoint.Key)
		if breakpoint.Key == "" || seen[breakpoint.Key] {
			continue
		}
		breakpoint.Label = firstNonEmpty(breakpoint.Label, breakpoint.Key)
		breakpoint.MinWidth = strings.TrimSpace(breakpoint.MinWidth)
		seen[breakpoint.Key] = true
		out = append(out, breakpoint)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

func controlsByKey(recipes []ComponentRecipe) map[string]StyleControl {
	out := map[string]StyleControl{}
	for _, recipe := range recipes {
		for _, control := range recipe.Controls {
			out[control.Key] = control
		}
	}
	return out
}

func tokenDeclarations(tokens TokenSet) map[string]string {
	out := map[string]string{}
	addTokenGroup(out, "color", tokens.Colors)
	addTokenGroup(out, "font", tokens.Typography)
	addTokenGroup(out, "space", tokens.Spacing)
	addTokenGroup(out, "radius", tokens.Radii)
	addTokenGroup(out, "shadow", tokens.Shadows)
	addTokenGroup(out, "motion", tokens.Motion)
	for key, value := range tokens.Custom {
		if strings.HasPrefix(key, "--") {
			out[key] = value
			continue
		}
		out["--"+key] = value
	}
	return out
}

func addTokenGroup(out map[string]string, prefix string, values map[string]string) {
	for key, value := range values {
		out["--"+prefix+"-"+key] = value
	}
}

func scopeDeclarations(scope StyleScope) map[string]string {
	out := map[string]string{}
	for key, value := range scope.Values {
		css := value.CSS()
		if css == "" {
			continue
		}
		out["--style-"+key] = css
	}
	return out
}

func writeRule(b *strings.Builder, selector string, declarations map[string]string) {
	if selector == "" || len(declarations) == 0 {
		return
	}
	keys := make([]string, 0, len(declarations))
	for key := range declarations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	b.WriteString(selector)
	b.WriteString("{")
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString(":")
		b.WriteString(declarations[key])
		b.WriteString(";")
	}
	b.WriteString("}\n")
}

func systemSelector(systemID string) string {
	return ".gosx-style-" + slug(systemID)
}

func scopeSelector(systemID string, scope StyleScope) string {
	if scope.Kind == ScopeSite {
		return systemSelector(systemID)
	}
	return systemSelector(systemID) + " ." + ScopeClass(scope)
}

func scopeKindAllowed(kind ScopeKind) bool {
	switch kind {
	case ScopeSite, ScopeRoute, ScopeTemplate, ScopeBlock, ScopeFlow, ScopeComponent, ScopeState:
		return true
	default:
		return false
	}
}

func optionAllowed(value string, options []StyleOption) bool {
	value = slug(value)
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func slug(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	var previousDash bool
	var previousLowerOrDigit bool
	for _, r := range value {
		switch {
		case r == '_' || r == '-' || unicode.IsSpace(r):
			if !previousDash && b.Len() > 0 {
				b.WriteByte('-')
				previousDash = true
			}
			previousLowerOrDigit = false
		case unicode.IsUpper(r):
			if previousLowerOrDigit && !previousDash {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
			previousDash = false
			previousLowerOrDigit = true
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			previousDash = false
			previousLowerOrDigit = true
		}
	}
	return strings.Trim(b.String(), "-")
}
