package showcase3d

import (
	"bytes"
	"testing"

	studio "m31labs.dev/gosx-studio"
)

// End-to-end proof of the Plugin v2 island channel: registering the real
// showcase3d plugin makes its island appear in the served engine bundle and at
// its own route — the plugin -> island -> served-bundle path the whole program
// depends on.
func TestPluginContributesIslandToEngineBundle(t *testing.T) {
	reg := studio.NewPluginRegistry()
	reg.Register(Plugin())

	bundle := studio.EngineRuntimeScriptWithPlugins(reg)
	if !bytes.Contains(bundle, []byte("GoSXShowcase3DRuntime")) {
		t.Fatal("showcase3d plugin island must appear in the engine bundle after registration")
	}

	if _, ok := reg.IslandRoutes()[IslandScriptName]; !ok {
		t.Fatalf("showcase3d island must be served at %q", IslandScriptName)
	}
}

// Registering the plugin must not perturb the built-in bundle: the built-ins are
// still present and the showcase island is strictly appended.
func TestShowcase3DIslandIsAppendedNotInjected(t *testing.T) {
	reg := studio.NewPluginRegistry()
	reg.Register(Plugin())
	bundle := studio.EngineRuntimeScriptWithPlugins(reg)

	builtin := bytes.Index(bundle, []byte("GoSXStudioInlineEditRuntime"))
	island := bytes.Index(bundle, []byte("GoSXShowcase3DRuntime"))
	if builtin < 0 {
		t.Fatal("built-in islands must remain in the bundle")
	}
	if island < builtin {
		t.Fatal("plugin island must be appended after the built-in islands")
	}
}
