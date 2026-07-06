package studio

// This file (originally host_runtime_test.go) covers the RuntimeConfig
// symbol that remains at the studio root after Slice 3 of the package
// restructure moved Mounter/MountRuntimes to
// m31labs.dev/gosx-studio/hostruntime (see runtime_config.go and
// hostruntime/host_runtime_test.go, which now carries
// TestMountRuntimesRegistersStudioOwnedAssets).

import (
	"reflect"
	"testing"
)

func TestDefaultRuntimeConfigExposesHostScriptsAndEngineHosts(t *testing.T) {
	config := DefaultRuntimeConfig("build 1")

	if config.AssetVersion != "build 1" {
		t.Fatalf("asset version = %q", config.AssetVersion)
	}
	if config.HostConfig().Labels.EditorTitle != DefaultShellConfig().Labels.EditorTitle {
		t.Fatalf("host labels = %#v", config.HostConfig().Labels)
	}
	if config.CanvasPreviewShell().OverlayAttr != "data-studio-preview-overlay" {
		t.Fatalf("preview shell = %#v", config.CanvasPreviewShell())
	}

	wantScripts := map[string]any{
		"engineRuntime":    EngineRuntimePath + "?v=build+1",
		"workbenchRuntime": WorkbenchRuntimePath + "?v=build+1",
		"commandRuntime":   CommandRuntimePath + "?v=build+1",
		"stateRuntime":     StateRuntimePath + "?v=build+1",
	}
	if got := config.Scripts(); !reflect.DeepEqual(got, wantScripts) {
		t.Fatalf("scripts = %#v, want %#v", got, wantScripts)
	}

	if got := config.CanvasEngineHost(); got["key"] != "canvas" || got["name"] != CanvasEngineName || got["mountId"] != "gosx-studio-canvas-engine" || got["class"] != "studio-canvas-engine-host" {
		t.Fatalf("canvas engine host = %#v", got)
	}
	if got := config.SiteMapEngineHost(); got["key"] != "site-map" || got["name"] != SiteMapEngineName || got["class"] != "studio-site-map-engine-host" {
		t.Fatalf("site-map engine host = %#v", got)
	}
	if got := config.BlockLayoutEngineHost(); got["key"] != "block-layout" || got["name"] != BlockLayoutEngineName || got["class"] != "studio-block-layout-engine-host" {
		t.Fatalf("block-layout engine host = %#v", got)
	}
	if got := config.Showcase3DEngineHost(); got["key"] != "showcase-3d" || got["name"] != Showcase3DEngineName || got["class"] != "studio-showcase3d-engine-host" {
		t.Fatalf("showcase-3d engine host = %#v", got)
	}

	hosts := config.EngineHosts()
	if len(hosts) != 1 || hosts[0]["key"] != "flow-designer" || hosts[0]["name"] != FlowDesignerName || hosts[0]["class"] != "studio-engine-host studio-engine-host--flow" {
		t.Fatalf("engine hosts = %#v", hosts)
	}
}
