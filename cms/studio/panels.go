package studio

import (
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-admin/calendar"
	"m31labs.dev/gosx-admin/workbench"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

type PanelHeadingOptions struct {
	Class  string
	Kicker string
	Title  string
}

type SiteNavItem struct {
	Key     string
	Label   string
	Href    string
	Summary string
	Class   string
	Active  bool
}

type SiteNavigatorOptions struct {
	Class        string
	HeadingClass string
	NavClass     string
	Kicker       string
	Title        string
	Label        string
	Mode         string
}

type InspectorHeaderOptions struct {
	Class          string
	Kicker         string
	ModeLabel      string
	SelectionLabel string
}

type ScopeCrumb struct {
	Label            string
	DynamicMode      bool
	DynamicSelection bool
}

type ScopeStripOptions struct {
	Class  string
	Label  string
	Crumbs []ScopeCrumb
}

type LayerPreviewAction struct {
	Label  string
	Class  string
	Source string
}

type LayerPreview struct {
	VisualClass string
	Kicker      string
	Title       string
	TitleSource string
	Body        string
	BodySource  string
	Actions     []LayerPreviewAction
}

type LayerItem struct {
	Key           string
	Label         string
	CardClass     string
	StatusClass   string
	StatusLabel   string
	DragLabel     string
	MoveUpLabel   string
	MoveDownLabel string
	KeyName       string
	OrderName     string
	EnabledName   string
	Order         int
	Enabled       bool
	Preview       LayerPreview
}

type LayerListOptions struct {
	Class          string
	PanelClass     string
	HeadingClass   string
	Kicker         string
	Title          string
	Mode           string
	BlockStudioKey string
}

type BlockLibraryItem struct {
	Key             string
	Label           string
	Summary         string
	Target          string
	ButtonLabel     string
	ButtonClass     string
	ButtonBaseClass string
	Active          bool
}

type BlockLibraryOptions struct {
	Class      string
	PanelClass string
	PanelKey   string
	Mode       string
	Title      string
}

type PanelLink struct {
	Key     string
	Label   string
	Summary string
	Href    string
	Class   string
}

type LinkGridOptions struct {
	Class      string
	PanelClass string
	PanelKey   string
	Mode       string
	GridClass  string
	Kicker     string
	Title      string
}

type FlowField struct {
	Name          string
	Label         string
	RequiredLabel string
}

type FlowAction struct {
	Key    string
	Label  string
	Fields []FlowField
}

type FlowStep struct {
	Key        string
	Label      string
	BlockCount int
	HasBlocks  bool
}

type FlowCard struct {
	Key                string
	Label              string
	Description        string
	Summary            string
	StatusClass        string
	StatusLabel        string
	CardClass          string
	Route              string
	EmbedTarget        string
	PrimaryHandlerRef  string
	RequiredFieldCount int
	CanExecute         bool
	HasRoute           bool
	HasEmbedTarget     bool
	HasPrimaryAction   bool
	Steps              []FlowStep
	Actions            []FlowAction
}

type FlowLibraryOptions struct {
	Class      string
	PanelClass string
	PanelKey   string
	Mode       string
	Kicker     string
	Title      string
}

type FieldAttribute struct {
	Name  string
	Value string
	Bool  bool
}

func SelectionScopeAttrs(keys ...string) []FieldAttribute {
	scopes := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			scopes = append(scopes, key)
		}
	}
	if len(scopes) == 0 {
		return nil
	}
	return []FieldAttribute{
		{Name: "data-studio-inspector-for", Value: strings.Join(scopes, " ")},
	}
}

type FieldAction struct {
	Label      string
	Href       string
	FormAction string
	FormMethod string
	SubmitKey  string
	Name       string
	Value      string
	Confirm    string
	Class      string
	Primary    bool
}

type InspectorFieldKind string

const (
	InspectorFieldInput    InspectorFieldKind = "input"
	InspectorFieldArea     InspectorFieldKind = "textarea"
	InspectorFieldSelect   InspectorFieldKind = "select"
	InspectorFieldCheckbox InspectorFieldKind = "checkbox"
	InspectorFieldCard     InspectorFieldKind = "card"
)

type InspectorFieldOption struct {
	Value    string
	Label    string
	Selected bool
}

type InspectorField struct {
	Kind           InspectorFieldKind
	ID             string
	Name           string
	Label          string
	Value          string
	Type           string
	Placeholder    string
	Required       bool
	Checked        bool
	Disabled       bool
	Rows           int
	Wide           bool
	Help           string
	Class          string
	CardTitle      string
	Options        []InspectorFieldOption
	Attrs          []FieldAttribute
	ContainerAttrs []FieldAttribute
	Actions        []FieldAction
}

type BlockInspectorOptions struct {
	Field          string
	Editable       string
	ActionLabel    string
	Label          string
	CardTitle      string
	Class          string
	Narrow         bool
	Attrs          []FieldAttribute
	ContainerAttrs []FieldAttribute
	Actions        []FieldAction
}

type InspectorFieldOverride struct {
	Kind           InspectorFieldKind
	ID             string
	Name           string
	Label          string
	Value          string
	Source         string
	Type           string
	Placeholder    string
	Required       bool
	Checked        bool
	Disabled       bool
	Rows           int
	Wide           bool
	Help           string
	Options        []InspectorFieldOption
	Attrs          []FieldAttribute
	ContainerAttrs []FieldAttribute
}

type BlockFieldInspectorOptions struct {
	NamePrefix      string
	IDPrefix        string
	SourcePrefix    string
	MediaListID     string
	Values          map[string]string
	MediaAltTargets map[string]string
	Overrides       map[string]InspectorFieldOverride
	ContainerAttrs  []FieldAttribute
	Wide            bool
}

type WorkbenchFieldInspectorOptions struct {
	NamePrefix      string
	IDPrefix        string
	SourcePrefix    string
	MediaListID     string
	Values          map[string]string
	MediaAltTargets map[string]string
	Overrides       map[string]InspectorFieldOverride
	ContainerAttrs  []FieldAttribute
	Wide            bool
	Disabled        bool
}

type LifecycleInspectorOptions struct {
	DraftState     lifecycle.DraftState
	PublishState   lifecycle.PublishState
	PreviewHref    string
	PublishHref    string
	PublishAction  string
	ScheduleHref   string
	ScheduleAction string
	ContainerAttrs []FieldAttribute
	Disabled       bool
}

type FlowConfigInspectorOptions struct {
	NamePrefix     string
	IDPrefix       string
	SourcePrefix   string
	Values         map[string]string
	PublishReview  string
	PublishAction  string
	ConfigureHref  string
	ContainerAttrs []FieldAttribute
	Disabled       bool
}

type FlowStepInspectorOptions struct {
	NamePrefix     string
	IDPrefix       string
	SourcePrefix   string
	Values         map[string]string
	ContainerAttrs []FieldAttribute
	Disabled       bool
}

type CalendarWidgetInspectorOptions struct {
	NamePrefix     string
	IDPrefix       string
	SourcePrefix   string
	Values         map[string]string
	ContainerAttrs []FieldAttribute
	Disabled       bool
}

func BlockInspectorFields(catalog []blockstudio.Definition, options map[string]BlockInspectorOptions) []InspectorField {
	out := []InspectorField{}
	for _, definition := range catalog {
		option, ok := options[definition.Key]
		if !ok {
			continue
		}
		out = append(out, BlockInspectorField(definition, option))
	}
	return out
}

func CatalogFieldInspectorFields(catalog []blockstudio.Definition, options map[string]BlockFieldInspectorOptions) []InspectorField {
	out := []InspectorField{}
	for _, definition := range catalog {
		option, ok := options[definition.Key]
		if !ok {
			continue
		}
		out = append(out, BlockFieldInspectorFields(definition, option)...)
	}
	return out
}

func BlockFieldInspectorFields(definition blockstudio.Definition, options BlockFieldInspectorOptions) []InspectorField {
	blockKey := normalizeKey(definition.Key)
	if blockKey == "" {
		return nil
	}
	containerAttrs := append([]FieldAttribute{}, options.ContainerAttrs...)
	if len(containerAttrs) == 0 {
		containerAttrs = SelectionScopeAttrs(blockKey)
	}
	out := make([]InspectorField, 0, len(definition.Fields))
	for _, blockField := range definition.Fields {
		name := strings.TrimSpace(blockField.Name)
		if name == "" {
			continue
		}
		override := options.Overrides[name]
		field := InspectorField{
			Kind:           firstInspectorKind(override.Kind, inspectorKindForBlockField(blockField.Kind)),
			ID:             firstNonEmpty(override.ID, inspectorFieldID(firstNonEmpty(options.IDPrefix, blockKey), name)),
			Name:           firstNonEmpty(override.Name, inspectorFieldName(firstNonEmpty(options.NamePrefix, blockKey), name)),
			Label:          firstNonEmpty(override.Label, blockField.Label, name),
			Value:          firstNonEmpty(override.Value, options.Values[name], blockFieldDefaultValue(blockField)),
			Type:           firstNonEmpty(override.Type, inspectorInputTypeForBlockField(blockField.Kind)),
			Placeholder:    firstNonEmpty(override.Placeholder, blockField.Placeholder, blockField.UI.Placeholder),
			Required:       override.Required || blockField.Required,
			Checked:        override.Checked,
			Disabled:       override.Disabled,
			Rows:           override.Rows,
			Wide:           options.Wide || override.Wide,
			Help:           firstNonEmpty(override.Help, blockField.Help),
			Options:        append([]InspectorFieldOption{}, override.Options...),
			Attrs:          append([]FieldAttribute{}, override.Attrs...),
			ContainerAttrs: append([]FieldAttribute{}, firstFieldAttrs(override.ContainerAttrs, containerAttrs)...),
		}
		if field.Kind == InspectorFieldArea && field.Rows <= 0 {
			field.Rows = 4
		}
		if field.Kind == InspectorFieldSelect && len(field.Options) == 0 {
			field.Options = inspectorOptionsForBlockField(blockField, field.Value)
		}
		if field.Kind == InspectorFieldCheckbox && !field.Checked {
			field.Checked = checkedInspectorValue(field.Value)
		}
		source := firstNonEmpty(override.Source, inspectorFieldSource(firstNonEmpty(options.SourcePrefix, blockKey+"."), name))
		if source != "" && fieldAttributeValue(field.Attrs, "data-studio-field-source") == "" {
			field.Attrs = append(field.Attrs, FieldAttribute{Name: "data-studio-field-source", Value: source})
		}
		if blockField.Kind == blockstudio.FieldImage && fieldAttributeValue(field.Attrs, "data-studio-field-editable") == "" {
			field.Attrs = append(field.Attrs, FieldAttribute{Name: "data-studio-field-editable", Value: "media"})
		}
		if blockField.Kind == blockstudio.FieldImage {
			field.Attrs = appendMediaFieldAttrs(field.Attrs, options.MediaListID, options.MediaAltTargets[name])
		}
		if picker := strings.TrimSpace(blockField.UI.Picker); picker != "" && fieldAttributeValue(field.Attrs, "data-studio-field-picker") == "" {
			field.Attrs = append(field.Attrs, FieldAttribute{Name: "data-studio-field-picker", Value: picker})
		}
		out = append(out, field)
	}
	return out
}

func WorkbenchFieldInspectorFields(fields []workbench.Field, options WorkbenchFieldInspectorOptions) []InspectorField {
	containerAttrs := append([]FieldAttribute{}, options.ContainerAttrs...)
	out := make([]InspectorField, 0, len(fields))
	for _, workbenchField := range fields {
		name := strings.TrimSpace(workbenchField.Name)
		if name == "" {
			continue
		}
		override := options.Overrides[name]
		field := InspectorField{
			Kind:           firstInspectorKind(override.Kind, inspectorKindForWorkbenchField(workbenchField.Kind)),
			ID:             firstNonEmpty(override.ID, inspectorFieldID(options.IDPrefix, name)),
			Name:           firstNonEmpty(override.Name, inspectorFieldName(options.NamePrefix, name)),
			Label:          firstNonEmpty(override.Label, workbenchField.Label, name),
			Value:          firstNonEmpty(override.Value, options.Values[name]),
			Type:           firstNonEmpty(override.Type, inspectorInputTypeForWorkbenchField(workbenchField.Kind)),
			Placeholder:    override.Placeholder,
			Required:       override.Required || workbenchField.Required,
			Checked:        override.Checked,
			Disabled:       options.Disabled || override.Disabled || workbenchField.ReadOnly,
			Rows:           override.Rows,
			Wide:           options.Wide || override.Wide,
			Help:           override.Help,
			Options:        append([]InspectorFieldOption{}, override.Options...),
			Attrs:          append([]FieldAttribute{}, override.Attrs...),
			ContainerAttrs: append([]FieldAttribute{}, firstFieldAttrs(override.ContainerAttrs, containerAttrs)...),
		}
		if field.Kind == InspectorFieldArea && field.Rows <= 0 {
			field.Rows = 4
		}
		if field.Kind == InspectorFieldSelect && len(field.Options) == 0 {
			field.Options = inspectorOptionsForWorkbenchField(workbenchField, field.Value)
		}
		if field.Kind == InspectorFieldCheckbox && !field.Checked {
			field.Checked = checkedInspectorValue(field.Value)
		}
		source := firstNonEmpty(override.Source, inspectorFieldSource(options.SourcePrefix, name))
		if source != "" && fieldAttributeValue(field.Attrs, "data-studio-field-source") == "" {
			field.Attrs = append(field.Attrs, FieldAttribute{Name: "data-studio-field-source", Value: source})
		}
		if workbenchField.Kind == workbench.FieldImage && fieldAttributeValue(field.Attrs, "data-studio-field-editable") == "" {
			field.Attrs = append(field.Attrs, FieldAttribute{Name: "data-studio-field-editable", Value: "media"})
		}
		if workbenchField.Kind == workbench.FieldImage {
			field.Attrs = appendMediaFieldAttrs(field.Attrs, options.MediaListID, options.MediaAltTargets[name])
		}
		out = append(out, field)
	}
	return out
}

func LifecycleInspectorFields(options LifecycleInspectorOptions) []InspectorField {
	containerAttrs := append([]FieldAttribute{}, options.ContainerAttrs...)
	if len(containerAttrs) == 0 {
		containerAttrs = SelectionScopeAttrs("lifecycle")
	}
	draftState := string(options.DraftState)
	if draftState == "" {
		draftState = string(lifecycle.DraftStateDraft)
	}
	publishState := string(options.PublishState)
	if publishState == "" {
		publishState = string(lifecycle.PublishStateDraft)
	}
	fields := []InspectorField{
		{
			Kind:           InspectorFieldSelect,
			ID:             "lifecycleDraftState",
			Name:           "lifecycleDraftState",
			Label:          "Draft state",
			Value:          draftState,
			Disabled:       options.Disabled,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs:          []FieldAttribute{{Name: "data-studio-field-source", Value: "lifecycle.draftState"}},
			Options: []InspectorFieldOption{
				{Value: string(lifecycle.DraftStateDraft), Label: "Draft"},
				{Value: string(lifecycle.DraftStatePreview), Label: "Preview"},
				{Value: string(lifecycle.DraftStateRollback), Label: "Rolled back"},
			},
		},
		{
			Kind:           InspectorFieldSelect,
			ID:             "lifecyclePublishState",
			Name:           "lifecyclePublishState",
			Label:          "Publish state",
			Value:          publishState,
			Disabled:       options.Disabled,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs:          []FieldAttribute{{Name: "data-studio-field-source", Value: "lifecycle.publishState"}},
			Options: []InspectorFieldOption{
				{Value: string(lifecycle.PublishStateDraft), Label: "Draft"},
				{Value: string(lifecycle.PublishStatePublished), Label: "Published"},
			},
		},
	}
	actions := []FieldAction{}
	if href := strings.TrimSpace(options.PreviewHref); href != "" {
		actions = append(actions, FieldAction{Label: "Open preview", Href: href, Primary: true})
	}
	if action := strings.TrimSpace(options.PublishAction); action != "" {
		for index := range actions {
			actions[index].Primary = false
		}
		actions = append(actions, FieldAction{
			Label:      "Publish",
			FormAction: action,
			SubmitKey:  "publish",
			Confirm:    "Publish this draft?",
			Primary:    true,
		})
	}
	if href := strings.TrimSpace(options.PublishHref); href != "" {
		actions = append(actions, FieldAction{Label: "Review publish", Href: href})
	}
	if action := strings.TrimSpace(options.ScheduleAction); action != "" {
		actions = append(actions, FieldAction{
			Label:      "Schedule",
			FormAction: action,
			SubmitKey:  "schedule",
			Confirm:    "Schedule this draft?",
		})
	}
	if href := strings.TrimSpace(options.ScheduleHref); href != "" {
		actions = append(actions, FieldAction{Label: "Schedule", Href: href})
	}
	fields = append(fields, InspectorField{
		Kind:           InspectorFieldCard,
		Label:          "Publish controls",
		CardTitle:      "Save stays draft until an explicit publish action.",
		Wide:           true,
		ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
		Attrs: []FieldAttribute{
			{Name: "data-studio-field-source", Value: "lifecycle.publish"},
			{Name: "data-studio-field-editable", Value: "lifecycle"},
			{Name: "data-studio-field-action", Value: "Review publish"},
		},
		Actions: actions,
	})
	return fields
}

func FlowConfigInspectorFields(flow FlowCard, options FlowConfigInspectorOptions) []InspectorField {
	flow.Key = normalizeKey(firstNonEmpty(flow.Key, flow.Label))
	if flow.Key == "" {
		return nil
	}
	containerAttrs := append([]FieldAttribute{}, options.ContainerAttrs...)
	if len(containerAttrs) == 0 {
		containerAttrs = SelectionScopeAttrs(flow.Key)
	}
	namePrefix := firstNonEmpty(options.NamePrefix, "flow"+pascalKey(flow.Key))
	idPrefix := firstNonEmpty(options.IDPrefix, namePrefix)
	sourcePrefix := firstNonEmpty(options.SourcePrefix, "flow."+flow.Key+".")
	route := firstNonEmpty(options.Values["route"], flow.Route)
	embedTarget := firstNonEmpty(options.Values["embedTarget"], flow.EmbedTarget)
	handlerRef := firstNonEmpty(options.Values["handlerRef"], flow.PrimaryHandlerRef)
	fields := []InspectorField{
		{
			Kind:           InspectorFieldInput,
			ID:             inspectorFieldID(idPrefix, "route"),
			Name:           inspectorFieldName(namePrefix, "route"),
			Label:          "Public route",
			Value:          route,
			Disabled:       true,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs: []FieldAttribute{
				{Name: "data-studio-field-source", Value: inspectorFieldSource(sourcePrefix, "route")},
				{Name: "data-studio-field-editable", Value: "routing"},
			},
		},
		{
			Kind:           InspectorFieldInput,
			ID:             inspectorFieldID(idPrefix, "embedTarget"),
			Name:           inspectorFieldName(namePrefix, "embedTarget"),
			Label:          "Embed target",
			Value:          embedTarget,
			Disabled:       true,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs: []FieldAttribute{
				{Name: "data-studio-field-source", Value: inspectorFieldSource(sourcePrefix, "embedTarget")},
				{Name: "data-studio-field-editable", Value: "routing"},
			},
		},
		{
			Kind:           InspectorFieldInput,
			ID:             inspectorFieldID(idPrefix, "handlerRef"),
			Name:           inspectorFieldName(namePrefix, "handlerRef"),
			Label:          "Handler ref",
			Value:          handlerRef,
			Disabled:       options.Disabled,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs: []FieldAttribute{
				{Name: "data-studio-field-source", Value: inspectorFieldSource(sourcePrefix, "handlerRef")},
				{Name: "data-studio-field-editable", Value: "flow"},
			},
		},
	}
	actions := []FieldAction{}
	if href := strings.TrimSpace(options.ConfigureHref); href != "" {
		actions = append(actions, FieldAction{Label: "Open flow", Href: href})
	}
	if action := strings.TrimSpace(options.PublishAction); action != "" {
		actions = append(actions, FieldAction{
			Label:      "Publish flow",
			FormAction: action,
			SubmitKey:  "publish-flow",
			Name:       "flowKey",
			Value:      flow.Key,
			Confirm:    "Publish this flow?",
			Primary:    true,
		})
	}
	fields = append(fields, InspectorField{
		Kind:           InspectorFieldCard,
		Label:          "Flow publish",
		CardTitle:      firstNonEmpty(flow.StatusLabel, "Draft"),
		Help:           firstNonEmpty(options.PublishReview, flowPublishReview(flow)),
		Wide:           true,
		ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
		Attrs: []FieldAttribute{
			{Name: "data-studio-field-source", Value: inspectorFieldSource(sourcePrefix, "publish")},
			{Name: "data-studio-field-editable", Value: "flow"},
			{Name: "data-studio-field-action", Value: "Publish flow"},
		},
		Actions: actions,
	})
	return fields
}

func FlowStepInspectorFields(flow FlowCard, options FlowStepInspectorOptions) []InspectorField {
	flow.Key = normalizeKey(firstNonEmpty(flow.Key, flow.Label))
	if flow.Key == "" {
		return nil
	}
	containerAttrs := append([]FieldAttribute{}, options.ContainerAttrs...)
	if len(containerAttrs) == 0 {
		containerAttrs = SelectionScopeAttrs(flow.Key)
	}
	namePrefix := firstNonEmpty(options.NamePrefix, "flow"+pascalKey(flow.Key)+"Step")
	idPrefix := firstNonEmpty(options.IDPrefix, namePrefix)
	sourcePrefix := firstNonEmpty(options.SourcePrefix, "flow."+flow.Key+".steps.")
	fields := []InspectorField{}
	for index, step := range flow.Steps {
		step.Key = normalizeKey(firstNonEmpty(step.Key, step.Label, fmtAny(index+1)))
		if step.Key == "" {
			continue
		}
		step.Label = firstNonEmpty(step.Label, step.Key)
		stepPrefix := pascalKey(step.Key)
		stepNamePrefix := namePrefix + stepPrefix
		stepIDPrefix := idPrefix + stepPrefix
		labelValue := firstNonEmpty(options.Values[step.Key+".label"], options.Values[step.Key], step.Label)
		labelSource := inspectorFieldSource(sourcePrefix+step.Key+".", "label")
		bodySource := inspectorFieldSource(sourcePrefix+step.Key+".", "body")
		fields = append(fields, InspectorField{
			Kind:           InspectorFieldInput,
			ID:             inspectorFieldID(stepIDPrefix, "label"),
			Name:           inspectorFieldName(stepNamePrefix, "label"),
			Label:          step.Label + " step label",
			Value:          labelValue,
			Disabled:       options.Disabled,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs: []FieldAttribute{
				{Name: "data-studio-field-source", Value: labelSource},
				{Name: "data-studio-field-editable", Value: "flow"},
			},
			Help: "Shown in flow progress, embedded forms, and publish review.",
		})
		bodyTitle := "No body blocks"
		if step.BlockCount == 1 {
			bodyTitle = "1 body block"
		} else if step.BlockCount > 1 {
			bodyTitle = fmtAny(step.BlockCount) + " body blocks"
		}
		fields = append(fields, InspectorField{
			Kind:           InspectorFieldCard,
			Label:          step.Label + " body",
			CardTitle:      bodyTitle,
			Help:           "Step body is a block document that publishes with this flow.",
			Wide:           true,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs: []FieldAttribute{
				{Name: "data-studio-field-source", Value: bodySource},
				{Name: "data-studio-field-editable", Value: "flow"},
			},
		})
	}
	return fields
}

func flowPublishReview(flow FlowCard) string {
	parts := []string{}
	if summary := strings.TrimSpace(flow.Summary); summary != "" {
		parts = append(parts, summary)
	}
	if route := strings.TrimSpace(flow.Route); route != "" {
		parts = append(parts, "route "+route)
	}
	if target := strings.TrimSpace(flow.EmbedTarget); target != "" {
		parts = append(parts, "embed target "+target)
	}
	if handler := strings.TrimSpace(flow.PrimaryHandlerRef); handler != "" {
		parts = append(parts, "handler "+handler)
	}
	if len(parts) == 0 {
		return "Publishes the current flow draft."
	}
	return "Publishes current draft: " + strings.Join(parts, "; ") + "."
}

func CalendarWidgetInspectorFields(contract calendar.ScheduleWidgetContract, options CalendarWidgetInspectorOptions) []InspectorField {
	contract = calendar.NormalizeScheduleWidgetContract(contract)
	containerAttrs := append([]FieldAttribute{}, options.ContainerAttrs...)
	if len(containerAttrs) == 0 {
		containerAttrs = SelectionScopeAttrs(contract.Key)
	}
	namePrefix := firstNonEmpty(options.NamePrefix, contract.Key+"Widget")
	idPrefix := firstNonEmpty(options.IDPrefix, namePrefix)
	sourcePrefix := firstNonEmpty(options.SourcePrefix, "calendar."+contract.Key+".")
	fields := make([]InspectorField, 0, len(contract.Recipe.Controls)+2)
	for _, control := range contract.Recipe.Controls {
		key := strings.TrimSpace(control.Key)
		if key == "" {
			continue
		}
		value := firstNonEmpty(options.Values[key], control.Default)
		field := InspectorField{
			Kind:           InspectorFieldSelect,
			ID:             inspectorFieldID(idPrefix, key),
			Name:           inspectorFieldName(namePrefix, key),
			Label:          firstNonEmpty(control.Label, labelize(key)),
			Value:          value,
			Required:       control.Required,
			Disabled:       options.Disabled,
			ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
			Attrs:          []FieldAttribute{{Name: "data-studio-field-source", Value: inspectorFieldSource(sourcePrefix, key)}},
			Options:        calendarControlOptions(control.Options, value),
		}
		fields = append(fields, field)
	}
	dataSummary := pluralize(len(contract.Data), "described schedule field", "described schedule fields")
	actionSummary := pluralize(len(contract.Actions), "registered schedule action", "registered schedule actions")
	fields = append(fields, InspectorField{
		Kind:           InspectorFieldCard,
		Label:          "Schedule data",
		CardTitle:      dataSummary,
		Wide:           true,
		ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
		Attrs: []FieldAttribute{
			{Name: "data-studio-field-source", Value: sourcePrefix + "data"},
			{Name: "data-studio-field-editable", Value: "schema"},
			{Name: "data-studio-field-action", Value: "Review fields"},
		},
	})
	fields = append(fields, InspectorField{
		Kind:           InspectorFieldCard,
		Label:          "Schedule actions",
		CardTitle:      actionSummary,
		Wide:           true,
		ContainerAttrs: append([]FieldAttribute{}, containerAttrs...),
		Attrs: []FieldAttribute{
			{Name: "data-studio-field-source", Value: sourcePrefix + "actions"},
			{Name: "data-studio-field-editable", Value: "actions"},
			{Name: "data-studio-field-action", Value: "Open schedule"},
		},
		Actions: calendarFieldActions(contract),
	})
	return fields
}

func BlockInspectorField(definition blockstudio.Definition, options BlockInspectorOptions) InspectorField {
	field := firstNonEmpty(options.Field, definition.Key+".collection")
	editable := firstNonEmpty(options.Editable, "source")
	attrs := append([]FieldAttribute{}, options.Attrs...)
	if fieldAttributeValue(attrs, "data-studio-field-source") == "" {
		attrs = append(attrs, FieldAttribute{Name: "data-studio-field-source", Value: field})
	}
	if fieldAttributeValue(attrs, "data-studio-field-editable") == "" {
		attrs = append(attrs, FieldAttribute{Name: "data-studio-field-editable", Value: editable})
	}
	if fieldAttributeValue(attrs, "data-studio-field-action") == "" {
		if actionLabel := firstNonEmpty(options.ActionLabel, primaryFieldAction(options.Actions).Label); actionLabel != "" {
			attrs = append(attrs, FieldAttribute{Name: "data-studio-field-action", Value: actionLabel})
		}
	}
	containerAttrs := append([]FieldAttribute{}, options.ContainerAttrs...)
	if len(containerAttrs) == 0 {
		containerAttrs = SelectionScopeAttrs(definition.Key)
	}
	return InspectorField{
		Kind:           InspectorFieldCard,
		Label:          firstNonEmpty(options.Label, definition.Label),
		CardTitle:      firstNonEmpty(options.CardTitle, definition.Summary),
		Wide:           !options.Narrow,
		Class:          options.Class,
		Attrs:          attrs,
		ContainerAttrs: containerAttrs,
		Actions:        append([]FieldAction{}, options.Actions...),
	}
}

type InspectorPanelOptions struct {
	Class        string
	PanelClass   string
	PanelKey     string
	Mode         string
	HeadingClass string
	Kicker       string
	Title        string
	DynamicTitle bool
}

type RevisionChangeItem struct {
	KindLabel string
	Path      string
}

type RevisionItem struct {
	ID             string
	Title          string
	ActionLabel    string
	Summary        string
	HasSummary     bool
	ChangeSummary  string
	HasDiff        bool
	CreatedLabel   string
	CreatedMachine string
	ChangeItems    []RevisionChangeItem
}

type RevisionHistoryOptions struct {
	Class         string
	PanelClass    string
	PanelKey      string
	HeaderClass   string
	Title         string
	EmptyText     string
	RestoreAction string
	CSRFToken     string
	Confirm       string
	ButtonLabel   string
}

func RenderPanelHeading(options PanelHeadingOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "studio-panel-heading")
	kicker := strings.TrimSpace(options.Kicker)
	title := strings.TrimSpace(options.Title)
	nodes := []gosx.Node{}
	if kicker != "" {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(kicker)))
	}
	if title != "" {
		nodes = append(nodes, gosx.El("h2", nil, gosx.Text(title)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(nodes...))
}

func RenderSiteNavigator(items []SiteNavItem, options SiteNavigatorOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "studio-nav-panel")
	navClass := firstNonEmpty(options.NavClass, "studio-page-list")
	label := firstNonEmpty(options.Label, "Site pages")
	mode := firstNonEmpty(options.Mode, "structure")
	links := []gosx.Node{}
	for _, item := range normalizeSiteNavItems(items) {
		classAttr := strings.TrimSpace(item.Class)
		if item.Active && !strings.Contains(" "+classAttr+" ", " is-active ") {
			classAttr = strings.TrimSpace(classAttr + " is-active")
		}
		attrs := []any{
			gosx.Attr("href", item.Href),
			gosx.Attr("data-gosx-link", "true"),
			gosx.Attr("data-studio-site-page", item.Key),
		}
		if classAttr != "" {
			attrs = append(attrs, gosx.Attr("class", classAttr))
		}
		children := []gosx.Node{gosx.El("span", nil, gosx.Text(item.Label))}
		if item.Summary != "" {
			children = append(children, gosx.El("small", nil, gosx.Text(item.Summary)))
		}
		links = append(links, gosx.El("a", gosx.Attrs(attrs...), gosx.Fragment(children...)))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-studio-mode-panel", mode),
	),
		RenderPanelHeading(PanelHeadingOptions{
			Class:  options.HeadingClass,
			Kicker: firstNonEmpty(options.Kicker, "Site"),
			Title:  firstNonEmpty(options.Title, "Pages"),
		}),
		gosx.El("nav", gosx.Attrs(
			gosx.Attr("class", navClass),
			gosx.Attr("aria-label", label),
		), gosx.Fragment(links...)),
	)
}

func RenderInspectorHeader(options InspectorHeaderOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "studio-inspector-head")
	kicker := firstNonEmpty(options.Kicker, "Properties")
	modeLabel := firstNonEmpty(options.ModeLabel, "Structure")
	selection := firstNonEmpty(options.SelectionLabel, "No selection")
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", className)),
		gosx.El("div", nil,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(kicker)),
			gosx.El("strong", gosx.Attrs(gosx.Attr("data-studio-mode-label", "true")), gosx.Text(modeLabel)),
		),
		gosx.El("output", gosx.Attrs(
			gosx.Attr("data-studio-selection-label", "true"),
			gosx.Attr("aria-live", "polite"),
		), gosx.Text(selection)),
	)
}

func RenderScopeStrip(options ScopeStripOptions) gosx.Node {
	className := firstNonEmpty(options.Class, "studio-scope-strip")
	label := firstNonEmpty(options.Label, "Inspector scope")
	crumbs := options.Crumbs
	if len(crumbs) == 0 {
		crumbs = []ScopeCrumb{
			{Label: "Site"},
			{Label: "Home"},
			{Label: "Structure", DynamicMode: true},
			{Label: "No selection", DynamicSelection: true},
		}
	}
	nodes := make([]gosx.Node, 0, len(crumbs))
	for _, crumb := range crumbs {
		text := firstNonEmpty(crumb.Label, "Item")
		if crumb.DynamicSelection {
			nodes = append(nodes, gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(text)))
			continue
		}
		attrs := gosx.Attrs()
		if crumb.DynamicMode {
			attrs = gosx.Attrs(gosx.Attr("data-studio-mode-label", "true"))
		}
		nodes = append(nodes, gosx.El("span", attrs, gosx.Text(text)))
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("aria-label", label),
	), gosx.Fragment(nodes...))
}

func RenderLayerList(layers []LayerItem, options LayerListOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "studio-nav-panel studio-nav-panel--layers")
	listClass := firstNonEmpty(options.Class, "home-section-list editor-block-list")
	mode := firstNonEmpty(options.Mode, "structure")
	blockStudioKey := firstNonEmpty(options.BlockStudioKey, "homepage")
	layerNodes := make([]gosx.Node, 0, len(layers))
	for _, layer := range normalizeLayerItems(layers) {
		layerNodes = append(layerNodes, renderLayerItem(layer))
	}
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", panelClass),
		gosx.Attr("data-studio-mode-panel", mode),
	),
		RenderPanelHeading(PanelHeadingOptions{
			Class:  options.HeadingClass,
			Kicker: firstNonEmpty(options.Kicker, "Page"),
			Title:  firstNonEmpty(options.Title, "Layers"),
		}),
		gosx.El("div", gosx.Attrs(
			gosx.Attr("class", listClass),
			gosx.Attr("data-block-studio", blockStudioKey),
		), gosx.Fragment(layerNodes...)),
	)
}

func RenderBlockLibrary(items []BlockLibraryItem, options BlockLibraryOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "editor-panel editor-panel--library")
	panelKey := firstNonEmpty(options.PanelKey, "blocks")
	mode := firstNonEmpty(options.Mode, "structure")
	className := firstNonEmpty(options.Class, "editor-block-library")
	title := firstNonEmpty(options.Title, "Blocks")
	buttons := make([]gosx.Node, 0, len(items))
	for _, item := range normalizeBlockLibraryItems(items) {
		attrs := []any{
			gosx.Attr("class", item.ButtonClass),
			gosx.Attr("type", "button"),
			gosx.Attr("data-editor-add-block", item.Target),
		}
		if item.ButtonBaseClass != "" {
			attrs = append(attrs, gosx.Attr("data-editor-button-base", item.ButtonBaseClass))
		}
		if item.Active {
			attrs = append(attrs, gosx.Attr("aria-pressed", "true"))
		}
		buttons = append(buttons, gosx.El("button", gosx.Attrs(attrs...),
			gosx.El("span", nil, gosx.Text(item.Label)),
			gosx.El("small", nil, gosx.Text(item.ButtonLabel)),
		))
	}
	return gosx.El("section", panelAttrs(panelClass, panelKey, mode),
		gosx.El("h2", nil, gosx.Text(title)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(buttons...)),
	)
}

func RenderLinkGridPanel(links []PanelLink, options LinkGridOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "editor-panel")
	panelKey := firstNonEmpty(options.PanelKey, "links")
	mode := firstNonEmpty(options.Mode, "structure")
	gridClass := firstNonEmpty(options.GridClass, "studio-commerce-grid")
	linkNodes := make([]gosx.Node, 0, len(links))
	for _, link := range normalizePanelLinks(links) {
		attrs := []any{
			gosx.Attr("href", link.Href),
			gosx.Attr("data-gosx-link", "true"),
			gosx.Attr("data-studio-panel-link", link.Key),
		}
		if link.Class != "" {
			attrs = append(attrs, gosx.Attr("class", link.Class))
		}
		linkNodes = append(linkNodes, gosx.El("a", gosx.Attrs(attrs...),
			gosx.El("strong", nil, gosx.Text(link.Label)),
			gosx.El("span", nil, gosx.Text(link.Summary)),
		))
	}
	return gosx.El("section", panelAttrs(panelClass, panelKey, mode),
		RenderPanelHeading(PanelHeadingOptions{Kicker: options.Kicker, Title: options.Title}),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", gridClass)), gosx.Fragment(linkNodes...)),
	)
}

func RenderFlowLibrary(flows []FlowCard, options FlowLibraryOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "editor-panel editor-panel--flows")
	panelKey := firstNonEmpty(options.PanelKey, "flows")
	mode := firstNonEmpty(options.Mode, "flows")
	className := firstNonEmpty(options.Class, "studio-flow-list")
	flowNodes := make([]gosx.Node, 0, len(flows))
	for _, flow := range normalizeFlowCards(flows) {
		flowNodes = append(flowNodes, renderFlowCard(flow))
	}
	return gosx.El("section", panelAttrs(panelClass, panelKey, mode),
		RenderPanelHeading(PanelHeadingOptions{
			Kicker: firstNonEmpty(options.Kicker, "Behavior"),
			Title:  firstNonEmpty(options.Title, "Flows"),
		}),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", className)), gosx.Fragment(flowNodes...)),
	)
}

func RenderInspectorPanel(fields []InspectorField, options InspectorPanelOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "editor-panel")
	panelKey := firstNonEmpty(options.PanelKey, "inspector")
	mode := firstNonEmpty(options.Mode, "content")
	fieldNodes := make([]gosx.Node, 0, len(fields))
	for _, field := range normalizeInspectorFields(fields) {
		fieldNodes = append(fieldNodes, renderInspectorField(field))
	}
	heading := RenderPanelHeading(PanelHeadingOptions{
		Class:  options.HeadingClass,
		Kicker: firstNonEmpty(options.Kicker, "Content"),
		Title:  firstNonEmpty(options.Title, "Selection"),
	})
	if options.DynamicTitle {
		heading = gosx.El("div", gosx.Attrs(gosx.Attr("class", firstNonEmpty(options.HeadingClass, "studio-panel-heading"))),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(firstNonEmpty(options.Kicker, "Content"))),
			gosx.El("h2", gosx.Attrs(gosx.Attr("data-studio-selection-label", "true")), gosx.Text(firstNonEmpty(options.Title, "Selection"))),
		)
	}
	return gosx.El("section", panelAttrs(panelClass, panelKey, mode), heading, gosx.Fragment(fieldNodes...))
}

func RenderRevisionHistory(revisions []RevisionItem, options RevisionHistoryOptions) gosx.Node {
	panelClass := firstNonEmpty(options.PanelClass, "panel")
	panelKey := firstNonEmpty(options.PanelKey, "versions")
	headerClass := firstNonEmpty(options.HeaderClass, "panel__header")
	title := firstNonEmpty(options.Title, "Version history")
	emptyText := firstNonEmpty(options.EmptyText, "No previous versions yet.")
	nodes := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", headerClass)),
			gosx.El("h2", nil, gosx.Text(title)),
		),
	}
	revisions = normalizeRevisionItems(revisions)
	if len(revisions) == 0 {
		nodes = append(nodes, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(emptyText)))
		return gosx.El("section", gosx.Attrs(
			gosx.Attr("class", panelClass),
			gosx.Attr("data-panel-key", panelKey),
		), gosx.Fragment(nodes...))
	}
	items := make([]gosx.Node, 0, len(revisions))
	for _, revision := range revisions {
		items = append(items, renderRevisionItem(revision, options))
	}
	nodes = append(nodes, gosx.El("ul", gosx.Attrs(gosx.Attr("class", "field-list field-list--stacked")), gosx.Fragment(items...)))
	return gosx.El("section", gosx.Attrs(
		gosx.Attr("class", panelClass),
		gosx.Attr("data-panel-key", panelKey),
	), gosx.Fragment(nodes...))
}

func renderLayerItem(layer LayerItem) gosx.Node {
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", layer.CardClass),
		gosx.Attr("draggable", "true"),
		gosx.Attr("tabindex", "-1"),
		gosx.Attr("data-block-studio-block", layer.Key),
		gosx.Attr("data-studio-block-label", layer.Label),
	),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", layer.KeyName),
			gosx.Attr("value", layer.Key),
		)),
		gosx.El("input", gosx.Attrs(
			gosx.Attr("class", "home-section-order"),
			gosx.Attr("name", layer.OrderName),
			gosx.Attr("value", layer.Order),
			gosx.Attr("type", "hidden"),
			gosx.Attr("data-block-studio-order", "true"),
		)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__chrome")),
			gosx.El("button", gosx.Attrs(
				gosx.Attr("class", "home-section-handle"),
				gosx.Attr("type", "button"),
				gosx.Attr("aria-label", layer.DragLabel),
				gosx.Attr("data-block-studio-handle", "true"),
			), gosx.Text("Drag")),
			gosx.El("span", gosx.Attrs(
				gosx.Attr("class", layer.StatusClass),
				gosx.Attr("data-editor-block-pill", "true"),
			), gosx.Text(layer.StatusLabel)),
		),
		renderLayerPreview(layer),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__actions")),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "editor-visibility")),
				gosx.El("input", layerVisibilityAttrs(layer.EnabledName, layer.Enabled)),
				gosx.Text(" "),
				gosx.El("span", gosx.Attrs(gosx.Attr("data-editor-block-status", "true")), gosx.Text(layer.StatusLabel)),
			),
			gosx.El("input", gosx.Attrs(
				gosx.Attr("type", "hidden"),
				gosx.Attr("name", layer.EnabledName),
				gosx.Attr("value", "off"),
			)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-order-actions")),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "home-section-move"),
					gosx.Attr("type", "button"),
					gosx.Attr("data-block-studio-move", "up"),
					gosx.Attr("aria-label", layer.MoveUpLabel),
				), gosx.Text("Move up")),
				gosx.El("button", gosx.Attrs(
					gosx.Attr("class", "home-section-move"),
					gosx.Attr("type", "button"),
					gosx.Attr("data-block-studio-move", "down"),
					gosx.Attr("aria-label", layer.MoveDownLabel),
				), gosx.Text("Move down")),
			),
		),
	)
}

func renderFlowCard(flow FlowCard) gosx.Node {
	stepNodes := make([]gosx.Node, 0, len(flow.Steps))
	for _, step := range flow.Steps {
		stepNodes = append(stepNodes, gosx.El("li", nil, gosx.Text(step.Label)))
	}
	fieldNodes := []gosx.Node{}
	for _, action := range flow.Actions {
		for _, field := range action.Fields {
			fieldNodes = append(fieldNodes, gosx.El("span", gosx.Attrs(gosx.Attr("data-flow-field", field.Name)),
				gosx.Text(field.Label),
				gosx.El("small", nil, gosx.Text(field.RequiredLabel)),
			))
		}
	}
	metaNodes := []gosx.Node{
		gosx.El("span", nil, gosx.Text(flow.Summary)),
		gosx.El("span", nil, gosx.Text(fmtAny(flow.RequiredFieldCount)+" required")),
	}
	if flow.HasPrimaryAction {
		metaNodes = append(metaNodes, gosx.El("span", nil, gosx.Text(flow.PrimaryHandlerRef)))
	}
	actionNodes := []gosx.Node{}
	if flow.HasRoute {
		actionNodes = append(actionNodes, gosx.El("a", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("href", flow.Route),
			gosx.Attr("data-gosx-link", "true"),
		), gosx.Text("Open")))
	}
	if flow.HasEmbedTarget {
		actionNodes = append(actionNodes, gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button studio-flow-card__embed button--secondary"),
			gosx.Attr("type", "button"),
			gosx.Attr("data-editor-add-block", flow.EmbedTarget),
			gosx.Attr("data-editor-button-base", "button studio-flow-card__embed"),
		), gosx.Text("Add to page")))
	}
	return gosx.El("article", gosx.Attrs(
		gosx.Attr("class", flow.CardClass),
		gosx.Attr("data-editor-flow", flow.Key),
		gosx.Attr("data-studio-flow-executable", boolAttr(flow.CanExecute)),
	),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-card__head")),
			gosx.El("div", nil,
				gosx.El("strong", nil, gosx.Text(flow.Label)),
				gosx.El("span", nil, gosx.Text(flow.Description)),
			),
			gosx.El("output", gosx.Attrs(gosx.Attr("class", flow.StatusClass)), gosx.Text(flow.StatusLabel)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-card__meta")), gosx.Fragment(metaNodes...)),
		gosx.El("ol", gosx.Attrs(gosx.Attr("class", "studio-flow-steps")), gosx.Fragment(stepNodes...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-flow-fields")), gosx.Fragment(fieldNodes...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(actionNodes...)),
	)
}

func renderInspectorField(field InspectorField) gosx.Node {
	attrs := []any{gosx.Attr("class", inspectorFieldClass(field))}
	for _, attr := range field.ContainerAttrs {
		if strings.TrimSpace(attr.Name) == "" {
			continue
		}
		if attr.Bool {
			attrs = append(attrs, gosx.BoolAttr(strings.TrimSpace(attr.Name)))
			continue
		}
		attrs = append(attrs, gosx.Attr(strings.TrimSpace(attr.Name), attr.Value))
	}
	if source := fieldAttributeValue(field.Attrs, "data-studio-field-source"); source != "" {
		attrs = append(attrs, gosx.Attr("data-studio-field-source", source))
	}
	if field.Kind == InspectorFieldCard {
		primaryAction := primaryFieldAction(field.Actions)
		if primaryAction.Href != "" && fieldAttributeValue(field.Attrs, "data-studio-field-action-href") == "" {
			attrs = append(attrs, gosx.Attr("data-studio-field-action-href", primaryAction.Href))
		}
		if primaryAction.FormAction != "" && fieldAttributeValue(field.Attrs, "data-studio-field-action-formaction") == "" {
			attrs = append(attrs, gosx.Attr("data-studio-field-action-formaction", primaryAction.FormAction))
		}
		if primaryAction.Label != "" && fieldAttributeValue(field.Attrs, "data-studio-field-action") == "" {
			attrs = append(attrs, gosx.Attr("data-studio-field-action", primaryAction.Label))
		}
		for _, attr := range field.Attrs {
			name := strings.TrimSpace(attr.Name)
			if name == "" || name == "data-studio-field-source" {
				continue
			}
			if attr.Bool {
				attrs = append(attrs, gosx.BoolAttr(name))
				continue
			}
			attrs = append(attrs, gosx.Attr(name, attr.Value))
		}
		attrs = append(attrs, gosx.Attr("tabindex", "-1"))
		actionNodes := make([]gosx.Node, 0, len(field.Actions))
		for _, action := range field.Actions {
			actionNodes = append(actionNodes, renderFieldAction(action))
		}
		cardNodes := []gosx.Node{gosx.El("strong", nil, gosx.Text(field.CardTitle))}
		if strings.TrimSpace(field.Help) != "" {
			cardNodes = append(cardNodes, gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(field.Help)))
		}
		cardNodes = append(cardNodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(actionNodes...)))
		return gosx.El("div", gosx.Attrs(attrs...),
			gosx.El("label", nil, gosx.Text(field.Label)),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-source-card")), gosx.Fragment(cardNodes...)),
		)
	}
	controlAttrs := inspectorControlAttrs(field)
	control := gosx.El("input", controlAttrs)
	switch field.Kind {
	case InspectorFieldArea:
		control = gosx.El("textarea", controlAttrs, gosx.Text(field.Value))
	case InspectorFieldSelect:
		optionNodes := make([]gosx.Node, 0, len(field.Options))
		for _, option := range field.Options {
			attrs := []any{gosx.Attr("value", option.Value)}
			if option.Selected || option.Value == field.Value {
				attrs = append(attrs, gosx.BoolAttr("selected"))
			}
			optionNodes = append(optionNodes, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(firstNonEmpty(option.Label, option.Value))))
		}
		control = gosx.El("select", controlAttrs, gosx.Fragment(optionNodes...))
	}
	children := []gosx.Node{
		gosx.El("label", gosx.Attrs(gosx.Attr("for", field.ID)), gosx.Text(field.Label)),
		control,
	}
	if strings.TrimSpace(field.Help) != "" {
		children = append(children, gosx.El("small", gosx.Attrs(gosx.Attr("class", "field-help")), gosx.Text(field.Help)))
	}
	return gosx.El("div", gosx.Attrs(attrs...), gosx.Fragment(children...))
}

func primaryFieldAction(actions []FieldAction) FieldAction {
	if len(actions) == 0 {
		return FieldAction{}
	}
	for _, action := range actions {
		if action.Primary {
			return action
		}
	}
	return actions[0]
}

func renderFieldAction(action FieldAction) gosx.Node {
	className := strings.TrimSpace(action.Class)
	if className == "" {
		className = "button button--secondary"
		if action.Primary {
			className = "button button--primary"
		}
	}
	if strings.TrimSpace(action.FormAction) != "" {
		attrs := []any{
			gosx.Attr("class", className),
			gosx.Attr("type", "submit"),
			gosx.Attr("formaction", action.FormAction),
			// This button submits a shared, ancestor-nested <form> (a card
			// field action has no form= attribute of its own — it relies on
			// DOM nesting inside whatever <form> the caller wraps the field
			// list in). If that shared form has ANY invalid control anywhere
			// in it — including one hidden by mode-gating CSS elsewhere in
			// the panel stack — native HTML5 validation silently cancels
			// this submission before the browser ever dispatches "submit",
			// regardless of which button was clicked. formnovalidate exempts
			// this action from that shared-form validation surface, mirroring
			// every workbench-form action button in the panels package
			// (Publish/Discard/Schedule, Restore, Save navigation, Save
			// checkout, Publish flow — see panels/*_panel.go) and
			// panels/inspector_fields.go's Home field-row actions.
			gosx.Attr("formnovalidate", "formnovalidate"),
			gosx.Attr("data-studio-field-action-formaction", action.FormAction),
		}
		if method := strings.TrimSpace(action.FormMethod); method != "" {
			attrs = append(attrs, gosx.Attr("formmethod", method))
		}
		if key := strings.TrimSpace(action.SubmitKey); key != "" {
			attrs = append(attrs, gosx.Attr("data-studio-submit-action", key))
		}
		if name := strings.TrimSpace(action.Name); name != "" {
			attrs = append(attrs, gosx.Attr("name", name))
			attrs = append(attrs, gosx.Attr("value", action.Value))
		}
		if confirm := strings.TrimSpace(action.Confirm); confirm != "" {
			attrs = append(attrs, gosx.Attr("data-admin-confirm", confirm))
		}
		return gosx.El("button", gosx.Attrs(attrs...), gosx.Text(action.Label))
	}
	return gosx.El("a", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("href", action.Href),
		gosx.Attr("data-gosx-link", "true"),
	), gosx.Text(action.Label))
}

func renderRevisionItem(revision RevisionItem, options RevisionHistoryOptions) gosx.Node {
	nodes := []gosx.Node{
		gosx.El("strong", nil, gosx.Text(revision.Title)),
		gosx.El("span", nil, gosx.Text(revision.ActionLabel)),
		gosx.El("time", gosx.Attrs(
			gosx.Attr("datetime", revision.CreatedMachine),
			gosx.Attr("data-viewer-time", "datetime"),
		), gosx.Text(revision.CreatedLabel)),
	}
	if revision.HasSummary {
		nodes = append(nodes, gosx.El("p", nil, gosx.Text(revision.Summary)))
	}
	if revision.HasDiff {
		changeNodes := make([]gosx.Node, 0, len(revision.ChangeItems))
		for _, change := range revision.ChangeItems {
			changeNodes = append(changeNodes, gosx.El("li", nil,
				gosx.El("span", nil, gosx.Text(change.KindLabel)),
				gosx.El("code", nil, gosx.Text(change.Path)),
			))
		}
		nodes = append(nodes,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "revision-diff-summary")), gosx.Text(revision.ChangeSummary)),
			gosx.El("ul", gosx.Attrs(gosx.Attr("class", "revision-diff-list")), gosx.Fragment(changeNodes...)),
		)
	}
	formNodes := []gosx.Node{}
	if strings.TrimSpace(options.CSRFToken) != "" {
		formNodes = append(formNodes, gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", "csrf_token"),
			gosx.Attr("value", options.CSRFToken),
		)))
	}
	formNodes = append(formNodes,
		gosx.El("input", gosx.Attrs(
			gosx.Attr("type", "hidden"),
			gosx.Attr("name", "revisionId"),
			gosx.Attr("value", revision.ID),
		)),
		gosx.El("button", gosx.Attrs(
			gosx.Attr("class", "button button--secondary"),
			gosx.Attr("type", "submit"),
			gosx.Attr("data-admin-confirm", firstNonEmpty(options.Confirm, "Restore these editor settings? Current look and feel will be saved in history first.")),
		), gosx.Text(firstNonEmpty(options.ButtonLabel, "Restore this version"))),
	)
	nodes = append(nodes, gosx.El("form", gosx.Attrs(
		gosx.Attr("class", "inline-form"),
		gosx.Attr("method", "post"),
		gosx.Attr("action", options.RestoreAction),
	), gosx.Fragment(formNodes...)))
	return gosx.El("li", nil, gosx.Fragment(nodes...))
}

func renderLayerPreview(layer LayerItem) gosx.Node {
	actionNodes := []gosx.Node{}
	for _, action := range layer.Preview.Actions {
		if strings.TrimSpace(action.Label) == "" {
			continue
		}
		actionNodes = append(actionNodes, gosx.El("span", layerPreviewAttrs(firstNonEmpty(action.Class, "button button--secondary"), action.Source), gosx.Text(action.Label)))
	}
	nodes := []gosx.Node{
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(firstNonEmpty(layer.Preview.Kicker, layer.Label))),
		gosx.El("h2", layerPreviewAttrs("", layer.Preview.TitleSource), gosx.Text(layer.Preview.Title)),
		gosx.El("p", layerPreviewAttrs("", layer.Preview.BodySource), gosx.Text(layer.Preview.Body)),
	}
	if len(actionNodes) > 0 {
		nodes = append(nodes, gosx.El("div", gosx.Attrs(gosx.Attr("class", "button-row")), gosx.Fragment(actionNodes...)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__preview")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", layer.Preview.VisualClass)),
			gosx.El("span", nil),
			gosx.El("span", nil),
			gosx.El("span", nil),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "editor-block__copy")), gosx.Fragment(nodes...)),
	)
}

func layerPreviewAttrs(className, source string) gosx.AttrList {
	attrs := []any{}
	if strings.TrimSpace(className) != "" {
		attrs = append(attrs, gosx.Attr("class", strings.TrimSpace(className)))
	}
	if strings.TrimSpace(source) != "" {
		attrs = append(attrs, gosx.Attr("data-editor-preview", strings.TrimSpace(source)))
	}
	return gosx.Attrs(attrs...)
}

func layerVisibilityAttrs(name string, enabled bool) gosx.AttrList {
	attrs := []any{
		gosx.Attr("type", "checkbox"),
		gosx.Attr("name", name),
		gosx.Attr("data-editor-block-visible", "true"),
	}
	if enabled {
		attrs = append(attrs, gosx.BoolAttr("checked"))
	}
	return gosx.Attrs(attrs...)
}

func inspectorControlAttrs(field InspectorField) gosx.AttrList {
	attrs := []any{
		gosx.Attr("id", field.ID),
		gosx.Attr("name", field.Name),
	}
	switch field.Kind {
	case InspectorFieldInput:
		attrs = append(attrs, gosx.Attr("type", firstNonEmpty(field.Type, "text")))
		attrs = append(attrs, gosx.Attr("value", field.Value))
	case InspectorFieldArea:
		if field.Rows > 0 {
			attrs = append(attrs, gosx.Attr("rows", field.Rows))
		}
	case InspectorFieldCheckbox:
		attrs = append(attrs, gosx.Attr("type", "checkbox"))
		attrs = append(attrs, gosx.Attr("value", "on"))
		if field.Checked {
			attrs = append(attrs, gosx.BoolAttr("checked"))
		}
	}
	if field.Placeholder != "" {
		attrs = append(attrs, gosx.Attr("placeholder", field.Placeholder))
	}
	if field.Required {
		attrs = append(attrs, gosx.BoolAttr("required"))
	}
	if field.Disabled {
		attrs = append(attrs, gosx.BoolAttr("disabled"))
	}
	for _, attr := range field.Attrs {
		if strings.TrimSpace(attr.Name) == "" || attr.Name == "data-studio-field-source" {
			continue
		}
		if attr.Bool {
			attrs = append(attrs, gosx.BoolAttr(strings.TrimSpace(attr.Name)))
			continue
		}
		attrs = append(attrs, gosx.Attr(strings.TrimSpace(attr.Name), attr.Value))
	}
	return gosx.Attrs(attrs...)
}

func inspectorFieldClass(field InspectorField) string {
	className := strings.TrimSpace(field.Class)
	if className == "" {
		className = "field-row"
	}
	if field.Wide && !strings.Contains(" "+className+" ", " field-row--wide ") {
		className += " field-row--wide"
	}
	return strings.TrimSpace(className)
}

func firstInspectorKind(values ...InspectorFieldKind) InspectorFieldKind {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return InspectorFieldInput
}

func firstFieldAttrs(values ...[]FieldAttribute) []FieldAttribute {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func inspectorKindForBlockField(kind blockstudio.FieldKind) InspectorFieldKind {
	switch kind {
	case blockstudio.FieldTextarea:
		return InspectorFieldArea
	case blockstudio.FieldSelect:
		return InspectorFieldSelect
	case blockstudio.FieldBoolean:
		return InspectorFieldCheckbox
	default:
		return InspectorFieldInput
	}
}

func inspectorKindForWorkbenchField(kind workbench.FieldKind) InspectorFieldKind {
	switch kind {
	case workbench.FieldTextarea:
		return InspectorFieldArea
	case workbench.FieldSelect:
		return InspectorFieldSelect
	case workbench.FieldBoolean:
		return InspectorFieldCheckbox
	default:
		return InspectorFieldInput
	}
}

func inspectorInputTypeForBlockField(kind blockstudio.FieldKind) string {
	switch kind {
	// handoff-4 (save-vocabulary audit): FieldURL used to render
	// type="url", but this codebase's own link fields (e.g. cms/blocks
	// Hero's "ctaUrl") legitimately hold SITE-RELATIVE targets like "/shop"
	// alongside absolute externals -- and the HTML "valid URL" constraint
	// native <input type="url"> enforces rejects a bare relative path,
	// which live-proved a false "Needs attention" (the row's
	// :has(:invalid) CSS hook, muddy-noni-commerce's public/site.css) on a
	// perfectly valid, already-saved CTA URL. NEEDS ATTENTION is reserved
	// for real validation problems; a false native-constraint positive on
	// legitimate data isn't one, so FieldURL renders as plain text (no
	// native format constraint) while FieldImage -- always populated from
	// the media picker's own resolved asset URL, never free-typed -- keeps
	// the stricter type="url" hint.
	case blockstudio.FieldImage:
		return "url"
	default:
		return "text"
	}
}

func inspectorInputTypeForWorkbenchField(kind workbench.FieldKind) string {
	switch kind {
	case workbench.FieldImage:
		return "url"
	case workbench.FieldDateTime:
		return "datetime-local"
	case workbench.FieldMoney:
		return "number"
	default:
		return "text"
	}
}

func inspectorFieldID(prefix, name string) string {
	prefix = strings.TrimSpace(prefix)
	name = strings.TrimSpace(name)
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	return prefix + strings.ToUpper(name[:1]) + name[1:]
}

func inspectorFieldName(prefix, name string) string {
	return inspectorFieldID(prefix, name)
}

func inspectorFieldSource(prefix, name string) string {
	prefix = strings.TrimSpace(prefix)
	name = strings.TrimSpace(name)
	if prefix == "" {
		return name
	}
	return prefix + name
}

func blockFieldDefaultValue(field blockstudio.FieldDefinition) string {
	switch field.Kind {
	case blockstudio.FieldBoolean:
		if field.Default.Bool {
			return "on"
		}
	case blockstudio.FieldImage:
		if field.Default.Media != nil {
			return strings.TrimSpace(field.Default.Media.URL)
		}
	default:
		return strings.TrimSpace(field.Default.String)
	}
	return ""
}

func inspectorOptionsForBlockField(field blockstudio.FieldDefinition, value string) []InspectorFieldOption {
	options := make([]InspectorFieldOption, 0, len(field.Options))
	for _, option := range field.Options {
		optionValue := strings.TrimSpace(option.Value)
		if optionValue == "" {
			continue
		}
		options = append(options, InspectorFieldOption{
			Value:    optionValue,
			Label:    firstNonEmpty(option.Label, optionValue),
			Selected: optionValue == value,
		})
	}
	return options
}

func inspectorOptionsForWorkbenchField(field workbench.Field, value string) []InspectorFieldOption {
	options := make([]InspectorFieldOption, 0, len(field.Options))
	for _, option := range field.Options {
		optionValue := strings.TrimSpace(option)
		if optionValue == "" {
			continue
		}
		options = append(options, InspectorFieldOption{
			Value:    optionValue,
			Label:    labelize(optionValue),
			Selected: optionValue == value,
		})
	}
	return options
}

func calendarControlOptions(options []calendar.WidgetOption, value string) []InspectorFieldOption {
	out := make([]InspectorFieldOption, 0, len(options))
	for _, option := range options {
		optionValue := strings.TrimSpace(option.Value)
		if optionValue == "" {
			continue
		}
		out = append(out, InspectorFieldOption{
			Value:    optionValue,
			Label:    firstNonEmpty(option.Label, labelize(optionValue)),
			Selected: optionValue == value,
		})
	}
	return out
}

func calendarFieldActions(contract calendar.ScheduleWidgetContract) []FieldAction {
	actions := []FieldAction{}
	if href := strings.TrimSpace(contract.PublicURL); href != "" {
		actions = append(actions, FieldAction{Label: "Open schedule", Href: href, Primary: true})
	}
	if href := strings.TrimSpace(contract.AdminURL); href != "" {
		actions = append(actions, FieldAction{Label: "Manage schedule", Href: href})
	}
	return actions
}

func appendMediaFieldAttrs(attrs []FieldAttribute, mediaListID, altTarget string) []FieldAttribute {
	if mediaListID = strings.TrimSpace(mediaListID); mediaListID != "" && fieldAttributeValue(attrs, "list") == "" {
		attrs = append(attrs, FieldAttribute{Name: "list", Value: mediaListID})
	}
	if altTarget = strings.TrimSpace(altTarget); altTarget != "" && fieldAttributeValue(attrs, "data-media-alt-target") == "" {
		attrs = append(attrs, FieldAttribute{Name: "data-media-alt-target", Value: altTarget})
	}
	return attrs
}

func checkedInspectorValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "yes", "checked":
		return true
	default:
		return false
	}
}

func labelize(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "-", " "))
	value = strings.TrimSpace(strings.ReplaceAll(value, "_", " "))
	if value == "" {
		return ""
	}
	parts := strings.Fields(value)
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func pluralize(count int, singular, plural string) string {
	label := plural
	if count == 1 {
		label = singular
	}
	return fmtAny(count) + " " + label
}

func pascalKey(value string) string {
	value = normalizeKey(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, part := range strings.Split(value, "-") {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}

func fieldAttributeValue(attrs []FieldAttribute, name string) string {
	for _, attr := range attrs {
		if attr.Name == name {
			return strings.TrimSpace(attr.Value)
		}
	}
	return ""
}

func panelAttrs(className, panelKey, mode string) gosx.AttrList {
	return gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-panel-key", panelKey),
		gosx.Attr("data-studio-mode-panel", mode),
	)
}

func normalizeSiteNavItems(items []SiteNavItem) []SiteNavItem {
	out := make([]SiteNavItem, 0, len(items))
	for _, item := range items {
		item.Key = normalizeKey(firstNonEmpty(item.Key, item.Label))
		item.Label = strings.TrimSpace(item.Label)
		item.Href = strings.TrimSpace(item.Href)
		item.Summary = strings.TrimSpace(item.Summary)
		item.Class = strings.TrimSpace(item.Class)
		if item.Key == "" || item.Label == "" || item.Href == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizeBlockLibraryItems(items []BlockLibraryItem) []BlockLibraryItem {
	out := make([]BlockLibraryItem, 0, len(items))
	for _, item := range items {
		item.Key = normalizeKey(firstNonEmpty(item.Key, item.Label))
		item.Label = strings.TrimSpace(item.Label)
		item.Summary = strings.TrimSpace(item.Summary)
		item.Target = normalizeKey(firstNonEmpty(item.Target, item.Key))
		item.ButtonLabel = firstNonEmpty(item.ButtonLabel, "Add")
		item.ButtonClass = firstNonEmpty(item.ButtonClass, "button button--secondary")
		item.ButtonBaseClass = strings.TrimSpace(item.ButtonBaseClass)
		if item.Key == "" || item.Label == "" || item.Target == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizePanelLinks(links []PanelLink) []PanelLink {
	out := make([]PanelLink, 0, len(links))
	for _, link := range links {
		link.Key = normalizeKey(firstNonEmpty(link.Key, link.Label))
		link.Label = strings.TrimSpace(link.Label)
		link.Summary = strings.TrimSpace(link.Summary)
		link.Href = strings.TrimSpace(link.Href)
		link.Class = strings.TrimSpace(link.Class)
		if link.Key == "" || link.Label == "" || link.Href == "" {
			continue
		}
		out = append(out, link)
	}
	return out
}

func normalizeFlowCards(flows []FlowCard) []FlowCard {
	out := make([]FlowCard, 0, len(flows))
	for _, flow := range flows {
		flow.Key = normalizeKey(firstNonEmpty(flow.Key, flow.Label))
		flow.Label = strings.TrimSpace(flow.Label)
		if flow.Key == "" || flow.Label == "" {
			continue
		}
		flow.CardClass = firstNonEmpty(flow.CardClass, "studio-flow-card")
		flow.StatusClass = firstNonEmpty(flow.StatusClass, "status")
		flow.StatusLabel = firstNonEmpty(flow.StatusLabel, "Draft")
		flow.Summary = strings.TrimSpace(flow.Summary)
		flow.Route = strings.TrimSpace(flow.Route)
		flow.EmbedTarget = normalizeKey(flow.EmbedTarget)
		flow.PrimaryHandlerRef = strings.TrimSpace(flow.PrimaryHandlerRef)
		if !flow.CanExecute && flow.PrimaryHandlerRef != "" {
			flow.CanExecute = true
		}
		out = append(out, flow)
	}
	return out
}

func normalizeInspectorFields(fields []InspectorField) []InspectorField {
	out := make([]InspectorField, 0, len(fields))
	for _, field := range fields {
		field.ID = strings.TrimSpace(field.ID)
		field.Name = strings.TrimSpace(field.Name)
		field.Label = strings.TrimSpace(field.Label)
		if field.Kind == "" {
			field.Kind = InspectorFieldInput
		}
		if field.Kind != InspectorFieldCard && (field.ID == "" || field.Name == "") {
			continue
		}
		if field.Label == "" {
			continue
		}
		if field.Kind == InspectorFieldArea && field.Rows <= 0 {
			field.Rows = 4
		}
		if field.Kind == InspectorFieldCard {
			field.CardTitle = firstNonEmpty(field.CardTitle, field.Value)
		}
		out = append(out, field)
	}
	return out
}

func normalizeRevisionItems(revisions []RevisionItem) []RevisionItem {
	out := make([]RevisionItem, 0, len(revisions))
	for _, revision := range revisions {
		revision.ID = strings.TrimSpace(revision.ID)
		revision.Title = firstNonEmpty(revision.Title, "Previous version")
		revision.ActionLabel = firstNonEmpty(revision.ActionLabel, "saved")
		revision.CreatedLabel = strings.TrimSpace(revision.CreatedLabel)
		revision.CreatedMachine = strings.TrimSpace(revision.CreatedMachine)
		if revision.ID == "" {
			continue
		}
		out = append(out, revision)
	}
	return out
}

func normalizeLayerItems(layers []LayerItem) []LayerItem {
	out := make([]LayerItem, 0, len(layers))
	for _, layer := range layers {
		layer.Key = normalizeKey(layer.Key)
		layer.Label = strings.TrimSpace(layer.Label)
		if layer.Key == "" || layer.Label == "" {
			continue
		}
		layer.CardClass = firstNonEmpty(layer.CardClass, "home-section-row editor-block editor-block--"+layer.Key)
		layer.StatusClass = firstNonEmpty(layer.StatusClass, "status")
		layer.StatusLabel = firstNonEmpty(layer.StatusLabel, "Hidden")
		layer.DragLabel = firstNonEmpty(layer.DragLabel, "Reorder "+layer.Label)
		layer.MoveUpLabel = firstNonEmpty(layer.MoveUpLabel, "Move "+layer.Label+" up")
		layer.MoveDownLabel = firstNonEmpty(layer.MoveDownLabel, "Move "+layer.Label+" down")
		if layer.Order <= 0 {
			layer.Order = len(out) + 1
		}
		formIndex := strings.TrimSpace(fmtAny(layer.Order - 1))
		layer.KeyName = firstNonEmpty(layer.KeyName, "homeSectionKey"+formIndex)
		layer.OrderName = firstNonEmpty(layer.OrderName, "homeSectionOrder"+formIndex)
		layer.EnabledName = firstNonEmpty(layer.EnabledName, "homeSectionEnabled"+formIndex)
		layer.Preview.VisualClass = firstNonEmpty(layer.Preview.VisualClass, "editor-block__visual editor-block__visual--"+layer.Key)
		layer.Preview.Kicker = firstNonEmpty(layer.Preview.Kicker, layer.Label)
		layer.Preview.Title = firstNonEmpty(layer.Preview.Title, layer.Label)
		out = append(out, layer)
	}
	return out
}
