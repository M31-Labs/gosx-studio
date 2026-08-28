package panels

import (
	"strings"

	"m31labs.dev/gosx"
)

type CollaborationPanelOptions struct {
	Endpoint string
	Resource string
	Title    string
	// ID names this section so the toolbar's compact collaborators summary
	// (shell.RenderWorkbenchCollaborationSummary) can point aria-controls
	// at it and the runtime can toggle it open as the detail view
	// (handoff-4, punch #7 -- presence to top chrome). Defaults to
	// "studio-collaboration-panel".
	ID string
	// Expanded renders the full panel visible by default when true. Default
	// (false) renders it [hidden] -- the toolbar summary button
	// (data-studio-collab-summary) is the always-visible at-a-glance
	// facepile+count, and clicking it reveals this section as the detail
	// view (collabruntime/island_runtime.js's toggleCollaborationDetail).
	// No existing test or product surface required this section visible
	// by default (audited: every reference_apps_collaboration*_test.ts
	// assertion targets attributes/text content, never .toBeVisible()/
	// .click() on this section or its Reconnect button).
	Expanded  bool
	RootAttrs map[string]any
}

func RenderCollaborationPanel(options CollaborationPanelOptions) gosx.Node {
	title := strings.TrimSpace(options.Title)
	if title == "" {
		title = "Collaboration"
	}
	id := strings.TrimSpace(options.ID)
	if id == "" {
		id = "studio-collaboration-panel"
	}
	attrs := []any{gosx.Attr("id", id), gosx.Attr("class", "studio-collaboration-panel"), gosx.Attr("data-gosx-studio-collaboration", "true"), gosx.Attr("data-studio-collab-detail", "true"), gosx.Attr("data-studio-collab-endpoint", strings.TrimSpace(options.Endpoint)), gosx.Attr("data-studio-collab-resource", strings.TrimSpace(options.Resource)), gosx.Attr("data-studio-collab-state", "offline"), gosx.Attr("aria-labelledby", "studio-collaboration-title"), gosx.Attr("hidden", !options.Expanded)}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-collaboration-panel__head")), gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Multiplayer")), gosx.El("h2", gosx.Attrs(gosx.Attr("id", "studio-collaboration-title")), gosx.Text(title))), gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-collab-status", "true"), gosx.Attr("aria-live", "polite")), gosx.Text("Offline"))),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "studio-collaboration-facepile"), gosx.Attr("data-studio-collab-facepile", "true"), gosx.Attr("aria-label", "Connected collaborators"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-collab-live", "true"), gosx.Attr("class", "studio-visually-hidden"), gosx.Attr("role", "status"), gosx.Attr("aria-live", "polite"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-collab-conflict", "true"), gosx.Attr("class", "studio-collaboration-conflict"), gosx.Attr("role", "alert"), gosx.Attr("hidden", true))),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-studio-collab-reconnect", "true")), gosx.Text("Reconnect")),
	)
}
