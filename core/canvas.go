package core

import "strings"

type CanvasWorkspace struct {
	RouteLabel   string
	PreviewURL   string
	PreviewShell CanvasPreviewShell
	Viewports    []CanvasViewport
	ZoomLevels   []CanvasZoomLevel
	Blocks       []CanvasBlock
	Actions      []CanvasAction
}

type CanvasViewport struct {
	Key    string
	Label  string
	Width  string
	Active bool
}

type CanvasZoomLevel struct {
	Key    string
	Label  string
	Scale  float64
	Active bool
}

type CanvasBlock struct {
	Key           string
	Label         string
	Summary       string
	GoSXComponent string
	Source        ComponentSource
	Binding       string
	Status        string
	Visible       bool
	Selected      bool
	Editable      bool
	Controls      []Control
}

type CanvasAction struct {
	Key     string
	Label   string
	Summary string
	Kind    CanvasActionKind
	Enabled bool
}

type CanvasPreviewShell struct {
	OverlayAttr         string
	DropLineAttr        string
	DockAttr            string
	DockTitleAttr       string
	FieldCountAttr      string
	ActionAttr          string
	OutlineTemplateAttr string
	OutlineLabelAttr    string
	FieldTemplateAttr   string
	FieldActionTextAttr string
	InlineEditClass     string
	StyleImpactNodeAttr string
	FrameAttr           string
	FrameURLAttr        string
	ViewportAttr        string
	ViewportCurrentAttr string
	ViewportWidthAttr   string
	SelectionLabelAttr  string
}

func DefaultCanvasActions() []CanvasAction {
	return []CanvasAction{
		{Key: string(CanvasActionReveal), Kind: CanvasActionReveal, Label: "Reveal", Summary: "Focus the selected block in the canvas.", Enabled: true},
		{Key: string(CanvasActionMoveUp), Kind: CanvasActionMoveUp, Label: "Move up", Summary: "Move the selected block earlier on the page.", Enabled: true},
		{Key: string(CanvasActionMoveDown), Kind: CanvasActionMoveDown, Label: "Move down", Summary: "Move the selected block later on the page.", Enabled: true},
		{Key: string(CanvasActionInlineText), Kind: CanvasActionInlineText, Label: "Edit text", Summary: "Edit the primary text field for the selection.", Enabled: true},
		{Key: string(CanvasActionContent), Kind: CanvasActionContent, Label: "Content", Summary: "Open content controls for the selected block.", Enabled: true},
		{Key: string(CanvasActionStyle), Kind: CanvasActionStyle, Label: "Style", Summary: "Open style controls for the selected block.", Enabled: true},
		{Key: string(CanvasActionToggleVisibility), Kind: CanvasActionToggleVisibility, Label: "Hide", Summary: "Toggle whether the selected block is visible.", Enabled: true},
	}
}

func DefaultCanvasPreviewShell() CanvasPreviewShell {
	return CanvasPreviewShell{
		OverlayAttr:         "data-studio-preview-overlay",
		DropLineAttr:        "data-studio-preview-drop-line",
		DockAttr:            "data-studio-preview-dock",
		DockTitleAttr:       "data-studio-preview-dock-title",
		FieldCountAttr:      "data-studio-preview-field-count",
		ActionAttr:          "data-studio-preview-action",
		OutlineTemplateAttr: "data-studio-preview-outline-template",
		OutlineLabelAttr:    "data-studio-preview-outline-label",
		FieldTemplateAttr:   "data-studio-preview-field-template",
		FieldActionTextAttr: "data-studio-preview-field-action-text",
		InlineEditClass:     "is-studio-inline-editing",
		StyleImpactNodeAttr: "data-studio-style-impact-node",
		FrameAttr:           "data-studio-preview-frame",
		FrameURLAttr:        "data-studio-preview-src",
		ViewportAttr:        "data-studio-preview-viewport",
		ViewportCurrentAttr: "data-studio-viewport-current",
		ViewportWidthAttr:   "data-studio-viewport-width",
		SelectionLabelAttr:  "data-studio-selection-label",
	}
}

func (canvas CanvasWorkspace) Normalize() CanvasWorkspace {
	canvas.RouteLabel = strings.TrimSpace(canvas.RouteLabel)
	canvas.PreviewURL = strings.TrimSpace(canvas.PreviewURL)
	canvas.PreviewShell = canvas.PreviewShell.Normalize()
	canvas.Viewports = normalizeCanvasViewports(canvas.Viewports)
	canvas.ZoomLevels = normalizeCanvasZoomLevels(canvas.ZoomLevels)
	canvas.Blocks = normalizeCanvasBlocks(canvas.Blocks)
	canvas.Actions = normalizeCanvasActions(canvas.Actions)
	return canvas
}

func (shell CanvasPreviewShell) Normalize() CanvasPreviewShell {
	defaults := DefaultCanvasPreviewShell()
	shell.OverlayAttr = FirstNonEmpty(strings.TrimSpace(shell.OverlayAttr), defaults.OverlayAttr)
	shell.DropLineAttr = FirstNonEmpty(strings.TrimSpace(shell.DropLineAttr), defaults.DropLineAttr)
	shell.DockAttr = FirstNonEmpty(strings.TrimSpace(shell.DockAttr), defaults.DockAttr)
	shell.DockTitleAttr = FirstNonEmpty(strings.TrimSpace(shell.DockTitleAttr), defaults.DockTitleAttr)
	shell.FieldCountAttr = FirstNonEmpty(strings.TrimSpace(shell.FieldCountAttr), defaults.FieldCountAttr)
	shell.ActionAttr = FirstNonEmpty(strings.TrimSpace(shell.ActionAttr), defaults.ActionAttr)
	shell.OutlineTemplateAttr = FirstNonEmpty(strings.TrimSpace(shell.OutlineTemplateAttr), defaults.OutlineTemplateAttr)
	shell.OutlineLabelAttr = FirstNonEmpty(strings.TrimSpace(shell.OutlineLabelAttr), defaults.OutlineLabelAttr)
	shell.FieldTemplateAttr = FirstNonEmpty(strings.TrimSpace(shell.FieldTemplateAttr), defaults.FieldTemplateAttr)
	shell.FieldActionTextAttr = FirstNonEmpty(strings.TrimSpace(shell.FieldActionTextAttr), defaults.FieldActionTextAttr)
	shell.InlineEditClass = FirstNonEmpty(strings.TrimSpace(shell.InlineEditClass), defaults.InlineEditClass)
	shell.StyleImpactNodeAttr = FirstNonEmpty(strings.TrimSpace(shell.StyleImpactNodeAttr), defaults.StyleImpactNodeAttr)
	shell.FrameAttr = FirstNonEmpty(strings.TrimSpace(shell.FrameAttr), defaults.FrameAttr)
	shell.FrameURLAttr = FirstNonEmpty(strings.TrimSpace(shell.FrameURLAttr), defaults.FrameURLAttr)
	shell.ViewportAttr = FirstNonEmpty(strings.TrimSpace(shell.ViewportAttr), defaults.ViewportAttr)
	shell.ViewportCurrentAttr = FirstNonEmpty(strings.TrimSpace(shell.ViewportCurrentAttr), defaults.ViewportCurrentAttr)
	shell.ViewportWidthAttr = FirstNonEmpty(strings.TrimSpace(shell.ViewportWidthAttr), defaults.ViewportWidthAttr)
	shell.SelectionLabelAttr = FirstNonEmpty(strings.TrimSpace(shell.SelectionLabelAttr), defaults.SelectionLabelAttr)
	return shell
}

func (canvas CanvasWorkspace) BlockCount() int {
	return len(canvas.Normalize().Blocks)
}

func (canvas CanvasWorkspace) VisibleBlockCount() int {
	count := 0
	for _, block := range canvas.Normalize().Blocks {
		if block.Visible {
			count++
		}
	}
	return count
}

func (canvas CanvasWorkspace) ControlCount() int {
	count := 0
	for _, block := range canvas.Normalize().Blocks {
		count += block.ControlCount()
	}
	return count
}

func (canvas CanvasWorkspace) ActiveViewport() CanvasViewport {
	viewports := canvas.Normalize().Viewports
	for _, viewport := range viewports {
		if viewport.Active {
			return viewport
		}
	}
	if len(viewports) == 0 {
		return CanvasViewport{}
	}
	return viewports[0]
}

func (canvas CanvasWorkspace) ActiveZoom() CanvasZoomLevel {
	levels := canvas.Normalize().ZoomLevels
	for _, level := range levels {
		if level.Active {
			return level
		}
	}
	if len(levels) == 0 {
		return CanvasZoomLevel{}
	}
	return levels[0]
}

func (canvas CanvasWorkspace) SelectedBlock() (CanvasBlock, bool) {
	for _, block := range canvas.Normalize().Blocks {
		if block.Selected {
			return block, true
		}
	}
	return CanvasBlock{}, false
}

func (block CanvasBlock) ControlCount() int {
	return len(block.Controls)
}

func (block CanvasBlock) Normalize() CanvasBlock {
	block.Key = strings.TrimSpace(block.Key)
	block.Label = strings.TrimSpace(block.Label)
	block.Summary = strings.TrimSpace(block.Summary)
	block.GoSXComponent = strings.TrimSpace(block.GoSXComponent)
	block.Source = normalizeComponentSource(block.Source)
	block.Binding = strings.TrimSpace(block.Binding)
	block.Status = strings.TrimSpace(block.Status)
	block.Controls = normalizeControls(block.Controls)
	return block
}

func (block CanvasBlock) NormalizedSource() ComponentSource {
	return normalizeComponentSource(block.Source)
}

func (action CanvasAction) NormalizedKind() CanvasActionKind {
	return normalizeCanvasActionKind(action.Kind)
}

func normalizeCanvasActionKind(kind CanvasActionKind) CanvasActionKind {
	switch CanvasActionKind(strings.TrimSpace(string(kind))) {
	case CanvasActionMoveUp:
		return CanvasActionMoveUp
	case CanvasActionMoveDown:
		return CanvasActionMoveDown
	case CanvasActionInlineText:
		return CanvasActionInlineText
	case CanvasActionContent:
		return CanvasActionContent
	case CanvasActionStyle:
		return CanvasActionStyle
	case CanvasActionToggleVisibility:
		return CanvasActionToggleVisibility
	default:
		return CanvasActionReveal
	}
}

func normalizeCanvasViewports(values []CanvasViewport) []CanvasViewport {
	if len(values) == 0 {
		values = []CanvasViewport{
			{Key: "desktop", Label: "Desktop", Width: "100%", Active: true},
			{Key: "tablet", Label: "Tablet", Width: "48rem"},
			{Key: "mobile", Label: "Mobile", Width: "24rem"},
		}
	}
	out := make([]CanvasViewport, 0, len(values))
	activeSet := false
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Width = strings.TrimSpace(value.Width)
		if value.Key == "" || value.Label == "" {
			continue
		}
		if value.Active {
			if activeSet {
				value.Active = false
			} else {
				activeSet = true
			}
		}
		out = append(out, value)
	}
	if len(out) > 0 && !activeSet {
		out[0].Active = true
	}
	return out
}

func normalizeCanvasZoomLevels(values []CanvasZoomLevel) []CanvasZoomLevel {
	if len(values) == 0 {
		values = []CanvasZoomLevel{
			{Key: "fit", Label: "Fit", Active: true},
			{Key: "75", Label: "75%", Scale: 0.75},
			{Key: "100", Label: "100%", Scale: 1},
			{Key: "125", Label: "125%", Scale: 1.25},
		}
	}
	out := make([]CanvasZoomLevel, 0, len(values))
	activeSet := false
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		if value.Key == "" || value.Label == "" {
			continue
		}
		if value.Active {
			if activeSet {
				value.Active = false
			} else {
				activeSet = true
			}
		}
		out = append(out, value)
	}
	if len(out) > 0 && !activeSet {
		out[0].Active = true
	}
	return out
}

func normalizeCanvasBlocks(values []CanvasBlock) []CanvasBlock {
	out := make([]CanvasBlock, 0, len(values))
	for _, value := range values {
		value = value.Normalize()
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeCanvasActions(values []CanvasAction) []CanvasAction {
	if len(values) == 0 {
		values = DefaultCanvasActions()
	}
	out := make([]CanvasAction, 0, len(values))
	for _, value := range values {
		value.Key = strings.TrimSpace(value.Key)
		value.Label = strings.TrimSpace(value.Label)
		value.Summary = strings.TrimSpace(value.Summary)
		value.Kind = normalizeCanvasActionKind(value.Kind)
		if value.Key == "" {
			value.Key = string(value.Kind)
		}
		if value.Label == "" {
			value.Label = CanvasActionKindLabel(value.Kind)
		}
		if value.Key == "" || value.Label == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
