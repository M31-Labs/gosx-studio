package studio

import "net/http"

// PluginIsland is a plugin-contributed client runtime island. It is bundled into
// the served engine script exactly like the built-in *runtime.Bundle() blobs and
// is also served at its own script route. This is the channel that lets a plugin
// ship browser behavior — e.g. a Stripe Payment Element island that publishes
// window.GoSXStripeRuntime — without forking the editor core. A plugin stays
// compiled-in (no dynamic loader); per-tenant enablement is a FeatureFlags
// concern via FeatureFlagKey.
type PluginIsland struct {
	// FeatureFlagKey gates per-tenant enablement, mirroring the built-in island
	// convention (e.g. inlineeditruntime.FeatureFlagKey). By convention it ends
	// with "-runtime-islands".
	FeatureFlagKey string
	// ScriptName is the served filename and the dedupe key, e.g.
	// "stripe-runtime.js".
	ScriptName string
	// Bundle returns the island IIFE — the JS that publishes the island's
	// window.GoSX*Runtime global. It is called lazily so an island can embed its
	// source like the built-ins.
	Bundle func() []byte
	// Contract couples the island's shipped JS to the typed RuntimeContract it
	// implements, so a plugin's advertised window.* API cannot drift from the
	// blob it ships. Reserved for the served runtime manifest; unused by bundling.
	Contract RuntimeContract
}

// scriptBytes returns the island's bundle bytes, tolerating a nil Bundle.
func (i PluginIsland) scriptBytes() []byte {
	if i.Bundle == nil {
		return nil
	}
	return i.Bundle()
}

// PluginRegistry accumulates the runtime artifacts contributed by registered
// plugins. It is host-held, not global: a plugin-aware host builds one, calls
// Register for each plugin, then serves EngineRuntimeScriptWithPlugins(reg) and
// mounts IslandRoutes(). An empty (or nil) registry yields byte-identical
// built-in output, so hosts that do not register plugin islands are unaffected.
//
// This is the keystone of the Plugin v2 contract: the island channel. Later
// channels (server routes, namespaced mutations, public renderers, config
// surfaces) accumulate on the same registry.
type PluginRegistry struct {
	islands []PluginIsland
	index   map[string]int // ScriptName -> position, for idempotent replace
}

// NewPluginRegistry returns an empty registry ready for Register.
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{index: map[string]int{}}
}

// Register accumulates a plugin's runtime islands. Islands are deduped by
// ScriptName: a later registration of the same ScriptName REPLACES the earlier
// one in place (preserving order), mirroring CompositionLibrary.RegisterPlugin
// template semantics, so re-registering a plugin is idempotent and config
// updates take effect. Islands with an empty ScriptName or a nil/empty Bundle
// are dropped, matching template normalization.
func (r *PluginRegistry) Register(p Plugin) {
	if r == nil {
		return
	}
	if r.index == nil {
		r.index = make(map[string]int, len(p.Islands))
	}
	for _, island := range p.Islands {
		if island.ScriptName == "" || len(island.scriptBytes()) == 0 {
			continue
		}
		if i, ok := r.index[island.ScriptName]; ok {
			r.islands[i] = island
			continue
		}
		r.index[island.ScriptName] = len(r.islands)
		r.islands = append(r.islands, island)
	}
}

// Islands returns the registered islands in registration order.
func (r *PluginRegistry) Islands() []PluginIsland {
	if r == nil {
		return nil
	}
	return r.islands
}

// IslandBundles returns each registered island's bundle bytes, in registration
// order, skipping empties. EngineRuntimeScriptWithPlugins appends these after
// the built-in island slice.
func (r *PluginRegistry) IslandBundles() [][]byte {
	if r == nil {
		return nil
	}
	out := make([][]byte, 0, len(r.islands))
	for _, island := range r.islands {
		if b := island.scriptBytes(); len(b) > 0 {
			out = append(out, b)
		}
	}
	return out
}

// IslandRoutes returns one http.Handler per registered island, keyed by
// ScriptName, each serving the island bundle as JavaScript via ScriptHandler. A
// plugin-aware host mounts these alongside the engine bundle so an island can be
// loaded on its own URL (mirroring how built-in runtimes are served).
func (r *PluginRegistry) IslandRoutes() map[string]http.Handler {
	if r == nil {
		return nil
	}
	routes := make(map[string]http.Handler, len(r.islands))
	for _, island := range r.islands {
		b := island.scriptBytes()
		if len(b) == 0 {
			continue
		}
		routes[island.ScriptName] = ScriptHandler(island.ScriptName, b)
	}
	return routes
}
