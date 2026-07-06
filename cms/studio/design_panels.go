package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type StudioOption struct {
	Value    string
	Label    string
	Selected bool
	Attrs    []FieldAttribute
}

type ChoiceCard struct {
	ID         string
	Name       string
	Value      string
	Label      string
	Summary    string
	Class      string
	Checked    bool
	CardAttrs  []FieldAttribute
	InputAttrs []FieldAttribute
}

type ChoiceGroup struct {
	Class     string
	GridClass string
	Legend    string
	Cards     []ChoiceCard
}

type SelectControl struct {
	ID      string
	Name    string
	Label   string
	Class   string
	Wide    bool
	Attrs   []FieldAttribute
	Options []StudioOption
}

type RadioControlGroup struct {
	Class   string
	Legend  string
	Name    string
	Attrs   []FieldAttribute
	Options []StudioOption
}

type ColorTokenControl struct {
	Key    string
	Name   string
	Label  string
	CSSVar string
	Value  string
}

type BrandPanelOptions struct {
	PanelClass       string
	PanelKey         string
	Mode             string
	Title            string
	Fields           []InspectorField
	LayoutFields     []InspectorField
	SnapChecked      bool
	SnapSize         int
	GridLabel        string
	ResetLabel       string
	PreviewLabel     string
	LogoURL          string
	LogoAlt          string
	LogoButtonLabel  string
	MediaHref        string
	MediaLabel       string
	MediaButtonClass string
}

type StyleSettingsPanelOptions struct {
	PanelClass         string
	PanelKey           string
	Mode               string
	Title              string
	WorkbenchHTML      string
	KitGroup           ChoiceGroup
	TemplateGroup      ChoiceGroup
	CustomBuilderClass string
	CustomNameField    InspectorField
	CustomControls     []SelectControl
	Palette            SelectControl
	Swatches           []ColorTokenControl
	StyleControls      []SelectControl
	ImageCrop          RadioControlGroup
	ColorTokens        []ColorTokenControl
}

func RenderBrandPanel(options BrandPanelOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "editor-panel")
	panelKey := firstNonEmpty(options.PanelKey, "brand")
	mode := firstNonEmpty(options.Mode, "brand")
	nodes := []gosx.Node{gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, "Brand")))}
	for _, field := range normalizeInspectorFields(options.Fields) {
		nodes = append(nodes, renderInspectorField(field))
	}
	layoutFields := normalizeInspectorFields(options.LayoutFields)
	if len(layoutFields) > 0 {
		gridNodes := make([]gosx.Node, 0, len(layoutFields))
		for _, field := range layoutFields {
			gridNodes = append(gridNodes, renderInspectorField(field))
		}
		nodes = append(nodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-control-grid")), gosx.Fragment(gridNodes...)))
	}
	nodes = append(nodes, renderBrandTools(options), renderBrandPreview(options))
	if strings.TrimSpace(options.MediaHref) != "" {
		nodes = append(nodes, gosx.El("a", gosx.Attrs(
			gosx.Attr("class", firstNonEmpty(options.MediaButtonClass, "button button--secondary")),
			gosx.Attr("href", options.MediaHref),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text(firstNonEmpty(options.MediaLabel, "Upload brand assets"))))
	}
	return gosx.El("section", panelAttrs(panelClass, panelKey, mode), gosx.Fragment(nodes...))
}

func RenderStyleSettingsPanel(options StyleSettingsPanelOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "editor-panel")
	panelKey := firstNonEmpty(options.PanelKey, "style")
	mode := firstNonEmpty(options.Mode, "style")
	nodes := []gosx.Node{gosx.El("h2", nil, gosx.Text(firstNonEmpty(options.Title, "Style")))}
	if strings.TrimSpace(options.WorkbenchHTML) != "" {
		nodes = append(nodes, gosx.RawHTML(options.WorkbenchHTML))
	}
	if len(options.KitGroup.Cards) > 0 {
		nodes = append(nodes, renderChoiceGroup(options.KitGroup))
	}
	if len(options.TemplateGroup.Cards) > 0 {
		nodes = append(nodes, renderChoiceGroup(options.TemplateGroup))
	}
	nodes = append(nodes, renderCustomTemplateBuilder(options))
	if options.Palette.Name != "" {
		nodes = append(nodes, renderPaletteField(options.Palette, options.Swatches))
	}
	controlNodes := []gosx.Node{}
	for _, control := range normalizeSelectControls(options.StyleControls) {
		controlNodes = append(controlNodes, renderSelectControl(control))
	}
	if options.ImageCrop.Name != "" {
		controlNodes = append(controlNodes, renderRadioControlGroup(options.ImageCrop))
	}
	if len(controlNodes) > 0 {
		nodes = append(nodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-control-grid")), gosx.Fragment(controlNodes...)))
	}
	if len(options.ColorTokens) > 0 {
		nodes = append(nodes, renderColorTokenGrid(options.ColorTokens))
	}
	return gosx.El("section", panelAttrs(panelClass, panelKey, mode), gosx.Fragment(nodes...))
}

func renderBrandTools(options BrandPanelOptions) gosx.Node {
	snapAttrs := []any{
		gosx.Attr("type", "checkbox"),
		gosx.Attr("data-editor-logo-snap", "true"),
	}
	if options.SnapChecked {
		snapAttrs = append(snapAttrs, gosx.BoolAttr("checked"))
	}
	snapSize := options.SnapSize
	if snapSize <= 0 {
		snapSize = 8
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-brand-tools")),
		gosx.El("label", gosx.Attrs(gosx.Attr("class", "editor-brand-snap")),
			gosx.El("input", gosx.Attrs(snapAttrs...)),
			gosx.El("span", nil, gosx.Text("Snap")),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--compact")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "logoSnapSize")), gosx.Text(firstNonEmpty(options.GridLabel, "Grid"))),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", "logoSnapSize"),
				gosx.Attr("type", "number"),
				gosx.Attr("min", "1"),
				gosx.Attr("max", "32"),
				gosx.Attr("value", snapSize),
				gosx.Attr("data-editor-logo-snap-size", "true"),
			)),
		),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "home-section-move"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-editor-logo-reset", "true"),
		), gosx.Text(firstNonEmpty(options.ResetLabel, "Reset"))),
	)
}

func renderBrandPreview(options BrandPanelOptions) gosx.Node {
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", "editor-brand-preview"),
		gosx.Attr("data-editor-brand-preview", "true"),
	),
		gosx.El("span", gosx.Attrs(gosx.Attr("class", "editor-brand-preview__corner")), gosx.Text(firstNonEmpty(options.PreviewLabel, "Top left"))),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "editor-brand-handle"),
			gosx.Attr("type", "button"),
			gosx.Attr("aria-label", firstNonEmpty(options.LogoButtonLabel, "Position logo")),
			gosx.Attr("data-editor-brand-handle", "true"),
		),
			gosx.El("img", gosx.Attrs(
				gosx.Attr("src", options.LogoURL),
				gosx.Attr("alt", firstNonEmpty(options.LogoAlt, "Site logo")),
				gosx.Attr("data-editor-brand-logo", "true"),
			)),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("class", "editor-brand-readout"),
			gosx.Attr("data-editor-logo-readout", "true"),
			gosx.Attr("aria-live", "polite"),
		)),
	)
}

func renderChoiceGroup(group ChoiceGroup) gosx.Node {
	cards := normalizeChoiceCards(group.Cards)
	cardNodes := make([]gosx.Node, 0, len(cards))
	for _, card := range cards {
		cardNodes = append(cardNodes, renderChoiceCard(card))
	}
	return gosx.El("fieldset", gosx.Attrs(gosx.Attr("class", firstNonEmpty(group.Class, "template-picker"))),
		gosx.El("legend", nil, gosx.Text(group.Legend)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(group.GridClass, "template-picker__grid"))), gosx.Fragment(cardNodes...)),
	)
}

func renderChoiceCard(card ChoiceCard) gosx.Node {
	inputAttrs := []any{
		gosx.Attr("type", "radio"),
		gosx.Attr("name", card.Name),
		gosx.Attr("value", card.Value),
	}
	if card.Checked {
		inputAttrs = append(inputAttrs, gosx.BoolAttr("checked"))
	}
	inputAttrs = appendFieldAttributes(inputAttrs, card.InputAttrs)
	cardAttrs := []any{gosx.Attr("class", firstNonEmpty(card.Class, "template-card"))}
	cardAttrs = appendFieldAttributes(cardAttrs, card.CardAttrs)
	return gosx.El("label", gosx.Attrs(cardAttrs...),
		gosx.El("input", gosx.Attrs(inputAttrs...)),
		gosx.El("span", nil,
			gosx.El("strong", nil, gosx.Text(card.Label)),
			gosx.El("small", nil, gosx.Text(card.Summary)),
		),
	)
}

func renderCustomTemplateBuilder(options StyleSettingsPanelOptions) gosx.Node {
	controlNodes := []gosx.Node{}
	if options.CustomNameField.ID != "" || options.CustomNameField.Name != "" {
		controlNodes = append(controlNodes, renderInspectorField(options.CustomNameField))
	}
	customControls := normalizeSelectControls(options.CustomControls)
	if len(customControls) > 0 {
		selectNodes := make([]gosx.Node, 0, len(customControls))
		for _, control := range customControls {
			selectNodes = append(selectNodes, renderSelectControl(control))
		}
		controlNodes = append(controlNodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-control-grid")), gosx.Fragment(selectNodes...)))
	}
	if len(controlNodes) == 0 {
		return gosx.Fragment()
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", firstNonEmpty(options.CustomBuilderClass, "custom-template-builder")),
		gosx.Attr("data-custom-template-builder", "true"),
	), gosx.Fragment(controlNodes...))
}

func renderPaletteField(control SelectControl, swatches []ColorTokenControl) gosx.Node {
	nodes := renderSelectControlContent(control)
	if len(swatches) > 0 {
		swatchNodes := make([]gosx.Node, 0, len(swatches))
		for _, swatch := range swatches {
			swatchNodes = append(swatchNodes, gosx.El("span", gosx.Attrs(
				gosx.Attr("class", "theme-swatch"),
				gosx.Attr("style", "background:"+swatch.Value),
				gosx.Attr("title", swatch.Label),
			)))
		}
		nodes = append(nodes, gosx.El("div", gosx.Attrs(
			gosx.Attr("class", "theme-swatch-row"),
			gosx.Attr("data-editor-theme-swatches", "true"),
		), gosx.Fragment(swatchNodes...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "field-row field-row--wide")), gosx.Fragment(nodes...))
}

func renderSelectControl(control SelectControl) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", selectFieldClass(control))),
		gosx.Fragment(renderSelectControlContent(control)...),
	)
}

func renderSelectControlContent(control SelectControl) []gosx.Node {
	optionNodes := make([]gosx.Node, 0, len(control.Options))
	for _, option := range normalizeStudioOptions(control.Options) {
		optionAttrs := []any{gosx.Attr("value", option.Value)}
		if option.Selected {
			optionAttrs = append(optionAttrs, gosx.BoolAttr("selected"))
		}
		optionAttrs = appendFieldAttributes(optionAttrs, option.Attrs)
		optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(optionAttrs...), gosx.Text(option.Label)))
	}
	selectAttrs := []any{
		gosx.Attr("id", control.ID),
		gosx.Attr("name", control.Name),
	}
	selectAttrs = appendFieldAttributes(selectAttrs, control.Attrs)
	return []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", control.ID)), gosx.Text(control.Label)),
		gosx.El("select", gosx.Attrs(selectAttrs...), gosx.Fragment(optionNodes...)),
	}
}

func renderRadioControlGroup(group RadioControlGroup) gosx.Node {
	options := normalizeStudioOptions(group.Options)
	nodes := make([]gosx.Node, 0, len(options))
	for _, option := range options {
		attrs := []any{
			gosx.Attr("type", "radio"),
			gosx.Attr("name", group.Name),
			gosx.Attr("value", option.Value),
		}
		if option.Selected {
			attrs = append(attrs, gosx.BoolAttr("checked"))
		}
		attrs = appendFieldAttributes(attrs, option.Attrs)
		nodes = append(nodes, gosx.El("label", nil,
			gosx.El("input", gosx.Attrs(attrs...)),
			gosx.Text(" "+option.Label),
		))
	}
	fieldsetAttrs := []any{gosx.Attr("class", firstNonEmpty(group.Class, "radio-row"))}
	fieldsetAttrs = appendFieldAttributes(fieldsetAttrs, group.Attrs)
	return gosx.El("fieldset", gosx.Attrs(fieldsetAttrs...),
		gosx.El("legend", nil, gosx.Text(group.Legend)),
		gosx.Fragment(nodes...),
	)
}

func renderColorTokenGrid(tokens []ColorTokenControl) gosx.Node {
	nodes := make([]gosx.Node, 0, len(tokens))
	for _, token := range tokens {
		token.Name = strings.TrimSpace(token.Name)
		if token.Name == "" {
			continue
		}
		nodes = append(nodes, gosx.El("label", gosx.Attrs(
			gosx.Attr("class", "color-field"),
			gosx.Attr("for", token.Name),
		),
			gosx.El("span", nil, gosx.Text(token.Label)),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("id", token.Name),
				gosx.Attr("name", token.Name),
				gosx.Attr("type", "color"),
				gosx.Attr("value", token.Value),
				gosx.Attr("data-editor-color-token", token.CSSVar),
				gosx.Attr("data-editor-color-key", token.Key),
			)),
		))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-color-grid")), gosx.Fragment(nodes...))
}

func selectFieldClass(control SelectControl) string {
	className := firstNonEmpty(control.Class, "field-row")
	if control.Wide && !strings.Contains(" "+className+" ", " field-row--wide ") {
		className += " field-row--wide"
	}
	return strings.TrimSpace(className)
}

func appendFieldAttributes(attrs []any, fields []FieldAttribute) []any {
	for _, attr := range fields {
		name := strings.TrimSpace(attr.Name)
		if name == "" {
			continue
		}
		if attr.Bool {
			attrs = append(attrs, gosx.BoolAttr(name))
			continue
		}
		attrs = append(attrs, gosx.Attr(name, attr.Value))
	}
	return attrs
}

func normalizeChoiceCards(cards []ChoiceCard) []ChoiceCard {
	out := make([]ChoiceCard, 0, len(cards))
	for _, card := range cards {
		card.Name = strings.TrimSpace(card.Name)
		card.Value = strings.TrimSpace(card.Value)
		card.Label = strings.TrimSpace(card.Label)
		card.Summary = strings.TrimSpace(card.Summary)
		if card.Name == "" || card.Value == "" || card.Label == "" {
			continue
		}
		out = append(out, card)
	}
	return out
}

func normalizeSelectControls(controls []SelectControl) []SelectControl {
	out := make([]SelectControl, 0, len(controls))
	for _, control := range controls {
		control.ID = strings.TrimSpace(control.ID)
		control.Name = strings.TrimSpace(control.Name)
		control.Label = strings.TrimSpace(control.Label)
		if control.ID == "" || control.Name == "" || control.Label == "" {
			continue
		}
		control.Options = normalizeStudioOptions(control.Options)
		out = append(out, control)
	}
	return out
}

func normalizeStudioOptions(options []StudioOption) []StudioOption {
	out := make([]StudioOption, 0, len(options))
	for _, option := range options {
		option.Value = strings.TrimSpace(option.Value)
		option.Label = strings.TrimSpace(option.Label)
		if option.Value == "" || option.Label == "" {
			continue
		}
		out = append(out, option)
	}
	return out
}
