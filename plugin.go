package studio

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
