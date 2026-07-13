package studio

import (
	"strings"
	"testing"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-admin/blockstudio"
	"m31labs.dev/gosx-admin/calendar"
	"m31labs.dev/gosx-admin/workbench"
	"m31labs.dev/gosx-studio/cms/lifecycle"
)

func TestRenderSiteNavigator(t *testing.T) {
	html := gosx.RenderHTML(RenderSiteNavigator([]SiteNavItem{
		{Key: "home", Label: "Home", Href: "/admin/editor", Active: true},
		{Key: "shop", Label: "Shop", Href: "/admin/products"},
	}, SiteNavigatorOptions{Class: "studio-nav-panel", Kicker: "Site", Title: "Pages"}))
	if !strings.Contains(html, `class="studio-nav-panel"`) || !strings.Contains(html, `data-studio-mode-panel="structure"`) {
		t.Fatalf("expected panel hooks, got: %s", html)
	}
	if !strings.Contains(html, `data-studio-site-page="home"`) || !strings.Contains(html, `class="is-active"`) || !strings.Contains(html, `data-gosx-link="true"`) {
		t.Fatalf("expected site nav link hooks, got: %s", html)
	}
}

func TestRenderInspectorHeaderAndScopeStrip(t *testing.T) {
	head := gosx.RenderHTML(RenderInspectorHeader(InspectorHeaderOptions{
		Class:          "studio-inspector-head",
		ModeLabel:      "Style",
		SelectionLabel: "Hero",
	}))
	if !strings.Contains(head, `data-studio-mode-label="true"`) || !strings.Contains(head, `data-studio-selection-label="true"`) || !strings.Contains(head, `Hero`) {
		t.Fatalf("unexpected inspector header: %s", head)
	}

	scope := gosx.RenderHTML(RenderScopeStrip(ScopeStripOptions{Class: "studio-scope-strip"}))
	if !strings.Contains(scope, `aria-label="Inspector scope"`) || !strings.Contains(scope, `<span>Home</span>`) || !strings.Contains(scope, `<output data-studio-selection-label="true">No selection</output>`) {
		t.Fatalf("unexpected scope strip: %s", scope)
	}
}

func TestRenderLayerListKeepsEditorContracts(t *testing.T) {
	html := gosx.RenderHTML(RenderLayerList([]LayerItem{
		{
			Key:           "hero",
			Label:         "Hero",
			CardClass:     "home-section-row editor-block editor-block--hero",
			StatusClass:   "status status--ready",
			StatusLabel:   "Visible",
			DragLabel:     "Reorder Hero",
			MoveUpLabel:   "Move Hero up",
			MoveDownLabel: "Move Hero down",
			KeyName:       "homeSectionKey0",
			OrderName:     "homeSectionOrder0",
			EnabledName:   "homeSectionEnabled0",
			Order:         1,
			Enabled:       true,
			Preview: LayerPreview{
				VisualClass: "editor-block__visual editor-block__visual--hero",
				Kicker:      "Hero",
				Title:       "Lead headline",
				TitleSource: "hero-headline",
				Body:        "Lead body",
				BodySource:  "hero-subhead",
				Actions: []LayerPreviewAction{
					{Label: "Shop", Class: "button button--primary", Source: "hero-cta"},
				},
			},
		},
	}, LayerListOptions{Kicker: "Home", Title: "Layers"}))
	checks := []string{
		`data-block-studio="homepage"`,
		`data-block-studio-block="hero"`,
		`data-studio-block-label="Hero"`,
		`name="homeSectionKey0"`,
		`name="homeSectionOrder0" value="1" type="hidden" data-block-studio-order="true"`,
		`data-block-studio-handle="true"`,
		`name="homeSectionEnabled0" data-editor-block-visible="true" checked`,
		`name="homeSectionEnabled0" value="off"`,
		`data-block-studio-move="up"`,
		`data-editor-preview="hero-headline"`,
		`data-editor-preview="hero-cta"`,
	}
	for _, check := range checks {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in layer markup: %s", check, html)
		}
	}
}

func TestRenderBlockLibraryAndLinkGrid(t *testing.T) {
	library := gosx.RenderHTML(RenderBlockLibrary([]BlockLibraryItem{
		{Key: "hero", Label: "Hero", Target: "hero", ButtonLabel: "On page", ButtonClass: "button button--ghost", Active: true},
	}, BlockLibraryOptions{PanelClass: "editor-panel editor-panel--library"}))
	for _, check := range []string{
		`data-panel-key="blocks"`,
		`data-studio-mode-panel="structure"`,
		`class="button button--ghost"`,
		`data-editor-add-block="hero"`,
		`aria-pressed="true"`,
	} {
		if !strings.Contains(library, check) {
			t.Fatalf("expected %q in block library: %s", check, library)
		}
	}

	links := gosx.RenderHTML(RenderLinkGridPanel([]PanelLink{
		{Key: "products", Label: "Products", Summary: "Catalog", Href: "/admin/products"},
	}, LinkGridOptions{PanelKey: "commerce", Mode: "commerce", Kicker: "Commerce", Title: "Storefront"}))
	if !strings.Contains(links, `data-studio-panel-link="products"`) || !strings.Contains(links, `<strong>Products</strong>`) || !strings.Contains(links, `data-gosx-link="true"`) {
		t.Fatalf("unexpected link grid: %s", links)
	}
}

func TestRenderFlowLibraryKeepsFlowAndInsertHooks(t *testing.T) {
	html := gosx.RenderHTML(RenderFlowLibrary([]FlowCard{
		{
			Key:                "contact",
			Label:              "Contact",
			Description:        "Contact form",
			Summary:            "2 steps",
			StatusClass:        "status status--ready",
			StatusLabel:        "Ready",
			CardClass:          "studio-flow-card studio-flow-card--ready",
			Route:              "/contact",
			EmbedTarget:        "contact",
			PrimaryHandlerRef:  "contact.submit",
			RequiredFieldCount: 3,
			HasRoute:           true,
			HasEmbedTarget:     true,
			HasPrimaryAction:   true,
			Steps:              []FlowStep{{Key: "start", Label: "Start"}},
			Actions: []FlowAction{{Fields: []FlowField{
				{Name: "email", Label: "Email", RequiredLabel: "Required"},
			}}},
		},
	}, FlowLibraryOptions{}))
	for _, check := range []string{
		`data-panel-key="flows"`,
		`data-editor-flow="contact"`,
		`contact.submit`,
		`data-flow-field="email"`,
		`href="/contact" data-gosx-link="true"`,
		`data-editor-add-block="contact"`,
		`data-editor-button-base="button studio-flow-card__embed"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in flow library: %s", check, html)
		}
	}
}

func TestRenderInspectorPanelAndRevisionHistory(t *testing.T) {
	inspector := gosx.RenderHTML(RenderInspectorPanel([]InspectorField{
		{
			Kind:  InspectorFieldInput,
			ID:    "heroHeadline",
			Name:  "heroHeadline",
			Label: "Headline",
			Value: "Lead",
			Wide:  true,
			Attrs: []FieldAttribute{
				{Name: "data-editor-source", Value: "hero-headline"},
				{Name: "data-studio-field-source", Value: "hero.headline"},
			},
		},
		{
			Kind:      InspectorFieldCard,
			Label:     "Hero media",
			CardTitle: "Featured image",
			Attrs: []FieldAttribute{
				{Name: "data-studio-field-source", Value: "hero.media"},
				{Name: "data-studio-field-editable", Value: "media"},
			},
			ContainerAttrs: SelectionScopeAttrs("hero", ""),
			Actions: []FieldAction{
				{Label: "Products", Href: "/admin/products"},
				{Label: "Media", Href: "/admin/media", Primary: true},
			},
		},
	}, InspectorPanelOptions{DynamicTitle: true, Title: "Hero"}))
	if !strings.Contains(inspector, `data-studio-selection-label="true"`) || !strings.Contains(inspector, `data-editor-source="hero-headline"`) || !strings.Contains(inspector, `data-studio-field-source="hero.media"`) || !strings.Contains(inspector, `data-studio-field-editable="media"`) || !strings.Contains(inspector, `data-studio-field-action-href="/admin/media"`) || !strings.Contains(inspector, `data-studio-inspector-for="hero"`) {
		t.Fatalf("unexpected inspector panel: %s", inspector)
	}

	revisions := gosx.RenderHTML(RenderRevisionHistory([]RevisionItem{
		{
			ID:             "rev_1",
			Title:          "Site",
			ActionLabel:    "saved",
			Summary:        "Updated hero",
			HasSummary:     true,
			ChangeSummary:  "1 changed field.",
			HasDiff:        true,
			CreatedLabel:   "May 17, 2026",
			CreatedMachine: "2026-05-17T12:00:00Z",
			ChangeItems:    []RevisionChangeItem{{KindLabel: "Changed", Path: "hero.headline"}},
		},
	}, RevisionHistoryOptions{RestoreAction: "/admin/editor/__actions/restoreRevision", CSRFToken: "csrf"}))
	for _, check := range []string{
		`data-panel-key="versions"`,
		`name="csrf_token" value="csrf"`,
		`name="revisionId" value="rev_1"`,
		`class="revision-diff-summary"`,
		`<code>hero.headline</code>`,
		`data-admin-confirm="Restore these editor settings? Current look and feel will be saved in history first."`,
	} {
		if !strings.Contains(revisions, check) {
			t.Fatalf("expected %q in revisions: %s", check, revisions)
		}
	}
}

func TestBlockInspectorFieldsBuildSelectionScopedSourceCards(t *testing.T) {
	fields := BlockInspectorFields([]blockstudio.Definition{
		{Key: "hero", Label: "Hero", Summary: "Lead block"},
		{Key: "products", Label: "Products", Summary: "Featured products"},
	}, map[string]BlockInspectorOptions{
		"products": {
			ActionLabel: "Manage products",
			Actions: []FieldAction{
				{Label: "Products", Href: "/admin/products", Primary: true},
				{Label: "Collections", Href: "/admin/categories"},
			},
		},
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-inspector-for="products"`,
		`data-studio-field-source="products.collection"`,
		`data-studio-field-editable="source"`,
		`data-studio-field-action="Manage products"`,
		`data-studio-field-action-href="/admin/products"`,
		`<label>Products</label>`,
		`<strong>Featured products</strong>`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in generated block inspector: %s", check, html)
		}
	}
	if strings.Contains(html, `data-studio-inspector-for="hero"`) {
		t.Fatalf("expected only configured block cards, got: %s", html)
	}
}

func TestBlockFieldInspectorFieldsBuildDescriptorControls(t *testing.T) {
	fields := BlockFieldInspectorFields(blockstudio.Definition{
		Key:   "hero",
		Label: "Hero",
		Fields: []blockstudio.FieldDefinition{
			{Name: "headline", Label: "Headline", Kind: blockstudio.FieldText, Required: true, Placeholder: "Opening line"},
			{Name: "body", Label: "Body", Kind: blockstudio.FieldTextarea},
			{Name: "image", Label: "Image", Kind: blockstudio.FieldImage, UI: blockstudio.FieldUI{Picker: "media"}},
			{Name: "visible", Label: "Visible", Kind: blockstudio.FieldBoolean},
			{Name: "tone", Label: "Tone", Kind: blockstudio.FieldSelect, Options: []blockstudio.FieldOption{
				{Value: "warm", Label: "Warm"},
				{Value: "plain", Label: "Plain"},
			}},
		},
	}, BlockFieldInspectorOptions{
		MediaListID: "editor-media-urls",
		Values: map[string]string{
			"headline": "Forest school",
			"visible":  "on",
			"tone":     "warm",
		},
		MediaAltTargets: map[string]string{"image": "heroImageAlt"},
		Overrides: map[string]InspectorFieldOverride{
			"headline": {
				ID:     "heroHeadline",
				Name:   "heroHeadline",
				Source: "hero.headline",
				Wide:   true,
				Attrs: []FieldAttribute{
					{Name: "data-editor-source", Value: "hero-headline"},
				},
			},
		},
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-inspector-for="hero"`,
		`id="heroHeadline"`,
		`name="heroHeadline"`,
		`value="Forest school"`,
		`required`,
		`placeholder="Opening line"`,
		`data-editor-source="hero-headline"`,
		`<textarea`,
		`id="heroBody"`,
		`type="url"`,
		`data-studio-field-editable="media"`,
		`data-studio-field-picker="media"`,
		`list="editor-media-urls"`,
		`data-media-alt-target="heroImageAlt"`,
		`type="checkbox"`,
		`checked`,
		`<select`,
		`value="warm" selected`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in generated fields: %s", check, html)
		}
	}
}

func TestWorkbenchFieldInspectorFieldsBuildFlowAndResourceControls(t *testing.T) {
	fields := WorkbenchFieldInspectorFields([]workbench.Field{
		{Name: "email", Label: "Email", Kind: workbench.FieldText, Required: true},
		{Name: "notes", Label: "Notes", Kind: workbench.FieldTextarea},
		{Name: "status", Label: "Status", Kind: workbench.FieldSelect, Options: []string{"draft", "ready"}},
		{Name: "startsAt", Label: "Starts at", Kind: workbench.FieldDateTime, ReadOnly: true},
	}, WorkbenchFieldInspectorOptions{
		IDPrefix:       "flowContact",
		NamePrefix:     "flowContact",
		SourcePrefix:   "flow.contact.",
		ContainerAttrs: SelectionScopeAttrs("contact"),
		Values:         map[string]string{"status": "ready"},
		Disabled:       true,
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-inspector-for="contact"`,
		`id="flowContactEmail"`,
		`name="flowContactEmail"`,
		`data-studio-field-source="flow.contact.email"`,
		`required`,
		`disabled`,
		`<textarea`,
		`<select`,
		`value="ready" selected`,
		`type="datetime-local"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in workbench fields: %s", check, html)
		}
	}
}

func TestLifecycleInspectorFieldsExposePreviewAndPublishState(t *testing.T) {
	fields := LifecycleInspectorFields(LifecycleInspectorOptions{
		DraftState:   lifecycle.DraftStatePreview,
		PublishState: lifecycle.PublishStateDraft,
		PreviewHref:  "/preview",
		PublishHref:  "/admin/editor/publish",
		Disabled:     true,
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-inspector-for="lifecycle"`,
		`id="lifecycleDraftState"`,
		`value="preview" selected`,
		`id="lifecyclePublishState"`,
		`value="draft" selected`,
		`disabled`,
		`data-studio-field-source="lifecycle.publish"`,
		`data-studio-field-editable="lifecycle"`,
		`data-studio-field-action-href="/preview"`,
		`href="/admin/editor/publish"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in lifecycle inspector: %s", check, html)
		}
	}
}

func TestLifecycleInspectorFieldsExposeSubmitActions(t *testing.T) {
	fields := LifecycleInspectorFields(LifecycleInspectorOptions{
		DraftState:    lifecycle.DraftStatePreview,
		PublishState:  lifecycle.PublishStateDraft,
		PreviewHref:   "/preview",
		PublishAction: "/admin/editor/__actions/publish",
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-field-action-formaction="/admin/editor/__actions/publish"`,
		`formaction="/admin/editor/__actions/publish"`,
		// Shared-form validation hardening (mirrors panels/*_panel.go and
		// panels/inspector_fields.go's Home field-row actions): this card
		// action button has no form= attribute of its own (nests inside
		// whatever <form> the caller wraps the field list in), so a hidden
		// invalid control elsewhere in that ancestor form would otherwise
		// silently cancel this submission.
		`formnovalidate="formnovalidate"`,
		`data-studio-submit-action="publish"`,
		`data-admin-confirm="Publish this draft?"`,
		`>Publish</button>`,
		`href="/preview" data-gosx-link="true"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in lifecycle submit action: %s", check, html)
		}
	}
}

func TestFlowConfigInspectorFieldsExposeRoutingAndPublishAction(t *testing.T) {
	fields := FlowConfigInspectorFields(FlowCard{
		Key:               "contact",
		Label:             "Contact",
		Route:             "/contact",
		EmbedTarget:       "contact",
		PrimaryHandlerRef: "contact.submit",
		Summary:           "1 steps / 1 actions / 3 fields",
		StatusLabel:       "Draft",
	}, FlowConfigInspectorOptions{
		PublishAction: "/admin/editor/__actions/publishFlow",
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-inspector-for="contact"`,
		`id="flowContactRoute"`,
		`name="flowContactRoute"`,
		`value="/contact"`,
		`disabled`,
		`data-studio-field-source="flow.contact.route"`,
		`id="flowContactHandlerRef"`,
		`name="flowContactHandlerRef"`,
		`value="contact.submit"`,
		`data-studio-field-editable="flow"`,
		`data-studio-field-action-formaction="/admin/editor/__actions/publishFlow"`,
		`name="flowKey" value="contact"`,
		`data-studio-submit-action="publish-flow"`,
		`Publishes current draft: 1 steps / 1 actions / 3 fields; route /contact; embed target contact; handler contact.submit.`,
		`>Publish flow</button>`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in flow config inspector: %s", check, html)
		}
	}
}

func TestFlowStepInspectorFieldsExposeStepLabelsAndBodyReview(t *testing.T) {
	fields := FlowStepInspectorFields(FlowCard{
		Key: "contact",
		Steps: []FlowStep{{
			Key:        "message",
			Label:      "Message",
			BlockCount: 2,
			HasBlocks:  true,
		}},
	}, FlowStepInspectorOptions{
		Values: map[string]string{"message.label": "Studio message"},
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-inspector-for="contact"`,
		`id="flowContactStepMessageLabel"`,
		`name="flowContactStepMessageLabel"`,
		`value="Studio message"`,
		`data-studio-field-source="flow.contact.steps.message.label"`,
		`data-studio-field-editable="flow"`,
		`Message body`,
		`2 body blocks`,
		`data-studio-field-source="flow.contact.steps.message.body"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in flow step inspector: %s", check, html)
		}
	}
}

func TestCalendarWidgetInspectorFieldsExposeRecipeAndActions(t *testing.T) {
	contract := calendar.DefaultScheduleWidgetContract(calendar.ScheduleWidgetOptions{
		Key:            "school-schedule",
		Label:          "School schedule",
		PublicHref:     "/schedule",
		AdminHref:      "/admin/schedule",
		RegisterAction: "schedule.register",
	})
	fields := CalendarWidgetInspectorFields(contract, CalendarWidgetInspectorOptions{
		Values:   map[string]string{"density": "airy"},
		Disabled: true,
	})
	html := gosx.RenderHTML(RenderInspectorPanel(fields, InspectorPanelOptions{}))
	for _, check := range []string{
		`data-studio-inspector-for="school-schedule"`,
		`data-studio-field-source="calendar.school-schedule.density"`,
		`value="airy" selected`,
		`required`,
		`disabled`,
		`Schedule data`,
		`10 described schedule fields`,
		`Schedule actions`,
		`2 registered schedule actions`,
		`data-studio-field-action-href="/schedule"`,
		`href="/admin/schedule"`,
	} {
		if !strings.Contains(html, check) {
			t.Fatalf("expected %q in calendar inspector: %s", check, html)
		}
	}
}
