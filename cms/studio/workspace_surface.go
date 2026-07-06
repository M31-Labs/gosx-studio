package studio

import (
	"fmt"
	"strings"

	"m31labs.dev/gosx-admin/workbench"
)

type WorkspaceSurface struct {
	SiteNav         []SiteNavItem
	Links           []PanelLink
	Commands        []Command
	InspectorFields []InspectorField
	ResourceCount   int
	ToolCount       int
}

type WorkspaceSurfaceOptions struct {
	HomeKey       string
	HomeLabel     string
	HomeHref      string
	HomeSummary   string
	HomeActive    bool
	ResourceGroup string
	ToolGroup     string
	MediaListID   string
	Disabled      bool
}

func BuildWorkspaceSurface(workspace workbench.Workspace, options WorkspaceSurfaceOptions) WorkspaceSurface {
	homeKey := normalizeKey(firstNonEmpty(options.HomeKey, "home"))
	homeLabel := firstNonEmpty(strings.TrimSpace(options.HomeLabel), "Home")
	homeHref := firstNonEmpty(strings.TrimSpace(options.HomeHref), "/admin/editor")
	homeSummary := firstNonEmpty(strings.TrimSpace(options.HomeSummary), "Live canvas and site structure.")
	surface := WorkspaceSurface{
		SiteNav: []SiteNavItem{{
			Key:     homeKey,
			Label:   homeLabel,
			Href:    homeHref,
			Summary: homeSummary,
			Active:  options.HomeActive,
		}},
		Commands: []Command{{
			Kind:     CommandLink,
			Key:      "open-" + homeKey,
			Label:    "Open " + lowerFirst(homeLabel),
			Summary:  homeSummary,
			Group:    firstNonEmpty(strings.TrimSpace(options.ResourceGroup), "Resources"),
			Href:     homeHref,
			Keywords: []string{homeKey, homeLabel},
		}},
	}
	for _, resource := range workspace.Resources {
		resource = normalizeWorkspaceResource(resource)
		if resource.Slug == "" || resource.Route == "" {
			continue
		}
		surface.ResourceCount++
		summary := workspaceResourceSummary(resource)
		surface.SiteNav = append(surface.SiteNav, SiteNavItem{
			Key:     resource.Slug,
			Label:   resource.Label,
			Href:    resource.Route,
			Summary: summary,
		})
		surface.Links = append(surface.Links, PanelLink{
			Key:     resource.Slug,
			Label:   resource.Label,
			Summary: summary,
			Href:    resource.Route,
		})
		surface.Commands = append(surface.Commands, Command{
			Kind:     CommandLink,
			Key:      "open-" + resource.Slug,
			Label:    "Open " + resource.Label,
			Summary:  summary,
			Group:    firstNonEmpty(strings.TrimSpace(options.ResourceGroup), "Resources"),
			Href:     resource.Route,
			Keywords: workspaceResourceKeywords(resource),
		})
		surface.InspectorFields = append(surface.InspectorFields, workspaceResourceInspectorFields(resource, options)...)
	}
	for _, tool := range workspace.Tools {
		tool = normalizeWorkspaceTool(tool)
		if tool.Slug == "" || tool.Route == "" {
			continue
		}
		surface.ToolCount++
		summary := workspaceToolSummary(tool)
		surface.Links = append(surface.Links, PanelLink{
			Key:     "tool-" + tool.Slug,
			Label:   tool.Label,
			Summary: summary,
			Href:    tool.Route,
		})
		surface.Commands = append(surface.Commands, Command{
			Kind:     CommandLink,
			Key:      "open-tool-" + tool.Slug,
			Label:    "Open " + tool.Label,
			Summary:  summary,
			Group:    firstNonEmpty(strings.TrimSpace(options.ToolGroup), "Tools"),
			Href:     tool.Route,
			Keywords: workspaceToolKeywords(tool),
		})
		surface.InspectorFields = append(surface.InspectorFields, workspaceToolInspectorField(tool))
	}
	surface.Commands = normalizeCommands(surface.Commands)
	return surface
}

func normalizeWorkspaceResource(resource workbench.Resource) workbench.Resource {
	resource.Slug = normalizeWorkspaceKey(resource.Slug)
	resource.Label = strings.TrimSpace(resource.Label)
	resource.Singular = strings.TrimSpace(resource.Singular)
	resource.Description = strings.TrimSpace(resource.Description)
	resource.Route = strings.TrimSpace(resource.Route)
	if resource.Label == "" {
		resource.Label = titleFromWorkspaceKey(resource.Slug)
	}
	return resource
}

func normalizeWorkspaceTool(tool workbench.Tool) workbench.Tool {
	tool.Slug = normalizeWorkspaceKey(tool.Slug)
	tool.Label = strings.TrimSpace(tool.Label)
	tool.Description = strings.TrimSpace(tool.Description)
	tool.Route = strings.TrimSpace(tool.Route)
	tool.Kind = strings.TrimSpace(tool.Kind)
	if tool.Label == "" {
		tool.Label = titleFromWorkspaceKey(tool.Slug)
	}
	return tool
}

func workspaceResourceSummary(resource workbench.Resource) string {
	summary := strings.TrimSpace(resource.Description)
	count := fmt.Sprintf("%d records", resource.Count)
	if resource.Count == 1 {
		count = "1 record"
	}
	if summary == "" {
		return count
	}
	return strings.TrimRight(summary, ".") + "; " + count + "."
}

func workspaceToolSummary(tool workbench.Tool) string {
	summary := strings.TrimSpace(tool.Description)
	if summary != "" {
		return summary
	}
	if tool.Kind != "" {
		return "Runtime tool: " + tool.Kind + "."
	}
	return "Runtime tool."
}

func workspaceResourceInspectorFields(resource workbench.Resource, options WorkspaceSurfaceOptions) []InspectorField {
	scope := "resource-" + resource.Slug
	fields := []InspectorField{{
		Kind:           InspectorFieldCard,
		Label:          resource.Label,
		CardTitle:      workspaceResourceInspectorTitle(resource),
		Help:           workspaceResourceInspectorHelp(resource),
		ContainerAttrs: SelectionScopeAttrs(scope),
		Attrs: []FieldAttribute{
			{Name: "data-studio-field-source", Value: "resource." + resource.Slug},
			{Name: "data-studio-resource", Value: resource.Slug},
			{Name: "data-studio-resource-capabilities", Value: strings.Join(resource.Capabilities, " ")},
			{Name: "data-studio-resource-actions", Value: workspaceActionKeySummary(resource.Actions)},
		},
		Actions: []FieldAction{{Label: "Open " + resource.Label, Href: resource.Route, Primary: true}},
	}}
	fields = append(fields, WorkbenchFieldInspectorFields(resource.Fields, WorkbenchFieldInspectorOptions{
		IDPrefix:       "resource" + workspaceFieldPrefix(resource.Slug),
		NamePrefix:     "resource" + workspaceFieldPrefix(resource.Slug),
		SourcePrefix:   "resource." + resource.Slug + ".",
		MediaListID:    options.MediaListID,
		Overrides:      workspaceResourceFieldOverrides(resource.Fields),
		ContainerAttrs: SelectionScopeAttrs(scope),
		Disabled:       options.Disabled,
	})...)
	if len(resource.Actions) > 0 {
		fields = append(fields, InspectorField{
			Kind:           InspectorFieldCard,
			Label:          "Actions",
			CardTitle:      resource.Label + " actions",
			Help:           workspaceActionSummary(resource.Actions),
			ContainerAttrs: SelectionScopeAttrs(scope),
			Attrs: []FieldAttribute{
				{Name: "data-studio-field-source", Value: "resource." + resource.Slug + ".actions"},
				{Name: "data-studio-resource", Value: resource.Slug},
				{Name: "data-studio-resource-actions", Value: workspaceActionKeySummary(resource.Actions)},
			},
		})
	}
	return fields
}

func workspaceToolInspectorField(tool workbench.Tool) InspectorField {
	return InspectorField{
		Kind:           InspectorFieldCard,
		Label:          tool.Label,
		CardTitle:      workspaceToolInspectorTitle(tool),
		Help:           workspaceToolInspectorHelp(tool),
		ContainerAttrs: SelectionScopeAttrs("tool-" + tool.Slug),
		Attrs: []FieldAttribute{
			{Name: "data-studio-field-source", Value: "tool." + tool.Slug},
			{Name: "data-studio-tool", Value: tool.Slug},
			{Name: "data-studio-tool-kind", Value: tool.Kind},
			{Name: "data-studio-tool-actions", Value: workspaceActionKeySummary(tool.Actions)},
		},
		Actions: []FieldAction{{Label: "Open " + tool.Label, Href: tool.Route, Primary: true}},
	}
}

func workspaceResourceFieldOverrides(fields []workbench.Field) map[string]InspectorFieldOverride {
	overrides := map[string]InspectorFieldOverride{}
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		overrides[name] = InspectorFieldOverride{
			Help: workspaceResourceFieldHelp(field),
			Attrs: []FieldAttribute{
				{Name: "data-studio-workspace-field-kind", Value: string(field.Kind)},
			},
		}
	}
	return overrides
}

func workspaceResourceInspectorTitle(resource workbench.Resource) string {
	singular := strings.TrimSpace(resource.Singular)
	if singular == "" {
		singular = resource.Label
	}
	return singular + " schema"
}

func workspaceResourceInspectorHelp(resource workbench.Resource) string {
	parts := []string{
		workspaceResourceSummary(resource),
		fmt.Sprintf("%d fields", len(resource.Fields)),
	}
	if len(resource.Actions) > 0 {
		parts = append(parts, fmt.Sprintf("%d actions", len(resource.Actions)))
	}
	if len(resource.Capabilities) > 0 {
		parts = append(parts, "Capabilities: "+strings.Join(resource.Capabilities, ", "))
	}
	return strings.Join(compactWorkspaceStrings(parts), " ")
}

func workspaceToolInspectorTitle(tool workbench.Tool) string {
	if tool.Kind != "" {
		return titleFromWorkspaceKey(tool.Kind)
	}
	return "Runtime tool"
}

func workspaceToolInspectorHelp(tool workbench.Tool) string {
	parts := []string{workspaceToolSummary(tool)}
	if len(tool.Actions) > 0 {
		parts = append(parts, "Actions: "+workspaceActionSummary(tool.Actions))
	}
	return strings.Join(compactWorkspaceStrings(parts), " ")
}

func workspaceResourceFieldHelp(field workbench.Field) string {
	parts := []string{string(field.Kind)}
	if field.Required {
		parts = append(parts, "required")
	} else {
		parts = append(parts, "optional")
	}
	if field.ReadOnly {
		parts = append(parts, "read-only")
	}
	if len(field.Options) > 0 {
		parts = append(parts, "options: "+strings.Join(field.Options, ", "))
	}
	return strings.Join(compactWorkspaceStrings(parts), "; ")
}

func workspaceActionSummary(actions []workbench.Action) string {
	out := make([]string, 0, len(actions))
	for _, action := range actions {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			label = titleFromWorkspaceKey(action.Name)
		}
		if label == "" {
			continue
		}
		kind := strings.TrimSpace(action.Kind)
		if kind != "" {
			label += " (" + kind + ")"
		}
		out = append(out, label)
	}
	return strings.Join(out, "; ")
}

func workspaceActionKeySummary(actions []workbench.Action) string {
	out := make([]string, 0, len(actions))
	for _, action := range actions {
		name := normalizeWorkspaceKey(action.Name)
		if name == "" {
			continue
		}
		kind := strings.TrimSpace(action.Kind)
		if kind != "" {
			name += ":" + kind
		}
		out = append(out, name)
	}
	return strings.Join(out, " ")
}

func workspaceResourceKeywords(resource workbench.Resource) []string {
	keywords := []string{resource.Slug, resource.Singular, strings.Join(resource.Capabilities, " ")}
	for _, field := range resource.Fields {
		keywords = append(keywords, field.Name, field.Label, string(field.Kind))
	}
	for _, action := range resource.Actions {
		keywords = append(keywords, action.Name, action.Label, action.Kind)
	}
	return compactWorkspaceStrings(keywords)
}

func workspaceToolKeywords(tool workbench.Tool) []string {
	keywords := []string{tool.Slug, tool.Kind}
	for _, action := range tool.Actions {
		keywords = append(keywords, action.Name, action.Label, action.Kind)
	}
	return compactWorkspaceStrings(keywords)
}

func workspaceFieldPrefix(value string) string {
	words := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(value, "_", " "), "-", " "))
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, "")
}

func normalizeWorkspaceKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	return value
}

func titleFromWorkspaceKey(value string) string {
	words := strings.Fields(strings.ReplaceAll(normalizeWorkspaceKey(value), "-", " "))
	for i, word := range words {
		if word == "" {
			continue
		}
		words[i] = strings.ToUpper(word[:1]) + word[1:]
	}
	return strings.Join(words, " ")
}

func compactWorkspaceStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
