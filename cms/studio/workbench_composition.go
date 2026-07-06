package studio

import "m31labs.dev/gosx"

type WorkbenchCompositionOptions struct {
	Commands                  []Command
	CommandPaletteNode        gosx.Node
	SaveStatusNode            gosx.Node
	CommandPalette            CommandPaletteOptions
	SaveStatus                SaveStatusOptions
	Workbench                 WorkbenchOptions
	Health                    HealthReport
	HealthPanel               HealthPanelOptions
	PublishReview             PublishReview
	PublishReviewPanel        PublishReviewOptions
	IncludeHealthPanel        bool
	IncludePublishReviewPanel bool
	IncludeRuntimeNodes       bool
}

type WorkbenchComposition struct {
	CommandPaletteNode     gosx.Node
	SaveStatusNode         gosx.Node
	CommandRuntimeNode     gosx.Node
	StateRuntimeNode       gosx.Node
	SiteCanvasRuntimeNode  gosx.Node
	WorkbenchRuntimeNode   gosx.Node
	FlowRuntimeNode        gosx.Node
	HealthPanelNode        gosx.Node
	PublishReviewPanelNode gosx.Node
	WorkbenchNode          gosx.Node
}

func ComposeWorkbench(shell Shell, options WorkbenchCompositionOptions) WorkbenchComposition {
	commands := normalizeCommands(append([]Command{}, options.Commands...))
	commandPaletteOptions := options.CommandPalette
	if len(commandPaletteOptions.Commands) == 0 {
		commandPaletteOptions.Commands = commands
	}
	commandPaletteNode := options.CommandPaletteNode
	if isZeroNode(commandPaletteNode) {
		commandPaletteNode = RenderCommandPalette(commandPaletteOptions)
	}
	saveStatusNode := options.SaveStatusNode
	if isZeroNode(saveStatusNode) {
		saveStatusNode = RenderSaveStatus(options.SaveStatus)
	}

	workbenchOptions := options.Workbench
	if len(workbenchOptions.Commands) == 0 {
		workbenchOptions.Commands = commands
	}
	healthPanelNode := gosx.Node{}
	if options.IncludeHealthPanel {
		healthPanelNode = RenderHealthPanel(options.Health, options.HealthPanel)
		workbenchOptions.CanvasFooter = append(workbenchOptions.CanvasFooter, healthPanelNode)
	}
	publishReviewPanelNode := gosx.Node{}
	if options.IncludePublishReviewPanel {
		publishReviewPanelNode = RenderPublishReviewPanel(options.PublishReview, options.PublishReviewPanel)
		workbenchOptions.CanvasFooter = append(workbenchOptions.CanvasFooter, publishReviewPanelNode)
	}

	composition := WorkbenchComposition{
		CommandPaletteNode:     commandPaletteNode,
		SaveStatusNode:         saveStatusNode,
		HealthPanelNode:        healthPanelNode,
		PublishReviewPanelNode: publishReviewPanelNode,
		WorkbenchNode:          RenderWorkbench(shell, workbenchOptions),
	}
	if options.IncludeRuntimeNodes {
		composition.WorkbenchRuntimeNode = RenderWorkbenchScript()
		composition.CommandRuntimeNode = RenderCommandPaletteScript()
		composition.StateRuntimeNode = RenderStudioStateScript()
		composition.SiteCanvasRuntimeNode = RenderSiteCanvasScript()
		composition.FlowRuntimeNode = RenderFlowEditorScript()
	}
	return composition
}

func (composition WorkbenchComposition) View() map[string]any {
	view := map[string]any{
		"commandPalette":         renderCompositionHTML(composition.CommandPaletteNode),
		"commandPaletteHTML":     renderCompositionHTML(composition.CommandPaletteNode),
		"commandPaletteNode":     composition.CommandPaletteNode,
		"saveStatus":             renderCompositionHTML(composition.SaveStatusNode),
		"saveStatusHTML":         renderCompositionHTML(composition.SaveStatusNode),
		"saveStatusNode":         composition.SaveStatusNode,
		"workbench":              renderCompositionHTML(composition.WorkbenchNode),
		"workbenchHTML":          renderCompositionHTML(composition.WorkbenchNode),
		"workbenchNode":          composition.WorkbenchNode,
		"healthPanelNode":        composition.HealthPanelNode,
		"publishReviewPanelNode": composition.PublishReviewPanelNode,
	}
	if !isZeroNode(composition.WorkbenchRuntimeNode) {
		view["workbenchRuntime"] = renderCompositionHTML(composition.WorkbenchRuntimeNode)
		view["workbenchRuntimeHTML"] = renderCompositionHTML(composition.WorkbenchRuntimeNode)
		view["workbenchRuntimeNode"] = composition.WorkbenchRuntimeNode
	}
	if !isZeroNode(composition.CommandRuntimeNode) {
		view["commandRuntime"] = renderCompositionHTML(composition.CommandRuntimeNode)
		view["commandRuntimeHTML"] = renderCompositionHTML(composition.CommandRuntimeNode)
		view["commandRuntimeNode"] = composition.CommandRuntimeNode
	}
	if !isZeroNode(composition.StateRuntimeNode) {
		view["stateRuntime"] = renderCompositionHTML(composition.StateRuntimeNode)
		view["stateRuntimeHTML"] = renderCompositionHTML(composition.StateRuntimeNode)
		view["stateRuntimeNode"] = composition.StateRuntimeNode
	}
	if !isZeroNode(composition.SiteCanvasRuntimeNode) {
		view["siteCanvasRuntime"] = renderCompositionHTML(composition.SiteCanvasRuntimeNode)
		view["siteCanvasRuntimeHTML"] = renderCompositionHTML(composition.SiteCanvasRuntimeNode)
		view["siteCanvasRuntimeNode"] = composition.SiteCanvasRuntimeNode
	}
	if !isZeroNode(composition.FlowRuntimeNode) {
		view["flowRuntime"] = renderCompositionHTML(composition.FlowRuntimeNode)
		view["flowRuntimeHTML"] = renderCompositionHTML(composition.FlowRuntimeNode)
		view["flowRuntimeNode"] = composition.FlowRuntimeNode
	}
	return view
}

func (composition WorkbenchComposition) MergeView(view map[string]any) map[string]any {
	if view == nil {
		view = map[string]any{}
	}
	for key, value := range composition.View() {
		view[key] = value
	}
	return view
}

func renderCompositionHTML(node gosx.Node) string {
	if isZeroNode(node) {
		return ""
	}
	return gosx.RenderHTML(node)
}

func isZeroNode(node gosx.Node) bool {
	return gosx.RenderHTML(node) == "<></>"
}
