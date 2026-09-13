package sitehost

import (
	"net/url"
	"strings"
	"testing"
)

func TestTextAlignmentButtonStylesColumnsAndAnchors(t *testing.T) {
	host, handler := newTestHost(t)
	id := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+id, `{"title":"Menu","slug":"menu","description":"x","blocks":[
		{"kind":"heading","text":"Centred heading","level":"2","align":"center"},
		{"kind":"paragraph","text":"Right-aligned words","align":"right"},
		{"kind":"paragraph","text":"Plain words","align":"nonsense"},
		{"kind":"button","text":"Outline","url":"/contact","look":"ghost","align":"center"},
		{"kind":"button","text":"Just a link","url":"/menu","look":"link"},
		{"kind":"columns","text":"One","text2":"Two","text3":"Three","count":"3"},
		{"kind":"columns","text":"Left","text2":"Right","text3":"ignored","count":"2"},
		{"kind":"section","style":"tinted","anchor":"Pricing Table!"},
		{"kind":"paragraph","text":"Under the anchor"}
	]}`)
	post(t, handler, "/admin/api/pages/"+id+"/publish", url.Values{})
	public := get(t, handler, "/menu").Body.String()
	for _, want := range []string{
		`<h2 class="site-align-center">Centred heading</h2>`, `<p class="site-align-right">Right-aligned words</p>`, `<p>Plain words</p>`,
		`<div class="site-button-row site-align-center"><a class="button button--ghost" href="/contact">Outline</a></div>`,
		`<a class="button button--link" href="/menu">Just a link</a>`,
		`<div class="site-columns site-columns--3"><div class="site-columns__col">One</div><div class="site-columns__col">Two</div><div class="site-columns__col">Three</div></div>`,
		`<div class="site-columns site-columns--2"><div class="site-columns__col">Left</div><div class="site-columns__col">Right</div></div>`,
		`<section class="site-section site-section--tinted" id="pricing-table">`,
	} {
		if !strings.Contains(public, want) {
			t.Fatalf("public page lacks %q:\n%s", want, public)
		}
	}
	editor := get(t, handler, "/admin/edit/"+id).Body.String()
	for _, want := range []string{
		`data-align="center" class="site-align-center" data-text="true" data-level="2"`, `<select data-button-look="true" aria-label="Button style"><option value="primary">Solid</option><option value="ghost" selected="selected">`,
		`data-columns="3"`, `data-col="3"`, `data-section-anchor="true" value="pricing-table"`,
	} {
		if !strings.Contains(editor, want) {
			t.Fatalf("editor lacks %q", want)
		}
	}
}
