package studio

import (
	"testing"

	"m31labs.dev/gosx-admin/workbench"
)

func TestBuildWorkspaceSurfaceProjectsResourcesAndTools(t *testing.T) {
	surface := BuildWorkspaceSurface(workbench.Workspace{
		Resources: []workbench.Resource{{
			Slug:         "Products",
			Label:        "Products",
			Singular:     "Product",
			Description:  "Sellable inventory.",
			Route:        "/admin/products",
			Count:        2,
			Capabilities: []string{"list", "create"},
			Fields: []workbench.Field{
				{Name: "title", Label: "Title", Kind: workbench.FieldText, Required: true},
				{Name: "image", Label: "Image", Kind: workbench.FieldImage, ReadOnly: true},
			},
			Actions: []workbench.Action{{Name: "save", Label: "Save product", Kind: "form"}},
		}},
		Tools: []workbench.Tool{{
			Slug:        "GraphQL",
			Label:       "Headless GraphQL",
			Description: "Content API.",
			Route:       "/api/graphql",
			Kind:        "headless-api",
			Actions:     []workbench.Action{{Name: "query-content", Label: "Query content", Kind: "graphql-query"}},
		}},
	}, WorkspaceSurfaceOptions{
		HomeHref:    "/admin/editor",
		HomeActive:  true,
		MediaListID: "editor-media",
		Disabled:    true,
	})
	if surface.ResourceCount != 1 || surface.ToolCount != 1 {
		t.Fatalf("unexpected counts: %#v", surface)
	}
	if len(surface.SiteNav) != 2 || !surface.SiteNav[0].Active || surface.SiteNav[1].Key != "products" {
		t.Fatalf("unexpected site nav: %#v", surface.SiteNav)
	}
	if len(surface.Links) != 2 || surface.Links[0].Summary != "Sellable inventory; 2 records." || surface.Links[1].Key != "tool-graphql" {
		t.Fatalf("unexpected links: %#v", surface.Links)
	}
	commands := map[string]Command{}
	for _, command := range surface.Commands {
		commands[command.Key] = command
	}
	if commands["open-home"].Href != "/admin/editor" || commands["open-products"].Group != "Resources" || commands["open-tool-graphql"].Group != "Tools" {
		t.Fatalf("unexpected commands: %#v", surface.Commands)
	}
	if len(surface.InspectorFields) != 5 {
		t.Fatalf("expected resource card, two fields, resource actions, and tool card, got %#v", surface.InspectorFields)
	}
	productCard := surface.InspectorFields[0]
	if productCard.CardTitle != "Product schema" || fieldAttributeValue(productCard.Attrs, "data-studio-resource") != "products" || productCard.Actions[0].Href != "/admin/products" {
		t.Fatalf("unexpected product card: %#v", productCard)
	}
	titleField := surface.InspectorFields[1]
	if titleField.ID != "resourceProductsTitle" || titleField.Name != "resourceProductsTitle" || !titleField.Required || !titleField.Disabled {
		t.Fatalf("unexpected title field: %#v", titleField)
	}
	if fieldAttributeValue(titleField.Attrs, "data-studio-workspace-field-kind") != "text" {
		t.Fatalf("expected field kind attr on title field: %#v", titleField.Attrs)
	}
	imageField := surface.InspectorFields[2]
	if fieldAttributeValue(imageField.Attrs, "list") != "editor-media" || fieldAttributeValue(imageField.Attrs, "data-studio-field-editable") != "media" {
		t.Fatalf("expected media field attrs: %#v", imageField.Attrs)
	}
	toolCard := surface.InspectorFields[4]
	if toolCard.CardTitle != "Headless Api" || fieldAttributeValue(toolCard.Attrs, "data-studio-tool") != "graphql" {
		t.Fatalf("unexpected tool card: %#v", toolCard)
	}
}

func TestBuildWorkspaceSurfaceSkipsIncompleteDescriptors(t *testing.T) {
	surface := BuildWorkspaceSurface(workbench.Workspace{
		Resources: []workbench.Resource{{Slug: "missing-route", Label: "Missing route"}},
		Tools:     []workbench.Tool{{Slug: "missing-route", Label: "Missing route"}},
	}, WorkspaceSurfaceOptions{})
	if surface.ResourceCount != 0 || surface.ToolCount != 0 || len(surface.Links) != 0 {
		t.Fatalf("expected incomplete descriptors to be skipped: %#v", surface)
	}
	if len(surface.SiteNav) != 1 || len(surface.Commands) != 1 {
		t.Fatalf("expected default home surface only: %#v", surface)
	}
}
