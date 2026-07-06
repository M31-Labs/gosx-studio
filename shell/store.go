package shell

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/core"
	"m31labs.dev/gosx-studio/sitemap"
)

// The Store interface and the cms/admin summary converters live in
// cmsbridge.go (the single seam where shell touches gosx-cms/gosx-admin types,
// per the restructure spec §1 "Shell summaries vs cms/admin types" seam).
// Everything below is pure shell chrome.

// HostShell composes a Store and a ShellConfig into the editor surface.
//
// Batch 1.5 ships a thin wrapper that adapts a Store + ShellConfig into the
// foundation New(Options) constructor. Later batches will replace this with
// the full surface composition (chrome / canvas / panels / etc.) once those
// surfaces land in gosx-studio.
func HostShell(store Store, shell ShellConfig) Shell {
	normalized := shell.Normalize()
	adapters := normalized.Adapters
	surface := authoring.NoCodeAuthoringSurface(normalized.SiteMap)
	opts := Options{
		Title:         strings.TrimSpace(normalized.Labels.EditorTitle),
		PreviewURL:    hostShellPreviewURL(normalized),
		SaveAction:    hostShellActionHref(normalized.Actions, "save"),
		RestoreAction: hostShellActionHref(normalized.Actions, "restore"),
		Navigation:    hostShellSections(normalized.Resources),
		Metrics:       hostShellMetrics(normalized.SiteMap),
		Modes:         hostShellModes(normalized.Modes),
		Canvas:        hostShellCanvas(normalized.SiteMap, normalized.Labels.EditorTitle),
		Actions:       hostShellActions(normalized.Actions),
		Extras: map[string]any{
			"resources": hostShellResourceViews(normalized.Resources),
			"authoring": authoring.AuthoringSurfaceView(surface),
			"siteMap":   sitemap.AuthoringSiteMapView(surface, sitemap.SiteMapViewOptions{}),
		},
	}
	if store != nil {
		opts.Left = store.LeftPanels()
		opts.Right = store.RightPanels()
		opts.Media = store.MediaAssets()
		opts.Revisions = store.Revisions()
		opts.Readiness = store.Readiness()
		if storeAdapters := normalizeResourceAdapters(store.Adapters(), nil); len(storeAdapters) > 0 {
			adapters = storeAdapters
		}
	}
	opts.Extras["adapters"] = hostShellAdapterViews(adapters)
	return New(opts)
}

// FeatureFlagAttrs returns the data-* attribute map for runtime-island feature
// flags enabled on the given ShellConfig.
func FeatureFlagAttrs(shell ShellConfig) map[string]string {
	out := make(map[string]string)
	for key, enabled := range shell.Normalize().FeatureFlags {
		if !enabled {
			continue
		}
		if !strings.HasSuffix(key, "-runtime-islands") {
			continue
		}
		out["data-gosx-studio-feature-flag-"+key] = "true"
	}
	return out
}

func hostShellPreviewURL(shell ShellConfig) string {
	if page, ok := shell.SiteMap.SelectedPage(); ok && strings.TrimSpace(page.Route) != "" {
		return page.Route
	}
	if resource, ok := shell.Resource("preview"); ok && resource.Href != "" {
		return resource.Href
	}
	if resource, ok := shell.Resource("public"); ok && resource.Href != "" {
		return resource.Href
	}
	return "/"
}

func hostShellActionHref(actions []ActionConfig, key string) string {
	key = core.NormalizeKey(key)
	for _, action := range actions {
		if core.NormalizeKey(action.Key) == key {
			return strings.TrimSpace(action.Href)
		}
	}
	return ""
}

func hostShellActions(actions []ActionConfig) []Action {
	out := make([]Action, 0, len(actions))
	for _, action := range actions {
		if strings.TrimSpace(action.Href) == "" {
			continue
		}
		out = append(out, Action{
			Key:     action.Key,
			Label:   action.Label,
			Href:    action.Href,
			Primary: action.Primary,
		})
	}
	return out
}

func hostShellSections(resources []ResourceConfig) []Section {
	out := make([]Section, 0, len(resources))
	for _, resource := range resources {
		section := Section{
			Key:     resource.Key,
			Label:   resource.Label,
			Summary: resource.Summary,
		}
		if resource.Href != "" {
			section.Actions = []Action{LinkAction("open-"+resource.Key, "Open", resource.Href)}
		}
		out = append(out, section)
	}
	return out
}

func hostShellMetrics(siteMap core.SiteMap) []Metric {
	siteMap = siteMap.Normalize()
	if len(siteMap.Pages) == 0 {
		return nil
	}
	return []Metric{
		NewMetric("pages", "Pages", len(siteMap.Pages)),
		NewMetric("sections", "Sections", siteMap.ComponentCount()),
		NewMetric("controls", "Controls", siteMap.ControlCount()),
	}
}

func hostShellModes(modes []ModeConfig) []Mode {
	out := make([]Mode, 0, len(modes))
	for index, mode := range modes {
		out = append(out, NewMode(mode.Key, mode.Label, index == 0))
	}
	return out
}

func hostShellCanvas(siteMap core.SiteMap, fallbackTitle string) CanvasSurface {
	page, ok := siteMap.SelectedPage()
	if !ok {
		return CanvasSurface{RouteLabel: fallbackTitle, SelectionLabel: "No selection", Zoom: "fit"}
	}
	return CanvasSurface{
		RouteLabel:     page.Label,
		SelectionLabel: hostShellSectionCountLabel(page.ComponentCount()),
		Zoom:           "fit",
	}
}

func hostShellSectionCountLabel(count int) string {
	switch count {
	case 0:
		return "No editable sections"
	case 1:
		return "1 editable section"
	default:
		return fmt.Sprintf("%d editable sections", count)
	}
}

func hostShellResourceViews(resources []ResourceConfig) []map[string]any {
	out := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		out = append(out, map[string]any{
			"key":        resource.Key,
			"label":      resource.Label,
			"kind":       resource.Kind,
			"href":       resource.Href,
			"summary":    resource.Summary,
			"hasHref":    resource.Href != "",
			"hasSummary": resource.Summary != "",
		})
	}
	return out
}

func hostShellAdapterViews(adapters []core.ResourceAdapter) []map[string]any {
	out := make([]map[string]any, 0, len(adapters))
	for _, adapter := range adapters {
		adapter = adapter.Normalize()
		if adapter.Label == "" {
			continue
		}
		out = append(out, map[string]any{
			"kind":         string(adapter.Kind),
			"label":        adapter.Label,
			"summary":      adapter.Summary,
			"surface":      string(adapter.Surface),
			"capabilities": hostShellResourceCapabilities(adapter.Capabilities),
			"bindings":     hostShellBindingViews(adapter.Bindings),
		})
	}
	return out
}

func hostShellResourceCapabilities(capabilities []core.ResourceCapability) []string {
	out := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		if value := strings.TrimSpace(string(capability)); value != "" {
			out = append(out, value)
		}
	}
	return out
}

func hostShellBindingViews(bindings []core.ResourceBinding) []map[string]any {
	out := make([]map[string]any, 0, len(bindings))
	for _, binding := range bindings {
		binding.Key = strings.TrimSpace(binding.Key)
		binding.Label = strings.TrimSpace(binding.Label)
		binding.Summary = strings.TrimSpace(binding.Summary)
		binding.Binding = strings.TrimSpace(binding.Binding)
		if binding.Label == "" {
			continue
		}
		out = append(out, map[string]any{
			"key":     binding.Key,
			"label":   binding.Label,
			"summary": binding.Summary,
			"binding": binding.Binding,
		})
	}
	return out
}
