package studio

import "m31labs.dev/gosx"

type BackendMediaLibraryProps struct {
	Kicker     string
	Title      string
	Summary    string
	Empty      string
	Class      string
	PanelTitle string
	Assets     []BackendMediaAsset
}

type BackendMediaAsset struct {
	Href        string
	URL         string
	Alt         string
	Filename    string
	ContentType string
	Size        string
	StatusLabel string
	IsImage     bool
	GOSXLink    bool
}

func RenderBackendMediaLibrary(props BackendMediaLibraryProps) gosx.Node {
	className := props.Class
	if className == "" {
		className = "admin-page"
	}
	return gosx.El("div", gosx.Attrs(
		gosx.Attr("class", className),
		gosx.Attr("data-gosx-studio-backend-media-library-renderer", "gosx-studio"),
	),
		RenderBackendMediaLibraryContent(props),
	)
}

func RenderBackendMediaLibraryContent(props BackendMediaLibraryProps) gosx.Node {
	return gosx.Fragment(
		RenderBackendMediaLibraryHeading(props),
		RenderBackendMediaLibraryAssets(props),
	)
}

func RenderBackendMediaLibraryHeading(props BackendMediaLibraryProps) gosx.Node {
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-heading")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text(props.Kicker)),
		gosx.El("h1", nil, gosx.Text(props.Title)),
		gosx.El("p", nil, gosx.Text(props.Summary)),
	)
}

func RenderBackendMediaLibraryAssets(props BackendMediaLibraryProps) gosx.Node {
	panelTitle := props.PanelTitle
	if panelTitle == "" {
		panelTitle = "Assets"
	}
	empty := props.Empty
	if empty == "" {
		empty = "No media assets yet."
	}
	children := []gosx.Node{
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "panel__header")),
			gosx.El("h2", nil, gosx.Text(panelTitle)),
		),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "media-grid")),
			gosx.Fragment(renderBackendMediaAssetCards(props.Assets)...),
		),
	}
	if len(props.Assets) == 0 {
		children = append(children, gosx.El("p", gosx.Attrs(gosx.Attr("class", "empty")), gosx.Text(empty)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "panel")), gosx.Fragment(children...))
}

func renderBackendMediaAssetCards(assets []BackendMediaAsset) []gosx.Node {
	nodes := make([]gosx.Node, 0, len(assets))
	for _, asset := range assets {
		attrs := []any{
			gosx.Attr("class", "media-card"),
			gosx.Attr("href", asset.Href),
		}
		if asset.GOSXLink {
			attrs = append(attrs, gosx.Attr("data-gosx-link", "true"))
		}
		nodes = append(nodes, gosx.El("a", gosx.Attrs(attrs...),
			renderBackendMediaAssetPreview(asset),
			gosx.El("strong", nil, gosx.Text(asset.Filename)),
			gosx.El("span", nil, gosx.Text(asset.Size)),
			gosx.El("span", nil, gosx.Text(asset.StatusLabel)),
		))
	}
	return nodes
}

func renderBackendMediaAssetPreview(asset BackendMediaAsset) gosx.Node {
	if asset.IsImage {
		return gosx.El("img", gosx.Attrs(
			gosx.Attr("src", asset.URL),
			gosx.Attr("alt", asset.Alt),
		))
	}
	return gosx.El("span", gosx.Attrs(gosx.Attr("class", "media-card__file")), gosx.Text(asset.ContentType))
}
