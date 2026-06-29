package studio

import (
	"strings"

	"m31labs.dev/gosx"
)

type WorkbenchFrameOptions struct {
	Class              string
	FormClass          string
	FormID             string
	Method             string
	Action             string
	StageClass         string
	LeftRailClass      string
	MainClass          string
	CanvasShellClass   string
	CanvasBarClass     string
	BoardClass         string
	RightRailClass     string
	CSRFName           string
	AuthoringURL       string
	ResizableRails     bool
	DisableCSRF        bool
	DisableToolbar     bool
	DisableAutosave    bool
	DisableClientState bool
	RootAttrs          map[string]any
	FormAttrs          map[string]any
	Toolbar            gosx.Node
	BeforeToolbar      []gosx.Node
	LeftRail           []gosx.Node
	Main               []gosx.Node
	CanvasBar          []gosx.Node
	Board              []gosx.Node
	CanvasStatus       []gosx.Node
	RightRail          []gosx.Node
	AfterForm          []gosx.Node
}

type WorkbenchRailResizerOptions struct {
	Class   string
	Side    string
	Label   string
	Min     string
	Max     string
	Default string
}

type WorkbenchFrameSegments struct {
	RootOpen         gosx.Node
	FormOpen         gosx.Node
	CSRFInput        gosx.Node
	StageOpen        gosx.Node
	LeftRailOpen     gosx.Node
	LeftRailClose    gosx.Node
	LeftResizer      gosx.Node
	MainOpen         gosx.Node
	CanvasShellOpen  gosx.Node
	BoardOpen        gosx.Node
	BoardClose       gosx.Node
	CanvasShellClose gosx.Node
	MainClose        gosx.Node
	RightResizer     gosx.Node
	RightRailOpen    gosx.Node
	RightRailClose   gosx.Node
	StageClose       gosx.Node
	FormClose        gosx.Node
	RootClose        gosx.Node
}

func RenderWorkbenchFrame(view map[string]any, options WorkbenchFrameOptions) gosx.Node {
	formChildren := make([]gosx.Node, 0, 5+len(options.AfterForm))
	if !options.DisableCSRF {
		formChildren = appendWorkbenchNode(formChildren, renderWorkbenchFrameCSRF(view, options))
	}
	formChildren = append(formChildren, options.BeforeToolbar...)
	if !options.DisableToolbar {
		toolbar := options.Toolbar
		if workbenchNodeEmpty(toolbar) {
			toolbar = RenderWorkbenchToolbar(view, WorkbenchToolbarOptions{})
		}
		formChildren = appendWorkbenchNode(formChildren, toolbar)
	}
	formChildren = append(formChildren, renderWorkbenchFrameStage(view, options))
	formChildren = append(formChildren, options.AfterForm...)

	return gosx.El("div", gosx.Attrs(workbenchFrameRootAttrs(view, options)...),
		gosx.El("form", gosx.Attrs(workbenchFrameFormAttrs(view, options)...), gosx.Fragment(formChildren...)),
	)
}

func RenderWorkbenchFrameSegments(view map[string]any, options WorkbenchFrameOptions) WorkbenchFrameSegments {
	return WorkbenchFrameSegments{
		RootOpen:         renderWorkbenchFrameOpenTag("div", workbenchFrameRootAttrs(view, options)),
		FormOpen:         renderWorkbenchFrameOpenTag("form", workbenchFrameFormAttrs(view, options)),
		CSRFInput:        renderWorkbenchFrameCSRF(view, options),
		StageOpen:        renderWorkbenchFrameOpenTag("div", workbenchFrameStageAttrs(options)),
		LeftRailOpen:     renderWorkbenchFrameOpenTag("aside", workbenchFrameRailAttrs(FirstNonEmpty(options.LeftRailClass, "studio-left-rail"), "left")),
		LeftRailClose:    gosx.RawHTML("</aside>"),
		LeftResizer:      RenderWorkbenchRailResizer(view, "left", WorkbenchRailResizerOptions{}),
		MainOpen:         renderWorkbenchFrameOpenTag("section", workbenchFrameMainAttrs(view, options)),
		CanvasShellOpen:  renderWorkbenchFrameOpenTag("div", workbenchFrameCanvasShellAttrs(view, options)),
		BoardOpen:        renderWorkbenchFrameOpenTag("div", workbenchFrameBoardAttrs(options)),
		BoardClose:       gosx.RawHTML("</div>"),
		CanvasShellClose: gosx.RawHTML("</div>"),
		MainClose:        gosx.RawHTML("</section>"),
		RightResizer:     RenderWorkbenchRailResizer(view, "right", WorkbenchRailResizerOptions{}),
		RightRailOpen:    renderWorkbenchFrameOpenTag("aside", workbenchFrameRailAttrs(FirstNonEmpty(options.RightRailClass, "editor-sidebar"), "right")),
		RightRailClose:   gosx.RawHTML("</aside>"),
		StageClose:       gosx.RawHTML("</div>"),
		FormClose:        gosx.RawHTML("</form>"),
		RootClose:        gosx.RawHTML("</div>"),
	}
}

func RenderWorkbenchRailResizer(view map[string]any, side string, options WorkbenchRailResizerOptions) gosx.Node {
	resizer := workbenchFrameResizerView(view, side)
	side = NormalizeKey(FirstNonEmpty(options.Side, workbenchMapString(resizer, "side"), side, "left"))
	className := FirstNonEmpty(options.Class, "studio-rail-resizer studio-rail-resizer--"+side)
	return gosx.El("button", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("type", "button"),
		gosx.Attr("role", "separator"),
		gosx.Attr("aria-orientation", "vertical"),
		gosx.Attr("aria-label", FirstNonEmpty(options.Label, workbenchMapString(resizer, "label"), "Resize rail")),
		gosx.Attr("aria-valuemin", FirstNonEmpty(options.Min, workbenchMapString(resizer, "min"))),
		gosx.Attr("aria-valuemax", FirstNonEmpty(options.Max, workbenchMapString(resizer, "max"))),
		gosx.Attr("aria-valuenow", FirstNonEmpty(options.Default, workbenchMapString(resizer, "default"))),
		gosx.Attr("data-studio-resizer", side),
		gosx.Attr("data-studio-rail-min", FirstNonEmpty(options.Min, workbenchMapString(resizer, "min"))),
		gosx.Attr("data-studio-rail-max", FirstNonEmpty(options.Max, workbenchMapString(resizer, "max"))),
		gosx.Attr("data-studio-rail-default", FirstNonEmpty(options.Default, workbenchMapString(resizer, "default"))),
		gosx.Attr("data-gosx-studio-rail-resizer-renderer", "gosx-studio"),
	))
}

func renderWorkbenchFrameStage(view map[string]any, options WorkbenchFrameOptions) gosx.Node {
	children := []gosx.Node{
		renderWorkbenchFrameRail("aside", FirstNonEmpty(options.LeftRailClass, "studio-left-rail"), "left", options.LeftRail),
	}
	if options.ResizableRails {
		children = append(children, RenderWorkbenchRailResizer(view, "left", WorkbenchRailResizerOptions{}))
	}
	children = append(children, renderWorkbenchFrameMain(view, options))
	if options.ResizableRails {
		children = append(children, RenderWorkbenchRailResizer(view, "right", WorkbenchRailResizerOptions{}))
	}
	children = append(children, renderWorkbenchFrameRail("aside", FirstNonEmpty(options.RightRailClass, "editor-sidebar"), "right", options.RightRail))

	return gosx.El("div", gosx.Attrs(workbenchFrameStageAttrs(options)...), gosx.Fragment(children...))
}

func renderWorkbenchFrameRail(tag, className, side string, nodes []gosx.Node) gosx.Node {
	return gosx.El(tag, gosx.Attrs(workbenchFrameRailAttrs(className, side)...), gosx.Fragment(nodes...))
}

func renderWorkbenchFrameMain(view map[string]any, options WorkbenchFrameOptions) gosx.Node {
	children := options.Main
	if len(children) == 0 {
		children = []gosx.Node{renderWorkbenchFrameCanvasShell(view, options)}
	}
	return gosx.El("section", gosx.Attrs(workbenchFrameMainAttrs(view, options)...), gosx.Fragment(children...))
}

func renderWorkbenchFrameCanvasShell(view map[string]any, options WorkbenchFrameOptions) gosx.Node {
	children := []gosx.Node{}
	if len(options.CanvasBar) > 0 {
		children = append(children, options.CanvasBar...)
	}
	children = append(children, gosx.El("div", gosx.Attrs(workbenchFrameBoardAttrs(options)...), gosx.Fragment(options.Board...)))
	children = append(children, options.CanvasStatus...)
	children = appendWorkbenchNode(children, RenderWorkbenchPreviewDockShell(view))

	return gosx.El("div", gosx.Attrs(workbenchFrameCanvasShellAttrs(view, options)...), gosx.Fragment(children...))
}

func RenderWorkbenchPreviewDockShell(view map[string]any) gosx.Node {
	previewShell := workbenchViewMap(view, "previewShell")
	if len(previewShell) == 0 {
		return gosx.Fragment()
	}
	return gosx.El("div", gosx.Attrs(workbenchPreviewDockAttrs(previewShell)...),
		gosx.El("strong", gosx.Attrs(workbenchPreviewDockTitleAttrs(previewShell)...), gosx.Text("Preview selection")),
		gosx.El("span", gosx.Attrs(
			gosx.Attr("hidden", "hidden"),
			gosx.Attr("data-gosx-studio-preview-breadcrumb", "true"),
		)),
		gosx.El("span", gosx.Attrs(gosx.Attr("data-gosx-studio-preview-dock-kind", "true")), gosx.Text("Block")),
		gosx.El("span", gosx.Attrs(workbenchPreviewFieldCountAttrs(previewShell)...)),
		gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-preview-dock-actions", "true")),
			workbenchPreviewDockButton(previewShell, "contentActionAttrs", "content", "Content"),
			workbenchPreviewDockButton(previewShell, "styleActionAttrs", "style", "Style"),
			workbenchPreviewDockButton(previewShell, "moveUpActionAttrs", "move-up", "Move up"),
			workbenchPreviewDockButton(previewShell, "moveDownActionAttrs", "move-down", "Move down"),
			workbenchPreviewDockButton(previewShell, "", "prev-field", "Prev field"),
			workbenchPreviewDockButton(previewShell, "", "next-field", "Next field"),
			workbenchPreviewDockButton(previewShell, "inlineTextAttrs", "field-action", "Open"),
			workbenchPreviewDockButton(previewShell, "visibilityAttrs", "toggle-visibility", "Visibility"),
			workbenchPreviewDockButton(previewShell, "", "clear", "Clear"),
		),
	)
}

func renderWorkbenchFrameCSRF(view map[string]any, options WorkbenchFrameOptions) gosx.Node {
	if !workbenchViewBool(view, "hasCSRF") {
		return gosx.Fragment()
	}
	token := workbenchViewString(view, "csrfToken")
	if token == "" {
		return gosx.Fragment()
	}
	return gosx.El("input", gosx.Attrs(
		gosx.Attr("type", "hidden"),
		gosx.Attr("name", FirstNonEmpty(options.CSRFName, "csrf_token")),
		gosx.Attr("value", token),
	))
}

func workbenchFrameRootAttrs(view map[string]any, options WorkbenchFrameOptions) []any {
	attrs := []any{
		gosx.Attr("class", FirstNonEmpty(options.Class, workbenchViewString(view, "class"), "gosx-studio-workbench")),
		gosx.Attr("data-gosx-studio-workbench", "true"),
		gosx.Attr("data-gosx-studio-frame-renderer", "gosx-studio"),
	}
	attrs = appendWorkbenchFrameAttrs(attrs, workbenchViewMap(view, "featureFlagAttrs"))
	return appendWorkbenchFrameAttrs(attrs, options.RootAttrs)
}

func workbenchFrameFormAttrs(view map[string]any, options WorkbenchFrameOptions) []any {
	attrs := []any{}
	if formID := FirstNonEmpty(options.FormID, workbenchViewString(view, "formID")); formID != "" {
		attrs = append(attrs, gosx.Attr("id", formID))
	}
	attrs = append(attrs,
		gosx.Attr("class", FirstNonEmpty(options.FormClass, workbenchViewString(view, "formClass"), "gosx-studio-workbench__form gosx-studio")),
		gosx.Attr("method", FirstNonEmpty(options.Method, workbenchViewString(view, "method"), "post")),
	)
	if action := FirstNonEmpty(options.Action, workbenchViewString(view, "action")); action != "" {
		attrs = append(attrs, gosx.Attr("action", action))
	}
	attrs = append(attrs,
		gosx.Attr("data-studio-workbench", "true"),
		gosx.Attr("data-editor-workbench", "true"),
		gosx.Attr("data-gosx-studio-state", "true"),
		gosx.Attr("data-studio-shell", workbenchViewString(view, "shellKey")),
		gosx.Attr("data-studio-block-count", workbenchViewString(view, "blockCount")),
		gosx.Attr("data-studio-media-count", workbenchViewString(view, "mediaCount")),
		gosx.Attr("data-studio-revision-count", workbenchViewString(view, "revisionCount")),
		gosx.Attr("data-studio-mode", workbenchViewString(view, "mode")),
	)
	if !options.DisableClientState {
		attrs = append(attrs, gosx.Attr("data-gosx-studio-client", "true"))
	}
	if !options.DisableAutosave {
		attrs = append(attrs,
			gosx.Attr("data-gosx-studio-autosave", "true"),
			gosx.Attr("data-gosx-studio-autosave-delay", workbenchViewString(view, "autosaveDelay")),
			gosx.Attr("data-gosx-studio-autosave-url", workbenchViewString(view, "autosaveURL")),
		)
	}
	if token := workbenchViewString(view, "csrfToken"); token != "" {
		attrs = append(attrs, gosx.Attr("data-gosx-csrf-token", token))
	}
	if authoringURL := FirstNonEmpty(options.AuthoringURL, workbenchViewString(view, "authoringURL")); authoringURL != "" {
		attrs = append(attrs, gosx.Attr("data-gosx-studio-authoring-url", authoringURL))
	}
	if styleSystemID := workbenchViewString(view, "styleSystemID"); styleSystemID != "" {
		attrs = append(attrs, gosx.Attr("data-studio-style-system", styleSystemID))
	}
	if styleSystemValid := workbenchViewString(view, "styleSystemValid"); styleSystemValid != "" {
		attrs = append(attrs, gosx.Attr("data-studio-style-valid", styleSystemValid))
	}
	return appendWorkbenchFrameAttrs(attrs, options.FormAttrs)
}

func workbenchFrameStageAttrs(options WorkbenchFrameOptions) []any {
	return []any{
		gosx.Attr("class", FirstNonEmpty(options.StageClass, "editor-stage")),
		gosx.Attr("data-studio-layout", "true"),
		gosx.Attr("data-gosx-studio-stage-renderer", "gosx-studio"),
	}
}

func workbenchFrameRailAttrs(className, side string) []any {
	return []any{
		gosx.Attr("class", className),
		gosx.Attr("data-studio-sidebar", side),
		gosx.Attr("data-gosx-studio-rail-renderer", "gosx-studio"),
	}
}

func workbenchFrameMainAttrs(view map[string]any, options WorkbenchFrameOptions) []any {
	return []any{
		gosx.Attr("class", FirstNonEmpty(options.MainClass, "editor-canvas")),
		gosx.Attr("aria-label", FirstNonEmpty(workbenchViewString(view, "canvasAriaLabel"), workbenchViewString(view, "routeLabel"), "Studio canvas")),
		gosx.Attr("data-panel-key", FirstNonEmpty(workbenchViewString(view, "canvasPanelKey"), NormalizeKey(workbenchViewString(view, "routeLabel")), "canvas")),
		gosx.Attr("data-gosx-studio-main-renderer", "gosx-studio"),
	}
}

func workbenchFrameCanvasShellAttrs(view map[string]any, options WorkbenchFrameOptions) []any {
	return []any{
		gosx.Attr("class", FirstNonEmpty(options.CanvasShellClass, "studio-canvas-shell")),
		gosx.Attr("data-studio-canvas", "true"),
		gosx.Attr("data-studio-preview-shell", "true"),
		gosx.Attr("data-studio-canvas-zoom", FirstNonEmpty(workbenchViewString(view, "zoom"), "fit")),
		gosx.Attr("data-gosx-studio-canvas-shell-renderer", "gosx-studio"),
	}
}

func workbenchFrameBoardAttrs(options WorkbenchFrameOptions) []any {
	return []any{
		gosx.Attr("class", FirstNonEmpty(options.BoardClass, "studio-canvas-board")),
		gosx.Attr("data-gosx-studio-board-renderer", "gosx-studio"),
	}
}

func renderWorkbenchFrameOpenTag(tag string, attrs []any) gosx.Node {
	html := gosx.RenderHTML(gosx.El(tag, gosx.Attrs(attrs...)))
	closeTag := "</" + tag + ">"
	html = strings.TrimSuffix(html, closeTag)
	return gosx.RawHTML(html)
}

func workbenchFrameResizerView(view map[string]any, side string) map[string]any {
	if side == "right" {
		return workbenchViewMap(view, "rightResizer")
	}
	return workbenchViewMap(view, "leftResizer")
}

func workbenchViewMap(view map[string]any, key string) map[string]any {
	if view == nil {
		return nil
	}
	switch typed := view[key].(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = value
		}
		return out
	default:
		return nil
	}
}

func workbenchViewBool(view map[string]any, key string) bool {
	if view == nil {
		return false
	}
	switch typed := view[key].(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func appendWorkbenchFrameAttrs(attrs []any, values map[string]any) []any {
	for key, value := range values {
		name := strings.TrimSpace(key)
		if name == "" || value == nil {
			continue
		}
		attrs = append(attrs, gosx.Attr(name, value))
	}
	return attrs
}

func workbenchPreviewDockAttrs(previewShell map[string]any) []any {
	attrs := blockLibraryPanelMapAttrs(workbenchViewMap(previewShell, "dockAttrs"))
	attrs = append(attrs,
		gosx.Attr("hidden", "hidden"),
		gosx.Attr("role", "toolbar"),
		gosx.Attr("aria-label", "Preview selection actions"),
		gosx.Attr("data-gosx-studio-preview-dock", "true"),
		gosx.Attr("data-gosx-studio-preview-dock-renderer", "gosx-studio"),
	)
	return attrs
}

func workbenchPreviewDockTitleAttrs(previewShell map[string]any) []any {
	attrs := blockLibraryPanelMapAttrs(workbenchViewMap(previewShell, "dockTitleAttrs"))
	attrs = append(attrs, gosx.Attr("data-gosx-studio-preview-dock-label", "true"))
	return attrs
}

func workbenchPreviewFieldCountAttrs(previewShell map[string]any) []any {
	attrs := blockLibraryPanelMapAttrs(workbenchViewMap(previewShell, "fieldCountAttrs"))
	attrs = append(attrs,
		gosx.Attr("hidden", "hidden"),
		gosx.Attr("data-gosx-studio-preview-field-meter", "true"),
	)
	return attrs
}

func workbenchPreviewDockButton(previewShell map[string]any, attrKey, command, label string) gosx.Node {
	attrs := []any{
		gosx.Attr("type", "button"),
		gosx.Attr("data-gosx-studio-preview-command", command),
	}
	if attrKey != "" {
		attrs = append(attrs, blockLibraryPanelMapAttrs(workbenchViewMap(previewShell, attrKey))...)
	}
	return gosx.El("button", gosx.Attrs(attrs...), gosx.Text(label))
}
