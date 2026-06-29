package studio

import "m31labs.dev/gosx"

type BackendEditorWorkbenchProps struct {
	View         map[string]any
	AuthoringURL string
	Toolbar      gosx.Node
	CanvasBar    gosx.Node
	CanvasStatus gosx.Node
	LeftRail     []gosx.Node
	Board        []gosx.Node
	RightRail    []gosx.Node
}

func RenderBackendEditorWorkbench(props BackendEditorWorkbenchProps) gosx.Node {
	options := WorkbenchFrameOptions{
		AuthoringURL:   props.AuthoringURL,
		ResizableRails: true,
		Toolbar:        props.Toolbar,
		LeftRail:       []gosx.Node{RenderBackendEditorLeftRail(props.LeftRail...)},
		Board:          []gosx.Node{RenderBackendEditorBoard(props.Board...)},
		RightRail:      []gosx.Node{RenderBackendEditorRightRail(props.RightRail...)},
	}
	if !workbenchNodeEmpty(props.CanvasBar) {
		options.CanvasBar = []gosx.Node{props.CanvasBar}
	}
	if !workbenchNodeEmpty(props.CanvasStatus) {
		options.CanvasStatus = []gosx.Node{props.CanvasStatus}
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-frame-slot", "true")),
		RenderWorkbenchFrame(props.View, options),
	)
}

func RenderBackendEditorLeftRail(nodes ...gosx.Node) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-left-rail-slot", "true")), gosx.Fragment(nodes...))
}

func RenderBackendEditorBoard(nodes ...gosx.Node) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-board-slot", "true")), gosx.Fragment(nodes...))
}

func RenderBackendEditorRightRail(nodes ...gosx.Node) gosx.Node {
	return gosx.El("div", gosx.Attrs(gosx.Attr("data-gosx-studio-right-rail-slot", "true")), gosx.Fragment(nodes...))
}
