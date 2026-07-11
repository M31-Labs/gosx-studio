package panels

import (
	"strings"

	"m31labs.dev/gosx"
)

type CollaborationPanelOptions struct {
	Endpoint  string
	Resource  string
	Title     string
	RootAttrs map[string]any
}

func RenderCollaborationPanel(options CollaborationPanelOptions) gosx.Node {
	title := strings.TrimSpace(options.Title)
	if title == "" {
		title = "Collaboration"
	}
	attrs := []any{gosx.Attr("class", "studio-collaboration-panel"), gosx.Attr("data-gosx-studio-collaboration", "true"), gosx.Attr("data-studio-collab-endpoint", strings.TrimSpace(options.Endpoint)), gosx.Attr("data-studio-collab-resource", strings.TrimSpace(options.Resource)), gosx.Attr("data-studio-collab-state", "offline"), gosx.Attr("aria-labelledby", "studio-collaboration-title")}
	attrs = appendBlockLibraryPanelAttrs(attrs, options.RootAttrs)
	return gosx.El("section", gosx.Attrs(attrs...),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "studio-collaboration-panel__head")), gosx.El("div", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Multiplayer")), gosx.El("h2", gosx.Attrs(gosx.Attr("id", "studio-collaboration-title")), gosx.Text(title))), gosx.El("output", gosx.Attrs(gosx.Attr("data-studio-collab-status", "true"), gosx.Attr("aria-live", "polite")), gosx.Text("Offline"))),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "studio-collaboration-facepile"), gosx.Attr("data-studio-collab-facepile", "true"), gosx.Attr("aria-label", "Connected collaborators"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-collab-live", "true"), gosx.Attr("class", "studio-visually-hidden"), gosx.Attr("role", "status"), gosx.Attr("aria-live", "polite"))),
		gosx.El("p", gosx.Attrs(gosx.Attr("data-studio-collab-conflict", "true"), gosx.Attr("class", "studio-collaboration-conflict"), gosx.Attr("role", "alert"), gosx.Attr("hidden", true))),
		gosx.El("button", gosx.Attrs(gosx.Attr("type", "button"), gosx.Attr("data-studio-collab-reconnect", "true")), gosx.Text("Reconnect")),
	)
}
