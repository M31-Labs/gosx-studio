package core

// Plugin is a compiled-in bundle of editor-surfacing artifacts. A plugin
// contributes ComponentTemplates that appear in the editor's block palette
// (CompositionLibrary.RegisterPlugin) and, on the Plugin v2 island channel,
// client runtime Islands that are bundled and served alongside the built-in
// runtimes (PluginRegistry.Register). Plugins are linked into the binary — there
// is no dynamic loader; per-tenant enablement is a FeatureFlags concern.
//
// Feature, ResourceAdapters, and RuntimeContracts remain reserved for the
// forthcoming Plugin v2 channels (feature/adapter merge + served runtime
// manifest); Islands is live today via PluginRegistry.
//
// Plugin lives in core (rather than moving with PluginRegistry to
// hostruntime in a later slice) because RegisterPlugin extends
// CompositionLibrary, and Go forbids defining new methods on a type alias to
// another package's type — RegisterPlugin must live alongside
// CompositionLibrary's real definition. PluginRegistry itself (the
// http.Handler/ScriptHandler-serving island registry) stays at the studio
// root for now; it is deferred to the hostruntime slice.
type Plugin struct {
	Key       string
	Label     string
	Summary   string
	Templates []ComponentTemplate

	// Plugin v2 island channel (live via PluginRegistry.Register):
	Islands []PluginIsland

	// Forward-compat (optional, unused by RegisterPlugin / PluginRegistry today):
	Feature          Feature
	ResourceAdapters []ResourceAdapter
	RuntimeContracts []RuntimeContract
}

// RegisterPlugin surfaces a plugin's ComponentTemplates in the editor block
// palette by adding them to the library's ComponentTemplates.
//
// Templates are normalized on registration (reusing ComponentTemplate.Normalize)
// and deduped by Key: a later registration of the same template Key REPLACES the
// earlier one in place (preserving order), so re-registering a plugin is
// idempotent and config updates take effect. Templates that fail normalization
// (empty Key/Label/GoSXComponent) are dropped, matching CompositionLibrary
// normalization.
func (l *CompositionLibrary) RegisterPlugin(p Plugin) {
	if l == nil {
		return
	}
	index := make(map[string]int, len(l.ComponentTemplates))
	for i, existing := range l.ComponentTemplates {
		index[existing.Key] = i
	}
	for _, template := range p.Templates {
		template = template.Normalize()
		if template.Key == "" || template.Label == "" || template.GoSXComponent == "" {
			continue
		}
		if i, ok := index[template.Key]; ok {
			l.ComponentTemplates[i] = template
			continue
		}
		index[template.Key] = len(l.ComponentTemplates)
		l.ComponentTemplates = append(l.ComponentTemplates, template)
	}
}

// PluginIsland is a plugin-contributed client runtime island. It is bundled into
// the served engine script exactly like the built-in *runtime.Bundle() blobs and
// is also served at its own script route. This is the channel that lets a plugin
// ship browser behavior — e.g. a Stripe Payment Element island that publishes
// window.GoSXStripeRuntime — without forking the editor core. A plugin stays
// compiled-in (no dynamic loader); per-tenant enablement is a FeatureFlags
// concern via FeatureFlagKey.
//
// PluginIsland is a pure data type (no host/UI imports), so it lives here in
// core alongside Plugin. PluginRegistry, which serves islands over HTTP, stays
// at the studio root pending the hostruntime slice.
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

// ScriptBytes returns the island's bundle bytes, tolerating a nil Bundle. It is
// exported (unlike its studio-root predecessor scriptBytes) so PluginRegistry,
// which stays at the studio root, can call it across the package boundary.
func (i PluginIsland) ScriptBytes() []byte {
	if i.Bundle == nil {
		return nil
	}
	return i.Bundle()
}
