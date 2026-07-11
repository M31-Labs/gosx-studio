package panels

import (
	"fmt"
	"sort"
	"strings"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/authoring"
	"m31labs.dev/gosx-studio/cms/media"
)

type AssetLibraryPanelOptions struct {
	ID, Action, CSRFToken, SelectedAssetID, SelectedBindingID string
	Assets                                                    []media.Asset
	Bindings                                                  []authoring.AssetBinding
	Target                                                    authoring.AssetBindingTarget
	OperationIDs                                              map[string]string
	Heads                                                     map[string]string
}

func RenderAssetLibraryPanel(o AssetLibraryPanelOptions) gosx.Node {
	if strings.TrimSpace(o.ID) == "" {
		o.ID = "studio-asset-library"
	}
	assets := append([]media.Asset(nil), o.Assets...)
	sort.Slice(assets, func(i, j int) bool { return assets[i].Filename < assets[j].Filename })
	children := []gosx.Node{
		gosx.El("header", nil, gosx.El("p", gosx.Attrs(gosx.Attr("class", "kicker")), gosx.Text("Assets")), gosx.El("h2", nil, gosx.Text("Asset library")), gosx.El("output", gosx.Attrs(gosx.Attr("aria-live", "polite")), gosx.Text(fmt.Sprintf("%d assets", len(assets))))),
		gosx.El("label", nil, gosx.Text("Search assets"), gosx.El("input", gosx.Attrs(gosx.Attr("type", "search"), gosx.Attr("name", "asset_search"), gosx.Attr("autocomplete", "off")))),
	}
	list := []gosx.Node{}
	for _, a := range assets {
		refs := 0
		for _, b := range o.Bindings {
			if b.AssetID == a.ID {
				refs++
			}
		}
		list = append(list, gosx.El("article", gosx.Attrs(gosx.Attr("data-studio-asset", a.ID), gosx.Attr("data-studio-asset-kind", string(a.Kind))),
			gosx.El("h3", nil, gosx.Text(a.Filename)), gosx.El("p", nil, gosx.Text(string(a.Kind)+" · "+fmt.Sprintf("%d references", refs))),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "#"+o.ID+"-inspector"), gosx.Attr("aria-label", "Inspect "+a.Filename)), gosx.Text("Inspect"))))
	}
	children = append(children, gosx.El("div", gosx.Attrs(gosx.Attr("role", "list"), gosx.Attr("aria-label", "Available assets")), gosx.Fragment(list...)))
	if a, ok := assetByID(assets, o.SelectedAssetID); ok {
		children = append(children, renderAssetInspector(o, a))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("id", o.ID), gosx.Attr("class", "editor-panel studio-asset-library"), gosx.Attr("data-studio-asset-library", "true")), gosx.Fragment(children...))
}

func renderAssetInspector(o AssetLibraryPanelOptions, a media.Asset) gosx.Node {
	refs := 0
	for _, b := range o.Bindings {
		if b.AssetID == a.ID {
			refs++
		}
	}
	fields := []gosx.Node{hidden(authoring.AssetFieldOperation, string(authoring.AssetBind)), hidden(authoring.AssetFieldOperationID, o.OperationIDs["bind"]), hidden(authoring.AssetFieldAssetID, a.ID), hidden(authoring.AssetFieldBindingID, authoring.AssetBindingID(o.Target)), hidden(authoring.AssetFieldProperty, o.Target.Property)}
	for _, bp := range []string{"base", "tablet", "mobile"} {
		alt := gosx.El("label", nil, gosx.Text("Alternative text"), gosx.El("input", gosx.Attrs(gosx.Attr("name", "asset_"+bp+"_alt"), gosx.Attr("value", a.Responsive[bp].Alt))))
		crop := gosx.El("label", nil, gosx.Text("Crop"), gosx.El("select", gosx.Attrs(gosx.Attr("name", "asset_"+bp+"_crop")), gosx.El("option", gosx.Attrs(gosx.Attr("value", "cover")), gosx.Text("Cover")), gosx.El("option", gosx.Attrs(gosx.Attr("value", "contain")), gosx.Text("Contain")), gosx.El("option", gosx.Attrs(gosx.Attr("value", "none")), gosx.Text("No crop"))))
		fx := gosx.El("label", nil, gosx.Text("Horizontal focal point"), gosx.El("input", gosx.Attrs(gosx.Attr("type", "range"), gosx.Attr("min", "0"), gosx.Attr("max", "1"), gosx.Attr("step", "0.01"), gosx.Attr("name", "asset_"+bp+"_focal_x"))))
		fy := gosx.El("label", nil, gosx.Text("Vertical focal point"), gosx.El("input", gosx.Attrs(gosx.Attr("type", "range"), gosx.Attr("min", "0"), gosx.Attr("max", "1"), gosx.Attr("step", "0.01"), gosx.Attr("name", "asset_"+bp+"_focal_y"))))
		fields = append(fields, gosx.El("fieldset", gosx.Attrs(gosx.Attr("data-studio-asset-breakpoint", bp)), gosx.El("legend", nil, gosx.Text(strings.Title(bp))), alt, crop, fx, fy))
	}
	if o.CSRFToken != "" {
		fields = append(fields, hidden("csrf_token", o.CSRFToken))
	}
	fields = append(fields, gosx.El("button", gosx.Attrs(gosx.Attr("type", "submit")), gosx.Text("Bind selected asset")))
	disabled := []any{gosx.Attr("type", "submit")}
	if refs > 0 {
		disabled = append(disabled, gosx.Attr("disabled", "disabled"), gosx.Attr("aria-describedby", o.ID+"-references"))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("id", o.ID+"-inspector"), gosx.Attr("aria-labelledby", o.ID+"-inspector-title"), gosx.Attr("data-studio-asset-inspector", a.ID)),
		gosx.El("h3", gosx.Attrs(gosx.Attr("id", o.ID+"-inspector-title")), gosx.Text(a.Filename)), gosx.El("p", gosx.Attrs(gosx.Attr("id", o.ID+"-references"), gosx.Attr("role", "status")), gosx.Text(fmt.Sprintf("Used by %d bindings. Delete is available only when unused.", refs))),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", o.Action), gosx.Attr("data-studio-asset-bind-form", "true")), gosx.Fragment(fields...)),
		gosx.El("button", gosx.Attrs(disabled...), gosx.Text("Delete asset")))
}

func assetByID(values []media.Asset, id string) (media.Asset, bool) {
	for _, v := range values {
		if v.ID == id {
			return v, true
		}
	}
	return media.Asset{}, false
}
