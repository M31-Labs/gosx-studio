package hostruntime

import (
	"net/http"
	"reflect"
	"testing"
)

type recordingMounter struct {
	patterns []string
	handlers []http.Handler
}

func (m *recordingMounter) Mount(pattern string, handler http.Handler) {
	m.patterns = append(m.patterns, pattern)
	m.handlers = append(m.handlers, handler)
}

func TestMountRuntimesRegistersStudioOwnedAssets(t *testing.T) {
	var mounter recordingMounter

	MountRuntimes(&mounter)

	want := []string{
		"GET " + StylesheetPath,
		"GET " + EngineRuntimePath,
		"GET " + WorkbenchRuntimePath,
		"GET " + CommandRuntimePath,
		"GET " + StateRuntimePath,
		"GET " + PreviewSubscriberPath,
		"GET " + CanvasSelectionBridgePath,
		"GET " + Canvas2DPainterPath,
		"GET " + CanvasWASMFreeClientPath,
		"GET " + CanvasInlineEditPath,
		"GET " + CanvasContextualPanelPath,
		"GET " + CanvasDefaultInlineInstallerPath,
	}
	if !reflect.DeepEqual(mounter.patterns, want) {
		t.Fatalf("mounted patterns = %#v, want %#v", mounter.patterns, want)
	}
	if len(mounter.handlers) != len(want) {
		t.Fatalf("mounted handlers = %d, want %d", len(mounter.handlers), len(want))
	}
	for i, handler := range mounter.handlers {
		if handler == nil {
			t.Fatalf("handler %d for %s is nil", i, mounter.patterns[i])
		}
	}

	MountRuntimes(nil)
}
