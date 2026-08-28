package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/hostruntime"
	hostshell "m31labs.dev/gosx-studio/shell"
)

func TestStudioAssetForkPreservesCommandAndStateSemanticContract(t *testing.T) {
	commandMarkers := []string{
		`gosxstudio:command`,
		`[data-studio-command-palette]`,
		`[data-studio-command]`,
		`[data-studio-command-search]`,
		`[data-studio-command-open]`,
		`data-studio-command-shortcut`,
	}
	stateMarkers := []string{
		`gosxstudio:save-state`,
		`gosxstudio:action-result`,
		`gosxstudio:history-state`,
		`gosxstudio:history-restore`,
		`[data-gosx-studio-state]`,
		`data-gosx-studio-client`,
		`data-gosx-studio-autosave`,
		`requestSubmit`,
		`restoreFormState`,
	}

	for _, bundle := range []struct {
		name    string
		script  string
		markers []string
	}{
		{name: "cms command", script: CommandPaletteScript(), markers: commandMarkers},
		{name: "host command", script: string(hostruntime.CommandRuntimeScript()), markers: commandMarkers},
		{name: "cms state", script: StateRuntimeScript(), markers: stateMarkers},
		{name: "host state", script: string(hostruntime.StateRuntimeScript()), markers: stateMarkers},
	} {
		t.Run(bundle.name, func(t *testing.T) {
			assertAssetMarkers(t, bundle.name, bundle.script, bundle.markers)
		})
	}
}

func TestStudioAssetForkPreservesWorkbenchChromeAndMarkupContract(t *testing.T) {
	workbenchMarkers := []string{
		`window.GoSXStudioWorkbenchRuntime`,
		`form[data-studio-workbench], form[data-editor-workbench]`,
		`[data-studio-mode-control]`,
		`[data-studio-viewport]`,
		`[data-studio-zoom]`,
		`[data-studio-rail-toggle]`,
		`[data-studio-focus-toggle]`,
		`[data-studio-activity-toggle]`,
		`[data-studio-preview-frame]`,
		`emit(form, "gosxstudio:mode-change"`,
		`emit(form, "gosxstudio:viewport-change"`,
		`emit(form, "gosxstudio:zoom-change"`,
		`emit(form, "gosxstudio:rail-change"`,
		`emit(form, "gosxstudio:focus-change"`,
		`emit(form, "gosxstudio:activity-change"`,
		`emit(form, "gosxstudio:workbench-layout"`,
	}
	assertAssetMarkers(t, "cms workbench", WorkbenchScript(), workbenchMarkers)
	assertAssetMarkers(t, "host workbench", string(hostruntime.WorkbenchRuntimeScript()), workbenchMarkers)

	contractMarkers := []string{
		`data-gosx-studio-preview="true"`,
		`data-studio-preview-frame="true"`,
		`data-studio-preview-src`,
		`data-studio-preview-route`,
	}
	cmsMarkup := gosx.RenderHTML(RenderPreviewFrame(PreviewFrameOptions{URL: "/", Title: "Preview"}))
	hostMarkup := gosx.RenderHTML(hostshell.RenderWorkbenchPageCanvas(hostshell.WorkbenchPageCanvasOptions{URL: "/"}))
	for _, marker := range contractMarkers {
		if !strings.Contains(cmsMarkup, marker) {
			t.Fatalf("CMS preview markup missing shared marker %q", marker)
		}
		if !strings.Contains(hostMarkup, marker) {
			t.Fatalf("host page-canvas markup missing shared marker %q", marker)
		}
	}
}

func TestStudioAssetForkKeepsPermittedWorkbenchDivergenceExplicit(t *testing.T) {
	cms := WorkbenchScript()
	host := string(hostruntime.WorkbenchRuntimeScript())
	for _, marker := range []string{
		`function applyPreviewPatch(`,
		`function previewDockForFrame(`,
		`gosxstudio:preview-patch`,
		`gosxstudio:inline-text-start`,
	} {
		if !strings.Contains(cms, marker) {
			t.Fatalf("CMS workbench must retain direct preview/inline ownership marker %q", marker)
		}
	}
	for _, marker := range []string{
		`window.GoSXStudioFieldRuntime`,
		`window.GoSXStudioInlineEditRuntime`,
		`window.GoSXStudioSelectionRuntime`,
		`window.__gosx_preview_runtime_island_postPatch`,
		`window.__gosx_preview_runtime_island_bindFrames`,
	} {
		if !strings.Contains(host, marker) {
			t.Fatalf("host workbench must retain delegated preview ownership marker %q", marker)
		}
	}
	if strings.Contains(host, `gosxstudio:preview-patch`) || strings.Contains(host, `function previewDockForFrame(`) {
		t.Fatalf("host workbench must not reclaim CMS-owned direct preview implementation")
	}
}

func assertAssetMarkers(t *testing.T, name, script string, markers []string) {
	t.Helper()
	if strings.TrimSpace(script) == "" {
		t.Fatalf("%s asset is empty", name)
	}
	for _, marker := range markers {
		if !strings.Contains(script, marker) {
			t.Fatalf("%s asset missing semantic marker %q", name, marker)
		}
	}
}
