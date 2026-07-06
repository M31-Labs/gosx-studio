package hostruntime

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"m31labs.dev/gosx-studio/core"
)

func testIsland(name string, body []byte) core.PluginIsland {
	return core.PluginIsland{
		FeatureFlagKey: "test-runtime-islands",
		ScriptName:     name,
		Bundle:         func() []byte { return body },
	}
}

// An empty or nil registry must leave the engine bundle byte-identical to the
// built-in-only output, so existing hosts are provably unaffected.
func TestEngineRuntimeScriptWithEmptyRegistryIsByteIdentical(t *testing.T) {
	want := EngineRuntimeScript()
	if got := EngineRuntimeScriptWithPlugins(NewPluginRegistry()); !bytes.Equal(want, got) {
		t.Fatalf("empty registry must be byte-identical: len want=%d got=%d", len(want), len(got))
	}
	if got := EngineRuntimeScriptWithPlugins(nil); !bytes.Equal(want, got) {
		t.Fatalf("nil registry must be byte-identical: len want=%d got=%d", len(want), len(got))
	}
}

// A registered plugin island's bytes must appear in the engine bundle, after the
// built-in islands (built-ins keep their order; plugins append).
func TestPluginIslandAppearsAfterBuiltins(t *testing.T) {
	marker := []byte("/*__test_island_marker__*/window.GoSXTestRuntime={};")
	reg := NewPluginRegistry()
	reg.Register(core.Plugin{Key: "test", Islands: []core.PluginIsland{testIsland("test-runtime.js", marker)}})

	bundle := EngineRuntimeScriptWithPlugins(reg)
	if !bytes.Contains(bundle, marker) {
		t.Fatal("plugin island bytes must appear in the engine bundle")
	}
	builtin := []byte("GoSXStudioInlineEditRuntime")
	if !bytes.Contains(bundle, builtin) {
		t.Fatal("built-in islands must remain present in the bundle")
	}
	if bytes.Index(bundle, builtin) > bytes.Index(bundle, marker) {
		t.Fatal("built-in islands must precede appended plugin islands")
	}
}

// Each registered island must be independently servable as JavaScript at its
// ScriptName.
func TestPluginIslandRouteServesJavaScript(t *testing.T) {
	marker := []byte("window.GoSXTestRuntime={};")
	reg := NewPluginRegistry()
	reg.Register(core.Plugin{Key: "test", Islands: []core.PluginIsland{testIsland("test-runtime.js", marker)}})

	h, ok := reg.IslandRoutes()["test-runtime.js"]
	if !ok {
		t.Fatal("IslandRoutes must serve the island at its ScriptName")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test-runtime.js", nil))
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=utf-8" {
		t.Fatalf("island must be served as JavaScript, got %q", ct)
	}
	if !bytes.Contains(rec.Body.Bytes(), marker) {
		t.Fatal("island route must serve the island bytes")
	}
}

// Re-registering the same ScriptName replaces in place (idempotent), matching
// template dedupe semantics; the later registration wins.
func TestPluginRegistryDedupesByScriptName(t *testing.T) {
	reg := NewPluginRegistry()
	reg.Register(core.Plugin{Islands: []core.PluginIsland{testIsland("dup.js", []byte("v1"))}})
	reg.Register(core.Plugin{Islands: []core.PluginIsland{testIsland("dup.js", []byte("v2"))}})
	if n := len(reg.Islands()); n != 1 {
		t.Fatalf("duplicate ScriptName must replace in place, got %d islands", n)
	}
	if got := string(reg.IslandBundles()[0]); got != "v2" {
		t.Fatalf("later registration must win, got %q", got)
	}
}

// Islands with no ScriptName or a nil/empty Bundle are dropped on registration.
func TestPluginRegistryDropsInvalidIslands(t *testing.T) {
	reg := NewPluginRegistry()
	reg.Register(core.Plugin{Islands: []core.PluginIsland{
		{ScriptName: "", Bundle: func() []byte { return []byte("x") }}, // no name
		{ScriptName: "nil-bundle.js"},                                  // nil bundle
		{ScriptName: "empty.js", Bundle: func() []byte { return nil }}, // empty bundle
	}})
	if n := len(reg.Islands()); n != 0 {
		t.Fatalf("invalid islands must be dropped, got %d", n)
	}
	if got := EngineRuntimeScriptWithPlugins(reg); !bytes.Equal(got, EngineRuntimeScript()) {
		t.Fatal("a registry holding only invalid islands must stay byte-identical")
	}
}

// A nil receiver is safe (defensive: hosts may pass a zero registry).
func TestNilPluginRegistryIsSafe(t *testing.T) {
	var reg *PluginRegistry
	reg.Register(core.Plugin{Islands: []core.PluginIsland{testIsland("x.js", []byte("y"))}})
	if reg.Islands() != nil || reg.IslandBundles() != nil || reg.IslandRoutes() != nil {
		t.Fatal("nil registry accessors must return nil")
	}
}
