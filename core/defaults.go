package core

func DefaultFeatures() []Feature {
	return []Feature{
		{
			Key:     "site-canvas",
			Label:   "Site canvas",
			Surface: SurfaceCanvas,
			Summary: "No-code page structure and preview editing surface.",
		},
		{
			Key:     "site-map",
			Label:   "Site map",
			Surface: SurfaceSiteMap,
			Summary: "Editable page graph where routes are composed from GoSX components.",
		},
		{
			Key:     "site-inspector",
			Label:   "Inspector",
			Surface: SurfaceInspector,
			Summary: "Plain-language controls for selection, style, content, and publish readiness.",
		},
		{
			Key:     "site-flows",
			Label:   "Flows",
			Surface: SurfaceFlow,
			Summary: "Visual automation and lifecycle flow authoring.",
		},
		{
			Key:     "publish-lifecycle",
			Label:   "Publish",
			Surface: SurfacePublish,
			Summary: "Review, schedule, and publish controls for non-technical site operators.",
		},
	}
}

func DefaultResourceAdapters() []ResourceAdapter {
	return []ResourceAdapter{
		{
			Kind:    ResourceMedia,
			Label:   "Media",
			Summary: "Images, video, generated assets, alt text, and focal points.",
			Surface: SurfaceInspector,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
				ResourceSearch,
				ResourcePreview,
			},
			Bindings: []ResourceBinding{
				{Key: "assets", Label: "Assets", Binding: "media.assets"},
				{Key: "uploads", Label: "Uploads", Binding: "media.uploads"},
			},
		},
		{
			Kind:    ResourcePages,
			Label:   "Pages",
			Summary: "Routes, page drafts, page templates, previews, and page publishing state.",
			Surface: SurfaceSiteMap,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
				ResourcePreview,
				ResourcePublish,
			},
			Bindings: []ResourceBinding{
				{Key: "routes", Label: "Routes", Binding: "pages.routes"},
				{Key: "drafts", Label: "Drafts", Binding: "pages.drafts"},
			},
		},
		{
			Kind:    ResourceProducts,
			Label:   "Products",
			Summary: "Product collections, categories, availability, and storefront placement.",
			Surface: SurfaceSiteMap,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceSearch,
				ResourcePreview,
			},
			Bindings: []ResourceBinding{
				{Key: "collection", Label: "Collection", Binding: "products.collection"},
				{Key: "categories", Label: "Categories", Binding: "products.categories"},
			},
		},
		{
			Kind:    ResourceOrders,
			Label:   "Orders",
			Summary: "Store activity that can inform storefront messaging without making Studio the order desk.",
			Surface: SurfacePublish,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
			},
			Bindings: []ResourceBinding{
				{Key: "activity", Label: "Activity", Binding: "orders.activity"},
			},
		},
		{
			Kind:    ResourceContacts,
			Label:   "Contacts",
			Summary: "Contact destinations, customer messages, and form routing context.",
			Surface: SurfaceFlow,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
			},
			Bindings: []ResourceBinding{
				{Key: "messages", Label: "Messages", Binding: "contacts.messages"},
				{Key: "destinations", Label: "Destinations", Binding: "contacts.destinations"},
			},
		},
		{
			Kind:    ResourceSettings,
			Label:   "Settings",
			Summary: "Site identity, theme, navigation, checkout, and operational settings.",
			Surface: SurfaceInspector,
			Capabilities: []ResourceCapability{
				ResourceRead,
				ResourceWrite,
				ResourcePreview,
			},
			Bindings: []ResourceBinding{
				{Key: "site", Label: "Site", Binding: "settings.site"},
				{Key: "theme", Label: "Theme", Binding: "settings.theme"},
			},
		},
		{
			Kind:    ResourceRevisions,
			Label:   "Revisions",
			Summary: "Saved versions, compare points, and restore actions.",
			Surface: SurfacePublish,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceRestore,
			},
			Bindings: []ResourceBinding{
				{Key: "history", Label: "History", Binding: "revisions.history"},
			},
		},
		{
			Kind:    ResourceLifecycle,
			Label:   "Lifecycle",
			Summary: "Readiness, scheduling, approvals, preview links, and publish decisions.",
			Surface: SurfacePublish,
			Capabilities: []ResourceCapability{
				ResourceRead,
				ResourceWrite,
				ResourcePreview,
				ResourcePublish,
			},
			Bindings: []ResourceBinding{
				{Key: "readiness", Label: "Readiness", Binding: "lifecycle.readiness"},
				{Key: "schedule", Label: "Schedule", Binding: "lifecycle.schedule"},
			},
		},
		{
			Kind:    ResourceFlows,
			Label:   "Flows",
			Summary: "Visual automations, form handlers, lifecycle actions, and publishable flow drafts.",
			Surface: SurfaceFlow,
			Capabilities: []ResourceCapability{
				ResourceList,
				ResourceRead,
				ResourceWrite,
				ResourceExecute,
			},
			Bindings: []ResourceBinding{
				{Key: "library", Label: "Library", Binding: "flows.library"},
				{Key: "drafts", Label: "Drafts", Binding: "flows.drafts"},
			},
		},
	}
}

func DefaultEngines() []Engine {
	return []Engine{
		{
			Key:     "canvas",
			Label:   "Canvas",
			Kind:    EngineCanvas,
			MountID: "gosx-studio-canvas-engine",
			Surface: SurfaceCanvas,
			Capabilities: []EngineCapability{
				CapabilityDragDrop,
				CapabilityPanZoom,
				CapabilitySelection,
				CapabilityInlineEdit,
				CapabilityPreview,
				CapabilityPersistence,
			},
		},
		{
			Key:     "site-map",
			Label:   "Site map",
			Kind:    EngineSiteMap,
			MountID: "gosx-studio-site-map-engine",
			Surface: SurfaceSiteMap,
			Capabilities: []EngineCapability{
				CapabilityPanZoom,
				CapabilitySelection,
				CapabilityPersistence,
			},
		},
		{
			Key:     "block-layout",
			Label:   "Block layout",
			Kind:    EngineBlockLayout,
			MountID: "gosx-studio-block-layout-engine",
			Surface: SurfaceCanvas,
			Capabilities: []EngineCapability{
				CapabilityDragDrop,
				CapabilitySelection,
				CapabilityPersistence,
			},
		},
		{
			Key:     "showcase-3d",
			Label:   "Showcase 3D",
			Kind:    EngineScene3D,
			MountID: "gosx-studio-showcase-3d-engine",
			Surface: SurfaceShowcase3D,
			Capabilities: []EngineCapability{
				CapabilityPreview,
				CapabilityPopout,
			},
		},
	}
}

func DefaultRuntimeContracts() []RuntimeContract {
	return []RuntimeContract{
		{
			Key:     "preview-runtime",
			Label:   "Preview runtime",
			Global:  "GoSXStudioPreviewRuntime",
			Surface: SurfaceCanvas,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "mount",
					Label:   "Mount preview shell",
					Summary: "Bind the GoSX-authored preview shell, overlays, dock controls, and inline editing affordances.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains preview workbench markup."},
					},
				},
				{
					Name:    "setBlockVisibility",
					Label:   "Set block visibility",
					Summary: "Apply visible or hidden state to a GoSX-backed preview block.",
					Payload: []RuntimePayloadField{
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "visible", Label: "Visible", Kind: ControlToggle, Required: true},
					},
				},
				{
					Name:    "applyCSS",
					Label:   "Apply custom CSS",
					Summary: "Apply editor-provided custom CSS to the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "cssText", Label: "CSS text", Kind: ControlRichText},
					},
				},
				{
					Name:    "applyTextUpdate",
					Label:   "Apply text update",
					Summary: "Mirror editor text and attribute edits into the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "sourceKey", Label: "Source key", Kind: ControlText},
						{Name: "frameTarget", Label: "Frame target", Kind: ControlText},
						{Name: "attrTarget", Label: "Attribute target", Kind: ControlText},
						{Name: "attrName", Label: "Attribute name", Kind: ControlText},
						{Name: "attrPrefix", Label: "Attribute prefix", Kind: ControlText},
						{Name: "attrSuffix", Label: "Attribute suffix", Kind: ControlText},
						{Name: "value", Label: "Value", Kind: ControlRichText},
					},
				},
				{
					Name:    "applyTheme",
					Label:   "Apply theme",
					Summary: "Apply selected kit, template, palette, image ratio, style classes, and color tokens to the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "kit", Label: "Kit", Kind: ControlChoice},
						{Name: "template", Label: "Template", Kind: ControlChoice},
						{Name: "palette", Label: "Palette", Kind: ControlChoice},
						{Name: "imageRatio", Label: "Image ratio", Kind: ControlChoice},
						{Name: "customClasses", Label: "Custom classes", Kind: ControlText},
						{Name: "styleClasses", Label: "Style classes", Kind: ControlText},
						{Name: "colors", Label: "Color tokens", Kind: ControlColor},
					},
				},
				{
					Name:    "applyStyleImpact",
					Label:   "Apply style impact",
					Summary: "Clear previous style-impact markers and mark preview nodes affected by a selected style scope.",
					Payload: []RuntimePayloadField{
						{Name: "selector", Label: "Scope selector", Kind: ControlText},
					},
				},
				{
					Name:    "applyFonts",
					Label:   "Apply font CSS",
					Summary: "Apply generated font-face and font-token CSS to the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "cssText", Label: "CSS text", Kind: ControlRichText},
					},
				},
				{
					Name:    "updateHeaderLogo",
					Label:   "Update header logo",
					Summary: "Update live header logo source, alt text, size, and offsets in the Studio shell and preview frames.",
					Payload: []RuntimePayloadField{
						{Name: "url", Label: "Logo URL", Kind: ControlMedia},
						{Name: "alt", Label: "Alt text", Kind: ControlText},
						{Name: "width", Label: "Width", Kind: ControlNumber},
						{Name: "x", Label: "X offset", Kind: ControlNumber},
						{Name: "y", Label: "Y offset", Kind: ControlNumber},
					},
				},
				{
					Name:    "requestInlineEdit",
					Label:   "Request inline edit",
					Summary: "Focus a field hotspot and open the inline text editing affordance for the selected block.",
					Payload: []RuntimePayloadField{
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "field", Label: "Field key", Kind: ControlText},
					},
				},
				{
					Name:    "cycleField",
					Label:   "Cycle field",
					Summary: "Move field selection within the selected preview block.",
					Payload: []RuntimePayloadField{
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "direction", Label: "Direction", Kind: ControlNumber, Required: true},
					},
				},
			},
		},
		{
			Key:     "workbench-runtime",
			Label:   "Workbench runtime",
			Global:  "GoSXStudioWorkbenchRuntime",
			Surface: SurfaceCanvas,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "bindRailResizers",
					Label:   "Bind rail resizers",
					Summary: "Bind pointer and keyboard resizing for GoSX-authored workbench rail handles.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the Studio workbench."},
					},
				},
				{
					Name:    "bindChrome",
					Label:   "Bind workbench chrome",
					Summary: "Bind GoSX-authored workbench mode controls, viewport controls, zoom controls, rail toggles, activity toggles, focus state, command-palette workbench commands, and saved layout state.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the Studio workbench."},
					},
				},
				{
					Name:    "setMode",
					Label:   "Set workbench mode",
					Summary: "Switch the active Studio mode and synchronize mode panels, labels, and mode change events.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "mode", Label: "Mode", Kind: ControlChoice, Required: true},
						{Name: "scroll", Label: "Scroll into view", Kind: ControlToggle},
					},
				},
				{
					Name:    "syncViewport",
					Label:   "Sync viewport",
					Summary: "Synchronize active viewport controls, preview frame sizing, and viewport readouts for the current page canvas.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "viewport", Label: "Viewport", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "activateViewport",
					Label:   "Activate viewport",
					Summary: "Apply an editor-selected viewport and emit a viewport change event for engine and inspector synchronization.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "viewport", Label: "Viewport", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "currentBreakpoint",
					Label:   "Current breakpoint",
					Summary: "Read the active canvas breakpoint so style controls can target the same viewport state as the workbench.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "setStyleState",
					Label:   "Set style state",
					Summary: "Switch the active style-state target for hover/focus previews and emit a style-state change event.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "state", Label: "Style state", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "syncZoom",
					Label:   "Sync zoom",
					Summary: "Synchronize active zoom controls and canvas scale without changing the underlying page model.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "zoom", Label: "Zoom", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "activateZoom",
					Label:   "Activate zoom",
					Summary: "Apply an editor-selected zoom level, persist workbench layout state, and emit a zoom change event.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "zoom", Label: "Zoom", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "toggleRail",
					Label:   "Toggle rail",
					Summary: "Open or collapse a Studio workbench rail and persist the chrome layout state.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "side", Label: "Rail side", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "toggleFocus",
					Label:   "Toggle focus",
					Summary: "Toggle canvas focus mode, synchronize rail state, and persist the workbench layout.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "toggleActivity",
					Label:   "Toggle activity",
					Summary: "Open or collapse the workbench activity rail and persist the chrome layout state.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "saveLayout",
					Label:   "Save layout",
					Summary: "Persist the current Studio workbench layout state for the active editor.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "currentRailWidth",
					Label:   "Current rail width",
					Summary: "Read the current width of a Studio workbench rail.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "side", Label: "Rail side", Kind: ControlChoice, Required: true},
						{Name: "handle", Label: "Rail handle", Kind: ControlSource},
					},
				},
				{
					Name:    "setRailWidth",
					Label:   "Set rail width",
					Summary: "Apply a clamped width to a Studio workbench rail and emit live or committed resize events.",
					Payload: []RuntimePayloadField{
						{Name: "form", Label: "Workbench form", Kind: ControlSource, Required: true},
						{Name: "side", Label: "Rail side", Kind: ControlChoice, Required: true},
						{Name: "width", Label: "Rail width", Kind: ControlNumber, Required: true},
						{Name: "handle", Label: "Rail handle", Kind: ControlSource},
						{Name: "committed", Label: "Committed", Kind: ControlToggle},
					},
				},
			},
		},
		{
			Key:     "selection-runtime",
			Label:   "Selection runtime",
			Global:  "GoSXStudioSelectionRuntime",
			Surface: SurfaceCanvas,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "bind",
					Label:   "Bind selection surface",
					Summary: "Bind block selection, workspace target selection, field focus, selection commandbar actions, and style-scope readouts for the Studio editor surface.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the Studio workbench."},
					},
				},
			},
		},
		{
			Key:     "field-runtime",
			Label:   "Field runtime",
			Global:  "GoSXStudioFieldRuntime",
			Surface: SurfaceInspector,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "bind",
					Label:   "Bind field utilities",
					Summary: "Bind field-to-preview mirroring and Studio copy controls for GoSX-authored inspector fields.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains Studio field controls."},
					},
				},
				{
					Name:    "bindMirroring",
					Label:   "Bind field mirroring",
					Summary: "Bind configured editor fields to preview text and attribute updates through the preview runtime.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains mirrored field controls."},
					},
				},
				{
					Name:    "bindClipboard",
					Label:   "Bind copy controls",
					Summary: "Bind Studio copy buttons with native clipboard and fallback behavior.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains copy controls."},
					},
				},
			},
		},
		{
			Key:     "brand-runtime",
			Label:   "Brand runtime",
			Global:  "GoSXStudioBrandRuntime",
			Surface: SurfaceInspector,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "bindLogo",
					Label:   "Bind logo controls",
					Summary: "Bind logo placement, snap, keyboard nudge, reset, and live-preview controls for the Brand inspector.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the Brand inspector."},
					},
				},
				{
					Name:    "updateHeaderLogo",
					Label:   "Update header logo",
					Summary: "Send Brand inspector logo source, alt text, size, and offsets to the preview runtime.",
					Payload: []RuntimePayloadField{
						{Name: "url", Label: "Logo URL", Kind: ControlMedia},
						{Name: "alt", Label: "Alt text", Kind: ControlText},
						{Name: "width", Label: "Width", Kind: ControlNumber},
						{Name: "x", Label: "X offset", Kind: ControlNumber},
						{Name: "y", Label: "Y offset", Kind: ControlNumber},
					},
				},
			},
		},
		{
			Key:     "style-runtime",
			Label:   "Style runtime",
			Global:  "GoSXStudioStyleRuntime",
			Surface: SurfaceInspector,
			Engine:  EngineCanvas,
			Methods: []RuntimeMethod{
				{
					Name:    "bindTheme",
					Label:   "Bind theme controls",
					Summary: "Bind theme kits, templates, color tokens, and image-ratio controls for live preview application.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains Look theme controls."},
					},
				},
				{
					Name:    "bindWorkbench",
					Label:   "Bind style workbench",
					Summary: "Bind Look inspector style recipes, reset controls, hover previews, and impact readouts.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the Look style workbench."},
					},
				},
				{
					Name:    "bindCSS",
					Label:   "Bind custom CSS",
					Summary: "Bind custom CSS controls and mirror CSS into the Studio preview runtime.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains custom CSS controls."},
					},
				},
				{
					Name:    "bindFonts",
					Label:   "Bind font controls",
					Summary: "Bind font family and URL controls and mirror generated font CSS into the Studio preview runtime.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains font controls."},
					},
				},
				{
					Name:    "applyTheme",
					Label:   "Apply theme",
					Summary: "Apply the current theme kit, template, palette, style classes, and color tokens to the preview runtime.",
				},
				{
					Name:    "syncControlButtons",
					Label:   "Sync style controls",
					Summary: "Refresh selected recipe states, inherited readouts, and reset affordances from current theme values.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains style controls."},
					},
				},
				{
					Name:    "showImpact",
					Label:   "Show style impact",
					Summary: "Preview or commit the affected site areas for a style control value.",
					Payload: []RuntimePayloadField{
						{Name: "name", Label: "Control name", Kind: ControlText, Required: true},
						{Name: "value", Label: "Control value", Kind: ControlChoice},
						{Name: "committed", Label: "Committed", Kind: ControlToggle},
					},
				},
				{
					Name:    "restoreImpact",
					Label:   "Restore style impact",
					Summary: "Restore the last committed style impact readout or clear transient hover previews.",
				},
				{
					Name:    "setControlValue",
					Label:   "Set style value",
					Summary: "Apply a Look inspector style control value and dispatch the underlying theme field changes.",
					Payload: []RuntimePayloadField{
						{Name: "name", Label: "Control name", Kind: ControlText, Required: true},
						{Name: "value", Label: "Control value", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "resetControlValue",
					Label:   "Reset style value",
					Summary: "Reset a Look inspector style control to the active starter kit value.",
					Payload: []RuntimePayloadField{
						{Name: "name", Label: "Control name", Kind: ControlText, Required: true},
					},
				},
			},
		},
		{
			Key:     "block-layout-runtime",
			Label:   "Block layout runtime",
			Global:  "GoSXStudioBlockLayoutRuntime",
			Surface: SurfaceCanvas,
			Engine:  EngineBlockLayout,
			Methods: []RuntimeMethod{
				{
					Name:    "rows",
					Label:   "Rows",
					Summary: "Return the editable block rows in a page layout list.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "rowKey",
					Label:   "Row key",
					Summary: "Read the stable GoSX block key for a layout row.",
					Payload: []RuntimePayloadField{
						{Name: "row", Label: "Layout row", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "rowForKey",
					Label:   "Row for key",
					Summary: "Find a block layout row by stable GoSX block key.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
					},
				},
				{
					Name:    "moveRow",
					Label:   "Move row",
					Summary: "Move a block row within the current layout.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "direction", Label: "Direction", Kind: ControlNumber, Required: true},
					},
				},
				{
					Name:    "renumber",
					Label:   "Renumber rows",
					Summary: "Refresh visible position labels after a block layout change.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
					},
				},
				{
					Name:    "selectRow",
					Label:   "Select row",
					Summary: "Select the row that represents a GoSX-backed page block.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
					},
				},
				{
					Name:    "commitReorder",
					Label:   "Commit reorder",
					Summary: "Commit a drag/drop reorder operation against the current block layout.",
					Payload: []RuntimePayloadField{
						{Name: "list", Label: "Layout list", Kind: ControlSource, Required: true},
						{Name: "key", Label: "Block key", Kind: ControlText, Required: true},
						{Name: "targetKey", Label: "Target block key", Kind: ControlText, Required: true},
						{Name: "position", Label: "Position", Kind: ControlChoice, Required: true},
					},
				},
				{
					Name:    "updateBlockLibraryState",
					Label:   "Update block library state",
					Summary: "Refresh component-palette availability and active state from the current page layout.",
					Payload: []RuntimePayloadField{
						{Name: "root", Label: "Root element", Kind: ControlSource, Summary: "Document or element that contains the block library."},
					},
				},
				{
					Name:    "updateVisibilityState",
					Label:   "Update visibility state",
					Summary: "Refresh a block row's visibility status, component-palette state, and live preview visibility.",
					Payload: []RuntimePayloadField{
						{Name: "check", Label: "Visibility control", Kind: ControlToggle, Required: true},
					},
				},
			},
		},
	}
}
