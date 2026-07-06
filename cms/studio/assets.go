package studio

import (
	_ "embed"

	"m31labs.dev/gosx"
)

//go:embed assets/command_palette.js
var commandPaletteRuntime string

//go:embed assets/state_runtime.js
var stateRuntime string

//go:embed assets/site_canvas_runtime.js
var siteCanvasRuntime string

//go:embed assets/workbench_runtime.js
var workbenchRuntime string

//go:embed assets/flow_editor_runtime.js
var flowEditorRuntime string

func CommandPaletteScript() string {
	return commandPaletteRuntime
}

func RenderCommandPaletteScript() gosx.Node {
	return gosx.RawHTML(`<script data-gosx-studio-command-runtime="true">` + commandPaletteRuntime + `</script>`)
}

func StateRuntimeScript() string {
	return stateRuntime
}

func RenderStudioStateScript() gosx.Node {
	return gosx.RawHTML(`<script data-gosx-studio-state-runtime="true">` + stateRuntime + `</script>`)
}

func SiteCanvasScript() string {
	return siteCanvasRuntime
}

func RenderSiteCanvasScript() gosx.Node {
	return gosx.RawHTML(`<script data-gosx-studio-site-canvas-runtime="true">` + siteCanvasRuntime + `</script>`)
}

func WorkbenchScript() string {
	return workbenchRuntime
}

func RenderWorkbenchScript() gosx.Node {
	return gosx.RawHTML(`<script data-gosx-studio-workbench-runtime="true">` + workbenchRuntime + `</script>`)
}

func FlowEditorScript() string {
	return flowEditorRuntime
}

func RenderFlowEditorScript() gosx.Node {
	return gosx.RawHTML(`<script data-gosx-studio-flow-editor-runtime="true">` + flowEditorRuntime + `</script>`)
}
