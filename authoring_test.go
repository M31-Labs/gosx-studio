package studio

import "testing"

func TestNoCodeAuthoringSurfaceBuildsDraftOperations(t *testing.T) {
	siteMap := SiteMap{Pages: []Page{
		{
			Key:           "home",
			Label:         "Home",
			Route:         "/",
			Group:         PageGroupSite,
			GoSXComponent: "HomePage",
			Selected:      true,
			Components: []Component{
				{
					Key:           "hero",
					Label:         "Hero",
					Summary:       "Lead message",
					GoSXComponent: "HomeHero",
					Source:        ComponentSourceHost,
					Binding:       "pages.home.hero",
					Editable:      true,
					Controls: []Control{
						{Key: "headline", Label: "Headline", Kind: ControlText, Binding: "pages.home.hero.headline", Value: "Forest days"},
					},
				},
			},
		},
	}, Library: CompositionLibrary{
		PageBlueprints: []PageBlueprint{{
			Key:           "landing",
			Label:         "Landing page",
			Summary:       "A focused page with a hero and signup block.",
			RoutePattern:  "/new-page",
			Group:         PageGroupContent,
			GoSXComponent: "LandingPage",
			Status:        "Ready",
			Components: []ComponentTemplate{
				{Key: "hero", Label: "Hero", GoSXComponent: "HomeHero", Source: ComponentSourceHost, DefaultBinding: "pages.new.hero"},
				{Key: "signup", Label: "Signup", GoSXComponent: "SignupFlow", Source: ComponentSourceCMS, DefaultBinding: "flows.signup"},
			},
		}},
		ComponentTemplates: []ComponentTemplate{{
			Key:            "gallery",
			Label:          "Gallery",
			Summary:        "Show selected images on the page.",
			Category:       "Media",
			GoSXComponent:  "ImageGallery",
			Source:         ComponentSourceCMS,
			DefaultBinding: "media.gallery",
			AddLabel:       "Add gallery",
			Controls: []Control{
				{Key: "source", Label: "Source", Kind: ControlMedia, Binding: "media.gallery"},
			},
		}},
	}}

	surface := NoCodeAuthoringSurface(siteMap)

	if !surface.HasSelectedPage || surface.SelectedPage.Key != "home" {
		t.Fatalf("selected page = %#v, has=%v", surface.SelectedPage, surface.HasSelectedPage)
	}
	if surface.IntentCount() != 2 {
		t.Fatalf("intent count = %d", surface.IntentCount())
	}
	createPage := surface.CreatePageIntents[0]
	if createPage.Key != "create-page:landing" || createPage.NormalizedKind() != CompositionIntentCreatePage || createPage.TargetRoute != "/new-page" {
		t.Fatalf("create-page intent = %#v", createPage)
	}
	if createPage.StepCount() != 3 || createPage.Steps[1].GoSXComponent != "HomeHero" || createPage.Steps[2].Binding != "flows.signup" {
		t.Fatalf("create-page steps = %#v", createPage.Steps)
	}
	addComponent := surface.AddComponentIntents[0]
	if addComponent.Key != "add-component:home:gallery" || addComponent.TargetPageKey != "home" || addComponent.TargetRoute != "/" {
		t.Fatalf("add-component intent = %#v", addComponent)
	}
	if addComponent.Label != "Add gallery" || addComponent.GoSXComponent != "ImageGallery" || addComponent.Binding != "media.gallery" {
		t.Fatalf("add-component details = %#v", addComponent)
	}
	if surface.Workspace.NodeCount() != 3 || surface.Workspace.LinkCount() != 2 || surface.Canvas.ViewBox != "0 0 100 100" {
		t.Fatalf("workspace = %#v canvas = %#v", surface.Workspace, surface.Canvas)
	}
}

func TestAuthoringSurfaceViewProjectsNoCodePlatformModel(t *testing.T) {
	surface := NoCodeAuthoringSurface(SiteMap{Pages: []Page{{
		Key:           "contact",
		Label:         "Contact",
		Route:         "/contact",
		Group:         PageGroupFlows,
		GoSXComponent: "ContactPage",
		Components: []Component{{
			Key:           "form",
			Label:         "Contact form",
			GoSXComponent: "ContactFlow",
			Source:        ComponentSourceCMS,
			Binding:       "flows.contact",
			Editable:      true,
			Controls: []Control{{
				Key:      "submit-label",
				Label:    "Submit label",
				Kind:     ControlText,
				Binding:  "flows.contact.submit.label",
				Required: true,
			}},
		}},
	}}, Library: CompositionLibrary{
		ComponentTemplates: []ComponentTemplate{{
			Key:           "faq",
			Label:         "FAQ",
			GoSXComponent: "FAQSection",
			Source:        ComponentSourceHost,
		}},
	}})

	view := AuthoringSurfaceView(surface)

	if view["pageCount"] != 1 || view["componentCount"] != 1 || view["controlCount"] != 1 || view["intentCount"] != 1 {
		t.Fatalf("authoring counts = %#v", view)
	}
	selected := view["selectedPage"].(map[string]any)
	if selected["key"] != "contact" || selected["groupLabel"] != "Flows" {
		t.Fatalf("selected page view = %#v", selected)
	}
	components := selected["components"].([]map[string]any)
	controls := components[0]["controls"].([]map[string]any)
	if components[0]["sourceLabel"] != "CMS" || controls[0]["kindLabel"] != "Text" || controls[0]["required"] != true {
		t.Fatalf("component/control views = %#v %#v", components[0], controls[0])
	}
	controlForm := controls[0]["formValues"].(map[string]string)
	if controlForm[AuthoringFieldOperation] != "save-control" || controlForm[AuthoringFieldControlKey] != "submit-label" || controlForm[AuthoringFieldComponentKey] != "form" {
		t.Fatalf("control mutation form = %#v", controlForm)
	}
	library := view["library"].(map[string]any)
	if library["templateCount"] != 1 {
		t.Fatalf("library view = %#v", library)
	}
	mutationFields := view["mutationFields"].(map[string]string)
	if mutationFields["operation"] != AuthoringFieldOperation || mutationFields["value"] != AuthoringFieldValue {
		t.Fatalf("mutation field names = %#v", mutationFields)
	}
	intents := view["addComponentIntents"].([]map[string]any)
	if len(intents) != 1 || intents[0]["key"] != "add-component:contact:faq" || intents[0]["targetRoute"] != "/contact" {
		t.Fatalf("intent views = %#v", intents)
	}
	intentForm := intents[0]["formValues"].(map[string]string)
	if intentForm[AuthoringFieldOperation] != "apply-intent" || intentForm[AuthoringFieldComponentTemplateKey] != "faq" || intentForm[AuthoringFieldPageKey] != "contact" {
		t.Fatalf("intent mutation form = %#v", intentForm)
	}
	workspace := view["workspace"].(map[string]any)
	if workspace["nodeCount"] != 3 || workspace["linkCount"] != 2 {
		t.Fatalf("workspace view = %#v", workspace)
	}
	canvas := workspace["canvas"].(map[string]any)
	if canvas["viewBox"] != "0 0 100 100" || len(canvas["nodes"].([]map[string]any)) != 3 {
		t.Fatalf("canvas view = %#v", canvas)
	}
}
