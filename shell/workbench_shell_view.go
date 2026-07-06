package shell

import (
	"strconv"
	"strings"

	"m31labs.dev/gosx-studio/core"
)

type WorkbenchShellSource struct {
	Title         string
	PreviewURL    string
	SaveAction    string
	BlockCount    int
	MediaCount    int
	RevisionCount int
	Metrics       []Metric
	Modes         []Mode
	Viewports     []Viewport
	Canvas        CanvasSurface
}

type WorkbenchShellViewOptions struct {
	Class              string
	FormClass          string
	FormID             string
	Method             string
	Action             string
	CSRFToken          string
	AutosaveURL        string
	AutosaveDelay      int
	Mode               string
	StyleSystemID      string
	StyleSystemValid   bool
	ToolbarKicker      string
	ToolbarTitle       string
	ModebarLabel       string
	MetricLabel        string
	CommandLauncher    string
	CommandTitle       string
	CommandSearchHint  string
	Commands           []map[string]any
	PreviewHref        string
	PreviewLabel       string
	SaveLabel          string
	RouteLabel         string
	SelectionLabel     string
	Zoom               string
	ViewportKey        string
	ViewportLabel      string
	PreviewURL         string
	PreviewTitle       string
	PreviewFrameTitle  string
	PreviewShell       core.CanvasPreviewShell
	ZoomLevels         []WorkbenchZoomLevel
	LeftResizer        WorkbenchRailResizer
	RightResizer       WorkbenchRailResizer
	LeftRailLabel      string
	RightRailLabel     string
	ActivityLabel      string
	FocusLabel         string
	CanvasAriaLabel    string
	CanvasPanelKey     string
	CommandEmptyTitle  string
	CommandEmptyDetail string
	FeatureFlagAttrs   map[string]any
	Extras             map[string]any
}

type WorkbenchZoomLevel struct {
	Key    string
	Label  string
	Active bool
}

type WorkbenchRailResizer struct {
	Side    string
	Label   string
	Min     int
	Max     int
	Default int
}

func WorkbenchShellSourceFromShell(shell Shell) WorkbenchShellSource {
	return WorkbenchShellSource{
		Title:         shell.Title,
		PreviewURL:    shell.PreviewURL,
		SaveAction:    shell.SaveAction,
		BlockCount:    shell.BlockCount,
		MediaCount:    shell.MediaCount,
		RevisionCount: shell.RevisionCount,
		Metrics:       append([]Metric(nil), shell.Metrics...),
		Modes:         append([]Mode(nil), shell.Modes...),
		Viewports:     append([]Viewport(nil), shell.Viewports...),
		Canvas:        shell.Canvas,
	}
}

func WorkbenchShellViewForShell(shell Shell, options WorkbenchShellViewOptions) map[string]any {
	return WorkbenchShellView(WorkbenchShellSourceFromShell(shell), options)
}

func WorkbenchShellView(source WorkbenchShellSource, options WorkbenchShellViewOptions) map[string]any {
	source = normalizeWorkbenchShellSource(source)
	action := core.FirstNonEmpty(options.Action, source.SaveAction)
	previewURL := core.FirstNonEmpty(options.PreviewURL, source.PreviewURL, "/")
	zoom := core.FirstNonEmpty(core.NormalizeKey(options.Zoom), source.Canvas.Zoom, "fit")
	routeLabel := core.FirstNonEmpty(options.RouteLabel, source.Canvas.RouteLabel, source.Title, "Preview")
	selectionLabel := core.FirstNonEmpty(options.SelectionLabel, source.Canvas.SelectionLabel, "No selection")
	viewportKey := core.FirstNonEmpty(options.ViewportKey, activeWorkbenchViewportKey(source.Viewports))
	viewportLabel := core.FirstNonEmpty(options.ViewportLabel, activeWorkbenchViewportLabel(source.Viewports))
	view := map[string]any{
		"class":              core.FirstNonEmpty(options.Class, "gosx-studio-workbench"),
		"formClass":          core.FirstNonEmpty(options.FormClass, "gosx-studio-workbench__form gosx-studio"),
		"formID":             strings.TrimSpace(options.FormID),
		"method":             core.FirstNonEmpty(options.Method, "post"),
		"action":             action,
		"hasAction":          action != "",
		"csrfToken":          strings.TrimSpace(options.CSRFToken),
		"hasCSRF":            strings.TrimSpace(options.CSRFToken) != "",
		"shellKey":           core.NormalizeKey(source.Title),
		"blockCount":         strconv.Itoa(source.BlockCount),
		"mediaCount":         strconv.Itoa(source.MediaCount),
		"revisionCount":      strconv.Itoa(source.RevisionCount),
		"metrics":            shellMetricViews(NormalizeMetrics(source.Metrics)),
		"modes":              shellModeViews(NormalizeModes(source.Modes)),
		"autosaveURL":        core.FirstNonEmpty(options.AutosaveURL, action),
		"autosaveDelay":      strconv.Itoa(firstPositive(options.AutosaveDelay, 1400)),
		"mode":               core.FirstNonEmpty(options.Mode, activeWorkbenchModeKey(source.Modes)),
		"styleSystemID":      strings.TrimSpace(options.StyleSystemID),
		"styleSystemValid":   core.BoolAttr(options.StyleSystemValid),
		"toolbarKicker":      core.FirstNonEmpty(options.ToolbarKicker, "Website"),
		"toolbarTitle":       core.FirstNonEmpty(options.ToolbarTitle, source.Title, "Editor"),
		"modebarLabel":       core.FirstNonEmpty(options.ModebarLabel, "Editor mode"),
		"metricLabel":        core.FirstNonEmpty(options.MetricLabel, "Workspace details"),
		"commandLauncher":    core.FirstNonEmpty(options.CommandLauncher, "Find"),
		"commandTitle":       core.FirstNonEmpty(options.CommandTitle, "Editor tools"),
		"commandSearchHint":  core.FirstNonEmpty(options.CommandSearchHint, "Find editing areas, sections, previews, and advanced tools"),
		"commands":           cloneWorkbenchCommandViews(options.Commands),
		"hasCommands":        len(options.Commands) > 0,
		"previewHref":        core.FirstNonEmpty(options.PreviewHref, previewURL),
		"previewLabel":       core.FirstNonEmpty(options.PreviewLabel, "Preview"),
		"saveLabel":          core.FirstNonEmpty(options.SaveLabel, "Save changes"),
		"routeLabel":         routeLabel,
		"selectionLabel":     selectionLabel,
		"zoom":               zoom,
		"viewportKey":        viewportKey,
		"viewportLabel":      viewportLabel,
		"viewports":          shellViewportViews(source.Viewports),
		"previewURL":         previewURL,
		"previewTitle":       core.FirstNonEmpty(options.PreviewTitle, source.Title+" preview", "Preview"),
		"previewFrameTitle":  core.FirstNonEmpty(options.PreviewFrameTitle, source.Title+" preview", "Preview"),
		"previewShell":       CanvasPreviewShellView(options.PreviewShell, previewURL),
		"zoomLevels":         WorkbenchZoomLevelViews(zoom, options.ZoomLevels),
		"leftResizer":        WorkbenchRailResizerView(options.LeftResizer, "left"),
		"rightResizer":       WorkbenchRailResizerView(options.RightResizer, "right"),
		"leftRailLabel":      core.FirstNonEmpty(options.LeftRailLabel, "Layers"),
		"rightRailLabel":     core.FirstNonEmpty(options.RightRailLabel, "Inspector"),
		"activityLabel":      core.FirstNonEmpty(options.ActivityLabel, "Activity"),
		"focusLabel":         core.FirstNonEmpty(options.FocusLabel, "Focus"),
		"canvasAriaLabel":    core.FirstNonEmpty(options.CanvasAriaLabel, routeLabel, source.Title, "Studio canvas"),
		"canvasPanelKey":     core.FirstNonEmpty(options.CanvasPanelKey, core.NormalizeKey(routeLabel), "canvas"),
		"commandEmptyTitle":  core.FirstNonEmpty(options.CommandEmptyTitle, "No commands"),
		"commandEmptyDetail": core.FirstNonEmpty(options.CommandEmptyDetail, "Try a different search."),
	}
	if len(options.FeatureFlagAttrs) > 0 {
		view["featureFlagAttrs"] = cloneAnyMap(options.FeatureFlagAttrs)
	}
	for key, value := range options.Extras {
		if _, exists := view[key]; !exists {
			view[key] = value
		}
	}
	return view
}

func CanvasPreviewShellView(shell core.CanvasPreviewShell, previewURL string) map[string]any {
	shell = shell.Normalize()
	return map[string]any{
		"overlayAttrs":         StudioAttrView(shell.OverlayAttr, "true"),
		"dropLineAttrs":        StudioAttrView(shell.DropLineAttr, "true"),
		"dockAttrs":            StudioAttrView(shell.DockAttr, "true"),
		"dockTitleAttrs":       StudioAttrView(shell.DockTitleAttr, "true"),
		"fieldCountAttrs":      StudioAttrView(shell.FieldCountAttr, "true"),
		"frameAttrs":           StudioAttrView(shell.FrameAttr, "true", shell.FrameURLAttr, previewURL),
		"outlineTemplateAttrs": StudioAttrView(shell.OutlineTemplateAttr, "true"),
		"outlineLabelAttrs":    StudioAttrView(shell.OutlineLabelAttr, "true"),
		"fieldTemplateAttrs":   StudioAttrView(shell.FieldTemplateAttr, "true"),
		"fieldActionTextAttrs": StudioAttrView(shell.FieldActionTextAttr, "true"),
		"moveUpActionAttrs":    StudioAttrView(shell.ActionAttr, string(core.CanvasActionMoveUp)),
		"moveDownActionAttrs":  StudioAttrView(shell.ActionAttr, string(core.CanvasActionMoveDown)),
		"inlineTextAttrs":      StudioAttrView(shell.ActionAttr, string(core.CanvasActionInlineText)),
		"contentActionAttrs":   StudioAttrView(shell.ActionAttr, string(core.CanvasActionContent)),
		"styleActionAttrs":     StudioAttrView(shell.ActionAttr, string(core.CanvasActionStyle)),
		"visibilityAttrs":      StudioAttrView(shell.ActionAttr, string(core.CanvasActionToggleVisibility)),
		"inlineEditClass":      shell.InlineEditClass,
		"styleImpactNodeAttr":  shell.StyleImpactNodeAttr,
		"viewportAttr":         shell.ViewportAttr,
		"viewportCurrentAttr":  shell.ViewportCurrentAttr,
		"viewportWidthAttr":    shell.ViewportWidthAttr,
		"selectionLabelAttr":   shell.SelectionLabelAttr,
	}
}

func StudioAttrView(pairs ...string) map[string]any {
	attrs := map[string]any{}
	for i := 0; i+1 < len(pairs); i += 2 {
		name := strings.TrimSpace(pairs[i])
		if name == "" {
			continue
		}
		attrs[name] = strings.TrimSpace(pairs[i+1])
	}
	return attrs
}

func WorkbenchZoomLevelViews(active string, levels []WorkbenchZoomLevel) []map[string]any {
	active = core.FirstNonEmpty(core.NormalizeKey(active), "fit")
	if len(levels) == 0 {
		levels = []WorkbenchZoomLevel{
			{Key: "fit", Label: "Fit"},
			{Key: "75", Label: "75%"},
			{Key: "100", Label: "100%"},
			{Key: "125", Label: "125%"},
		}
	}
	for _, level := range levels {
		if level.Active {
			active = core.NormalizeKey(level.Key)
			break
		}
	}
	out := make([]map[string]any, 0, len(levels))
	for _, level := range levels {
		key := core.NormalizeKey(level.Key)
		label := strings.TrimSpace(level.Label)
		if key == "" || label == "" {
			continue
		}
		out = append(out, map[string]any{
			"key":     key,
			"label":   label,
			"pressed": key == active,
		})
	}
	return out
}

func WorkbenchRailResizerView(resizer WorkbenchRailResizer, side string) map[string]any {
	side = core.NormalizeKey(core.FirstNonEmpty(resizer.Side, side, "left"))
	defaults := defaultWorkbenchRailResizer(side)
	resizer.Side = side
	resizer.Label = core.FirstNonEmpty(resizer.Label, defaults.Label)
	resizer.Min = firstPositive(resizer.Min, defaults.Min)
	resizer.Max = firstPositive(resizer.Max, defaults.Max)
	resizer.Default = firstPositive(resizer.Default, defaults.Default)
	return map[string]any{
		"side":    resizer.Side,
		"label":   resizer.Label,
		"min":     strconv.Itoa(resizer.Min),
		"max":     strconv.Itoa(resizer.Max),
		"default": strconv.Itoa(resizer.Default),
	}
}

func normalizeWorkbenchShellSource(source WorkbenchShellSource) WorkbenchShellSource {
	source.Title = core.FirstNonEmpty(source.Title, "Website editor")
	source.PreviewURL = core.FirstNonEmpty(source.PreviewURL, "/")
	source.SaveAction = strings.TrimSpace(source.SaveAction)
	source.Modes = NormalizeModes(source.Modes)
	source.Viewports = NormalizeViewports(source.Viewports)
	source.Canvas = normalizeShellCanvas(source.Canvas, source.Title)
	return source
}

func activeWorkbenchModeKey(modes []Mode) string {
	for _, mode := range NormalizeModes(modes) {
		if mode.Active {
			return mode.Key
		}
	}
	return "structure"
}

func activeWorkbenchViewportKey(viewports []Viewport) string {
	for _, viewport := range NormalizeViewports(viewports) {
		if viewport.Active {
			return viewport.Key
		}
	}
	return "desktop"
}

func activeWorkbenchViewportLabel(viewports []Viewport) string {
	for _, viewport := range NormalizeViewports(viewports) {
		if viewport.Active {
			return viewport.Label
		}
	}
	return "Desktop"
}

func defaultWorkbenchRailResizer(side string) WorkbenchRailResizer {
	if side == "right" {
		return WorkbenchRailResizer{Side: "right", Label: "Resize inspector rail", Min: 320, Max: 544, Default: 416}
	}
	return WorkbenchRailResizer{Side: "left", Label: "Resize layers rail", Min: 256, Max: 448, Default: 320}
}

func cloneWorkbenchCommandViews(commands []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(commands))
	for _, command := range commands {
		out = append(out, cloneAnyMap(command))
	}
	return out
}

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func firstPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
