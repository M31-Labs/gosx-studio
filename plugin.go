package studio

// Plugin is a compiled-in bundle of editor-surfacing artifacts. The wedge keeps
// this minimal: a plugin contributes ComponentTemplates that appear in the
// editor's block palette once registered into a host SiteMap's Library. There is
// no dynamic loader, manifest, or interface — plugins are linked into the binary.
//
// Feature, ResourceAdapters, and RuntimeContracts are reserved for forward
// compatibility (so later tasks can attach a plugin's feature surface and
// runtime wiring) but are unused by registration today.
type Plugin struct {
	Key       string
	Label     string
	Summary   string
	Templates []ComponentTemplate

	// Forward-compat (optional, unused by RegisterPlugin):
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
